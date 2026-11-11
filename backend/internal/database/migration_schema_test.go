package database

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"

	"simpeg/backend/internal/models"

	"gorm.io/gorm/schema"
)

type migrationSchema struct {
	Tables  map[string]map[string]bool
	Indexes map[string][][]string
}

var (
	createTablePattern = regexp.MustCompile(`(?is)CREATE\s+TABLE(?:\s+IF\s+NOT\s+EXISTS)?\s+"([^"]+)"\s*\((.*?)\);`)
	columnPattern      = regexp.MustCompile(`(?m)^\s*"([^"]+)"\s+`)
	addColumnPattern   = regexp.MustCompile(`(?i)ALTER\s+TABLE\s+"([^"]+)"\s+ADD\s+COLUMN\s+"([^"]+)"`)
	indexPattern       = regexp.MustCompile(`(?is)CREATE\s+(?:UNIQUE\s+)?INDEX(?:\s+IF\s+NOT\s+EXISTS)?\s+"?[a-z0-9_]+"?\s+ON\s+"([^"]+)"\s*\(([^)]+)\)`)
	uniquePattern      = regexp.MustCompile(`(?i)UNIQUE\s*\(([^)]+)\)`)
	inlineUnique       = regexp.MustCompile(`(?im)^\s*"([^"]+)"\s+[^,\n]*\bUNIQUE\b`)
	quotedColumn       = regexp.MustCompile(`"([^"]+)"`)
)

func loadMigrationSchema() (migrationSchema, error) {
	entries, err := os.ReadDir(migrationsDir())
	if err != nil {
		return migrationSchema{}, err
	}
	schemaState := migrationSchema{Tables: map[string]map[string]bool{}, Indexes: map[string][][]string{}}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(migrationsDir(), entry.Name()))
		if err != nil {
			return migrationSchema{}, err
		}
		schemaState.addMigration(string(content))
	}
	return schemaState, nil
}

func (s *migrationSchema) addMigration(sql string) {
	for _, match := range createTablePattern.FindAllStringSubmatch(sql, -1) {
		table, body := match[1], match[2]
		if s.Tables[table] == nil {
			s.Tables[table] = map[string]bool{}
		}
		for _, column := range columnPattern.FindAllStringSubmatch(body, -1) {
			s.Tables[table][column[1]] = true
		}
		for _, constraint := range uniquePattern.FindAllStringSubmatch(body, -1) {
			s.addIndex(table, constraint[1])
		}
		for _, field := range inlineUnique.FindAllStringSubmatch(body, -1) {
			s.addIndex(table, `"`+field[1]+`"`)
		}
	}
	for _, match := range addColumnPattern.FindAllStringSubmatch(sql, -1) {
		if s.Tables[match[1]] == nil {
			s.Tables[match[1]] = map[string]bool{}
		}
		s.Tables[match[1]][match[2]] = true
	}
	for _, match := range indexPattern.FindAllStringSubmatch(sql, -1) {
		s.addIndex(match[1], match[2])
	}
}

func (s *migrationSchema) addIndex(table, columnList string) {
	columns := quotedColumn.FindAllStringSubmatch(columnList, -1)
	if len(columns) == 0 {
		return
	}
	index := make([]string, 0, len(columns))
	for _, column := range columns {
		index = append(index, column[1])
	}
	s.Indexes[table] = append(s.Indexes[table], index)
}

func mappedGormTable(model any) (string, error) {
	parsed, err := schema.Parse(model, &sync.Map{}, PostgresNamingStrategy{})
	if err != nil {
		return "", err
	}
	return parsed.Table, nil
}

func TestGormModelsHaveMigrationTables(t *testing.T) {
	schemaState, err := loadMigrationSchema()
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range models.All() {
		typeName := reflect.TypeOf(model).Elem().Name()
		table, err := mappedGormTable(model)
		if err != nil {
			t.Errorf("model %s: %v", typeName, err)
			continue
		}
		if _, ok := schemaState.Tables[table]; !ok {
			t.Errorf("model GORM %s maps to table %q missing from migrations", typeName, table)
		}
	}
}

func TestCriticalQueryPathsHaveMigrationIndexes(t *testing.T) {
	schemaState, err := loadMigrationSchema()
	if err != nil {
		t.Fatal(err)
	}
	critical := []struct{ table, column string }{
		{"user", "homebaseId"}, {"user", "email"}, {"session", "token"},
		{"Homebase", "parentId"}, {"auditLog", "unitId"}, {"auditLog", "createdAt"},
	}
	for _, item := range critical {
		indexed := false
		for _, index := range schemaState.Indexes[item.table] {
			for _, column := range index {
				if column == item.column {
					indexed = true
				}
			}
		}
		if !indexed {
			t.Errorf("migration index missing for %s.%s", item.table, item.column)
		}
	}
}
