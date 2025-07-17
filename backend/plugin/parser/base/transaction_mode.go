package base

import (
	"regexp"
	"strings"

	"github.com/bytebase/bytebase/backend/common"
)

// TransactionMode is an alias for common.TransactionMode for backward compatibility.
type TransactionMode = common.TransactionMode

const (
	// TransactionModeOn wraps the script in a single transaction.
	TransactionModeOn = common.TransactionModeOn
	// TransactionModeOff executes the script's statements sequentially in auto-commit mode.
	TransactionModeOff = common.TransactionModeOff
	// TransactionModeUnspecified means no explicit mode was specified.
	TransactionModeUnspecified = common.TransactionModeUnspecified
)

// txnModeDirectiveRegex matches the transaction mode directive at the beginning of a SQL script.
// Format: -- txn-mode = on|off
var txnModeDirectiveRegex = regexp.MustCompile(`(?i)^\s*--\s*txn-mode\s*=\s*(on|off)\s*$`)

// ParseTransactionMode extracts the transaction mode directive from the SQL script.
// It checks the first line of the script for the -- txn-mode = on|off directive.
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
