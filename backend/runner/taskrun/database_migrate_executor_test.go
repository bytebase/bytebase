package taskrun

import (
	"context"
	"database/sql"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
)

func TestGetPrependStatements(t *testing.T) {
	tests := []struct {
		name        string
		engine      storepb.Engine
		statement   string
		want        string
		wantErr     bool
		description string
	}{
		// Non-PostgreSQL engines should return empty string
		{
			name:        "mysql engine returns empty",
			engine:      storepb.Engine_MYSQL,
			statement:   "SET role = 'admin';",
			want:        "",
			wantErr:     false,
			description: "Non-PostgreSQL engines should always return empty string",
		},
		{
			name:        "oracle engine returns empty",
			engine:      storepb.Engine_ORACLE,
			statement:   "SET role = 'admin';",
			want:        "",
			wantErr:     false,
			description: "Non-PostgreSQL engines should always return empty string",
		},

		// PostgreSQL role statements
		{
			name:        "set role with single quotes",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET role = 'admin';",
			want:        "SET role = 'admin';",
			wantErr:     false,
			description: "Should return the exact text of SET role statement",
		},
		{
			name:        "set role without quotes",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET role = admin;",
			want:        "SET role = admin;",
			wantErr:     false,
			description: "Should handle unquoted role names",
		},
		{
			name:        "set role with double quotes",
			engine:      storepb.Engine_POSTGRES,
			statement:   `SET role = "admin_user";`,
			want:        `SET role = "admin_user";`,
			wantErr:     false,
			description: "Should handle double-quoted role names",
		},
		{
			name:        "set role to none",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET role = none;",
			want:        "SET role = none;",
			wantErr:     false,
			description: "Should handle ROLE set to NONE",
		},
		{
			name:        "set role with whitespace",
			engine:      storepb.Engine_POSTGRES,
			statement:   "  SET    role   =   'admin'  ;  ",
			want:        "SET    role   =   'admin'  ;",
			wantErr:     false,
			description: "Should preserve internal whitespace but trim leading/trailing",
		},

		// PostgreSQL search_path statements
		{
			name:        "set search_path single schema",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET search_path = 'public';",
			want:        "SET search_path = 'public';",
			wantErr:     false,
			description: "Should return search_path statement",
		},
		{
			name:        "set search_path multiple schemas",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET search_path = schema1, schema2, public;",
			want:        "SET search_path = schema1, schema2, public;",
			wantErr:     false,
			description: "Should handle multiple schemas in search_path",
		},
		{
			name:        "set search_path with quotes",
			engine:      storepb.Engine_POSTGRES,
			statement:   `SET search_path = "schema with spaces", public;`,
			want:        `SET search_path = "schema with spaces", public;`,
			wantErr:     false,
			description: "Should handle quoted schema names with spaces",
		},
		{
			name:        "set search_path to default",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET search_path TO DEFAULT;",
			want:        "SET search_path TO DEFAULT;",
			wantErr:     false,
			description: "Should handle search_path TO DEFAULT syntax",
		},

		// Multiple statements - should return first matching
		{
			name:        "multiple statements with role first",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET role = 'admin'; SET search_path = public; SELECT 1;",
			want:        "SET role = 'admin';",
			wantErr:     false,
			description: "Should return first SET role statement when multiple statements present",
		},
		{
			name:        "multiple statements with search_path first",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET search_path = public; SET role = 'admin'; SELECT 1;",
			want:        "SET search_path = public;",
			wantErr:     false,
			description: "Should return first SET search_path statement when multiple statements present",
		},
		{
			name:        "multiple statements no role or search_path",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET work_mem = '256MB'; SET max_parallel_workers = 4; SELECT 1;",
			want:        "",
			wantErr:     false,
			description: "Should return empty when no role or search_path statements present",
		},

		// Other SET statements should be ignored
		{
			name:        "set timezone ignored",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET timezone = 'UTC';",
			want:        "",
			wantErr:     false,
			description: "SET statements other than role/search_path should be ignored",
		},
		{
			name:        "set work_mem ignored",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET work_mem = '256MB';",
			want:        "",
			wantErr:     false,
			description: "SET work_mem should be ignored",
		},
		{
			name:        "set statement_timeout ignored",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET statement_timeout = 30000;",
			want:        "",
			wantErr:     false,
			description: "SET statement_timeout should be ignored",
		},

		// Non-SET statements should be ignored
		{
			name:        "select statement ignored",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SELECT * FROM users WHERE role = 'admin';",
			want:        "",
			wantErr:     false,
			description: "Non-SET statements should be ignored",
		},
		{
			name:        "create table ignored",
			engine:      storepb.Engine_POSTGRES,
			statement:   "CREATE TABLE role (id int, name text);",
			want:        "",
			wantErr:     false,
			description: "DDL statements should be ignored even if they contain 'role' keyword",
		},
		{
			name:        "update statement ignored",
			engine:      storepb.Engine_POSTGRES,
			statement:   "UPDATE users SET role = 'admin' WHERE id = 1;",
			want:        "",
			wantErr:     false,
			description: "DML statements should be ignored",
		},

		// Case variations
		{
			name:        "uppercase SET ROLE",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET ROLE = 'admin';",
			want:        "SET ROLE = 'admin';",
			wantErr:     false,
			description: "Should handle uppercase keywords",
		},
		{
			name:        "mixed case Set Role",
			engine:      storepb.Engine_POSTGRES,
			statement:   "Set Role = 'admin';",
			want:        "Set Role = 'admin';",
			wantErr:     false,
			description: "Should handle mixed case keywords",
		},

		// Complex statements that should be parsed correctly
		{
			name:        "set role with complex name",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET role = 'application_admin_readonly_2024';",
			want:        "SET role = 'application_admin_readonly_2024';",
			wantErr:     false,
			description: "Should handle complex role names",
		},
		{
			name:        "set search_path with current user",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET search_path = '$user', public;",
			want:        "SET search_path = '$user', public;",
			wantErr:     false,
			description: "Should handle special values like $user in search_path",
		},

		// Test pg_query_go node.GetText() behavior with whitespace and formatting
		{
			name:        "statement with newlines",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET role = \n'admin';",
			want:        "SET role = \n'admin';",
			wantErr:     false,
			description: "Should preserve newlines in the original statement text",
		},
		{
			name:        "statement with tabs",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET\trole\t=\t'admin';",
			want:        "SET\trole\t=\t'admin';",
			wantErr:     false,
			description: "Should preserve tab characters in the original statement text",
		},
		{
			name:        "statement with comments",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET role = 'admin'; -- Set admin role",
			want:        "SET role = 'admin';",
			wantErr:     false,
			description: "Should handle statements with comments (pg_query_go behavior)",
		},

		// Error cases
		{
			name:        "invalid syntax",
			engine:      storepb.Engine_POSTGRES,
			statement:   "SET role = ;",
			want:        "",
			wantErr:     true,
			description: "Should return error for invalid PostgreSQL syntax",
		},
		{
			name:        "empty statement",
			engine:      storepb.Engine_POSTGRES,
			statement:   "",
			want:        "",
			wantErr:     false,
			description: "Should handle empty statement gracefully",
		},
		{
			name:        "whitespace only statement",
			engine:      storepb.Engine_POSTGRES,
			statement:   "   \n\t  ",
			want:        "",
			wantErr:     false,
			description: "Should handle whitespace-only statement gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPrependStatements(tt.engine, tt.statement)
			if tt.wantErr {
				require.Error(t, err, "Expected error for test case: %s", tt.description)
			} else {
				require.NoError(t, err, "Unexpected error for test case: %s", tt.description)
				require.Equal(t, tt.want, got, "Test case: %s", tt.description)
			}
		})
	}
}

