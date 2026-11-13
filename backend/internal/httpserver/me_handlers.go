package httpserver

import (
	"errors"
	"net/http"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	meapp "simpeg/backend/internal/me/app"
	"simpeg/backend/internal/me/domain"
)

type updateMeRequest struct {
	// Nama, bila hadir, harus lolos aturan domain (dipangkas, 1-200 karakter).
	Nama *string `json:"nama"`
	// NoHp, bila hadir, harus berupa nomor Indonesia yang valid; teks kosong
	// mengosongkan nomor tersimpan.
	NoHp *string `json:"noHp"`
}

// HandleUpdateMe mengubah profil principal yang sedang masuk. Principal hanya
// dapat mengubah profilnya sendiri; identifier record selalu berasal dari
// sesi terverifikasi, bukan dari body permintaan.
func HandleUpdateMe(me *meapp.Service, policy auth.Policy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := auth.PrincipalFromContext(r.Context())
		if !ok {
			apierror.Write(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
			return
		}

		var request updateMeRequest
		if err := decodeJSON(w, r, &request); err != nil {
			writeErrorWith(w, http.StatusBadRequest, apierror.Payload{Code: apierror.CodeInvalidJSON})
			return
		}

		_, err := me.Update(r.Context(), principal.UserID, domain.UpdateProfileInput{
			Nama: request.Nama,
			NoHp: request.NoHp,
		})
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrValidation):
				writeValidationError(w)
			case errors.Is(err, domain.ErrNotFound):
				writeErrorWith(w, http.StatusNotFound, apierror.Payload{Code: apierror.CodeNotFound})
			default:
				writeErrorWith(w, http.StatusInternalServerError, apierror.Payload{
					Code:    apierror.CodeInternal,
					Message: "Profil tidak dapat disimpan.",
				})
			}
			return
		}

		// Setelah mutasi, muat ulang respons dari record terbaru agar klien
		// menerima keadaan kanonik (mis. spasi yang dipangkas).
		respondProfile(w, r, principal, policy, me)
	}
}
