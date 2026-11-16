package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"simpeg/backend/internal/attendance"
	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	bpjsapp "simpeg/backend/internal/bpjs/app"
	bpjsrepository "simpeg/backend/internal/bpjs/repository"
	"simpeg/backend/internal/cache"
	"simpeg/backend/internal/cloudinary"
	"simpeg/backend/internal/config"
	"simpeg/backend/internal/crypto"
	"simpeg/backend/internal/database"
	"simpeg/backend/internal/directory/app"
	"simpeg/backend/internal/directory/repository"
	"simpeg/backend/internal/holiday"
	honorapp "simpeg/backend/internal/honorsks/app"
	honorrepository "simpeg/backend/internal/honorsks/repository"
	"simpeg/backend/internal/httpserver"
	"simpeg/backend/internal/jobs"
	kontrakapp "simpeg/backend/internal/kontrak/app"
	kontrakrepository "simpeg/backend/internal/kontrak/repository"
	meapp "simpeg/backend/internal/me/app"
	merepository "simpeg/backend/internal/me/repository"
	"simpeg/backend/internal/observability"
	"simpeg/backend/internal/outbox"
	payrollapp "simpeg/backend/internal/payroll/app"
	payrollrepository "simpeg/backend/internal/payroll/repository"
	payscaleapp "simpeg/backend/internal/payscale/app"
	payscalerepository "simpeg/backend/internal/payscale/repository"
	pemberhentianapp "simpeg/backend/internal/pemberhentian/app"
	pemberhentianrepository "simpeg/backend/internal/pemberhentian/repository"
	pensiunapp "simpeg/backend/internal/pensiun/app"
	pensiunrepository "simpeg/backend/internal/pensiun/repository"
	pph21clapp "simpeg/backend/internal/pph21ter/app"
	pph21trepository "simpeg/backend/internal/pph21ter/repository"
	"simpeg/backend/internal/qstash"
	rbacpkg "simpeg/backend/internal/rbac"
	rbacapp "simpeg/backend/internal/rbac/app"
	rbacrepository "simpeg/backend/internal/rbac/repository"
	skapp "simpeg/backend/internal/sk/app"
	skrepository "simpeg/backend/internal/sk/repository"
	skpapp "simpeg/backend/internal/skp/app"
	skprepository "simpeg/backend/internal/skp/repository"
	"simpeg/backend/internal/slo"
	sloapp "simpeg/backend/internal/slo/app"
	"simpeg/backend/internal/slo/domain"
	slorepository "simpeg/backend/internal/slo/repository"
	thrapp "simpeg/backend/internal/thr/app"
	thrrepository "simpeg/backend/internal/thr/repository"
	"simpeg/backend/internal/timeutil"
	tridharmaapp "simpeg/backend/internal/tridharma/app"
	tridharmarepository "simpeg/backend/internal/tridharma/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if len(cfg.PIIEncryptionKey) > 0 {
		if err := crypto.Configure(cfg.PIIEncryptionKey); err != nil {
			log.Fatalf("crypto: %v", err)
		}
	} else {
		log.Printf("crypto: PII_ENCRYPTION_KEY tidak dikonfigurasi; kolom PII tidak dapat dibaca/ditulis (hanya sah di luar production)")
	}

	db, err := database.OpenPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection setup failed: %v", err)
	}
	if err := database.ConfigurePool(db, database.PoolOptions{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
	}); err != nil {
		log.Fatalf("database pool configuration failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("database handle setup failed: %v", err)
	}
	defer sqlDB.Close()

	pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	if err := sqlDB.PingContext(pingCtx); err != nil {
		cancelPing()
		log.Fatalf("database connection failed")
	}
	cancelPing()

	jwtVerifier := auth.NewJWTVerifier(cfg.BetterAuthSecret)
	resolver := auth.NewCombinedSessionResolver(jwtVerifier, db)
	sessionIssuer := auth.NewGormSessionIssuer(db, cfg.SessionLifetime)
	store := repository.GormStore{DB: db}
	rbacStore := rbacrepository.GormStore{DB: db}
	rbacService := rbacapp.NewService(rbacStore)
	attendanceStore := attendance.GormStore{DB: db}
	deviceAuth := attendance.NewGormDeviceAuthenticator(db)
	skpStore := skprepository.GormStore{DB: db}
	tridharmaStore := tridharmarepository.GormStore{DB: db}

	// Role/permission grant matrix (Opsi A, disetujui 2026-11-10).
	// Deny-by-default: role yang tidak tercantum atau izin yang tidak disebutkan
	// ditolak. Matriks didefinisikan di cmd/server/policy.go.
	staticPolicy := RolePermissionMatrix()

	// PolicyAdapter menggabungkan grant statis (fallback) dengan evaluator
	// RBAC berbasis scoped assignment (DEC-087). Handler lama memakai
	// Allows (grant statis); handler baru dapat memakai AllowsScoped untuk
	// evaluasi scoped. Migrasi bertahap: grant statis tetap sumber kebenaran
	// hingga data RBAC terisi.
	policy := rbacpkg.NewPolicyAdapter(staticPolicy, rbacService)

	// Payroll store harus didefinisikan sebelum directoryServices karena
	// digunakan untuk layanan tunjangan dan potongan.
	payrollStore := payrollrepository.GormStore{DB: db}
	payrollService := payrollapp.NewService(payrollStore, policy, store)

	// Layanan administratif presensi: pemetaan PIN mesin, penurunan Presensi
	// dari AttendanceEvent, pengisian manual, dan laporan alpa (DEC-077).
	attendanceAdmin := attendance.NewAdminService(attendance.NewGormAdminStore(db), policy, store)

	// Layanan BPJS Kesehatan/Ketenagakerjaan (DEC-067/DEC-070).
	bpjsStore := bpjsrepository.NewGormStore(db)
	bpjsService := bpjsapp.NewService(bpjsStore, policy, store)

	// Layanan PPh 21 skema TER (PMK 168/2023).
	pph21Store := pph21trepository.NewGormStore(db)
	pph21Service := pph21clapp.NewService(pph21Store, policy)

	// Layanan honor mengajar per SKS (dosen LB), DEC-067/DEC-070.
	honorStore := honorrepository.NewGormStore(db)
	honorService := honorapp.NewService(honorStore, policy, store)

	// Layanan skala gaji pokok yayasan (penggajian inti, DEC-067/DEC-068).
	payscaleStore := payscalerepository.GormStore{DB: db}
	payscaleService := payscaleapp.NewService(payscaleStore, policy)

	// Layanan pembuatan slip otomatis (penggajian inti, DEC-067/DEC-075).
	payrollGenerate := payrollapp.NewGenerateSlipService(payrollStore, policy)

	// Layanan pemberhentian pegawai (DEC-083).
	pemberhentianStore := pemberhentianrepository.GormStore{DB: db}
	pemberhentianService := pemberhentianapp.NewService(pemberhentianStore, policy, store)

	// Layanan THR (konfigurasi tahunan + generate slip THR).
	thrStore := thrrepository.GormStore{DB: db}
	thrService := thrapp.NewService(thrStore, policy)

	// Layanan kontrak & perpanjangan pasca-BUP serta master pesangonAturan.
	kontrakStore := kontrakrepository.GormStore{DB: db}
	kontrakService := kontrakapp.NewService(kontrakStore, policy, store)
	pesangonStore := kontrakrepository.PesangonGormStore{DB: db}
	pesangonService := kontrakapp.NewPesangonService(pesangonStore, policy)

	// Layanan pensiun (konfigurasi per kategori status + kepesertaan DPLK).
	pensiunStore := pensiunrepository.GormStore{DB: db}
	pensiunService := pensiunapp.NewService(pensiunStore, policy)

	// Layanan baca Surat Keputusan (SK terbit dari alur kerja; baca saja).
	skStore := skrepository.GormStore{DB: db}
	skService := skapp.NewService(skStore, policy, store)

	// Layanan profil mandiri: principal mengubah nama/nomor HP miliknya sendiri.
	meService := meapp.NewService(merepository.GormStore{DB: db})

	// Cache daftar kategori pegawai memakai Redis bila dikonfigurasi. Tanpa
	// Redis, pembacaan tetap langsung ke PostgreSQL.
	var categoryCache *cache.Cache
	if cfg.RedisEnabled() {
		categoryCache = cache.New(cache.NewRESTClient(cfg.RedisURL, cfg.RedisToken), cache.Config{
			Namespace: "simpeg",
			TTL:       app.CategoryCacheTTL,
		})
	}

	// Lapisan aplikasi menerima port repository; lapisan transport menerima
	// use case. Domain menyimpan aturan, repository menyimpan persistensi.
	// Layanan tanda tangan unggah Cloudinary. Hanya tersedia bila ketiga
	// variabel CLOUDINARY_* diisi; bila tidak, endpoint tanda tangan unggah
	// tidak didaftarkan dan mengembalikan 503.
	cloudinaryService := cloudinary.New(cloudinary.Config{
		CloudName: cfg.Cloudinary.CloudName,
		APIKey:    cfg.Cloudinary.APIKey,
		APISecret: cfg.Cloudinary.APISecret,
	})

	directoryServices := httpserver.DirectoryServices{
		Employees:     app.NewEmployeeService(store, policy, store),
		Organizations: app.NewOrganizationService(store, policy, store),
		Categories:    app.NewCategoryService(store, categoryCache),
		Audits:        app.NewAuditService(store, policy, store),
		SKP:           skpapp.NewSKPService(skpStore, policy, store),
		Tridharma:     tridharmaapp.NewTridharmaService(tridharmaStore, policy, store),
		Keluarga:      app.NewKeluargaService(store, policy, store),
		Golongan:      app.NewGolonganService(store, policy, store),
		Jabatan:       app.NewJabatanService(store, policy),
		Arsip:         app.NewArsipService(store, policy, store),
		Sertifikasi:   app.NewSertifikasiService(store, policy, store),
		Pelatihan:     app.NewPelatihanService(store, policy, store),
		Sister:        app.NewSisterService(store, policy, store),
		Penghargaan:   app.NewPenghargaanService(store, policy, store),
		Tunjangan:     payrollapp.NewTunjanganService(payrollStore, policy, store),
		Potongan:      payrollapp.NewPotonganService(payrollStore, policy, store),
		Leave:         app.NewLeaveService(store, policy, store),
	}

	// Kalender hari libur nasional. Penyedia hanya dipakai untuk penyegaran
	// yang dipicu admin; pembacaan kalender selalu dari tabel HariLibur.
	holidayStore := &holiday.GormStore{DB: db}
	var holidayProvider holiday.Provider
	if cfg.HolidayProviderEnabled() {
		holidayProvider = &holiday.APICoIDClient{BaseURL: cfg.HolidayAPIURL, APIKey: cfg.HolidayAPIKey}
	}
	var holidayCache *cache.Cache
	if cfg.RedisEnabled() {
		holidayCache = cache.New(cache.NewRESTClient(cfg.RedisURL, cfg.RedisToken), cache.Config{
			Namespace: "simpeg",
			TTL:       24 * time.Hour,
		})
	}
	holidayService := holiday.NewService(holidayStore, holidayProvider, holidayCache)

	// Logger terstruktur dipakai bersama oleh log akses, peristiwa keamanan,
	// callback QStash, dan worker outbox.
	logger := observability.New(os.Stdout)

	// Pengoleki metrik SLO dan pemeriksaan target layanan. Pengoleki hanya
	// aktif bila SLO_ENABLED disetel; tanpa itu backend tidak menulis metrik
	// apa pun dan tidak menambah beban pada jalur permintaan. Dasbor admin
	// tetap terdaftar dan menampilkan tabel kosong bila pengoleki tidak
	// pernah menulis metrik.
	sloDashboard := sloapp.NewDashboardService(sloapp.DashboardOptions{
		Repository: slorepository.NewGormStore(db),
	})
	var sloCollector *slo.Collector
	var sloService *sloapp.Service
	if cfg.SLOEnabled() {
		sloStore := slorepository.NewGormStore(db)
		sloCollector = slo.NewCollector(slo.CollectorOptions{
			Store:      sloStore,
			WindowSize: cfg.SLOWindow(),
			Logger:     logger,
		})
		sloService = sloapp.NewService(sloapp.Options{
			Repository: sloStore,
			Outbox:     outbox.NewGormStore(db),
			Target: domain.Target{
				P95:         time.Duration(cfg.SLO.P95TargetMS) * time.Millisecond,
				ErrorRate:   cfg.SLO.ErrorRateTarget,
				MinRequests: cfg.SLO.MinRequests,
			},
			WindowSize: cfg.SLOWindow(),
			Recipients: cfg.SLO.AlertRecipients,
		})
	}

	// Callback QStash diautentikasi dengan signature QStash, bukan sesi
	// bearer. Subjek signature harus cocok dengan URL tujuan yang dipakai
	// penjadwal.
	auditCleaner := audit.NewCleaner(db, cfg.AuditRetentionDays)
	var qstashOptions *httpserver.QStashOptions
	var qstashPublisher *qstash.Publisher
	var sloOptions *httpserver.SLOOptions
	if cfg.QStashEnabled() {
		signingKeys := []string{cfg.QStashCurrentSigningKey, cfg.QStashNextSigningKey}
		callbackPath := httpserver.APIPrefix + "/internal/qstash/audit/cleanup"
		destination := jobs.Destination(cfg.AppBaseURL, callbackPath)
		payrollNotifyDestination := jobs.Destination(cfg.AppBaseURL, httpserver.APIPrefix+"/internal/qstash/payroll/notify")
		sloDestination := jobs.Destination(cfg.AppBaseURL, httpserver.APIPrefix+"/internal/qstash/slo/evaluate")
		verifier := qstash.NewVerifier(signingKeys, destination)
		payrollVerifier := qstash.NewVerifier(signingKeys, payrollNotifyDestination)
		qstashOptions = &httpserver.QStashOptions{
			Verifier:        verifier,
			Cleaner:         auditCleaner,
			Recorder:        directoryServices.Audits,
			PayrollVerifier: payrollVerifier,
			PayrollReminder: func(ctx context.Context) (int, error) {
				return payrollService.ProcessReminders(ctx, timeutil.NowUTC())
			},
			Logger: logger,
		}
		if sloService != nil {
			sloOptions = &httpserver.SLOOptions{
				Verifier:  qstash.NewVerifier(signingKeys, sloDestination),
				Evaluator: sloService,
				Recorder:  directoryServices.Audits,
				Logger:    logger,
			}
		}
		qstashPublisher = &qstash.Publisher{
			BaseURL:      cfg.QStashURL,
			SchedulesURL: cfg.QStashSchedulesURL,
			Token:        cfg.QStashToken,
		}
	}

	mux := httpserver.NewMuxWithOptions(httpserver.MuxOptions{
		Resolver:           resolver,
		Policy:             policy,
		Directory:          directoryServices,
		Attendance:         attendanceStore,
		AttendanceRead:     attendance.NewReadService(attendanceStore, store, policy),
		AttendanceAdmin:    attendanceAdmin,
		SlipVerification:   payrollStore,
		DeviceAuth:         deviceAuth,
		Payroll:            payrollService,
		PayrollGenerate:    payrollGenerate,
		PayrollSlipOrgName: cfg.PayrollSlipOrgName,
		AppBaseURL:         cfg.AppBaseURL,
		Pemberhentian:      pemberhentianService,
		Thr:                thrService,
		Kontrakt:           kontrakService,
		Pesangon:           pesangonService,
		Pensiun:            pensiunService,
		SK:                 skService,
		Payscale:           payscaleService,
		HonorSks:           honorService,
		Bpjs:               bpjsService,
		Pph21Ter:           pph21Service,
		Holiday:            holidayService,
		QStash:             qstashOptions,
		SLO:                sloOptions,
		SLODashboard:       sloDashboard,
		OpsOutbox:          outbox.NewGormStore(db),
		RBAC:               rbacrepository.GormStore{DB: db},
		SessionRevoker:     auth.GormSessionRevoker{DB: db},
		SessionIssuer:      sessionIssuer,
		Cloudinary:         cloudinaryService,
		Me:                 meService,
		ReadinessCheckers:  []httpserver.DependencyChecker{database.PingChecker{DB: db}},
	})

	rateLimiter := httpserver.NewRateLimiter(httpserver.RateLimitOptions{
		Limit:        cfg.RateLimitRequests,
		Window:       cfg.RateLimitWindow,
		StrictPaths:  []string{httpserver.APIPrefix + "/auth", httpserver.APIPrefix + "/me"},
		StrictLimit:  cfg.RateLimitAuthRequests,
		StrictWindow: cfg.RateLimitAuthWindow,
	})

	// Rantai handler dari dalam ke luar: router, penanganan error JSON,
	// pembatas laju, CORS, batas waktu statement, header keamanan, pencatatan
	// penolakan otorisasi, peristiwa keamanan, log akses, dan request ID.
	apiHandler := httpserver.JSONErrorMiddleware(httpserver.VersionMiddleware(httpserver.VersionGuard(mux)))
	corsHandler := httpserver.CORSMiddleware(httpserver.CORSOptions{
		TrustedOrigins: cfg.TrustedOrigins,
		AllowedMethods: httpserver.DefaultCORSMethods(),
		AllowedHeaders: httpserver.DefaultCORSHeaders(),
	})(rateLimiter.Middleware(apiHandler))
	timeoutHandler := httpserver.StatementTimeoutMiddleware(cfg.DBStatementTimeout)(corsHandler)
	securedHandler := httpserver.SecurityHeadersMiddleware(timeoutHandler)
	denialHandler := auth.DenialRecorderMiddleware(auth.GormDenialRecorder{DB: db})(securedHandler)
	securityHandler := observability.SecurityEventMiddleware(observability.SecurityEventOptions{
		Logger:    logger,
		AuthPaths: []string{httpserver.APIPrefix + "/auth"},
	})(denialHandler)
	accessHandler := observability.AccessLogMiddleware(observability.AccessLogOptions{
		Logger: logger,
	})(securityHandler)
	handler := observability.RequestIDMiddleware(accessHandler)
	if sloCollector != nil {
		// Pengoleksi metrik berada di luar rantai handler agar yang diukur
		// adalah keseluruhan waktu penanganan permintaan.
		handler = sloCollector.Middleware(handler)
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Worker latar: pengirim email outbox, konsumen pekerjaan RabbitMQ
	// (mode self-hosted), dan pengoleki metrik SLO. Semua berhenti saat
	// proses dimatikan.
	workerCtx, stopWorker := context.WithCancel(context.Background())
	defer stopWorker()
	startOutboxWorker(workerCtx, cfg, db, payrollService, logger)

	// Pengoleki metrik SLO menulis agregasi latensi secara berkala. Worker
	// hanya berjalan bila pemeriksaan SLO diaktifkan.
	if sloCollector != nil {
		go sloCollector.Run(workerCtx)
		logger.Info("slo_collector_started", observability.Fields{
			"window_minutes": cfg.SLO.WindowMinutes,
		})
	}

	// Pekerjaan terjadwal: konsumen RabbitMQ (self-hosted) atau jadwal
	// QStash (cloud). Keduanya tidak berjalan bersamaan; RABBITMQ_URL
	// diutamakan dan pendaftaran jadwal QStash dilewati.
	startJobConsumer(workerCtx, cfg, auditCleaner, directoryServices.Audits, payrollService, sloService, logger)
	ensureQStashSchedules(context.Background(), cfg, qstashPublisher, sloService, logger)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("server_starting", observability.Fields{"addr": server.Addr, "env": cfg.AppEnv})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server_error", observability.Fields{"error": err.Error()})
			os.Exit(1)
		}
	}()

	<-shutdown
	logger.Info("shutdown_signal_received", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown_error", observability.Fields{"error": err.Error()})
		os.Exit(1)
	}

	logger.Info("server_stopped", nil)
}
