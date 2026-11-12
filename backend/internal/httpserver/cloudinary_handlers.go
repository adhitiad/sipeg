package httpserver

import (
	"net/http"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/cloudinary"
)

// handleCloudinaryUploadSignature mengembalikan tanda tangan unggah Cloudinary
// beserta parameter yang dibutuhkan klien untuk mengunggah berkas langsung ke
// Cloudinary. Endpoint ini memerlukan autentikasi siapa saja (pegawai,
// operator, atasan, admin) karena berkas yang diunggah milik pengguna itu
// sendiri.
//
// Respons JSON:
//
//	{"cloudName","apiKey","timestamp","signature","folder?","resourceType?"}
func handleCloudinaryUploadSignature(w http.ResponseWriter, r *http.Request, svc *cloudinary.Service) {
	if svc == nil || !svc.Enabled() {
		apierror.Write(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}

	sig, err := svc.SignUpload(nil)
	if err != nil {
		apierror.Write(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}

	writeJSON(w, http.StatusOK, sig)
}

// registerCloudinaryRoutes mendaftarkan endpoint tanda tangan unggah Cloudinary.
func registerCloudinaryRoutes(mux *http.ServeMux, resolver auth.SessionResolver, svc *cloudinary.Service) {
	if svc == nil {
		return
	}
	mux.Handle("GET "+APIPrefix+"/cloudinary/upload-signature",
		auth.Authenticate(resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleCloudinaryUploadSignature(w, r, svc)
		})))
}
