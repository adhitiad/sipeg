package app

import (
	"context"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/directory/repository"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
)

// EmployeeService menjalankan use case pegawai: daftar, baca, buat, ubah,
// arsip, dan impor massal.
//
// Setiap use case memeriksa permission principal lebih dulu, lalu membatasi
// data pada cakupan unit principal bila permission-nya berscope unit.
type EmployeeService struct {
	repo   repository.EmployeeRepository
	policy auth.Policy
	scope  scopeResolver
}

// NewEmployeeService membuat service pegawai.
func NewEmployeeService(repo repository.EmployeeRepository, policy auth.Policy, scopeRepo repository.ScopeRepository) *EmployeeService {
	return &EmployeeService{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo, policy)}
}

// List mengembalikan daftar pegawai sesuai filter yang sudah diminta.
//
// Filter cakupan unit ditambahkan di sini, bukan di lapisan transport,
// sehingga setiap jalur pembacaan pegawai melewati aturan yang sama.
func (s *EmployeeService) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.User], error) {
	switch {
	case allows(s.policy, principal, "employee:read:any"):
		// Role global: filter unit tetap opsional lewat query param.
	case allows(s.policy, principal, "employee:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.User]{}, err
		}
		if scope.Empty() {
			// Tanpa penugasan unit tidak ada data yang boleh terlihat.
			return listing.EmptyPage[models.User](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = scope.List()
	case allows(s.policy, principal, "employee:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.User]{}, domain.ErrForbidden
	}

	users, total, err := s.repo.ListEmployees(ctx, options)
	if err != nil {
		return listing.Page[models.User]{}, err
	}
	return listing.Page[models.User]{
		Items:      users,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// Get mengembalikan satu pegawai bila principal berhak membacanya.
func (s *EmployeeService) Get(ctx context.Context, principal auth.Principal, id string) (models.User, error) {
	needsScope := allows(s.policy, principal, "employee:read:homebase") && !allows(s.policy, principal, "employee:read:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.User{}, err
	}

	user, err := s.repo.GetEmployee(ctx, id)
	if err != nil {
		return models.User{}, err
	}
	if !s.canRead(scoped, user) {
		return models.User{}, domain.ErrForbidden
	}
	return user, nil
}

// Create membuat satu profil pegawai baru.
func (s *EmployeeService) Create(ctx context.Context, principal auth.Principal, input domain.CreateEmployee) (models.User, error) {
	if err := input.Validate(); err != nil {
		return models.User{}, err
	}

	needsScope := allows(s.policy, principal, "employee:create:homebase") && !allows(s.policy, principal, "employee:create")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.User{}, err
	}
	if !s.canCreate(scoped, input.HomebaseID) {
		return models.User{}, domain.ErrForbidden
	}

	user := models.User{
		Nama:              input.Nama,
		Email:             input.Email,
		EmailVerified:     false,
		Aktif:             true,
		Nip:               domain.CleanOptional(input.NIP),
		KategoriPegawaiId: input.KategoriPegawaiID,
		HomebaseId:        input.HomebaseID,
		JabatanStruktural: domain.CleanOptional(input.JabatanStruktural),
		JabatanFungsional: domain.CleanOptional(input.JabatanFungsional),
	}
	return s.repo.CreateEmployee(ctx, principal.UserID, user)
}

// Update memperbarui profil pegawai bila principal berhak mengubahnya.
func (s *EmployeeService) Update(ctx context.Context, principal auth.Principal, id string, patch domain.EmployeePatch) (models.User, error) {
	if !patch.HasChanges() {
		return models.User{}, domain.ErrValidation
	}
	if err := patch.Validate(); err != nil {
		return models.User{}, err
	}

	needsScope := allows(s.policy, principal, "employee:update:homebase") && !allows(s.policy, principal, "employee:update:any")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return models.User{}, err
	}

	user, err := s.repo.GetEmployee(ctx, id)
	if err != nil {
		return models.User{}, err
	}
	if !s.canWrite(scoped, user) {
		return models.User{}, domain.ErrForbidden
	}
	return s.repo.UpdateEmployee(ctx, principal.UserID, id, patch)
}

// Archive menonaktifkan pegawai bila principal berwenang.
func (s *EmployeeService) Archive(ctx context.Context, principal auth.Principal, id string) error {
	needsScope := allows(s.policy, principal, "employee:archive:homebase") && !allows(s.policy, principal, "employee:archive")
	scoped, err := s.scopedPrincipal(ctx, principal, needsScope)
	if err != nil {
		return err
	}

	user, err := s.repo.GetEmployee(ctx, id)
	if err != nil {
		return err
	}
	homebaseID := valueOrEmpty(user.HomebaseId)
	permitted := false
	for _, permission := range []auth.Permission{"employee:archive", "employee:archive:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, user.ID, homebaseID, permission, s.policy, scoped.scope) {
			permitted = true
			break
		}
	}
	if !permitted {
		return domain.ErrForbidden
	}
	return s.repo.ArchiveEmployee(ctx, principal.UserID, id)
}

// Import membuat banyak pegawai sekaligus.
//
// Pegawai hasil impor oleh role berscope unit ditempatkan pada unit penugasan
// principal. Template impor tidak memuat kolom unit.
func (s *EmployeeService) Import(ctx context.Context, principal auth.Principal, employees []models.User) ([]models.User, error) {
	if len(employees) == 0 {
		return nil, domain.ErrValidation
	}
	if err := s.assignImportHomebase(principal, employees); err != nil {
		return nil, err
	}
	return s.repo.CreateEmployees(ctx, principal.UserID, employees)
}

// assignImportHomebase menempatkan pegawai hasil impor pada unit principal bila
// import dijalankan oleh role dengan cakupan unit.
func (s *EmployeeService) assignImportHomebase(principal auth.Principal, employees []models.User) error {
	if allows(s.policy, principal, "employee:import") {
		return nil
	}
	if !allows(s.policy, principal, "employee:import:homebase") {
		return domain.ErrForbidden
	}
	if principal.HomebaseID == "" {
		return domain.ErrForbidden
	}
	homebaseID := principal.HomebaseID
	for index := range employees {
		employees[index].HomebaseId = &homebaseID
	}
	return nil
}

// scopedPrincipal menyusun principal beserta cakupan unitnya ketika
// principal memegang permission homebase. Cakupan hanya diresolusi bila
// diperlukan agar tidak ada query tambahan untuk role global.
func (s *EmployeeService) scopedPrincipal(ctx context.Context, principal auth.Principal, needsScope bool) (scopedPrincipal, error) {
	if !needsScope {
		return scopedPrincipal{principal: principal}, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return scopedPrincipal{}, err
	}
	return scopedPrincipal{principal: principal, scope: scope}, nil
}

// canRead memeriksa apakah principal boleh membaca satu record pegawai.
func (s *EmployeeService) canRead(scoped scopedPrincipal, user models.User) bool {
	homebaseID := valueOrEmpty(user.HomebaseId)
	for _, permission := range []auth.Permission{"employee:read:any", "employee:read:homebase", "employee:read:self"} {
		if auth.CanAccessUserInScope(scoped.principal, user.ID, homebaseID, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

// canCreate memeriksa apakah principal boleh membuat pegawai pada unit tertentu.
// Profil tanpa unit hanya boleh dibuat oleh role dengan cakupan global.
func (s *EmployeeService) canCreate(scoped scopedPrincipal, homebaseID *string) bool {
	if allows(s.policy, scoped.principal, "employee:create") {
		return true
	}
	if !allows(s.policy, scoped.principal, "employee:create:homebase") {
		return false
	}
	if homebaseID == nil {
		return false
	}
	return scoped.scope.Contains(*homebaseID)
}

// canWrite memeriksa apakah principal boleh mengubah satu record pegawai.
func (s *EmployeeService) canWrite(scoped scopedPrincipal, user models.User) bool {
	homebaseID := valueOrEmpty(user.HomebaseId)
	for _, permission := range []auth.Permission{"employee:update:any", "employee:update:homebase"} {
		if auth.CanAccessUserInScope(scoped.principal, user.ID, homebaseID, permission, s.policy, scoped.scope) {
			return true
		}
	}
	return false
}

// scopedPrincipal memuat principal beserta cakupan unit efektifnya.
type scopedPrincipal struct {
	principal auth.Principal
	scope     auth.UnitScope
}

// valueOrEmpty mengembalikan nilai string dari pointer, atau string kosong
// bila pointer kosong.
func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
