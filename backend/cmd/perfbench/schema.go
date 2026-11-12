package main

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"

	"simpeg/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// columnMismatch memuat satu perbedaan kolom antara model dan database.
type columnMismatch struct {
	Table  string
	Column string
	Kind   string
	Detail string
}

// schemaReport memuat hasil pemeriksaan keselarasan skema.
type schemaReport struct {
	Missing []columnMismatch
	Extra   []columnMismatch
	NoTable []string
	Checked int
}

// runSchema membandingkan kolom yang dinyatakan model GORM dengan kolom yang
// benar-benar ada pada database target.
//
// Pemeriksaan ini menjawab pertanyaan yang tidak bisa dijawab dari kode:
// apakah skema pada database target sudah selaras dengan model. Migrasi yang
// ada belum tentu sudah diterapkan, dan model yang ada belum tentu selaras
// dengan tabel yang dibuat migrasi.
func runSchema(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("schema", flag.ContinueOnError)
	verbose := flags.Bool("verbose", false, "tampilkan seluruh model yang diperiksa")
	if err := flags.Parse(args); err != nil {
		return err
	}
	db, err := openDB(ctx)
	if err != nil {
		return err
	}
	report, err := inspectSchema(ctx, db)
	if err != nil {
		return err
	}
	fmt.Printf("model diperiksa: %d\n", report.Checked)
	if len(report.Missing) == 0 && len(report.Extra) == 0 && len(report.NoTable) == 0 {
		fmt.Println("selaras: seluruh kolom model ada pada database dan tidak ada kolom asing")
		return nil
	}
	if len(report.NoTable) > 0 {
		fmt.Printf("\ntabel tidak ada pada database (%d):\n", len(report.NoTable))
		for _, name := range report.NoTable {
			fmt.Printf("  - %s\n", name)
		}
	}
	if len(report.Missing) > 0 {
		fmt.Printf("\nkolom ada di model tapi tidak ada di database (%d):\n", len(report.Missing))
		for _, item := range report.Missing {
			fmt.Printf("  - %s.%s (%s)\n", item.Table, item.Column, item.Detail)
		}
	}
	if len(report.Extra) > 0 {
		fmt.Printf("\nkolom ada di database tapi tidak ada di model (%d):\n", len(report.Extra))
		for _, item := range report.Extra {
			fmt.Printf("  - %s.%s (%s)\n", item.Table, item.Column, item.Detail)
		}
	}
	if *verbose {
		fmt.Println("\ndaftar model yang diperiksa:")
		for _, name := range report.NoTable {
			fmt.Printf("  - %s (tabel belum ada)\n", name)
		}
	}
	return nil
}

// inspectSchema menjalankan perbandingan model dan database.
func inspectSchema(ctx context.Context, db *gorm.DB) (schemaReport, error) {
	report := schemaReport{}
	actual, err := databaseColumns(ctx, db)
	if err != nil {
		return report, err
	}
	for _, model := range models.All() {
		parsed, err := parseModel(db, model)
		if err != nil {
			return report, err
		}
		table := parsed.Table
		report.Checked++
		columns, exists := actual[table]
		if !exists {
			report.NoTable = append(report.NoTable, table)
			continue
		}
		for _, column := range parsed.DBNames {
			if !contains(columns, column) {
				report.Missing = append(report.Missing, columnMismatch{
					Table:  table,
					Column: column,
					Kind:   "missing",
					Detail: "model menulis kolom ini pada setiap INSERT",
				})
			}
		}
		for _, column := range columns {
			if !contains(parsed.DBNames, column) {
				report.Extra = append(report.Extra, columnMismatch{
					Table:  table,
					Column: column,
					Kind:   "extra",
					Detail: "kolom ini tidak dinyatakan model",
				})
			}
		}
	}
	sort.Strings(report.NoTable)
	return report, nil
}

// parseModel membaca skema model memakai naming strategy yang sama dengan
// aplikasi.
func parseModel(db *gorm.DB, model any) (*schema.Schema, error) {
	statement := &gorm.Statement{DB: db}
	if err := statement.Parse(model); err != nil {
		return nil, fmt.Errorf("baca skema model: %w", err)
	}
	return statement.Schema, nil
}

// databaseColumns membaca seluruh kolom pada skema public.
func databaseColumns(ctx context.Context, db *gorm.DB) (map[string][]string, error) {
	type row struct {
		Table  string
		Column string
	}
	var rows []row
	err := db.WithContext(ctx).Raw(
		`SELECT table_name AS "table", column_name AS "column"
		   FROM information_schema.columns
		  WHERE table_schema = 'public'
		  ORDER BY table_name, ordinal_position`,
	).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("baca kolom database: %w", err)
	}
	result := make(map[string][]string)
	for _, item := range rows {
		result[item.Table] = append(result[item.Table], item.Column)
	}
	return result, nil
}

// contains memeriksa keberadaan nilai dalam daftar.
func contains(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}
