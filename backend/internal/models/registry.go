package models

// All mengembalikan seluruh model GORM yang memetakan tabel pada skema acuan.
//
// IDModel dan BaseModel sengaja tidak disertakan: keduanya adalah struct dasar
// yang di-embed, bukan tabel. Registry ini dipakai audit paritas skema untuk
// memastikan setiap model GORM punya pasangan pada prisma/schema.prisma.
func All() []any {
	return []any{
		&User{}, &Homebase{}, &Kategori{}, &Delegasi{}, &RiwayatLogin{},
		&KomponenGaji{}, &Gaji{}, &GajiTunjangan{}, &GajiPotongan{}, &HariLibur{},
		&JadwalGaji{}, &SasaranKinerja{}, &Pengumuman{}, &Notifikasi{},
		&Lembur{}, &TugasBelajar{}, &Survei{}, &PertanyaanSurvei{}, &JawabanSurvei{},
		&PartisipasiSurvei{}, &Presensi{}, &MesinAbsensi{}, &AttendanceEvent{},
		&TemplateSidikJari{}, &Role{}, &RiwayatJabatan{}, &RiwayatPendidikan{},
		&RiwayatGaji{}, &Cuti{}, &TerusanCuti{}, &SaldoCuti{}, &PeriodeSakit{},
		&Pesangon{}, &UjiKapabilitas{}, &MutasiJabatan{}, &Dokumen{}, &Peringatan{},
		&Pemberhentian{}, &Paklaring{}, &PesanKeluar{}, &AuditLog{},
		&Session{}, &Account{}, &Verification{}, &TwoFactor{},
		&OutboxMessage{}, &Pembayaran{},
	}
}
