package database

import (
	"errors"
	"net/url"
	"strings"

	"simpeg/backend/internal/timeutil"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type PrismaNamingStrategy struct {
	schema.NamingStrategy
}

var prismaTableNames = map[string]string{
	"User":              "user",
	"Homebase":          "Homebase",
	"Kategori":          "kategoriPegawai",
	"Delegasi":          "delegasi",
	"RiwayatLogin":      "riwayatLogin",
	"KomponenGaji":      "KomponenGaji",
	"Gaji":              "gaji",
	"GajiTunjangan":     "gajiTunjangan",
	"GajiPotongan":      "gajiPotongan",
	"HariLibur":         "HariLibur",
	"JadwalGaji":        "jadwalGaji",
	"SasaranKinerja":    "SasaranKinerja",
	"Pengumuman":        "Pengumuman",
	"Notifikasi":        "Notifikasi",
	"Lembur":            "Lembur",
	"TugasBelajar":      "TugasBelajar",
	"Survei":            "Survei",
	"PertanyaanSurvei":  "PertanyaanSurvei",
	"JawabanSurvei":     "JawabanSurvei",
	"PartisipasiSurvei": "PartisipasiSurvei",
	"Presensi":          "Presensi",
	"MesinAbsensi":      "MesinAbsensi",
	"AttendanceEvent":   "AttendanceEvent",
	"TemplateSidikJari": "TemplateSidikJari",
	"Role":              "Role",
	"RiwayatJabatan":    "RiwayatJabatan",
	"RiwayatPendidikan": "RiwayatPendidikan",
	"RiwayatGaji":       "RiwayatGaji",
	"Cuti":              "Cuti",
	"TerusanCuti":       "TerusanCuti",
	"SaldoCuti":         "SaldoCuti",
	"PeriodeSakit":      "PeriodeSakit",
	"Pesangon":          "Pesangon",
	"UjiKapabilitas":    "UjiKapabilitas",
	"MutasiJabatan":     "MutasiJabatan",
	"Dokumen":           "Dokumen",
	"Peringatan":        "Peringatan",
	"Pemberhentian":     "Pemberhentian",
	"Paklaring":         "paklaring",
	"PesanKeluar":       "pesanKeluar",
	"AuditLog":          "auditLog",
	"Session":           "session",
	"Account":           "account",
	"Verification":      "verification",
	"TwoFactor":         "twoFactor",
	"OutboxMessage":     "outboxMessage",
	"Pembayaran":        "pembayaran",
}

func (PrismaNamingStrategy) TableName(name string) string {
	if tableName, ok := prismaTableNames[name]; ok {
		return tableName
	}
	return name
}

func (PrismaNamingStrategy) ColumnName(_ string, name string) string {
	if name == "ID" {
		return "id"
	}
	if name == "" {
		return name
	}
	return strings.ToLower(name[:1]) + name[1:]
}

// OpenPostgres membuka koneksi PostgreSQL dengan kebijakan zona waktu.
//
// Penyimpanan memakai UTC: kolom `timestamp without time zone` pada skema
// acuan menyimpan jam dinding UTC, sehingga `now()` pada default database
// dan nilai yang ditulis GORM memakai basis yang sama. Tampilan ke pengguna
// dikonversi ke Asia/Jakarta oleh paket `timeutil`.
func OpenPostgres(dsn string) (*gorm.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("database DSN is required")
	}

	return gorm.Open(postgres.Open(withSessionTimeZone(dsn)), &gorm.Config{
		NamingStrategy: PrismaNamingStrategy{},
		TranslateError: true,
		NowFunc:        timeutil.NowUTC,
	})
}

// withSessionTimeZone memaksa parameter runtime `timezone=UTC` pada DSN.
//
// Parameter ini berlaku untuk setiap koneksi pada pool sehingga default
// `now()` dan `CURRENT_TIMESTAMP` selalu menghasilkan jam dinding UTC,
// sama dengan nilai yang ditulis GORM lewat `timeutil.NowUTC`.
func withSessionTimeZone(dsn string) string {
	trimmed := strings.TrimSpace(dsn)
	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return trimmed
		}
		query := parsed.Query()
		query.Set("timezone", "UTC")
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}
	if strings.Contains(strings.ToLower(trimmed), "timezone=") {
		return trimmed
	}
	return trimmed + " timezone=UTC"
}
