package repository

import (
	"context"
	"errors"
	"fmt"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ListCuti membaca daftar pengajuan cuti sesuai filter, cakupan, dan pagination.
func (s GormStore) ListCuti(ctx context.Context, options listing.Options) ([]models.Cuti, int64, error) {
	query := s.DB.WithContext(ctx).Model(&models.Cuti{})
	if options.UserID != "" {
		query = query.Where("userId = ?", options.UserID)
	}
	if len(options.ScopeHomebaseIDs) > 0 {
		query = query.Where("userId IN (SELECT id FROM \"user\" WHERE homebaseId IN ?)", options.ScopeHomebaseIDs)
	}
	if options.Search != "" {
		pattern := "%" + escapeLike(options.Search) + "%"
		query = query.Where("(jenis ILIKE ? ESCAPE '\\' OR alasan ILIKE ? ESCAPE '\\' OR alamatCuti ILIKE ? ESCAPE '\\')", pattern, pattern, pattern)
	}
	if options.Status != "" {
		query = query.Where("status = ?", options.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count cuti: %w", err)
	}
	var items []models.Cuti
	order := listing.OrderClause(domain.CutiSortColumns, options.Sort, options.Order, "tanggalMulai")
	err := query.Order(order).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list cuti: %w", err)
	}
	return items, total, nil
}

// GetCuti membaca satu pengajuan cuti menurut id.
func (s GormStore) GetCuti(ctx context.Context, id string) (models.Cuti, error) {
	var item models.Cuti
	err := s.DB.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Cuti{}, domain.ErrNotFound
		}
		return models.Cuti{}, fmt.Errorf("read cuti: %w", err)
	}
	return item, nil
}

// CreateCuti membuat pengajuan cuti baru dan menulis jejak audit dalam satu
// transaksi. Status awal selalu MENUNGGU.
func (s GormStore) CreateCuti(ctx context.Context, actorID string, item models.Cuti) (models.Cuti, error) {
	item.Status = models.StatusMenunggu
	item.DisetujuiBy = nil
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		return createAudit(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionCutiCreated,
			TargetID: &item.ID,
			UnitID:   &item.UserId,
			Note:     strPtr(string(item.Jenis)),
		})
	})
	return item, err
}

// UpdateCuti mengubah field pengajuan cuti yang masih MENUNGGU.
//
// Catatan yang diterima adalah keadaan akhir pengajuan, bukan daftar
// perubahan, sehingga kolom yang boleh ditulis tetap satu sumber kebenaran.
func (s GormStore) UpdateCuti(ctx context.Context, actorID, id string, merged models.Cuti) (models.Cuti, error) {
	var item models.Cuti
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		if item.Status != models.StatusMenunggu {
			return domain.ErrValidation
		}
		updates := cutiColumns(merged)
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload cuti: %w", err)
		}
		return createAudit(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionCutiUpdated,
			TargetID: optionalRef(id),
			UnitID:   &item.UserId,
		})
	})
	return item, err
}

// ApproveCuti menyetujui pengajuan cuti; hanya bisa dilakukan pada status
// MENUNGGU. disetujuiBy diisi oleh actor.
func (s GormStore) ApproveCuti(ctx context.Context, actorID, id string, catatan *string) (models.Cuti, error) {
	var item models.Cuti
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		if item.Status != models.StatusMenunggu {
			return domain.ErrValidation
		}
		updates := map[string]any{
			"status":      models.StatusDisetujui,
			"disetujuiBy": actorID,
			"updatedBy":   actorID,
		}
		if catatan != nil {
			updates["catatan"] = catatan
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload cuti: %w", err)
		}
		note := "cuti disetujui"
		if catatan != nil {
			note = *catatan
		}
		return createAudit(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionCutiApproved,
			TargetID: &item.ID,
			UnitID:   &item.UserId,
			Note:     &note,
		})
	})
	return item, err
}

// RejectCuti menolak pengajuan cuti; hanya bisa dilakukan pada status MENUNGGU.
func (s GormStore) RejectCuti(ctx context.Context, actorID, id string, catatan *string) (models.Cuti, error) {
	var item models.Cuti
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return classifyRecordError(err)
		}
		if item.Status != models.StatusMenunggu {
			return domain.ErrValidation
		}
		updates := map[string]any{
			"status":      models.StatusDitolak,
			"disetujuiBy": actorID,
			"updatedBy":   actorID,
		}
		if catatan != nil {
			updates["catatan"] = catatan
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return classifyRecordError(err)
		}
		if err := tx.First(&item, "id = ?", id).Error; err != nil {
			return fmt.Errorf("reload cuti: %w", err)
		}
		note := "cuti ditolak"
		if catatan != nil {
			note = *catatan
		}
		return createAudit(ctx, tx, audit.Entry{
			ActorID:  actorID,
			Action:   audit.ActionCutiRejected,
			TargetID: &item.ID,
			UnitID:   &item.UserId,
			Note:     &note,
		})
	})
	return item, err
}

// cutiColumns menyusun map pembaruan dari keadaan akhir pengajuan.
//
// Kolom yang ditulis dibatasi pada field yang boleh diubah ketika pengajuan
// masih MENUNGGU. Kolom status, approver, dan cap waktu ditangani jalur
// persetujuan sehingga tidak dapat ditulis dari permintaan pengguna.
func cutiColumns(item models.Cuti) map[string]any {
	return map[string]any{
		"jenis":              item.Jenis,
		"tanggalMulai":       item.TanggalMulai,
		"tanggalSelesai":     item.TanggalSelesai,
		"alamatCuti":         item.AlamatCuti,
		"alasan":             item.Alasan,
		"subJenis":           item.SubJenis,
		"nomorSurat":         item.NomorSurat,
		"tanggalSuratDokter": item.TanggalSuratDokter,
		"jenisEviden":        item.JenisEviden,
		"fileEvidenUrl":      item.FileEvidenUrl,
		"waktuEviden":        item.WaktuEviden,
		"catatanEviden":      item.CatatanEviden,
		"persenGaji":         item.PersenGaji,
		"untukHaji":          item.UntukHaji,
	}
}

func strPtr(s string) *string { return &s }

// UserHomebaseID mencari homebaseId sebuah pegawai dari tabel user.
func (s GormStore) UserHomebaseID(ctx context.Context, userID string) (string, error) {
	var homebaseID string
	err := s.DB.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Pluck("homebaseId", &homebaseID).Error
	if err != nil {
		return "", fmt.Errorf("read user homebase: %w", err)
	}
	return homebaseID, nil
}
