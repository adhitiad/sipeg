package httpserver

import (
	"errors"
	"net/http"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
)

// RegisterLogoutRoute mendaftarkan endpoint pencabutan sesi.
// Route hanya didaftarkan bila revoker tersedia.
func RegisterLogoutRoute(mux *http.ServeMux, resolver auth.SessionResolver, revoker auth.SessionRevoker) {
	if revoker == nil {
		return
	}
	mux.Handle("POST "+APIPrefix+"/auth/logout", auth.Authenticate(resolver)(http.HandlerFunc(logoutHandler(revoker))))
}

func logoutHandler(revoker auth.SessionRevoker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := auth.SessionTokenFromContext(r.Context())
		if !ok || token == "" {
			writeErrorResponse(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
			return
		}
		principal, _ := auth.PrincipalFromContext(r.Context())
		// Sesi yang sudah tidak ada dianggap berhasil: logout harus idempoten.
		if err := revoker.RevokeSession(r.Context(), token, principal); err != nil && !errors.Is(err, auth.ErrSessionNotFound) {
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
