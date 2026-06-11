package ui

import (
	"fmt"
	"os"
	"strings"
)

func ParseTags(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag == "" {
			continue
		}
		tags = append(tags, tag)
	}
	return tags
}

func ParseMetadataLines(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	out := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid metadata line %q; expected key=value", line)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("invalid metadata line %q; key is required", line)
		}
		out[key] = value
	}
	return out, nil
}

func ValidateBankID(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("bank_id is required")
	}
	if strings.ContainsAny(value, " \t\n\r") {
		return fmt.Errorf("bank_id must not contain whitespace")
	}
	return nil
}

func WritePrivateText(path string, content []byte) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path is required")
	}
	return os.WriteFile(path, content, 0o600)
}
