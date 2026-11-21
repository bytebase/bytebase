package ghost

import (
	"context"
	"errors"
	"testing"

	"github.com/bytebase/omni/mysql/analysis"
	omnimysqlparser "github.com/bytebase/omni/mysql/parser"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

type fakeGhostDriver struct {
	db.Driver
	queries   []string
	responses map[string]func() ([]*v1pb.QueryResult, error)
}

func (d *fakeGhostDriver) QueryConn(_ context.Context, statement string, _ db.QueryContext) ([]*v1pb.QueryResult, error) {
	// Enforce omni's MySQL parser/read-only gate so queries rejected before reaching MySQL 8.4+ fail in tests.
	parsed, err := omnimysqlparser.Parse(statement)
	if err != nil {
		return nil, err
	}
	if len(parsed.Items) != 1 {
		return nil, errors.New("expected single statement")
	}
	if readOnly, returnsData := analysis.ValidateQuery(parsed.Items); !readOnly || !returnsData {
		return nil, errors.New("statement does not pass read-only query validation: " + statement)
	}
	d.queries = append(d.queries, statement)
	fn, ok := d.responses[statement]
	if !ok {
		return nil, errors.New("unexpected query: " + statement)
	}
	return fn()
}

func singleStringQueryResult(val string) ([]*v1pb.QueryResult, error) {
	return []*v1pb.QueryResult{
		{
			Rows: []*v1pb.QueryRow{
				{Values: []*v1pb.RowValue{{Kind: &v1pb.RowValue_StringValue{StringValue: val}}}},
			},
		},
	}, nil
}

func TestValidateBinlogAccessWithNilDriver(t *testing.T) {
	require.NotPanics(t, func() {
		result := ValidateBinlogAccess(context.Background(), nil, &storepb.DataSource{Host: "example.com", Username: "admin"})
		require.False(t, result.Valid)
		require.Equal(t, validationQueryFailed, result.FailureReason)
		require.ErrorContains(t, result.Error, "driver is nil")
	})
}

func TestSupportsEngine(t *testing.T) {
	tests := map[storepb.Engine]bool{
		storepb.Engine_MYSQL:              true,
		storepb.Engine_MARIADB:            true,
		storepb.Engine_TIDB:               false,
		storepb.Engine_OCEANBASE:          false,
		storepb.Engine_STARROCKS:          false,
		storepb.Engine_POSTGRES:           false,
		storepb.Engine_SPANNER:            false,
		storepb.Engine_COCKROACHDB:        false,
		storepb.Engine_ENGINE_UNSPECIFIED: false,
	}
	for engine, want := range tests {
		t.Run(engine.String(), func(t *testing.T) {
			require.Equal(t, want, SupportsEngine(engine))
		})
	}
}

func TestValidateBinlogAccess(t *testing.T) {
	driver := &fakeGhostDriver{
		responses: map[string]func() ([]*v1pb.QueryResult, error){
			"SELECT @@log_bin": func() ([]*v1pb.QueryResult, error) {
				return []*v1pb.QueryResult{
					{Rows: []*v1pb.QueryRow{{Values: []*v1pb.RowValue{{Kind: &v1pb.RowValue_Int64Value{Int64Value: 1}}}}}},
				}, nil
			},
			"SHOW BINARY LOGS": func() ([]*v1pb.QueryResult, error) {
				return []*v1pb.QueryResult{{}}, nil
			},
			"SHOW GRANTS": func() ([]*v1pb.QueryResult, error) {
				return singleStringQueryResult("GRANT REPLICATION SLAVE ON *.* TO `admin`@`%`")
			},
			"SELECT @@binlog_format": func() ([]*v1pb.QueryResult, error) {
				return singleStringQueryResult("ROW")
			},
		},
	}

	result := ValidateBinlogAccess(context.Background(), driver, &storepb.DataSource{Host: "example.com", Username: "admin"})
	require.True(t, result.Valid)
	require.NoError(t, result.Error)
	require.Empty(t, result.FailureReason)
	require.True(t, result.BinlogEnabled)
	require.True(t, result.HasPrivilege)
	require.Equal(t, "ROW", result.BinlogFormat)
	require.Equal(t, []string{
		"SELECT @@log_bin",
		"SHOW BINARY LOGS",
		"SHOW GRANTS",
		"SELECT @@binlog_format",
	}, driver.queries)
}

func TestValidateBinlogAccessDisabledStopsBeforeShowBinaryLogs(t *testing.T) {
	driver := &fakeGhostDriver{
		responses: map[string]func() ([]*v1pb.QueryResult, error){
			"SELECT @@log_bin": func() ([]*v1pb.QueryResult, error) {
				return singleStringQueryResult("OFF")
			},
		},
	}

	result := ValidateBinlogAccess(context.Background(), driver, &storepb.DataSource{Host: "example.com", Username: "admin"})
	require.False(t, result.Valid)
	require.Equal(t, binlogDisabled, result.FailureReason)
	require.False(t, result.BinlogEnabled)
	require.Equal(t, []string{"SELECT @@log_bin"}, driver.queries)
}

func TestValidateBinlogAccessMissingBinaryLogPrivilege(t *testing.T) {
	driver := &fakeGhostDriver{
		responses: map[string]func() ([]*v1pb.QueryResult, error){
			"SELECT @@log_bin": func() ([]*v1pb.QueryResult, error) {
				return singleStringQueryResult("ON")
			},
			"SHOW BINARY LOGS": func() ([]*v1pb.QueryResult, error) {
				return []*v1pb.QueryResult{{Error: "Access denied; you need (at least one of) the SUPER, REPLICATION CLIENT privilege(s) for this operation"}}, nil
			},
		},
	}

	result := ValidateBinlogAccess(context.Background(), driver, &storepb.DataSource{Host: "example.com", Username: "admin"})
	require.False(t, result.Valid)
	require.Equal(t, binlogStatusInaccessible, result.FailureReason)
	require.True(t, result.BinlogEnabled)
	require.Equal(t, []string{"SELECT @@log_bin", "SHOW BINARY LOGS"}, driver.queries)
}

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
