package v2

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestGetQuerySpan(t *testing.T) {
	const (
		defaultDatabase = "db"
	)

	var (
		mockDatabaseMetadataGetter = func(_ context.Context, databaseName string) (*model.DatabaseMetadata, error) {
			databaseMetadata := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
				Name: defaultDatabase,
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "t",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "a",
									},
									{
										Name: "b",
									},
									{
										Name: "c",
									},
									{
										Name: "d",
									},
								},
							},
						},
					},
				},
			})
			if databaseName == defaultDatabase {
				return databaseMetadata, nil
			}
			return nil, errors.Errorf("database %q not found", databaseName)
		}
	)
	type testCase struct {
		statement string
		want      *base.QuerySpan
	}

	testCases := []testCase{
		{
			statement: "SELECT * FROM t",
			want: &base.QuerySpan{
				Results: []*base.QuerySpanResult{
					{
						Name: "a",
						SourceColumns: base.SourceColumnSet{
							{
								Database: defaultDatabase,
								Schema:   "public",
								Table:    "t",
								Column:   "a",
							}: true,
						},
					},
					{
						Name: "b",
						SourceColumns: base.SourceColumnSet{
							{
								Database: defaultDatabase,
								Schema:   "public",
								Table:    "t",
								Column:   "b",
							}: true,
						},
					},
					{
						Name: "c",
						SourceColumns: base.SourceColumnSet{
							{
								Database: defaultDatabase,
								Schema:   "public",
								Table:    "t",
								Column:   "c",
							}: true,
						},
					},
					{
						Name: "d",
						SourceColumns: base.SourceColumnSet{
							{
								Database: defaultDatabase,
								Schema:   "public",
								Table:    "t",
								Column:   "d",
							}: true,
						},
					},
				},
			},
		},
	}

	a := require.New(t)

	for _, tc := range testCases {
		got, err := GetQuerySpan(context.Background(), tc.statement, defaultDatabase, mockDatabaseMetadataGetter)
		if err != nil {
			t.Errorf("GetQuerySpan(%q) got error: %v", tc.statement, err)
		}

		a.Equal(tc.want, got)
	}
}
