package database

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	"simpeg/backend/internal/models"

	"gorm.io/gorm/schema"
)

// prismaModel adalah hasil parsing satu blok "model" pada schema acuan.
type prismaModel struct {
	Name    string
	Table   string
	Columns map[string]bool
	// Raw menyimpan isi blok model untuk parsing indeks.
	Raw string
	// Indexes memuat seluruh indeks yang dideklarasikan model pada skema acuan.
	Indexes []indexSpec
}

// scalarTypes adalah tipe skalar Prisma; field bertipe ini adalah kolom.
var scalarTypes = map[string]bool{
	"String": true, "Int": true, "Float": true, "Boolean": true,
	"DateTime": true, "Json": true, "Decimal": true, "BigInt": true,
	"Bytes": true, "Unsupported": true,
}

var (
	modelBlockPattern = regexp.MustCompile(`(?ms)^model\s+(\w+)\s*\{(.*?)^\}`)
	enumPattern       = regexp.MustCompile(`(?m)^enum\s+(\w+)\s*\{`)
	mapAttribute      = regexp.MustCompile(`@map\("([^"]+)"\)`)
	commentPattern    = regexp.MustCompile(`//.*$`)
)

// loadPrismaSchema membaca dan mengurai prisma/schema.prisma.
func loadPrismaSchema(path string) (map[string]prismaModel, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(content)

	enums := map[string]bool{}
	for _, match := range enumPattern.FindAllStringSubmatch(text, -1) {
		enums[match[1]] = true
	}

	models := map[string]prismaModel{}
	for _, match := range modelBlockPattern.FindAllStringSubmatch(text, -1) {
		name, body := match[1], match[2]
		model := prismaModel{Name: name, Table: name, Columns: map[string]bool{}, Raw: body}

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
			if strings.HasPrefix(line, "@@") {
				continue
			}
			if strings.HasPrefix(line, "///") {
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
			// Field bertipe model lain adalah relasi, bukan kolom.
			if !scalarTypes[baseType] && !enums[baseType] {
				continue
			}

			column := fieldName
			if mapped := mapAttribute.FindStringSubmatch(line); mapped != nil {
				column = mapped[1]
			}
			model.Columns[column] = true
		}
		models[name] = model
	}
	return models, nil
}

// gormColumns mengembalikan nama kolom yang dipetakan GORM untuk sebuah model.
func gormColumns(model any) (string, map[string]bool, error) {
	parsed, err := schema.Parse(model, &sync.Map{}, PrismaNamingStrategy{})
	if err != nil {
		return "", nil, err
	}
	columns := map[string]bool{}
	for _, field := range parsed.Fields {
		if field.DBName != "" {
			columns[field.DBName] = true
		}
	}
	return parsed.Table, columns, nil
}

// referenceSchemaPath mencari prisma/schema.prisma relatif terhadap paket ini.
func referenceSchemaPath() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrInvalid
	}
	// internal/database -> internal -> backend -> repo root
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(currentFile))))
	return filepath.Join(repoRoot, "prisma", "schema.prisma"), nil
}

// TestGormModelsMatchReferenceSchema memastikan setiap model GORM memiliki
// pasangan pada skema acuan dengan nama tabel dan nama kolom yang sama.
//
// Kegagalan test ini berarti drift antara prisma/schema.prisma dan mapping
// GORM: query produksi akan memakai kolom yang tidak ada di database.
func TestGormModelsMatchReferenceSchema(t *testing.T) {
	path, err := referenceSchemaPath()
	if err != nil {
		t.Fatal(err)
	}
	reference, err := loadPrismaSchema(path)
	if err != nil {
		t.Fatalf("cannot read reference schema: %v", err)
	}
	if len(reference) == 0 {
		t.Fatal("reference schema contains no models")
	}

	for _, model := range models.All() {
		typeName := reflect.TypeOf(model).Elem().Name()
		referenceModel, ok := reference[typeName]
		if !ok {
			t.Errorf("model GORM %s tidak ada pada skema acuan", typeName)
			continue
		}

		table, columns, err := gormColumns(model)
		if err != nil {
			t.Errorf("model %s: %v", typeName, err)
			continue
		}
		if table != referenceModel.Table {
			t.Errorf("model %s: tabel GORM %q, skema acuan %q", typeName, table, referenceModel.Table)
		}

		missing := []string{}
		for column := range referenceModel.Columns {
			if !columns[column] {
				missing = append(missing, column)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("model %s: kolom pada skema acuan belum dipetakan di GORM: %s", typeName, strings.Join(missing, ", "))
		}

		extra := []string{}
		for column := range columns {
			if !referenceModel.Columns[column] {
				extra = append(extra, column)
			}
		}
		sort.Strings(extra)
		if len(extra) > 0 {
			t.Errorf("model %s: kolom GORM tidak ada pada skema acuan: %s", typeName, strings.Join(extra, ", "))
		}
	}
}

// TestEveryReferenceModelHasGormModel memastikan tidak ada tabel pada skema
// acuan yang terlewat dari model GORM.
func TestEveryReferenceModelHasGormModel(t *testing.T) {
	path, err := referenceSchemaPath()
	if err != nil {
		t.Fatal(err)
	}
	reference, err := loadPrismaSchema(path)
	if err != nil {
		t.Fatal(err)
	}

	mapped := map[string]bool{}
	for _, model := range models.All() {
		mapped[reflect.TypeOf(model).Elem().Name()] = true
	}

	missing := []string{}
	for name := range reference {
		if !mapped[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("model pada skema acuan belum dipetakan di GORM: %s", strings.Join(missing, ", "))
	}
}
