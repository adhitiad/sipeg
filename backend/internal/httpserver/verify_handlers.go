package httpserver

import (
	"context"
	"errors"
	"net/http"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/models"
	payrolldomain "simpeg/backend/internal/payroll/domain"
	"simpeg/backend/internal/timeutil"
)

// slipVerificationLookup membaca slip via token verifikasi publik.
// Diimplementasikan oleh payrollrepository.GormStore.
type slipVerificationLookup interface {
	FindSlipByVerificationToken(ctx context.Context, token string) (models.Gaji, error)
}

// PublicSlipVerificationDTO adalah jawaban halaman verifikasi publik slip:
// tidak pernah memuat nominal maupun data pribadi di luar nama.
type PublicSlipVerificationDTO struct {
	Nama         string `json:"nama"`
	Periode      string `json:"periode"`
	PeriodeLabel string `json:"periodeLabel"`
	Status       string `json:"status"`
	StatusLabel  string `json:"statusLabel"`
}

type slipVerificationAPI struct {
	lookup slipVerificationLookup
}

// RegisterVerificationRoutes mendaftarkan endpoint verifikasi slip publik.
// Endpoint ini sengaja tanpa sesi: halaman verifikasi adalah dokumen publik
// yang hanya membutuhkan token dari QR slip.
func RegisterVerificationRoutes(mux *http.ServeMux, lookup slipVerificationLookup) {
	if lookup == nil {
		return
	}
	api := slipVerificationAPI{lookup: lookup}
	mux.Handle("GET "+APIPrefix+"/verify/slip/{token}", http.HandlerFunc(api.getSlipVerification))
}

func (api slipVerificationAPI) getSlipVerification(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if !payrolldomain.ValidSlipVerificationToken(token) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	slip, err := api.lookup.FindSlipByVerificationToken(r.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, payrolldomain.ErrInvalidSlipToken):
			writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		case errors.Is(err, payrolldomain.ErrNotFound):
			writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
		default:
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		}
		return
	}
	nama := ""
	if slip.Pegawai != nil {
		nama = slip.Pegawai.Nama
	}
	period := timeutil.InLocation(slip.Periode)
	parsed, parseErr := payrolldomain.ParsePeriod(period.Format("2006-01"))
	periode := ""
	periodeLabel := ""
	if parseErr == nil {
		periode = parsed.Key()
		periodeLabel = parsed.Label()
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, PublicSlipVerificationDTO{
		Nama:         nama,
		Periode:      periode,
		PeriodeLabel: periodeLabel,
		Status:       string(slip.Status),
		StatusLabel:  payrolldomain.StatusLabel(slip.Status),
	})
}
