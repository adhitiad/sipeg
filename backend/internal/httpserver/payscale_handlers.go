package httpserver

import (
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/payscale/app"
	"simpeg/backend/internal/payscale/domain"
	"simpeg/backend/internal/timeutil"
)

// SkalaGajiDTO adalah bentuk publik master skala gaji pokok yayasan.
type SkalaGajiDTO struct {
	ID             string     `json:"id"`
	Kode           string     `json:"kode"`
	Nama           string     `json:"nama"`
	BerlakuMulai   time.Time  `json:"berlakuMulai"`
	BerlakuSelesai *time.Time `json:"berlakuSelesai,omitempty"`
	Aktif          bool       `json:"aktif"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// SkalaGajiDetailDTO adalah bentuk publik baris nominal (tingkat × kategori).
type SkalaGajiDetailDTO struct {
	ID               string  `json:"id"`
	SkalaGajiId      string  `json:"skalaGajiId"`
	KategoriStatusId string  `json:"kategoriStatusId"`
	KategoriKode     string  `json:"kategoriKode"`
	KategoriNama     string  `json:"kategoriNama"`
	Tingkat          int     `json:"tingkat"`
	Nominal          float64 `json:"nominal"`
}

// PetaKategoriStatusDTO adalah bentuk publik taxonomy kategori status pegawai.
type PetaKategoriStatusDTO struct {
	ID               string  `json:"id"`
	Kode             string  `json:"kode"`
	Nama             string  `json:"nama"`
	Deskripsi        *string `json:"deskripsi,omitempty"`
	GajiDariNegara   bool    `json:"gajiDariNegara"`
	ButuhKontrak     bool    `json:"butuhKontrak"`
	WajibBpjs        bool    `json:"wajibBpjs"`
	BupSesuaiYayasan bool    `json:"bupSesuaiYayasan"`
	Aktif            bool    `json:"aktif"`
}

type kategoriLookup struct {
	byID map[string]models.KategoriStatusPegawai
}

func newKategoriLookup(rows []models.KategoriStatusPegawai) kategoriLookup {
	lookup := kategoriLookup{byID: map[string]models.KategoriStatusPegawai{}}
	for _, row := range rows {
		lookup.byID[row.ID] = row
	}
	return lookup
}

func (l kategoriLookup) kategori(id string) (string, string) {
	kid, ok := l.byID[id]
	if !ok {
		return "", ""
	}
	return kid.Kode, kid.Nama
}

func skalaGajiDTO(skala models.SkalaGaji) SkalaGajiDTO {
	var berlakuSelesai *time.Time
	if skala.BerlakuSelesai != nil {
		converted := timeutil.InLocation(*skala.BerlakuSelesai)
		berlakuSelesai = &converted
	}
	return SkalaGajiDTO{
		ID:             skala.ID,
		Kode:           skala.Kode,
		Nama:           skala.Nama,
		BerlakuMulai:   timeutil.InLocation(skala.BerlakuMulai),
		BerlakuSelesai: berlakuSelesai,
		Aktif:          skala.Aktif,
		CreatedAt:      timeutil.InLocation(skala.CreatedAt),
		UpdatedAt:      timeutil.InLocation(skala.UpdatedAt),
	}
}

func skalaGajiDetailDTO(detail models.SkalaGajiDetail, lookup kategoriLookup) SkalaGajiDetailDTO {
	kategoriKode, kategoriNama := lookup.kategori(detail.KategoriStatusId)
	return SkalaGajiDetailDTO{
		ID:               detail.ID,
		SkalaGajiId:      detail.SkalaGajiId,
		KategoriStatusId: detail.KategoriStatusId,
		KategoriKode:     kategoriKode,
		KategoriNama:     kategoriNama,
		Tingkat:          detail.Tingkat,
		Nominal:          detail.Nominal,
	}
}

func writeSkalaError(w http.ResponseWriter, err error) {
	switch err {
	case domain.ErrNotFound:
		writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
	case domain.ErrConflict:
		writeErrorResponse(w, http.StatusConflict, apierror.CodeConflict)
	case domain.ErrForbidden:
		writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
	case domain.ErrValidation:
		writeValidationError(w)
	default:
		writeErrorResponse(w, http.StatusInternalServerError, apierror.CodeInternal)
	}
}

type payscaleAPI struct {
	service *app.Service
}

func (api payscaleAPI) listSkala(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r, domain.SkalaSortFields, "kode")
	if err != nil || textLength(strings.TrimSpace(r.URL.Query().Get("search"))) > listing.MaxSearchLength {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	options := listOptionsFromQuery(query, listing.Options{Search: strings.TrimSpace(r.URL.Query().Get("search"))})

	principal, _ := auth.PrincipalFromContext(r.Context())
	page, err := api.service.List(r.Context(), principal, options)
	if err != nil {
		writeSkalaError(w, err)
		return
	}
	items := make([]SkalaGajiDTO, 0, len(page.Items))
	for _, skala := range page.Items {
		items = append(items, skalaGajiDTO(skala))
	}
	writeJSON(w, http.StatusOK, PageDTO[SkalaGajiDTO]{
		Items: items, Page: page.Page, PageSize: page.PageSize,
		TotalItems: page.TotalItems, TotalPages: page.TotalPages,
	})
}

func (api payscaleAPI) getSkala(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("skalaID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	skala, details, err := api.service.GetWithDetails(r.Context(), principal, id)
	if err != nil {
		writeSkalaError(w, err)
		return
	}
	kategoriRows, err := api.service.ListKategoriStatus(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, apierror.CodeInternal)
		return
	}
	lookup := newKategoriLookup(kategoriRows)
	items := make([]SkalaGajiDetailDTO, 0, len(details))
	for _, detail := range details {
		items = append(items, skalaGajiDetailDTO(detail, lookup))
	}
	type response struct {
		SkalaGajiDTO
		Details []SkalaGajiDetailDTO `json:"details"`
	}
	writeJSON(w, http.StatusOK, response{SkalaGajiDTO: skalaGajiDTO(skala), Details: items})
}

func (api payscaleAPI) createSkala(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Kode           string  `json:"kode"`
		Nama           string  `json:"nama"`
		BerlakuMulai   string  `json:"berlakuMulai"`
		BerlakuSelesai *string `json:"berlakuSelesai,omitempty"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	start, err := time.Parse("2006-01-02", strings.TrimSpace(request.BerlakuMulai))
	if err != nil {
		writeValidationError(w)
		return
	}
	var end *time.Time
	if request.BerlakuSelesai != nil && strings.TrimSpace(*request.BerlakuSelesai) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*request.BerlakuSelesai))
		if err != nil {
			writeValidationError(w)
			return
		}
		end = &parsed
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	skala, err := api.service.Create(r.Context(), principal, domain.CreateSkalaGaji{
		Kode: strings.TrimSpace(request.Kode), Nama: strings.TrimSpace(request.Nama),
		BerlakuMulai: start, BerlakuSelesai: end,
	})
	if err != nil {
		writeSkalaError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, skalaGajiDTO(skala))
}

