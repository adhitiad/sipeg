package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/directory/app"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/models"
	payrollapp "simpeg/backend/internal/payroll/app"
	skpapp "simpeg/backend/internal/skp/app"
	"simpeg/backend/internal/timeutil"
	tridharmaapp "simpeg/backend/internal/tridharma/app"
)

// directoryAPI memegang use case yang dipakai oleh handler direktori.
type directoryAPI struct {
	employees     *app.EmployeeService
	organizations *app.OrganizationService
	categories    *app.CategoryService
	audits        *app.AuditService
	skp           *skpapp.SKPService
	tridharma     *tridharmaapp.TridharmaService
	keluarga      *app.KeluargaService
	golongan      *app.GolonganService
	arsip         *app.ArsipService
	sertifikasi   *app.SertifikasiService
	pelatihan     *app.PelatihanService
	sister        *app.SisterService
	penghargaan   *app.PenghargaanService
	tunjangan     *payrollapp.TunjanganService
	potongan      *payrollapp.PotonganService
	leave         *app.LeaveService
}

// EmployeeDTO adalah bentuk publik data pegawai.
type EmployeeDTO struct {
	ID                string    `json:"id"`
	Nama              string    `json:"nama"`
	Email             string    `json:"email"`
	NIP               *string   `json:"nip"`
	NIDN              *string   `json:"nidn"`
	NIDK              *string   `json:"nidk"`
	KategoriPegawaiID *string   `json:"kategoriPegawaiId"`
	HomebaseID        *string   `json:"homebaseId"`
	JabatanStruktural *string   `json:"jabatanStruktural"`
	JabatanFungsional *string   `json:"jabatanFungsional"`
	Aktif             bool      `json:"aktif"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// OrganizationDTO adalah bentuk publik unit organisasi.
type OrganizationDTO struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description *string                `json:"description"`
	Jenis       models.JenisOrganisasi `json:"jenis"`
	ParentID    *string                `json:"parentId"`
	Aktif       bool                   `json:"aktif"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// PageDTO adalah bentuk umum hasil daftar berpaginasi.
type PageDTO[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request must contain one JSON value")
		}
		return err
	}
	return nil
}

func pagination(r *http.Request) (int, int, error) {
	page, err := queryInt(r.URL.Query(), "page", 1, 1, 10_000)
	if err != nil {
		return 0, 0, err
	}
	pageSize, err := queryInt(r.URL.Query(), "pageSize", 20, 1, 100)
	return page, pageSize, err
}

func queryInt(values url.Values, key string, fallback, min, max int) (int, error) {
	raw := values.Get(key)
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < min || parsed > max {
		return 0, fmt.Errorf("%s must be between %d and %d", key, min, max)
	}
	return parsed, nil
}

func validUUID(value string) bool {
	return domain.ValidUUID(value)
}

func textLength(value string) int {
	return domain.TextLength(value)
}

func validateEmail(value string) bool {
	return domain.ValidEmail(value)
}

func employeeDTO(user models.User) EmployeeDTO {
	return EmployeeDTO{
		ID: user.ID, Nama: user.Nama, Email: user.Email, NIP: user.Nip,
		NIDN: user.Nidn, NIDK: user.Nidk,
		KategoriPegawaiID: user.KategoriPegawaiId, HomebaseID: user.HomebaseId,
		JabatanStruktural: user.JabatanStruktural, JabatanFungsional: user.JabatanFungsional,
		Aktif:     user.Aktif,
		CreatedAt: timeutil.InLocation(user.CreatedAt),
		UpdatedAt: timeutil.InLocation(user.UpdatedAt),
	}
}

func organizationDTO(organization models.Homebase) OrganizationDTO {
	return OrganizationDTO{
		ID: organization.ID, Name: organization.Name, Description: organization.Description,
		Jenis: organization.Jenis, ParentID: organization.ParentId, Aktif: organization.Aktif,
		CreatedAt: timeutil.InLocation(organization.CreatedAt),
		UpdatedAt: timeutil.InLocation(organization.UpdatedAt),
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// writeAppError memetakan error lapisan aplikasi menjadi bentuk error HTTP
// yang seragam.
func writeAppError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrHasActiveMember):
		writeErrorResponse(w, http.StatusConflict, apierror.CodeConflict)
	case errors.Is(err, domain.ErrForbidden):
		writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
	case errors.Is(err, domain.ErrInvalidRef), errors.Is(err, domain.ErrInvalidParent), errors.Is(err, domain.ErrValidation):
		writeValidationError(w)
	default:
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
	}
}

// decodePatchFields mengubah body PATCH menjadi peta field yang ditandai
// hadir, kosong (null), atau berisi nilai teks.
//
// Field yang tidak dikirim tidak termasuk peta; field yang dikirim dengan
// nilai null ditandai Null sehingga dapat mengosongkan kolom yang boleh
// dikosongkan.
func decodePatchFields(w http.ResponseWriter, r *http.Request, allowed map[string]bool) (map[string]domain.Field, error) {
	var values map[string]json.RawMessage
	if err := decodeJSON(w, r, &values); err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, errors.New("at least one field is required")
	}

	fields := make(map[string]domain.Field, len(values))
	for key, raw := range values {
		if !allowed[key] {
			return nil, fmt.Errorf("field %s is not writable", key)
		}
		field := domain.Field{Present: true}
		if string(raw) == "null" {
			field.Null = true
			fields[key] = field
			continue
		}
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return nil, err
		}
		field.Value = text
		fields[key] = field
	}
	return fields, nil
}
