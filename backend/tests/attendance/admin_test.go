package attendance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"simpeg/backend/internal/audit"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"

	. "simpeg/backend/internal/attendance"
)

type fakeAdminStore struct {
	pins            []models.MesinAbsensiPin
	users           map[string]models.User
	mapped          []MappedEmployee
	events          map[string]map[string]int64
	cuti            map[string][]CutiWindow
	libur           map[string]bool
	presensi        map[string]models.Presensi
	alpaRows        []MonthlyAlpa
	createdCount    int
	updatedCount    int
	inserted        []models.Presensi
	federatedAudits int
}

func presensiKey(userID string, day time.Time) string {
	return userID + "|" + day.Format("2006-01-02")
}

func (r *fakeAdminStore) ListPins(_ context.Context, scopeIDs []string) ([]PinView, error) {
	views := []PinView{}
	for _, pin := range r.pins {
		user, ok := r.users[pin.UserId]
		if !ok {
			continue
		}
		if scopeIDs != nil && !inScopeTest(scopeIDs, user.HomebaseId) {
			continue
		}
		views = append(views, PinView{ID: pin.ID, MachinePin: pin.MachinePin, UserID: pin.UserId, Nama: user.Nama, Aktif: pin.Aktif})
	}
	return views, nil
}

func inScopeTest(scopeIDs []string, homebaseID *string) bool {
	if scopeIDs == nil {
		return true
	}
	if homebaseID == nil {
		return false
	}
	for _, id := range scopeIDs {
		if id == *homebaseID {
			return true
		}
	}
	return false
}

func (r *fakeAdminStore) CreatePin(_ context.Context, machinePin, userID string, catatan *string, _ string) (models.MesinAbsensiPin, error) {
	for _, pin := range r.pins {
		if pin.MachinePin == machinePin || pin.UserId == userID {
			return models.MesinAbsensiPin{}, ErrPinDuplicate
		}
	}
	pin := models.MesinAbsensiPin{MachinePin: machinePin, UserId: userID, Aktif: true, Catatan: catatan}
	pin.ID = fmt.Sprintf("pin-%d", len(r.pins)+1)
	r.pins = append(r.pins, pin)
	return pin, nil
}

func (r *fakeAdminStore) DeletePin(_ context.Context, pinID string, scopeIDs []string, _ string) error {
	for index, pin := range r.pins {
		if pin.ID != pinID {
			continue
		}
		user, ok := r.users[pin.UserId]
		if !ok || !inScopeTest(scopeIDs, user.HomebaseId) {
			return ErrPinNotFound
		}
		r.pins = append(r.pins[:index], r.pins[index+1:]...)
		return nil
	}
	return ErrPinNotFound
}

func (r *fakeAdminStore) UserIdentityOf(_ context.Context, userID string) (models.User, error) {
	user, ok := r.users[userID]
	if !ok {
		return models.User{}, fmt.Errorf("pegawai tidak ditemukan")
	}
	return user, nil
}

func (r *fakeAdminStore) UserByEmail(_ context.Context, email string) (models.User, error) {
	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}
	return models.User{}, fmt.Errorf("pegawai tidak ditemukan")
}

func (r *fakeAdminStore) MappedEmployees(_ context.Context, scopeIDs []string) ([]MappedEmployee, error) {
	out := []MappedEmployee{}
	for _, employee := range r.mapped {
		user, ok := r.users[employee.UserID]
		if !ok || !inScopeTest(scopeIDs, user.HomebaseId) {
			continue
		}
		out = append(out, employee)
	}
	return out, nil
}

func (r *fakeAdminStore) EventsPerPinPerDay(_ context.Context, _, _ time.Time) (map[string]map[string]int64, error) {
	return r.events, nil
}

func (r *fakeAdminStore) ApprovedCutiPerUser(_ context.Context, _ []string, _, _ time.Time) (map[string][]CutiWindow, error) {
	return r.cuti, nil
}

func (r *fakeAdminStore) HariLiburDates(_ context.Context, _, _ time.Time) (map[string]bool, error) {
	return r.libur, nil
}

func (r *fakeAdminStore) PresensiExists(_ context.Context, userID string, tanggal time.Time) (string, bool, error) {
	row, ok := r.presensi[presensiKey(userID, tanggal)]
	if !ok {
		return "", false, nil
	}
	if row.Sumber == nil {
		return "", true, nil
	}
	return *row.Sumber, true, nil
}

