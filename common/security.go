package common

import "strings"

// EscapeForLogging escapes characters for logging to prevent injection.
func EscapeForLogging(input string) string {
	escaped := strings.ReplaceAll(input, "\n", "")
	escaped = strings.ReplaceAll(escaped, "\t", "")
	return escaped
}
