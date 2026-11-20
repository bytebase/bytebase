// Package transaction defines transaction execution settings shared by database drivers.
package transaction

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