func TestExecuteSheetMigration(t *testing.T) {
	tests := []struct {
		name             string
		statement        string
		driverErr        error
		ghostErr         error
		wantErr          string
		wantExecuteCalls int
		wantGhostCalls   int
	}{
		{
			name:             "standard migration uses driver execute",
			statement:        "ALTER TABLE users ADD COLUMN status VARCHAR(50);",
			wantExecuteCalls: 1,
			wantGhostCalls:   0,
		},
		{
			name:             "gh-ost directive uses ghost executor",
			statement:        "-- gh-ost = {}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			wantExecuteCalls: 0,
			wantGhostCalls:   1,
		},
		{
			name:             "standard migration returns driver error",
			statement:        "ALTER TABLE users ADD COLUMN status VARCHAR(50);",
			driverErr:        context.Canceled,
			wantErr:          context.Canceled.Error(),
			wantExecuteCalls: 1,
			wantGhostCalls:   0,
		},
		{
			name:             "gh-ost migration returns ghost error",
			statement:        "-- gh-ost = {}\nALTER TABLE users ADD COLUMN status VARCHAR(50);",
			ghostErr:         context.DeadlineExceeded,
			wantErr:          context.DeadlineExceeded.Error(),
			wantExecuteCalls: 0,
			wantGhostCalls:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver := &fakeMigrationDriver{executeErr: tt.driverErr}
			sheet := &store.SheetMessage{Statement: tt.statement}
			ghostCalls := 0

			err := executeSheetMigration(context.Background(), driver, sheet, db.ExecuteOptions{}, func() error {
				ghostCalls++
				return tt.ghostErr
			})

			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.wantErr)
			}
			require.Len(t, driver.executedStatements, tt.wantExecuteCalls)
			require.Equal(t, tt.wantGhostCalls, ghostCalls)
		})
	}
}

type fakeMigrationDriver struct {
	executedStatements []string
	executeErr         error
}

func (d *fakeMigrationDriver) Open(_ context.Context, _ storepb.Engine, _ db.ConnectionConfig) (db.Driver, error) {
	return d, nil
}

func (*fakeMigrationDriver) Close(_ context.Context) error {
	return nil
}

func (*fakeMigrationDriver) Ping(_ context.Context) error {
	return nil
}

func (*fakeMigrationDriver) GetDB() *sql.DB {
	return nil
}

func (d *fakeMigrationDriver) Execute(_ context.Context, statement string, _ db.ExecuteOptions) (int64, error) {
	d.executedStatements = append(d.executedStatements, statement)
	return 0, d.executeErr
}

func (*fakeMigrationDriver) QueryConn(_ context.Context, _ *sql.Conn, _ string, _ db.QueryContext) ([]*v1pb.QueryResult, error) {
	return nil, nil
}

func (*fakeMigrationDriver) SyncInstance(_ context.Context) (*db.InstanceMetadata, error) {
	return nil, nil
}

func (*fakeMigrationDriver) SyncDBSchema(_ context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	return nil, nil
}

func (*fakeMigrationDriver) Dump(_ context.Context, _ io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	return nil
}
