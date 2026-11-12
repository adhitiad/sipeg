// Perintah migrate menerapkan migrasi SQL Goose pada database yang ditunjuk
// DATABASE_URL.
//
// Perintah ini dipakai oleh container migrasi pada deployment docker/podman
// dan bisa juga dijalankan langsung dari mesin pengembang, sehingga backend
// tidak perlu mengandalkan CLI goose yang terpasang terpisah. Migrasi
// dibawa dalam binary lewat embedded filesystem dari package migrations.
//
// Pemakaian:
//
//	migrate           # menerapkan migrasi yang belum jalan (goose up)
//	migrate status    # menampilkan status setiap migrasi
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"simpeg/backend/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

// run membaca perintah, membuka koneksi database, dan menjalankan goose.
func run(ctx context.Context, args []string) error {
	command := "up"
	if len(args) > 0 {
		command = strings.TrimSpace(strings.ToLower(args[0]))
	}
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL wajib diisi")
	}
	if command != "up" && command != "status" {
		return fmt.Errorf("perintah tidak dikenal %q (gunakan up atau status)", command)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("buka koneksi database: %w", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.Files)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialek goose: %w", err)
	}

	switch command {
	case "up":
		if err := goose.UpContext(ctx, db, "."); err != nil {
			return fmt.Errorf("terapkan migrasi: %w", err)
		}
		version, err := goose.GetDBVersionContext(ctx, db)
		if err != nil {
			return fmt.Errorf("baca versi database: %w", err)
		}
		fmt.Printf("migrasi diterapkan, versi database: %d\n", version)
	case "status":
		if err := goose.StatusContext(ctx, db, "."); err != nil {
			return fmt.Errorf("status migrasi: %w", err)
		}
	}
	return nil
}
