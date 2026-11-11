package httpserver

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"simpeg/backend/internal/models"
)

const (
	employeeImportFileLimit = 5 << 20
	employeeImportRowLimit  = 1000
)

var employeeImportHeaders = []string{
	"nama", "email", "nip", "jabatanStruktural", "jabatanFungsional",
}

type employeeImportRow struct {
	number int
	user   models.User
}

func parseEmployeeImport(filename string, contents []byte) ([]employeeImportRow, []employeeImportIssue) {
	if len(contents) == 0 || len(contents) > employeeImportFileLimit {
		return nil, []employeeImportIssue{{Row: 1, Field: "file", Code: "invalid_file"}}
	}
	var records [][]string
	var err error
	switch strings.ToLower(filepathExtension(filename)) {
	case ".csv":
		records, err = parseCSVRecords(contents)
	case ".xls":
		records, err = parseLegacyXLSRecords(contents)
	case ".xlsx":
		records, err = parseXLSXRecords(contents)
	default:
		err = errors.New("unsupported format")
	}
	if err != nil {
		return nil, []employeeImportIssue{{Row: 1, Field: "file", Code: "invalid_file"}}
	}
	return validateEmployeeImportRecords(records)
}

func filepathExtension(filename string) string {
	index := strings.LastIndexByte(filename, '.')
	if index < 0 {
		return ""
	}
	return filename[index:]
}

func parseCSVRecords(contents []byte) ([][]string, error) {
	if len(contents) == 0 || !utf8.Valid(contents) || bytes.IndexByte(contents, 0) >= 0 {
		return nil, errors.New("invalid csv")
	}
	reader := csv.NewReader(bytes.NewReader(contents))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = false
	records := make([][]string, 0)
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errors.New("invalid csv")
		}
		if len(records) == 0 && len(record) > 0 {
			record[0] = strings.TrimPrefix(record[0], "\ufeff")
		}
		records = append(records, record)
		if len(records) > employeeImportRowLimit+1 {
			return records, nil
		}
	}
	return records, nil
}

func validateEmployeeImportRecords(records [][]string) ([]employeeImportRow, []employeeImportIssue) {
	if len(records) == 0 || !sameHeaders(records[0]) {
		return nil, []employeeImportIssue{{Row: 1, Field: "header", Code: "invalid_header"}}
	}
	rows := make([]employeeImportRow, 0, len(records)-1)
	issues := make([]employeeImportIssue, 0)
	emails := make(map[string]int)
	nips := make(map[string]int)
	nonEmptyRows := 0
	for index, values := range records[1:] {
		rowNumber := index + 2
		if allBlank(values) {
			continue
		}
		nonEmptyRows++
		if nonEmptyRows > employeeImportRowLimit {
			issues = append(issues, employeeImportIssue{Row: rowNumber, Field: "file", Code: "too_many_rows"})
			break
		}
		if len(values) != len(employeeImportHeaders) {
			issues = append(issues, employeeImportIssue{Row: rowNumber, Field: "row", Code: "invalid_column_count"})
			continue
		}
		normalized := make([]string, len(values))
		for column, value := range values {
			normalized[column] = strings.TrimSpace(value)
			if containsControlCharacter(value) {
				issues = append(issues, employeeImportIssue{
					Row: rowNumber, Field: employeeImportHeaders[column], Code: "invalid_value",
				})
			}
			if looksLikeFormula(value) {
				issues = append(issues, employeeImportIssue{
					Row: rowNumber, Field: employeeImportHeaders[column], Code: "formula_not_allowed",
				})
			}
		}
		user, rowIssues := employeeImportUserFromValues(rowNumber, normalized)
		issues = append(issues, rowIssues...)
		if user.Email != "" {
			if _, exists := emails[user.Email]; exists {
				issues = append(issues, employeeImportIssue{Row: rowNumber, Field: "email", Code: "duplicate_in_file"})
			} else {
				emails[user.Email] = rowNumber
			}
		}
		if user.Nip != nil {
			key := *user.Nip
			if _, exists := nips[key]; exists {
				issues = append(issues, employeeImportIssue{Row: rowNumber, Field: "nip", Code: "duplicate_in_file"})
			} else {
				nips[key] = rowNumber
			}
		}
		rows = append(rows, employeeImportRow{number: rowNumber, user: user})
	}
	if nonEmptyRows == 0 {
		issues = append(issues, employeeImportIssue{Row: 2, Field: "row", Code: "empty_file"})
	}
	if len(issues) != 0 {
		return nil, issues
	}
	return rows, nil
}

func sameHeaders(values []string) bool {
	if len(values) != len(employeeImportHeaders) {
		return false
	}
	for index, header := range employeeImportHeaders {
		if values[index] != header {
			return false
		}
	}
	return true
}

func employeeImportUserFromValues(row int, values []string) (models.User, []employeeImportIssue) {
	user := models.User{
		Nama: values[0], Email: strings.ToLower(values[1]), Aktif: true,
		EmailVerified: false,
	}
	if values[2] != "" {
		user.Nip = stringPointer(values[2])
	}
	if values[3] != "" {
		user.JabatanStruktural = stringPointer(values[3])
	}
	if values[4] != "" {
		user.JabatanFungsional = stringPointer(values[4])
	}
	issues := make([]employeeImportIssue, 0)
	if user.Nama == "" || textLength(user.Nama) > 200 {
		issues = append(issues, employeeImportIssue{Row: row, Field: "nama", Code: "required_or_too_long"})
	}
	if !validateEmail(user.Email) {
		issues = append(issues, employeeImportIssue{Row: row, Field: "email", Code: "invalid_email"})
	}
	if user.Nip != nil && textLength(*user.Nip) > 50 {
		issues = append(issues, employeeImportIssue{Row: row, Field: "nip", Code: "too_long"})
	}
	for _, field := range []struct {
		name  string
		value *string
	}{
		{name: "jabatanStruktural", value: user.JabatanStruktural},
		{name: "jabatanFungsional", value: user.JabatanFungsional},
	} {
		if field.value != nil && textLength(*field.value) > 200 {
			issues = append(issues, employeeImportIssue{Row: row, Field: field.name, Code: "too_long"})
		}
	}
	return user, issues
}

func allBlank(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func looksLikeFormula(value string) bool {
	trimmed := strings.TrimSpace(value)
	return len(trimmed) > 0 && strings.ContainsRune("=+-@", rune(trimmed[0]))
}

func containsControlCharacter(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}

func stringPointer(value string) *string { return &value }
