//go:build integration

// Package integration menjalankan server SIMPEG sebenarnya terhadap PostgreSQL
// disposable. Tes gagal (bukan Skip) bila SIMPEG_TEST_DATABASE_URL tidak ada.
package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
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
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"

	"simpeg/backend/internal/auth"
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

func TestPostgresMigrationsAndAllRoleAuthorization(t *testing.T) {
	t.Run("all repository migrations are applied", func(t *testing.T) {
		var version int64
		if err := testDB.Raw(`SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied = TRUE`).Scan(&version).Error; err != nil {
			t.Fatalf("read Goose version: %v", err)
		}
		if version < 33 {
			t.Fatalf("migration version=%d; all 33 repository migrations are required", version)
		}
	})

	roleCases := []struct {
		role              string
		rbacStatus        int
		payrollReadStatus int
	}{
		{role: "admin", rbacStatus: http.StatusOK, payrollReadStatus: http.StatusOK},
		{role: "operator", rbacStatus: http.StatusForbidden, payrollReadStatus: http.StatusOK},
		{role: "atasan", rbacStatus: http.StatusForbidden, payrollReadStatus: http.StatusForbidden},
		{role: "pegawai", rbacStatus: http.StatusForbidden, payrollReadStatus: http.StatusOK},
	}
	for _, tc := range roleCases {
		t.Run("role_"+tc.role, func(t *testing.T) {
			principal := testData.Actors[tc.role]
			for _, path := range []string{"/api/v1/employees", "/api/v1/cuti"} {
				status, body := apiRequest(t, principal, http.MethodGet, path, nil)
				if status != http.StatusOK {
					t.Fatalf("role %s GET %s status=%d body=%s; want 200", tc.role, path, status, body)
				}
			}
			status, body := apiRequest(t, principal, http.MethodGet, "/api/v1/admin/rbac/roles", nil)
			if status != tc.rbacStatus {
				t.Fatalf("role %s RBAC catalog status=%d body=%s; want %d", tc.role, status, body, tc.rbacStatus)
			}
			status, body = apiRequest(t, principal, http.MethodGet, "/api/v1/payroll/slips", nil)
			if status != tc.payrollReadStatus {
				t.Fatalf("role %s payroll list status=%d body=%s; want %d", tc.role, status, body, tc.payrollReadStatus)
			}
		})
	}
}

func TestRBACAssignmentsAreAuthoritativeAndRevokedImmediately(t *testing.T) {
	pegawai := testData.Actors["pegawai"]
	status, body := apiRequest(t, pegawai, http.MethodGet, "/api/v1/admin/rbac/roles", nil)
	if status != http.StatusForbidden {
		t.Fatalf("pegawai without RBAC assignment must not read RBAC catalog: status=%d body=%s", status, body)
	}

	var roleID, permissionID int64
	if err := testDB.Raw(
		`INSERT INTO "rbac_roles" ("code", "name", "tier_level") VALUES (?, ?, ?) RETURNING "id"`,
		"IT_INTEGRATION_"+strings.ReplaceAll(uuid.NewString(), "-", "")[:10], "Integration RBAC Reader", 4,
	).Scan(&roleID).Error; err != nil {
		t.Fatalf("create RBAC role: %v", err)
	}
	if err := testDB.Raw(
		`INSERT INTO "rbac_permissions" ("module", "action") VALUES (?, ?) RETURNING "id"`,
		"rbac", "read",
	).Scan(&permissionID).Error; err != nil {
		t.Fatalf("create RBAC permission: %v", err)
	}
	if err := testDB.Exec(
		`INSERT INTO "rbac_role_permissions" ("role_id", "permission_id") VALUES (?, ?)`,
		roleID, permissionID,
	).Error; err != nil {
		t.Fatalf("grant RBAC permission: %v", err)
	}

	var assignmentID int64
	if err := testDB.Raw(
		`INSERT INTO "rbac_user_role_scopes" ("user_id", "role_id", "denied_permissions")
		 VALUES (?, ?, '[]'::jsonb) RETURNING "id"`,
		pegawai.ID, roleID,
	).Scan(&assignmentID).Error; err != nil {
		t.Fatalf("create RBAC user assignment: %v", err)
	}

	status, body = apiRequest(t, pegawai, http.MethodGet, "/api/v1/admin/rbac/roles", nil)
	if status != http.StatusOK {
		t.Fatalf("active RBAC assignment with permission must grant access: status=%d body=%s", status, body)
	}

	if err := testDB.Exec(
		`UPDATE "rbac_user_role_scopes" SET "denied_permissions" = '["rbac:read"]'::jsonb WHERE "id" = ?`,
		assignmentID,
	).Error; err != nil {
		t.Fatalf("apply RBAC permission blacklist: %v", err)
	}
	status, body = apiRequest(t, pegawai, http.MethodGet, "/api/v1/admin/rbac/roles", nil)
	if status != http.StatusForbidden {
		t.Fatalf("RBAC blacklist must override granted permission on next request: status=%d body=%s", status, body)
	}

	if err := testDB.Exec(
		`UPDATE "rbac_user_role_scopes" SET "denied_permissions" = '[]'::jsonb WHERE "id" = ?`,
		assignmentID,
	).Error; err != nil {
		t.Fatalf("clear RBAC blacklist: %v", err)
	}
	if err := testDB.Exec(
		`DELETE FROM "rbac_role_permissions" WHERE "role_id" = ? AND "permission_id" = ?`,
		roleID, permissionID,
	).Error; err != nil {
		t.Fatalf("revoke RBAC permission: %v", err)
	}
	status, body = apiRequest(t, pegawai, http.MethodGet, "/api/v1/admin/rbac/roles", nil)
	if status != http.StatusForbidden {
		t.Fatalf("revoked RBAC grant must be denied on the next request without cache delay: status=%d body=%s", status, body)
	}

	if err := testDB.Exec(`DELETE FROM "rbac_user_role_scopes" WHERE "id" = ?`, assignmentID).Error; err != nil {
		t.Fatalf("remove integration RBAC assignment: %v", err)
	}
	if err := testDB.Exec(`DELETE FROM "rbac_roles" WHERE "id" = ?`, roleID).Error; err != nil {
		t.Fatalf("remove integration RBAC role: %v", err)
	}
	if err := testDB.Exec(`DELETE FROM "rbac_permissions" WHERE "id" = ?`, permissionID).Error; err != nil {
		t.Fatalf("remove integration RBAC permission: %v", err)
	}
}