func (r *fakeAdminStore) InsertPresensi(_ context.Context, row models.Presensi) error {
	r.presensi[presensiKey(row.UserId, row.Tanggal)] = row
	r.createdCount++
	r.inserted = append(r.inserted, row)
	return nil
}

func (r *fakeAdminStore) UpdatePresensiDerived(_ context.Context, row models.Presensi, _ string) error {
	key := presensiKey(row.UserId, row.Tanggal)
	existing, ok := r.presensi[key]
	if !ok {
		return fmt.Errorf("baris tidak ada")
	}
	if existing.Sumber == nil || *existing.Sumber != "derivasi_mesin" {
		return fmt.Errorf("baris presensi tidak lagi milik derivasi mesin; rangkaian dihentikan")
	}
	existing.Status = row.Status
	existing.Keterangan = row.Keterangan
	r.presensi[key] = existing
	r.updatedCount++
	return nil
}

func (r *fakeAdminStore) UpsertPresensiManual(_ context.Context, row models.Presensi, _ string) error {
	r.presensi[presensiKey(row.UserId, row.Tanggal)] = row
	return nil
}

func (r *fakeAdminStore) CountAlpaPerMonth(_ context.Context, _ []string, _, _ time.Time) ([]MonthlyAlpa, error) {
	return r.alpaRows, nil
}

func (r *fakeAdminStore) EmployeesInScope(_ context.Context, scopeIDs []string, ids []string) ([]models.User, error) {
	out := []models.User{}
	for _, id := range ids {
		user, ok := r.users[id]
		if ok && inScopeTest(scopeIDs, user.HomebaseId) {
			out = append(out, user)
		}
	}
	return out, nil
}

func (r *fakeAdminStore) WriteAudit(_ context.Context, _ audit.Entry) error {
	r.federatedAudits++
	return nil
}

func adminPolicy() auth.StaticPolicy {
	return auth.StaticPolicy{
		"admin":    {"attendance:presensi:manage": true, "attendance:read:any": true},
		"operator": {"attendance:presensi:manage": true},
		"pegawai":  {},
	}
}

func ptrString(value string) *string { return &value }

func seededDeriveStore() *fakeAdminStore {
	homeA := "unit-a"
	homeB := "unit-b"
	ptra := func(v string) *string { return &v }
	store := &fakeAdminStore{
		users: map[string]models.User{
			"user-a1": {IDModel: models.IDModel{ID: "user-a1"}, Email: "a1@example.test", Nama: "Pegawai A1", HomebaseId: ptra(homeA)},
			"user-a2": {IDModel: models.IDModel{ID: "user-a2"}, Email: "a2@example.test", Nama: "Pegawai A2", HomebaseId: ptra(homeA)},
			"user-b1": {IDModel: models.IDModel{ID: "user-b1"}, Email: "b1@example.test", Nama: "Pegawai B1", HomebaseId: ptra(homeB)},
		},
		mapped: []MappedEmployee{
			{UserID: "user-a1", Nama: "Pegawai A1", MachinePin: "pin-a1"},
			{UserID: "user-a2", Nama: "Pegawai A2", MachinePin: "pin-a2"},
			{UserID: "user-b1", Nama: "Pegawai B1", MachinePin: "pin-b1"},
		},
		events: map[string]map[string]int64{},
		cuti:   map[string][]CutiWindow{},
		libur:  map[string]bool{"2026-06-05": true},
		presensi: map[string]models.Presensi{
			"user-a2|2026-06-02": {UserId: "user-a2", Tanggal: time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC), Status: models.StatusPresensiHadir, Sumber: ptrString("manual")},
		},
	}
	day := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 7; i++ {
		d := day.AddDate(0, 0, i)
		store.events[d.Format("2006-01-02")] = map[string]int64{"pin-a1": 2}
	}
	store.cuti["user-b1"] = []CutiWindow{
		{Start: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC), Status: models.StatusPresensiCuti},
	}
	return store
}

