// Package app memuat logika aplikasi (use case) untuk area penggajian.
//
// Lapisan ini memeriksa permission principal, membatasi slip gaji pada cakupan
// unitnya, menyusun pesan notifikasi gaji, lalu meneruskan pencatatan
// pembayaran ke port repository.
package app

import (
	"context"
	"fmt"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/listing"
	"simpeg/backend/internal/models"
	"simpeg/backend/internal/notify"
	"simpeg/backend/internal/payroll/domain"
	"simpeg/backend/internal/payroll/repository"
)

// ScopeRepository adalah port resolusi cakupan unit pada pohon organisasi.
type ScopeRepository interface {
	ListUnitScopeIDs(context.Context, string) ([]string, error)
}

// Service menjalankan use case penggajian: daftar slip, baca slip, dan
// pencatatan konfirmasi pembayaran.
type Service struct {
	repo   repository.Repository
	policy auth.Policy
	scope  scopeResolver
}

// NewService membuat service penggajian.
func NewService(repo repository.Repository, policy auth.Policy, scopeRepo ScopeRepository) *Service {
	return &Service{repo: repo, policy: policy, scope: newScopeResolver(scopeRepo)}
}

// List mengembalikan daftar slip gaji sesuai cakupan principal.
func (s *Service) List(ctx context.Context, principal auth.Principal, options listing.Options) (listing.Page[models.Gaji], error) {
	switch {
	case allows(s.policy, principal, "payroll:read:any"):
		// Role global: seluruh slip visible, penyaring unit tetap opsional.
	case allows(s.policy, principal, "payroll:read:homebase"):
		scope, err := s.scope.resolve(ctx, principal)
		if err != nil {
			return listing.Page[models.Gaji]{}, err
		}
		if scope.Empty() {
			return listing.EmptyPage[models.Gaji](options.Page, options.PageSize), nil
		}
		options.ScopeHomebaseIDs = scope.List()
	case allows(s.policy, principal, "payroll:read:self"):
		options.UserID = principal.UserID
	default:
		return listing.Page[models.Gaji]{}, domain.ErrForbidden
	}

	slips, total, err := s.repo.ListSlips(ctx, options)
	if err != nil {
		return listing.Page[models.Gaji]{}, err
	}
	return listing.Page[models.Gaji]{
		Items:      slips,
		Page:       options.Page,
		PageSize:   options.PageSize,
		TotalItems: total,
		TotalPages: listing.PageCount(total, options.PageSize),
	}, nil
}

// Get mengembalikan satu slip gaji bila principal berhak membacanya.
func (s *Service) Get(ctx context.Context, principal auth.Principal, id string) (models.Gaji, error) {
	slip, err := s.repo.GetSlip(ctx, id)
	if err != nil {
		return models.Gaji{}, err
	}
	permitted, err := s.canRead(ctx, principal, slip)
	if err != nil {
		return models.Gaji{}, err
	}
	if !permitted {
		return models.Gaji{}, domain.ErrForbidden
	}
	return slip, nil
}

// RecordPayment mencatat konfirmasi pembayaran untuk sebuah slip.
//
// -slip hanya dapat dibayar satu kali dan hanya bila statusnya sudah TERBIT.
// Notifikasi gaji diantrekan dalam transaksi yang sama sehingga email tidak
// pernah terkirim tanpa bukti pembayaran yang tersimpan.
func (s *Service) RecordPayment(ctx context.Context, principal auth.Principal, slipID string, input domain.PaymentInput) (models.Gaji, error) {
	if err := input.Validate(); err != nil {
		return models.Gaji{}, err
	}
	if !allows(s.policy, principal, "payroll:manage") {
		return models.Gaji{}, domain.ErrForbidden
	}

	slip, err := s.repo.GetSlip(ctx, slipID)
	if err != nil {
		return models.Gaji{}, err
	}
	if !domain.CanMarkPaid(slip.Status) {
		return models.Gaji{}, domain.ErrConflict
	}
	if slip.Pegawai == nil || slip.Pegawai.Email == "" {
		return models.Gaji{}, domain.ErrValidation
	}

	period, err := domain.ParsePeriod(slip.Periode.Format("2006-01"))
	if err != nil {
		return models.Gaji{}, err
	}
	payload, err := notify.SalaryPaidPayload(slip.Pegawai.Email, slip.Pegawai.Nama, period.Key(), slip.UserId)
	if err != nil {
		return models.Gaji{}, fmt.Errorf("build salary notice: %w", err)
	}

	notification := domain.Notification{
		Kind:      notify.KindSalaryPaid,
		DedupeKey: "salary-paid:" + period.Key() + ":" + slip.UserId,
		Payload:   string(payload),
	}
	updated, err := s.repo.RecordPayment(ctx, principal.UserID, slipID, input, notification)
	if err != nil {
		return models.Gaji{}, err
	}

	// Data pegawai dipertahankan dari pembacaan sebelumnya supaya slip hasil
	// pencatatan tetap memuat nama pemilik slip.
	slip.Status = updated.Status
	slip.TanggalDibayar = updated.TanggalDibayar
	slip.KonfirmasiPembayaranAt = updated.KonfirmasiPembayaranAt
	slip.KonfirmasiOlehId = updated.KonfirmasiOlehId
	return slip, nil
}

// ConfirmedPayment melaporkan apakah pembayaran untuk pegawai dan periode
// tertentu sudah terkonfirmasi. Sumber ini dipakai pengirim email agar
// notifikasi gaji hanya terkirim setelah pembayaran tercatat.
func (s *Service) ConfirmedPayment(ctx context.Context, userID, period string) (bool, error) {
	return s.repo.ConfirmedPayment(ctx, userID, period)
}

// canRead memeriksa apakah principal boleh membaca sebuah slip gaji.
func (s *Service) canRead(ctx context.Context, principal auth.Principal, slip models.Gaji) (bool, error) {
	if allows(s.policy, principal, "payroll:read:any") {
		return true, nil
	}
	if allows(s.policy, principal, "payroll:read:self") {
		return principal.UserID == slip.UserId, nil
	}
	if !allows(s.policy, principal, "payroll:read:homebase") {
		return false, nil
	}
	scope, err := s.scope.resolve(ctx, principal)
	if err != nil {
		return false, err
	}
	if scope.Empty() {
		return false, nil
	}
	if slip.Pegawai == nil || slip.Pegawai.HomebaseId == nil {
		return false, nil
	}
	return scope.Contains(*slip.Pegawai.HomebaseId), nil
}

// scopeResolver menyelesaikan cakupan unit seorang principal.
type scopeResolver struct {
	repo ScopeRepository
}

// newScopeResolver membuat resolver cakupan unit.
func newScopeResolver(repo ScopeRepository) scopeResolver {
	return scopeResolver{repo: repo}
}

// resolve mengembalikan cakupan unit principal: unit penugasannya beserta
// seluruh turunan pada pohon Homebase.
func (s scopeResolver) resolve(ctx context.Context, principal auth.Principal) (auth.UnitScope, error) {
	if principal.HomebaseID == "" || s.repo == nil {
		return auth.NewUnitScope("", nil), nil
	}
	unitIDs, err := s.repo.ListUnitScopeIDs(ctx, principal.HomebaseID)
	if err != nil {
		return auth.UnitScope{}, err
	}
	return auth.NewUnitScope(principal.HomebaseID, unitIDs), nil
}

// allows melaporkan apakah role principal memiliki permission tertentu.
func allows(policy auth.Policy, principal auth.Principal, permission auth.Permission) bool {
	return policy != nil && policy.Allows(principal.RoleName, permission)
}
