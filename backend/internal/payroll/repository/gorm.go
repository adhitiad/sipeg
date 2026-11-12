package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/outbox"
	"simpeg/backend/internal/payroll/domain"
	"simpeg/backend/internal/timeutil"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormStore adalah adapter persistensi slip gaji berbasis GORM.
type GormStore struct {
	DB *gorm.DB
}

// ListSlips membaca daftar slip gaji sesuai filter dan pagination.
//
// Slip milik pegawai lain tidak terlihat bila principal tidak memegang
// cakupan unit; penyaringan memakai subquery ke tabel pegawai.
func (s GormStore) ListSlips(ctx context.Context, options listing.Options) ([]models.Gaji, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.Gaji{}).Preload("Pegawai").Preload("GajiTunjangans").Preload("GajiPotongans")
	if options.UserID != "" {
		query = query.Where(`"userId" = ?`, options.UserID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where(
			`"userId" IN (SELECT "id" FROM "user" WHERE "homebaseId" IN ?)`,
			options.ScopeHomebaseIDs,
		)
	}
	if options.Periode != "" {
		period, err := domain.ParsePeriod(options.Periode)
		if err != nil {
			return nil, 0, err
		}
		query = query.Where("periode = ?", timeutil.DateOnly(period.Start()))
	}
	if options.Status != "" {
		query = query.Where("status = ?", options.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count payroll slips: %w", err)
	}
	var slips []models.Gaji
	order := listing.OrderClause(domain.SlipSortColumns, options.Sort, options.Order, "periode")
	err := query.Order(order).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Find(&slips).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list payroll slips: %w", err)
	}
	return slips, total, nil
}

// GetSlip membaca satu slip gaji beserta data pegawai pemiliknya.
func (s GormStore) GetSlip(ctx context.Context, id string) (models.Gaji, error) {
	var slip models.Gaji
	err := s.DB.WithContext(ctx).Preload("Pegawai").Preload("GajiTunjangans").Preload("GajiPotongans").Where("id = ?", id).First(&slip).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Gaji{}, domain.ErrNotFound
		}
		return models.Gaji{}, fmt.Errorf("read payroll slip: %w", err)
	}
	return slip, nil
}

// RecordPayment mencatat konfirmasi pembayaran untuk sebuah slip.
//
// Seluruh perubahan berjalan dalam satu transaksi: baris konfirmasi dibuat,
// slip ditandai TERBAYAR, notifikasi diantrekan pada antrean keluar persisten,
// dan jejak audit ditulis. Notifikasi diantrekan di dalam transaksi yang sama
// agar email tidak terkirim tanpa bukti pembayaran yang tersimpan.
func (s GormStore) RecordPayment(ctx context.Context, actorID, slipID string, input domain.PaymentInput, notification domain.Notification) (models.Gaji, error) {
	var slip models.Gaji
	var confirmedAt, paidAt time.Time
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", slipID).First(&slip).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("read payroll slip: %w", err)
		}
		if !domain.CanMarkPaid(slip.Status) {
			return domain.ErrConflict
		}

		confirmedAt = timeutil.NowUTC()
		paidAt = timeutil.DateOnly(input.TanggalBayar)
		payment := models.Pembayaran{
			GajiId:             slip.ID,
			ReferensiTransaksi: input.ReferensiTransaksi,
			WaktuKonfirmasi:    confirmedAt,
			PelakuPengesahan:   input.PelakuPengesahan,
			Keterangan:         input.Keterangan,
		}
		if err := tx.Create(&payment).Error; err != nil {
			return fmt.Errorf("record payroll payment: %w", err)
		}

		updates := map[string]any{
			"status":                 models.StatusSlipTerbayar,
			"tanggalDibayar":         paidAt,
			"konfirmasiPembayaranAt": confirmedAt,
			"konfirmasiOlehId":       actorID,
		}
		if err := tx.Model(&models.Gaji{}).Where("id = ?", slipID).Updates(updates).Error; err != nil {
			return fmt.Errorf("update payroll slip: %w", err)
		}

		if notification.Kind != "" {
			queue := outbox.GormStore{DB: tx}
			if _, err := queue.Enqueue(ctx, notification.Kind, notification.DedupeKey, notification.Payload); err != nil {
				return fmt.Errorf("enqueue salary notice: %w", err)
			}
		}

		note := fmt.Sprintf("slip %s periode %s", slip.NomorSlip, slip.Periode.Format("2006-01"))
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionPaymentConfirmed,
			TargetID: &slip.ID,
			Note:     &note,
		})
	})
	if err != nil {
		return models.Gaji{}, err
	}

	slip.Status = models.StatusSlipTerbayar
	slip.TanggalDibayar = &paidAt
	slip.KonfirmasiPembayaranAt = &confirmedAt
	slip.KonfirmasiOlehId = &actorID
	return slip, nil
}

// ConfirmedPayment melaporkan apakah pembayaran untuk pegawai dan periode
// tertentu sudah tercatat sebagai terkonfirmasi.
//
// Konfirmasi dibaca dari baris `pembayaran` yang menempel pada slip, bukan
// dari status slip saja, sehingga bukti pembayaran selalu dapat ditelusuri.
func (s GormStore) ConfirmedPayment(ctx context.Context, userID, period string) (bool, error) {
	if userID == "" || period == "" {
		return false, nil
	}
	parsed, err := domain.ParsePeriod(period)
	if err != nil {
		return false, nil
	}

	var count int64
	err = s.DB.WithContext(ctx).Model(&models.Pembayaran{}).
		Joins(`JOIN "gaji" ON "gaji"."id" = "pembayaran"."gajiId"`).
		Where(`"gaji"."userId" = ? AND "gaji"."periode" = ? AND "gaji"."status" = ?`,
			userID, timeutil.DateOnly(parsed.Start()), models.StatusSlipTerbayar).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check payroll payment confirmation: %w", err)
	}
	return count > 0, nil
}

// createPayrollAudit menulis entri jejak audit untuk area penggajian.
func createPayrollAudit(ctx context.Context, tx *gorm.DB, actorID, action string, targetID, unitID *string) error {
	return audit.Write(ctx, tx, audit.Entry{
		ActorID:  actorID,
		Action:   action,
		TargetID: targetID,
		UnitID:   unitID,
	})
}

func classifyRecordError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return domain.ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return domain.ErrConflict
	default:
		return err
	}
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

func optionalRef(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
