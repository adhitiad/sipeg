// Package attendance memuat orkestrasi absensi: pemetaan machinePin mesin ke
// pegawai, penurunan status harian `Presensi` dari `AttendanceEvent`, pengisian
// manual oleh admin/operator, dan laporan pengawas alpa (SP1).
//
// Aturan penurunan status harian (DEC-076/DEC-077, setuju 2026-11-14):
// hari kerja = Senin–Sabtu; Minggu dan hari libur nasional (tabel HariLibur)
// membentuk hari libur ('LD'). Pegawai dengan ≥1 tap pada tanggal tsb → 'H';
// tanpa tap: cuti disetujui → 'C' (cuti sakit → 'S'), periode sakit → 'S',
// hari libur → 'LD', selain itu → 'A'. Baris `Presensi` yang diisi manual
// (sumber selain 'derivasi_mesin') tidak pernah ditimpa oleh derivasi.
package attendance

import (
	"context"
	"errors"
	"time"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

var (
	// ErrPresensiForbidden menandai aksi yang di luar cakupan principal.
	ErrPresensiForbidden = errors.New("attendance presensi action is not permitted")
	// ErrAlpaSuspectRange menandai rentang bulan yang tidak sah.
	ErrAlpaSuspectRange = errors.New("alpa suspect range is not valid")
)

// MappedEmployee adalah pegawai yang terpetakan ke sebuah machinePin.
type MappedEmployee struct {
	UserID     string
	Nama       string
	MachinePin string
}

// PinView adalah representasi publik pemetaan machinePin.
type PinView struct {
	ID         string  `json:"id"`
	MachinePin string  `json:"machinePin"`
	UserID     string  `json:"userId"`
	Nama       string  `json:"nama"`
	Aktif      bool    `json:"aktif"`
	Catatan    *string `json:"catatan"`
	CreatedAt  string  `json:"createdAt"`
}

// UpsertPresensiInput adalah input pengisian manual Presensi.
type UpsertPresensiInput struct {
	UserID     string
	Tanggal    time.Time
	Status     models.StatusPresensi
	Keterangan *string
}

// DeriveSummary melaporkan hasil satu rangkaian penurunan Presensi.
type DeriveSummary struct {
	From          string                          `json:"from"`
	To            string                          `json:"to"`
	Created       int64                           `json:"created"`
	Updated       int64                           `json:"updated"`
	SkippedManual int64                           `json:"skippedManual"`
	ByStatus      map[models.StatusPresensi]int64 `json:"byStatus"`
	// AuditFailed melaporkan kegagalan penulisan jejak audit (best-effort).
	AuditFailed bool `json:"auditFailed,omitempty"`
}

// AlpaSuspect adalah satu baris laporan alpa berulang (untuk diterbitkan SP manual).
type AlpaSuspect struct {
	UserID     string  `json:"userId"`
	Nama       string  `json:"nama"`
	HomebaseID *string `json:"homebaseId"`
	// Alpa memuat jumlah hari alpa per bulan (kunci "YYYY-MM").
	Alpa      map[string]int64 `json:"alpa"`
	TotalAlpa int64            `json:"totalAlpa"`
}

// AdminStore adalah port persistensi untuk area presensi administratif.
type AdminStore interface {
	ListPins(context.Context, []string) ([]PinView, error)
	// CreatePin menyimpan pemetaan dan menulis audit dalam satu transaksi.
	CreatePin(context.Context, string, string, *string, string) (models.MesinAbsensiPin, error)
	// DeletePin menghapus pemetaan dalam cakupan dan menulis audit.
	DeletePin(context.Context, string, []string, string) error
	UserIdentityOf(context.Context, string) (models.User, error)
	UserByEmail(context.Context, string) (models.User, error)
	MappedEmployees(context.Context, []string) ([]MappedEmployee, error)
	EventsPerPinPerDay(context.Context, time.Time, time.Time) (map[string]map[string]int64, error)
	ApprovedCutiPerUser(context.Context, []string, time.Time, time.Time) (map[string][]CutiWindow, error)
	HariLiburDates(context.Context, time.Time, time.Time) (map[string]bool, error)
	PresensiExists(context.Context, string, time.Time) (sumber string, found bool, err error)
	InsertPresensi(context.Context, models.Presensi) error
	UpdatePresensiDerived(context.Context, models.Presensi, string) error
	// UpsertPresensiManual menulis baris manual dan menulis audit.
	UpsertPresensiManual(context.Context, models.Presensi, string) error
	CountAlpaPerMonth(context.Context, []string, time.Time, time.Time) ([]MonthlyAlpa, error)
	EmployeesInScope(context.Context, []string, []string) ([]models.User, error)
	// WriteAudit menulis satu entri jejak audit.
	WriteAudit(context.Context, audit.Entry) error
}

// CutiWindow memuat rentang cuti yang disetujui dan jenis pemotongannya.
type CutiWindow struct {
	Start  time.Time
	End    time.Time
	Status models.StatusPresensi
}

// MonthlyAlpa adalah jumlah hari alpa satu pegawai pada satu bulan.
type MonthlyAlpa struct {
	UserID string
	Month  string
	Count  int64
}

// newMonthlyAlpa membuat satu baris jumlah alpa.
func newMonthlyAlpa(userID, month string, count int64) MonthlyAlpa {
	return MonthlyAlpa{UserID: userID, Month: month, Count: count}
}

// AdminService menjalankan use case administratif presensi.
type AdminService struct {
	store  AdminStore
	policy auth.Policy
	scope  UnitScopeReader
}

// NewAdminService membuat service presensi administratif.
func NewAdminService(store AdminStore, policy auth.Policy, scope UnitScopeReader) *AdminService {
	return &AdminService{store: store, policy: policy, scope: scope}
}

const maxDeriveWindowDays = 93

// scopeIDs mengembalikan daftar unit cakupan principal: nil = lintas organisasi
// (admin), slice kosong = tanpa cakupan, atau daftar unit + turunannya.
func (s *AdminService) scopeIDs(ctx context.Context, principal auth.Principal) ([]string, error) {
	if s.policy == nil || !s.policy.Allows(principal.RoleName, "attendance:presensi:manage") {
		return nil, ErrPresensiForbidden
	}
	if s.policy.Allows(principal.RoleName, "attendance:read:any") {
		return nil, nil
	}
	if principal.HomebaseID == "" || s.scope == nil {
		return []string{}, nil
	}
	return s.scope.ListUnitScopeIDs(ctx, principal.HomebaseID)
}

func inScope(scopeIDs []string, homebaseID *string) bool {
	if scopeIDs == nil {
		return true
	}
	if len(scopeIDs) == 0 || homebaseID == nil {
		return false
	}
	for _, id := range scopeIDs {
		if id == *homebaseID {
			return true
		}
	}
	return false
}
