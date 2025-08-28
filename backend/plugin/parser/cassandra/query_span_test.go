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

			got, err := GetQuerySpan(ctx, gCtx, tt.statement, "test_keyspace", "", false)
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

			// Check that SourceColumns at QuerySpan level contains all columns from Results
			if !got.Results[0].SelectAsterisk {
				for _, result := range got.Results {
					for col := range result.SourceColumns {
						require.Contains(t, got.SourceColumns, col, "Missing column in QuerySpan.SourceColumns: %+v", col)
					}
				}
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

			_, err := GetQuerySpan(ctx, gCtx, tt.statement, "test_keyspace", "", false)
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
	}{
		{
			name:      "Simple equality WHERE clause",
			statement: "SELECT * FROM users WHERE id = 123",
			expectedPredicateColumns: []base.ColumnResource{
				{Database: "test_keyspace", Table: "users", Column: "id"},
			},
		},
		{
			name:      "Multiple AND conditions",
			statement: "SELECT name FROM users WHERE id = 123 AND status = 'active'",
			expectedPredicateColumns: []base.ColumnResource{
				{Database: "test_keyspace", Table: "users", Column: "id"},
				{Database: "test_keyspace", Table: "users", Column: "status"},
			},
		},
		{
			name:      "WHERE with comparison operators",
			statement: "SELECT * FROM users WHERE age > 18 AND age < 65",
			expectedPredicateColumns: []base.ColumnResource{
				{Database: "test_keyspace", Table: "users", Column: "age"},
			},
		},
		{
			name:      "WHERE with IN clause",
			statement: "SELECT * FROM users WHERE status IN ('active', 'pending')",
			expectedPredicateColumns: []base.ColumnResource{
				{Database: "test_keyspace", Table: "users", Column: "status"},
			},
		},
		{
			name:      "WHERE with double-quoted columns",
			statement: `SELECT * FROM users WHERE "UserId" = 123 AND "UserStatus" = 'active'`,
			expectedPredicateColumns: []base.ColumnResource{
				{Database: "test_keyspace", Table: "users", Column: "UserId"},
				{Database: "test_keyspace", Table: "users", Column: "UserStatus"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gCtx := base.GetQuerySpanContext{}

			got, err := GetQuerySpan(ctx, gCtx, tt.statement, "test_keyspace", "", false)
			require.NoError(t, err)
			require.NotNil(t, got)

			// Check that all expected columns are in PredicateColumns
			for _, expectedCol := range tt.expectedPredicateColumns {
				require.Contains(t, got.PredicateColumns, expectedCol,
					"Missing predicate column: %+v", expectedCol)
				// Also verify they're in SourceColumns
				require.Contains(t, got.SourceColumns, expectedCol,
					"Missing source column: %+v", expectedCol)
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
		{
			name:         "SELECT statement",
			statement:    "SELECT * FROM users",
			expectedType: base.Select,
		},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gCtx := base.GetQuerySpanContext{}

			got, err := GetQuerySpan(ctx, gCtx, tt.statement, "test_keyspace", "", false)
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

			got, err := GetQuerySpan(ctx, gCtx, tt.statement, "test_keyspace", "", false)
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
