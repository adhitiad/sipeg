//go:build integration

// Package integration menjalankan server SIMPEG sebenarnya terhadap PostgreSQL
// disposable. Tes gagal (bukan Skip) bila SIMPEG_TEST_DATABASE_URL tidak ada.
package integration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"gorm.io/gorm"

	"simpeg/backend/internal/database"
	"simpeg/backend/migrations"
)

const (
	disposableDatabaseName = "simpeg_integration"
	integrationSecret      = "simpeg-integration-secret-0123456789"
	integrationPIIKey      = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
)

type actor struct {
	ID, Token, Role, HomebaseID, Email string
}

type fixture struct {
	Actors                    map[string]actor
	RootUnitID, ChildUnitID   string
	RemoteUnitID              string
	RootEmployee, ChildEmployee string
	RemoteEmployee            string
}

var (
	testDB     *gorm.DB
	testSQL    *sql.DB
	testData   fixture
	baseURL    string
	serverCmd  *exec.Cmd
	serverDone chan error
)

func TestMain(m *testing.M) {
	dsn := strings.TrimSpace(os.Getenv("SIMPEG_TEST_DATABASE_URL"))
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "INTEGRATION_STATUS=NOT RUN — SIMPEG_TEST_DATABASE_URL wajib diisi; suite tidak boleh dilewati.")
		os.Exit(2)
	}
	if err := validateDisposableDSN(dsn); err != nil {
		fmt.Fprintf(os.Stderr, "INTEGRATION_STATUS=FAIL — database guard: %v\n", err)
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var err error
	testSQL, err = sql.Open("pgx", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "INTEGRATION_STATUS=FAIL — open PostgreSQL: %v\n", err)
		os.Exit(1)
	}
	if err = testSQL.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "INTEGRATION_STATUS=FAIL — PostgreSQL tidak siap: %v\n", err)
		_ = testSQL.Close()
		os.Exit(1)
	}

	goose.SetBaseFS(migrations.Files)
	if err = goose.SetDialect("postgres"); err != nil {
		fmt.Fprintf(os.Stderr, "INTEGRATION_STATUS=FAIL — configure Goose: %v\n", err)
		_ = testSQL.Close()
		os.Exit(1)
	}
	if err = goose.UpContext(ctx, testSQL, "."); err != nil {
		fmt.Fprintf(os.Stderr, "INTEGRATION_STATUS=FAIL — seluruh migrasi PostgreSQL wajib sukses: %v\n", err)
		_ = testSQL.Close()
		os.Exit(1)
	}

	testDB, err = database.OpenPostgres(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "INTEGRATION_STATUS=FAIL — open GORM: %v\n", err)
		_ = testSQL.Close()
		os.Exit(1)
	}
	testData, err = seedFixture(ctx, testDB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "INTEGRATION_STATUS=FAIL — seed fixture: %v\n", err)
		_ = testSQL.Close()
		os.Exit(1)
	}
	if err = startProductionServer(dsn); err != nil {
		fmt.Fprintf(os.Stderr, "INTEGRATION_STATUS=FAIL — server composition: %v\n", err)
		stopProductionServer()
		_ = testSQL.Close()
		os.Exit(1)
	}

	exitCode := m.Run()
	stopProductionServer()
	_ = testSQL.Close()
	if exitCode == 0 {
		fmt.Fprintln(os.Stdout, "INTEGRATION_STATUS=PASS — PostgreSQL disposable, migrasi, empat role, scope, cuti, dan payroll tervalidasi.")
	} else {
		fmt.Fprintf(os.Stderr, "INTEGRATION_STATUS=FAIL — go test exit code %d\n", exitCode)
	}
	os.Exit(exitCode)
}

func validateDisposableDSN(dsn string) error {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("DSN tidak valid: %w", err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return errors.New("scheme harus postgres atau postgresql")
	}
	if strings.Trim(parsed.Path, "/") != disposableDatabaseName {
		return fmt.Errorf("database harus bernama %q", disposableDatabaseName)
	}
	switch parsed.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return nil
	default:
		return fmt.Errorf("host %q ditolak; gunakan PostgreSQL lokal/container disposable", parsed.Hostname())
	}
}

