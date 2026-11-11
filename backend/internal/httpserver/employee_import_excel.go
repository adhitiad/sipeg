package httpserver

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/richardlehane/mscfb"
	"github.com/shakinm/xlsReader/xls"
	"github.com/xuri/excelize/v2"
)

const employeeImportExpandedLimit = 10 << 20

func parseXLSXRecords(contents []byte) (records [][]string, err error) {
	defer recoverSpreadsheetPanic(&err)
	if err = inspectXLSXArchive(contents); err != nil {
		return nil, err
	}
	file, err := excelize.OpenReader(bytes.NewReader(contents), excelize.Options{
		UnzipSizeLimit: employeeImportExpandedLimit, UnzipXMLSizeLimit: employeeImportExpandedLimit,
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	sheets := file.GetSheetList()
	if len(sheets) != 1 {
		return nil, errors.New("workbook must have one sheet")
	}
	visible, err := file.GetSheetVisible(sheets[0])
	if err != nil || !visible {
		return nil, errors.New("hidden sheet is not allowed")
	}
	records, err = file.GetRows(sheets[0])
	if err != nil || len(records) > employeeImportRowLimit+1 {
		return nil, errors.New("invalid worksheet")
	}
	if err = padEmployeeImportRows(records); err != nil {
		return nil, err
	}
	for rowIndex, row := range records {
		for columnIndex, value := range row {
			if value == "" {
				continue
			}
			cell, _ := excelize.CoordinatesToCellName(columnIndex+1, rowIndex+1)
			cellType, cellErr := file.GetCellType(sheets[0], cell)
			if cellErr != nil || (cellType != excelize.CellTypeInlineString && cellType != excelize.CellTypeSharedString) {
				return nil, errors.New("cells must contain text")
			}
		}
	}
	return records, nil
}

func inspectXLSXArchive(contents []byte) error {
	if len(contents) < 4 || !bytes.Equal(contents[:2], []byte{'P', 'K'}) {
		return errors.New("invalid xlsx signature")
	}
	reader, err := zip.NewReader(bytes.NewReader(contents), int64(len(contents)))
	if err != nil || len(reader.File) > 128 {
		return errors.New("invalid xlsx archive")
	}
	seen := make(map[string]bool, len(reader.File))
	var expanded uint64
	var workbook, worksheet *zip.File
	for _, entry := range reader.File {
		name := entry.Name
		lower := strings.ToLower(name)
		if seen[lower] || strings.Contains(name, "\\") || hasUnsafeZipPath(name) {
			return errors.New("unsafe archive entry")
		}
		seen[lower] = true
		if entry.UncompressedSize64 > employeeImportExpandedLimit ||
			expanded > employeeImportExpandedLimit-entry.UncompressedSize64 {
			return errors.New("xlsx expanded size exceeded")
		}
		expanded += entry.UncompressedSize64
		if strings.Contains(lower, "vba") || strings.HasPrefix(lower, "xl/externallinks/") ||
			lower == "xl/connections.xml" {
			return errors.New("macros or external connections are not allowed")
		}
		if lower == "xl/workbook.xml" {
			workbook = entry
		}
		if strings.HasPrefix(lower, "xl/worksheets/") && strings.HasSuffix(lower, ".xml") {
			if worksheet != nil {
				return errors.New("multiple worksheets are not allowed")
			}
			worksheet = entry
		}
	}
	if workbook == nil || worksheet == nil {
		return errors.New("missing workbook or worksheet")
	}
	if err := inspectWorkbookXML(workbook); err != nil {
		return err
	}
	return inspectWorksheetXML(worksheet)
}

func inspectWorkbookXML(entry *zip.File) error {
	content, err := readZipEntry(entry)
	if err != nil {
		return err
	}
	decoder := xml.NewDecoder(bytes.NewReader(content))
	sheetCount := 0
	for {
		token, tokenErr := decoder.Token()
		if errors.Is(tokenErr, io.EOF) {
			break
		}
		if tokenErr != nil {
			return errors.New("invalid workbook xml")
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local == "definedName" {
			return errors.New("defined formulas are not allowed")
		}
		if start.Name.Local == "sheet" {
			sheetCount++
			for _, attr := range start.Attr {
				if attr.Name.Local == "state" && attr.Value != "visible" {
					return errors.New("hidden sheet is not allowed")
				}
			}
		}
	}
	if sheetCount != 1 {
		return errors.New("workbook must have one sheet")
	}
	return nil
}

func inspectWorksheetXML(entry *zip.File) error {
	content, err := readZipEntry(entry)
	if err != nil {
		return err
	}
	decoder := xml.NewDecoder(bytes.NewReader(content))
	for {
		token, tokenErr := decoder.Token()
		if errors.Is(tokenErr, io.EOF) {
			return nil
		}
		if tokenErr != nil {
			return errors.New("invalid worksheet xml")
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local == "f" {
			return errors.New("formulas are not allowed")
		}
		if start.Name.Local == "dimension" {
			for _, attr := range start.Attr {
				if attr.Name.Local == "ref" {
					end := strings.Split(attr.Value, ":")
					column, row, parseErr := cellReference(end[len(end)-1])
					if parseErr != nil || column > len(employeeImportHeaders) || row > employeeImportRowLimit+1 {
						return errors.New("worksheet dimension limit exceeded")
					}
				}
			}
		}
		if start.Name.Local == "row" {
			for _, attr := range start.Attr {
				if attr.Name.Local == "r" {
					row, parseErr := strconv.Atoi(attr.Value)
					if parseErr != nil || row < 1 || row > employeeImportRowLimit+1 {
						return errors.New("worksheet row limit exceeded")
					}
				}
			}
		}
		if start.Name.Local == "c" {
			for _, attr := range start.Attr {
				if attr.Name.Local == "r" {
					column, row, parseErr := cellReference(attr.Value)
					if parseErr != nil || column > len(employeeImportHeaders) || row > employeeImportRowLimit+1 {
						return errors.New("worksheet cell limit exceeded")
					}
				}
			}
		}
	}
}

func cellReference(reference string) (int, int, error) {
	column, row := 0, 0
	index := 0
	for index < len(reference) && reference[index] >= 'A' && reference[index] <= 'Z' {
		column = column*26 + int(reference[index]-'A'+1)
		index++
	}
	if index == 0 || index == len(reference) {
		return 0, 0, errors.New("invalid cell reference")
	}
	row, err := strconv.Atoi(reference[index:])
	if err != nil || row < 1 {
		return 0, 0, errors.New("invalid cell reference")
	}
	return column, row, nil
}

func readZipEntry(entry *zip.File) ([]byte, error) {
	reader, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	content, err := io.ReadAll(io.LimitReader(reader, employeeImportExpandedLimit+1))
	if err != nil || len(content) > employeeImportExpandedLimit {
		return nil, errors.New("expanded entry limit exceeded")
	}
	return content, nil
}

func hasUnsafeZipPath(name string) bool {
	for _, part := range strings.Split(name, "/") {
		if part == ".." || part == "." {
			return true
		}
	}
	return strings.HasPrefix(name, "/")
}

func parseLegacyXLSRecords(contents []byte) (records [][]string, err error) {
	defer recoverSpreadsheetPanic(&err)
	if len(contents) < 8 || !bytes.Equal(contents[:8], []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}) {
		return nil, errors.New("invalid xls signature")
	}
	reader, err := mscfb.New(bytes.NewReader(contents))
	if err != nil {
		return nil, err
	}
	var workbookStream []byte
	rootFound := false
	for _, entry := range reader.File {
		path := strings.ToLower(strings.Join(append(entry.Path, entry.Name), "/"))
		if strings.Contains(path, "vba") || strings.Contains(path, "macro") ||
			strings.Contains(path, "project") {
			return nil, errors.New("macros are not allowed")
		}
		if entry.Name == "Root Entry" {
			rootFound = true
		}
		if entry.Name != "Workbook" && entry.Name != "Book" {
			continue
		}
		if workbookStream != nil || entry.Size <= 0 || entry.Size > employeeImportFileLimit {
			return nil, errors.New("invalid workbook stream")
		}
		workbookStream = make([]byte, int(entry.Size))
		read, readErr := entry.ReadAt(workbookStream, 0)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return nil, readErr
		}
		if read != len(workbookStream) {
			return nil, errors.New("truncated workbook stream")
		}
	}
	if !rootFound || len(workbookStream) == 0 {
		return nil, errors.New("missing workbook stream")
	}
	if err = rejectBIFFFormulas(workbookStream); err != nil {
		return nil, err
	}
	workbook, err := xls.OpenReader(bytes.NewReader(contents))
	if err != nil || workbook.GetNumberSheets() != 1 {
		return nil, errors.New("invalid xls workbook")
	}
	sheet, err := workbook.GetSheet(0)
	if err != nil || sheet.GetNumberRows() > employeeImportRowLimit+1 {
		return nil, errors.New("worksheet row limit exceeded")
	}
	for rowIndex := 0; rowIndex < sheet.GetNumberRows(); rowIndex++ {
		row, rowErr := sheet.GetRow(rowIndex)
		if rowErr != nil {
			return nil, errors.New("invalid worksheet row")
		}
		cells := row.GetCols()
		if len(cells) > len(employeeImportHeaders) {
			return nil, errors.New("unexpected worksheet columns")
		}
		values := make([]string, len(cells))
		for column, cell := range cells {
			cellType := cell.GetType()
			if strings.Contains(cellType, "FakeBlank") {
				continue
			}
			if !strings.Contains(cellType, "LabelSSt") && !strings.Contains(cellType, "LabelBIFF") {
				return nil, errors.New("non-text cell")
			}
			values[column] = cell.GetString()
		}
		records = append(records, values)
	}
	if err = padEmployeeImportRows(records); err != nil {
		return nil, err
	}
	return records, nil
}

func padEmployeeImportRows(records [][]string) error {
	for index := range records {
		if len(records[index]) > len(employeeImportHeaders) {
			return errors.New("unexpected worksheet columns")
		}
		for len(records[index]) < len(employeeImportHeaders) {
			records[index] = append(records[index], "")
		}
	}
	return nil
}

func rejectBIFFFormulas(stream []byte) error {
	var sheetOffsets []int
	_, err := scanBIFFRecords(stream, 0, true, &sheetOffsets)
	if err != nil || len(sheetOffsets) == 0 {
		return errors.New("invalid xls records")
	}
	for _, offset := range sheetOffsets {
		if _, err := scanBIFFRecords(stream, offset, false, nil); err != nil {
			return err
		}
	}
	return nil
}

func scanBIFFRecords(stream []byte, start int, globals bool, sheetOffsets *[]int) (int, error) {
	for offset := start; offset+4 <= len(stream); {
		recordID := binary.LittleEndian.Uint16(stream[offset : offset+2])
		recordLength := int(binary.LittleEndian.Uint16(stream[offset+2 : offset+4]))
		end := offset + 4 + recordLength
		if end > len(stream) {
			return 0, errors.New("invalid xls record length")
		}
		if recordID == 0x0006 || recordID == 0x0221 || recordID == 0x04bc || recordID == 0x0236 {
			return 0, errors.New("formulas are not allowed")
		}
		if globals && recordID == 0x0085 && recordLength >= 4 {
			*sheetOffsets = append(*sheetOffsets, int(binary.LittleEndian.Uint32(stream[offset+4:offset+8])))
		}
		if recordID == 0x000a {
			return end, nil
		}
		offset = end
	}
	return 0, errors.New("unterminated xls records")
}

func recoverSpreadsheetPanic(err *error) {
	if recovered := recover(); recovered != nil {
		*err = errors.New("invalid spreadsheet")
	}
}
