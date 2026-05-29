package ghost

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBinlogValidationResultGetUserFriendlyError(t *testing.T) {
	tests := []struct {
		name        string
		result      *BinlogValidationResult
		wantTitle   string
		wantContent string
	}{
		{
			name: "valid result",
			result: &BinlogValidationResult{
				Valid: true,
			},
			wantTitle:   "",
			wantContent: "",
		},
		{
			name: "binlog status inaccessible",
			result: &BinlogValidationResult{
				Valid:         false,
				BinlogEnabled: false,
				Error:         errors.New("cannot access binary logs - ensure user has REPLICATION CLIENT privilege"),
				FailureReason: binlogStatusInaccessible,
			},
			wantTitle:   "gh-ost migration prerequisites not met",
			wantContent: "Cannot access binary log status. Ensure the Bytebase admin user has REPLICATION CLIENT privilege.",
		},
		{
			name: "binary logging disabled",
			result: &BinlogValidationResult{
				Valid:         false,
				BinlogEnabled: false,
				Error:         errors.New("binary logging is not enabled on this MySQL instance"),
				FailureReason: binlogDisabled,
			},
			wantTitle:   "gh-ost migration prerequisites not met",
			wantContent: "Binary logging is not enabled on this MySQL instance.",
		},
		{
			name: "missing replication privilege",
			result: &BinlogValidationResult{
				Valid:             false,
				BinlogEnabled:     true,
				HasPrivilege:      false,
				MissingPrivileges: []string{"REPLICATION SLAVE"},
				Error:             errors.New("user does not have REPLICATION SLAVE privilege required for gh-ost"),
				FailureReason:     missingReplicationPrivilege,
			},
			wantTitle:   "gh-ost migration prerequisites not met",
			wantContent: "Database user is missing required privilege: REPLICATION SLAVE\nPlease grant REPLICATION SLAVE or an equivalent replication privilege to the Bytebase admin user.",
		},
		{
			name: "unsupported binlog format",
			result: &BinlogValidationResult{
				Valid:         false,
				BinlogEnabled: true,
				HasPrivilege:  true,
				BinlogFormat:  "statement",
				Error:         errors.New("binlog_format is statement, but gh-ost requires ROW or MIXED format"),
				FailureReason: unsupportedBinlogFormat,
			},
			wantTitle:   "gh-ost migration prerequisites not met",
			wantContent: "Current binlog_format is statement, but gh-ost requires ROW or MIXED format.\nPlease change it with:\nSET GLOBAL binlog_format='ROW'",
		},
		{
			name: "generic validation query failure",
			result: &BinlogValidationResult{
				Valid:         false,
				BinlogEnabled: false,
				Error:         errors.New("failed to check if binary logging is enabled: access denied"),
				FailureReason: validationQueryFailed,
			},
			wantTitle:   "gh-ost migration prerequisites not met",
			wantContent: "Validation failed: failed to check if binary logging is enabled: access denied",
		},
		{
			name: "unknown invalid result falls back to error",
			result: &BinlogValidationResult{
				Valid: false,
				Error: errors.New("unexpected validator failure"),
			},
			wantTitle:   "gh-ost migration prerequisites not met",
			wantContent: "Validation failed: unexpected validator failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotContent := tt.result.GetUserFriendlyError()
			require.Equal(t, tt.wantTitle, gotTitle)
			require.Equal(t, tt.wantContent, gotContent)
		})
	}
}
