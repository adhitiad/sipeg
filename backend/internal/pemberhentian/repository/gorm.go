// GormStore menyimpan adapter GORM untuk port pemberhentian pegawai (DEC-083).
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/models"
)

// GormStore mengimplementasikan Store dengan PostgreSQL.
type GormStore struct {
	DB *gorm.DB
}

// auditWrite menulis satu entri jejak audit dalam transaksi.
func auditWrite(ctx context.Context, tx *gorm.DB, entry audit.Entry) error {
	return audit.Write(ctx, tx, entry)
}

// EmployeeHomebaseID mengembalikan unit penugasan pegawai; nil bila belum diisi.
func (s GormStore) EmployeeHomebaseID(ctx context.Context, userID string) (*string, error) {
	var row struct {
		HomebaseID *string `gorm:"column:homebaseId"`
	}
	if err := s.DB.WithContext(ctx).Table(`"user"`).Select(`"homebaseId"`).Where(`"id" = ?`, userID).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("read homebase pegawai: %w", err)
	}
	return row.HomebaseID, nil
}

// EmployeeExists melaporkan apakah pegawai dikenal.
func (s GormStore) EmployeeExists(ctx context.Context, userID string) (bool, error) {
	var count int64
	if err := s.DB.WithContext(ctx).Table(`"user"`).Where(`"id" = ?`, userID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("cek pegawai: %w", err)
	}
	return count > 0, nil
}

// EmployeeStartDate mengembalikan tanggal mulai kerja pegawai.
func (s GormStore) EmployeeStartDate(ctx context.Context, userID string) (*time.Time, error) {
	var row struct {
		TanggalMasuk *time.Time `gorm:"column:tanggalMasuk"`
	}
	if err := s.DB.WithContext(ctx).Table(`"user"`).Select(`"tanggalMasuk"`).Where(`"id" = ?`, userID).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tanggal masuk pegawai: %w", err)
	}
	return row.TanggalMasuk, nil
}

// EmployeeKategoriStatusID mengembalikan kategori status pegawai.
func (s GormStore) EmployeeKategoriStatusID(ctx context.Context, userID string) (*string, error) {
	var row struct {
		KategoriStatusID *string `gorm:"column:kategoriStatusId"`
	}
	if err := s.DB.WithContext(ctx).Table(`"user"`).Select(`"kategoriStatusId"`).Where(`"id" = ?`, userID).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("read kategori status pegawai: %w", err)
	}
	return row.KategoriStatusID, nil
}

// PendingForUser melaporkan apakah pegawai memiliki pengajuan MENUNGGU.
func (s GormStore) PendingForUser(ctx context.Context, userID string) (bool, error) {
	var count int64
	err := s.DB.WithContext(ctx).Model(&models.Pemberhentian{}).
		Where(`"userId" = ? AND "status" = ?`, userID, models.StatusMenunggu).
		Count(&count).Error
	return count > 0, err
}

// RulesFor membaca pesangonAturan per kategori status.
func (s GormStore) RulesFor(ctx context.Context, kategoriStatusID string) (map[int]float64, error) {
	var rows []models.PesangonAturan
	if err := s.DB.WithContext(ctx).Where(`"kategoriStatusId" = ?`, kategoriStatusID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("read pesangonAturan: %w", err)
	}
	rules := make(map[int]float64, len(rows))
	for _, row := range rows {
		rules[row.MasaKerjaTahun] = row.PesangonBulan
	}
	return rules, nil
}

