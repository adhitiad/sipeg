package httpserver

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/crypto"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/timeutil"
)

func (api directoryAPI) listEmployees(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, domain.EmployeeSortFields(), "nama")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	homebaseID := r.URL.Query().Get("homebaseId")
	if err != nil || textLength(search) > listing.MaxSearchLength || (homebaseID != "" && !validUUID(homebaseID)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{Search: search, HomebaseID: homebaseID})

	page, err := api.employees.List(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}

	items := make([]EmployeeDTO, 0, len(page.Items))
	for _, user := range page.Items {
		items = append(items, employeeDTO(user))
	}
	writeJSON(w, http.StatusOK, PageDTO[EmployeeDTO]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

// exportEmployees meng-stream pegawai aktif dalam cakupan principal sebagai
// berkas CSV. Kolom sort dan pagination diabaikan: urutan selalu nama naik
// lalu id naik dan tidak ada batas jumlah baris. Filter sama dengan endpoint
// daftar (search, homebaseId). Ekspor yang berhasil dicatat sebagai aksi
// employee.exported setelah stream selesai.
//
// Header dan status 200 baru dikirim saat baris pertama tiba (atau saat
// stream selesai tanpa baris). Kegagalan membaca batch pertama karena itu
// masih dapat dijawab sebagai 500, bukan berkas kosong yang tampak berhasil.
// Berkas tidak memuat NIK, NPWP, maupun data rekening bank.
func (api directoryAPI) exportEmployees(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	homebaseID := r.URL.Query().Get("homebaseId")
	if textLength(search) > listing.MaxSearchLength || (homebaseID != "" && !validUUID(homebaseID)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())

	var (
		writer  *csv.Writer
		started bool
	)
	start := func() error {
		filename := "employees-export-" + timeutil.InLocation(time.Now()).Format("20060102T150405") + ".csv"
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
		w.WriteHeader(http.StatusOK)
		// BOM UTF-8 agar Microsoft Excel membuka berkas dengan benar.
		if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
			return err
		}
		writer = csv.NewWriter(w)
		if err := writer.Write(employeeImportHeaders); err != nil {
			return err
		}
		writer.Flush()
		flushResponse(w)
		started = true
		return nil
	}

	count := 0
	err := api.employees.Stream(r.Context(), principal, listing.Options{Search: search, HomebaseID: homebaseID}, func(user models.User) error {
		if !started {
			if err := start(); err != nil {
				return err
			}
		}
		count++
		nip := strValue(crypto.PtrString(user.Nip))
		jabatanStruktural := strValue(user.JabatanStruktural)
		jabatanFungsional := strValue(user.JabatanFungsional)
		if err := writer.Write([]string{
			user.Nama,
			user.Email,
			nip,
			jabatanStruktural,
			jabatanFungsional,
		}); err != nil {
			return err
		}
		if count%100 == 0 {
			writer.Flush()
			flushResponse(w)
		}
		return nil
	})
	if err != nil {
		if !started {
			// Kegagalan izin adalah keputusan aplikasi; kegagalan baca adalah
			// kegagalan internal. Keduanya masih dapat dijawab sebagai JSON
			// selama belum ada baris yang dikirim.
			if errors.Is(err, domain.ErrForbidden) {
				writeAppError(w, err)
				return
			}
			writeErrorResponse(w, http.StatusInternalServerError, apierror.CodeInternal)
		}
		return
	}
	if !started {
		if err := start(); err != nil {
			return
		}
	}
	writer.Flush()
	flushResponse(w)

	// Catat ekspor setelah stream selesai; stream yang terputus di tengah
	// tidak dicatat sebagai ekspor (hanya diagnostics di log aplikasi).
	note := fmt.Sprintf("%d baris diekspor", count)
	if err := api.audits.Record(r.Context(), audit.Entry{
		ActorID: principal.UserID,
		Action:  audit.ActionEmployeeExported,
		UnitID:  optionalStringRef(principal.HomebaseID),
		Note:    &note,
	}); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, apierror.CodeInternal)
	}
}

func (api directoryAPI) summaryEmployees(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	summary, err := api.employees.Summary(r.Context(), principal)
	if err != nil {
		writeAppError(w, err)
		return
	}
	byUnit := make([]UnitCountDTO, 0, len(summary.ByUnit))
	for _, unit := range summary.ByUnit {
		byUnit = append(byUnit, UnitCountDTO{UnitID: unit.UnitID, Count: unit.Count})
	}
	writeJSON(w, http.StatusOK, EmployeeSummaryDTO{
		Total:    summary.Total,
		Active:   summary.Active,
		Inactive: summary.Inactive,
		ByUnit:   byUnit,
	})
}

func (api directoryAPI) getEmployee(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("employeeId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	user, err := api.employees.Get(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, employeeDTO(user))
}

type employeeCreateRequest struct {
	Nama              string  `json:"nama"`
	Email             string  `json:"email"`
	NIP               *string `json:"nip"`
	NIDN              *string `json:"nidn"`
	NIDK              *string `json:"nidk"`
	KategoriPegawaiID *string `json:"kategoriPegawaiId"`
	HomebaseID        *string `json:"homebaseId"`
	JabatanStruktural *string `json:"jabatanStruktural"`
	JabatanFungsional *string `json:"jabatanFungsional"`
}

func (api directoryAPI) createEmployee(w http.ResponseWriter, r *http.Request) {
	var request employeeCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	input := domain.CreateEmployee{
		Nama:              strings.TrimSpace(request.Nama),
		Email:             strings.ToLower(strings.TrimSpace(request.Email)),
		NIP:               request.NIP,
		NIDN:              request.NIDN,
		NIDK:              request.NIDK,
		KategoriPegawaiID: request.KategoriPegawaiID,
		HomebaseID:        request.HomebaseID,
		JabatanStruktural: request.JabatanStruktural,
		JabatanFungsional: request.JabatanFungsional,
	}

	if !requireOptionalUUID(w, request.KategoriPegawaiID, request.HomebaseID) {
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	created, err := api.employees.Create(r.Context(), principal, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, employeeDTO(created))
}

func (api directoryAPI) updateEmployee(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("employeeId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	fields, err := decodePatchFields(w, r, domain.EmployeePatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	for _, key := range []string{"kategoriPegawaiId", "homebaseId"} {
		if field, ok := fields[key]; ok {
			if field.Present && !field.Null && !domain.ValidUUID(strings.TrimSpace(field.Value)) {
				writeValidationError(w)
				return
			}
		}
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.employees.Update(r.Context(), principal, id, domain.NewEmployeePatch(fields))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, employeeDTO(updated))
}

type employeeRoleRequest struct {
	RoleName string `json:"roleName"`
}

type employeeRoleResponse struct {
	EmployeeID string `json:"employeeId"`
	RoleName   string `json:"roleName"`
}

func (api directoryAPI) assignEmployeeRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("employeeId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	var request employeeRoleRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.employees.AssignRole(r.Context(), principal, id, request.RoleName)
	if err != nil {
		writeAppError(w, err)
		return
	}
	roleName, err := domain.NormalizeRoleName(request.RoleName)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, employeeRoleResponse{EmployeeID: updated.ID, RoleName: roleName})
}

func optionalStringRef(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (api directoryAPI) archiveEmployee(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("employeeId")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.employees.Archive(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