func (api payscaleAPI) updateSkala(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("skalaID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.SkalaGajiPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	skala, err := api.service.Update(r.Context(), principal, id, domain.NewSkalaGajiPatch(fields))
	if err != nil {
		writeSkalaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skalaGajiDTO(skala))
}

func (api payscaleAPI) listKategoriStatus(w http.ResponseWriter, r *http.Request) {
	rows, err := api.service.ListKategoriStatus(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, apierror.CodeInternal)
		return
	}
	dto := make([]PetaKategoriStatusDTO, 0, len(rows))
	for _, row := range rows {
		dto = append(dto, PetaKategoriStatusDTO{
			ID:               row.ID,
			Kode:             row.Kode,
			Nama:             row.Nama,
			Deskripsi:        row.Deskripsi,
			GajiDariNegara:   row.GajiDariNegara,
			ButuhKontrak:     row.ButuhKontrak,
			WajibBpjs:        row.WajibBpjs,
			BupSesuaiYayasan: row.BupSesuaiYayasan,
			Aktif:            row.Aktif,
		})
	}
	writeJSON(w, http.StatusOK, dto)
}

func (api payscaleAPI) listDetails(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("skalaID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	skala, details, err := api.service.GetWithDetails(r.Context(), principal, id)
	if err != nil {
		writeSkalaError(w, err)
		return
	}
	_ = skala
	kategoriRows, err := api.service.ListKategoriStatus(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, apierror.CodeInternal)
		return
	}
	lookup := newKategoriLookup(kategoriRows)
	items := make([]SkalaGajiDetailDTO, 0, len(details))
	for _, detail := range details {
		items = append(items, skalaGajiDetailDTO(detail, lookup))
	}
	writeJSON(w, http.StatusOK, items)
}

