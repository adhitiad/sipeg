package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormStore adalah adapter persistensi berbasis GORM. Ia memenuhi seluruh
// port repository pada area direktori.
type GormStore struct {
	DB *gorm.DB
}

// ListEmployees membaca daftar pegawai aktif sesuai filter dan pagination.
func (s GormStore) ListEmployees(ctx context.Context, options listing.Options) ([]models.User, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.User{}).Where("aktif = ?", true)
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("(nama ILIKE ? ESCAPE '\\' OR nip ILIKE ? ESCAPE '\\')", pattern, pattern)
	}
	if options.HomebaseID != "" {
		query = query.Where(`"homebaseId" = ?`, options.HomebaseID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where(`"homebaseId" IN ?`, options.ScopeHomebaseIDs)
	}
	if options.UserID != "" {
		query = query.Where("id = ?", options.UserID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count employees: %w", err)
	}
	var employees []models.User
	order := listing.OrderClause(domain.EmployeeSortColumns, options.Sort, options.Order, "nama")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&employees).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list employees: %w", err)
	}
	return employees, total, nil
}

// GetEmployee membaca satu pegawai aktif.
func (s GormStore) GetEmployee(ctx context.Context, id string) (models.User, error) {
	var employee models.User
	err := s.DB.WithContext(ctx).Where("id = ? AND aktif = ?", id, true).First(&employee).Error
	return employee, classifyRecordError(err)
}

// CreateEmployee membuat satu pegawai dan mencatat jejak audit dalam satu
// transaksi.
func (s GormStore) CreateEmployee(ctx context.Context, actorID string, employee models.User) (models.User, error) {
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := validateEmployeeReferences(tx, employee); err != nil {
			return err
		}
		if err := tx.Create(&employee).Error; err != nil {
			return classifyRecordError(err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionEmployeeCreated, TargetID: &employee.ID, UnitID: employee.HomebaseId})
	})
	return employee, err
}

// UpdateEmployee memperbarui profil pegawai dan mencatat jejak audit dalam
// satu transaksi.
func (s GormStore) UpdateEmployee(ctx context.Context, actorID, id string, patch domain.EmployeePatch) (models.User, error) {
	updates := employeeColumns(patch)
	var employee models.User
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND aktif = ?", id, true).First(&employee).Error; err != nil {
			return classifyRecordError(err)
		}
		if homebaseID, ok := updates["homebaseId"]; ok && homebaseID != nil {
			if err := requireActiveOrganization(tx, homebaseID.(string)); err != nil {
				return err
			}
		}
		if categoryID, ok := updates["kategoriPegawaiId"]; ok && categoryID != nil {
			var count int64
			if err := tx.Model(&models.Kategori{}).Where("id = ? AND aktif = ?", categoryID, true).Count(&count).Error; err != nil {
				return fmt.Errorf("validate employee category: %w", err)
			}
			if count == 0 {
				return domain.ErrInvalidRef
			}
		}
		if err := tx.Model(&employee).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&employee, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload employee: %w", err)
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionEmployeeUpdated, TargetID: optionalRef(id), UnitID: employee.HomebaseId})
	})
	return employee, err
}

// ArchiveEmployee menonaktifkan pegawai tanpa menghapus barisnya.
func (s GormStore) ArchiveEmployee(ctx context.Context, actorID, id string) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var employee models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND aktif = ?", id, true).First(&employee).Error; err != nil {
			return classifyRecordError(err)
		}
		result := tx.Model(&models.User{}).Where("id = ? AND aktif = ?", id, true).Update("aktif", false)
		if result.Error != nil {
			return fmt.Errorf("archive employee: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return createAudit(ctx, tx, audit.Entry{ActorID: actorID, Action: audit.ActionEmployeeArchived, TargetID: optionalRef(id), UnitID: employee.HomebaseId})
	})
}

// AssignEmployeeRole mengubah role dan mencatat audit dalam satu transaksi.
func (s GormStore) AssignEmployeeRole(ctx context.Context, actorID, id, roleName string) (models.User, error) {
	var employee models.User
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND aktif = ?", id, true).First(&employee).Error; err != nil {
			return classifyRecordError(err)
		}
		var role models.Role
		if err := tx.Where("name = ?", roleName).First(&role).Error; err != nil {
			return classifyRoleError(err)
		}
		if employee.RoleId != nil && *employee.RoleId == role.ID {
			return nil
		}
		if err := tx.Model(&employee).Updates(map[string]any{"roleId": role.ID}).Error; err != nil {
			return err
		}
		employee.RoleId = &role.ID
		note := "role=" + roleName
		return createAudit(ctx, tx, audit.Entry{
			ActorID: actorID, Action: audit.ActionEmployeeRoleUpdated,
			TargetID: optionalRef(id), UnitID: employee.HomebaseId, Note: &note,
		})
	})
	if err != nil {
		return models.User{}, err
	}
	return employee, err
}

