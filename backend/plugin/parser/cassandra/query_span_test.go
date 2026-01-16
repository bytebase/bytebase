package cassandra

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetQuerySpan(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      base.QuerySpan
	}{
		{
			name:      "SELECT with specific columns",
			statement: "SELECT name, email FROM users WHERE id = 123",
			want: base.QuerySpan{
				Type: base.Select,
				Results: []base.QuerySpanResult{
					{
						Name: "name",
						SourceColumns: base.SourceColumnSet{
							base.ColumnResource{
								Database: "test_keyspace",
								Table:    "users",
								Column:   "name",
							}: true,
						},
						IsPlainField: true,
					},
					{
						Name: "email",
						SourceColumns: base.SourceColumnSet{
							base.ColumnResource{
								Database: "test_keyspace",
								Table:    "users",
								Column:   "email",
							}: true,
						},
						IsPlainField: true,
					},
				},
			},
		},
		{
			name:      "SELECT * (asterisk) - falls back without metadata",
			statement: "SELECT * FROM products",
			want: base.QuerySpan{
				Type: base.Select,
				Results: []base.QuerySpanResult{
					{
						Name:           "",
						SourceColumns:  base.SourceColumnSet{},
						SelectAsterisk: true,
					},
				},
			},
		},
		{
			name:      "SELECT with keyspace.table notation",
			statement: "SELECT id, name FROM myapp.customers",
			want: base.QuerySpan{
				Type: base.Select,
				Results: []base.QuerySpanResult{
					{
						Name: "id",
						SourceColumns: base.SourceColumnSet{
							base.ColumnResource{
								Database: "myapp",
								Table:    "customers",
								Column:   "id",
							}: true,
						},
						IsPlainField: true,
					},
					{
						Name: "name",
						SourceColumns: base.SourceColumnSet{
							base.ColumnResource{
								Database: "myapp",
								Table:    "customers",
								Column:   "name",
							}: true,
						},
						IsPlainField: true,
					},
				},
			},
		},
		{
			name:      "SELECT with double-quoted table name",
			statement: `SELECT id FROM "MyTable"`,
			want: base.QuerySpan{
				Type: base.Select,
				Results: []base.QuerySpanResult{
					{
						Name: "id",
						SourceColumns: base.SourceColumnSet{
							base.ColumnResource{
								Database: "test_keyspace",
								Table:    "MyTable",
								Column:   "id",
							}: true,
						},
						IsPlainField: true,
					},
				},
			},
		},
		{
			name:      "SELECT with double-quoted column names",
			statement: `SELECT "FirstName", "LastName" FROM users`,
			want: base.QuerySpan{
				Type: base.Select,
				Results: []base.QuerySpanResult{
					{
						Name: "FirstName",
						SourceColumns: base.SourceColumnSet{
							base.ColumnResource{
								Database: "test_keyspace",
								Table:    "users",
								Column:   "FirstName",
							}: true,
						},
						IsPlainField: true,
					},
					{
						Name: "LastName",
						SourceColumns: base.SourceColumnSet{
							base.ColumnResource{
								Database: "test_keyspace",
								Table:    "users",
								Column:   "LastName",
							}: true,
						},
						IsPlainField: true,
					},
				},
			},
		},
		{
			name:      "SELECT with double-quoted keyspace and table",
			statement: `SELECT id FROM "MyKeyspace"."MyTable"`,
			want: base.QuerySpan{
				Type: base.Select,
				Results: []base.QuerySpanResult{
					{
						Name: "id",
						SourceColumns: base.SourceColumnSet{
							base.ColumnResource{
								Database: "MyKeyspace",
								Table:    "MyTable",
								Column:   "id",
							}: true,
						},
						IsPlainField: true,
					},
				},
			},
		},
		{
			name:      "SELECT with alias using AS",
			statement: `SELECT name AS user_name, email AS user_email FROM users`,
			want: base.QuerySpan{
				Type: base.Select,
				Results: []base.QuerySpanResult{
					{
						Name: "user_name",
						SourceColumns: base.SourceColumnSet{
							base.ColumnResource{
								Database: "test_keyspace",
								Table:    "users",
								Column:   "name",
							}: true,
						},
						IsPlainField: true,
					},
					{
						Name: "user_email",
						SourceColumns: base.SourceColumnSet{
							base.ColumnResource{
								Database: "test_keyspace",
								Table:    "users",
								Column:   "email",
							}: true,
						},
						IsPlainField: true,
					},
				},
			},
		},
		{
			name:      "SELECT with double-quoted alias",
			statement: `SELECT name AS "User Name", email AS "User Email" FROM users`,
			want: base.QuerySpan{
				Type: base.Select,
				Results: []base.QuerySpanResult{
					{
						Name: "User Name",
						SourceColumns: base.SourceColumnSet{
							base.ColumnResource{
								Database: "test_keyspace",
								Table:    "users",
								Column:   "name",
							}: true,
						},
						IsPlainField: true,
					},
					{
						Name: "User Email",
						SourceColumns: base.SourceColumnSet{
							base.ColumnResource{
								Database: "test_keyspace",
								Table:    "users",
								Column:   "email",
							}: true,
						},
						IsPlainField: true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gCtx := base.GetQuerySpanContext{}

			got, err := GetQuerySpan(ctx, gCtx, base.Statement{Text: tt.statement}, "test_keyspace", "", false)
			require.NoError(t, err)
			require.NotNil(t, got)

			// Check Type field
			require.Equal(t, tt.want.Type, got.Type, "Query type mismatch")

			// Check Results length
			require.Equal(t, len(tt.want.Results), len(got.Results), "Result count mismatch")

			// Check each result
			for i, wantResult := range tt.want.Results {
				gotResult := got.Results[i]

				require.Equal(t, wantResult.Name, gotResult.Name, "Column name mismatch at index %d", i)
				require.Equal(t, wantResult.SelectAsterisk, gotResult.SelectAsterisk, "SelectAsterisk mismatch at index %d", i)
				require.Equal(t, wantResult.IsPlainField, gotResult.IsPlainField, "IsPlainField mismatch at index %d", i)

				// Check source columns if not SELECT *
				if !wantResult.SelectAsterisk {
					for col := range wantResult.SourceColumns {
						require.Contains(t, gotResult.SourceColumns, col, "Missing source column %+v", col)
					}
				}
			}

			// Check that SourceColumns at QuerySpan level contains table resources (not column resources)
			// SourceColumns should contain table-level access indicators (Column field is empty)
			if len(got.Results) > 0 && !got.Results[0].SelectAsterisk {
				foundTable := false
				for resource := range got.SourceColumns {
					if resource.Column == "" && resource.Table != "" {
						foundTable = true
						break
					}
				}
				require.True(t, foundTable, "SourceColumns should contain at least one table resource")
			}
		})
	}
}

func TestGetQuerySpanWithErrors(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		wantErr   bool
	}{
		{
			name:      "Invalid CQL syntax",
			statement: "SELECT FROM WHERE",
			wantErr:   true,
		},
		{
			name:      "Empty statement",
			statement: "",
			wantErr:   false, // Empty statement doesn't error, just returns empty span
		},
		{
			name:      "Unclosed quoted identifier",
			statement: `SELECT "unclosed FROM users`,
			wantErr:   true,
		},
		{
			name:      "Invalid SELECT syntax",
			statement: "SELECT , , FROM users",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gCtx := base.GetQuerySpanContext{}

			_, err := GetQuerySpan(ctx, gCtx, base.Statement{Text: tt.statement}, "test_keyspace", "", false)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWhereColumnExtraction(t *testing.T) {
	tests := []struct {
		name                     string
		statement                string
		expectedPredicateColumns []base.ColumnResource
		expectedTable            string // Table that should be in SourceColumns
	}{
		{
			name:      "Simple equality WHERE clause",
			statement: "SELECT * FROM users WHERE id = 123",
			expectedPredicateColumns: []base.ColumnResource{
				{Database: "test_keyspace", Table: "users", Column: "id"},
			},
			expectedTable: "users",
		},
		{
			name:      "Multiple AND conditions",
			statement: "SELECT name FROM users WHERE id = 123 AND status = 'active'",
			expectedPredicateColumns: []base.ColumnResource{
				{Database: "test_keyspace", Table: "users", Column: "id"},
				{Database: "test_keyspace", Table: "users", Column: "status"},
			},
			expectedTable: "users",
		},
		{
			name:      "WHERE with comparison operators",
			statement: "SELECT * FROM users WHERE age > 18 AND age < 65",
			expectedPredicateColumns: []base.ColumnResource{
				{Database: "test_keyspace", Table: "users", Column: "age"},
			},
			expectedTable: "users",
		},
		{
			name:      "WHERE with IN clause",
			statement: "SELECT * FROM users WHERE status IN ('active', 'pending')",
			expectedPredicateColumns: []base.ColumnResource{
				{Database: "test_keyspace", Table: "users", Column: "status"},
			},
			expectedTable: "users",
		},
		{
			name:      "WHERE with double-quoted columns",
			statement: `SELECT * FROM users WHERE "UserId" = 123 AND "UserStatus" = 'active'`,
			expectedPredicateColumns: []base.ColumnResource{
				{Database: "test_keyspace", Table: "users", Column: "UserId"},
				{Database: "test_keyspace", Table: "users", Column: "UserStatus"},
			},
			expectedTable: "users",
		},
		{
			name:      "UPDATE with WHERE clause",
			statement: "UPDATE users SET status = 'inactive' WHERE id = 123 AND age > 65",
			expectedPredicateColumns: []base.ColumnResource{
				{Database: "test_keyspace", Table: "users", Column: "id"},
				{Database: "test_keyspace", Table: "users", Column: "age"},
			},
			expectedTable: "users",
		},
		{
			name:      "DELETE with WHERE clause",
			statement: "DELETE FROM users WHERE status = 'deleted' AND created_at < 1000000",
			expectedPredicateColumns: []base.ColumnResource{
				{Database: "test_keyspace", Table: "users", Column: "status"},
				{Database: "test_keyspace", Table: "users", Column: "created_at"},
			},
			expectedTable: "users",
		},
		{
			name:      "UPDATE with IN clause",
			statement: "UPDATE products SET category = 'archived' WHERE id IN (1, 2, 3)",
			expectedPredicateColumns: []base.ColumnResource{
				{Database: "test_keyspace", Table: "products", Column: "id"},
			},
			expectedTable: "products",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gCtx := base.GetQuerySpanContext{}

			got, err := GetQuerySpan(ctx, gCtx, base.Statement{Text: tt.statement}, "test_keyspace", "", false)
			require.NoError(t, err)
			require.NotNil(t, got)

			// Check that all expected columns are in PredicateColumns
			for _, expectedCol := range tt.expectedPredicateColumns {
				require.Contains(t, got.PredicateColumns, expectedCol,
					"Missing predicate column: %+v", expectedCol)
			}

			// Verify SourceColumns contains table resource (not individual columns)
			tableResource := base.ColumnResource{
				Database: "test_keyspace",
				Table:    tt.expectedTable,
				Column:   "", // Empty column means table-level access
			}
			require.Contains(t, got.SourceColumns, tableResource,
				"SourceColumns should contain table resource")
		})
	}
}

func TestSourceColumnsTableResources(t *testing.T) {
	tests := []struct {
		name             string
		statement        string
		expectedResource base.ColumnResource
	}{
		{
			name:      "SELECT from single table",
			statement: "SELECT * FROM users",
			expectedResource: base.ColumnResource{
				Database: "test_keyspace",
				Table:    "users",
				Column:   "",
			},
		},
		{
			name:      "SELECT with keyspace.table",
			statement: "SELECT * FROM myapp.customers",
			expectedResource: base.ColumnResource{
				Database: "myapp",
				Table:    "customers",
				Column:   "",
			},
		},
		{
			name:      "INSERT statement",
			statement: "INSERT INTO products (id, name) VALUES (1, 'Widget')",
			expectedResource: base.ColumnResource{
				Database: "test_keyspace",
				Table:    "products",
				Column:   "",
			},
		},
		{
			name:      "UPDATE statement",
			statement: "UPDATE inventory SET quantity = 10 WHERE id = 1",
			expectedResource: base.ColumnResource{
				Database: "test_keyspace",
				Table:    "inventory",
				Column:   "",
			},
		},
		{
			name:      "DELETE statement",
			statement: "DELETE FROM orders WHERE id = 1",
			expectedResource: base.ColumnResource{
				Database: "test_keyspace",
				Table:    "orders",
				Column:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gCtx := base.GetQuerySpanContext{}

			got, err := GetQuerySpan(ctx, gCtx, base.Statement{Text: tt.statement}, "test_keyspace", "", false)
			require.NoError(t, err)
			require.NotNil(t, got)

			// Verify SourceColumns contains exactly the expected table resource
			require.Contains(t, got.SourceColumns, tt.expectedResource,
				"SourceColumns should contain table resource: %+v", tt.expectedResource)

			// Ensure no column-level resources are in SourceColumns
			for resource := range got.SourceColumns {
				require.Empty(t, resource.Column,
					"SourceColumns should only contain table resources (Column should be empty)")
			}
		})
	}
}

func TestQueryTypeDetection(t *testing.T) {
	tests := []struct {
		name         string
		statement    string
		expectedType base.QueryType
	}{
		// DQL (Data Query Language)
		{
			name:         "SELECT statement",
			statement:    "SELECT * FROM users",
			expectedType: base.Select,
		},
		// DML (Data Manipulation Language)
		{
			name:         "INSERT statement",
			statement:    "INSERT INTO users (id, name) VALUES (1, 'John')",
			expectedType: base.DML,
		},
		{
			name:         "UPDATE statement",
			statement:    "UPDATE users SET name = 'Jane' WHERE id = 1",
			expectedType: base.DML,
		},
		{
			name:         "DELETE statement",
			statement:    "DELETE FROM users WHERE id = 1",
			expectedType: base.DML,
		},
		// DDL (Data Definition Language) - Table operations
		{
			name:         "CREATE TABLE statement",
			statement:    "CREATE TABLE users (id uuid PRIMARY KEY, name text)",
			expectedType: base.DDL,
		},
		{
			name:         "ALTER TABLE statement",
			statement:    "ALTER TABLE users ADD email text",
			expectedType: base.DDL,
		},
		{
			name:         "DROP TABLE statement",
			statement:    "DROP TABLE users",
			expectedType: base.DDL,
		},
		// DDL - Keyspace operations
		{
			name:         "CREATE KEYSPACE statement",
			statement:    "CREATE KEYSPACE myapp WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}",
			expectedType: base.DDL,
		},
		{
			name:         "ALTER KEYSPACE statement",
			statement:    "ALTER KEYSPACE myapp WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 3}",
			expectedType: base.DDL,
		},
		{
			name:         "DROP KEYSPACE statement",
			statement:    "DROP KEYSPACE myapp",
			expectedType: base.DDL,
		},
		// DDL - Index operations
		{
			name:         "CREATE INDEX statement",
			statement:    "CREATE INDEX user_email_idx ON users (email)",
			expectedType: base.DDL,
		},
		{
			name:         "DROP INDEX statement",
			statement:    "DROP INDEX user_email_idx",
			expectedType: base.DDL,
		},
		// DDL - Materialized View operations
		{
			name:         "CREATE MATERIALIZED VIEW statement",
			statement:    "CREATE MATERIALIZED VIEW user_summary AS SELECT id, name FROM users WHERE id IS NOT NULL PRIMARY KEY (id)",
			expectedType: base.DDL,
		},
		{
			name:         "ALTER MATERIALIZED VIEW statement",
			statement:    "ALTER MATERIALIZED VIEW user_summary WITH compression = {'sstable_compression': 'LZ4Compressor'}",
			expectedType: base.DDL,
		},
		{
			name:         "DROP MATERIALIZED VIEW statement",
			statement:    "DROP MATERIALIZED VIEW user_summary",
			expectedType: base.DDL,
		},
		// DDL - Type operations
		{
			name:         "CREATE TYPE statement",
			statement:    "CREATE TYPE address (street text, city text, zip text)",
			expectedType: base.DDL,
		},
		{
			name:         "ALTER TYPE statement",
			statement:    "ALTER TYPE address ADD country text",
			expectedType: base.DDL,
		},
		{
			name:         "DROP TYPE statement",
			statement:    "DROP TYPE address",
			expectedType: base.DDL,
		},
		// DDL - Function operations
		{
			name:         "CREATE FUNCTION statement",
			statement:    "CREATE FUNCTION myfunction(val int) RETURNS NULL ON NULL INPUT RETURNS int LANGUAGE java AS 'return val * 2;'",
			expectedType: base.DDL,
		},
		{
			name:         "DROP FUNCTION statement",
			statement:    "DROP FUNCTION myfunction",
			expectedType: base.DDL,
		},
		// DDL - Trigger operations
		{
			name:         "CREATE TRIGGER statement",
			statement:    "CREATE TRIGGER mytrigger USING 'org.apache.cassandra.triggers.AuditTrigger'",
			expectedType: base.DDL,
		},
		{
			name:         "DROP TRIGGER statement",
			statement:    "DROP TRIGGER mytrigger ON users",
			expectedType: base.DDL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gCtx := base.GetQuerySpanContext{}

			got, err := GetQuerySpan(ctx, gCtx, base.Statement{Text: tt.statement}, "test_keyspace", "", false)
			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, tt.expectedType, got.Type, "Query type mismatch for %s", tt.statement)
		})
	}
}

func TestSelectAsteriskWithMetadata(t *testing.T) {
	tests := []struct {
		name         string
		statement    string
		hasMetadata  bool
		expectExpand bool
	}{
		{
			name:         "SELECT * without metadata function",
			statement:    "SELECT * FROM users",
			hasMetadata:  false,
			expectExpand: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gCtx := base.GetQuerySpanContext{}
			// Note: tt.hasMetadata would be used to set up mock metadata function
			// For now, we're testing the fallback behavior without metadata

			got, err := GetQuerySpan(ctx, gCtx, base.Statement{Text: tt.statement}, "test_keyspace", "", false)
			require.NoError(t, err)
			require.NotNil(t, got)

			if !tt.expectExpand && len(got.Results) > 0 {
				// Should have SelectAsterisk flag set
				require.True(t, got.Results[0].SelectAsterisk, "Expected SelectAsterisk flag to be true")
				require.Empty(t, got.Results[0].Name, "Expected empty name for SELECT *")
			}
		})
	}
}
