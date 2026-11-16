package httpserver

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"simpeg/backend/internal/apierror"
	"simpeg/backend/internal/attendance"
	"simpeg/backend/internal/auth"
	"simpeg/backend/internal/models"
)

// attendanceAdminAPI memuat use case administratif presensi (pemetaan PIN mesin,
// pengisian manual, derivasi, dan laporan alpa).
type attendanceAdminAPI struct {
	service *attendance.AdminService
}

type pinCreateRequest struct {
	MachinePin string  `json:"machinePin"`
	UserID     string  `json:"userId"`
	Catatan    *string `json:"catatan"`
}

type presensiUpsertRequest struct {
	UserID     string  `json:"userId"`
	Tanggal    string  `json:"tanggal"`
	Status     string  `json:"status"`
	Keterangan *string `json:"keterangan"`
}

type derivePresensiRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

const maxPinImportRows = 2000
const maxPinImportBytes = 2 << 20

// RegisterAttendanceAdminRoutes mendaftarkan endpoint administratif presensi.
// Di luar cakupan pegawai (presensi tidak diekspos ke role pegawai).
func RegisterAttendanceAdminRoutes(mux *http.ServeMux, resolver auth.SessionResolver, policy auth.Policy, service *attendance.AdminService) {
	if service == nil {
		return
	}
	api := attendanceAdminAPI{service: service}
	protected := func(handler http.Handler) http.Handler {
		return auth.Authenticate(resolver)(auth.RequirePermission(policy, "attendance:presensi:manage", handler))
	}

	mux.Handle("GET "+APIPrefix+"/attendance/mesin-pin", protected(http.HandlerFunc(api.listPins)))
	mux.Handle("POST "+APIPrefix+"/attendance/mesin-pin", protected(http.HandlerFunc(api.createPin)))
	mux.Handle("DELETE "+APIPrefix+"/attendance/mesin-pin/{pinID}", protected(http.HandlerFunc(api.deletePin)))
	mux.Handle("GET "+APIPrefix+"/attendance/mesin-pin/import-template", protected(http.HandlerFunc(api.importTemplate)))
	mux.Handle("POST "+APIPrefix+"/attendance/mesin-pin/import", protected(http.HandlerFunc(api.importPins)))
	mux.Handle("PUT "+APIPrefix+"/attendance/presensi", protected(http.HandlerFunc(api.upsertPresensi)))
	mux.Handle("POST "+APIPrefix+"/attendance/presensi/derive", protected(http.HandlerFunc(api.derive)))
	mux.Handle("GET "+APIPrefix+"/attendance/alpa-suspects", protected(http.HandlerFunc(api.alpaSuspects)))
}

