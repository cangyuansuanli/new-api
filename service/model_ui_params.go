package service

import (
	"encoding/json"
	"strings"
)

func ParseJSONStringArray(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func MustJSONString(value interface{}, fallback string) string {
	if value == nil {
		return fallback
	}
	b, err := json.Marshal(value)
	if err != nil {
		return fallback
	}
	return string(b)
}
