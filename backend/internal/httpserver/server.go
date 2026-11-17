package httpserver

import (
	"context"
	"net/http"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/attendance"
	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	bpjsapp "simpeg/backend/internal/bpjs/app"
	"simpeg/backend/internal/cloudinary"
	"simpeg/backend/internal/holiday"
	honorapp "simpeg/backend/internal/honorsks/app"
	kontrakapp "simpeg/backend/internal/kontrak/app"
	meapp "simpeg/backend/internal/me/app"
	"simpeg/backend/internal/observability"
	"simpeg/backend/internal/payroll/app"
	payscaleapp "simpeg/backend/internal/payscale/app"
	pemberhentianapp "simpeg/backend/internal/pemberhentian/app"
	pensiunapp "simpeg/backend/internal/pensiun/app"
	peringatanapp "simpeg/backend/internal/peringatan/app"
	pph21clapp "simpeg/backend/internal/pph21ter/app"
	"simpeg/backend/internal/qstash"
	skapp "simpeg/backend/internal/sk/app"
	thrapp "simpeg/backend/internal/thr/app"
)

// MuxOptions mengumpulkan dependensi yang dibutuhkan untuk menyusun API mux.
type MuxOptions struct {
	Resolver       auth.SessionResolver
	Policy         auth.Policy
	Directory      DirectoryServices
	Attendance     attendance.Store
	AttendanceRead *attendance.ReadService
	// AttendanceAdmin menjalankan use case presensi administratif (pemetaan
	// PIN, derivasi, manual, laporan alpa). Bila nil, endpoint tidak
	// didaftarkan.
	AttendanceAdmin *attendance.AdminService
	// Pemberhentian menjalankan use case pemberhentian pegawai.
	Pemberhentian *pemberhentianapp.Service
	// Peringatan menjalankan use case penerbitan SP1 (peringatan). Bila nil,
	// endpoint peringatan tidak didaftarkan.
	Peringatan *peringatanapp.Service
	// Externalize perpanjangan & kontrak — dijamin tidak nil bila nil
	// (endpoint tidak didaftarkan).
	Kontrakt *kontrakapp.Service
	Pesangon *kontrakapp.PesangonService
	// Pensiun menjalankan use case master pensiun (konfigurasi + kepesertaan).
	Pensiun *pensiunapp.Service
	// SK menjalankan use case baca Surat Keputusan (baca saja).
	SK *skapp.Service
	// THR menjalankan use case THR (konfigurasi + pembuatan slip THR).
	Thr *thrapp.Service // SlipVerification membuka halaman verifikasi slip publik lewat token
	// slip. Bila nil, endpoint verifikasi tidak didaftarkan.
	SlipVerification slipVerificationLookup
	// DeviceAuth mengaktifkan autentikasi perangkat pada endpoint ingestion
	// presensi (X-Device-ID + Bearer token), terpisah dari sesi pengguna.
	DeviceAuth attendance.DeviceAuthenticator
	// Payroll menjalankan use case penggajian. Bila nil, endpoint penggajian
	// tidak didaftarkan.
	Payroll *app.Service
	// PayrollGenerate menjalankan pembuatan slip otomatis. Bila nil, endpoint
	// generate tidak didaftarkan.
	PayrollGenerate *app.GenerateSlipService
	// PayrollSlipOrgName adalah nama instansi pada kop slip PDF. Kosong
	// berarti slip menampilkan label generik.
	PayrollSlipOrgName string
	// AppBaseURL adalah origin aplikasi untuk tautan verifikasi publik slip
	// (dan tautan pada pemberitahuan email). Kosong berarti slip PDF tidak
	// memuat QR verifikasi.
	AppBaseURL string
	// Payscale menjalankan use case master skala gaji pokok yayasan. Bila
	// nil, endpoint skala gaji tidak didaftarkan.
	Payscale *payscaleapp.Service
	// HonorSks menjalankan use case honor mengajar per SKS (dosen LB). Bila
	// nil, endpoint honor tidak didaftarkan.
	HonorSks *honorapp.Service
	// Bpjs menjalankan use case BPJS Kesehatan/Ketenagakerjaan. Bila nil,
	// endpoint BPJS tidak didaftarkan.
	Bpjs *bpjsapp.Service
	// Pph21Ter menjalankan use case PPh 21 skema TER (PMK 168/2023). Bila
	// nil, endpoint TER tidak didaftarkan.
	Pph21Ter *pph21clapp.Service
	// Holiday menjalankan use case kalender hari libur. Bila nil, endpoint
	// kalender tidak didaftarkan.
	Holiday *holiday.Service
	// QStash mendaftarkan endpoint callback pekerjaan asinkron. Bila nil,
	// callback tidak didaftarkan.
	QStash *QStashOptions
	// SLO mendaftarkan endpoint callback pemeriksaan target SLO. Bila nil,
	// pemeriksaan terjadwal tidak didaftarkan.
	SLO *SLOOptions
	// SLODashboard mengaktifkan GET /api/v1/admin/slo/metrics bila tidak nil.
	// Endpoint bersifat read-only dan dibatasi permission slo:read (admin).
	SLODashboard SLODashboardReader
	// OpsOutbox mengaktifkan GET /api/v1/admin/ops/status bila tidak nil.
	// Endpoint bersifat read-only dan dibatasi permission ops:read (admin).
	OpsOutbox OutboxStatsReader
	// RBAC mengaktifkan endpoint admin pembacaan master RBAC berbasis scoped
	// assignment bila tidak nil. Endpoint bersifat read-only dan dibatasi
	// permission rbac:read (admin).
	RBAC RBACReader
	// SessionRevoker mengaktifkan POST /api/v1/auth/logout bila tidak nil.
	SessionRevoker auth.SessionRevoker
	// Cloudinary menyediakan tanda tangan unggah berkas dokumen. Bila nil,
	// endpoint tanda tangan unggah tidak didaftarkan.
	Cloudinary *cloudinary.Service
	// SessionIssuer mengaktifkan POST /api/v1/auth/session (penerbitan sesi
	// bootstrap) bila tidak nil. Endpoint ini akan digantikan oleh runtime
	// Better Auth saat tersedia.
	SessionIssuer auth.SessionIssuer
	// Me mengaktifkan GET dan PATCH /api/v1/me untuk profil mandiri. Bila
	// nil, GET /me hanya mengembalikan identitas sesi dan PATCH tidak
	// didaftarkan.
	Me *meapp.Service
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
	// PayrollReminder menjalankan pekerjaan pemberitahuan slip gaji H-1.
	PayrollReminder func(ctx context.Context) (int, error)
	// Logger menerima catatan keamanan untuk callback yang ditolak. Nil
	// berarti penolakan signature tidak dicatat ke log.
	Logger *observability.Logger
}

