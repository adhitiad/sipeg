package domain

import (
	"testing"

	"simpeg/backend/internal/listing"
)

func TestSortFieldsAreStableAndNonEmpty(t *testing.T) {
	lists := [][]string{
		EmployeeSortFields(), OrganizationSortFields(), AuditLogSortFields(), CategorySortFields(),
	}
	for _, fields := range lists {
		if len(fields) == 0 {
			t.Fatal("sort field list must not be empty")
		}
		for index := 1; index < len(fields); index++ {
			if fields[index-1] >= fields[index] {
				t.Fatalf("sort fields must be sorted and unique: %v", fields)
			}
		}
	}
}

func TestOrderClauseFallsBackToDefaultColumn(t *testing.T) {
	clause := listing.OrderClause(EmployeeSortColumns, "unknown", "desc", "nama")
	if clause != `"nama" DESC` {
		t.Fatalf("clause = %q", clause)
	}
}

func TestOrderClauseIgnoresUntrustedSortValue(t *testing.T) {
	clause := listing.OrderClause(EmployeeSortColumns, `nama"; DROP TABLE "user"`, "asc", "nama")
	if clause != `"nama" ASC` {
		t.Fatalf("clause = %q", clause)
	}
}

func TestPageCount(t *testing.T) {
	cases := []struct {
		total    int64
		pageSize int
		want     int
	}{
		{total: 0, pageSize: 20, want: 0},
		{total: 1, pageSize: 20, want: 1},
		{total: 20, pageSize: 20, want: 1},
		{total: 21, pageSize: 20, want: 2},
	}
	for _, test := range cases {
		if got := listing.PageCount(test.total, test.pageSize); got != test.want {
			t.Fatalf("PageCount(%d, %d) = %d, want %d", test.total, test.pageSize, got, test.want)
		}
	}
}
