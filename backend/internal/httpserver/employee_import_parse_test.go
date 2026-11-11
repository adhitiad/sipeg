package httpserver

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"os"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseEmployeeImportCSV(t *testing.T) {
	input := "nama,email,nip,jabatanStruktural,jabatanFungsional\r\n" +
		"Pegawai Uji,pegawai@example.test,NIP-001,,Dosen\r\n"
	rows, issues := parseEmployeeImport("pegawai.csv", []byte(input))
	if len(issues) != 0 || len(rows) != 1 {
		t.Fatalf("rows=%#v issues=%#v", rows, issues)
	}
	if rows[0].number != 2 || rows[0].user.Email != "pegawai@example.test" ||
		rows[0].user.Nip == nil || *rows[0].user.Nip != "NIP-001" ||
		rows[0].user.JabatanStruktural != nil || rows[0].user.JabatanFungsional == nil {
		t.Fatalf("unexpected imported row: %#v", rows[0])
	}
	if rows[0].user.RoleId != nil || rows[0].user.HomebaseId != nil || !rows[0].user.Aktif {
		t.Fatalf("import assigned protected or default fields: %#v", rows[0].user)
	}
}

func TestParseEmployeeImportRejectsHeadersAndInvalidRows(t *testing.T) {
	tests := []struct {
		name  string
		input string
		field string
		code  string
	}{
		{
			name: "unexpected role column",
			input: "nama,email,nip,jabatanStruktural,jabatanFungsional,roleId\n" +
				"Pegawai Uji,pegawai@example.test,,,,admin\n",
			field: "header", code: "invalid_header",
		},
		{
			name: "invalid email",
			input: "nama,email,nip,jabatanStruktural,jabatanFungsional\n" +
				"Pegawai Uji,bukan-email,,,\n",
			field: "email", code: "invalid_email",
		},
		{
			name: "duplicate identity",
			input: "nama,email,nip,jabatanStruktural,jabatanFungsional\n" +
				"Pegawai Satu,pegawai@example.test,NIP-1,,\n" +
				"Pegawai Dua,PEGAWAI@example.test,NIP-2,,\n",
			field: "email", code: "duplicate_in_file",
		},
		{
			name: "duplicate NIP",
			input: "nama,email,nip,jabatanStruktural,jabatanFungsional\n" +
				"Pegawai Satu,one@example.test,NIP-1,,\n" +
				"Pegawai Dua,two@example.test,NIP-1,,\n",
			field: "nip", code: "duplicate_in_file",
		},
		{
			name: "formula-like CSV value",
			input: "nama,email,nip,jabatanStruktural,jabatanFungsional\n" +
				"=1+1,pegawai@example.test,,,\n",
			field: "nama", code: "formula_not_allowed",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rows, issues := parseEmployeeImport("pegawai.csv", []byte(test.input))
			if len(rows) != 0 || !hasImportIssue(issues, test.field, test.code) {
				t.Fatalf("rows=%#v issues=%#v", rows, issues)
			}
		})
	}
}

func TestParseEmployeeImportXLSX(t *testing.T) {
	data := makeXLSX(t, false, false)
	rows, issues := parseEmployeeImport("pegawai.xlsx", data)
	if len(issues) != 0 || len(rows) != 1 || rows[0].user.Email != "pegawai@example.test" {
		t.Fatalf("rows=%#v issues=%#v", rows, issues)
	}
}

