package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/outbox"
	"simpeg/backend/internal/payroll/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Reminder adalah slip gaji yang siap diberi pemberitahuan ketersediaan H+1.
//
// Reminder membawa tanggal gajian unit pemilik slip sehingga lapisan aplikasi
// dapat menentukan jatuh tempo pemberitahuan tanpa menulis ulang aturan
// kalender pada query.
type Reminder struct {
	// SlipID adalah slip gaji yang perlu diberi pemberitahuan.
	SlipID string `gorm:"column:slipId"`
	// UserID adalah pemilik slip.
	UserID string `gorm:"column:userId"`
	// Email adalah alamat email pemilik slip.
	Email string `gorm:"column:email"`
	// Nama adalah nama pemilik slip.
	Nama string `gorm:"column:nama"`
	// Periode adalah periode penggajian slip.
	Periode time.Time `gorm:"column:periode"`
	// TanggalGajian adalah tanggal gajian unit pemilik slip.
	TanggalGajian int `gorm:"column:tanggalGajian"`
}

// UpsertSchedule menyimpan jadwal tanggal gajian untuk satu unit.
//
// Jadwal ditulis bersama jejak audit dalam satu transaksi sehingga riwayat
// perubahan jadwal selalu ada. Unit yang tidak dikenal ditolak sebagai data
// tidak ditemukan.
func (s GormStore) UpsertSchedule(ctx context.Context, actorID string, input domain.ScheduleInput) (models.JadwalGaji, error) {
	var schedule models.JadwalGaji
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var units int64
		if err := tx.Model(&models.Homebase{}).Where("id = ?", input.UnitID).Count(&units).Error; err != nil {
			return fmt.Errorf("read payroll schedule unit: %w", err)
		}
		if units == 0 {
			return domain.ErrNotFound
		}

		record := models.JadwalGaji{
			UnitId:        input.UnitID,
			TanggalGajian: input.TanggalGajian,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "unitId"}},
			DoUpdates: clause.AssignmentColumns([]string{"tanggalGajian", "updatedAt"}),
		}).Create(&record).Error; err != nil {
			return fmt.Errorf("save payroll schedule: %w", err)
		}
		if err := tx.Where("unitId = ?", input.UnitID).First(&schedule).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("read payroll schedule: %w", err)
		}

		note := fmt.Sprintf("tanggal gajian unit %s: tanggal %d", input.UnitID, input.TanggalGajian)
		unitID := input.UnitID
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionPayrollScheduleUpdated,
			TargetID: &schedule.ID,
			UnitID:   &unitID,
			Note:     &note,
		})
	})
	if err != nil {
		return models.JadwalGaji{}, err
	}
	return schedule, nil
}

// ListSchedules membaca daftar jadwal tanggal gajian.
//
// unitIDs nil berarti seluruh unit (permission global). Slice kosong berarti
// tidak ada unit yang boleh dijangkau, sehingga hasilnya selalu kosong.
// Unit di luar daftar tidak terlihat, sehingga operator hanya membaca jadwal
// unit dalam cakupannya.
func (s GormStore) ListSchedules(ctx context.Context, unitIDs []string) ([]models.JadwalGaji, error) {
	if unitIDs != nil && len(unitIDs) == 0 {
		return []models.JadwalGaji{}, nil
	}
	query := s.DB.WithContext(ctx).Model(&models.JadwalGaji{}).Preload("Unit")
	if unitIDs != nil {
		query = query.Where("unitId IN ?", unitIDs)
	}
	var schedules []models.JadwalGaji
	if err := query.Order("\"unitId\"").Find(&schedules).Error; err != nil {
		return nil, fmt.Errorf("list payroll schedules: %w", err)
	}
	return schedules, nil
}

// FindDueReminders membaca slip gaji yang belum diberi pemberitahuan H+1.
//
// Slip yang memenuhi syarat adalah slip berstatus TERBIT atau TERBAYAR milik
// pegawai yang unitnya memiliki jadwal gajian. Periode slip dibatasi sampai
// periode berjalan karena slip periode berikutnya belum waktunya diberi
// pemberitahuan.
//
// Slip yang pesan pemberitahuannya sudah pernah diantrekan dilewati lewat
// kunci idempotensi antrean keluar, sehingga setiap slip hanya diberi satu
// pemberitahuan.
func (s GormStore) FindDueReminders(ctx context.Context, until time.Time) ([]Reminder, error) {
	var rows []Reminder
	err := s.DB.WithContext(ctx).Raw(`
		SELECT g."id" AS "slipId",
		       g."userId" AS "userId",
		       u."email" AS "email",
		       u."nama" AS "nama",
		       g."periode" AS "periode",
		       j."tanggalGajian" AS "tanggalGajian"
		FROM "gaji" g
		JOIN "user" u ON u."id" = g."userId"
		JOIN "jadwalGaji" j ON j."unitId" = u."homebaseId"
		WHERE g."status" IN (?, ?)
		  AND g."periode" <= ?
		  AND NOT EXISTS (
		      SELECT 1 FROM "outboxMessage" o
		      WHERE o."dedupeKey" = 'slip-available:' || g."id"::text
		  )
		ORDER BY g."periode" ASC, u."nama" ASC`,
		models.StatusSlipTerbit, models.StatusSlipTerbayar, until).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("find payroll reminders: %w", err)
	}
	return rows, nil
}

// EnqueueNotifications menulis pesan notifikasi ke antrean keluar.
//
// Pesan ditulis sekaligus lewat satu operasi batch sehingga jumlah statement
// tidak bertambah satu per slip. Pesan dengan kunci idempotensi yang sudah ada
// dilewati, sehingga pekerjaan harian boleh dijalankan berulang kali tanpa
// menghasilkan pemberitahuan ganda. Nilai balik adalah jumlah pesan yang
// benar-benar diantrekan.
func (s GormStore) EnqueueNotifications(ctx context.Context, notifications []domain.Notification) (int, error) {
	messages := make([]outbox.Message, 0, len(notifications))
	for _, notification := range notifications {
		if notification.Kind == "" {
			continue
		}
		messages = append(messages, outbox.NewMessage(notification.Kind, notification.DedupeKey, notification.Payload))
	}
	if len(messages) == 0 {
		return 0, nil
	}
	queue := outbox.GormStore{DB: s.DB}
	queued, err := queue.EnqueueBatch(ctx, messages)
	if err != nil {
		return 0, fmt.Errorf("enqueue payroll notice: %w", err)
	}
	return queued, nil
}
