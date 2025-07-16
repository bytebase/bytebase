package base

import (
	"fmt"
	"regexp"
	"strings"
)

// TransactionMode represents the transaction execution mode for a migration script.
type TransactionMode string

const (
	// TransactionModeOn wraps the script in a single transaction.
	TransactionModeOn TransactionMode = "on"
	// TransactionModeOff executes the script's statements sequentially in auto-commit mode.
	TransactionModeOff TransactionMode = "off"
	// TransactionModeUnspecified means no explicit mode was specified.
	TransactionModeUnspecified TransactionMode = ""
)

// txnModeDirectiveRegex matches the transaction mode directive at the beginning of a SQL script.
// Format: -- txn_mode = on|off
var txnModeDirectiveRegex = regexp.MustCompile(`(?i)^\s*--\s*txn_mode\s*=\s*(on|off)\s*$`)

// ParseTransactionMode extracts the transaction mode directive from the SQL script.
// It checks the first line of the script for the -- txn_mode = on|off directive.
// Returns the transaction mode and the SQL script without the directive.
func ParseTransactionMode(script string) (TransactionMode, string) {
	lines := strings.Split(script, "\n")
	if len(lines) == 0 {
		return TransactionModeUnspecified, script
	}

	// Check the first line for the transaction mode directive
	matches := txnModeDirectiveRegex.FindStringSubmatch(lines[0])
	if len(matches) == 2 {
		mode := strings.ToLower(matches[1])
		// Remove the directive line from the script
		remainingScript := strings.Join(lines[1:], "\n")
		return TransactionMode(mode), strings.TrimSpace(remainingScript)
	}

	return TransactionModeUnspecified, script
}

// IsValid returns true if the transaction mode is valid.
func (m TransactionMode) IsValid() bool {
	return m == TransactionModeOn || m == TransactionModeOff || m == TransactionModeUnspecified
}

// String returns the string representation of the transaction mode.
func (m TransactionMode) String() string {
	return string(m)
}

// PrependTransactionMode prepends the transaction mode directive to a SQL script.
// If the mode is unspecified, the script is returned unchanged.
func PrependTransactionMode(script string, mode TransactionMode) string {
	if mode == TransactionModeUnspecified || !mode.IsValid() {
		return script
	}
	directive := fmt.Sprintf("-- txn_mode = %s", mode)
	if script == "" {
		return directive
	}
	return fmt.Sprintf("%s\n%s", directive, script)
}