type ProfileResponse struct {
	UserID      string   `json:"userId"`
	Email       string   `json:"email"`
	RoleName    string   `json:"roleName,omitempty"`
	HomebaseID  string   `json:"homebaseId,omitempty"`
	Permissions []string `json:"permissions"`
	Nama        string   `json:"nama,omitempty"`
	NoHp        string   `json:"noHp,omitempty"`
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
	mux.Handle("GET "+APIPrefix+"/me", auth.Authenticate(options.Resolver)(HandleMe(options.Policy, options.Me)))
	if options.Me != nil {
		mux.Handle("PATCH "+APIPrefix+"/me", auth.Authenticate(options.Resolver)(HandleUpdateMe(options.Me, options.Policy)))
	}
	if options.Directory.Employees != nil {
		RegisterDirectoryRoutes(mux, options.Resolver, options.Policy, options.Directory)
	}
	if options.Attendance != nil {
		RegisterAttendanceRoutes(mux, options.Resolver, options.Policy, options.Attendance, options.DeviceAuth)
	}
	if options.AttendanceRead != nil {
		RegisterAttendanceReadRoutes(mux, options.Resolver, options.Policy, options.AttendanceRead)
	}
	if options.AttendanceAdmin != nil {
		RegisterAttendanceAdminRoutes(mux, options.Resolver, options.Policy, options.AttendanceAdmin)
	}
	if options.Pemberhentian != nil {
		RegisterPemberhentianRoutes(mux, options.Resolver, options.Policy, options.Pemberhentian)
	}
	if options.Peringatan != nil {
		RegisterPeringatanRoutes(mux, options.Resolver, options.Policy, options.Peringatan)
	}
	if options.Kontrakt != nil || options.Pesangon != nil {
		RegisterKontrakRoutes(mux, options.Resolver, options.Policy, options.Kontrakt, options.Pesangon)
	}
	if options.Pensiun != nil {
		RegisterPensiunRoutes(mux, options.Resolver, options.Policy, options.Pensiun)
	}
	if options.SK != nil {
		RegisterSKRoutes(mux, options.Resolver, options.Policy, options.SK)
	}
	if options.Thr != nil {
		RegisterThrRoutes(mux, options.Resolver, options.Policy, options.Thr)
	}
	RegisterVerificationRoutes(mux, options.SlipVerification)
	if options.Payroll != nil {
		var auditRecorder AuditRecorder
		if options.Directory.Audits != nil {
			auditRecorder = options.Directory.Audits
		}
		RegisterPayrollRoutes(mux, options.Resolver, options.Policy, options.Payroll, options.PayrollGenerate, PayrollBranding{
			Organization:  options.PayrollSlipOrgName,
			VerifyBaseURL: options.AppBaseURL,
		}, auditRecorder)
	}
	if options.Payscale != nil {
		RegisterPayscaleRoutes(mux, options.Resolver, options.Policy, options.Payscale)
	}
	if options.HonorSks != nil {
		RegisterHonorSksRoutes(mux, options.Resolver, options.Policy, options.HonorSks)
	}
	if options.Bpjs != nil {
		RegisterBpjsRoutes(mux, options.Resolver, options.Policy, options.Bpjs)
	}
	if options.Pph21Ter != nil {
		RegisterPph21TerRoutes(mux, options.Resolver, options.Policy, options.Pph21Ter)
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
	if options.SLODashboard != nil {
		RegisterSLOAdminRoutes(mux, options.Resolver, options.Policy, options.SLODashboard)
	}
	if options.OpsOutbox != nil {
		RegisterOpsRoutes(mux, options.Resolver, options.Policy, options.OpsOutbox)
	}
	if options.RBAC != nil {
		RegisterRBACAdminRoutes(mux, options.Resolver, options.Policy, options.RBAC)
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
// Permission yang dikirim berasal dari policy server sehingga klien tidak
// perlu menduplikasi matriks akses. Bila layanan profil mandiri tersedia,
// respons ikut memuat nama dan nomor HP supaya klien bisa menampilkan form
// profil tanpa panggilan terpisah.
func HandleMe(policy auth.Policy, me *meapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := auth.PrincipalFromContext(r.Context())
		if !ok {
			apierror.Write(w, http.StatusUnauthorized, apierror.CodeUnauthorized)
			return
		}

		respondProfile(w, r, principal, policy, me)
	}
}

// respondProfile menyusun ProfileResponse dari principal, policy, dan
// (opsional) record profil terbaru.
func respondProfile(w http.ResponseWriter, r *http.Request, principal auth.Principal, policy auth.Policy, me *meapp.Service) {
	// Daftar permission efektif per principal: mulai dari grant statis
	// (PermissionLister), lalu disaring dengan evaluator RBAC (ScopedPolicy)
	// bila tersedia, sehingga pencabutan RBAC tercermin di /me (DEC-096).
	permissions := []string{}
	var candidates []auth.Permission
	if lister, ok := policy.(auth.PermissionLister); ok && lister != nil {
		candidates = lister.List(principal.RoleName)
	}
	for _, permission := range candidates {
		allowed := false
		if sp, ok := policy.(auth.ScopedPolicy); ok {
			allowed = sp.AllowsScoped(r.Context(), principal, permission)
		} else if policy != nil {
			allowed = policy.Allows(principal.RoleName, permission)
		}
		if allowed {
			permissions = append(permissions, string(permission))
		}
	}

	response := ProfileResponse{
		UserID:      principal.UserID,
		Email:       principal.Email,
		RoleName:    principal.RoleName,
		HomebaseID:  principal.HomebaseID,
		Permissions: permissions,
	}
	if me != nil {
		user, err := me.Get(r.Context(), principal.UserID)
		if err != nil {
			writeErrorWith(w, http.StatusInternalServerError, apierror.Payload{
				Code:    apierror.CodeInternal,
				Message: "Profil tidak dapat dimuat.",
			})
			return
		}
		response.Nama = user.Nama
		if user.NoHp != nil {
			response.NoHp = *user.NoHp
		}
	}

	writeJSON(w, http.StatusOK, response)
}
