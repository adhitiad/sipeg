// Transport HTTP untuk master pensiun: konfigurasi per kategori status dan
// kepesertaan DPLK per pegawai (khusus admin — `pensiun:manage`).
package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"

	"simpeg/backend/internal/pensiun/app"
	"simpeg/backend/internal/pensiun/domain"
)

// pensiunAPI memegang use case pensiun.
type pensiunAPI struct {
	service *app.Service
}

// RegisterPensiunRoutes mendaftarkan endpoint master pensiun.
func RegisterPensiunRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, service *app.Service) {
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	protectedAny := func(permissions []auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, permissions, handler))
	}
	api := pensiunAPI{service: service}
	mux.Handle("GET "+APIPrefix+"/pensiun/konfigurasi", protectedAny([]auth.Permission{"pensiun:manage", "pensiun:read:homebase"}, http.HandlerFunc(api.listKonfigurasi)))
	mux.Handle("POST "+APIPrefix+"/pensiun/konfigurasi", protected("pensiun:manage", http.HandlerFunc(api.upsertKonfigurasi)))
	mux.Handle("PATCH "+APIPrefix+"/pensiun/konfigurasi/{id}", protected("pensiun:manage", http.HandlerFunc(api.updateKonfigurasi)))
	mux.Handle("DELETE "+APIPrefix+"/pensiun/konfigurasi/{id}", protected("pensiun:manage", http.HandlerFunc(api.deleteKonfigurasi)))
	mux.Handle("GET "+APIPrefix+"/pensiun/kepesertaan", protectedAny([]auth.Permission{"pensiun:manage", "pensiun:read:homebase"}, http.HandlerFunc(api.listKepesertaan)))
	mux.Handle("POST "+APIPrefix+"/pensiun/kepesertaan", protected("pensiun:manage", http.HandlerFunc(api.upsertKepesertaan)))
	mux.Handle("PATCH "+APIPrefix+"/pensiun/kepesertaan/{id}", protected("pensiun:manage", http.HandlerFunc(api.updateKepesertaan)))
	mux.Handle("DELETE "+APIPrefix+"/pensiun/kepesertaan/{id}", protected("pensiun:manage", http.HandlerFunc(api.deactivateKepesertaan)))
}

// PensiunKonfigurasiDTO adalah representasi satu baris konfigurasi pensiun.
type PensiunKonfigurasiDTO struct {
	ID                   string  `json:"id"`
	KategoriStatusID     string  `json:"kategoriStatusId"`
	BupUmur              int     `json:"bupUmur"`
	DplkWajib            bool    `json:"dplkWajib"`
	IuranPegawaiPct      float64 `json:"iuranPegawaiPct"`
	IuranPemberiKerjaPct float64 `json:"iuranPemberiKerjaPct"`
	PenyediaDplk         *string `json:"penyediaDplk,omitempty"`
	BerlakuMulai         string  `json:"berlakuMulai"`
	Aktif                bool    `json:"aktif"`
	CreatedAt            string  `json:"createdAt"`
}

// PensiunKepesertaanDTO adalah representasi satu kepesertaan DPLK.
type PensiunKepesertaanDTO struct {
	ID                  string  `json:"id"`
	UserID              string  `json:"userId"`
	Nama                string  `json:"nama,omitempty"`
	PenyediaDplk        string  `json:"penyediaDplk"`
	NomorKepesertaan    *string `json:"nomorKepesertaan,omitempty"`
	TanggalMulaiPensiun *string `json:"tanggalMulaiPensiun,omitempty"`
	Mulai               string  `json:"mulai"`
	Aktif               bool    `json:"aktif"`
	CreatedAt           string  `json:"createdAt"`
}

