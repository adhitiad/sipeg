package auth

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	"gorm.io/gorm"
	"simpeg/backend/internal/audit"
)

// Denial merangkum satu penolakan akses karena role tidak memegang izin.
type Denial struct {
	// Actor adalah principal yang meminta akses.
	Actor Principal
	// Permissions adalah izin yang dibutuhkan untuk akses tersebut.
	Permissions []Permission
	// Method adalah method HTTP permintaan.
	Method string
	// Route adalah pola route yang diakses, tanpa nilai parameter.
	Route string
	// IP adalah alamat host pemohon.
	IP string
}

// DenialRecorder menerima catatan penolakan otorisasi.
type DenialRecorder interface {
	RecordDenial(ctx context.Context, denial Denial) error
}

type denialRecorderKey struct{}

// WithDenialRecorder menyimpan recorder penolakan pada context permintaan.
func WithDenialRecorder(ctx context.Context, recorder DenialRecorder) context.Context {
	return context.WithValue(ctx, denialRecorderKey{}, recorder)
}

// DenialRecorderFromContext mengembalikan recorder penolakan dari context.
func DenialRecorderFromContext(ctx context.Context) (DenialRecorder, bool) {
	recorder, ok := ctx.Value(denialRecorderKey{}).(DenialRecorder)
	return recorder, ok
}

// DenialRecorderMiddleware menyuntikkan recorder penolakan ke context setiap
// permintaan. Recorder nil berarti penolakan tidak dicatat.
func DenialRecorderMiddleware(recorder DenialRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if recorder == nil {
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithDenialRecorder(r.Context(), recorder)))
		})
	}
}

// GormDenialRecorder menulis penolakan otorisasi ke tabel auditLog.
type GormDenialRecorder struct {
	DB *gorm.DB
}

var errDenialDB = errors.New("denial recorder database error")

// RecordDenial menulis satu entri audit untuk penolakan akses.
func (r GormDenialRecorder) RecordDenial(ctx context.Context, denial Denial) error {
	if r.DB == nil {
		return errDenialDB
	}

	note := denialNote(denial)
	entry := audit.Entry{
		ActorID: denial.Actor.UserID,
		Action:  audit.ActionAuthorizationDenied,
		Note:    &note,
	}
	if denial.Actor.HomebaseID != "" {
		unitID := denial.Actor.HomebaseID
		entry.UnitID = &unitID
	}
	if denial.IP != "" {
		ip := denial.IP
		entry.IP = &ip
	}
	return audit.Write(ctx, r.DB, entry)
}

// denialNote menyusun keterangan penolakan. Isinya hanya izin dan pola
// route, sehingga isi sumber daya yang ditolak tidak ikut tersimpan.
func denialNote(denial Denial) string {
	permissions := make([]string, 0, len(denial.Permissions))
	for _, permission := range denial.Permissions {
		permissions = append(permissions, string(permission))
	}
	note := "izin ditolak: " + strings.Join(permissions, ", ")
	if denial.Route != "" {
		note += " pada " + denial.Route
	}
	return note
}

// recordDenial mencatat penolakan ke recorder pada context bila tersedia.
// Kegagalan pencatatan tidak mengubah hasil penolakan.
func recordDenial(r *http.Request, principal Principal, permissions ...Permission) {
	recorder, ok := DenialRecorderFromContext(r.Context())
	if !ok {
		return
	}
	_ = recorder.RecordDenial(r.Context(), Denial{
		Actor:       principal,
		Permissions: permissions,
		Method:      r.Method,
		Route:       routeOf(r),
		IP:          hostOf(r.RemoteAddr),
	})
}

// routeOf mengembalikan pola route yang cocok. Pola route memuat placeholder
// dan bukan nilai parameter, sehingga aman disimpan pada jejak audit.
func routeOf(r *http.Request) string {
	if r.Pattern != "" {
		return r.Pattern
	}
	return r.Method + " " + r.URL.Path
}

// hostOf mengambil host dari alamat remote tanpa port.
func hostOf(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
