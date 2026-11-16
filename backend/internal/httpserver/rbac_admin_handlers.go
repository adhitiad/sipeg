package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

// RBACReader adalah port pembacaan dan penulisan master RBAC berbasis scoped
// assignment. Mencakup operasi baca (list) dan tulis (CRUD) untuk role,
// permission, matriks role-permission, assignment user-role, dan delegasi.
type RBACReader interface {
	ListRoles(ctx context.Context) ([]models.RbacRole, error)
	ListPermissions(ctx context.Context) ([]models.RbacPermission, error)
	ListRolePermissions(ctx context.Context) ([]models.RbacRolePermission, error)
	ListUserRoleScopes(ctx context.Context, userID string) ([]models.RbacUserRoleScope, error)

	CreateRole(ctx context.Context, role models.RbacRole) (models.RbacRole, error)
	UpdateRole(ctx context.Context, id uint, role models.RbacRole) (models.RbacRole, error)
	DeleteRole(ctx context.Context, id uint) error

	CreatePermission(ctx context.Context, perm models.RbacPermission) (models.RbacPermission, error)
	DeletePermission(ctx context.Context, id uint) error

	AssignPermissionToRole(ctx context.Context, roleID, permID uint) error
	RevokePermissionFromRole(ctx context.Context, roleID, permID uint) error

	CreateUserRoleScope(ctx context.Context, scope models.RbacUserRoleScope) (models.RbacUserRoleScope, error)
	DeleteUserRoleScope(ctx context.Context, id uint) error

	ListDelegasi(ctx context.Context, userID string) ([]models.Delegasi, error)
	CreateDelegasi(ctx context.Context, d models.Delegasi) (models.Delegasi, error)
	RevokeDelegasi(ctx context.Context, id uint) error
}