func classifyRoleError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrInvalidRef
	}
	return err
}

// employeeColumns memetakan patch domain menjadi kolom database.
//
// Kolom memakai nama skema acuan; nilai yang dikosongkan pada permintaan
// disimpan sebagai NULL.
func employeeColumns(patch domain.EmployeePatch) map[string]any {
	updates := make(map[string]any)
	if patch.Nama.Present {
		updates["nama"] = strings.TrimSpace(patch.Nama.Value)
	}
	if patch.NIP.Present {
		nip := strings.TrimSpace(patch.NIP.Value)
		if patch.NIP.Null || nip == "" {
			updates["nip"] = nil
		} else {
			updates["nip"] = nip
		}
	}
	if patch.NIDN.Present {
		nidn := strings.TrimSpace(patch.NIDN.Value)
		if patch.NIDN.Null || nidn == "" {
			updates["nidn"] = nil
		} else {
			updates["nidn"] = nidn
		}
	}
	if patch.NIDK.Present {
		nidk := strings.TrimSpace(patch.NIDK.Value)
		if patch.NIDK.Null || nidk == "" {
			updates["nidk"] = nil
		} else {
			updates["nidk"] = nidk
		}
	}
	if patch.KategoriPegawaiID.Present {
		if patch.KategoriPegawaiID.Null {
			updates["kategoriPegawaiId"] = nil
		} else {
			updates["kategoriPegawaiId"] = patch.KategoriPegawaiID.Value
		}
	}
	if patch.HomebaseID.Present {
		if patch.HomebaseID.Null {
			updates["homebaseId"] = nil
		} else {
			updates["homebaseId"] = patch.HomebaseID.Value
		}
	}
	if patch.JabatanStruktural.Present {
		if patch.JabatanStruktural.Null {
			updates["jabatan_struktural"] = nil
		} else {
			updates["jabatan_struktural"] = strings.TrimSpace(patch.JabatanStruktural.Value)
		}
	}
	if patch.JabatanFungsional.Present {
		if patch.JabatanFungsional.Null {
			updates["jabatan_fungsional"] = nil
		} else {
			updates["jabatan_fungsional"] = strings.TrimSpace(patch.JabatanFungsional.Value)
		}
	}
	return updates
}

// validateEmployeeReferences memastikan referensi pegawai menunjuk record aktif.
func validateEmployeeReferences(tx *gorm.DB, employee models.User) error {
	if employee.HomebaseId != nil {
		if err := requireActiveOrganization(tx, *employee.HomebaseId); err != nil {
			return err
		}
	}
	if employee.KategoriPegawaiId != nil {
		var count int64
		err := tx.Model(&models.Kategori{}).Where("id = ? AND aktif = ?", *employee.KategoriPegawaiId, true).Count(&count).Error
		if err != nil {
			return fmt.Errorf("validate employee category: %w", err)
		}
		if count == 0 {
			return domain.ErrInvalidRef
		}
	}
	return nil
}

// requireActiveOrganization memastikan unit yang dirujuk ada dan aktif.
func requireActiveOrganization(tx *gorm.DB, id string) error {
	var organization models.Homebase
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND aktif = ?", id, true).First(&organization).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrInvalidRef
		}
		return fmt.Errorf("validate organization: %w", err)
	}
	return nil
}

// createAudit menulis satu entri jejak audit di dalam transaksi berjalan.
func createAudit(ctx context.Context, tx *gorm.DB, entry audit.Entry) error {
	return audit.Write(ctx, tx, entry)
}

// optionalRef mengubah string kosong menjadi nil agar kolom nullable tidak
// diisi dengan string kosong.
func optionalRef(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// importUnitID menentukan unit cakupan untuk entri ringkasan impor.
// Seluruh baris impor oleh satu operator berada pada unit yang sama.
func importUnitID(employees []models.User) *string {
	for _, employee := range employees {
		if employee.HomebaseId != nil {
			return employee.HomebaseId
		}
	}
	return nil
}

// classifyRecordError memetakan error GORM menjadi error domain.
func classifyRecordError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return domain.ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return domain.ErrConflict
	default:
		return err
	}
}

// escapeLike meloloskan karakter khusus pada pola ILIKE.
func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}
