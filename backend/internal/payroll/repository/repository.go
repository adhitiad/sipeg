// Package repository memuat port persistensi dan adapter GORM untuk area
// penggajian.
//
// Pembayaran gaji dicatat bersama slipnya dalam satu transaksi: baris
// konfirmasi dibuat, slip ditandai TERBAYAR, notifikasi diantrekan, dan jejak
// audit ditulis. Bila salah satu langkah gagal, seluruh perubahan dibatalkan.
//
// Jadwal tanggal gajian per unit dan pembacaan slip yang siap diberi
// pemberitahuan H+1 memakai port yang sama.
package repository

import (
	"context"
	"time"

	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/payroll/domain"
)

// Repository adalah port persistensi data penggajian.
type Repository interface {
	// ListSlips membaca daftar slip gaji sesuai filter dan pagination.
	ListSlips(context.Context, listing.Options) ([]models.Gaji, int64, error)
	// GetSlip membaca satu slip gaji beserta data pegawai pemiliknya.
	GetSlip(context.Context, string) (models.Gaji, error)
	// RecordPayment mencatat konfirmasi pembayaran untuk sebuah slip.
	RecordPayment(context.Context, string, string, domain.PaymentInput, domain.Notification) (models.Gaji, error)
	// ConfirmedPayment melaporkan apakah pembayaran untuk pegawai dan
	// periode tertentu sudah tercatat sebagai terkonfirmasi.
	ConfirmedPayment(context.Context, string, string) (bool, error)
	// UpsertSchedule menyimpan jadwal tanggal gajian untuk satu unit.
	UpsertSchedule(context.Context, string, domain.ScheduleInput) (models.JadwalGaji, error)
	// ListSchedules membaca daftar jadwal tanggal gajian pada unit-unit
	// yang diberikan. unitIDs nil berarti seluruh unit; slice kosong
	// berarti tidak ada unit yang boleh dijangkau.
	ListSchedules(context.Context, []string) ([]models.JadwalGaji, error)
	// FindDueReminders membaca slip gaji yang siap diberi pemberitahuan
	// ketersediaan H+1 hingga periode berjalan.
	FindDueReminders(context.Context, time.Time) ([]Reminder, error)
	// EnqueueNotifications menulis pesan notifikasi ke antrean keluar dan
	// mengembalikan jumlah pesan yang benar-benar ditulis.
	EnqueueNotifications(context.Context, []domain.Notification) (int, error)
}
