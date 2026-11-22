package ghost

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

type binlogValidationFailureReason string

const (
	binlogStatusInaccessible    binlogValidationFailureReason = "binlog_status_inaccessible"
	binlogDisabled              binlogValidationFailureReason = "binlog_disabled"
	missingReplicationPrivilege binlogValidationFailureReason = "missing_replication_privilege"
	unsupportedBinlogFormat     binlogValidationFailureReason = "unsupported_binlog_format"
	validationQueryFailed       binlogValidationFailureReason = "validation_query_failed"
)

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

	if driver == nil {
		result.Valid = false
		result.FailureReason = validationQueryFailed
		result.Error = errors.New("database driver is nil")
		return result
	}

	query := func(statement string) (*v1pb.QueryResult, error) {
		results, err := driver.QueryConn(ctx, statement, db.QueryContext{
			MaximumSQLResultSize: common.DefaultMaximumSQLResultSize,
		})
		if err != nil {
			return nil, err
		}
		if len(results) == 0 {
			return nil, errors.New("query returned no result")
		}
		if results[0].Error != "" {
			return nil, errors.New(results[0].Error)
		}
		return results[0], nil
	}

	// Test 1: Check if binary logging is enabled before SHOW BINARY LOGS, which errors when log_bin is OFF.
	logBinRes, err := query("SELECT @@log_bin")
	if err != nil {
		result.Valid = false
		result.FailureReason = validationQueryFailed
		result.Error = errors.Wrap(err, "failed to check if binary logging is enabled")
		return result
	}
	logBin, ok := singleCellString(logBinRes)
	if !ok {
		result.Valid = false
		result.FailureReason = validationQueryFailed
		result.Error = errors.New("failed to check if binary logging is enabled: empty result")
		return result
	}

	result.BinlogEnabled = (logBin == "1" || strings.ToUpper(logBin) == "ON")
	if !result.BinlogEnabled {
		result.Valid = false
		result.FailureReason = binlogDisabled
		result.Error = errors.New("binary logging is not enabled on this MySQL instance")
		return result
	}

	// Test 2: Check if we can access binary logs.
	// SHOW BINARY LOGS parses under omni's MySQL 8.0 grammar and requires REPLICATION CLIENT on MySQL 8.0, 8.4+, and MariaDB.
	if _, err := query("SHOW BINARY LOGS"); err != nil {
		result.Valid = false
		result.FailureReason = binlogStatusInaccessible
		result.Error = errors.New("cannot access binary logs - ensure user has REPLICATION CLIENT privilege")
		slog.Error("binlog access validation failed: cannot access binary logs",
			slog.String("host", adminDataSource.GetHost()),
			slog.String("user", adminDataSource.GetUsername()))
		return result
	}

	// Test 3: Check user privileges
	grantsRes, err := query("SHOW GRANTS")
	if err != nil {
		result.Valid = false
		result.FailureReason = validationQueryFailed
		result.Error = errors.Wrap(err, "failed to check user grants")
		return result
	}

	for _, row := range grantsRes.GetRows() {
		if len(row.GetValues()) == 0 {
			continue
		}
		grant, ok := rowValueString(row.Values[0])
		if !ok {
			continue
		}
		result.CurrentGrants = append(result.CurrentGrants, grant)

		upperGrant := strings.ToUpper(grant)
		if strings.Contains(upperGrant, "REPLICATION SLAVE") ||
			strings.Contains(upperGrant, "ALL PRIVILEGES") {
			result.HasPrivilege = true
		}
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
	formatRes, err := query("SELECT @@binlog_format")
	if err != nil {
		result.Valid = false
		result.FailureReason = validationQueryFailed
		result.Error = errors.Wrap(err, "failed to check binlog format")
		return result
	}
	binlogFormat, ok := singleCellString(formatRes)
	if !ok {
		result.Valid = false
		result.FailureReason = validationQueryFailed
		result.Error = errors.New("failed to check binlog format: empty result")
		return result
	}
	result.BinlogFormat = binlogFormat

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

func singleCellString(res *v1pb.QueryResult) (string, bool) {
	if res == nil || len(res.Rows) == 0 || len(res.Rows[0].GetValues()) == 0 {
		return "", false
	}
	return rowValueString(res.Rows[0].Values[0])
}

func rowValueString(v *v1pb.RowValue) (string, bool) {
	if v == nil {
		return "", false
	}
	switch kind := v.Kind.(type) {
	case *v1pb.RowValue_StringValue:
		return kind.StringValue, true
	case *v1pb.RowValue_Int64Value:
		return strconv.FormatInt(kind.Int64Value, 10), true
	case *v1pb.RowValue_Uint64Value:
		return strconv.FormatUint(kind.Uint64Value, 10), true
	case *v1pb.RowValue_Int32Value:
		return strconv.FormatInt(int64(kind.Int32Value), 10), true
	case *v1pb.RowValue_Uint32Value:
		return strconv.FormatUint(uint64(kind.Uint32Value), 10), true
	case *v1pb.RowValue_BoolValue:
		return strconv.FormatBool(kind.BoolValue), true
	case *v1pb.RowValue_BytesValue:
		return string(kind.BytesValue), true
	default:
		return "", false
	}
}

// GetUserFriendlyError returns a user-friendly error message based on validation results
func (r *BinlogValidationResult) GetUserFriendlyError() (title, content string) {
	if r.Valid {
		return "", ""
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