func TestJWTClaimsCannotOverrideDatabaseRoleOrAccountStatus(t *testing.T) {
	operator := testData.Actors["operator"]
	var originalRoleID string
	if err := testDB.Raw(`SELECT "roleId"::text FROM "user" WHERE "id" = ?`, operator.ID).Scan(&originalRoleID).Error; err != nil {
		t.Fatalf("read original operator role: %v", err)
	}
	t.Cleanup(func() {
		_ = testDB.Exec(`UPDATE "user" SET "aktif" = TRUE, "roleId" = ? WHERE "id" = ?`, originalRoleID, operator.ID).Error
	})
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.BetterAuthClaims{
		UserID: operator.ID,
		Email: operator.Email,
		RoleName: "admin",
		HomebaseID: testData.RemoteUnitID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}).SignedString([]byte(integrationSecret))
	if err != nil {
		t.Fatalf("sign stale-role integration JWT: %v", err)
	}
	jwtActor := operator
	jwtActor.Token = token

	status, body := apiRequest(t, jwtActor, http.MethodGet, "/api/v1/admin/rbac/roles", nil)
	if status != http.StatusForbidden {
		t.Fatalf("stale admin JWT claim must use DB role operator and be denied: status=%d body=%s", status, body)
	}

	adminRoleID := testData.Actors["admin"]
	var adminRole string
	if err := testDB.Raw(`SELECT "roleId"::text FROM "user" WHERE "id" = ?`, adminRoleID.ID).Scan(&adminRole).Error; err != nil {
		t.Fatalf("read DB admin role ID: %v", err)
	}
	if err := testDB.Exec(`UPDATE "user" SET "roleId" = ? WHERE "id" = ?`, adminRole, operator.ID).Error; err != nil {
		t.Fatalf("promote integration operator in database: %v", err)
	}
	status, body = apiRequest(t, jwtActor, http.MethodGet, "/api/v1/admin/rbac/roles", nil)
	if status != http.StatusOK {
		t.Fatalf("JWT principal must reflect latest DB role after promotion: status=%d body=%s", status, body)
	}

	if err := testDB.Exec(`UPDATE "user" SET "aktif" = FALSE WHERE "id" = ?`, operator.ID).Error; err != nil {
		t.Fatalf("deactivate integration account: %v", err)
	}
	status, body = apiRequest(t, jwtActor, http.MethodGet, "/api/v1/admin/rbac/roles", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("inactive account JWT must be rejected: status=%d body=%s", status, body)
	}
	if err := testDB.Exec(`UPDATE "user" SET "aktif" = TRUE WHERE "id" = ?`, operator.ID).Error; err != nil {
		t.Fatalf("restore integration account: %v", err)
	}
}

