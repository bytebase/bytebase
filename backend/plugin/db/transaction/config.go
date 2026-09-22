// Package transaction defines transaction execution settings shared by database drivers.
package transaction

import "database/sql"

// Mode represents the transaction execution mode for a migration script.
type Mode string

const (
	// ModeOn wraps the script in a single transaction.
	ModeOn Mode = "on"
	// ModeOff executes the script's statements sequentially in auto-commit mode.
	ModeOff Mode = "off"
	// ModeUnspecified means no explicit mode was specified.
	ModeUnspecified Mode = ""
)

// IsolationLevel represents the transaction isolation level.
type IsolationLevel string

const (
	// IsolationLevelDefault uses the database's default isolation level.
	IsolationLevelDefault IsolationLevel = ""
	// IsolationLevelReadUncommitted allows dirty reads.
	IsolationLevelReadUncommitted IsolationLevel = "READ UNCOMMITTED"
	// IsolationLevelReadCommitted prevents dirty reads.
	IsolationLevelReadCommitted IsolationLevel = "READ COMMITTED"
	// IsolationLevelRepeatableRead prevents dirty reads and non-repeatable reads.
	IsolationLevelRepeatableRead IsolationLevel = "REPEATABLE READ"
	// IsolationLevelSerializable provides the highest isolation level.
	IsolationLevelSerializable IsolationLevel = "SERIALIZABLE"
)

// Config represents the complete transaction configuration.
type Config struct {
	Mode      Mode
	Isolation IsolationLevel
}

// DefaultMode returns the default transaction mode.
// All engines default to "on" (transactional) for safety and backward compatibility.
// Users can explicitly set "-- txn-mode = off" when needed for engines with limited transactional DDL support.
func DefaultMode() Mode {
	return ModeOn
}

// ToSQLIsolation converts our IsolationLevel to database/sql.IsolationLevel
func ToSQLIsolation(level IsolationLevel) sql.IsolationLevel {
	switch level {
	case IsolationLevelReadUncommitted:
		return sql.LevelReadUncommitted
	case IsolationLevelReadCommitted:
		return sql.LevelReadCommitted
	case IsolationLevelRepeatableRead:
		return sql.LevelRepeatableRead
	case IsolationLevelSerializable:
		return sql.LevelSerializable
	default:
		return sql.LevelDefault
	}
}
