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

// JenisEviden menandai bentuk bukti yang menyertai pengajuan cuti.
type JenisEviden string

const (
	// JenisEvidenSuratDokter adalah surat dokter, wajib untuk cuti sakit dan
	// cuti melahirkan.
	JenisEvidenSuratDokter JenisEviden = "SURAT_DOKTER"
	// JenisEvidenFoto adalah foto atau tangkapan layar bertimestamp, dipakai
	// untuk cuti harian (izin penting).
	JenisEvidenFoto JenisEviden = "FOTO"
	// JenisEvidenLainnya adalah bentuk bukti lain yang dicatat beserta waktu
	// dan keterangan.
	JenisEvidenLainnya JenisEviden = "LAINNYA"
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

type JenisHubunganKeluarga string

const (
	HubunganSuamiIstri JenisHubunganKeluarga = "SUAMI_ISTRI"
	HubunganAnak       JenisHubunganKeluarga = "ANAK"
)

type StatusTanggungan string

const (
	TanggunganYa    StatusTanggungan = "TANGGUNGAN"
	TanggunganTidak StatusTanggungan = "BUKAN_TANGGUNGAN"
)

type JenisTunjangan string

const (
	TunjanganStruktural   JenisTunjangan = "STRUKTURAL"
	TunjanganFungsional   JenisTunjangan = "FUNGSIONAL"
	TunjanganBeras        JenisTunjangan = "BERAS"
	TunjanganMakanan      JenisTunjangan = "MAKANAN"
	TunjanganTransportasi JenisTunjangan = "TRANSPORTASI"
)

type JenisPotongan string

const (
	PotonganPPH21               JenisPotongan = "PPH21"
	PotonganBPJSKesehatan       JenisPotongan = "BPJS_KESEHATAN"
	PotonganBPJSKetenagakerjaan JenisPotongan = "BPJS_KETENAGAKERJAAN"
	PotonganKasBon              JenisPotongan = "KAS BON"
	PotonganKoperasi            JenisPotongan = "KOPERASI"
)

type JenisDokumenArsip string

const (
	DokumenKTP                   JenisDokumenArsip = "KTP"
	DokumenNPWP                  JenisDokumenArsip = "NPWP"
	DokumenKartuKeluarga         JenisDokumenArsip = "KARTU_KELUARGA"
	DokumenRekeningBank          JenisDokumenArsip = "REKENING_BANK"
	DokumenSKPengangkatanYayasan JenisDokumenArsip = "SK_PENGANGKATAN_YAYASAN"
	DokumenSKMendikbud           JenisDokumenArsip = "SK_MENDIKBUD"
	DokumenPaktaIntegritas       JenisDokumenArsip = "PAKTA_INTEGRITAS"
	DokumenLainnya               JenisDokumenArsip = "LAINNYA"
	DokumenFoto                  JenisDokumenArsip = "FOTO"
	DokumenIjazahTerakhir        JenisDokumenArsip = "IJAHAN_TERAKHIR"
	DokumenSKKepegawaianKontrak  JenisDokumenArsip = "SK_KEPEGAWAIAN_KONTRAK"
	DokumenBPJSKetenagakerjaan   JenisDokumenArsip = "BPJS_KETENAGAKERJAAN"
	DokumenBPJSkesehatan         JenisDokumenArsip = "BPJS_KESEHATAN"
	DokumenSertifikatSerdos      JenisDokumenArsip = "SERTIFIKAT_SERDOS"
)

type JenisSertifikasi string

const (
	SertifikasiSerdos     JenisSertifikasi = "SERDOS"
	SertifikasiProfesi    JenisSertifikasi = "PROFESI"
	SertifikasiKompetensi JenisSertifikasi = "KOMPETENSI"
	SertifikasiLainnya    JenisSertifikasi = "LAINNYA"
)

type StatusSertifikasi string

const (
	SertifikasiAktif      StatusSertifikasi = "AKTIF"
	SertifikasiKedaluarsa StatusSertifikasi = "KEDALUWARSA"
)

type JenisPelatihan string

const (
	JenisPelatihanDiklat     JenisPelatihan = "DIKLAT"
	JenisPelatihanLokakarya  JenisPelatihan = "LOKAKARYA"
	JenisPelatihanSoftSkills JenisPelatihan = "SOFT_SKILLS"
	JenisPelatihanLainnya    JenisPelatihan = "LAINNYA"
)

type SistemEksternal string

const (
	SistemSISTER  SistemEksternal = "SISTER"
	SistemPDDIKTI SistemEksternal = "PDDIKTI"
	SistemLainnya SistemEksternal = "LAINNYA"
)

type StatusSinkronisasi string

const (
	SinkronisasiAktif    StatusSinkronisasi = "AKTIF"
	SinkronisasiNonaktif StatusSinkronisasi = "NONAKTIF"
)

type JenisPenghargaan string

const (
	PenghargaanDosenTeladan JenisPenghargaan = "DOSEN_TELOHAN"
	PenghargaanSatyalancana JenisPenghargaan = "SATYALANCANA"
	PenghargaanLainnya      JenisPenghargaan = "LAINNYA"
)
