package models

type JenisKelamin string

const (
	JenisKelaminL JenisKelamin = "L"
	JenisKelaminP JenisKelamin = "P"
)

type StatusPresensi string

const (
	StatusPresensiHadir StatusPresensi = "H"
	StatusPresensiIzin  StatusPresensi = "I"
	StatusPresensiSakit StatusPresensi = "S"
	StatusPresensiAlpa  StatusPresensi = "A"
	StatusPresensiCuti  StatusPresensi = "C"
	StatusPresensiLibur StatusPresensi = "LD"
)

type JenisCuti string

const (
	JenisCutiTahunan             JenisCuti = "TAHUNAN"
	JenisCutiPanjang             JenisCuti = "PANJANG"
	JenisCutiBersalin            JenisCuti = "BERSALIN"
	JenisCutiSakit               JenisCuti = "SAKIT"
	JenisCutiSakitBerkepanjangan JenisCuti = "SAKIT_BERKEPANJANGAN"
	JenisCutiIzinPenting         JenisCuti = "IZIN_PENTING"
	JenisCutiPolitik             JenisCuti = "POLITIK"
)

type SubJenisIzinPenting string

const (
	SubJenisMenikah         SubJenisIzinPenting = "MENIKAH"
	SubJenisMelahirkanIstri SubJenisIzinPenting = "MELAHIRKAN_ISTRI"
	SubJenisMenikahkanAnak  SubJenisIzinPenting = "MENIKAHKAN_ANAK"
	SubJenisKeluargaInti    SubJenisIzinPenting = "KELUARGA_INTI"
	SubJenisKeluargaLain    SubJenisIzinPenting = "KELUARGA_LAIN"
)

type StatusPersetujuan string

const (
	StatusMenunggu  StatusPersetujuan = "MENUNGGU"
	StatusDisetujui StatusPersetujuan = "DISETUJUI"
	StatusDitolak   StatusPersetujuan = "DITOLAK"
)

type TingkatTerusan string

const (
	TingkatPimpinan TingkatTerusan = "PIMPINAN"
	TingkatRektor   TingkatTerusan = "REKTOR"
	TingkatYayasan  TingkatTerusan = "YAYASAN"
)

type StatusSlipGaji string

const (
	StatusSlipDraf     StatusSlipGaji = "DRAF"
	StatusSlipTerbit   StatusSlipGaji = "TERBIT"
	StatusSlipTerbayar StatusSlipGaji = "TERBAYAR"
)

type HasilUjiKapabilitas string

const (
	HasilUjiLayakKembali HasilUjiKapabilitas = "LAYAK_KEMBALI"
	HasilUjiBerhenti     HasilUjiKapabilitas = "BERHENTI_HORMAT"
)

type ArahKomponenGaji string

const (
	ArahTunjangan ArahKomponenGaji = "TUNJANGAN"
	ArahPotongan  ArahKomponenGaji = "POTONGAN"
)

type JenisJabatan string

const (
	JenisJabatanStruktural JenisJabatan = "STRUKTURAL"
	JenisJabatanFungsional JenisJabatan = "FUNGSIONAL"
)

type JenisOrganisasi string

const (
	JenisOrganisasiUnit     JenisOrganisasi = "UNIT"
	JenisOrganisasiFakultas JenisOrganisasi = "FAKULTAS"
	JenisOrganisasiJurusan  JenisOrganisasi = "JURUSAN"
	JenisOrganisasiBiro     JenisOrganisasi = "BIRO"
	JenisOrganisasiBagian   JenisOrganisasi = "BAGIAN"
)

type CakupanDelegasi string

const (
	CakupanCuti   CakupanDelegasi = "CUTI"
	CakupanLembur CakupanDelegasi = "LEMBUR"
	CakupanSemua  CakupanDelegasi = "SEMUA"
)

type JenisMutasi string

const (
	JenisMutasiPromosi JenisMutasi = "PROMOSI"
	JenisMutasiMutasi  JenisMutasi = "MUTASI"
	JenisMutasiRotasi  JenisMutasi = "ROTASI"
	JenisMutasiDemosi  JenisMutasi = "DEMOSI"
)

type TingkatPeringatan string

const (
	PeringatanLisan       TingkatPeringatan = "LISAN"
	PeringatanTertulisI   TingkatPeringatan = "TERTULIS_I"
	PeringatanTertulisII  TingkatPeringatan = "TERTULIS_II"
	PeringatanTertulisIII TingkatPeringatan = "TERTULIS_III"
)

type JenisSanksi string

const (
	SanksiTeguran         JenisSanksi = "TEGURAN"
	SanksiPenundaanKGB    JenisSanksi = "PENUNDAAN_KGB"
	SanksiPotongTunjangan JenisSanksi = "POTONG_TUNJANGAN"
)

type JenisPemberhentian string

const (
	PemberhentianBUP            JenisPemberhentian = "BUP"
	PemberhentianAtasPermintaan JenisPemberhentian = "ATAS_PERMINTAAN_SENDIRI"
	PemberhentianHormat         JenisPemberhentian = "HORMAT"
	PemberhentianTidakHormat    JenisPemberhentian = "TIDAK_HORMAT"
)

type KanalPesan string

const (
	KanalWhatsApp KanalPesan = "WA"
	KanalEmail    KanalPesan = "EMAIL"
	KanalInApp    KanalPesan = "INAPP"
)

type StatusPesan string

const (
	StatusPesanDraf     StatusPesan = "DRAF"
	StatusPesanAntrean  StatusPesan = "ANTREAN"
	StatusPesanTerkirim StatusPesan = "TERKIRIM"
	StatusPesanGagal    StatusPesan = "GAGAL"
)
