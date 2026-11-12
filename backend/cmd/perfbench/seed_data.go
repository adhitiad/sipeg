package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"simpeg/backend/internal/models"

	"gorm.io/gorm"
)

// parseSeedPlan membaca skala seeding dari argumen baris perintah.
func parseSeedPlan(args []string) (seedPlan, error) {
	flags := flag.NewFlagSet("seed", flag.ContinueOnError)
	employees := flags.Int("employees", 5000, "jumlah pegawai sintetis")
	months := flags.Int("months", 6, "jumlah bulan slip gaji per pegawai")
	audits := flags.Int("audit", 50000, "jumlah entri jejak audit")
	if err := flags.Parse(args); err != nil {
		return seedPlan{}, err
	}
	if *employees < 1 || *months < 1 || *audits < 1 {
		return seedPlan{}, fmt.Errorf("jumlah data harus lebih besar dari nol")
	}
	return seedPlan{Employees: *employees, Months: *months, AuditLogs: *audits}, nil
}

// seedEmployees membuat pegawai sintetis yang tersebar pada seluruh unit.
//
// Pegawai pertama menjadi admin agar pengukuran end-to-end punya principal
// dengan cakupan seluruh organisasi.
func seedEmployees(
	ctx context.Context,
	db *gorm.DB,
	plan seedPlan,
	roles map[string]string,
	units []models.Homebase,
	categories []models.Kategori,
) ([]models.User, error) {
	employees := make([]models.User, 0, plan.Employees)
	scopeUnits := units[1:]
	joined := timeutilDate(2020, 1, 1)
	for index := 0; index < plan.Employees; index++ {
		unit := scopeUnits[index%len(scopeUnits)]
		category := categories[index%len(categories)]
		role := "pegawai"
		switch {
		case index == 0:
			role = "admin"
		case index%50 == 0:
			role = "operator"
		case index%20 == 0:
			role = "atasan"
		}
		employee := models.User{
			Email:             fmt.Sprintf("pegawai-%06d@%s", index, seedEmailDomain),
			Nama:              fmt.Sprintf("Pegawai Sintetis %06d", index),
			Aktif:             true,
			EmailVerified:     true,
			RoleId:            stringPtr(roles[role]),
			HomebaseId:        &unit.ID,
			KategoriPegawaiId: &category.ID,
			TanggalMasuk:      timePtr(joined),
			JenisKelamin:      genderFor(index),
		}
		if index%2 == 0 {
			employee.Nip = stringPtr(fmt.Sprintf("NIP%08d", index))
		}
		if index%4 == 0 {
			employee.Nidn = stringPtr(fmt.Sprintf("NIDN%08d", index))
		}
		if index%4 == 1 {
			employee.Nidk = stringPtr(fmt.Sprintf("NIDK%08d", index))
		}
		employees = append(employees, employee)
	}
	if err := db.WithContext(ctx).CreateInBatches(&employees, seedBatchSize).Error; err != nil {
		return nil, fmt.Errorf("buat pegawai: %w", err)
	}
	return employees, nil
}

// genderFor menentukan jenis kelamin sintetis secara bergilir.
func genderFor(index int) *models.JenisKelamin {
	value := models.JenisKelaminP
	if index%2 == 0 {
		value = models.JenisKelaminL
	}
	return &value
}

