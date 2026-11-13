package httpserver

import "simpeg/backend/internal/auth"

// Exports untuk pengujian httpserver.
var (
	ParseEmployeeImport        = parseEmployeeImport
	AttendanceIngestPermission = attendanceIngestPermission
	ParseListQuery             = parseListQuery
	AuditLogDTOFromModel       = auditLogDTO
)

type EmployeeImportIssue = employeeImportIssue
type AttendanceEventResponse = attendanceEventResponse

const (
	DeviceIDHeader          = deviceIDHeader
	EmployeeImportFileLimit = employeeImportFileLimit
)

var EmployeeImportHeaders = employeeImportHeaders

// helper to satisfy compiler
var _ auth.Permission

var (
	InspectXLSXArchive            = inspectXLSXArchive
	ParseLegacyXLSRecords         = parseLegacyXLSRecords
	ValidateEmployeeImportRecords = validateEmployeeImportRecords
	RejectBIFFFormulas            = rejectBIFFFormulas
)
