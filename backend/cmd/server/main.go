package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"simpeg/backend/internal/attendance"
	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/cache"
	"simpeg/backend/internal/config"
	"simpeg/backend/internal/database"
	"simpeg/backend/internal/directory/app"
	"simpeg/backend/internal/directory/repository"
	"simpeg/backend/internal/holiday"
	"simpeg/backend/internal/httpserver"
	"simpeg/backend/internal/jobs"
	"simpeg/backend/internal/notify"
	"simpeg/backend/internal/observability"
	"simpeg/backend/internal/outbox"
	payrollapp "simpeg/backend/internal/payroll/app"
	payrollrepository "simpeg/backend/internal/payroll/repository"
	"simpeg/backend/internal/qstash"
	"simpeg/backend/internal/timeutil"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
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
	attendanceStore := attendance.GormStore{DB: db}
	deviceAuth := attendance.NewGormDeviceAuthenticator(db)
	// Role/permission grant matrix (Opsi A, disetujui 2026-11-10).
	// Deny-by-default: role yang tidak tercantum atau izin yang tidak disebutkan ditolak.
	policy := auth.StaticPolicy{
		"admin": {
			"employee:read:any":       true,
			"employee:create":         true,
			"employee:update:any":     true,
			"employee:archive":        true,
			"employee:role:manage":    true,
			"employee:import":         true,
			"organization:read":       true,
			"organization:manage":     true,
			"attendance:event:ingest": true,
			"attendance:read:any":     true,
			"audit:read":              true,
			"category:read":           true,
			"payroll:read:any":        true,
			"payroll:manage":          true,
			"payroll:schedule":        true,
			"holiday:read":            true,
			"holiday:manage":          true,
			"auth:session:issue":      true,
		},
		"operator": {
			// Cakupan unit: unit penugasan beserta seluruh turunannya.
			"employee:read:homebase":       true,
			"employee:create:homebase":     true,
			"employee:update:homebase":     true,
			"employee:import:homebase":     true,
			"organization:read:homebase":   true,
			"organization:manage:homebase": true,
			"audit:read:homebase":          true,
			"category:read":                true,
			"payroll:read:homebase":        true,
			"payroll:schedule:homebase":    true,
			"holiday:read":                 true,
			"attendance:read:homebase":     true,
		},
		"atasan": {
			"employee:read:homebase":     true,
			"organization:read:homebase": true,
			"category:read":              true,
			"holiday:read":               true,
			"attendance:read:homebase":   true,
		},
		"pegawai": {
			"employee:read:self":   true,
			"payroll:read:self":    true,
			"attendance:read:self": true,
		},
	}
	// Cache daftar kategori pegawai memakai Redis bila dikonfigurasi. Tanpa
	// Redis, pembacaan tetap langsung ke PostgreSQL.
	var categoryCache *cache.Cache
	if cfg.RedisEnabled() {
		categoryCache = cache.New(cache.NewRESTClient(cfg.RedisURL, cfg.RedisToken), cache.Config{
			Namespace: "simpeg",
			TTL:       5 * time.Minute,
		})
	}

	// Lapisan aplikasi menerima port repository; lapisan transport menerima
	// use case. Domain menyimpan aturan, repository menyimpan persistensi.
	directoryServices := httpserver.DirectoryServices{
		Employees:     app.NewEmployeeService(store, policy, store),
		Organizations: app.NewOrganizationService(store, policy, store),
		Categories:    app.NewCategoryService(store, categoryCache),
		Audits:        app.NewAuditService(store, policy, store),
	}
	payrollStore := payrollrepository.GormStore{DB: db}
	payrollService := payrollapp.NewService(payrollStore, policy, store)

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

	// Callback QStash diautentikasi dengan signature QStash, bukan sesi
	// bearer. Subjek signature harus cocok dengan URL tujuan yang dipakai
	// penjadwal.
	auditCleaner := audit.NewCleaner(db, cfg.AuditRetentionDays)
	var qstashOptions *httpserver.QStashOptions
	var qstashPublisher *qstash.Publisher
	if cfg.QStashEnabled() {
		signingKeys := []string{cfg.QStashCurrentSigningKey, cfg.QStashNextSigningKey}
		callbackPath := httpserver.APIPrefix + "/internal/qstash/audit/cleanup"
		destination := jobs.Destination(cfg.AppBaseURL, callbackPath)
		payrollNotifyDestination := jobs.Destination(cfg.AppBaseURL, httpserver.APIPrefix+"/internal/qstash/payroll/notify")
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
		}
		qstashPublisher = &qstash.Publisher{
			BaseURL:      cfg.QStashURL,
			SchedulesURL: cfg.QStashSchedulesURL,
			Token:        cfg.QStashToken,
		}
	}

	mux := httpserver.NewMuxWithOptions(httpserver.MuxOptions{
		Resolver:          resolver,
		Policy:            policy,
		Directory:         directoryServices,
		Attendance:        attendanceStore,
		AttendanceRead:    attendance.NewReadService(attendanceStore, store, policy),
		DeviceAuth:        deviceAuth,
		Payroll:           payrollService,
		Holiday:           holidayService,
		QStash:            qstashOptions,
		SessionRevoker:    auth.GormSessionRevoker{DB: db},
		SessionIssuer:     sessionIssuer,
		ReadinessCheckers: []httpserver.DependencyChecker{database.PingChecker{DB: db}},
	})

	logger := observability.New(os.Stdout)
	rateLimiter := httpserver.NewRateLimiter(httpserver.RateLimitOptions{
		Limit:        cfg.RateLimitRequests,
		Window:       cfg.RateLimitWindow,
		StrictPaths:  []string{httpserver.APIPrefix + "/auth", httpserver.APIPrefix + "/me"},
		StrictLimit:  cfg.RateLimitAuthRequests,
		StrictWindow: cfg.RateLimitAuthWindow,
	})

	handler := observability.RequestIDMiddleware(
		observability.AccessLogMiddleware(observability.AccessLogOptions{
			Logger: logger,
			PrincipalFromContext: func(ctx context.Context) (string, string, bool) {
				principal, ok := auth.PrincipalFromContext(ctx)
				if !ok {
					return "", "", false
				}
				return principal.UserID, principal.RoleName, true
			},
		})(httpserver.SecurityHeadersMiddleware(
			httpserver.StatementTimeoutMiddleware(cfg.DBStatementTimeout)(
				httpserver.CORSMiddleware(httpserver.CORSOptions{
					TrustedOrigins: cfg.TrustedOrigins,
					AllowedMethods: httpserver.DefaultCORSMethods(),
					AllowedHeaders: httpserver.DefaultCORSHeaders(),
				})(rateLimiter.Middleware(
					httpserver.JSONErrorMiddleware(httpserver.VersionMiddleware(httpserver.VersionGuard(mux)))))))))

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Worker outbox mengirim email dari antrean keluar persisten. Worker
	// hanya berjalan bila SMTP dikonfigurasi.
	workerCtx, stopWorker := context.WithCancel(context.Background())
	defer stopWorker()
	if cfg.SMTPEnabled() {
		emailSender := notify.NewSMTPSender(notify.SMTPConfig{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUser,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
			UseTLS:   cfg.SMTPSecure,
			Timeout:  15 * time.Second,
		})
		deliverer := &notify.EmailDeliverer{
			Sender:  emailSender,
			BaseURL: cfg.AppBaseURL,
			// Sumber konfirmasi pembayaran adalah modul penggajian: email gaji
			// hanya terkirim bila pembayaran sudah tercatat di database.
			Confirmations: payrollService,
		}
		dispatcher := outbox.NewDispatcher(outbox.NewGormStore(db), deliverer, outbox.DispatcherOptions{
			BatchSize:   20,
			Interval:    10 * time.Second,
			MaxAttempts: 5,
			BaseBackoff: 30 * time.Second,
			MaxBackoff:  30 * time.Minute,
			Logger:      slog.Default(),
		})
		go func() {
			if err := dispatcher.RunClaimAndDeliverLoop(workerCtx); err != nil {
				logger.Error("outbox_worker_stopped", observability.Fields{"error": err.Error()})
			}
		}()
		logger.Info("outbox_worker_started", observability.Fields{"interval_seconds": 10})
	}

	// QStash menjadwalkan pekerjaan asinkron. Jadwal mendaftarkan callback
	// pembersihan jejak audit dan callback pemberitahuan slip gaji H+1;
	// keduanya idempoten sehingga percobaan ulang QStash tidak merusak hasil.
	if qstashPublisher != nil {
		ensureSchedule := func(callbackPath, cron, kind string) {
			destination := jobs.Destination(cfg.AppBaseURL, callbackPath)
			payload, err := json.Marshal(map[string]string{"kind": kind})
			if err != nil {
				logger.Error("qstash_job_payload_failed", observability.Fields{"error": err.Error()})
				return
			}
			scheduleID, err := jobs.EnsureSchedule(context.Background(), qstashPublisher, qstash.ScheduleRequest{
				Destination: destination,
				Body:        string(payload),
				Cron:        cron,
				Retries:     3,
			})
			if err != nil {
				logger.Error("qstash_schedule_failed", observability.Fields{"error": err.Error(), "destination": destination})
				return
			}
			logger.Info("qstash_schedule_ready", observability.Fields{
				"schedule_id": scheduleID,
				"destination": destination,
				"cron":        cron,
			})
		}
		ensureSchedule(httpserver.APIPrefix+"/internal/qstash/audit/cleanup", jobs.CronAuditCleanup, httpserver.JobKindAuditCleanup)
		ensureSchedule(httpserver.APIPrefix+"/internal/qstash/payroll/notify", jobs.CronPayrollNotify, httpserver.JobKindPayrollNotify)
	}

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