type RBACRoleDTO struct {
	ID        uint   `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	TierLevel int    `json:"tierLevel"`
}

// RBACPermissionDTO adalah bentuk publik permission master RBAC.
type RBACPermissionDTO struct {
	ID     uint   `json:"id"`
	Module string `json:"module"`
	Action string `json:"action"`
}

// RBACUserRoleScopeDTO adalah bentuk publik assignment user-role dengan scope.
type RBACUserRoleScopeDTO struct {
	ID                uint    `json:"id"`
	UserID            string  `json:"userId"`
	RoleID            uint    `json:"roleId"`
	BiroCode          *string `json:"biroCode"`
	FakultasID        *string `json:"fakultasId"`
	JurusanID         *string `json:"jurusanId"`
	UnitID            *string `json:"unitId"`
	DeniedPermissions []byte  `json:"deniedPermissions"`
}

// RegisterRBACAdminRoutes mendaftarkan endpoint admin pembacaan dan penulisan
// master RBAC. Endpoint dibatasi permission rbac:read (baca) dan rbac:manage
// (tulis); deny-by-default untuk role lain.
func RegisterRBACAdminRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, reader RBACReader) {
	if reader == nil {
		return
	}
	api := rbacAdminAPI{reader: reader}
	protected := func(permission auth.Permission, handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, permission, handler))
	}
	mux.Handle("GET "+APIPrefix+"/admin/rbac/roles", protected("rbac:read", http.HandlerFunc(api.roles)))
	mux.Handle("GET "+APIPrefix+"/admin/rbac/permissions", protected("rbac:read", http.HandlerFunc(api.permissions)))
	mux.Handle("GET "+APIPrefix+"/admin/rbac/users/{userId}/scopes", protected("rbac:read", http.HandlerFunc(api.userScopes)))
	mux.Handle("POST "+APIPrefix+"/admin/rbac/roles", protected("rbac:manage", http.HandlerFunc(api.createRole)))
	mux.Handle("PATCH "+APIPrefix+"/admin/rbac/roles/{id}", protected("rbac:manage", http.HandlerFunc(api.updateRole)))
	mux.Handle("DELETE "+APIPrefix+"/admin/rbac/roles/{id}", protected("rbac:manage", http.HandlerFunc(api.deleteRole)))
	mux.Handle("POST "+APIPrefix+"/admin/rbac/permissions", protected("rbac:manage", http.HandlerFunc(api.createPermission)))
	mux.Handle("DELETE "+APIPrefix+"/admin/rbac/permissions/{id}", protected("rbac:manage", http.HandlerFunc(api.deletePermission)))
	mux.Handle("POST "+APIPrefix+"/admin/rbac/roles/{roleId}/permissions", protected("rbac:manage", http.HandlerFunc(api.assignPermission)))
	mux.Handle("DELETE "+APIPrefix+"/admin/rbac/roles/{roleId}/permissions/{permissionId}", protected("rbac:manage", http.HandlerFunc(api.revokePermission)))
	mux.Handle("POST "+APIPrefix+"/admin/rbac/users/{userId}/scopes", protected("rbac:manage", http.HandlerFunc(api.createScope)))
	mux.Handle("DELETE "+APIPrefix+"/admin/rbac/scopes/{id}", protected("rbac:manage", http.HandlerFunc(api.deleteScope)))
	mux.Handle("GET "+APIPrefix+"/admin/rbac/users/{userId}/delegasi", protected("rbac:read", http.HandlerFunc(api.listDelegasi)))
	mux.Handle("POST "+APIPrefix+"/admin/rbac/users/{userId}/delegasi", protected("rbac:manage", http.HandlerFunc(api.createDelegasi)))
	mux.Handle("DELETE "+APIPrefix+"/admin/rbac/delegasi/{id}", protected("rbac:manage", http.HandlerFunc(api.revokeDelegasi)))
}

type rbacAdminAPI struct {
	reader RBACReader
}

func (api rbacAdminAPI) roles(w http.ResponseWriter, r *http.Request) {
	items, err := api.reader.ListRoles(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	out := make([]RBACRoleDTO, 0, len(items))
	for _, it := range items {
		out = append(out, RBACRoleDTO{ID: it.ID, Code: it.Code, Name: it.Name, TierLevel: it.TierLevel})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (api rbacAdminAPI) permissions(w http.ResponseWriter, r *http.Request) {
	items, err := api.reader.ListPermissions(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	out := make([]RBACPermissionDTO, 0, len(items))
	for _, it := range items {
		out = append(out, RBACPermissionDTO{ID: it.ID, Module: it.Module, Action: it.Action})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (api rbacAdminAPI) userScopes(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userId")
	items, err := api.reader.ListUserRoleScopes(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	out := make([]RBACUserRoleScopeDTO, 0, len(items))
	for _, it := range items {
		out = append(out, RBACUserRoleScopeDTO{
			ID:                it.ID,
			UserID:            it.UserID,
			RoleID:            it.RoleID,
			BiroCode:          it.BiroCode,
			FakultasID:        it.FakultasID,
			JurusanID:         it.JurusanID,
			UnitID:            it.UnitID,
			DeniedPermissions: it.DeniedPermissions,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (api rbacAdminAPI) createRole(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		TierLevel int    `json:"tierLevel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	role, err := api.reader.CreateRole(r.Context(), models.RbacRole{Code: body.Code, Name: body.Name, TierLevel: body.TierLevel})
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusCreated, RBACRoleDTO{ID: role.ID, Code: role.Code, Name: role.Name, TierLevel: role.TierLevel})
}

