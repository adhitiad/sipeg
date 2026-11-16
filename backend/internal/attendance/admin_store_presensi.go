package attendance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/models"
)

// MappedEmployees mengembalikan pegawai yang aktif terpetakan ke sebuah machinePin.
func newMappedEmployee(userID, nama, machinePin string) MappedEmployee {
	return MappedEmployee{UserID: userID, Nama: nama, MachinePin: machinePin}
}

func (s *GormAdminStore) MappedEmployees(ctx context.Context, scopeIDs []string) ([]MappedEmployee, error) {
	type row struct {
		UserID     string `gorm:"column:user_id"`
		Nama       string `gorm:"column:nama"`
		MachinePin string `gorm:"column:machinePin"`
	}
	rows := []row{}
	query := s.DB.WithContext(ctx).
		Table(`"mesinAbsensiPin" AS p`).
		Select(`u."id" AS user_id, u."nama", p."machinePin"`).
		Joins(`JOIN "user" AS u ON u."id" = p."userId"`).
		Where(`p."aktif" = ?`, true).
		Order(`u."nama" ASC`)
	if optionalScopeIDs(scopeIDs) {
		if len(scopeIDs) == 0 {
			return []MappedEmployee{}, nil
		}
		query = query.Where(`u."homebaseId" IN ?`, scopeIDs)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list mapped employees: %w", err)
	}
	employees := make([]MappedEmployee, 0, len(rows))
	for _, row := range rows {
		employees = append(employees, newMappedEmployee(row.UserID, row.Nama, row.MachinePin))
	}
	return employees, nil
}

// EventsPerPinPerDay mengembalikan jumlah tap per machinePin per hari
// berdasarkan kalender Asia/Jakarta (zona waktu operasional). Tap pada
// 00:00–07:00 WIB jatuh pada tanggal Jakarta yang sama, bukan tanggal UTC
// sebelumnya.
func (s *GormAdminStore) EventsPerPinPerDay(ctx context.Context, from, to time.Time) (map[string]map[string]int64, error) {
	type row struct {
		Day   string `gorm:"column:day"`
		Pin   string `gorm:"column:pin"`
		Count int64  `gorm:"column:count"`
	}
	rows := []row{}
	if err := s.DB.WithContext(ctx).
		Table(`"AttendanceEvent"`).
		Select(`to_char(date_trunc('day', "occurredAt" AT TIME ZONE 'Asia/Jakarta'), 'YYYY-MM-DD') AS day, "machinePin" AS pin, count(*) AS count`).
		Where(`("occurredAt" AT TIME ZONE 'Asia/Jakarta') >= ?::timestamp AND ("occurredAt" AT TIME ZONE 'Asia/Jakarta') < ?::timestamp`,
			from.Format("2006-01-02")+" 00:00:00", to.AddDate(0, 0, 1).Format("2006-01-02")+" 00:00:00").
		Group("1, 2").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("count attendance events: %w", err)
	}
	byDay := map[string]map[string]int64{}
	for _, row := range rows {
		perPin := byDay[row.Day]
		if perPin == nil {
			perPin = map[string]int64{}
			byDay[row.Day] = perPin
		}
		perPin[row.Pin] = row.Count
	}
	return byDay, nil
}