func pensiunKonfigurasiDTOFromModel(item models.PensiunKonfigurasi) PensiunKonfigurasiDTO {
	return PensiunKonfigurasiDTO{
		ID:                   item.ID,
		KategoriStatusID:     item.KategoriStatusId,
		BupUmur:              item.BupUmur,
		DplkWajib:            item.DplkWajib,
		IuranPegawaiPct:      item.IuranPegawaiPct,
		IuranPemberiKerjaPct: item.IuranPemberiKerjaPct,
		PenyediaDplk:         item.PenyediaDplk,
		BerlakuMulai:         item.BerlakuMulai.Format("2006-01-02"),
		Aktif:                item.Aktif,
		CreatedAt:            item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func pensiunKepesertaanDTOFromModel(item models.PensiunKepesertaan) PensiunKepesertaanDTO {
	var tanggal *string
	if item.TanggalMulaiPensiun != nil {
		s := item.TanggalMulaiPensiun.Format("2006-01-02")
		tanggal = &s
	}
	return PensiunKepesertaanDTO{
		ID:                  item.ID,
		UserID:              item.UserId,
		PenyediaDplk:        item.PenyediaDplk,
		NomorKepesertaan:    item.NomorKepesertaan,
		TanggalMulaiPensiun: tanggal,
		Mulai:               item.Mulai.Format("2006-01-02"),
		Aktif:               item.Aktif,
		CreatedAt:           item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (api pensiunAPI) listKonfigurasi(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	items, err := api.service.ListKonfigurasi(r.Context(), principal, r.URL.Query().Get("kategoriStatusId"))
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
			return
		}
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	out := make([]PensiunKonfigurasiDTO, 0, len(items))
	for _, item := range items {
		out = append(out, pensiunKonfigurasiDTOFromModel(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

type pensiunKonfigurasiRequest struct {
	KategoriStatusID     string  `json:"kategoriStatusId"`
	BupUmur              int     `json:"bupUmur"`
	DplkWajib            bool    `json:"dplkWajib"`
	IuranPegawaiPct      float64 `json:"iuranPegawaiPct"`
	IuranPemberiKerjaPct float64 `json:"iuranPemberiKerjaPct"`
	PenyediaDplk         string  `json:"penyediaDplk"`
	BerlakuMulai         string  `json:"berlakuMulai"`
	Aktif                *bool   `json:"aktif"`
}

func decodeKonfigurasiRequest(w http.ResponseWriter, r *http.Request) (domain.KonfigurasiInput, bool) {
	var request pensiunKonfigurasiRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return domain.KonfigurasiInput{}, false
	}
	berlaku, err := time.Parse("2006-01-02", strings.TrimSpace(request.BerlakuMulai))
	if err != nil {
		writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
		return domain.KonfigurasiInput{}, false
	}
	if strings.TrimSpace(request.KategoriStatusID) != "" && !validUUID(strings.TrimSpace(request.KategoriStatusID)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return domain.KonfigurasiInput{}, false
	}
	aktual := true
	if request.Aktif != nil {
		aktual = *request.Aktif
	}
	return domain.KonfigurasiInput{
		KategoriStatusID:     strings.TrimSpace(request.KategoriStatusID),
		BupUmur:              request.BupUmur,
		DplkWajib:            request.DplkWajib,
		IuranPegawaiPct:      request.IuranPegawaiPct,
		IuranPemberiKerjaPct: request.IuranPemberiKerjaPct,
		PenyediaDplk:         request.PenyediaDplk,
		BerlakuMulai:         berlaku,
		Aktif:                aktual,
	}, true
}

func (api pensiunAPI) upsertKonfigurasi(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeKonfigurasiRequest(w, r)
	if !ok {
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.service.UpsertKonfigurasi(r.Context(), principal, input)
	if err != nil {
		writePensiunError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, pensiunKonfigurasiDTOFromModel(item))
}

func (api pensiunAPI) updateKonfigurasi(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	input, ok := decodeKonfigurasiRequest(w, r)
	if !ok {
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.service.UpdateKonfigurasi(r.Context(), principal, id, input)
	if err != nil {
		writePensiunError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pensiunKonfigurasiDTOFromModel(item))
}

func (api pensiunAPI) deleteKonfigurasi(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.service.DeleteKonfigurasi(r.Context(), principal, id); err != nil {
		writePensiunError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api pensiunAPI) listKepesertaan(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	rows, err := api.service.ListKepesertaan(r.Context(), principal)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
			return
		}
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	out := make([]PensiunKepesertaanDTO, 0, len(rows))
	for _, row := range rows {
		dto := pensiunKepesertaanDTOFromModel(row.PensiunKepesertaan)
		dto.Nama = row.Nama
		out = append(out, dto)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

type pensiunKepesertaanRequest struct {
	UserID              string  `json:"userId"`
	PenyediaDplk        string  `json:"penyediaDplk"`
	NomorKepesertaan    string  `json:"nomorKepesertaan"`
	TanggalMulaiPensiun *string `json:"tanggalMulaiPensiun"`
	Mulai               string  `json:"mulai"`
	Aktif               *bool   `json:"aktif"`
}

func decodeKepesertaanRequest(w http.ResponseWriter, r *http.Request) (domain.KepesertaanInput, bool) {
	var request pensiunKepesertaanRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return domain.KepesertaanInput{}, false
	}
	mulai, err := time.Parse("2006-01-02", strings.TrimSpace(request.Mulai))
	if err != nil {
		writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
		return domain.KepesertaanInput{}, false
	}
	if strings.TrimSpace(request.UserID) != "" && !validUUID(strings.TrimSpace(request.UserID)) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return domain.KepesertaanInput{}, false
	}
	var tanggal *time.Time
	if request.TanggalMulaiPensiun != nil && strings.TrimSpace(*request.TanggalMulaiPensiun) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*request.TanggalMulaiPensiun))
		if err != nil {
			writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
			return domain.KepesertaanInput{}, false
		}
		tanggal = &parsed
	}
	aktual := true
	if request.Aktif != nil {
		aktual = *request.Aktif
	}
	return domain.KepesertaanInput{
		UserID:              strings.TrimSpace(request.UserID),
		PenyediaDplk:        request.PenyediaDplk,
		NomorKepesertaan:    request.NomorKepesertaan,
		TanggalMulaiPensiun: tanggal,
		Mulai:               mulai,
		Aktif:               aktual,
	}, true
}

func (api pensiunAPI) upsertKepesertaan(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeKepesertaanRequest(w, r)
	if !ok {
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.service.UpsertKepesertaan(r.Context(), principal, input)
	if err != nil {
		writePensiunError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, pensiunKepesertaanDTOFromModel(item))
}

func (api pensiunAPI) updateKepesertaan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	input, ok := decodeKepesertaanRequest(w, r)
	if !ok {
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	item, err := api.service.UpdateKepesertaan(r.Context(), principal, id, input)
	if err != nil {
		writePensiunError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pensiunKepesertaanDTOFromModel(item))
}

func (api pensiunAPI) deactivateKepesertaan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validUUID(id) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.service.DeactivateKepesertaan(r.Context(), principal, id); err != nil {
		writePensiunError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writePensiunError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
	case errors.Is(err, domain.ErrValidation):
		writeErrorResponse(w, http.StatusUnprocessableEntity, apierror.CodeInvalidQuery)
	case errors.Is(err, domain.ErrConfigNotFound), errors.Is(err, domain.ErrKepesertaanNotFound):
		writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
	case errors.Is(err, domain.ErrUserNotFound):
		writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
	default:
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
	}
}