// Create menulis pengajuan MENUNGGU. Jenis pemberhentian (HORMAT/TIDAK_HORMAT)
// diisi oleh lapisan aplikasi sebelum pemanggilan ini.
func (s GormStore) Create(ctx context.Context, actor string, input ProposeInput, unitID *string) (models.Pemberhentian, error) {
	item := models.Pemberhentian{
		UserId:     input.UserID,
		Jenis:      input.Jenis,
		Alasan:     &input.Alasan,
		Tanggal:    input.Tanggal,
		Status:     models.StatusMenunggu,
		Keterangan: input.Keterangan,
		CreatedBy:  &actor,
	}
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.Pemberhentian{}).
			Where(`"userId" = ? AND "status" = ?`, input.UserID, models.StatusMenunggu).
			Count(&count).Error; err != nil {
			return fmt.Errorf("count pending: %w", err)
		}
		if count > 0 {
			return ErrPending
		}
		if err := tx.Create(&item).Error; err != nil {
			return fmt.Errorf("create pemberhentian: %w", err)
		}
		targetID := item.ID
		note := "pengajuan pemberhentian: " + input.Alasan
		return auditWrite(ctx, tx, audit.Entry{
			ActorID:  actor,
			Action:   audit.ActionPemberhentianProposed,
			TargetID: &targetID,
			UnitID:   unitID,
			Note:     &note,
		})
	})
	return item, err
}

// Get membaca satu pengajuan/keputusan.
func (s GormStore) Get(ctx context.Context, id string) (models.Pemberhentian, error) {
	var item models.Pemberhentian
	if err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Pemberhentian{}, ErrNotFound
		}
		return models.Pemberhentian{}, fmt.Errorf("read pemberhentian: %w", err)
	}
	return item, nil
}

// List membaca pengajuan/keputusan sesuai filter dan cakupan.
func (s GormStore) List(ctx context.Context, options ListOptions) ([]models.Pemberhentian, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.Pemberhentian{})
	if options.UserID != "" {
		query = query.Where(`"userId" = ?`, options.UserID)
	}
	if options.Status != "" {
		query = query.Where(`"status" = ?`, options.Status)
	}
	if len(options.HomebaseIDs) > 0 {
		query = query.Where(`"userId" IN (SELECT "id" FROM "user" WHERE "homebaseId" IN ?)`, options.HomebaseIDs)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count pemberhentian: %w", err)
	}
	if options.Limit == 0 {
		options.Limit = 50
	}
	var items []models.Pemberhentian
	if err := query.Order(`"createdAt" DESC`).Limit(options.Limit).Offset(options.Offset).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list pemberhentian: %w", err)
	}
	return items, total, nil
}

// ErrNotFound menandai catatan pemberhentian yang tidak ditemukan.
var ErrNotFound = errors.New("pemberhentian not found")

// ErrPending menandai pengajuan yang masih menunggu persetujuan untuk
// pegawai yang sama.
var ErrPending = errors.New("pemberhentian sedang menunggu persetujuan")

// ErrNotPending menandai keputusan pada pengajuan yang tidak MENUNGGU.
var ErrNotPending = errors.New("pemberhentian tidak dalam status menunggu persetujuan")