// ApprovedCutiPerUser mengembalikan rentang cuti yang disetujui per pegawai.
// Jenis cuti sakit (SAKIT/SAKIT_BERKEPANJANGAN) menjadi status 'S'; jenis lain
// menjadi status 'C'.
func (s *GormAdminStore) ApprovedCutiPerUser(ctx context.Context, scopeIDs []string, from, to time.Time) (map[string][]CutiWindow, error) {
	type row struct {
		UserID string    `gorm:"column:userId"`
		Jenis  string    `gorm:"column:jenis"`
		Start  time.Time `gorm:"column:start"`
		End    time.Time `gorm:"column:end"`
	}
	rows := []row{}
	query := s.DB.WithContext(ctx).
		Table(`"Cuti"`).
		Select(`"userId", "jenis", "tanggalMulai" AS start, "tanggalSelesai" AS end`).
		Where(`"status" = ? AND "tanggalSelesai" >= ? AND "tanggalMulai" <= ?`, models.StatusDisetujui, from, to)
	if optionalScopeIDs(scopeIDs) {
		if len(scopeIDs) == 0 {
			return map[string][]CutiWindow{}, nil
		}
		query = query.Where(`"userId" IN (SELECT "id" FROM "user" WHERE "homebaseId" IN ?)`, scopeIDs)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("read approved cuti: %w", err)
	}
	byUser := map[string][]CutiWindow{}
	for _, row := range rows {
		status := models.StatusPresensiCuti
		if row.Jenis == string(models.JenisCutiSakit) || row.Jenis == string(models.JenisCutiSakitBerkepanjangan) {
			status = models.StatusPresensiSakit
		}
		start := time.Date(row.Start.Year(), row.Start.Month(), row.Start.Day(), 0, 0, 0, 0, time.UTC)
		end := time.Date(row.End.Year(), row.End.Month(), row.End.Day(), 0, 0, 0, 0, time.UTC)
		byUser[row.UserID] = append(byUser[row.UserID], CutiWindow{Start: start, End: end, Status: status})
	}
	return byUser, nil
}

// HariLiburDates mengembalikan himpunan tanggal libur nasional dalam rentang.
func (s *GormAdminStore) HariLiburDates(ctx context.Context, from, to time.Time) (map[string]bool, error) {
	type row struct {
		Day string `gorm:"column:day"`
	}
	rows := []row{}
	if err := s.DB.WithContext(ctx).
		Table(`"HariLibur"`).
		Select(`to_char("tanggal", 'YYYY-MM-DD') AS day`).
		Where(`"tanggal" >= ? AND "tanggal" < ?`, from, to.AddDate(0, 0, 1)).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("read hari libur: %w", err)
	}
	libur := map[string]bool{}
	for _, row := range rows {
		libur[row.Day] = true
	}
	return libur, nil
}

// PresensiExists membaca sumber baris Presensi untuk (userId, tanggal);
// found=false bila baris belum ada. Sumber kosong dianggap tidak ada nilai.
func (s *GormAdminStore) PresensiExists(ctx context.Context, userID string, tanggal time.Time) (string, bool, error) {
	var row struct {
		Sumber *string `gorm:"column:sumber"`
	}
	err := s.DB.WithContext(ctx).
		Table(`"Presensi"`).
		Select(`"sumber"`).
		Where(`"userId" = ? AND "tanggal" = ?`, userID, tanggal).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read presensi: %w", err)
	}
	if row.Sumber == nil {
		return "", true, nil
	}
	return *row.Sumber, true, nil
}

// InsertPresensi menulis baris Presensi baru.
func (s *GormAdminStore) InsertPresensi(ctx context.Context, row models.Presensi) error {
	if err := s.DB.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("presensi sudah ada untuk tanggal tersebut")
		}
		return fmt.Errorf("insert presensi: %w", err)
	}
	return nil
}