func TestDeriveStatuses(t *testing.T) {
	store := seededDeriveStore()
	service := NewAdminService(store, adminPolicy(), nil)
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	summary, err := service.Derive(context.Background(), auth.Principal{UserID: "admin-1", RoleName: "admin"}, from, to)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}

	row := func(userID string, day int) models.Presensi {
		d := time.Date(2026, 6, day, 0, 0, 0, 0, time.UTC)
		presensi, ok := store.presensi[presensiKey(userID, d)]
		if !ok {
			t.Fatalf("presensi %s %s tidak ditulis", userID, d.Format("2006-01-02"))
		}
		return presensi
	}

	for day := 1; day <= 7; day++ {
		if got := row("user-a1", day).Status; got != models.StatusPresensiHadir {
			t.Fatalf("a1 tanggal %d: status %s, want H", day, got)
		}
	}

	// a2: tanpa tap; 2026-06-05 libur nasional -> LD; Minggu -> LD; sisanya A.
	// Baris manual 2026-06-02 dipertahankan -> H, bukan A.
	if got := row("user-a2", 2).Status; got != models.StatusPresensiHadir {
		t.Fatalf("a2 2026-06-02: status %s, want H (baris manual dipertahankan)", got)
	}
	if got := row("user-a2", 5).Status; got != models.StatusPresensiLibur {
		t.Fatalf("a2 2026-06-05: status %s, want LD (libur nasional)", got)
	}
	for _, day := range []int{1, 3, 4, 6} {
		if isSunday(time.Date(2026, 6, day, 0, 0, 0, 0, time.UTC)) {
			continue
		}
		if got := row("user-a2", day).Status; got != models.StatusPresensiAlpa {
			t.Fatalf("a2 tanggal %d: status %s, want A", day, got)
		}
	}

	// b1: cuti 01..03 -> C; sisanya mengikuti aturan LD/A.
	for day := 1; day <= 3; day++ {
		if got := row("user-b1", day).Status; got != models.StatusPresensiCuti {
			t.Fatalf("b1 tanggal %d: status %s, want C", day, got)
		}
	}
	if got := row("user-b1", 5).Status; got != models.StatusPresensiLibur {
		t.Fatalf("b1 2026-06-05: status %s, want LD", got)
	}
	if summary.SkippedManual != 1 {
		t.Fatalf("skippedManual = %d, want 1", summary.SkippedManual)
	}
	if summary.ByStatus[models.StatusPresensiHadir] != 7 {
		t.Fatalf("jumlah H = %d, want 7", summary.ByStatus[models.StatusPresensiHadir])
	}
	if summary.AuditFailed {
		t.Fatal("audit gagal ditulis; seharusnya berhasil")
	}
	if summary.Created != 20 || summary.Updated != 0 {
		t.Fatalf("created=%d updated=%d, want 20/0", summary.Created, summary.Updated)
	}
}

func isSunday(day time.Time) bool {
	return day.Weekday() == time.Sunday
}

func TestDeriveOutOfRangeRejected(t *testing.T) {
	store := seededDeriveStore()
	service := NewAdminService(store, adminPolicy(), nil)
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 120)
	if _, err := service.Derive(context.Background(), auth.Principal{UserID: "admin-1", RoleName: "admin"}, from, to); err == nil {
		t.Fatal("rentang lebih dari 93 hari harus ditolak")
	}
}

func TestAlpaSuspectsFlagsConsecutiveMonths(t *testing.T) {
	store := seededDeriveStore()
	store.alpaRows = []MonthlyAlpa{
		{UserID: "user-a1", Month: "2026-06", Count: 2},
		{UserID: "user-a1", Month: "2026-07", Count: 3},
		{UserID: "user-a2", Month: "2026-06", Count: 1},
	}
	service := NewAdminService(store, adminPolicy(), nil)
	suspects, err := service.AlpaSuspects(context.Background(), auth.Principal{UserID: "admin-1", RoleName: "admin"}, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), 2)
	if err != nil {
		t.Fatalf("suspects: %v", err)
	}
	if len(suspects) != 1 {
		t.Fatalf("jumlah suspects = %d, want 1 (%+v)", len(suspects), suspects)
	}
	if suspects[0].UserID != "user-a1" {
		t.Fatalf("suspect = %s, want user-a1", suspects[0].UserID)
	}
	if suspects[0].TotalAlpa != 5 {
		t.Fatalf("totalAlpa = %d, want 5", suspects[0].TotalAlpa)
	}
}

func TestAdminServiceForbidsPegawai(t *testing.T) {
	store := seededDeriveStore()
	service := NewAdminService(store, adminPolicy(), nil)
	_, err := service.ListPins(context.Background(), auth.Principal{UserID: "user-1", RoleName: "pegawai"})
	if err == nil {
		t.Fatal("pegawai tidak boleh mengelola presensi administratif")
	}
	pin, err := service.CreatePin(context.Background(), auth.Principal{UserID: "pegawai-1", RoleName: "pegawai"}, "pin-x", "user-a1", nil)
	if err == nil {
		t.Fatalf("pegawai tidak boleh membuat pemetaan PIN (dibuat: %+v)", pin)
	}
}