func (api rbacAdminAPI) updateRole(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	var body struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		TierLevel int    `json:"tierLevel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	role, err := api.reader.UpdateRole(r.Context(), uint(id), models.RbacRole{Code: body.Code, Name: body.Name, TierLevel: body.TierLevel})
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, RBACRoleDTO{ID: role.ID, Code: role.Code, Name: role.Name, TierLevel: role.TierLevel})
}

func (api rbacAdminAPI) deleteRole(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	if err := api.reader.DeleteRole(r.Context(), uint(id)); err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (api rbacAdminAPI) createPermission(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Module string `json:"module"`
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	perm, err := api.reader.CreatePermission(r.Context(), models.RbacPermission{Module: body.Module, Action: body.Action})
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusCreated, RBACPermissionDTO{ID: perm.ID, Module: perm.Module, Action: perm.Action})
}

func (api rbacAdminAPI) deletePermission(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	if err := api.reader.DeletePermission(r.Context(), uint(id)); err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (api rbacAdminAPI) assignPermission(w http.ResponseWriter, r *http.Request) {
	roleID, err := strconv.ParseUint(r.PathValue("roleId"), 10, 64)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	var body struct {
		PermissionID uint `json:"permissionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	if err := api.reader.AssignPermissionToRole(r.Context(), uint(roleID), body.PermissionID); err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"assigned": true})
}

func (api rbacAdminAPI) revokePermission(w http.ResponseWriter, r *http.Request) {
	roleID, err := strconv.ParseUint(r.PathValue("roleId"), 10, 64)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	permID, err := strconv.ParseUint(r.PathValue("permissionId"), 10, 64)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	if err := api.reader.RevokePermissionFromRole(r.Context(), uint(roleID), uint(permID)); err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true})
}

func (api rbacAdminAPI) createScope(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userId")
	var body struct {
		RoleID            uint   `json:"roleId"`
		BiroCode          string `json:"biroCode"`
		FakultasID        string `json:"fakultasId"`
		JurusanID         string `json:"jurusanId"`
		UnitID            string `json:"unitId"`
		DeniedPermissions []byte `json:"deniedPermissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	scope := models.RbacUserRoleScope{UserID: userID, RoleID: body.RoleID, DeniedPermissions: body.DeniedPermissions}
	if body.BiroCode != "" {
		v := body.BiroCode
		scope.BiroCode = &v
	}
	if body.FakultasID != "" {
		v := body.FakultasID
		scope.FakultasID = &v
	}
	if body.JurusanID != "" {
		v := body.JurusanID
		scope.JurusanID = &v
	}
	if body.UnitID != "" {
		v := body.UnitID
		scope.UnitID = &v
	}
	created, err := api.reader.CreateUserRoleScope(r.Context(), scope)
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusCreated, RBACUserRoleScopeDTO{
		ID:                created.ID,
		UserID:            created.UserID,
		RoleID:            created.RoleID,
		BiroCode:          created.BiroCode,
		FakultasID:        created.FakultasID,
		JurusanID:         created.JurusanID,
		UnitID:            created.UnitID,
		DeniedPermissions: created.DeniedPermissions,
	})
}

func (api rbacAdminAPI) deleteScope(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	if err := api.reader.DeleteUserRoleScope(r.Context(), uint(id)); err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (api rbacAdminAPI) listDelegasi(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userId")
	items, err := api.reader.ListDelegasi(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (api rbacAdminAPI) createDelegasi(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userId")
	var body struct {
		PemberiUserId string    `json:"pemberiUserId"`
		Cakupan       string    `json:"cakupan"`
		Mulai         time.Time `json:"mulai"`
		Selesai       time.Time `json:"selesai"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	if body.PemberiUserId == "" {
		body.PemberiUserId = userID
	}
	d := models.Delegasi{
		PemberiUserId:  body.PemberiUserId,
		PenerimaUserId: userID,
		Cakupan:        models.CakupanDelegasi(body.Cakupan),
		Mulai:          body.Mulai,
		Selesai:        body.Selesai,
		Aktif:          true,
	}
	created, err := api.reader.CreateDelegasi(r.Context(), d)
	if err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (api rbacAdminAPI) revokeDelegasi(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidQuery)
		return
	}
	if err := api.reader.RevokeDelegasi(r.Context(), uint(id)); err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true})
}
