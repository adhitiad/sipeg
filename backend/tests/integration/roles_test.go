//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"simpeg/backend/internal/auth"
)

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

func keys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for key := range values { result = append(result, key) }
	return result
}
