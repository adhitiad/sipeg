package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

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

// sessionIssueRequest adalah body permintaan penerbitan sesi bootstrap.
type sessionIssueRequest struct {
	UserID string `json:"userId"`
}

// sessionIssueResponse adalah body respons penerbitan sesi bootstrap. Token
// dikembalikan agar dapat segera digunakan sebagai Authorization: Bearer <token>
// pada klien uji/bootstrap. Endpoint ini akan digantikan oleh flow login milik
// runtime Better Auth, jangan diperlakukan sebagai akhir untuk login pengguna.
type sessionIssueResponse struct {
	UserID    string    `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
	Token     string    `json:"token"`
}

// RegisterSessionIssueRoute mendaftarkan endpoint penerbitan sesi (bootstrap).
// Hanya didaftarkan bila resolver dan issuer tersedia. Panggilan memerlukan
// sesi yang valid dan permission auth:session:issue (saat ini hanya admin).
//
// Endpoint sengaja tidak mempercayai role dari request: role dan homebase yang
// disimpan pada sesi baru diambil dari data server-side (tabel user/role),
// sehingga tidak dapat digunakan untuk naikkan role.
func RegisterSessionIssueRoute(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, issuer auth.SessionIssuer) {
	if resolver == nil || issuer == nil {
		return
	}
	mux.Handle("POST "+APIPrefix+"/auth/session",
		auth.Authenticate(resolver)(
			auth.RequirePermission(policy, "auth:session:issue", http.HandlerFunc(sessionIssueHandler(issuer)))))
}

func sessionIssueHandler(issuer auth.SessionIssuer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req sessionIssueRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
			writeValidationError(w)
			return
		}

		token, expiresAt, err := issuer.IssueSession(r.Context(), req.UserID)
		if err != nil {
			if errors.Is(err, auth.ErrSessionNotFound) {
				writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
				return
			}
			writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
			return
		}

		writeJSON(w, http.StatusCreated, sessionIssueResponse{
			UserID:    req.UserID,
			ExpiresAt: expiresAt,
			Token:     token,
		})
	}
}
