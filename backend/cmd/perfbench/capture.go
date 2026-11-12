package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"gorm.io/gorm"
)

// capturedStatement adalah satu statement SQL yang terekam beserta
// parameternya. Urutan argumen cocok dengan placeholder `$N` pada SQL.
type capturedStatement struct {
	SQL  string
	Args []interface{}
}

// sqlRecorder membungkus connection pool GORM untuk merekam statement SQL
// yang benar-benar dikirim ke PostgreSQL. Perekaman ini membuat pengukuran
// memakai statement yang sama persis dengan jalur produksi, bukan query yang
// ditulis ulang secara manual.
type sqlRecorder struct {
	pool gorm.ConnPool

	mu     sync.Mutex
	active bool
	stmts  []capturedStatement
}

// newSQLRecorder membungkus pool database dengan perekam statement.
func newSQLRecorder(db *gorm.DB) *sqlRecorder {
	recorder := &sqlRecorder{pool: db.Statement.ConnPool}
	// Statement pada session baru mewarisi pool dari statement akar, jadi
	// kedua tempat harus dibungkus agar query apa pun ikut terekam.
	db.Statement.ConnPool = recorder
	db.ConnPool = recorder
	return recorder
}

// start mengaktifkan perekaman dan membersihkan rekaman sebelumnya.
func (r *sqlRecorder) start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active = true
	r.stmts = nil
}

// stop menonaktifkan perekaman dan mengembalikan statement yang terekam.
func (r *sqlRecorder) stop() []capturedStatement {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active = false
	return append([]capturedStatement(nil), r.stmts...)
}

// record menyimpan satu statement beserta argumennya.
func (r *sqlRecorder) record(query string, args []interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.active {
		return
	}
	copied := make([]interface{}, len(args))
	copy(copied, args)
	r.stmts = append(r.stmts, capturedStatement{SQL: query, Args: copied})
}

// PrepareContext meneruskan prepared statement ke pool asli.
func (r *sqlRecorder) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return r.pool.PrepareContext(ctx, query)
}

// ExecContext merekam lalu menjalankan statement.
func (r *sqlRecorder) ExecContext(
	ctx context.Context,
	query string,
	args ...interface{},
) (sql.Result, error) {
	r.record(query, args)
	return r.pool.ExecContext(ctx, query, args...)
}

// QueryContext merekam lalu menjalankan statement.
func (r *sqlRecorder) QueryContext(
	ctx context.Context,
	query string,
	args ...interface{},
) (*sql.Rows, error) {
	r.record(query, args)
	return r.pool.QueryContext(ctx, query, args...)
}

// QueryRowContext merekam lalu menjalankan statement.
func (r *sqlRecorder) QueryRowContext(
	ctx context.Context,
	query string,
	args ...interface{},
) *sql.Row {
	r.record(query, args)
	return r.pool.QueryRowContext(ctx, query, args...)
}

// scenarioError menggabungkan pesan scenario dengan konteks nama.
func scenarioError(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("skenario %s: %w", name, err)
}
