// Transport HTTP untuk baca Surat Keputusan (SK). Modul baca-saja: SK terbit
// dari alur kerja, penerbitan manual belum diputuskan (`DEC-091`).
package httpserver

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"

	"simpeg/backend/internal/sk/app"
	"simpeg/backend/internal/sk/domain"
)

// skAPI memegang use case SK.
type skAPI struct {
	service *app.Service
}

// RegisterSKRoutes mendaftarkan endpoint baca SK.
func RegisterSKRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, service *app.Service) {
	protectedAny := func(permissions []auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, permissions, handler))
	}
	api := skAPI{service: service}
	mux.Handle("GET "+APIPrefix+"/surat-keputusan", protectedAny([]auth.Permission{"sk:read:any", "sk:read:homebase"}, http.HandlerFunc(api.list)))
	mux.Handle("GET "+APIPrefix+"/surat-keputusan/{id}", protectedAny([]auth.Permission{"sk:read:any", "sk:read:homebase"}, http.HandlerFunc(api.get)))
}

// SKDTO adalah representasi satu surat keputusan.
type SKDTO struct {
	ID             string  `json:"id"`
	NomorSk        string  `json:"nomorSk"`
	Jenis          string  `json:"jenis"`
	Penerbit       string  `json:"penerbit"`
	UserID         string  `json:"userId"`
	Nama           string  `json:"nama,omitempty"`
	HomebaseID     *string `json:"homebaseId,omitempty"`
	TanggalSk      string  `json:"tanggalSk"`
	MulaiBerlaku   *string `json:"mulaiBerlaku,omitempty"`
	SelesaiBerlaku *string `json:"selesaiBerlaku,omitempty"`
	FileURL        *string `json:"fileUrl,omitempty"`
	Metadata       string  `json:"metadata,omitempty"`
	CreatedAt      string  `json:"createdAt"`
}

func skDTOFromRow(row domain.Row) SKDTO {
	var mulai *string
	if row.MulaiBerlaku != nil {
		s := row.MulaiBerlaku.Format("2006-01-02")
		mulai = &s
	}
	var selesai *string
	if row.SelesaiBerlaku != nil {
		s := row.SelesaiBerlaku.Format("2006-01-02")
		selesai = &s
	}
	dto := SKDTO{
		ID:             row.ID,
		NomorSk:        row.NomorSk,
		Jenis:          string(row.Jenis),
		Penerbit:       string(row.Penerbit),
		Nama:           row.Nama,
		HomebaseID:     row.HomebaseID,
		TanggalSk:      row.TanggalSk.Format("2006-01-02"),
		MulaiBerlaku:   mulai,
		SelesaiBerlaku: selesai,
		FileURL:        row.FileUrl,
		Metadata:       row.Metadata,
		CreatedAt:      row.CreatedAt.UTC().Format(time.RFC3339),
	}
	if row.UserId != nil {
		dto.UserID = *row.UserId
	}
	return dto
}

func (api skAPI) list(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	query := r.URL.Query()
	filter := domain.Filter{
		Options: listing.Options{
			Search:   query.Get("search"),
			Sort:     query.Get("sort"),
			Order:    query.Get("order"),
			UserID:   query.Get("userId"),
			Page:     1,
			PageSize: 25,
		},
		Jenis:    query.Get("jenis"),
		Penerbit: query.Get("penerbit"),
	}
	if raw := query.Get("page"); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil || page < 1 {
			writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
			return
		}
		filter.Page = page
	}
	if raw := query.Get("pageSize"); raw != "" {
		size, err := strconv.Atoi(raw)
		if err != nil || size < 1 || size > 100 {
			writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
			return
		}
		filter.PageSize = size
	}

	page, err := api.service.List(r.Context(), principal, filter)
	if err != nil {
		writeSKError(w, err)
		return
	}
	items := make([]SKDTO, 0, len(page.Items))
	for _, row := range page.Items {
		items = append(items, skDTOFromRow(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":      items,
		"page":       page.Page,
		"pageSize":   page.PageSize,
		"totalItems": page.TotalItems,
		"totalPages": page.TotalPages,
	})
}

func (api skAPI) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	row, err := api.service.Get(r.Context(), principal, id)
	if err != nil {
		writeSKError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skDTOFromRow(row))
}

func writeSKError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
	case errors.Is(err, domain.ErrValidation):
		writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
	case errors.Is(err, domain.ErrNotFound):
		writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
	default:
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
	}
}
