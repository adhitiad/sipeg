package database

import (
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"simpeg/backend/internal/models"

	"gorm.io/gorm/schema"
)

// indexSpec menjelaskan satu indeks: kolom mana yang diindeks dan apakah unik.
type indexSpec struct {
	Name    string
	Columns []string
	Unique  bool
}

var (
	prismaIndexPattern  = regexp.MustCompile(`@@index\(\[([^\]]*)\]\s*(?:,\s*name:\s*"([^"]+)")?\s*\)`)
	prismaUniquePattern = regexp.MustCompile(`@@unique\(\[([^\]]*)\]\s*(?:,\s*name:\s*"([^"]+)")?\s*\)`)
)

// referenceIndexSpecs mengembalikan seluruh indeks yang dideklarasikan model
// pada skema acuan, termasuk indeks unik tingkat field.
func referenceIndexSpecs(model prismaModel, fieldColumns map[string]string) []indexSpec {
	specs := []indexSpec{}

	for _, match := range prismaIndexPattern.FindAllStringSubmatch(model.Raw, -1) {
		specs = append(specs, indexSpec{Name: match[2], Columns: resolveColumns(match[1], fieldColumns)})
	}
	for _, match := range prismaUniquePattern.FindAllStringSubmatch(model.Raw, -1) {
		specs = append(specs, indexSpec{Name: match[2], Columns: resolveColumns(match[1], fieldColumns), Unique: true})
	}
	// Indeks unik tingkat field ditulis pada baris field sebagai "@unique" atau "@id".
	for _, line := range strings.Split(model.Raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "@@") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		if !strings.Contains(trimmed, "@unique") && !strings.Contains(trimmed, "@id") {
			continue
		}
		column := fields[0]
		if mapped, ok := fieldColumns[column]; ok {
			column = mapped
		}
		specs = append(specs, indexSpec{Columns: []string{column}, Unique: true})
	}
	return specs
}

// resolveColumns memetakan nama field prisma menjadi nama kolom.
func resolveColumns(fieldList string, fieldColumns map[string]string) []string {
	columns := []string{}
	for _, field := range strings.Split(fieldList, ",") {
		name := strings.TrimSpace(field)
		if name == "" {
			continue
		}
		if mapped, ok := fieldColumns[name]; ok {
			name = mapped
		}
		columns = append(columns, name)
	}
	return columns
}

// gormIndexSpecs mengembalikan indeks yang dideklarasikan model GORM.
func gormIndexSpecs(model any) ([]indexSpec, error) {
	parsed, err := schema.Parse(model, &sync.Map{}, PrismaNamingStrategy{})
	if err != nil {
		return nil, err
	}
	specs := []indexSpec{}
	for _, index := range parsed.ParseIndexes() {
		columns := []string{}
		for _, field := range index.Fields {
			columns = append(columns, field.DBName)
		}
		specs = append(specs, indexSpec{
			Name:    index.Name,
			Columns: columns,
			Unique:  index.Class == "UNIQUE",
		})
	}
	return specs, nil
}

// sameIndex membandingkan dua indeks berdasarkan kolom dan keunikan.
// Nama indeks diabaikan karena GORM membuat nama sendiri untuk tag
// uniqueIndex, sedangkan prisma dapat memakai indeks tanpa nama.
func sameIndex(left, right indexSpec) bool {
	if left.Unique != right.Unique || len(left.Columns) != len(right.Columns) {
		return false
	}
	sortedLeft := append([]string(nil), left.Columns...)
	sortedRight := append([]string(nil), right.Columns...)
	sort.Strings(sortedLeft)
	sort.Strings(sortedRight)
	for index := range sortedLeft {
		if sortedLeft[index] != sortedRight[index] {
			return false
		}
	}
	return true
}

