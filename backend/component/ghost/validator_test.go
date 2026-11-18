package ghost

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// fakeGhostDriver stands in for a driver that owns its pool. Only GetDB is
// exercised by the validation path.
type fakeGhostDriver struct {
	db.Driver
	sqlDB *sql.DB
}

func (d *fakeGhostDriver) GetDB() *sql.DB { return d.sqlDB }

func TestValidateBinlogAccessWithoutDirectConnection(t *testing.T) {
	driver := &fakeGhostDriver{}

	require.NotPanics(t, func() {
		result := ValidateBinlogAccess(context.Background(), driver, &storepb.DataSource{Host: "example.com", Username: "admin"})
		require.False(t, result.Valid)
		require.Equal(t, noDirectConnection, result.FailureReason)
		require.ErrorIs(t, result.Error, errNoDirectConnection)

		title, content := result.GetUserFriendlyError()
		require.Equal(t, "gh-ost is not supported on this instance", title)
		require.Contains(t, content, "direct database/sql connection")
	})
}

func TestValidateBinlogAccessWithDirectConnection(t *testing.T) {
	sqlDB := sql.OpenDB(validationConnector{})
	defer sqlDB.Close()

	result := ValidateBinlogAccess(context.Background(), &fakeGhostDriver{sqlDB: sqlDB}, &storepb.DataSource{Host: "example.com", Username: "admin"})
	require.True(t, result.Valid)
	require.NoError(t, result.Error)
	require.Empty(t, result.FailureReason)
	require.True(t, result.BinlogEnabled)
	require.True(t, result.HasPrivilege)
	require.Equal(t, "ROW", result.BinlogFormat)
}

// validationRowSets answers the queries ValidateBinlogAccess issues after the
// binlog status probe, which runs through ExecContext.
var validationRowSets = map[string][][]driver.Value{
	"SELECT @@log_bin":       {{"1"}},
	"SHOW GRANTS":            {{"GRANT REPLICATION SLAVE ON *.* TO `admin`@`%`"}},
	"SELECT @@binlog_format": {{"ROW"}},
}

type validationConnector struct{}

func (validationConnector) Connect(context.Context) (driver.Conn, error) {
	return validationConn{}, nil
}
func (validationConnector) Driver() driver.Driver { return validationDriver{} }

type validationDriver struct{}

func (validationDriver) Open(string) (driver.Conn, error) { return validationConn{}, nil }

type validationConn struct{}

func (validationConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}
func (validationConn) Close() error { return nil }
func (validationConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (validationConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	if query == "SHOW MASTER STATUS" {
		return driver.RowsAffected(0), nil
	}
	return nil, errors.New("unexpected exec: " + query)
}

func (validationConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	values, ok := validationRowSets[query]
	if !ok {
		return nil, errors.New("unexpected query: " + query)
	}
	return &staticRows{values: values}, nil
}

type staticRows struct {
	values [][]driver.Value
	next   int
}

func (*staticRows) Columns() []string { return []string{"value"} }
func (*staticRows) Close() error      { return nil }

func (r *staticRows) Next(dest []driver.Value) error {
	if r.next >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.next])
	r.next++
	return nil
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
			name: "no direct connection",
			result: &BinlogValidationResult{
				Valid:         false,
				FailureReason: noDirectConnection,
				Error:         errNoDirectConnection,
			},
			wantTitle:   "gh-ost is not supported on this instance",
			wantContent: "gh-ost needs a direct database/sql connection, but the database driver for this instance does not expose one.",
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
