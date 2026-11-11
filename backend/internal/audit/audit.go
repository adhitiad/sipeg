// Package audit menyediakan penulisan jejak perubahan yang konsisten untuk
// seluruh paket backend. Jejak audit bersifat append-only: tidak ada jalur
// hapus atau ubah di paket ini maupun di paket lain.
package audit

import (
	"context"
	"errors"

	"simpeg/backend/internal/models"

	"gorm.io/gorm"
)

// Nama aksi yang dicatat backend.
const (
	ActionEmployeeCreated     = "employee.created"
	ActionEmployeeUpdated     = "employee.updated"
	ActionEmployeeArchived    = "employee.archived"
	ActionEmployeeImported    = "employee.imported"
	ActionEmployeeRoleUpdated = "employee.role.updated"

	ActionOrganizationCreated  = "organization.created"
	ActionOrganizationUpdated  = "organization.updated"
	ActionOrganizationArchived = "organization.archived"

	ActionAttendanceEventIngested = "attendance.event.ingested"
	ActionSessionRevoked          = "session.revoked"
	ActionAuditAccessed           = "audit.accessed"

	// ActionPaymentConfirmed menandai konfirmasi pembayaran gaji yang tercatat.
	// Jejak ini tidak memuat nominal gaji.
	ActionPaymentConfirmed = "payroll.payment.confirmed"

	// ActionPayrollScheduleUpdated menandai perubahan jadwal tanggal gajian
	// sebuah unit.
	ActionPayrollScheduleUpdated = "payroll.schedule.updated"

	// ActionPayrollNotified menandai pekerjaan terjadwal pemberitahuan slip
	// gaji H+1. Jejak hanya memuat jumlah, bukan penerima.
	ActionPayrollNotified = "payroll.notified"

	// ActionHolidayRefreshed menandai sinkronisasi ulang kalender hari libur
	// sebuah tahun. Jejak hanya memuat jumlah entri, bukan isinya.
	ActionHolidayRefreshed = "holiday.refreshed"

	// ActionAuditCleaned menandai pembersihan jejak audit yang kedaluwarsa.
	ActionAuditCleaned = "audit.cleaned"
)

// Entry menggambarkan satu catatan jejak audit.
type Entry struct {
	// ActorID adalah pelaku aksi; kosong untuk aksi sistem.
	ActorID string
	// Action adalah nama aksi stabil.
	Action string
	// TargetID adalah id objek yang terpengaruh, bila ada.
	TargetID *string
	// UnitID adalah unit cakupan saat aksi terjadi. Akses audit dibatasi
	// berdasarkan kolom ini, bukan berdasarkan aktor.
	UnitID *string
	// Note adalah keterangan minimal tanpa isi data pribadi.
	Note *string
	// IP adalah alamat pemohon bila diketahui.
	IP *string
}

// Write menulis satu entri audit. tx boleh nil untuk memakai koneksi langsung.
func Write(ctx context.Context, tx *gorm.DB, entry Entry) error {
	if tx == nil {
		return errors.New("audit write requires a database handle")
	}
	record := models.AuditLog{
		Aksi:       entry.Action,
		Sasaran:    entry.TargetID,
		Keterangan: entry.Note,
		Ip:         entry.IP,
		UnitId:     entry.UnitID,
	}
	if entry.ActorID != "" {
		actor := entry.ActorID
		record.AktorId = &actor
	}
	if err := tx.WithContext(ctx).Create(&record).Error; err != nil {
		return err
	}
	return nil
}
