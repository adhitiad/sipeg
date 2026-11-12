package main

import (
	"context"
	"fmt"
	"time"

	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"

	"gorm.io/gorm"
)

// seedPlan memuat skala data sintetis untuk pengukuran.
type seedPlan struct {
	Employees int
	Months    int
	AuditLogs int
}

// seedEmailDomain dipakai untuk alamat email sintetis. Domain `.test` tidak
// dapat menerima email sehingga data uji tidak pernah sampai ke kotak masuk
// orang sungguhan.
const seedEmailDomain = "simpeg-perf.test"

// perfAdminEmail adalah alamat admin sintetis yang dipakai pengukuran
// end-to-end.
const perfAdminEmail = "admin@" + seedEmailDomain

// seedBatchSize adalah ukuran batch insert. Nilai ini menjaga statement
// INSERT tetap di bawah batas parameter PostgreSQL saat data sintetis
// bertambah banyak.
const seedBatchSize = 500

// runSeed mengisi database uji dengan data sintetis yang mewakili skala
// produksi: pohon unit organisasi, pegawai, slip gaji, dan jejak audit.
func runSeed(ctx context.Context, args []string) error {
	plan, err := parseSeedPlan(args)
	if err != nil {
		return err
	}
	db, err := openDB(ctx)
	if err != nil {
		return err
	}
	started := time.Now()
	if err := truncateAppTables(ctx, db); err != nil {
		return err
	}
	roles, err := seedRoles(ctx, db)
	if err != nil {
		return err
	}
	units, err := seedUnits(ctx, db)
	if err != nil {
		return err
	}
	categories, err := seedCategories(ctx, db)
	if err != nil {
		return err
	}
	employees, err := seedEmployees(ctx, db, plan, roles, units, categories)
	if err != nil {
		return err
	}
	if err := seedPayroll(ctx, db, plan, employees); err != nil {
		return err
	}
	if err := seedAuditLogs(ctx, db, plan, roles, units, employees); err != nil {
		return err
	}
	if err := seedCalendar(ctx, db, units); err != nil {
		return err
	}
	fmt.Printf("seed selesai dalam %s: %d pegawai, %d unit, %d bulan slip, %d jejak audit\n",
		time.Since(started).Round(time.Millisecond), len(employees), len(units), plan.Months, plan.AuditLogs)
	return nil
}

// truncateAppTables mengosongkan tabel aplikasi agar seeding dapat diulang.
func truncateAppTables(ctx context.Context, db *gorm.DB) error {
	const statement = `TRUNCATE TABLE
		"user", "Homebase", "Role", "kategoriPegawai", "auditLog", "gaji",
		"gajiTunjangan", "gajiPotongan", "pembayaran", "jadwalGaji",
		"HariLibur", "session", "account", "outboxMessage", "skpPeriode",
		"skpTargetKinerja", "logAktivitasHarian", "penilaianKinerjaAkhir",
		"periodeAkademik", "evaluasiBkd", "tridharmaPendidikan",
		"tridharmaPenelitian", "tridharmaPengabdian", "tridharmaPenunjang"
		RESTART IDENTITY CASCADE`
	if err := db.WithContext(ctx).Exec(statement).Error; err != nil {
		return fmt.Errorf("kosongkan tabel: %w", err)
	}
	return nil
}

// seedRoles membuat empat role yang dipakai matriks permission.
func seedRoles(ctx context.Context, db *gorm.DB) (map[string]string, error) {
	definitions := []struct{ name, description string }{
		{"admin", "Administrator SIMPEG"},
		{"operator", "Operator unit organisasi"},
		{"atasan", "Atasan unit organisasi"},
		{"pegawai", "Pegawai"},
	}
	roles := make([]models.Role, 0, len(definitions))
	for _, definition := range definitions {
		roles = append(roles, models.Role{Name: definition.name, Description: &definition.description})
	}
	if err := db.WithContext(ctx).Create(&roles).Error; err != nil {
		return nil, fmt.Errorf("buat role: %w", err)
	}
	result := make(map[string]string, len(roles))
	for _, role := range roles {
		result[role.Name] = role.ID
	}
	return result, nil
}

// seedUnits membuat pohon unit: satu akar, delapan fakultas, jurusan, dan
// bagian di bawahnya.
func seedUnits(ctx context.Context, db *gorm.DB) ([]models.Homebase, error) {
	root := models.Homebase{
		Name:        "Universitas Subang",
		Jenis:       models.JenisOrganisasiUnit,
		Description: stringPtr("Unit akar untuk pengukuran"),
		Aktif:       true,
	}
	if err := db.WithContext(ctx).Create(&root).Error; err != nil {
		return nil, fmt.Errorf("buat unit akar: %w", err)
	}
	units := []models.Homebase{root}
	for faculty := 0; faculty < 8; faculty++ {
		parent := root.ID
		for _, child := range []struct {
			name  string
			jenis models.JenisOrganisasi
			count int
		}{
			{fmt.Sprintf("Fakultas %d", faculty+1), models.JenisOrganisasiFakultas, 4},
			{fmt.Sprintf("Biro %d", faculty+1), models.JenisOrganisasiBiro, 2},
		} {
			for index := 0; index < child.count; index++ {
				unit := models.Homebase{
					Name:  fmt.Sprintf("%s %d", child.name, index+1),
					Jenis: child.jenis,
					Aktif: true,
				}
				unit.ParentId = &parent
				if err := db.WithContext(ctx).Create(&unit).Error; err != nil {
					return nil, fmt.Errorf("buat unit: %w", err)
				}
				units = append(units, unit)
				parent = unit.ID
			}
		}
	}
	return units, nil
}

// seedCategories membuat kategori pegawai acuan.
func seedCategories(ctx context.Context, db *gorm.DB) ([]models.Kategori, error) {
	names := []string{"Dosen", "Tenaga Kependidikan", "Peneliti", "Asisten Dosen", "Pranata", "Laboran"}
	categories := make([]models.Kategori, 0, len(names))
	for index, name := range names {
		categories = append(categories, models.Kategori{
			Kode:        fmt.Sprintf("K%02d", index+1),
			Nama:        name,
			UsiaPensiun: 58,
			Aktif:       true,
		})
	}
	if err := db.WithContext(ctx).Create(&categories).Error; err != nil {
		return nil, fmt.Errorf("buat kategori: %w", err)
	}
	return categories, nil
}

// stringPtr mengembalikan pointer ke string.
func stringPtr(value string) *string { return &value }

// timePtr mengembalikan pointer ke waktu.
func timePtr(value time.Time) *time.Time { return &value }

// nowUTC mengembalikan waktu saat ini dalam UTC.
func nowUTC() time.Time { return timeutil.NowUTC() }
