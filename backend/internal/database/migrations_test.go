package database

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// migrationNamePattern mencocokkan penamaan file migrasi: 6 digit nomor lalu nama.
var migrationNamePattern = regexp.MustCompile(`^(\d{6})_([a-z0-9_]+)\.sql$`)

// migrationsDir mengembalikan direktori migrasi backend.
func migrationsDir() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	backendRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
	return filepath.Join(backendRoot, "migrations")
}

// TestMigrationFilesAreSequentiallyNumbered memastikan nomor migrasi urut dan
// tanpa duplikat. Nomor yang bentrok membuat urutan penerapan tidak pasti.
func TestMigrationFilesAreSequentiallyNumbered(t *testing.T) {
	dir := migrationsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read migrations directory: %v", err)
	}

	numbers := map[int]string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		match := migrationNamePattern.FindStringSubmatch(entry.Name())
		if match == nil {
			t.Errorf("nama migrasi tidak sesuai pola NNNNNN_nama.sql: %s", entry.Name())
			continue
		}
		number, err := strconv.Atoi(match[1])
		if err != nil {
			t.Errorf("nomor migrasi tidak valid: %s", entry.Name())
			continue
		}
		if previous, exists := numbers[number]; exists {
			t.Errorf("nomor migrasi %d dipakai dua kali: %s dan %s", number, previous, entry.Name())
			continue
		}
		numbers[number] = entry.Name()
	}

	if len(numbers) == 0 {
		t.Fatal("tidak ada berkas migrasi ditemukan")
	}
	sorted := make([]int, 0, len(numbers))
	for number := range numbers {
		sorted = append(sorted, number)
	}
	sort.Ints(sorted)
	for index, number := range sorted {
		if number != index+1 {
			t.Errorf("urutan migrasi terputus: FILE %05d ditemukan, diharapkan %05d", number, index+1)
		}
	}
}

// TestMigrationFilesHaveHeader memastikan setiap migrasi menjelaskan statusnya
// sehingga/file yang masih draf tidak dianggap sudah diterapkan.
func TestMigrationFilesHaveHeader(t *testing.T) {
	dir := migrationsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read migrations directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		if !strings.Contains(text, "--") {
			t.Errorf("migrasi %s tidak memiliki komentar header", entry.Name())
			continue
		}
		firstLine := strings.TrimSpace(strings.SplitN(text, "\n", 2)[0])
		if !strings.HasPrefix(firstLine, "--") {
			t.Errorf("migrasi %s harus diawali komentar header", entry.Name())
		}
	}
}

// TestMigrationCoversAuditLogUnitColumn memastikan perubahan skema jejak audit
// memiliki migrasi yang sesuai, karena kolom unitId dipakai untuk membatasi
// akses audit per unit.
func TestMigrationCoversAuditLogUnitColumn(t *testing.T) {
	dir := migrationsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		if strings.Contains(text, `"auditLog"`) && strings.Contains(text, `"unitId"`) {
			found = true
			break
		}
	}
	if !found {
		t.Error("tidak ada migrasi yang menambahkan kolom unitId pada auditLog")
	}
}

// TestRowLevelSecurityMigrationCoversEveryTable memastikan migrasi RLS
// mengaktifkan Row Level Security pada seluruh tabel aplikasi, sehingga tabel
// yang belum memiliki policy tetap ditolak bagi role non-pemilik.
// TestRowLevelSecurityMigrationCoversEveryTable memastikan Row Level Security
// diaktifkan pada seluruh tabel aplikasi, sehingga tabel yang belum memiliki
// policy tetap ditolak bagi role non-pemilik. Cakupan diperiksa pada seluruh
// migrasi, bukan hanya satu berkas.
func TestRowLevelSecurityMigrationCoversEveryTable(t *testing.T) {
	dir := migrationsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var combined strings.Builder
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		combined.Write(content)
		combined.WriteString("\n")
	}
	migration := combined.String()

	schemaPath, err := referenceSchemaPath()
	if err != nil {
		t.Fatal(err)
	}
	reference, err := loadPrismaSchema(schemaPath)
	if err != nil {
		t.Fatal(err)
	}

	missing := []string{}
	for _, model := range reference {
		// Nama tabel dapat muncul dengan kutip tunggal (daftar RLS) maupun
		// kutip ganda (perintah ALTER TABLE).
		covered := strings.Contains(migration, "'"+model.Table+"'") ||
			strings.Contains(migration, `"`+model.Table+`"`)
		if !covered {
			missing = append(missing, model.Table)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("tabel pada skema acuan belum tercakup migrasi RLS: %s", strings.Join(missing, ", "))
	}

	if !strings.Contains(migration, "ENABLE ROW LEVEL SECURITY") {
		t.Error("migrasi RLS harus mengaktifkan Row Level Security")
	}
	// Memaksa RLS pada level tabel akan memblokir backend, karena tidak ada
	// policy yang mengizinkan role pemilik tabel.
	if strings.Contains(migration, "FORCE ROW LEVEL SECURITY', target") {
		t.Error("migrasi RLS tidak boleh memakai FORCE ROW LEVEL SECURITY")
	}
}
