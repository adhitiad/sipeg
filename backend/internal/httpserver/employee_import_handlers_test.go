package httpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
)

const employeeImportCSV = "nama,email,nip,jabatanStruktural,jabatanFungsional\n" +
	"Pegawai Uji,pegawai@example.test,NIP-001,,Dosen\n"

func TestEmployeeImportEndpointsRequirePermission(t *testing.T) {
	mux := testDirectoryMux(&fakeDirectoryStore{}, nil)
	for _, endpoint := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: APIPrefix + "/employees/import-template.csv"},
		{method: http.MethodPost, path: APIPrefix + "/employees/import"},
	} {
		for _, authorization := range []string{"", "Bearer valid-session"} {
			request := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			if authorization != "" {
				request.Header.Set("Authorization", authorization)
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			expected := http.StatusUnauthorized
			if authorization != "" {
				expected = http.StatusForbidden
			}
			if response.Code != expected {
				t.Errorf("%s %s authorization %q: got %d, want %d", endpoint.method, endpoint.path, authorization, response.Code, expected)
			}
		}
	}
}

func TestEmployeeImportTemplateAndCSVUpload(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := testImportMux(store)
	templateRequest := httptest.NewRequest(http.MethodGet, APIPrefix+"/employees/import-template.csv", nil)
	templateRequest.Header.Set("Authorization", "Bearer valid-session")
	templateResponse := httptest.NewRecorder()
	mux.ServeHTTP(templateResponse, templateRequest)
	if templateResponse.Code != http.StatusOK ||
		!strings.Contains(templateResponse.Header().Get("Content-Type"), "text/csv") ||
		templateResponse.Body.String() != strings.Join(employeeImportHeaders, ",")+"\r\n" {
		t.Fatalf("template response: %d %q", templateResponse.Code, templateResponse.Body.String())
	}

	request := employeeImportRequest(t, "employees.csv", employeeImportCSV)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || store.batchCalls != 1 || len(store.employees) != 1 {
		t.Fatalf("upload status=%d body=%s batches=%d employees=%d", response.Code, response.Body, store.batchCalls, len(store.employees))
	}
	if store.actorID != "user-1" || store.employees[0].Email != "pegawai@example.test" ||
		store.employees[0].RoleId != nil || store.employees[0].Nik != nil || store.employees[0].BankRekening != nil {
		t.Fatalf("unexpected imported employee: %#v", store.employees[0])
	}
	var result map[string]int
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result["imported"] != 1 {
		t.Fatalf("response = %s", response.Body)
	}
}

func TestEmployeeImportValidatesAllRowsBeforeWriting(t *testing.T) {
	store := &fakeDirectoryStore{}
	mux := testImportMux(store)
	input := "nama,email,nip,jabatanStruktural,jabatanFungsional\n" +
		"Pegawai Satu,one@example.test,NIP-1,,\n" +
		"Pegawai Dua,bukan-email,NIP-2,,\n"
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, employeeImportRequest(t, "employees.csv", input))
	if response.Code != http.StatusUnprocessableEntity || store.batchCalls != 0 {
		t.Fatalf("status=%d batches=%d body=%s", response.Code, store.batchCalls, response.Body)
	}
	if !strings.Contains(response.Body.String(), `"row":3`) ||
		!strings.Contains(response.Body.String(), `"field":"email"`) {
		t.Fatalf("missing actionable row error: %s", response.Body)
	}
}

func TestEmployeeImportRejectsDuplicateAndInvalidHeaders(t *testing.T) {
	tests := []struct {
		name string
		csv  string
	}{
		{
			name: "duplicate email",
			csv: "nama,email,nip,jabatanStruktural,jabatanFungsional\n" +
				"Pegawai Satu,pegawai@example.test,,,\n" +
				"Pegawai Dua,PEGAWAI@example.test,,,\n",
		},
		{
			name: "unexpected role column",
			csv: "nama,email,nip,jabatanStruktural,jabatanFungsional,roleId\n" +
				"Pegawai Uji,pegawai@example.test,,,,admin\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &fakeDirectoryStore{}
			response := httptest.NewRecorder()
			testImportMux(store).ServeHTTP(response, employeeImportRequest(t, "employees.csv", test.csv))
			if response.Code != http.StatusUnprocessableEntity || store.batchCalls != 0 {
				t.Fatalf("status=%d batches=%d body=%s", response.Code, store.batchCalls, response.Body)
			}
		})
	}
}

func TestEmployeeImportMapsDatabaseConflictToRow(t *testing.T) {
	store := &fakeDirectoryStore{err: &domain.EmployeeConflictError{Index: 0, Field: "email"}}
	response := httptest.NewRecorder()
	testImportMux(store).ServeHTTP(response, employeeImportRequest(t, "employees.csv", employeeImportCSV))
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"row":2`) ||
		!strings.Contains(response.Body.String(), `"code":"duplicate"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body)
	}
}

func TestEmployeeImportRejectsOversizedFile(t *testing.T) {
	store := &fakeDirectoryStore{}
	data := bytes.Repeat([]byte("x"), employeeImportFileLimit+1)
	response := httptest.NewRecorder()
	testImportMux(store).ServeHTTP(response, employeeImportRequest(t, "employees.csv", string(data)))
	if response.Code != http.StatusRequestEntityTooLarge || store.batchCalls != 0 {
		t.Fatalf("status=%d batches=%d body=%s", response.Code, store.batchCalls, response.Body)
	}
}

func TestEmployeeImportStoreFailureDoesNotReportSuccess(t *testing.T) {
	store := &fakeDirectoryStore{err: errors.New("database detail")}
	response := httptest.NewRecorder()
	testImportMux(store).ServeHTTP(response, employeeImportRequest(t, "employees.csv", employeeImportCSV))
	if response.Code != http.StatusServiceUnavailable || strings.Contains(response.Body.String(), "database detail") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body)
	}
}

func testImportMux(store *fakeDirectoryStore) *http.ServeMux {
	return testDirectoryMux(store, map[string]map[auth.Permission]bool{
		"operator": {"employee:import": true},
	})
}

func employeeImportRequest(t *testing.T, filename, content string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, APIPrefix+"/employees/import", &body)
	request.Header.Set("Authorization", "Bearer valid-session")
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
