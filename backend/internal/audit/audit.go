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

	// ActionProfileUpdated menandai pembaruan profil mandiri (nama atau nomor
	// HP) oleh principal sendiri. Jejak hanya memuat nama kolom, bukan nilai.
	ActionProfileUpdated = "profile.updated"

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

	// ActionSlipPDFDownloaded menandai pengunduhan PDF slip gaji. Dicatat
	// best-effort untuk observabilitas/forensik; bukan pembatas jumlah
	// unduhan.
	ActionSlipPDFDownloaded = "payroll.slip.pdf.downloaded"
	// ActionPayrollNotified menandai pekerjaan terjadwal pemberitahuan slip
	// gaji H-1. Jejak hanya memuat jumlah, bukan penerima.
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
	// ActionCutiCanceled menandai pembatalan pengajuan cuti (MENUNGGU
	// atau DISETUJUI) oleh pemegang permission cuti:approve; saldo yang
	// terpotong dikembalikan untuk cuti tahunan.
	ActionCutiCanceled = "cuti.canceled"

	// ActionPemberhentianProposed menandai pengajuan pemberhentian dibuat.
	ActionPemberhentianProposed = "pemberhentian.proposed"
	// ActionPemberhentianApproved menandai pengajuan pemberhentian disetujui.
	ActionPemberhentianApproved = "pemberhentian.approved"
	// ActionPemberhentianRejected menandai pengajuan pemberhentian ditolak.
	ActionPemberhentianRejected = "pemberhentian.rejected"
	// ActionPensiunKonfigurasiCreated menandai penambahan konfigurasi
	// pensiun per kategori status (effective-dated).
	ActionPensiunKonfigurasiCreated = "pensiun.konfigurasi.created"
	// ActionPensiunKonfigurasiUpdated menandai perubahan konfigurasi pensiun.
	ActionPensiunKonfigurasiUpdated = "pensiun.konfigurasi.updated"
	// ActionPensiunKonfigurasiDeleted menandai penghapusan konfigurasi pensiun.
	ActionPensiunKonfigurasiDeleted = "pensiun.konfigurasi.deleted"
	// ActionPensiunKepesertaanCreated menandai penambahan kepesertaan DPLK pegawai.
	ActionPensiunKepesertaanCreated = "pensiun.kepesertaan.created"
	// ActionPensiunKepesertaanUpdated menandai perubahan kepesertaan DPLK pegawai.
	ActionPensiunKepesertaanUpdated = "pensiun.kepesertaan.updated"
	// ActionPensiunKepesertaanDeactivated menandai penonaktifan kepesertaan DPLK.
	ActionPensiunKepesertaanDeactivated = "pensiun.kepesertaan.deactivated"
	// ActionPerpanjanganCreated menandai perpanjangan kontrak pasca-BUP
	// dibuat (DEC-083).
	ActionPerpanjanganCreated = "perpanjangan.created"

	// ActionThrConfigUpdated menandai perubahan konfigurasi THR per tahun
	// (besaran, persentase, tanggal berlaku) oleh admin atau operator.
	ActionThrConfigUpdated = "payroll.thr.config.updated"

	// ActionThrConfigDeleted menandai penghapusan konfigurasi THR per tahun.
	ActionThrConfigDeleted = "payroll.thr.config.deleted"

	// ActionThrGenerated menandai ringkasan pembuatan THR sebuah pegawai
	// (slip penerimaan terpisah, tanpa potongan normal).
	ActionThrGenerated = "payroll.thr.generated"

	ActionJabatanCreated  = "jabatan.created"
	ActionJabatanUpdated  = "jabatan.updated"
	ActionJabatanArchived = "jabatan.archived"

	// Aksi pembuatan slip oleh mesin hitung.
	// ActionSlipPublished menandai slip gaji DRAF diterbitkan menjadi
	// TERBIT (prasyarat konfirmasi pembayaran dan pemberitahuan H-1).
	ActionSlipPublished = "payroll.slip.published"

	ActionSlipGenerated = "payroll.slip.generated"

	// Aksi area presensi administratif.
	ActionPinCreated       = "attendance.pin.created"
	ActionPinDeleted       = "attendance.pin.deleted"
	ActionPinImported      = "attendance.pin.imported"
	ActionPresensiUpserted = "attendance.presensi.upserted"
	ActionPresensiDerived  = "attendance.presensi.derived"

	// Aksi data skala gaji pokok yayasan.
	ActionSkalaCreated       = "skala.created"
	ActionSkalaUpdated       = "skala.updated"
	ActionSkalaDetailCreated = "skala.detail.created"
	ActionSkalaDetailUpdated = "skala.detail.updated"
	ActionSkalaDetailDeleted = "skala.detail.deleted"

	// Aksi honor mengajar per SKS (dosen LB).
	ActionHonorRateCreated     = "honor.rate.created"
	ActionHonorRateUpdated     = "honor.rate.updated"
	ActionHonorPeriodeAssigned = "honor.periode.assigned"
	ActionHonorPeriodeUpdated  = "honor.periode.updated"
	ActionHonorPeriodeRemoved  = "honor.periode.removed"

	// Aksi BPJS konfigurasi dan kepesertaan.
	ActionBpjsKonfigurasiCreated = "bpjs.konfigurasi.created"
	ActionBpjsKonfigurasiUpdated = "bpjs.konfigurasi.updated"
	ActionBpjsPesertaCreated     = "bpjs.peserta.created"
	ActionBpjsPesertaUpdated     = "bpjs.peserta.updated"

	// Aksi PPh 21 TER.
	ActionPPh21TerCreated = "pph21.ter.created"
	ActionPPh21TerUpdated = "pph21.ter.updated"
	ActionPPh21TerDeleted = "pph21.ter.deleted"
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