func (api payscaleAPI) createDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("skalaID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	var request struct {
		KategoriStatusId string  `json:"kategoriStatusId"`
		Tingkat          int     `json:"tingkat"`
		Nominal          float64 `json:"nominal"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	created, err := api.service.AddDetail(r.Context(), principal, domain.CreateSkalaGajiDetail{
		SkalaGajiId: id, KategoriStatusId: request.KategoriStatusId, Tingkat: request.Tingkat, Nominal: request.Nominal,
	})
	if err != nil {
		writeSkalaError(w, err)
		return
	}
	kategoriRows, err := api.service.ListKategoriStatus(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, apierror.CodeInternal)
		return
	}
	lookup := newKategoriLookup(kategoriRows)
	writeJSON(w, http.StatusCreated, skalaGajiDetailDTO(created, lookup))
}

func (api payscaleAPI) updateDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("detailID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	fields, err := decodePatchFields(w, r, domain.SkalaGajiDetailPatchFields)
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	detail, err := api.service.UpdateDetail(r.Context(), principal, id, domain.NewSkalaGajiDetailPatch(fields))
	if err != nil {
		writeSkalaError(w, err)
		return
	}
	kategoriRows, err := api.service.ListKategoriStatus(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, apierror.CodeInternal)
		return
	}
	lookup := newKategoriLookup(kategoriRows)
	writeJSON(w, http.StatusOK, skalaGajiDetailDTO(detail, lookup))
}

func (api payscaleAPI) deleteDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("detailID")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.service.DeleteDetail(r.Context(), principal, id); err != nil {
		writeSkalaError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RegisterPayscaleRoutes mendaftarkan endpoint master skala gaji pokok yayasan.
func RegisterPayscaleRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, service *app.Service) {
	if service == nil {
		return
	}
	api := payscaleAPI{service: service}
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	protectedAny := func(permissions []auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, permissions, handler))
	}
	mux.Handle("GET "+APIPrefix+"/payroll/scale/kategori-status", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase"}, http.HandlerFunc(api.listKategoriStatus)))
	mux.Handle("GET "+APIPrefix+"/payroll/scale", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase"}, http.HandlerFunc(api.listSkala)))
	mux.Handle("POST "+APIPrefix+"/payroll/scale", protected("payroll:manage", http.HandlerFunc(api.createSkala)))
	mux.Handle("GET "+APIPrefix+"/payroll/scale/{skalaID}", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase"}, http.HandlerFunc(api.getSkala)))
	mux.Handle("PATCH "+APIPrefix+"/payroll/scale/{skalaID}", protected("payroll:manage", http.HandlerFunc(api.updateSkala)))
	mux.Handle("GET "+APIPrefix+"/payroll/scale/{skalaID}/detail", protectedAny([]auth.Permission{"payroll:read:any", "payroll:read:homebase"}, http.HandlerFunc(api.listDetails)))
	mux.Handle("POST "+APIPrefix+"/payroll/scale/{skalaID}/detail", protected("payroll:manage", http.HandlerFunc(api.createDetail)))
	mux.Handle("PATCH "+APIPrefix+"/payroll/scale/detail/{detailID}", protected("payroll:manage", http.HandlerFunc(api.updateDetail)))
	mux.Handle("DELETE "+APIPrefix+"/payroll/scale/detail/{detailID}", protected("payroll:manage", http.HandlerFunc(api.deleteDetail)))
}
