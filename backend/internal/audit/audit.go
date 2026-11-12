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
	ActionSessionIssued           = "session.issued"
	ActionSessionRevoked          = "session.revoked"
	ActionAuditAccessed           = "audit.accessed"

	// ActionAuthorizationDenied menandai akses yang ditolak karena role tidak
	// memegang izin yang diminta. Jejak hanya memuat aktor, izin, dan pola
	// route, bukan isi sumber daya yang ditolak.
	ActionAuthorizationDenied = "authz.denied"

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

	// ActionAuditExported menandai ekspor jejak audit ke berkas CSV.
	// Jejak hanya memuat aktor dan jumlah baris, bukan isi jejak yang diekspor.
	ActionAuditExported = "audit.exported"

	// ActionSLOEvaluated menandai hasil pemeriksaan target SLO terjadwal.
	ActionSLOEvaluated = "slo.evaluated"

	// Aksi data keluarga pegawai.
	ActionKeluargaCreated  = "keluarga.created"
	ActionKeluargaUpdated  = "keluarga.updated"
	ActionKeluargaArchived = "keluarga.archived"

	// Aksi data riwayat golongan.
	ActionGolonganCreated  = "golongan.created"
	ActionGolonganUpdated  = "golongan.updated"
	ActionGolonganArchived = "golongan.archived"

	// Aksi data arsip dokumen pegawai.
	ActionArsipCreated  = "arsip.created"
	ActionArsipUpdated  = "arsip.updated"
	ActionArsipArchived = "arsip.archived"

	// Aksi data riwayat sertifikasi.
	ActionSertifikasiCreated  = "sertifikasi.created"
	ActionSertifikasiUpdated  = "sertifikasi.updated"
	ActionSertifikasiArchived = "sertifikasi.archived"

	// Aksi data riwayat pelatihan.
	ActionPelatihanCreated  = "pelatihan.created"
	ActionPelatihanUpdated  = "pelatihan.updated"
	ActionPelatihanArchived = "pelatihan.archived"

	// Aksi data pemetaan SISTER/PDDIKTI.
	ActionSisterCreated  = "sister.created"
	ActionSisterUpdated  = "sister.updated"
	ActionSisterArchived = "sister.archived"

	// Aksi data riwayat penghargaan.
	ActionPenghargaanCreated  = "penghargaan.created"
	ActionPenghargaanUpdated  = "penghargaan.updated"
	ActionPenghargaanArchived = "penghargaan.archived"

	// Aksi data komponen tunjangan.
	ActionTunjanganCreated  = "tunjangan.created"
	ActionTunjanganUpdated  = "tunjangan.updated"
	ActionTunjanganArchived = "tunjangan.archived"

	// Aksi data komponen potongan.
	ActionPotonganCreated  = "potongan.created"
	ActionPotonganUpdated  = "potongan.updated"
	ActionPotonganArchived = "potongan.archived"

	// Aksi data pengajuan cuti.
	ActionCutiCreated  = "cuti.created"
	ActionCutiUpdated  = "cuti.updated"
	ActionCutiApproved = "cuti.approved"
	ActionCutiRejected = "cuti.rejected"
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
	record := newRecord(entry)
	if err := tx.WithContext(ctx).Create(&record).Error; err != nil {
		return err
	}
	return nil
}

// WriteBatch menulis banyak entri audit dalam satu operasi.
//
// Jalur volume tinggi, misalnya impor massal pegawai, memakai fungsi ini agar
// jumlah statement tidak bertambah satu per baris. Entri ditulis dalam beberapa
// batch agar batas parameter PostgreSQL tidak terlampaui.
func WriteBatch(ctx context.Context, tx *gorm.DB, entries []Entry) error {
	if tx == nil {
		return errors.New("audit write requires a database handle")
	}
	if len(entries) == 0 {
		return nil
	}
	for start := 0; start < len(entries); start += auditBatchSize {
		end := start + auditBatchSize
		if end > len(entries) {
			end = len(entries)
		}
		records := make([]models.AuditLog, 0, end-start)
		for _, entry := range entries[start:end] {
			records = append(records, newRecord(entry))
		}
		if err := tx.WithContext(ctx).Create(&records).Error; err != nil {
			return err
		}
	}
	return nil
}

// auditBatchSize membatasi jumlah baris per statement INSERT agar jumlah
// parameter binding tetap di bawah batas PostgreSQL.
const auditBatchSize = 500

// newRecord memetakan entri audit menjadi baris tabel auditLog.
func newRecord(entry Entry) models.AuditLog {
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
	return record
}