func (api attendanceAdminAPI) listPins(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	items, err := api.service.ListPins(r.Context(), principal)
	if err != nil {
		writeAttendanceAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (api attendanceAdminAPI) createPin(w http.ResponseWriter, r *http.Request) {
	var request pinCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	userID := strings.TrimSpace(request.UserID)
	if !validUUID(userID) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	view, err := api.service.CreatePin(r.Context(), principal, request.MachinePin, userID, request.Catatan)
	if err != nil {
		writeAttendanceAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (api attendanceAdminAPI) deletePin(w http.ResponseWriter, r *http.Request) {
	pinID := r.PathValue("pinID")
	if !validUUID(pinID) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	if err := api.service.DeletePin(r.Context(), principal, pinID); err != nil {
		writeAttendanceAdminError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api attendanceAdminAPI) importTemplate(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="mesin-pin-import-template.csv"`)
	_, _ = io.WriteString(w, "pin,email\r\n12345,pegawai@example.test\r\n")
}

func (api attendanceAdminAPI) importPins(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxPinImportBytes+64*1024)
	reader, err := r.MultipartReader()
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeBadRequest)
		return
	}
	part, err := reader.NextPart()
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeBadRequest)
		return
	}
	if part.FormName() != "file" || part.FileName() == "" {
		_ = part.Close()
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeBadRequest)
		return
	}
	contents, ok := readPinImportPart(w, part)
	if !ok {
		return
	}
	extension := strings.ToLower(filepath.Ext(part.FileName()))
	rows, issues := parsePinImport(extension, contents)
	if issues != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":  "file tidak dapat dibaca",
			"issues": issues,
		})
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	imported, importIssues, err := api.service.ImportPins(r.Context(), principal, rows)
	if err != nil {
		writeAttendanceAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"imported": imported,
		"issues":   importIssues,
	})
}

func readPinImportPart(w http.ResponseWriter, part *multipart.Part) ([]byte, bool) {
	contents, err := io.ReadAll(part)
	_ = part.Close()
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeBadRequest)
		return nil, false
	}
	return contents, true
}

func parsePinImport(extension string, contents []byte) ([]attendance.ImportPinRow, []attendanceAdminIssue) {
	switch extension {
	case ".csv":
		return parsePinImportCSV(contents)
	case ".xlsx":
		return parsePinImportXLSX(contents)
	default:
		return nil, []attendanceAdminIssue{{Row: 0, Field: "file", Code: "unsupported", Message: "ekstensi harus .csv atau .xlsx"}}
	}
}

type attendanceAdminIssue = attendance.ImportPinIssue

func parsePinImportCSV(contents []byte) ([]attendance.ImportPinRow, []attendanceAdminIssue) {
	text := strings.TrimPrefix(string(contents), "\ufeff")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	delimiter := ','
	if firstLine := firstLineOf(text); firstLine != "" && strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		delimiter = ';'
	}
	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, []attendanceAdminIssue{{Row: 0, Field: "file", Code: "invalid", Message: "CSV tidak dapat diurai"}}
	}
	return pinImportRows(records)
}

func parsePinImportXLSX(contents []byte) ([]attendance.ImportPinRow, []attendanceAdminIssue) {
	file, err := excelize.OpenReader(bytes.NewReader(contents))
	if err != nil {
		return nil, []attendanceAdminIssue{{Row: 0, Field: "file", Code: "invalid", Message: "XLSX tidak dapat dibuka"}}
	}
	defer func() { _ = file.Close() }()
	sheets := file.GetSheetList()
	if len(sheets) != 1 {
		return nil, []attendanceAdminIssue{{Row: 0, Field: "file", Code: "invalid", Message: "workbook harus memiliki satu sheet"}}
	}
	records, err := file.GetRows(sheets[0])
	if err != nil || len(records) > maxPinImportRows+1 {
		return nil, []attendanceAdminIssue{{Row: 0, Field: "file", Code: "too_large", Message: fmt.Sprintf("maksimal %d baris data", maxPinImportRows)}}
	}
	return pinImportRows(records)
}

func pinImportRows(records [][]string) ([]attendance.ImportPinRow, []attendanceAdminIssue) {
	if len(records) == 0 {
		return nil, []attendanceAdminIssue{{Row: 0, Field: "file", Code: "empty", Message: "file kosong"}}
	}
	header := map[string]int{}
	for index, cell := range records[0] {
		header[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(cell, "\ufeff")))] = index
	}
	pinCol, pinOK := header["pin"]
	if !pinOK {
		pinCol, pinOK = header["machinepin"]
	}
	emailCol, emailOK := header["email"]
	if !pinOK || !emailOK {
		return nil, []attendanceAdminIssue{{Row: 1, Field: "header", Code: "required", Message: "kolom wajib: pin, email"}}
	}
	rows := []attendance.ImportPinRow{}
	for index, record := range records[1:] {
		rowNumber := index + 2
		if len(record) == 0 || (strings.TrimSpace(cellAt(record, pinCol)) == "" && strings.TrimSpace(cellAt(record, emailCol)) == "") {
			continue
		}
		if len(rows) >= maxPinImportRows {
			return nil, []attendanceAdminIssue{{Row: rowNumber, Field: "file", Code: "too_large", Message: fmt.Sprintf("maksimal %d baris data", maxPinImportRows)}}
		}
		rows = append(rows, attendance.ImportPinRow{
			Row:        rowNumber,
			MachinePin: strings.TrimSpace(cellAt(record, pinCol)),
			Email:      strings.TrimSpace(cellAt(record, emailCol)),
		})
	}
	return rows, nil
}

func cellAt(record []string, index int) string {
	if index >= len(record) {
		return ""
	}
	return record[index]
}

func (api attendanceAdminAPI) upsertPresensi(w http.ResponseWriter, r *http.Request) {
	var request presensiUpsertRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	userID := strings.TrimSpace(request.UserID)
	if !validUUID(userID) {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidID)
		return
	}
	tanggal, err := time.Parse("2006-01-02", strings.TrimSpace(request.Tanggal))
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	row, created, err := api.service.UpsertPresensi(r.Context(), principal, attendance.UpsertPresensiInput{
		UserID:     userID,
		Tanggal:    tanggal,
		Status:     models.StatusPresensi(strings.TrimSpace(strings.ToUpper(request.Status))),
		Keterangan: request.Keterangan,
	})
	if err != nil {
		writeAttendanceAdminError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, map[string]any{
		"userId":     row.UserId,
		"tanggal":    row.Tanggal.Format("2006-01-02"),
		"status":     row.Status,
		"keterangan": row.Keterangan,
		"created":    created,
	})
}

func (api attendanceAdminAPI) derive(w http.ResponseWriter, r *http.Request) {
	var request derivePresensiRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, apierror.CodeInvalidJSON)
		return
	}
	from, err := time.Parse("2006-01-02", strings.TrimSpace(request.From))
	if err != nil {
		writeValidationError(w)
		return
	}
	to, err := time.Parse("2006-01-02", strings.TrimSpace(request.To))
	if err != nil {
		writeValidationError(w)
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	summary, err := api.service.Derive(r.Context(), principal, from, to)
	if err != nil {
		writeAttendanceAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (api attendanceAdminAPI) alpaSuspects(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	months := 2
	if raw := strings.TrimSpace(query.Get("months")); raw != "" {
		values, err := strconv.Atoi(raw)
		if err != nil {
			writeValidationError(w)
			return
		}
		months = values
	}
	var until time.Time
	if raw := strings.TrimSpace(query.Get("until")); raw != "" {
		parsed, err := time.Parse("2006-01", raw)
		if err != nil {
			writeValidationError(w)
			return
		}
		until = parsed
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	suspects, err := api.service.AlpaSuspects(r.Context(), principal, until, months)
	if err != nil {
		writeAttendanceAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"months": months,
		"items":  suspects,
	})
}

func writeAttendanceAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, attendance.ErrPresensiForbidden):
		writeErrorResponse(w, http.StatusForbidden, apierror.CodeForbidden)
	case errors.Is(err, attendance.ErrPinDuplicate):
		writeErrorResponse(w, http.StatusConflict, apierror.CodeConflict)
	case errors.Is(err, attendance.ErrPinNotFound):
		writeErrorResponse(w, http.StatusNotFound, apierror.CodeNotFound)
	case errors.Is(err, attendance.ErrAlpaSuspectRange):
		writeValidationError(w)
	default:
		writeErrorResponse(w, http.StatusServiceUnavailable, apierror.CodeServiceUnavailable)
	}
}

func firstLineOf(text string) string {
	if index := strings.IndexByte(text, '\n'); index >= 0 {
		return text[:index]
	}
	return text
}
