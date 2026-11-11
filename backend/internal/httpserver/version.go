package httpserver

import (
	"net/http"
	"regexp"
	"strings"

	"simpeg/backend/internal/apierror"
)

// Version adalah versi API yang sedang berjalan. Versi naik hanya bila ada
// perubahan yang merusak kompatibilitas (perubahan additive tetap pada v1).
const Version = "v1"

// APIPrefix adalah prefix path untuk seluruh endpoint berversi.
const APIPrefix = "/api/" + Version

// VersionHeader adalah header respons yang memuat versi API yang dipakai.
const VersionHeader = "X-API-Version"

// versionPattern mengenali segmen path versi seperti "v1" atau "v2".
var versionPattern = regexp.MustCompile(`^v[0-9]+$`)

// VersionMiddleware menambahkan header versi pada setiap respons.
func VersionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(VersionHeader, Version)
		next.ServeHTTP(w, r)
	})
}

// VersionGuard menolak path API yang tidak menyertakan versi path sebelum
// router sempat memprosesnya. Path dengan versi lain yang tidak dikenal
// menghasilkan 404; path tanpa versi menghasilkan 400 beserta path yang benar.
//
// Guard ini berjalan di luar router agar router tetap dapat membedakan
// 404 dan 405 pada path yang sudah berversi.
func VersionGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/") || path == "/api" {
			segment := ""
			if rest := strings.TrimPrefix(path, "/api"); rest != "" {
				segment = strings.TrimPrefix(rest, "/")
				if index := strings.Index(segment, "/"); index >= 0 {
					segment = segment[:index]
				}
			}
			if segment != Version {
				if versionPattern.MatchString(segment) {
					apierror.Write(w, http.StatusNotFound, apierror.CodeNotFound)
					return
				}
				apierror.WriteWith(w, http.StatusBadRequest, apierror.Payload{
					Code:    apierror.CodeUnversionedPath,
					Message: "Endpoint API harus menyertakan versi path. Gunakan " + APIPrefix + ".",
				})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