// Decide mengubah status pengajuan. Pada DISETUJUI, menerbitkan SK PHK
// dengan nomor urut per tahun dan menulis kompensasi; pada DITOLAK, hanya
// mengubah status. Audit ditulis dalam transaksi.
func (s GormStore) Decide(ctx context.Context, actor, id string, approve bool, kompensasiBulan float64, note *string, unitID *string) (models.Pemberhentian, error) {
	var item models.Pemberhentian
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("lock pemberhentian: %w", err)
		}
		if item.Status != models.StatusMenunggu {
			return ErrNotPending
		}
		if approve {
			var seq int64
			if err := tx.Model(&models.SuratKeputusan{}).
				Where(`"jenis" = ? AND extract(year from "tanggalSk") = ?`, models.JenisSkPhk, item.Tanggal.Year()).
				Count(&seq).Error; err != nil {
				return fmt.Errorf("count sk phk: %w", err)
			}
			nomor := fmt.Sprintf("SK-PHK-%d-%04d", item.Tanggal.Year(), seq+1)

			catatan := ""
			if note != nil {
				catatan = *note
			}
			alasan := ""
			if item.Alasan != nil {
				alasan = *item.Alasan
			}
			sk := models.SuratKeputusan{
				Jenis:        models.JenisSkPhk,
				Penerbit:     models.TingkatSkYayasan,
				UserId:       &item.UserId,
				NomorSk:      nomor,
				TanggalSk:    item.Tanggal,
				MulaiBerlaku: &item.Tanggal,
				Metadata:     buildMetadata(map[string]any{"alasan": alasan, "kompensasiBulan": kompensasiBulan, "catatan": catatan}),
			}
			if err := tx.Create(&sk).Error; err != nil {
				return fmt.Errorf("create surat keputusan: %w", err)
			}
			hakTerakhir := nomor
			item.Keterangan = &catatan
			item.HakTerakhir = &hakTerakhir
			item.KompensasiBulan = kompensasiBulan
			if err := tx.Model(&item).Updates(map[string]any{
				"status":          models.StatusDisetujui,
				"keterangan":      catatan,
				"hakTerakhir":     hakTerakhir,
				"kompensasiBulan": kompensasiBulan,
			}).Error; err != nil {
				return fmt.Errorf("approve pemberhentian: %w", err)
			}

			// Pegawai dinonaktifkan pada saat setuju (`DEC-083`): slip tidak
			// dapat digenerate untuk periode setelah bulan pemberhentian; slip
			// bulan pemberhentian memuat komponen PESANGON. Slip DRAF untuk
			// periode setelah bulan pemberhentian yang sudah terlanjur
			// digenerate ditandai BATAL dalam transaksi yang sama (tidak
			// dapat digenerate ulang, tidak dapat dicetak, tidak dapat
			// dikonfirmasi pembayaran).
			if err := tx.Table(`"user"`).Where(`"id" = ?`, item.UserId).UpdateColumn("aktif", false).Error; err != nil {
				return fmt.Errorf("deactivate pegawai: %w", err)
			}
			periodAfter := time.Date(item.Tanggal.Year(), item.Tanggal.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
			result := tx.Model(&models.Gaji{}).
				Where(`"userId" = ? AND "status" = ? AND "periode" >= ?`, item.UserId, models.StatusSlipDraf, periodAfter).
				Updates(map[string]any{"status": models.StatusSlipBatal})
			if result.Error != nil {
				return fmt.Errorf("cancel slip DRAF: %w", result.Error)
			}
			batalCount := result.RowsAffected

			targetID := item.ID
			noteOut := fmt.Sprintf("disetujui; SK %s; kompensasi %.0f bulan; pegawai dinonaktifkan", nomor, kompensasiBulan)
			if batalCount > 0 {
				noteOut += fmt.Sprintf("; %d slip DRAF masa depan dibatalkan", batalCount)
			}
			return auditWrite(ctx, tx, audit.Entry{
				ActorID:  actor,
				Action:   audit.ActionPemberhentianApproved,
				TargetID: &targetID,
				UnitID:   unitID,
				Note:     &noteOut,
			})
		}

		catatan := ""
		if note != nil {
			catatan = *note
		}
		if err := tx.Model(&item).Updates(map[string]any{
			"status":     models.StatusDitolak,
			"keterangan": catatan,
		}).Error; err != nil {
			return fmt.Errorf("reject pemberhentian: %w", err)
		}
		targetID := item.ID
		noteOut := "ditolak"
		if note != nil {
			noteOut = *note
		}
		return auditWrite(ctx, tx, audit.Entry{
			ActorID:  actor,
			Action:   audit.ActionPemberhentianRejected,
			TargetID: &targetID,
			UnitID:   unitID,
			Note:     &noteOut,
		})
	})
	return item, err
}

// buildMetadata menyusun objek JSON dari map secara deterministic-ish; nilai
// yang tidak bisa dirender diabaikan.
func buildMetadata(values map[string]any) string {
	var sb strings.Builder
	sb.WriteString(`{`)
	first := true
	for k, v := range values {
		if !first {
			sb.WriteString(", ")
		}
		first = false
		var raw string
		switch t := v.(type) {
		case string:
			raw = fmt.Sprintf("%q", t)
		case float64:
			raw = fmt.Sprintf("%v", t)
		case int:
			raw = fmt.Sprintf("%d", t)
		case bool:
			raw = fmt.Sprintf("%v", t)
		default:
			continue
		}
		fmt.Fprintf(&sb, "%q: %s", k, raw)
	}
	sb.WriteString(`}`)
	return sb.String()
}
