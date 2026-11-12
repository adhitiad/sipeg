package httpserver

import (
	"net/http"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/app"
	payrollapp "simpeg/backend/internal/payroll/app"
	skpapp "simpeg/backend/internal/skp/app"
	tridharmaapp "simpeg/backend/internal/tridharma/app"
)

// DirectoryServices memuat use case yang dipakai oleh modul direktori.
type DirectoryServices struct {
	Employees     *app.EmployeeService
	Organizations *app.OrganizationService
	Categories    *app.CategoryService
	Audits        *app.AuditService
	SKP           *skpapp.SKPService
	Tridharma     *tridharmaapp.TridharmaService
	Keluarga      *app.KeluargaService
	Golongan      *app.GolonganService
	Arsip         *app.ArsipService
	Sertifikasi   *app.SertifikasiService
	Pelatihan     *app.PelatihanService
	Sister        *app.SisterService
	Penghargaan   *app.PenghargaanService
	Tunjangan     *payrollapp.TunjanganService
	Potongan      *payrollapp.PotonganService
	Leave         *app.LeaveService
}

// RegisterDirectoryRoutes mendaftarkan seluruh endpoint direktori.
func RegisterDirectoryRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, services DirectoryServices) {
	api := directoryAPI{
		employees:     services.Employees,
		organizations: services.Organizations,
		categories:    services.Categories,
		audits:        services.Audits,
		skp:           services.SKP,
		tridharma:     services.Tridharma,
		leave:         services.Leave,
	}
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	protectedAny := func(permissions []auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequireAnyPermission(policy, permissions, handler))
	}

	mux.Handle("GET "+APIPrefix+"/employees", protectedAny([]auth.Permission{"employee:read:any", "employee:read:homebase", "employee:read:self"}, http.HandlerFunc(api.listEmployees)))
	mux.Handle("POST "+APIPrefix+"/employees", protectedAny([]auth.Permission{"employee:create", "employee:create:homebase"}, http.HandlerFunc(api.createEmployee)))
	mux.Handle("GET "+APIPrefix+"/employees/import-template.csv", protectedAny([]auth.Permission{"employee:import", "employee:import:homebase"}, http.HandlerFunc(api.employeeImportTemplate)))
	mux.Handle("POST "+APIPrefix+"/employees/import", protectedAny([]auth.Permission{"employee:import", "employee:import:homebase"}, http.HandlerFunc(api.importEmployees)))
	mux.Handle("GET "+APIPrefix+"/employees/{employeeId}", protectedAny([]auth.Permission{"employee:read:any", "employee:read:homebase", "employee:read:self"}, http.HandlerFunc(api.getEmployee)))
	mux.Handle("PATCH "+APIPrefix+"/employees/{employeeId}", protectedAny([]auth.Permission{"employee:update:any", "employee:update:homebase"}, http.HandlerFunc(api.updateEmployee)))
	mux.Handle("PUT "+APIPrefix+"/employees/{employeeId}/role", protected("employee:role:manage", http.HandlerFunc(api.assignEmployeeRole)))
	mux.Handle("DELETE "+APIPrefix+"/employees/{employeeId}", protectedAny([]auth.Permission{"employee:archive", "employee:archive:homebase"}, http.HandlerFunc(api.archiveEmployee)))

	mux.Handle("GET "+APIPrefix+"/organizations", protectedAny([]auth.Permission{"organization:read", "organization:read:homebase"}, http.HandlerFunc(api.listOrganizations)))
	mux.Handle("POST "+APIPrefix+"/organizations", protectedAny([]auth.Permission{"organization:manage", "organization:manage:homebase"}, http.HandlerFunc(api.createOrganization)))
	mux.Handle("GET "+APIPrefix+"/organizations/{organizationID}", protectedAny([]auth.Permission{"organization:read", "organization:read:homebase"}, http.HandlerFunc(api.getOrganization)))
	mux.Handle("PATCH "+APIPrefix+"/organizations/{organizationID}", protectedAny([]auth.Permission{"organization:manage", "organization:manage:homebase"}, http.HandlerFunc(api.updateOrganization)))
	mux.Handle("DELETE "+APIPrefix+"/organizations/{organizationID}", protectedAny([]auth.Permission{"organization:manage", "organization:manage:homebase"}, http.HandlerFunc(api.archiveOrganization)))

	mux.Handle("GET "+APIPrefix+"/audit-logs", protectedAny([]auth.Permission{"audit:read", "audit:read:homebase"}, http.HandlerFunc(api.listAuditLogs)))
	mux.Handle("GET "+APIPrefix+"/audit-logs/export", protectedAny([]auth.Permission{"audit:read", "audit:read:homebase"}, http.HandlerFunc(api.exportAuditLogs)))
	mux.Handle("GET "+APIPrefix+"/categories", protected("category:read", http.HandlerFunc(api.listCategories)))

	// ═══════════════════════════ HR Extensions ═══════════════════════════

	if api.keluarga != nil {
		mux.Handle("GET "+APIPrefix+"/keluarga", protectedAny([]auth.Permission{"keluarga:read:any", "keluarga:read:homebase", "keluarga:read:self"}, http.HandlerFunc(api.listKeluarga)))
		mux.Handle("POST "+APIPrefix+"/keluarga", protectedAny([]auth.Permission{"keluarga:create", "keluarga:create:homebase"}, http.HandlerFunc(api.createKeluarga)))
		mux.Handle("GET "+APIPrefix+"/keluarga/{keluargaId}", protectedAny([]auth.Permission{"keluarga:read:any", "keluarga:read:homebase", "keluarga:read:self"}, http.HandlerFunc(api.getKeluarga)))
		mux.Handle("PATCH "+APIPrefix+"/keluarga/{keluargaId}", protectedAny([]auth.Permission{"keluarga:update:any", "keluarga:update:homebase"}, http.HandlerFunc(api.updateKeluarga)))
		mux.Handle("DELETE "+APIPrefix+"/keluarga/{keluargaId}", protectedAny([]auth.Permission{"keluarga:archive", "keluarga:archive:homebase"}, http.HandlerFunc(api.archiveKeluarga)))
	}
	if api.golongan != nil {
		mux.Handle("GET "+APIPrefix+"/golongan", protectedAny([]auth.Permission{"golongan:read:any", "golongan:read:homebase", "golongan:read:self"}, http.HandlerFunc(api.listGolongan)))
		mux.Handle("POST "+APIPrefix+"/golongan", protectedAny([]auth.Permission{"golongan:create", "golongan:create:homebase"}, http.HandlerFunc(api.createGolongan)))
		mux.Handle("GET "+APIPrefix+"/golongan/{golonganId}", protectedAny([]auth.Permission{"golongan:read:any", "golongan:read:homebase", "golongan:read:self"}, http.HandlerFunc(api.getGolongan)))
		mux.Handle("PATCH "+APIPrefix+"/golongan/{golonganId}", protectedAny([]auth.Permission{"golongan:update:any", "golongan:update:homebase"}, http.HandlerFunc(api.updateGolongan)))
		mux.Handle("DELETE "+APIPrefix+"/golongan/{golonganId}", protectedAny([]auth.Permission{"golongan:archive", "golongan:archive:homebase"}, http.HandlerFunc(api.archiveGolongan)))
	}
	if api.arsip != nil {
		mux.Handle("GET "+APIPrefix+"/arsip", protectedAny([]auth.Permission{"arsip:read:any", "arsip:read:homebase", "arsip:read:self"}, http.HandlerFunc(api.listArsip)))
		mux.Handle("POST "+APIPrefix+"/arsip", protectedAny([]auth.Permission{"arsip:create", "arsip:create:homebase"}, http.HandlerFunc(api.createArsip)))
		mux.Handle("GET "+APIPrefix+"/arsip/{arsipId}", protectedAny([]auth.Permission{"arsip:read:any", "arsip:read:homebase", "arsip:read:self"}, http.HandlerFunc(api.getArsip)))
		mux.Handle("PATCH "+APIPrefix+"/arsip/{arsipId}", protectedAny([]auth.Permission{"arsip:update:any", "arsip:update:homebase"}, http.HandlerFunc(api.updateArsip)))
		mux.Handle("DELETE "+APIPrefix+"/arsip/{arsipId}", protectedAny([]auth.Permission{"arsip:archive", "arsip:archive:homebase"}, http.HandlerFunc(api.archiveArsip)))
	}
	if api.sertifikasi != nil {
		mux.Handle("GET "+APIPrefix+"/sertifikasi", protectedAny([]auth.Permission{"sertifikasi:read:any", "sertifikasi:read:homebase", "sertifikasi:read:self"}, http.HandlerFunc(api.listSertifikasi)))
		mux.Handle("POST "+APIPrefix+"/sertifikasi", protectedAny([]auth.Permission{"sertifikasi:create", "sertifikasi:create:homebase"}, http.HandlerFunc(api.createSertifikasi)))
		mux.Handle("GET "+APIPrefix+"/sertifikasi/{sertifikasiId}", protectedAny([]auth.Permission{"sertifikasi:read:any", "sertifikasi:read:homebase", "sertifikasi:read:self"}, http.HandlerFunc(api.getSertifikasi)))
		mux.Handle("PATCH "+APIPrefix+"/sertifikasi/{sertifikasiId}", protectedAny([]auth.Permission{"sertifikasi:update:any", "sertifikasi:update:homebase"}, http.HandlerFunc(api.updateSertifikasi)))
		mux.Handle("DELETE "+APIPrefix+"/sertifikasi/{sertifikasiId}", protectedAny([]auth.Permission{"sertifikasi:archive", "sertifikasi:archive:homebase"}, http.HandlerFunc(api.archiveSertifikasi)))
	}
	if api.pelatihan != nil {
		mux.Handle("GET "+APIPrefix+"/pelatihan", protectedAny([]auth.Permission{"pelatihan:read:any", "pelatihan:read:homebase", "pelatihan:read:self"}, http.HandlerFunc(api.listPelatihan)))
		mux.Handle("POST "+APIPrefix+"/pelatihan", protectedAny([]auth.Permission{"pelatihan:create", "pelatihan:create:homebase"}, http.HandlerFunc(api.createPelatihan)))
		mux.Handle("GET "+APIPrefix+"/pelatihan/{pelatihanId}", protectedAny([]auth.Permission{"pelatihan:read:any", "pelatihan:read:homebase", "pelatihan:read:self"}, http.HandlerFunc(api.getPelatihan)))
		mux.Handle("PATCH "+APIPrefix+"/pelatihan/{pelatihanId}", protectedAny([]auth.Permission{"pelatihan:update:any", "pelatihan:update:homebase"}, http.HandlerFunc(api.updatePelatihan)))
		mux.Handle("DELETE "+APIPrefix+"/pelatihan/{pelatihanId}", protectedAny([]auth.Permission{"pelatihan:archive", "pelatihan:archive:homebase"}, http.HandlerFunc(api.archivePelatihan)))
	}
	if api.sister != nil {
		mux.Handle("GET "+APIPrefix+"/sister", protectedAny([]auth.Permission{"sister:read:any", "sister:read:homebase", "sister:read:self"}, http.HandlerFunc(api.listSister)))
		mux.Handle("POST "+APIPrefix+"/sister", protectedAny([]auth.Permission{"sister:create", "sister:create:homebase"}, http.HandlerFunc(api.createSister)))
		mux.Handle("GET "+APIPrefix+"/sister/{sisterId}", protectedAny([]auth.Permission{"sister:read:any", "sister:read:homebase", "sister:read:self"}, http.HandlerFunc(api.getSister)))
		mux.Handle("PATCH "+APIPrefix+"/sister/{sisterId}", protectedAny([]auth.Permission{"sister:update:any", "sister:update:homebase"}, http.HandlerFunc(api.updateSister)))
		mux.Handle("DELETE "+APIPrefix+"/sister/{sisterId}", protectedAny([]auth.Permission{"sister:archive", "sister:archive:homebase"}, http.HandlerFunc(api.archiveSister)))
		mux.Handle("POST "+APIPrefix+"/sister/{sisterId}/sync", protected("sister:manage", http.HandlerFunc(api.syncSister)))
	}
	if api.penghargaan != nil {
		mux.Handle("GET "+APIPrefix+"/penghargaan", protectedAny([]auth.Permission{"penghargaan:read:any", "penghargaan:read:homebase", "penghargaan:read:self"}, http.HandlerFunc(api.listPenghargaan)))
		mux.Handle("POST "+APIPrefix+"/penghargaan", protectedAny([]auth.Permission{"penghargaan:create", "penghargaan:create:homebase"}, http.HandlerFunc(api.createPenghargaan)))
		mux.Handle("GET "+APIPrefix+"/penghargaan/{penghargaanId}", protectedAny([]auth.Permission{"penghargaan:read:any", "penghargaan:read:homebase", "penghargaan:read:self"}, http.HandlerFunc(api.getPenghargaan)))
		mux.Handle("PATCH "+APIPrefix+"/penghargaan/{penghargaanId}", protectedAny([]auth.Permission{"penghargaan:update:any", "penghargaan:update:homebase"}, http.HandlerFunc(api.updatePenghargaan)))
		mux.Handle("DELETE "+APIPrefix+"/penghargaan/{penghargaanId}", protectedAny([]auth.Permission{"penghargaan:archive", "penghargaan:archive:homebase"}, http.HandlerFunc(api.archivePenghargaan)))
	}
	if api.tunjangan != nil {
		mux.Handle("GET "+APIPrefix+"/tunjangan", protectedAny([]auth.Permission{"tunjangan:read:any", "tunjangan:read:homebase", "tunjangan:read:self"}, http.HandlerFunc(api.listTunjangan)))
		mux.Handle("POST "+APIPrefix+"/tunjangan", protectedAny([]auth.Permission{"tunjangan:create", "tunjangan:create:homebase"}, http.HandlerFunc(api.createTunjangan)))
		mux.Handle("GET "+APIPrefix+"/tunjangan/{tunjanganId}", protectedAny([]auth.Permission{"tunjangan:read:any", "tunjangan:read:homebase", "tunjangan:read:self"}, http.HandlerFunc(api.getTunjangan)))
		mux.Handle("PATCH "+APIPrefix+"/tunjangan/{tunjanganId}", protectedAny([]auth.Permission{"tunjangan:update:any", "tunjangan:update:homebase"}, http.HandlerFunc(api.updateTunjangan)))
		mux.Handle("DELETE "+APIPrefix+"/tunjangan/{tunjanganId}", protectedAny([]auth.Permission{"tunjangan:archive", "tunjangan:archive:homebase"}, http.HandlerFunc(api.archiveTunjangan)))
	}
	if api.potongan != nil {
		mux.Handle("GET "+APIPrefix+"/potongan", protectedAny([]auth.Permission{"potongan:read:any", "potongan:read:homebase", "potongan:read:self"}, http.HandlerFunc(api.listPotongan)))
		mux.Handle("POST "+APIPrefix+"/potongan", protectedAny([]auth.Permission{"potongan:create", "potongan:create:homebase"}, http.HandlerFunc(api.createPotongan)))
		mux.Handle("GET "+APIPrefix+"/potongan/{potonganId}", protectedAny([]auth.Permission{"potongan:read:any", "potongan:read:homebase", "potongan:read:self"}, http.HandlerFunc(api.getPotongan)))
		mux.Handle("PATCH "+APIPrefix+"/potongan/{potonganId}", protectedAny([]auth.Permission{"potongan:update:any", "potongan:update:homebase"}, http.HandlerFunc(api.updatePotongan)))
		mux.Handle("DELETE "+APIPrefix+"/potongan/{potonganId}", protectedAny([]auth.Permission{"potongan:archive", "potongan:archive:homebase"}, http.HandlerFunc(api.archivePotongan)))
	}
	if api.leave != nil {
		mux.Handle("GET "+APIPrefix+"/cuti", protectedAny([]auth.Permission{"cuti:read:any", "cuti:read:homebase", "cuti:read:self"}, http.HandlerFunc(api.listCuti)))
		mux.Handle("POST "+APIPrefix+"/cuti", protectedAny([]auth.Permission{"cuti:create", "cuti:create:homebase"}, http.HandlerFunc(api.createCuti)))
		mux.Handle("GET "+APIPrefix+"/cuti/{cutiId}", protectedAny([]auth.Permission{"cuti:read:any", "cuti:read:homebase", "cuti:read:self"}, http.HandlerFunc(api.getCuti)))
		mux.Handle("PATCH "+APIPrefix+"/cuti/{cutiId}", protectedAny([]auth.Permission{"cuti:update:any", "cuti:update:homebase"}, http.HandlerFunc(api.updateCuti)))
		mux.Handle("POST "+APIPrefix+"/cuti/{cutiId}/approve", protectedAny([]auth.Permission{"cuti:approve:any", "cuti:approve:homebase"}, http.HandlerFunc(api.approveCuti)))
		mux.Handle("POST "+APIPrefix+"/cuti/{cutiId}/reject", protectedAny([]auth.Permission{"cuti:approve:any", "cuti:approve:homebase"}, http.HandlerFunc(api.rejectCuti)))
	}

	// ═══════════════════════════ SKP ═══════════════════════════
	if api.skp != nil {
		mux.Handle("GET "+APIPrefix+"/skp/periode", protectedAny([]auth.Permission{"skp:periode:read:any", "skp:periode:read:homebase", "skp:periode:read:self"}, http.HandlerFunc(api.listSKPPeriode)))
		mux.Handle("POST "+APIPrefix+"/skp/periode", protectedAny([]auth.Permission{"skp:periode:manage:any", "skp:periode:manage:homebase"}, http.HandlerFunc(api.createSKPPeriode)))
		mux.Handle("GET "+APIPrefix+"/skp/periode/{periodeId}", protectedAny([]auth.Permission{"skp:periode:read:any", "skp:periode:read:homebase", "skp:periode:read:self"}, http.HandlerFunc(api.getSKPPeriode)))
		mux.Handle("PATCH "+APIPrefix+"/skp/periode/{periodeId}", protectedAny([]auth.Permission{"skp:periode:manage:any", "skp:periode:manage:homebase"}, http.HandlerFunc(api.updateSKPPeriode)))
		mux.Handle("DELETE "+APIPrefix+"/skp/periode/{periodeId}", protectedAny([]auth.Permission{"skp:periode:manage:any", "skp:periode:manage:homebase"}, http.HandlerFunc(api.deleteSKPPeriode)))

		mux.Handle("GET "+APIPrefix+"/skp/target", protectedAny([]auth.Permission{"skp:target:read:any", "skp:target:read:homebase", "skp:target:read:self"}, http.HandlerFunc(api.listSKPTarget)))
		mux.Handle("POST "+APIPrefix+"/skp/target", protectedAny([]auth.Permission{"skp:target:manage:any", "skp:target:manage:homebase"}, http.HandlerFunc(api.createSKPTarget)))
		mux.Handle("GET "+APIPrefix+"/skp/target/{targetId}", protectedAny([]auth.Permission{"skp:target:read:any", "skp:target:read:homebase", "skp:target:read:self"}, http.HandlerFunc(api.getSKPTarget)))
		mux.Handle("PATCH "+APIPrefix+"/skp/target/{targetId}", protectedAny([]auth.Permission{"skp:target:manage:any", "skp:target:manage:homebase"}, http.HandlerFunc(api.updateSKPTarget)))
		mux.Handle("DELETE "+APIPrefix+"/skp/target/{targetId}", protectedAny([]auth.Permission{"skp:target:manage:any", "skp:target:manage:homebase"}, http.HandlerFunc(api.deleteSKPTarget)))

		mux.Handle("GET "+APIPrefix+"/skp/log", protectedAny([]auth.Permission{"skp:log:read:any", "skp:log:read:homebase", "skp:log:read:self"}, http.HandlerFunc(api.listLogAktivitas)))
		mux.Handle("POST "+APIPrefix+"/skp/log", protectedAny([]auth.Permission{"skp:log:manage:any", "skp:log:manage:homebase"}, http.HandlerFunc(api.createLogAktivitas)))
		mux.Handle("GET "+APIPrefix+"/skp/log/{logId}", protectedAny([]auth.Permission{"skp:log:read:any", "skp:log:read:homebase", "skp:log:read:self"}, http.HandlerFunc(api.getLogAktivitas)))
		mux.Handle("PATCH "+APIPrefix+"/skp/log/{logId}", protectedAny([]auth.Permission{"skp:log:manage:any", "skp:log:manage:homebase"}, http.HandlerFunc(api.updateLogAktivitas)))
		mux.Handle("DELETE "+APIPrefix+"/skp/log/{logId}", protectedAny([]auth.Permission{"skp:log:manage:any", "skp:log:manage:homebase"}, http.HandlerFunc(api.deleteLogAktivitas)))

		mux.Handle("GET "+APIPrefix+"/skp/penilaian", protectedAny([]auth.Permission{"skp:penilaian:read:any", "skp:penilaian:read:homebase", "skp:penilaian:read:self"}, http.HandlerFunc(api.listPenilaian)))
		mux.Handle("POST "+APIPrefix+"/skp/penilaian", protectedAny([]auth.Permission{"skp:penilaian:manage:any", "skp:penilaian:manage:homebase"}, http.HandlerFunc(api.createPenilaian)))
		mux.Handle("GET "+APIPrefix+"/skp/penilaian/{penilaianId}", protectedAny([]auth.Permission{"skp:penilaian:read:any", "skp:penilaian:read:homebase", "skp:penilaian:read:self"}, http.HandlerFunc(api.getPenilaian)))
		mux.Handle("PATCH "+APIPrefix+"/skp/penilaian/{penilaianId}", protectedAny([]auth.Permission{"skp:penilaian:manage:any", "skp:penilaian:manage:homebase"}, http.HandlerFunc(api.updatePenilaian)))
		mux.Handle("DELETE "+APIPrefix+"/skp/penilaian/{penilaianId}", protectedAny([]auth.Permission{"skp:penilaian:manage:any", "skp:penilaian:manage:homebase"}, http.HandlerFunc(api.deletePenilaian)))
	}

	// ═══════════════════════════ Tridharma ═══════════════════════════
	if api.tridharma != nil {
		mux.Handle("GET "+APIPrefix+"/tridharma/periode", protectedAny([]auth.Permission{"tridharma:periode:read:any", "tridharma:periode:read:homebase", "tridharma:periode:read:self"}, http.HandlerFunc(api.listPeriodeAkademik)))
		mux.Handle("POST "+APIPrefix+"/tridharma/periode", protectedAny([]auth.Permission{"tridharma:periode:manage:any", "tridharma:periode:manage:homebase"}, http.HandlerFunc(api.createPeriodeAkademik)))
		mux.Handle("GET "+APIPrefix+"/tridharma/periode/{periodeId}", protectedAny([]auth.Permission{"tridharma:periode:read:any", "tridharma:periode:read:homebase", "tridharma:periode:read:self"}, http.HandlerFunc(api.getPeriodeAkademik)))
		mux.Handle("PATCH "+APIPrefix+"/tridharma/periode/{periodeId}", protectedAny([]auth.Permission{"tridharma:periode:manage:any", "tridharma:periode:manage:homebase"}, http.HandlerFunc(api.updatePeriodeAkademik)))
		mux.Handle("DELETE "+APIPrefix+"/tridharma/periode/{periodeId}", protectedAny([]auth.Permission{"tridharma:periode:manage:any", "tridharma:periode:manage:homebase"}, http.HandlerFunc(api.deletePeriodeAkademik)))

		mux.Handle("GET "+APIPrefix+"/tridharma/evaluasi", protectedAny([]auth.Permission{"tridharma:evaluasi:read:any", "tridharma:evaluasi:read:homebase", "tridharma:evaluasi:read:self"}, http.HandlerFunc(api.listEvaluasiBKD)))
		mux.Handle("POST "+APIPrefix+"/tridharma/evaluasi", protectedAny([]auth.Permission{"tridharma:evaluasi:manage:any", "tridharma:evaluasi:manage:homebase"}, http.HandlerFunc(api.createEvaluasiBKD)))
		mux.Handle("GET "+APIPrefix+"/tridharma/evaluasi/{evaluasiId}", protectedAny([]auth.Permission{"tridharma:evaluasi:read:any", "tridharma:evaluasi:read:homebase", "tridharma:evaluasi:read:self"}, http.HandlerFunc(api.getEvaluasiBKD)))
		mux.Handle("PATCH "+APIPrefix+"/tridharma/evaluasi/{evaluasiId}", protectedAny([]auth.Permission{"tridharma:evaluasi:manage:any", "tridharma:evaluasi:manage:homebase"}, http.HandlerFunc(api.updateEvaluasiBKD)))
		mux.Handle("DELETE "+APIPrefix+"/tridharma/evaluasi/{evaluasiId}", protectedAny([]auth.Permission{"tridharma:evaluasi:manage:any", "tridharma:evaluasi:manage:homebase"}, http.HandlerFunc(api.deleteEvaluasiBKD)))

		mux.Handle("GET "+APIPrefix+"/tridharma/pendidikan", protectedAny([]auth.Permission{"tridharma:pendidikan:read:any", "tridharma:pendidikan:read:homebase", "tridharma:pendidikan:read:self"}, http.HandlerFunc(api.listTridharmaPendidikan)))
		mux.Handle("POST "+APIPrefix+"/tridharma/pendidikan", protectedAny([]auth.Permission{"tridharma:pendidikan:manage:any", "tridharma:pendidikan:manage:homebase"}, http.HandlerFunc(api.createTridharmaPendidikan)))
		mux.Handle("GET "+APIPrefix+"/tridharma/pendidikan/{pendidikanId}", protectedAny([]auth.Permission{"tridharma:pendidikan:read:any", "tridharma:pendidikan:read:homebase", "tridharma:pendidikan:read:self"}, http.HandlerFunc(api.getTridharmaPendidikan)))
		mux.Handle("PATCH "+APIPrefix+"/tridharma/pendidikan/{pendidikanId}", protectedAny([]auth.Permission{"tridharma:pendidikan:manage:any", "tridharma:pendidikan:manage:homebase"}, http.HandlerFunc(api.updateTridharmaPendidikan)))
		mux.Handle("DELETE "+APIPrefix+"/tridharma/pendidikan/{pendidikanId}", protectedAny([]auth.Permission{"tridharma:pendidikan:manage:any", "tridharma:pendidikan:manage:homebase"}, http.HandlerFunc(api.deleteTridharmaPendidikan)))

		mux.Handle("GET "+APIPrefix+"/tridharma/penelitian", protectedAny([]auth.Permission{"tridharma:penelitian:read:any", "tridharma:penelitian:read:homebase", "tridharma:penelitian:read:self"}, http.HandlerFunc(api.listTridharmaPenelitian)))
		mux.Handle("POST "+APIPrefix+"/tridharma/penelitian", protectedAny([]auth.Permission{"tridharma:penelitian:manage:any", "tridharma:penelitian:manage:homebase"}, http.HandlerFunc(api.createTridharmaPenelitian)))
		mux.Handle("GET "+APIPrefix+"/tridharma/penelitian/{penelitianId}", protectedAny([]auth.Permission{"tridharma:penelitian:read:any", "tridharma:penelitian:read:homebase", "tridharma:penelitian:read:self"}, http.HandlerFunc(api.getTridharmaPenelitian)))
		mux.Handle("PATCH "+APIPrefix+"/tridharma/penelitian/{penelitianId}", protectedAny([]auth.Permission{"tridharma:penelitian:manage:any", "tridharma:penelitian:manage:homebase"}, http.HandlerFunc(api.updateTridharmaPenelitian)))
		mux.Handle("DELETE "+APIPrefix+"/tridharma/penelitian/{penelitianId}", protectedAny([]auth.Permission{"tridharma:penelitian:manage:any", "tridharma:penelitian:manage:homebase"}, http.HandlerFunc(api.deleteTridharmaPenelitian)))

		mux.Handle("GET "+APIPrefix+"/tridharma/pengabdian", protectedAny([]auth.Permission{"tridharma:pengabdian:read:any", "tridharma:pengabdian:read:homebase", "tridharma:pengabdian:read:self"}, http.HandlerFunc(api.listTridharmaPengabdian)))
		mux.Handle("POST "+APIPrefix+"/tridharma/pengabdian", protectedAny([]auth.Permission{"tridharma:pengabdian:manage:any", "tridharma:pengabdian:manage:homebase"}, http.HandlerFunc(api.createTridharmaPengabdian)))
		mux.Handle("GET "+APIPrefix+"/tridharma/pengabdian/{pengabdianId}", protectedAny([]auth.Permission{"tridharma:pengabdian:read:any", "tridharma:pengabdian:read:homebase", "tridharma:pengabdian:read:self"}, http.HandlerFunc(api.getTridharmaPengabdian)))
		mux.Handle("PATCH "+APIPrefix+"/tridharma/pengabdian/{pengabdianId}", protectedAny([]auth.Permission{"tridharma:pengabdian:manage:any", "tridharma:pengabdian:manage:homebase"}, http.HandlerFunc(api.updateTridharmaPengabdian)))
		mux.Handle("DELETE "+APIPrefix+"/tridharma/pengabdian/{pengabdianId}", protectedAny([]auth.Permission{"tridharma:pengabdian:manage:any", "tridharma:pengabdian:manage:homebase"}, http.HandlerFunc(api.deleteTridharmaPengabdian)))

		mux.Handle("GET "+APIPrefix+"/tridharma/penunjang", protectedAny([]auth.Permission{"tridharma:penunjang:read:any", "tridharma:penunjang:read:homebase", "tridharma:penunjang:read:self"}, http.HandlerFunc(api.listTridharmaPenunjang)))
		mux.Handle("POST "+APIPrefix+"/tridharma/penunjang", protectedAny([]auth.Permission{"tridharma:penunjang:manage:any", "tridharma:penunjang:manage:homebase"}, http.HandlerFunc(api.createTridharmaPenunjang)))
		mux.Handle("GET "+APIPrefix+"/tridharma/penunjang/{penunjangId}", protectedAny([]auth.Permission{"tridharma:penunjang:read:any", "tridharma:penunjang:read:homebase", "tridharma:penunjang:read:self"}, http.HandlerFunc(api.getTridharmaPenunjang)))
		mux.Handle("PATCH "+APIPrefix+"/tridharma/penunjang/{penunjangId}", protectedAny([]auth.Permission{"tridharma:penunjang:manage:any", "tridharma:penunjang:manage:homebase"}, http.HandlerFunc(api.updateTridharmaPenunjang)))
		mux.Handle("DELETE "+APIPrefix+"/tridharma/penunjang/{penunjangId}", protectedAny([]auth.Permission{"tridharma:penunjang:manage:any", "tridharma:penunjang:manage:homebase"}, http.HandlerFunc(api.deleteTridharmaPenunjang)))
	}
}
