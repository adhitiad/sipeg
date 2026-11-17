//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

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

