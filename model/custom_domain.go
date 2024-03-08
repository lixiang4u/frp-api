package model

import (
	"encoding/json"
	"os"
)

func LoadCustomDomainMap() (m map[string]string) {
	m = make(map[string]string)
	buf, err := os.ReadFile("custom-domain.json")
	if err != nil {
		return m
	}
	if err = json.Unmarshal(buf, &m); err != nil {
		return m
	}
	return m
}
