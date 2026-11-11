package httpserver

import (
	"net/http"
	"strings"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
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
		KategoriPegawaiID: request.KategoriPegawaiID,
		HomebaseID:        request.HomebaseID,
		JabatanStruktural: request.JabatanStruktural,
		JabatanFungsional: request.JabatanFungsional,
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

	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.employees.Update(r.Context(), principal, id, domain.NewEmployeePatch(fields))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, employeeDTO(updated))
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
