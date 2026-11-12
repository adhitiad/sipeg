package main

import (
	"context"
	"fmt"

	directoryrepository "simpeg/backend/internal/directory/repository"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	payrollrepository "simpeg/backend/internal/payroll/repository"

	"gorm.io/gorm"
)

// scenario adalah satu query yang diukur. Fungsi Run memakai repository
// produksi sehingga SQL yang dibangkitkan sama dengan SQL aplikasi.
type scenario struct {
	// Name adalah nama skenario yang dipakai pada baris hasil.
	Name string
	// Description menjelaskan query yang diukur.
	Description string
	// Run menjalankan query sekali.
	Run func(ctx context.Context) error
}

// scenarioEnv memuat objek bersama yang dibutuhkan seluruh skenario.
type scenarioEnv struct {
	DB *gorm.DB
	// ScopeIDs adalah daftar unit dalam cakupan operator.
	ScopeIDs []string
	// RootUnitID adalah unit akar pohon organisasi.
	RootUnitID string
	// UserID adalah pegawai pertama untuk query cakupan "self".
	UserID string
}

// buildScenarios menyusun daftar skenario pengukuran memakai repository
// produksi. Query yang diukur adalah jalur yang benar-benar dipakai handler
// pada modul direktori dan penggajian.
func buildScenarios(env *scenarioEnv) []scenario {
	directory := directoryrepository.GormStore{DB: env.DB}
	payroll := payrollrepository.GormStore{DB: env.DB}

	return []scenario{
		{
			Name:        "baseline.roundtrip",
			Description: "perjalanan bolak-balik kosong ke PostgreSQL sebagai dasar pembanding",
			Run: func(ctx context.Context) error {
				return env.DB.WithContext(ctx).Exec("SELECT 1").Error
			},
		},
		{
			Name:        "employees.list.admin",
			Description: "daftar pegawai tanpa filter, cakupan admin, 20 baris",
			Run: func(ctx context.Context) error {
				_, _, err := directory.ListEmployees(ctx, listing.Options{
					Sort: "nama", Order: "asc", Page: 1, PageSize: 20,
				})
				return err
			},
		},
		{
			Name:        "employees.list.page50",
			Description: "daftar pegawai tanpa filter, 50 baris per halaman",
			Run: func(ctx context.Context) error {
				_, _, err := directory.ListEmployees(ctx, listing.Options{
					Sort: "nama", Order: "asc", Page: 1, PageSize: 50,
				})
				return err
			},
		},
		{
			Name:        "employees.list.search",
			Description: "daftar pegawai dengan pencarian ILIKE nama",
			Run: func(ctx context.Context) error {
				_, _, err := directory.ListEmployees(ctx, listing.Options{
					Search: "Pegawai Sintetis 0001", Sort: "nama", Order: "asc", Page: 1, PageSize: 20,
				})
				return err
			},
		},
		{
			Name:        "employees.list.scope",
			Description: "daftar pegawai dengan penyaringan cakupan unit operator",
			Run: func(ctx context.Context) error {
				_, _, err := directory.ListEmployees(ctx, listing.Options{
					ScopeHomebaseIDs: env.ScopeIDs,
					Sort:             "nama", Order: "asc", Page: 1, PageSize: 20,
				})
				return err
			},
		},
		{
			Name:        "audit.list.admin",
			Description: "daftar jejak audit tanpa filter, 20 baris",
			Run: func(ctx context.Context) error {
				_, _, err := directory.ListAuditLogs(ctx, listing.Options{
					Sort: "createdAt", Order: "desc", Page: 1, PageSize: 20,
				})
				return err
			},
		},
		{
			Name:        "audit.list.scope",
			Description: "daftar jejak audit dengan penyaringan cakupan unit",
			Run: func(ctx context.Context) error {
				_, _, err := directory.ListAuditLogs(ctx, listing.Options{
					ScopeHomebaseIDs: env.ScopeIDs,
					Sort:             "createdAt", Order: "desc", Page: 1, PageSize: 20,
				})
				return err
			},
		},
		{
			Name:        "audit.list.filtered",
			Description: "daftar jejak audit dengan penyaring jenis aksi",
			Run: func(ctx context.Context) error {
				_, _, err := directory.ListAuditLogs(ctx, listing.Options{
					Aksi: "employee.created",
					Sort: "createdAt", Order: "desc", Page: 1, PageSize: 20,
				})
				return err
			},
		},
		{
			Name:        "categories.list",
			Description: "daftar kategori pegawai",
			Run: func(ctx context.Context) error {
				_, _, err := directory.ListCategories(ctx, listing.Options{
					Sort: "kode", Order: "asc", Page: 1, PageSize: 50,
				})
				return err
			},
		},
		{
			Name:        "scope.resolve",
			Description: "penyelesaian cakupan unit dengan recursive CTE",
			Run: func(ctx context.Context) error {
				_, err := directory.ListUnitScopeIDs(ctx, env.RootUnitID)
				return err
			},
		},
		{
			Name:        "payroll.list.admin",
			Description: "daftar slip gaji tanpa filter, 20 baris beserta pramuat",
			Run: func(ctx context.Context) error {
				_, _, err := payroll.ListSlips(ctx, listing.Options{
					Sort: "periode", Order: "desc", Page: 1, PageSize: 20,
				})
				return err
			},
		},
		{
			Name:        "payroll.list.periode",
			Description: "daftar slip gaji dengan penyaring periode bulanan",
			Run: func(ctx context.Context) error {
				_, _, err := payroll.ListSlips(ctx, listing.Options{
					Periode: "2026-02",
					Sort:    "periode", Order: "desc", Page: 1, PageSize: 20,
				})
				return err
			},
		},
		{
			Name:        "payroll.list.scope",
			Description: "daftar slip gaji dengan penyaringan cakupan unit",
			Run: func(ctx context.Context) error {
				_, _, err := payroll.ListSlips(ctx, listing.Options{
					ScopeHomebaseIDs: env.ScopeIDs,
					Sort:             "periode", Order: "desc", Page: 1, PageSize: 20,
				})
				return err
			},
		},
		{
			Name:        "payroll.list.self",
			Description: "daftar slip gaji milik satu pegawai (cakupan self)",
			Run: func(ctx context.Context) error {
				_, _, err := payroll.ListSlips(ctx, listing.Options{
					UserID: env.UserID,
					Sort:   "periode", Order: "desc", Page: 1, PageSize: 20,
				})
				return err
			},
		},
	}
}

// loadScenarioEnv menyusun konteks pengukuran dari data yang sudah di-seed.
func loadScenarioEnv(ctx context.Context, db *gorm.DB) (*scenarioEnv, error) {
	var root models.Homebase
	if err := db.WithContext(ctx).Where("\"parentId\" IS NULL").First(&root).Error; err != nil {
		return nil, fmt.Errorf("cari unit akar: %w", err)
	}
	var employee models.User
	if err := db.WithContext(ctx).Where("aktif = ?", true).Order("\"createdAt\"").First(&employee).Error; err != nil {
		return nil, fmt.Errorf("cari pegawai: %w", err)
	}
	store := directoryrepository.GormStore{DB: db}
	scopeIDs, err := store.ListUnitScopeIDs(ctx, root.ID)
	if err != nil {
		return nil, fmt.Errorf("hitung cakupan unit: %w", err)
	}
	return &scenarioEnv{
		DB:         db,
		ScopeIDs:   scopeIDs,
		RootUnitID: root.ID,
		UserID:     employee.ID,
	}, nil
}