// seedPayroll membuat slip gaji bulanan beserta tunjangan, potongan, dan
// konfirmasi pembayaran. Hanya slip berstatus TERBAYAR yang memiliki
// konfirmasi pembayaran, sesuai aturan domain penggajian.
func seedPayroll(ctx context.Context, db *gorm.DB, plan seedPlan, employees []models.User) error {
	base := timeutilDate(2026, 1, 1)
	slips := make([]models.Gaji, 0, plan.Employees*plan.Months)
	for month := 0; month < plan.Months; month++ {
		periode := base.AddDate(0, month, 0)
		for index, employee := range employees {
			pokok := 3_000_000 + float64(index%40)*125_000
			total := pokok + 750_000
			status := models.StatusSlipDraf
			if month < plan.Months-1 {
				status = models.StatusSlipTerbayar
			}
			slips = append(slips, models.Gaji{
				UserId:        employee.ID,
				NomorSlip:     fmt.Sprintf("SLIP-%s-%06d", periode.Format("2006-01"), index),
				Periode:       periode,
				TanggalGaji:   periode.AddDate(0, 0, 25),
				GajiPokok:     pokok,
				TotalGaji:     total,
				Status:        status,
				PotonganPph21: total * 0.05,
			})
		}
	}
	if err := db.WithContext(ctx).CreateInBatches(&slips, seedBatchSize).Error; err != nil {
		return fmt.Errorf("buat slip gaji: %w", err)
	}

	tunjangan := make([]models.GajiTunjangan, 0, len(slips))
	potongan := make([]models.GajiPotongan, 0, len(slips))
	payments := make([]models.Pembayaran, 0, len(slips))
	for index := range slips {
		slip := &slips[index]
		tunjangan = append(tunjangan, models.GajiTunjangan{
			GajiId:  slip.ID,
			Jenis:   "TUNJANGAN_FUNGSIONAL",
			Nominal: 750_000,
		})
		potongan = append(potongan, models.GajiPotongan{
			GajiId:  slip.ID,
			Jenis:   "PPH21",
			Nominal: slip.TotalGaji * 0.05,
		})
		if slip.Status != models.StatusSlipTerbayar {
			continue
		}
		payments = append(payments, models.Pembayaran{
			GajiId:           slip.ID,
			WaktuKonfirmasi:  slip.TanggalGaji.AddDate(0, 0, 1),
			PelakuPengesahan: perfAdminEmail,
		})
	}
	if err := db.WithContext(ctx).CreateInBatches(&tunjangan, seedBatchSize).Error; err != nil {
		return fmt.Errorf("buat tunjangan: %w", err)
	}
	if err := db.WithContext(ctx).CreateInBatches(&potongan, seedBatchSize).Error; err != nil {
		return fmt.Errorf("buat potongan: %w", err)
	}
	if err := db.WithContext(ctx).CreateInBatches(&payments, seedBatchSize).Error; err != nil {
		return fmt.Errorf("buat pembayaran: %w", err)
	}
	return nil
}

// seedAuditLogs membuat jejak audit tersebar pada rentang waktu lalu lalu.
func seedAuditLogs(
	ctx context.Context,
	db *gorm.DB,
	plan seedPlan,
	roles map[string]string,
	units []models.Homebase,
	employees []models.User,
) error {
	actions := []string{
		"employee.created", "employee.updated", "employee.archived",
		"organization.updated", "audit.accessed", "payroll.payment.confirmed",
	}
	entries := make([]models.AuditLog, 0, plan.AuditLogs)
	now := nowUTC()
	for index := 0; index < plan.AuditLogs; index++ {
		employee := employees[index%len(employees)]
		unit := units[index%len(units)]
		actor := roles["admin"]
		if index%3 == 0 {
			actor = roles["operator"]
		}
		created := now.Add(-time.Duration(index) * time.Minute)
		note := fmt.Sprintf("catatan jejak audit sintetis %d", index)
		entries = append(entries, models.AuditLog{
			AktorId:    &actor,
			Aksi:       actions[index%len(actions)],
			Sasaran:    &employee.ID,
			Keterangan: &note,
			Ip:         stringPtr("10.0.0.1"),
			UnitId:     &unit.ID,
			CreatedAt:  created,
		})
	}
	if err := db.WithContext(ctx).CreateInBatches(&entries, seedBatchSize).Error; err != nil {
		return fmt.Errorf("buat jejak audit: %w", err)
	}
	return nil
}

// seedCalendar membuat jadwal gajian per unit dan kalender hari libur.
func seedCalendar(ctx context.Context, db *gorm.DB, units []models.Homebase) error {
	schedules := make([]models.JadwalGaji, 0, len(units))
	for index, unit := range units {
		schedules = append(schedules, models.JadwalGaji{
			UnitId:        unit.ID,
			TanggalGajian: 20 + index%9,
		})
	}
	if err := db.WithContext(ctx).Create(&schedules).Error; err != nil {
		return fmt.Errorf("buat jadwal gaji: %w", err)
	}
	holidays := make([]models.HariLibur, 0, 12)
	for month := 1; month <= 12; month++ {
		holidays = append(holidays, models.HariLibur{
			Tanggal: timeutilDate(2026, time.Month(month), 17),
			Nama:    fmt.Sprintf("Hari Libur Sintetis %d", month),
		})
	}
	if err := db.WithContext(ctx).Create(&holidays).Error; err != nil {
		return fmt.Errorf("buat hari libur: %w", err)
	}
	return nil
}

// timeutilDate menyusun waktu UTC pada tanggal tertentu.
func timeutilDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