func TestOpaqueSessionRevocationAndExpiryAreEnforced(t *testing.T) {
	admin := testData.Actors["admin"]
	var originalExpiry time.Time
	if err := testDB.Raw(`SELECT "expiresAt" FROM "session" WHERE "token" = ?`, admin.Token).Scan(&originalExpiry).Error; err != nil {
		t.Fatalf("read original session expiry: %v", err)
	}
	t.Cleanup(func() {
		_ = testDB.Exec(`UPDATE "session" SET "expiresAt" = ? WHERE "token" = ?`, originalExpiry, admin.Token).Error
	})
	if err := testDB.Exec(
		`UPDATE "session" SET "expiresAt" = ? WHERE "token" = ?`,
		time.Now().UTC().Add(-time.Minute), admin.Token,
	).Error; err != nil {
		t.Fatalf("expire integration session: %v", err)
	}
	status, body := apiRequest(t, admin, http.MethodGet, "/api/v1/employees", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("expired opaque session must be rejected: status=%d body=%s", status, body)
	}
}

func TestEmployeeListAndDetailEnforceUnitAndSelfScope(t *testing.T) {
	cases := []struct {
		role       string
		mustSee    []string
		mustNotSee []string
	}{
		{"admin", []string{testData.RootEmployee, testData.ChildEmployee, testData.RemoteEmployee}, nil},
		{"operator", []string{testData.RootEmployee, testData.ChildEmployee}, []string{testData.RemoteEmployee}},
		{"atasan", []string{testData.RootEmployee, testData.ChildEmployee}, []string{testData.RemoteEmployee}},
		{"pegawai", []string{testData.Actors["pegawai"].ID}, []string{testData.RootEmployee, testData.ChildEmployee, testData.RemoteEmployee}},
	}

	for _, tc := range cases {
		t.Run("scope_"+tc.role, func(t *testing.T) {
			status, body := apiRequest(t, testData.Actors[tc.role], http.MethodGet, "/api/v1/employees?page=1&pageSize=100", nil)
			if status != http.StatusOK {
				t.Fatalf("list status=%d body=%s", status, body)
			}
			var page struct {
				Items []struct {
					ID string `json:"id"`
				} `json:"items"`
			}
			if err := json.Unmarshal(body, &page); err != nil {
				t.Fatalf("decode employee page: %v; body=%s", err, body)
			}
			visible := map[string]bool{}
			for _, item := range page.Items { visible[item.ID] = true }
			for _, id := range tc.mustSee {
				if !visible[id] { t.Errorf("role %s must see %s; visible=%v", tc.role, id, keys(visible)) }
			}
			for _, id := range tc.mustNotSee {
				if visible[id] { t.Errorf("role %s must not see %s", tc.role, id) }
			}
		})
	}

	for _, role := range []string{"operator", "atasan", "pegawai"} {
		t.Run("cross_unit_detail_"+role, func(t *testing.T) {
			status, body := apiRequest(t, testData.Actors[role], http.MethodGet, "/api/v1/employees/"+testData.RemoteEmployee, nil)
			if status != http.StatusForbidden {
				t.Fatalf("role %s cross-unit detail status=%d body=%s; want 403", role, status, body)
			}
		})
	}
	if status, body := apiRequest(t, testData.Actors["admin"], http.MethodGet, "/api/v1/employees/"+testData.RemoteEmployee, nil); status != http.StatusOK {
		t.Fatalf("admin cross-unit detail status=%d body=%s; want 200", status, body)
	}
}