// UpdatePresensiDerived memperbarui baris turunan mesin. Bila baris sudah
// berubah menjadi manual di tengah rangkaian, error dikembalikan agar
// rangkaian gagal dengan aman (bukan menimpa data manual).
func (s *GormAdminStore) UpdatePresensiDerived(ctx context.Context, row models.Presensi, actorID string) error {
	result := s.DB.WithContext(ctx).
		Model(&models.Presensi{}).
		Where(`"userId" = ? AND "tanggal" = ? AND "sumber" = ?`, row.UserId, row.Tanggal, "derivasi_mesin").
		Updates(map[string]any{
			"status":     row.Status,
			"keterangan": row.Keterangan,
			"sumber":     "derivasi_mesin",
			"updatedBy":  actorID,
		})
	if result.Error != nil {
		return fmt.Errorf("update presensi: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("baris presensi tidak lagi milik derivasi mesin; rangkaian dihentikan")
	}
	return nil
}

// UpsertPresensiManual menulis baris manual dan menulis audit dalam satu transaksi.
func (s *GormAdminStore) UpsertPresensiManual(ctx context.Context, row models.Presensi, actorID string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		homebase, err := s.userHomebase(ctx, tx, row.UserId)
		if err != nil {
			return err
		}
		var existing models.Presensi
		err = tx.Select("id").Where(`"userId" = ? AND "tanggal" = ?`, row.UserId, row.Tanggal).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row.IDModel.ID = ""
			if err := tx.Create(&row).Error; err != nil {
				if isUniqueViolation(err) {
					return fmt.Errorf("presensi sudah ada untuk tanggal tersebut")
				}
				return fmt.Errorf("insert presensi: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("read presensi: %w", err)
		} else {
			if err := tx.Model(&existing).Updates(map[string]any{
				"status":     row.Status,
				"keterangan": row.Keterangan,
				"sumber":     "manual",
				"updatedBy":  actorID,
			}).Error; err != nil {
				return fmt.Errorf("update presensi: %w", err)
			}
		}
		targetID := row.UserId + "@" + row.Tanggal.Format("2006-01-02")
		return audit.Write(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionPresensiUpserted,
			TargetID: &targetID,
			UnitID:   homebase,
			Note:     ptr(string(row.Status)),
		})
	})
}

// CountAlpaPerMonth menghitung hari alpa ('A') per pegawai per bulan.
func (s *GormAdminStore) CountAlpaPerMonth(ctx context.Context, scopeIDs []string, from, to time.Time) ([]MonthlyAlpa, error) {
	type row struct {
		UserID string `gorm:"column:userId"`
		Month  string `gorm:"column:month"`
		Count  int64  `gorm:"column:count"`
	}
	rows := []row{}
	query := s.DB.WithContext(ctx).
		Table(`"Presensi"`).
		Select(`"userId", to_char(date_trunc('month', "tanggal"), 'YYYY-MM') AS month, count(*) AS count`).
		Where(`"status" = ? AND "tanggal" >= ? AND "tanggal" < ?`, models.StatusPresensiAlpa, from, to).
		Group("1, 2")
	if optionalScopeIDs(scopeIDs) {
		if len(scopeIDs) == 0 {
			return []MonthlyAlpa{}, nil
		}
		query = query.Where(`"userId" IN (SELECT "id" FROM "user" WHERE "homebaseId" IN ?)`, scopeIDs)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("count alpa per month: %w", err)
	}
	result := make([]MonthlyAlpa, 0, len(rows))
	for _, row := range rows {
		result = append(result, newMonthlyAlpa(row.UserID, row.Month, row.Count))
	}
	return result, nil
}

// EmployeesInScope mengembalikan id/nama/unit pegawai dalam cakupan.
func (s *GormAdminStore) EmployeesInScope(ctx context.Context, scopeIDs []string, ids []string) ([]models.User, error) {
	if len(ids) == 0 {
		return []models.User{}, nil
	}
	var rows []models.User
	query := s.DB.WithContext(ctx).Table(`"user"`).Select(`"id", "nama", "homebaseId"`).Where(`"id" IN ?`, ids)
	if optionalScopeIDs(scopeIDs) {
		if len(scopeIDs) == 0 {
			return []models.User{}, nil
		}
		query = query.Where(`"homebaseId" IN ?`, scopeIDs)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("read employees in scope: %w", err)
	}
	return rows, nil
}

// WriteAudit menulis satu entri jejak audit dalam transaksi sendiri.
func (s *GormAdminStore) WriteAudit(ctx context.Context, entry audit.Entry) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return audit.Write(ctx, tx, entry)
	})
}
