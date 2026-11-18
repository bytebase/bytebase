package ghost

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

type binlogValidationFailureReason string

const (
	noDirectConnection          binlogValidationFailureReason = "no_direct_connection"
	binlogStatusInaccessible    binlogValidationFailureReason = "binlog_status_inaccessible"
	binlogDisabled              binlogValidationFailureReason = "binlog_disabled"
	missingReplicationPrivilege binlogValidationFailureReason = "missing_replication_privilege"
	unsupportedBinlogFormat     binlogValidationFailureReason = "unsupported_binlog_format"
	validationQueryFailed       binlogValidationFailureReason = "validation_query_failed"
)

// errNoDirectConnection explains why a driver cannot run gh-ost's own queries.
var errNoDirectConnection = errors.New("gh-ost needs a direct database/sql connection, but this driver does not expose one")

// RequireDirectConnection returns the driver's database/sql pool, or an error
// when GetDB is nil. gh-ost queries the database itself, so a driver that owns
// its pool and exposes nothing here cannot run a gh-ost migration.
func RequireDirectConnection(driver db.Driver) (*sql.DB, error) {
	sqlDB := driver.GetDB()
	if sqlDB == nil {
		return nil, errNoDirectConnection
	}
	return sqlDB, nil
}

// SupportsEngine reports whether gh-ost can drive the engine. gh-ost needs MySQL
// replication, which only MySQL and MariaDB provide; the frontend offers the
// ghost directive for these engines only.
func SupportsEngine(engine storepb.Engine) bool {
	return engine == storepb.Engine_MYSQL || engine == storepb.Engine_MARIADB
}

// BinlogValidationResult contains detailed results of binlog access validation
type BinlogValidationResult struct {
	// Core validation state
	Valid         bool
	Error         error
	FailureReason binlogValidationFailureReason

	// Detailed findings for specific error messages
	BinlogEnabled     bool
	BinlogFormat      string
	HasPrivilege      bool
	MissingPrivileges []string // Specific privileges that are missing
	CurrentGrants     []string // Current grants for debugging
}

