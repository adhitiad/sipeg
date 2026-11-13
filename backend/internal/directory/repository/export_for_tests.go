package repository

import "simpeg/backend/internal/directory/domain"

// Exports untuk pengujian di backend/tests/directory/repository.
func ClassifyRecordError(err error) error {
	return classifyRecordError(err)
}

func EscapeLike(value string) string {
	return escapeLike(value)
}

func EmployeeColumns(patch domain.EmployeePatch) (map[string]any, error) {
	return employeeColumns(patch)
}

func OrganizationColumns(patch domain.OrganizationPatch) map[string]any {
	return organizationColumns(patch)
}
