// internal/logger/sanitise.go
package logger

import "strings"

// SensitiveKeys lists argument keys whose values are always redacted
var SensitiveKeys = map[string]bool{
	"password":      true,
	"token":         true,
	"api_key":       true,
	"secret":        true,
	"authorization": true,
	"ssn":           true,
	"national_id":   true,
	"bank_account":  true,
}

// Arguments returns a sanitised copy of the arguments map safe for DEBUG logging.
func Arguments(args map[string]any) map[string]any {
	safe := make(map[string]any, len(args))
	for k, v := range args {
		if SensitiveKeys[strings.ToLower(k)] {
			safe[k] = "[REDACTED]"
		} else {
			safe[k] = v
		}
	}
	return safe
}

// ArgKeys returns only the keys of the arguments map — safe for INFO logging.
func ArgKeys(args map[string]any) []string {
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	return keys
}

// Body sanitises an HTTP body string for DEBUG logging.
func Body(s string) string {
	if len(s) > 500 {
		return s[:500] + "... [truncated]"
	}
	return s
}
