// Package repository memuat port persistensi pengajuan dan keputusan
// pemberhentian pegawai (DEC-083).
package repository

import (
	"context"
	"time"

	"simpeg/backend/internal/models"
)

// ProposeInput memuat isian pengajuan pemberhentian.
type ProposeInput struct {
	UserID     string
	Alasan     string
	Jenis      models.JenisPemberhentian
	Tanggal    time.Time
	Keterangan *string
}

// ListOptions membatasi daftar pengajuan/keputusan.
type ListOptions struct {
	UserID string
	Status models.StatusPersetujuan
	// HomebaseIDs membatasi untuk operator (cakupan unit). Kosong berarti
	// seluruh unit.
	HomebaseIDs []string
	Limit       int
	Offset      int
}

// Store adalah port persistensi pemberhentian pegawai.
type Store interface {
	// EmployeeHomebaseID mengembalikan unit penugasan pegawai; nil bila
	// belum diisi.
	EmployeeHomebaseID(ctx context.Context, userID string) (*string, error)
	// EmployeeStartDate mengembalikan tanggal mulai kerja pegawai; nil bila
	// tidak terisi.
	EmployeeStartDate(ctx context.Context, userID string) (*time.Time, error)
	// EmployeeKategoriStatusID mengembalikan kategori status pegawai; nil
	// bila tidak terisi.
	EmployeeKategoriStatusID(ctx context.Context, userID string) (*string, error)
	// PendingForUser melaporkan apakah pegawai memiliki pengajuan MENUNGGU.
	PendingForUser(ctx context.Context, userID string) (bool, error)
	// RulesFor membaca pesangonAturan per kategori status; map kosong bila
	// belum ada baris.
	RulesFor(ctx context.Context, kategoriStatusID string) (map[int]float64, error)
	// Create menulis pengajuan MENUNGGU dalam transaksi bersama audit.
	Create(ctx context.Context, actor string, input ProposeInput, unitID *string) (models.Pemberhentian, error)
	// Get membaca satu pengajuan/keputusan.
	Get(ctx context.Context, id string) (models.Pemberhentian, error)
	// List membaca pengajuan/keputusan sesuai filter dan cakupan.
	List(ctx context.Context, options ListOptions) ([]models.Pemberhentian, int64, error)
	// Decide mengubah status pengajuan menjadi DISETUJUI atau DITOLAK,
	// menerbitkan SK PHK beserta nomor urutnya, menulis kompensasi, dan
	// mencatat audit. Field kompensasiBulan diisi oleh lapisan app.
	Decide(ctx context.Context, actor, id string, approve bool, kompensasiBulan float64, note *string, unitID *string) (models.Pemberhentian, error)
}
