package attendance

import (
	"context"
	"errors"
	"strings"
	"time"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"
)

var (
	// ErrPinDuplicate menandai machinePin atau userId yang sudah terpetakan.
	ErrPinDuplicate = errors.New("machine pin or employee is already mapped")
	// ErrPinNotFound menandai pemetaan yang diminta tidak ada (atau di luar cakupan).
	ErrPinNotFound = errors.New("machine pin mapping not found")
)

// ListPins mengembalikan seluruh pemetaan dalam cakupan principal.
func (s *AdminService) ListPins(ctx context.Context, principal auth.Principal) ([]PinView, error) {
	scopeIDs, err := s.scopeIDs(ctx, principal)
	if err != nil {
		return nil, err
	}
	return s.store.ListPins(ctx, scopeIDs)
}

// resolveTarget memvalidasi bahwa pegawai userID berada dalam cakupan principal
// dan mengembalikan identitasnya.
func (s *AdminService) resolveTarget(ctx context.Context, principal auth.Principal, userID string) (models.User, error) {
	scopeIDs, err := s.scopeIDs(ctx, principal)
	if err != nil {
		return models.User{}, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return models.User{}, ErrPresensiForbidden
	}
	user, err := s.store.UserIdentityOf(ctx, userID)
	if err != nil || !inScope(scopeIDs, user.HomebaseId) {
		return models.User{}, ErrPresensiForbidden
	}
	return user, nil
}

// CreatePin memetakan machinePin ke pegawai (userId).
func (s *AdminService) CreatePin(ctx context.Context, principal auth.Principal, machinePin, userID string, catatan *string) (PinView, error) {
	user, err := s.resolveTarget(ctx, principal, userID)
	if err != nil {
		return PinView{}, err
	}
	machinePin = strings.TrimSpace(machinePin)
	if machinePin == "" {
		return PinView{}, ErrPresensiForbidden
	}
	created, err := s.store.CreatePin(ctx, machinePin, user.ID, catatan, principal.UserID)
	if err != nil {
		return PinView{}, err
	}
	return PinView{
		ID:         created.ID,
		MachinePin: created.MachinePin,
		UserID:     created.UserId,
		Nama:       user.Nama,
		Aktif:      created.Aktif,
		Catatan:    created.Catatan,
		CreatedAt:  timeutil.FormatDateTime(created.CreatedAt),
	}, nil
}

// DeletePin menghapus pemetaan dalam cakupan principal.
func (s *AdminService) DeletePin(ctx context.Context, principal auth.Principal, pinID string) error {
	scopeIDs, err := s.scopeIDs(ctx, principal)
	if err != nil {
		return err
	}
	if strings.TrimSpace(pinID) == "" {
		return ErrPinNotFound
	}
	return s.store.DeletePin(ctx, pinID, scopeIDs, principal.UserID)
}

// ImportPinRow adalah satu baris impor massal pemetaan PIN.
type ImportPinRow struct {
	Row        int
	MachinePin string
	Email      string
}

// ImportPinIssue melaporkan satu baris impor yang gagal.
type ImportPinIssue struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ImportPins memproses impor massal pemetaan machinePin → pegawai (email).
// Baris yang gagal dikumpulkan ke `issues`; baris valid tetap diproses.
// Import bersifat idempoten terhadap pegawai yang sama: pin yang sudah
// terpetakan ke pegawai lain ditolak dengan `duplicate`.
func (s *AdminService) ImportPins(ctx context.Context, principal auth.Principal, rows []ImportPinRow) (int, []ImportPinIssue, error) {
	scopeIDs, err := s.scopeIDs(ctx, principal)
	if err != nil {
		return 0, nil, err
	}
	issues := []ImportPinIssue{}
	imported := 0
	for _, row := range rows {
		machinePin := strings.TrimSpace(row.MachinePin)
		email := strings.TrimSpace(strings.ToLower(row.Email))
		if machinePin == "" || email == "" {
			issues = append(issues, ImportPinIssue{Row: row.Row, Field: "machinePin", Code: "required", Message: "pin dan email wajib diisi"})
			continue
		}
		user, err := s.store.UserByEmail(ctx, email)
		if err != nil {
			issues = append(issues, ImportPinIssue{Row: row.Row, Field: "email", Code: "not_found", Message: "pegawai dengan email tersebut tidak ditemukan"})
			continue
		}
		if !inScope(scopeIDs, user.HomebaseId) {
			issues = append(issues, ImportPinIssue{Row: row.Row, Field: "email", Code: "out_of_scope", Message: "pegawai di luar cakupan unit penugasan"})
			continue
		}
		if _, err := s.store.CreatePin(ctx, machinePin, user.ID, nil, principal.UserID); err != nil {
			issues = append(issues, ImportPinIssue{Row: row.Row, Field: "machinePin", Code: "duplicate", Message: "machinePin atau pegawai sudah terpetakan"})
			continue
		}
		imported++
	}
	return imported, issues, nil
}

// UpsertPresensiManual mengisi/mengganti satu baris Presensi secara manual.
// Baris manual tidak pernah ditimpa oleh derivasi.
func (s *AdminService) UpsertPresensi(ctx context.Context, principal auth.Principal, input UpsertPresensiInput) (models.Presensi, bool, error) {
	user, err := s.resolveTarget(ctx, principal, input.UserID)
	if err != nil {
		return models.Presensi{}, false, err
	}
	if input.Tanggal.IsZero() || !isValidPresensiStatus(input.Status) {
		return models.Presensi{}, false, ErrPresensiForbidden
	}
	row := models.Presensi{
		UserId:     user.ID,
		Tanggal:    time.Date(input.Tanggal.Year(), input.Tanggal.Month(), input.Tanggal.Day(), 0, 0, 0, 0, time.UTC),
		Status:     input.Status,
		Keterangan: input.Keterangan,
		Sumber:     ptr("manual"),
	}
	login := principal.UserID
	row.UpdatedBy = &login
	created := true
	if _, found, err := s.store.PresensiExists(ctx, user.ID, row.Tanggal); err == nil && found {
		created = false
	}
	if err := s.store.UpsertPresensiManual(ctx, row, principal.UserID); err != nil {
		return models.Presensi{}, false, err
	}
	return row, created, nil
}

func isValidPresensiStatus(status models.StatusPresensi) bool {
	switch status {
	case models.StatusPresensiHadir, models.StatusPresensiIzin, models.StatusPresensiSakit,
		models.StatusPresensiAlpa, models.StatusPresensiCuti, models.StatusPresensiLibur:
		return true
	default:
		return false
	}
}

func ptr[T any](value T) *T { return &value }
