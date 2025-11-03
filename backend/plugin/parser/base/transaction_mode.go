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

// ParseTransactionConfig extracts both transaction mode and isolation level directives from the SQL script.
// It scans comment lines at the top of the file for directives:
// - -- txn-mode = on|off
// - -- txn-isolation = READ UNCOMMITTED|READ COMMITTED|REPEATABLE READ|SERIALIZABLE
// Directives can appear in any order within the top comment lines.
// Scanning stops at the first non-comment, non-empty line.
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

	directiveLines := make(map[int]bool)

	// Scan lines from the top, stopping at first non-comment/non-empty line
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// If it's not a comment, stop scanning for directives
		if !strings.HasPrefix(trimmed, "--") {
			break
		}

		// Check for transaction mode directive
		if matches := txnModeDirectiveRegex.FindStringSubmatch(line); len(matches) == 2 {
			mode := strings.ToLower(matches[1])
			config.Mode = common.TransactionMode(mode)
			directiveLines[i] = true
			continue
		}

		// Check for isolation level directive
		if matches := txnIsolationDirectiveRegex.FindStringSubmatch(line); len(matches) == 2 {
			isolation := strings.ToUpper(matches[1])
			// Normalize the spacing
			isolation = strings.ReplaceAll(isolation, "  ", " ")
			// Note: We set the value as-is here. Invalid values will be caught
			// by the specific database driver during execution, allowing each
			// database to validate according to its supported levels.
			config.Isolation = common.IsolationLevel(isolation)
			directiveLines[i] = true
			continue
		}
	}

	// Remove directive lines from the script
	if len(directiveLines) > 0 {
		var remainingLines []string
		for i, line := range lines {
			if !directiveLines[i] {
				remainingLines = append(remainingLines, line)
			}
		}
		return config, strings.TrimSpace(strings.Join(remainingLines, "\n"))
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
