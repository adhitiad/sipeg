// Pemeriksaan aturan cuti yang membutuhkan data pegawai (masa kerja dan
// saldo tahunan) dan pemeriksaan tumpang tindih. Pemeriksaan bersifat
// best-effort pada pengajuan; penegakan akhir dilakukan ulang secara
// atomik saat persetujuan.
package app

import (
	"context"

	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/models"
)

// checkLeavePolicy memeriksa aturan yang tidak dapat diputuskan murni dari
// isi pengajuan: masa kerja minimum, kecukupan saldo cuti tahunan, dan
// tumpang tindih dengan pengajuan MENUNGGU/DISETUJUI milik pegawai yang
// sama.
//
// Bila tanggal masuk pegawai tidak terisi, syarat masa kerja dilewati
// (dicatat sebagai asumsi kerja di `DEC-079`). Untuk cuti panjang, syarat
// 6 tahun dapat digantikan ibadah haji (UntukHaji).
func (s *LeaveService) checkLeavePolicy(ctx context.Context, item models.Cuti, excludeID string) error {
	switch item.Jenis {
	case models.JenisCutiTahunan:
		start, err := s.repo.EmployeeStartDate(ctx, item.UserId)
		if err != nil {
			return err
		}
		if !domain.MasaKerjaCukup(start, domain.MasaKerjaCutiTahunan, item.TanggalMulai) {
			return domain.ErrValidation
		}
		days := domain.HariKerjaAntara(item.TanggalMulai, item.TanggalSelesai)
		row, err := s.repo.SaldoCuti(ctx, item.UserId, item.TanggalMulai.Year())
		if err != nil {
			return err
		}
		if days <= 0 || domain.SisaCuti(row) < days {
			return domain.ErrValidation
		}
	case models.JenisCutiPanjang:
		if !item.UntukHaji {
			start, err := s.repo.EmployeeStartDate(ctx, item.UserId)
			if err != nil {
				return err
			}
			if !domain.MasaKerjaPanjangSufficient(start, item.TanggalMulai) {
				return domain.ErrValidation
			}
		}
	}
	return s.checkOverlap(ctx, item, excludeID)
}

// checkKategoriCuti menegakkan aturan hak cuti per kategori status pegawai
// (DEC-092). Untuk kategori yang membatasi (DOSEN_LB, DOSEN_DPK), jenis
// cuti tahunan, panjang, dan bersalin ditolak. Kategori yang belum ditetapkan
// (kode kosong) tidak membatasi (asumsi kerja, backward compatible).
func (s *LeaveService) checkKategoriCuti(ctx context.Context, item models.Cuti) error {
	kode, err := s.repo.EmployeeKategoriStatusKode(ctx, item.UserId)
	if err != nil {
		return err
	}
	if !domain.JenisCutiDiizinkanUntukKategori(item.Jenis, kode) {
		return domain.ErrValidation
	}
	return nil
}

// checkOverlap menolak pengajuan yang tumpang tindih dengan pengajuan
// MENUNGGU/DISETUJUI milik pegawai yang sama.
func (s *LeaveService) checkOverlap(ctx context.Context, item models.Cuti, excludeID string) error {
	overlap, err := s.repo.OverlappingCuti(ctx, item.UserId, item.TanggalMulai, item.TanggalSelesai, excludeID)
	if err != nil {
		return err
	}
	if overlap > 0 {
		return domain.ErrConflict
	}
	return nil
}