func startProductionServer(dsn string) error {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return errors.New("resolve test file path")
	}
	backendRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("reserve test port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	baseURL = "http://127.0.0.1:" + strconv.Itoa(port)

	binary := filepath.Join(os.TempDir(), "simpeg-integration-server-"+strconv.Itoa(os.Getpid()))
	build := exec.Command("go", "build", "-o", binary, "./cmd/server")
	build.Dir = backendRoot
	if output, buildErr := build.CombinedOutput(); buildErr != nil {
		return fmt.Errorf("build API server: %w\n%s", buildErr, output)
	}

	serverCmd = exec.Command(binary)
	serverCmd.Dir = backendRoot
	serverCmd.Env = overrideEnv(os.Environ(),
		"APP_ENV=development",
		"PORT="+strconv.Itoa(port),
		"DATABASE_URL="+dsn,
		"BETTER_AUTH_SECRET="+integrationSecret,
		"PII_ENCRYPTION_KEY="+integrationPIIKey,
		"TRUSTED_ORIGINS="+baseURL,
		"APP_BASE_URL="+baseURL,
		"RABBITMQ_URL=",
		"QSTASH_TOKEN=",
		"QSTASH_CURRENT_SIGNING_KEY=",
		"QSTASH_NEXT_SIGNING_KEY=",
		"SMTP_HOST=",
		"UPSTASH_REDIS_REST_URL=",
		"UPSTASH_REDIS_REST_TOKEN=",
		"CLOUDINARY_CLOUD_NAME=",
		"CLOUDINARY_API_KEY=",
		"CLOUDINARY_API_SECRET=",
	)
	serverCmd.Stdout, serverCmd.Stderr = os.Stdout, os.Stderr
	if err := serverCmd.Start(); err != nil {
		_ = os.Remove(binary)
		return fmt.Errorf("start API binary: %w", err)
	}
	serverDone = make(chan error, 1)
	go func() { serverDone <- serverCmd.Wait() }()

	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(40 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case err := <-serverDone:
			serverDone = nil
			_ = os.Remove(binary)
			return fmt.Errorf("API server exited before readiness: %w", err)
		default:
		}
		response, requestErr := client.Get(baseURL + "/ready")
		if requestErr == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("readiness status %d", response.StatusCode)
		} else {
			lastErr = requestErr
		}
		time.Sleep(250 * time.Millisecond)
	}
	_ = os.Remove(binary)
	return fmt.Errorf("API tidak ready dalam 40 detik: %v", lastErr)
}

func stopProductionServer() {
	if serverCmd == nil || serverCmd.Process == nil {
		return
	}
	_ = serverCmd.Process.Signal(os.Interrupt)
	if serverDone != nil {
		select {
		case <-serverDone:
		case <-time.After(5 * time.Second):
			_ = serverCmd.Process.Kill()
			<-serverDone
		}
	}
	_ = os.Remove(serverCmd.Path)
	serverCmd, serverDone = nil, nil
}

func overrideEnv(base []string, overrides ...string) []string {
	overrideKeys := make(map[string]bool, len(overrides))
	for _, item := range overrides {
		key, _, ok := strings.Cut(item, "=")
		if ok {
			overrideKeys[key] = true
		}
	}
	result := make([]string, 0, len(base)+len(overrides))
	for _, item := range base {
		key, _, ok := strings.Cut(item, "=")
		if !ok || overrideKeys[key] {
			continue
		}
		result = append(result, item)
	}
	return append(result, overrides...)
}