func TestParseEmployeeImportRejectsXLSXFormulaAndMacro(t *testing.T) {
	formula, issues := parseEmployeeImport("pegawai.xlsx", makeXLSX(t, true, false))
	if len(formula) != 0 || !hasImportIssue(issues, "file", "invalid_file") {
		t.Fatalf("formula rows=%#v issues=%#v", formula, issues)
	}
	macroArchive := makeArchive(t, map[string]string{
		"xl/workbook.xml":            `<workbook><sheets><sheet name="Employees" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/worksheets/sheet1.xml":   `<worksheet><dimension ref="A1:E2"/><sheetData/></worksheet>`,
		"xl/vbaProject.bin":          "synthetic",
		"xl/_rels/workbook.xml.rels": `<Relationships/>`,
		"[Content_Types].xml":        `<Types/>`,
		"docProps/core.xml":          `<coreProperties/>`,
	})
	if err := inspectXLSXArchive(macroArchive); err == nil || !strings.Contains(err.Error(), "macros") {
		t.Fatalf("macro archive error = %v", err)
	}
}

func TestParseEmployeeImportXLS(t *testing.T) {
	data, err := os.ReadFile("testdata/employee_import_valid.xls")
	if err != nil {
		t.Fatal(err)
	}
	records, parseErr := parseLegacyXLSRecords(data)
	if parseErr != nil {
		t.Fatalf("parse legacy XLS: %v", parseErr)
	}
	rows, issues := validateEmployeeImportRecords(records)
	if len(issues) != 0 || len(rows) != 1 || rows[0].number != 2 ||
		rows[0].user.Email != "pegawai@example.test" || rows[0].user.Nip == nil ||
		*rows[0].user.Nip != "NIP-001" {
		t.Fatalf("rows=%#v issues=%#v", rows, issues)
	}
}

func TestLegacyXLSFormulaRecordIsRejected(t *testing.T) {
	stream := make([]byte, 20)
	binary.LittleEndian.PutUint16(stream[0:2], 0x0085)
	binary.LittleEndian.PutUint16(stream[2:4], 4)
	binary.LittleEndian.PutUint32(stream[4:8], 12)
	binary.LittleEndian.PutUint16(stream[8:10], 0x000a)
	binary.LittleEndian.PutUint16(stream[12:14], 0x0006)
	binary.LittleEndian.PutUint16(stream[16:18], 0x000a)
	if err := rejectBIFFFormulas(stream); err == nil || !strings.Contains(err.Error(), "formulas") {
		t.Fatalf("formula scan error = %v", err)
	}
}

func TestParseEmployeeImportRejectsUnsupportedAndMalformedFiles(t *testing.T) {
	for _, test := range []struct {
		name string
		data []byte
	}{
		{name: "unsupported extension", data: []byte("contents")},
		{name: "invalid xls signature", data: []byte("not an xls file")},
	} {
		t.Run(test.name, func(t *testing.T) {
			filename := "pegawai.bin"
			if test.name == "invalid xls signature" {
				filename = "pegawai.xls"
			}
			rows, issues := parseEmployeeImport(filename, test.data)
			if len(rows) != 0 || len(issues) == 0 {
				t.Fatalf("rows=%#v issues=%#v", rows, issues)
			}
		})
	}
}

func makeXLSX(t *testing.T, withFormula, withExtraHeader bool) []byte {
	t.Helper()
	file := excelize.NewFile()
	defer file.Close()
	sheet := file.GetSheetName(0)
	values := append([]string(nil), employeeImportHeaders...)
	if withExtraHeader {
		values = append(values, "roleId")
	}
	for column, value := range values {
		cell, err := excelize.CoordinatesToCellName(column+1, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := file.SetCellValue(sheet, cell, value); err != nil {
			t.Fatal(err)
		}
	}
	for column, value := range []string{"Pegawai Uji", "pegawai@example.test", "NIP-001", "", ""} {
		cell, err := excelize.CoordinatesToCellName(column+1, 2)
		if err != nil {
			t.Fatal(err)
		}
		if err := file.SetCellValue(sheet, cell, value); err != nil {
			t.Fatal(err)
		}
	}
	if withFormula {
		if err := file.SetCellFormula(sheet, "E2", `"Dosen"`); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	if err := file.Write(&output); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func makeArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func hasImportIssue(issues []employeeImportIssue, field, code string) bool {
	for _, issue := range issues {
		if issue.Field == field && issue.Code == code {
			return true
		}
	}
	return false
}