// TestGormIndexesMatchReferenceColumns memastikan setiap indeks GORM mengindeks
// kolom yang sama dengan indeks pada skema acuan. Skema acuan adalah sumber
// kebenaran; nama indeks GORM tidak perlu sama karena DDL dibuat oleh migrasi.
func TestGormIndexesMatchReferenceColumns(t *testing.T) {
	path, err := referenceSchemaPath()
	if err != nil {
		t.Fatal(err)
	}
	reference, err := loadPrismaSchemaWithIndexes(path)
	if err != nil {
		t.Fatalf("cannot read reference schema: %v", err)
	}

	for _, model := range models.All() {
		typeName := reflect.TypeOf(model).Elem().Name()
		referenceModel, ok := reference[typeName]
		if !ok {
			t.Errorf("model GORM %s tidak ada pada skema acuan", typeName)
			continue
		}

		gormSpecs, err := gormIndexSpecs(model)
		if err != nil {
			t.Errorf("model %s: %v", typeName, err)
			continue
		}
		for _, gormSpec := range gormSpecs {
			matched := false
			for _, referenceSpec := range referenceModel.Indexes {
				if sameIndex(gormSpec, referenceSpec) {
					matched = true
					break
				}
			}
			if !matched {
				t.Errorf("model %s: indeks GORM %q pada kolom %v tidak memiliki padanan di skema acuan",
					typeName, gormSpec.Name, gormSpec.Columns)
			}
		}
	}
}

// TestCriticalQueryPathsHaveIndexes memastikan kolom yang dipakai jalur query
// utama memiliki indeks pendukung pada skema acuan. Tanpa indeks, penyaring
// cakupan unit dan pencarian sesi menjadi pemindaian penuh.
func TestCriticalQueryPathsHaveIndexes(t *testing.T) {
	path, err := referenceSchemaPath()
	if err != nil {
		t.Fatal(err)
	}
	reference, err := loadPrismaSchemaWithIndexes(path)
	if err != nil {
		t.Fatal(err)
	}

	critical := []struct {
		model   string
		column  string
		purpose string
	}{
		{"User", "homebaseId", "penyaring cakupan unit pada daftar pegawai"},
		{"User", "email", "pencarian identitas unik saat impor"},
		{"Session", "token", "resolusi sesi pada setiap permintaan"},
		{"Homebase", "parentId", "penelusuran pohon unit untuk cakupan turunan"},
		{"AuditLog", "unitId", "penyaring cakupan unit pada jejak audit"},
		{"AuditLog", "createdAt", "urutan jejak audit terbaru"},
	}

	for _, item := range critical {
		model, ok := reference[item.model]
		if !ok {
			t.Errorf("model %s tidak ada pada skema acuan", item.model)
			continue
		}
		indexed := false
		for _, spec := range model.Indexes {
			for _, column := range spec.Columns {
				if column == item.column {
					indexed = true
				}
			}
		}
		if !indexed {
			t.Errorf("kolom %s.%s tidak terindeks pada skema acuan (%s)", item.model, item.column, item.purpose)
		}
	}
}

// loadPrismaSchemaWithIndexes mengurai skema acuan termasuk indeks per model.
func loadPrismaSchemaWithIndexes(path string) (map[string]prismaModel, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(content)

	enums := map[string]bool{}
	for _, match := range enumPattern.FindAllStringSubmatch(text, -1) {
		enums[match[1]] = true
	}

	parsed := map[string]prismaModel{}
	for _, match := range modelBlockPattern.FindAllStringSubmatch(text, -1) {
		name, body := match[1], match[2]
		model := prismaModel{Name: name, Table: name, Columns: map[string]bool{}, Raw: body}
		fieldColumns := map[string]string{}

		for _, rawLine := range strings.Split(body, "\n") {
			line := strings.TrimSpace(commentPattern.ReplaceAllString(rawLine, ""))
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "@@map(") {
				if mapped := mapAttribute.FindStringSubmatch(line); mapped != nil {
					model.Table = mapped[1]
				}
				continue
			}
			if strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "///") {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			fieldName, fieldType := fields[0], fields[1]
			baseType := strings.TrimSuffix(strings.TrimSuffix(fieldType, "?"), "[]")
			if index := strings.Index(baseType, "("); index >= 0 {
				baseType = baseType[:index]
			}
			if !scalarTypes[baseType] && !enums[baseType] {
				continue
			}
			column := fieldName
			if mapped := mapAttribute.FindStringSubmatch(line); mapped != nil {
				column = mapped[1]
			}
			model.Columns[column] = true
			fieldColumns[fieldName] = column
		}

		model.Indexes = referenceIndexSpecs(model, fieldColumns)
		parsed[name] = model
	}
	return parsed, nil
}
