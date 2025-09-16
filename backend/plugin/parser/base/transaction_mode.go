package base

import (
	"database/sql"
	"regexp"
	"strings"

	"github.com/bytebase/bytebase/backend/common"
)

// Directive regex patterns
var (
	// txnModeDirectiveRegex matches the transaction mode directive.
	// Format: -- txn-mode = on|off
	txnModeDirectiveRegex = regexp.MustCompile(`(?i)^\s*--\s*txn-mode\s*=\s*(on|off)\s*$`)

	// txnIsolationDirectiveRegex matches the isolation level directive.
	// Format: -- txn-isolation = READ UNCOMMITTED|READ COMMITTED|REPEATABLE READ|SERIALIZABLE
	txnIsolationDirectiveRegex = regexp.MustCompile(`(?i)^\s*--\s*txn-isolation\s*=\s*(READ\s+UNCOMMITTED|READ\s+COMMITTED|REPEATABLE\s+READ|SERIALIZABLE)\s*$`)
)

// ParseTransactionMode extracts the transaction mode directive from the SQL script.
// It checks the first line of the script for the -- txn-mode = on|off directive.
// Returns the transaction mode and the SQL script without the directive.
// Deprecated: Use ParseTransactionConfig instead for full transaction configuration support.
func ParseTransactionMode(script string) (common.TransactionMode, string) {
	config, cleanScript := ParseTransactionConfig(script)
	return config.Mode, cleanScript
}

// ParseTransactionConfig extracts both transaction mode and isolation level directives from the SQL script.
// It checks the first two lines for directives:
// - Line 1: -- txn-mode = on|off
// - Line 2: -- txn-isolation = READ UNCOMMITTED|READ COMMITTED|REPEATABLE READ|SERIALIZABLE
// Returns the transaction configuration and the SQL script without the directives.
func ParseTransactionConfig(script string) (common.TransactionConfig, string) {
	config := common.TransactionConfig{
		Mode:      common.TransactionModeUnspecified,
		Isolation: common.IsolationLevelDefault,
	}

	lines := strings.Split(script, "\n")
	if len(lines) == 0 {
		return config, script
	}

	startLine := 0

	// Check the first line for transaction mode directive
	if len(lines) > 0 {
		matches := txnModeDirectiveRegex.FindStringSubmatch(lines[0])
		if len(matches) == 2 {
			mode := strings.ToLower(matches[1])
			config.Mode = common.TransactionMode(mode)
			startLine = 1
		}
	}

	// Check for isolation level directive on the current line
	if startLine < len(lines) {
		matches := txnIsolationDirectiveRegex.FindStringSubmatch(lines[startLine])
		if len(matches) == 2 {
			isolation := strings.ToUpper(matches[1])
			// Normalize the spacing
			isolation = strings.ReplaceAll(isolation, "  ", " ")
			config.Isolation = common.IsolationLevel(isolation)
			startLine++
		}
	}

	// Return the script without the directive lines
	if startLine > 0 {
		remainingScript := strings.Join(lines[startLine:], "\n")
		return config, strings.TrimSpace(remainingScript)
	}

	return config, script
}

// ConvertToSQLIsolation converts our IsolationLevel to database/sql.IsolationLevel
func ConvertToSQLIsolation(level common.IsolationLevel) sql.IsolationLevel {
	switch level {
	case common.IsolationLevelReadUncommitted:
		return sql.LevelReadUncommitted
	case common.IsolationLevelReadCommitted:
		return sql.LevelReadCommitted
	case common.IsolationLevelRepeatableRead:
		return sql.LevelRepeatableRead
	case common.IsolationLevelSerializable:
		return sql.LevelSerializable
	default:
		return sql.LevelDefault
	}
}
