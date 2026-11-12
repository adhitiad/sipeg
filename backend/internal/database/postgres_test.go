package database

import (
	"net/url"
	"sync"
	"testing"

	"simpeg/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm/schema"
)

func TestPostgresNamingStrategy(t *testing.T) {
	strategy := PostgresNamingStrategy{}
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{name: "mapped table", got: strategy.TableName("User"), expected: "user"},
		{name: "mapped camel table", got: strategy.TableName("GajiTunjangan"), expected: "gajiTunjangan"},
		{name: "attendance event table", got: strategy.TableName("AttendanceEvent"), expected: "AttendanceEvent"},
		{name: "default PostgreSQL table", got: strategy.TableName("Homebase"), expected: "Homebase"},
		{name: "camel column", got: strategy.ColumnName("", "UserId"), expected: "userId"},
		{name: "ID column", got: strategy.ColumnName("", "ID"), expected: "id"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.expected {
				t.Fatalf("got %q, want %q", test.got, test.expected)
			}
		})
	}
}

func TestGORMUsesPostgresTableAndColumnNames(t *testing.T) {
	parsed, err := schema.Parse(&models.User{}, &sync.Map{}, PostgresNamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Table != "user" {
		t.Fatalf("table name = %q, want %q", parsed.Table, "user")
	}

	for fieldName, expected := range map[string]string{
		"EmailVerified": "emailVerified",
		"JenisKelamin":  "jenis_kelamin",
		"NoKkEnc":       "no_kk_encrypted",
	} {
		field := parsed.LookUpField(fieldName)
		if field == nil {
			t.Fatalf("field %q was not parsed", fieldName)
		}
		if field.DBName != expected {
			t.Errorf("column for %s = %q, want %q", fieldName, field.DBName, expected)
		}
	}
}

func TestGORMOrganizationFieldsUsePostgresNames(t *testing.T) {
	parsed, err := schema.Parse(&models.Homebase{}, &sync.Map{}, PostgresNamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	for fieldName, expected := range map[string]string{
		"Jenis":    "jenis",
		"ParentId": "parentId",
		"Aktif":    "aktif",
	} {
		field := parsed.LookUpField(fieldName)
		if field == nil {
			t.Fatalf("field %q was not parsed", fieldName)
		}
		if field.DBName != expected {
			t.Errorf("column for %s = %q, want %q", fieldName, field.DBName, expected)
		}
	}
}

func TestWithSessionTimeZoneForcesUTC(t *testing.T) {
	// DSN tanpa kredensial: pemindaian kebocoran connection string di CI
	// menolak pola password, dan fungsi ini hanya mengubah query parameter.
	urlDSN := "postgres://simpeg@localhost:5432/simpeg?sslmode=disable"
	got := withSessionTimeZone(urlDSN)
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("timezone") != "UTC" {
		t.Fatalf("timezone = %q, want UTC", parsed.Query().Get("timezone"))
	}
	if parsed.Query().Get("sslmode") != "disable" {
		t.Fatalf("existing query parameters were dropped: %q", parsed.RawQuery)
	}

	keyValueDSN := "host=localhost user=postgres dbname=simpeg"
	if got := withSessionTimeZone(keyValueDSN); got != keyValueDSN+" timezone=UTC" {
		t.Fatalf("key/value DSN = %q", got)
	}
}

func TestEmbeddedModelHookGeneratesUUID(t *testing.T) {
	user := models.User{Email: "test@example.invalid", Nama: "Test"}
	if err := user.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := uuid.Parse(user.ID); err != nil {
		t.Fatalf("model hook did not assign a UUID: %v", err)
	}
}

func TestAttendanceEventMappingMatchesReferenceAndIndexes(t *testing.T) {
	parsed, err := schema.Parse(&models.AttendanceEvent{}, &sync.Map{}, PostgresNamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Table != "AttendanceEvent" {
		t.Fatalf("table name = %q, want %q", parsed.Table, "AttendanceEvent")
	}
	for fieldName, expected := range map[string]string{
		"MachineId":      "machineId",
		"MachinePin":     "machinePin",
		"OccurredAt":     "occurredAt",
		"IdempotencyKey": "idempotencyKey",
		"CreatedAt":      "createdAt",
	} {
		field := parsed.LookUpField(fieldName)
		if field == nil || field.DBName != expected {
			t.Errorf("column for %s = %v, want %q", fieldName, field, expected)
		}
	}

	for _, index := range parsed.ParseIndexes() {
		if index.Name == "attendance_event_machine_idempotency_key" {
			if len(index.Fields) != 2 || index.Fields[0].Field.DBName != "machineId" || index.Fields[1].Field.DBName != "idempotencyKey" || index.Class != "UNIQUE" {
				t.Fatalf("idempotency index does not match the reference: %+v", index)
			}
			return
		}
	}
	t.Fatal("idempotency unique index was not mapped")
}