func TestLeaveSubmissionApprovalAndCancellationPersistAtomically(t *testing.T) {
	pegawai, atasan := testData.Actors["pegawai"], testData.Actors["atasan"]
	payload := map[string]any{
		"userId": pegawai.ID, "jenis": "TAHUNAN",
		"tanggalMulai": "2031-03-03", "tanggalSelesai": "2031-03-04",
		"alamatCuti": "Alamat uji integrasi",
		"alasan": "Pengujian alur cuti PostgreSQL disposable",
	}
	status, body := apiRequest(t, pegawai, http.MethodPost, "/api/v1/cuti", payload)
	if status != http.StatusCreated {
		t.Fatalf("create leave status=%d body=%s; want 201", status, body)
	}
	var created struct {
		ID string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &created); err != nil { t.Fatalf("decode leave: %v; body=%s", err, body) }
	if created.ID == "" || created.Status != "MENUNGGU" { t.Fatalf("unexpected initial state: %+v", created) }

	status, body = apiRequest(t, pegawai, http.MethodPost, "/api/v1/cuti/"+created.ID+"/approve", map[string]any{"catatan": "must be denied"})
	if status != http.StatusForbidden { t.Fatalf("pegawai approval status=%d body=%s; want 403", status, body) }

	status, body = apiRequest(t, atasan, http.MethodPost, "/api/v1/cuti/"+created.ID+"/approve", map[string]any{"catatan": "Disetujui oleh atasan uji"})
	if status != http.StatusOK { t.Fatalf("atasan approval status=%d body=%s; want 200", status, body) }

	var leaveStatus, approverID string
	if err := testDB.Raw(`SELECT "status"::text, "disetujuiBy"::text FROM "Cuti" WHERE "id" = ?`, created.ID).Row().Scan(&leaveStatus, &approverID); err != nil {
		t.Fatalf("read approved leave: %v", err)
	}
	if leaveStatus != "DISETUJUI" || approverID != atasan.ID {
		t.Fatalf("approved leave state=%s approver=%s", leaveStatus, approverID)
	}

	var used int
	if err := testDB.Raw(`SELECT "terpakai" FROM "SaldoCuti" WHERE "userId" = ? AND "tahun" = 2031`, pegawai.ID).Scan(&used).Error; err != nil {
		t.Fatalf("read annual leave balance: %v", err)
	}
	if used != 2 { t.Fatalf("annual leave used=%d; want 2", used) }
	assertAuditAction(t, "cuti.created")
	assertAuditAction(t, "cuti.approved")

	status, body = apiRequest(t, pegawai, http.MethodPost, "/api/v1/cuti/"+created.ID+"/cancel", map[string]any{"catatan": "Pembatalan untuk verifikasi saldo"})
	if status != http.StatusOK { t.Fatalf("self cancellation status=%d body=%s; want 200", status, body) }
	if err := testDB.Raw(`SELECT "status"::text FROM "Cuti" WHERE "id" = ?`, created.ID).Scan(&leaveStatus).Error; err != nil {
		t.Fatalf("read cancelled leave: %v", err)
	}
	if leaveStatus != "BATAL" { t.Fatalf("leave status=%s; want BATAL", leaveStatus) }
	if err := testDB.Raw(`SELECT "terpakai" FROM "SaldoCuti" WHERE "userId" = ? AND "tahun" = 2031`, pegawai.ID).Scan(&used).Error; err != nil {
		t.Fatalf("read refunded leave balance: %v", err)
	}
	if used != 0 { t.Fatalf("annual leave used after cancellation=%d; want 0", used) }
	assertAuditAction(t, "cuti.canceled")
}

