package ui

import "strings"

var sensitiveEnvParts = [...]string{"KEY", "TOKEN", "SECRET", "PASSWORD", "ACCESS"}

func RedactEnvValue(key, value string) string {
	upperKey := strings.ToUpper(key)
	for _, part := range sensitiveEnvParts {
		if strings.Contains(upperKey, part) {
			if value == "" {
				return "unset"
			}
			return "set"
		}
	}

	if value == "" {
		return "unset"
	}
	return value
}
