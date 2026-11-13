package listing

import (
	. "simpeg/backend/internal/listing"
	"testing"
)

// TestOrderClauseNeverEmitsColumnlessClause memastikan klausa ORDER selalu
// memuat kolom. Kolom bawaan yang tidak ada pada allowlist dulu menghasilkan
// " ASC" saja, yang ditolak PostgreSQL sebagai kesalahan sintaks.
func TestOrderClauseNeverEmitsColumnlessClause(t *testing.T) {
	columns := map[string]string{"nama": `"nama"`}

	got := OrderClause(columns, "tidak-ada", "asc", "juga-tidak-ada")
	if got != `"id" ASC` {
		t.Fatalf("clause = %q, want %q", got, `"id" ASC`)
	}
}

func TestOrderClauseKeepsAllowlistedDirection(t *testing.T) {
	columns := map[string]string{"nama": `"nama"`}

	if got := OrderClause(columns, "nama", "DESC", "nama"); got != `"nama" DESC` {
		t.Fatalf("clause = %q", got)
	}
	if got := OrderClause(columns, "", "asc", "nama"); got != `"nama" ASC` {
		t.Fatalf("fallback clause = %q", got)
	}
}
