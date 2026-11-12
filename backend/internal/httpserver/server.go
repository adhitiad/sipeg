package httpserver

import (
	"context"
	"net/http"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/attendance"
	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/cloudinary"
	"simpeg/backend/internal/holiday"
	"simpeg/backend/internal/observability"
	"simpeg/backend/internal/payroll/app"
	"simpeg/backend/internal/qstash"
)

// MuxOptions mengumpulkan dependensi yang dibutuhkan untuk menyusun API mux.
type MuxOptions struct {
	Resolver       auth.SessionResolver
	Policy         auth.Policy
	Directory      DirectoryServices
	Attendance     attendance.Store
	AttendanceRead *attendance.ReadService
	// DeviceAuth mengaktifkan autentikasi perangkat pada endpoint ingestion
	// presensi (X-Device-ID + Bearer token), terpisah dari sesi pengguna.
	DeviceAuth attendance.DeviceAuthenticator
	// Payroll menjalankan use case penggajian. Bila nil, endpoint penggajian
	// tidak didaftarkan.
	Payroll *app.Service
	// Holiday menjalankan use case kalender hari libur. Bila nil, endpoint
	// kalender tidak didaftarkan.
	Holiday *holiday.Service
	// QStash mendaftarkan endpoint callback pekerjaan asinkron. Bila nil,
	// callback tidak didaftarkan.
	QStash *QStashOptions
	// SLO mendaftarkan endpoint callback pemeriksaan target SLO. Bila nil,
	// pemeriksaan terjadwal tidak didaftarkan.
	SLO *SLOOptions
	// SessionRevoker mengaktifkan POST /api/v1/auth/logout bila tidak nil.
	SessionRevoker auth.SessionRevoker
	// Cloudinary menyediakan tanda tangan unggah berkas dokumen. Bila nil,
	// endpoint tanda tangan unggah tidak didaftarkan.
	Cloudinary *cloudinary.Service
	// SessionIssuer mengaktifkan POST /api/v1/auth/session (penerbitan sesi
	// bootstrap) bila tidak nil. Endpoint ini akan digantikan oleh runtime
	// Better Auth saat tersedia.
	SessionIssuer auth.SessionIssuer
	// ReadinessCheckers diperiksa pada GET /ready.
	ReadinessCheckers []DependencyChecker
	// ReadinessTimeout membatasi waktu pemeriksaan tiap dependency.
	ReadinessTimeout time.Duration
}

// QStashOptions memuat dependensi endpoint callback QStash.
type QStashOptions struct {
	// Verifier memeriksa signature callback terhadap raw body.
	Verifier *qstash.Verifier
	// Cleaner menjalankan pekerjaan pembersihan jejak audit.
	Cleaner *audit.Cleaner
	// Recorder mencatat hasil pekerjaan ke jejak audit.
	Recorder AuditRecorder
	// PayrollVerifier memeriksa signature callback pemberitahuan slip gaji.
	// Bila diisi bersama PayrollReminder, endpoint pemberitahuan didaftarkan.
	PayrollVerifier *qstash.Verifier
	// PayrollReminder menjalankan pekerjaan pemberitahuan slip gaji H+1.
	PayrollReminder func(ctx context.Context) (int, error)
	// Logger menerima catatan keamanan untuk callback yang ditolak. Nil
	// berarti penolakan signature tidak dicatat ke log.
	Logger *observability.Logger
}

type ProfileResponse struct {
	UserID     string `json:"userId"`
	Email      string `json:"email"`
	RoleName   string `json:"roleName,omitempty"`
	HomebaseID string `json:"homebaseId,omitempty"`
}

// NewMux menyusun mux minimal tanpa modul direktori maupun presensi.
func NewMux(resolver auth.SessionResolver) *http.ServeMux {
	return NewMuxWithOptions(MuxOptions{Resolver: resolver})
}

// NewMuxWithServices menyusun mux yang memuat modul direktori dan presensi.
func NewMuxWithServices(resolver auth.SessionResolver, policy auth.Policy, directory DirectoryServices, attendanceStore attendance.Store) *http.ServeMux {
	return NewMuxWithOptions(MuxOptions{
		Resolver:   resolver,
		Policy:     policy,
		Directory:  directory,
		Attendance: attendanceStore,
	})
}

// NewMuxWithOptions menyusun seluruh route API beserta modul dan dependensinya.
func NewMuxWithOptions(options MuxOptions) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /health", http.HandlerFunc(HandleHealth))
	mux.Handle("GET /ready", NewReadinessHandler(options.ReadinessCheckers, options.ReadinessTimeout))
	mux.Handle("GET "+APIPrefix+"/me", auth.Authenticate(options.Resolver)(http.HandlerFunc(HandleMe)))
	if options.Directory.Employees != nil {
		RegisterDirectoryRoutes(mux, options.Resolver, options.Policy, options.Directory)
	}
	if options.Attendance != nil {
		RegisterAttendanceRoutes(mux, options.Resolver, options.Policy, options.Attendance, options.DeviceAuth)
	}
	if options.AttendanceRead != nil {
		RegisterAttendanceReadRoutes(mux, options.Resolver, options.Policy, options.AttendanceRead)
	}
	if options.Payroll != nil {
		RegisterPayrollRoutes(mux, options.Resolver, options.Policy, options.Payroll)
	}
	if options.Holiday != nil {
		RegisterHolidayRoutes(mux, options.Resolver, options.Policy, options.Holiday)
	}
	if options.QStash != nil {
		RegisterQStashRoutes(mux, options.QStash)
	}
	if options.SLO != nil {
		RegisterSLORoutes(mux, options.SLO)
	}
	RegisterLogoutRoute(mux, options.Resolver, options.SessionRevoker)
	if options.SessionIssuer != nil {
		RegisterSessionIssueRoute(mux, options.Resolver, options.Policy, options.SessionIssuer)
	}
	registerCloudinaryRoutes(mux, options.Resolver, options.Cloudinary)
	return mux
}

// HandleHealth adalah liveness: proses hidup. Tidak gagal hanya karena dependency
// eksternal sedang tidak tersedia.
func HandleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleMe mengembalikan profil principal dari sesi terverifikasi.
func HandleMe(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		apierror.Write(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, ProfileResponse{
		UserID:     principal.UserID,
		Email:      principal.Email,
		RoleName:   principal.RoleName,
		HomebaseID: principal.HomebaseID,
	})
}
