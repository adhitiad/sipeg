package domain

import "strings"

var roleNames = map[string]bool{
	"admin": true, "operator": true, "atasan": true, "pegawai": true,
}

func NormalizeRoleName(value string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(value))
	if !roleNames[name] {
		return "", ErrValidation
	}
	return name, nil
}
