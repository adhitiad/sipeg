package httpserver

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/directory/domain"
	"simpeg/backend/internal/models"
)

const employeeImportRequestLimit = employeeImportFileLimit + 64*1024

type employeeImportIssue struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (api directoryAPI) employeeImportTemplate(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="employee-import-template.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, strings.Join(employeeImportHeaders, ",")+"\r\n")
}

func (api directoryAPI) importEmployees(w http.ResponseWriter, r *http.Request) {
	filename, contents, status := readEmployeeImportFile(w, r)
	if status != 0 {
		writeErrorResponse(w, status, employeeImportErrorCode(status))
		return
	}
	rows, issues := parseEmployeeImport(filename, contents)
	if len(issues) != 0 {
		writeEmployeeImportIssues(w, http.StatusUnprocessableEntity, "validation_error", issues)
		return
	}

	employees := make([]employeeImportRow, 0, len(rows))
	users := make([]models.User, 0, len(rows))
	for _, row := range rows {
		employees = append(employees, row)
		users = append(users, row.user)
	}

	principal, _ := auth.PrincipalFromContext(r.Context())
	created, err := api.employees.Import(r.Context(), principal, users)
	if err != nil {
		var conflict *domain.EmployeeConflictError
		if errors.As(err, &conflict) && conflict.Index >= 0 && conflict.Index < len(employees) {
			issue := employeeImportIssue{Row: employees[conflict.Index].number, Field: conflict.Field, Code: "duplicate"}
			writeEmployeeImportIssues(w, http.StatusConflict, "identity_conflict", []employeeImportIssue{issue})
			return
		}
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int{"imported": len(created)})
}

func readEmployeeImportFile(w http.ResponseWriter, r *http.Request) (string, []byte, int) {
	r.Body = http.MaxBytesReader(w, r.Body, employeeImportRequestLimit)
	reader, err := r.MultipartReader()
	if err != nil {
		return "", nil, http.StatusBadRequest
	}
	part, err := reader.NextPart()
	if err != nil {
		if isRequestTooLarge(err) {
			return "", nil, http.StatusRequestEntityTooLarge
		}
		return "", nil, http.StatusBadRequest
	}
	if part.FormName() != "file" || part.FileName() == "" {
		_ = part.Close()
		return "", nil, http.StatusBadRequest
	}
	filename := filepath.Base(part.FileName())
	data, err := io.ReadAll(io.LimitReader(part, employeeImportFileLimit+1))
	_ = part.Close()
	if err != nil {
		if isRequestTooLarge(err) {
			return "", nil, http.StatusRequestEntityTooLarge
		}
		return "", nil, http.StatusBadRequest
	}
	if len(data) > employeeImportFileLimit {
		return "", nil, http.StatusRequestEntityTooLarge
	}
	if _, err := reader.NextPart(); err == nil {
		return "", nil, http.StatusBadRequest
	} else if !errors.Is(err, io.EOF) {
		if isRequestTooLarge(err) {
			return "", nil, http.StatusRequestEntityTooLarge
		}
		return "", nil, http.StatusBadRequest
	}
	if filename == "." || filename == string(filepath.Separator) {
		return "", nil, http.StatusBadRequest
	}
	return filename, data, 0
}

func isRequestTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

func employeeImportErrorCode(status int) string {
	if status == http.StatusRequestEntityTooLarge {
		return "file_too_large"
	}
	return "invalid_multipart"
}

func writeEmployeeImportIssues(w http.ResponseWriter, status int, code string, issues []employeeImportIssue) {
	for index := range issues {
		issues[index].Message = employeeImportIssueMessage(issues[index])
	}
	writeErrorWith(w, status, apierror.Payload{
		Code:    code,
		Message: "Berkas tidak diimpor; perbaiki baris yang ditandai lalu unggah kembali.",
		Rows:    issues,
	})
}

func employeeImportIssueMessage(issue employeeImportIssue) string {
	switch issue.Code {
	case "invalid_header":
		return "Gunakan header template dengan nama dan urutan kolom yang tepat."
	case "invalid_column_count":
		return "Baris harus memiliki tepat lima kolom sesuai template."
	case "invalid_file":
		return "Berkas tidak valid atau memuat konten yang tidak didukung; gunakan template resmi."
	case "invalid_value":
		return "Nilai tidak boleh memuat karakter kontrol."
	case "formula_not_allowed":
		return "Rumus tidak diizinkan; masukkan nilai teks biasa."
	case "invalid_email":
		return "Masukkan alamat email yang valid."
	case "required_or_too_long":
		return "Nama wajib diisi dan maksimal 200 karakter."
	case "too_long":
		if issue.Field == "nip" {
			return "NIP maksimal 50 karakter."
		}
		return "Jabatan maksimal 200 karakter."
	case "duplicate_in_file":
		return "Email atau NIP ini muncul lebih dari sekali dalam berkas."
	case "too_many_rows":
		return "Berkas melebihi batas 1000 baris data."
	case "empty_file":
		return "Tambahkan setidaknya satu baris pegawai."
	case "duplicate":
		return "Email atau identitas pegawai sudah digunakan."
	default:
		return "Periksa nilai pada kolom ini."
	}
}