// ValidateBinlogAccess performs comprehensive validation of gh-ost prerequisites
// Returns a structured result that can be used for both plan checks and execution
func ValidateBinlogAccess(ctx context.Context, driver db.Driver, adminDataSource *storepb.DataSource) *BinlogValidationResult {
	result := &BinlogValidationResult{
		Valid:             true,
		BinlogEnabled:     false,
		HasPrivilege:      false,
		MissingPrivileges: []string{},
		CurrentGrants:     []string{},
	}

	sqlDB, err := RequireDirectConnection(driver)
	if err != nil {
		result.Valid = false
		result.FailureReason = noDirectConnection
		result.Error = err
		slog.Error("binlog access validation failed: driver exposes no database/sql connection",
			slog.String("host", adminDataSource.GetHost()),
			slog.String("user", adminDataSource.GetUsername()))
		return result
	}

	// Test 1: Check if we can access binary log status
	// Try both old and new MySQL commands for compatibility
	canAccessBinlog := false
	if _, err := sqlDB.ExecContext(ctx, "SHOW MASTER STATUS"); err == nil {
		canAccessBinlog = true
	} else if _, err := sqlDB.ExecContext(ctx, "SHOW BINARY LOG STATUS"); err == nil {
		canAccessBinlog = true
	}

	if !canAccessBinlog {
		result.Valid = false
		result.FailureReason = binlogStatusInaccessible
		result.Error = errors.New("cannot access binary logs - ensure user has REPLICATION CLIENT privilege")
		slog.Error("binlog access validation failed: cannot access binary logs",
			slog.String("host", adminDataSource.GetHost()),
			slog.String("user", adminDataSource.GetUsername()))
		return result
	}

	// Test 2: Check if binary logging is enabled
	var logBin string
	row := sqlDB.QueryRowContext(ctx, "SELECT @@log_bin")
	if err := row.Scan(&logBin); err != nil {
		result.Valid = false
		result.FailureReason = validationQueryFailed
		result.Error = errors.Wrap(err, "failed to check if binary logging is enabled")
		return result
	}

	result.BinlogEnabled = (logBin == "1" || strings.ToUpper(logBin) == "ON")
	if !result.BinlogEnabled {
		result.Valid = false
		result.FailureReason = binlogDisabled
		result.Error = errors.New("binary logging is not enabled on this MySQL instance")
		return result
	}

	// Test 3: Check user privileges
	rows, err := sqlDB.QueryContext(ctx, "SHOW GRANTS")
	if err != nil {
		result.Valid = false
		result.FailureReason = validationQueryFailed
		result.Error = errors.Wrap(err, "failed to check user grants")
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var grant string
		if err := rows.Scan(&grant); err != nil {
			slog.Warn("failed to scan grant",
				slog.String("host", adminDataSource.GetHost()),
				slog.String("user", adminDataSource.GetUsername()),
				slog.String("error", err.Error()))
			continue
		}
		result.CurrentGrants = append(result.CurrentGrants, grant)

		upperGrant := strings.ToUpper(grant)
		if strings.Contains(upperGrant, "REPLICATION SLAVE") ||
			strings.Contains(upperGrant, "ALL PRIVILEGES") {
			result.HasPrivilege = true
		}
	}

	if err := rows.Err(); err != nil {
		result.Valid = false
		result.FailureReason = validationQueryFailed
		result.Error = errors.Wrap(err, "error reading grants")
		return result
	}

	if !result.HasPrivilege {
		result.Valid = false
		result.FailureReason = missingReplicationPrivilege
		result.MissingPrivileges = append(result.MissingPrivileges, "REPLICATION SLAVE")
		result.Error = errors.New("user does not have REPLICATION SLAVE privilege required for gh-ost")
		slog.Error("missing REPLICATION SLAVE privilege",
			slog.String("host", adminDataSource.GetHost()),
			slog.String("user", adminDataSource.GetUsername()),
			slog.Any("grants", result.CurrentGrants))
		return result
	}

	// Test 4: Check binlog format (must be ROW or MIXED, not STATEMENT)
	row = sqlDB.QueryRowContext(ctx, "SELECT @@binlog_format")
	if err := row.Scan(&result.BinlogFormat); err != nil {
		result.Valid = false
		result.FailureReason = validationQueryFailed
		result.Error = errors.Wrap(err, "failed to check binlog format")
		return result
	}

	if strings.ToUpper(result.BinlogFormat) == "STATEMENT" {
		result.Valid = false
		result.FailureReason = unsupportedBinlogFormat
		result.Error = errors.Errorf("binlog_format is %s, but gh-ost requires ROW or MIXED format", result.BinlogFormat)
		return result
	}

	// All checks passed
	slog.Info("binlog access validation passed",
		slog.String("host", adminDataSource.GetHost()),
		slog.String("user", adminDataSource.GetUsername()),
		slog.String("binlog_format", result.BinlogFormat))

	return result
}

// GetUserFriendlyError returns a user-friendly error message based on validation results
func (r *BinlogValidationResult) GetUserFriendlyError() (title, content string) {
	if r.Valid {
		return "", ""
	}

	if r.FailureReason == noDirectConnection {
		return "gh-ost is not supported on this instance",
			"gh-ost needs a direct database/sql connection, but the database driver for this instance does not expose one."
	}

	title = "gh-ost migration prerequisites not met"

	switch r.FailureReason {
	case binlogStatusInaccessible:
		return title, "Cannot access binary log status. Ensure the Bytebase admin user has REPLICATION CLIENT privilege."
	case binlogDisabled:
		return title, "Binary logging is not enabled on this MySQL instance."
	case missingReplicationPrivilege:
		missingPrivileges := strings.Join(r.MissingPrivileges, ", ")
		if missingPrivileges == "" {
			missingPrivileges = "REPLICATION SLAVE"
		}
		return title, fmt.Sprintf("Database user is missing required privilege: %s\n", missingPrivileges) +
			"Please grant REPLICATION SLAVE or an equivalent replication privilege to the Bytebase admin user."
	case unsupportedBinlogFormat:
		return title, fmt.Sprintf("Current binlog_format is %s, but gh-ost requires ROW or MIXED format.\n", r.BinlogFormat) +
			"Please change it with:\n" +
			"SET GLOBAL binlog_format='ROW'"
	case validationQueryFailed:
		if r.Error != nil {
			return title, fmt.Sprintf("Validation failed: %v", r.Error)
		}
		return title, "Validation failed"
	}

	if r.Error != nil {
		return title, fmt.Sprintf("Validation failed: %v", r.Error)
	}
	return title, "Unknown validation error occurred"
}
