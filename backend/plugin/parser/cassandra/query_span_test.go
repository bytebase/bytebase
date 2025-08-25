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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gCtx := base.GetQuerySpanContext{}

			got, err := GetQuerySpan(ctx, gCtx, tt.statement, "test_keyspace", "", false)
			require.NoError(t, err)
			require.NotNil(t, got)

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
