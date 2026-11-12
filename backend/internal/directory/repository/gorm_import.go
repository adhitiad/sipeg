package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/models"

	"gorm.io/gorm"
)

// identityChunkSize membatasi jumlah baris per query pemeriksaan identitas.
//
// Nilai ini menjaga daftar parameter SQL tetap wajar pada impor besar dan
// membatasi ukuran hasil yang dikembalikan database per query.
const identityChunkSize = 250

// CreateEmployees membuat banyak pegawai sekaligus. Bila salah satu baris
// bentrok, seluruh transaksi dibatalkan.
//
// Pemeriksaan duplikat memakai satu query per chunk identitas, bukan satu
// query per baris, sehingga jumlah statement tidak tumbuh sebanding dengan
// jumlah baris impor. Jejak audit ditulis dalam satu operasi batch.
func (s GormStore) CreateEmployees(ctx context.Context, actorID string, employees []models.User) ([]models.User, error) {
	if len(employees) == 0 {
		return nil, domain.ErrConflict
	}
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := scanExistingIdentities(tx, employees)
		if err != nil {
			return err
		}
		if index, field, conflict := existing.firstConflict(employees); conflict {
			return &domain.EmployeeConflictError{Index: index, Field: field}
		}

		entries := make([]audit.Entry, 0, len(employees)+1)
		for i := range employees {
			employees[i].EmailVerified = false
			employees[i].Aktif = true
			if err := tx.Create(&employees[i]).Error; err != nil {
				if errors.Is(classifyRecordError(err), domain.ErrConflict) {
					return &domain.EmployeeConflictError{Index: i, Field: "identity"}
				}
				return err
			}
			entries = append(entries, audit.Entry{
				ActorID:  actorID,
				Action:   audit.ActionEmployeeCreated,
				TargetID: &employees[i].ID,
				UnitID:   employees[i].HomebaseId,
			})
		}
		// Entri ringkasan membedakan impor massal dari pembuatan satuan.
		note := fmt.Sprintf("%d pegawai diimpor", len(employees))
		entries = append(entries, audit.Entry{
			ActorID: actorID,
			Action:  audit.ActionEmployeeImported,
			Note:    &note,
			UnitID:  importUnitID(employees),
		})
		return audit.WriteBatch(ctx, tx, entries)
	})
	if err != nil {
		return nil, err
	}
	return employees, nil
}

// existingIdentity menyimpan nilai identitas pegawai yang sudah ada di
// database. Email dinormalkan huruf kecil, sedangkan NIP, NIDN, dan NIDK
// dibandingkan apa adanya sesuai indeks unik.
type existingIdentity struct {
	emails map[string]struct{}
	nips   map[string]struct{}
	nidns  map[string]struct{}
	nidks  map[string]struct{}
}

// scanExistingIdentities mencari nilai identitas yang sudah terpakai.
//
// Pencarian dilakukan per chunk agar daftar parameter SQL tidak membengkak,
// dan seluruh chunk diperiksa sebelum satu pun baris ditulis.
func scanExistingIdentities(tx *gorm.DB, employees []models.User) (existingIdentity, error) {
	found := existingIdentity{
		emails: map[string]struct{}{},
		nips:   map[string]struct{}{},
		nidns:  map[string]struct{}{},
		nidks:  map[string]struct{}{},
	}
	for start := 0; start < len(employees); start += identityChunkSize {
		end := start + identityChunkSize
		if end > len(employees) {
			end = len(employees)
		}
		if err := found.scanChunk(tx, employees[start:end]); err != nil {
			return found, err
		}
	}
	return found, nil
}

// scanChunk memeriksa satu potongan baris impor terhadap data yang sudah ada.
func (found *existingIdentity) scanChunk(tx *gorm.DB, chunk []models.User) error {
	emails := make([]string, 0, len(chunk))
	nips := make([]string, 0, len(chunk))
	nidns := make([]string, 0, len(chunk))
	nidks := make([]string, 0, len(chunk))
	for _, employee := range chunk {
		emails = append(emails, strings.ToLower(employee.Email))
		if employee.Nip != nil {
			nips = append(nips, *employee.Nip)
		}
		if employee.Nidn != nil {
			nidns = append(nidns, *employee.Nidn)
		}
		if employee.Nidk != nil {
			nidks = append(nidks, *employee.Nidk)
		}
	}

	var existingEmails []string
	if len(emails) > 0 {
		err := tx.Model(&models.User{}).Where("LOWER(email) IN ?", emails).
			Pluck("email", &existingEmails).Error
		if err != nil {
			return fmt.Errorf("scan existing employee email: %w", err)
		}
		for _, email := range existingEmails {
			found.emails[strings.ToLower(email)] = struct{}{}
		}
	}

	var existingNips []string
	if len(nips) > 0 {
		err := tx.Model(&models.User{}).Where("nip IN ?", nips).
			Pluck("nip", &existingNips).Error
		if err != nil {
			return fmt.Errorf("scan existing employee nip: %w", err)
		}
		for _, nip := range existingNips {
			found.nips[nip] = struct{}{}
		}
	}

	var existingNidns []string
	if len(nidns) > 0 {
		err := tx.Model(&models.User{}).Where("nidn IN ?", nidns).
			Pluck("nidn", &existingNidns).Error
		if err != nil {
			return fmt.Errorf("scan existing employee nidn: %w", err)
		}
		for _, nidn := range existingNidns {
			found.nidns[nidn] = struct{}{}
		}
	}

	var existingNidks []string
	if len(nidks) > 0 {
		err := tx.Model(&models.User{}).Where("nidk IN ?", nidks).
			Pluck("nidk", &existingNidks).Error
		if err != nil {
			return fmt.Errorf("scan existing employee nidk: %w", err)
		}
		for _, nidk := range existingNidks {
			found.nidks[nidk] = struct{}{}
		}
	}
	return nil
}

// firstConflict mengembalikan baris pertama yang identitasnya bentrok dengan
// data yang sudah ada, beserta nama kolomnya.
//
// Urutan pemeriksaan mengikuti urutan kolom pada model: email, NIP, NIDN, lalu
// NIDK. Bentrok di dalam batch sendiri bukan hasil pemeriksaan ini; yang
// tertangkap indeks unik saat penyimpanan.
func (found existingIdentity) firstConflict(employees []models.User) (int, string, bool) {
	for index, employee := range employees {
		if _, exists := found.emails[strings.ToLower(employee.Email)]; exists {
			return index, "email", true
		}
		if employee.Nip != nil {
			if _, exists := found.nips[*employee.Nip]; exists {
				return index, "nip", true
			}
		}
		if employee.Nidn != nil {
			if _, exists := found.nidns[*employee.Nidn]; exists {
				return index, "nidn", true
			}
		}
		if employee.Nidk != nil {
			if _, exists := found.nidks[*employee.Nidk]; exists {
				return index, "nidk", true
			}
		}
	}
	return 0, "", false
}
