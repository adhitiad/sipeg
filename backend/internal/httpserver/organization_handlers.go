package httpserver

import (
	"net/http"
	"strings"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/listing"
)

func (api directoryAPI) listOrganizations(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, domain.OrganizationSortFields(), "name")
	jenis := domain.NormalizeOrganizationType(r.URL.Query().Get("jenis"))
	parentID := r.URL.Query().Get("parentId")
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if err != nil || (jenis != "" && !domain.ValidOrganizationType(jenis)) ||
		(parentID != "" && !validUUID(parentID)) || textLength(search) > listing.MaxSearchLength {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	options := listOptionsFromQuery(query, listing.Options{
		Search: search, Jenis: jenis, ParentID: parentID,
	})

	page, err := api.organizations.List(r.Context(), principal, options)
	if err != nil {
		writeAppError(w, err)
		return
	}

	items := make([]OrganizationDTO, 0, len(page.Items))
	for _, organization := range page.Items {
		items = append(items, organizationDTO(organization))
	}
	writeJSON(w, http.StatusOK, PageDTO[OrganizationDTO]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api directoryAPI) getOrganization(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("organizationID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	organization, err := api.organizations.Get(r.Context(), principal, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, organizationDTO(organization))
}

type organizationWriteRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Jenis       string  `json:"jenis"`
	ParentID    *string `json:"parentId"`
}

func (api directoryAPI) createOrganization(w http.ResponseWriter, r *http.Request) {
	var request organizationWriteRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}

	jenis := domain.NormalizeOrganizationType(request.Jenis)
	if jenis == "" {
		jenis = domain.DefaultOrganizationType
	}
	input := domain.CreateOrganization{
		Name:        strings.TrimSpace(request.Name),
		Description: request.Description,
		Jenis:       jenis,
		ParentID:    request.ParentID,
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	created, err := api.organizations.Create(r.Context(), principal, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, organizationDTO(created))
}

func (api directoryAPI) updateOrganization(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("organizationID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	fields, err := decodePatchFields(w, r, domain.OrganizationPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	updated, err := api.organizations.Update(r.Context(), principal, id, domain.NewOrganizationPatch(fields))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, organizationDTO(updated))
}

func (api directoryAPI) archiveOrganization(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("organizationID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.organizations.Archive(r.Context(), principal, id); err != nil {
		writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
