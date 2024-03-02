package utils

import (
	"github.com/go-jose/go-jose/v3/json"
	"strings"
)

func ToJsonString(v any) string {
	buff, _ := json.MarshalIndent(v, "", "\t")
	return string(buff)
}

func NewHostName(str ...string) string {
	return HashString(strings.Join(str, ","))[:6]
}