func seedFixture(ctx context.Context, db *gorm.DB) (fixture, error) {
	f := fixture{Actors: map[string]actor{}}
	roleIDs := make(map[string]string)
	for _, role := range []string{"admin", "operator", "atasan", "pegawai"} {
		id := uuid.NewString()
		if err := db.WithContext(ctx).Exec(
			`INSERT INTO "Role" ("id", "name") VALUES (?, ?) ON CONFLICT ("name") DO NOTHING`,
			id, role,
		).Error; err != nil {
			return f, fmt.Errorf("seed role %s: %w", role, err)
		}
		var storedID string
		if err := db.WithContext(ctx).Raw(`SELECT "id"::text FROM "Role" WHERE "name" = ?`, role).Scan(&storedID).Error; err != nil {
			return f, fmt.Errorf("read role %s: %w", role, err)
		}
		if storedID == "" {
			return f, fmt.Errorf("role %s was not created", role)
		}
		roleIDs[role] = storedID
	}

	f.RootUnitID, f.ChildUnitID, f.RemoteUnitID = uuid.NewString(), uuid.NewString(), uuid.NewString()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	if err := db.WithContext(ctx).Exec(
		`INSERT INTO "Homebase" ("id", "name") VALUES (?, ?)`, f.RootUnitID, "IT Root "+suffix,
	).Error; err != nil {
		return f, fmt.Errorf("seed root unit: %w", err)
	}
	if err := db.WithContext(ctx).Exec(
		`INSERT INTO "Homebase" ("id", "name", "parentId") VALUES (?, ?, ?)`, f.ChildUnitID, "IT Child "+suffix, f.RootUnitID,
	).Error; err != nil {
		return f, fmt.Errorf("seed child unit: %w", err)
	}
	if err := db.WithContext(ctx).Exec(
		`INSERT INTO "Homebase" ("id", "name") VALUES (?, ?)`, f.RemoteUnitID, "IT Remote "+suffix,
	).Error; err != nil {
		return f, fmt.Errorf("seed remote unit: %w", err)
	}

	createUser := func(role, name, homebaseID string, withSession bool) (actor, error) {
		id := uuid.NewString()
		email := "integration-" + strings.ReplaceAll(id, "-", "") + "@example.invalid"
		joined := time.Date(2018, time.January, 2, 0, 0, 0, 0, time.UTC)
		if err := db.WithContext(ctx).Exec(
			`INSERT INTO "user" ("id", "email", "nama", "roleId", "homebaseId", "tanggalMasuk", "aktif")
			 VALUES (?, ?, ?, ?, ?, ?, TRUE)`,
			id, email, name+" "+suffix, roleIDs[role], homebaseID, joined,
		).Error; err != nil {
			return actor{}, fmt.Errorf("seed user %s: %w", name, err)
		}
		item := actor{ID: id, Role: role, HomebaseID: homebaseID, Email: email}
		if withSession {
			item.Token = "it-session-" + strings.ReplaceAll(uuid.NewString(), "-", "")
			if err := db.WithContext(ctx).Exec(
				`INSERT INTO "session" ("id", "userId", "expiresAt", "token") VALUES (?, ?, ?, ?)`,
				uuid.NewString(), id, time.Now().UTC().Add(24*time.Hour), item.Token,
			).Error; err != nil {
				return actor{}, fmt.Errorf("seed session %s: %w", name, err)
			}
		}
		return item, nil
	}

	for _, role := range []string{"admin", "operator", "atasan", "pegawai"} {
		item, err := createUser(role, "Actor "+role, f.RootUnitID, true)
		if err != nil {
			return f, err
		}
		f.Actors[role] = item
	}
	rootEmployee, err := createUser("pegawai", "Employee Root", f.RootUnitID, false)
	if err != nil { return f, err }
	childEmployee, err := createUser("pegawai", "Employee Child", f.ChildUnitID, false)
	if err != nil { return f, err }
	remoteEmployee, err := createUser("pegawai", "Employee Remote", f.RemoteUnitID, false)
	if err != nil { return f, err }
	f.RootEmployee, f.ChildEmployee, f.RemoteEmployee = rootEmployee.ID, childEmployee.ID, remoteEmployee.ID
	return f, nil
}