func TestPayrollPublicationAndPaymentAreRoleGuardedAndTransactional(t *testing.T) {
	admin, operator, payrollUser := testData.Actors["admin"], testData.Actors["operator"], testData.Actors["pegawai"]
	slipID := uuid.NewString()
	period := time.Date(2031, time.February, 1, 0, 0, 0, 0, time.UTC)
	const salaryAmount int64 = 987654321
	if err := testDB.Exec(
		`INSERT INTO "gaji" ("id", "userId", "nomorSlip", "periode", "tanggalGaji", "gajiPokok", "totalGaji", "status")
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'DRAF')`,
		slipID, payrollUser.ID, "IT-SLIP-"+strings.ReplaceAll(slipID, "-", ""), period, period, float64(salaryAmount), float64(salaryAmount),
	).Error; err != nil { t.Fatalf("seed payroll draft: %v", err) }

	status, body := apiRequest(t, operator, http.MethodPost, "/api/v1/payroll/slips/"+slipID+"/terbit", nil)
	if status != http.StatusForbidden { t.Fatalf("operator publish status=%d body=%s; want 403", status, body) }
	assertPayrollStatus(t, slipID, "DRAF")

	status, body = apiRequest(t, admin, http.MethodPost, "/api/v1/payroll/slips/"+slipID+"/terbit", nil)
	if status != http.StatusOK { t.Fatalf("admin publish status=%d body=%s; want 200", status, body) }
	assertPayrollStatus(t, slipID, "TERBIT")
	assertAuditAction(t, "payroll.slip.published")

	payment := map[string]any{
		"referensiTransaksi": "IT-BANK-" + strings.ReplaceAll(slipID, "-", "")[:12],
		"pelakuPengesahan": "Petugas Integrasi",
		"tanggalBayar": "2031-02-28",
		"keterangan": "Konfirmasi pembayaran dari pengujian PostgreSQL",
	}
	status, body = apiRequest(t, operator, http.MethodPost, "/api/v1/payroll/slips/"+slipID+"/payment", payment)
	if status != http.StatusForbidden { t.Fatalf("operator payment status=%d body=%s; want 403", status, body) }
	assertPayrollStatus(t, slipID, "TERBIT")
	assertPaymentCount(t, slipID, 0)

	status, body = apiRequest(t, admin, http.MethodPost, "/api/v1/payroll/slips/"+slipID+"/payment", payment)
	if status != http.StatusOK { t.Fatalf("admin payment status=%d body=%s; want 200", status, body) }
	assertPayrollStatus(t, slipID, "TERBAYAR")
	assertPaymentCount(t, slipID, 1)
	assertAuditAction(t, "payroll.payment.confirmed")

	dedupeKey := "salary-paid:2031-02:" + payrollUser.ID
	var outboxPayload string
	if err := testDB.Raw(`SELECT "payload" FROM "outboxMessage" WHERE "dedupeKey" = ?`, dedupeKey).Scan(&outboxPayload).Error; err != nil {
		t.Fatalf("read salary outbox: %v", err)
	}
	if strings.TrimSpace(outboxPayload) == "" { t.Fatal("payment transaction did not create an outbox message") }
	if strings.Contains(outboxPayload, strconv.FormatInt(salaryAmount, 10)) ||
		strings.Contains(strings.ToLower(outboxPayload), "totalgaji") ||
		strings.Contains(strings.ToLower(outboxPayload), "gajipokok") {
		t.Fatalf("salary amount/fields leaked into notification payload: %s", outboxPayload)
	}

	status, body = apiRequest(t, admin, http.MethodPost, "/api/v1/payroll/slips/"+slipID+"/payment", payment)
	if status != http.StatusConflict { t.Fatalf("duplicate payment status=%d body=%s; want 409", status, body) }
	assertPaymentCount(t, slipID, 1)
}

func apiRequest(t *testing.T, principal actor, method, path string, payload any) (int, []byte) {
	t.Helper()
	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(payload)
		if err != nil { t.Fatalf("encode request: %v", err) }
		body = bytes.NewReader(encoded)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil { t.Fatalf("create request: %v", err) }
	request.Header.Set("Authorization", "Bearer "+principal.Token)
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil { t.Fatalf("%s %s request failed: %v", method, path, err) }
	defer response.Body.Close()
	var responseBody bytes.Buffer
	if _, err := responseBody.ReadFrom(response.Body); err != nil { t.Fatalf("read response: %v", err) }
	return response.StatusCode, responseBody.Bytes()
}

func assertPayrollStatus(t *testing.T, slipID, expected string) {
	t.Helper()
	var actual string
	if err := testDB.Raw(`SELECT "status"::text FROM "gaji" WHERE "id" = ?`, slipID).Scan(&actual).Error; err != nil {
		t.Fatalf("read payroll status: %v", err)
	}
	if actual != expected { t.Fatalf("payroll status=%s; want %s", actual, expected) }
}

func assertPaymentCount(t *testing.T, slipID string, expected int64) {
	t.Helper()
	var count int64
	if err := testDB.Raw(`SELECT COUNT(*) FROM "pembayaran" WHERE "gajiId" = ?`, slipID).Scan(&count).Error; err != nil {
		t.Fatalf("count payment records: %v", err)
	}
	if count != expected { t.Fatalf("payment count=%d; want %d", count, expected) }
}

func assertAuditAction(t *testing.T, action string) {
	t.Helper()
	var count int64
	if err := testDB.Raw(`SELECT COUNT(*) FROM "auditLog" WHERE "aksi" = ?`, action).Scan(&count).Error; err != nil {
		t.Fatalf("count audit action %s: %v", action, err)
	}
	if count == 0 { t.Fatalf("audit action %q was not persisted", action) }
}

func keys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for key := range values { result = append(result, key) }
	return result
}
