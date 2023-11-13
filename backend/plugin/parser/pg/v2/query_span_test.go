package v2

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v2"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestGetQuerySpanResult(t *testing.T) {
	type testCase struct {
		Description       string `yaml:"description"`
		Statement         string `yaml:"statement"`
		ConnectedDatabase string `yaml:"connectedDatabase"`
		// Metadata is the protojson encoded storepb.DatabaseSchemaMetadata,
		// if it's empty, we will use the defaultDatabaseMetadata.
		Metadata   string                  `yaml:"metadata"`
		SpanResult []*base.QuerySpanResult `yaml:"spanResult"`
	}

	const (
		record       = true
		testDataPath = "testdata/query_span_result.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(testDataPath)
	a.NoError(err)

	var testCases []testCase
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(err)
	a.NoError(yamlFile.Close())
	a.NoError(yaml.Unmarshal(byteValue, &testCases))

	for i, tc := range testCases {
		metadata := &storepb.DatabaseSchemaMetadata{}
		a.NoError(protojson.Unmarshal([]byte(tc.Metadata), metadata))
		databaseMetadataGetter := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
		result, err := GetQuerySpan(context.TODO(), tc.Statement, tc.ConnectedDatabase, databaseMetadataGetter)
		a.NoError(err)
		if record {
			testCases[i].SpanResult = result.Results
		} else {
			a.Equal(tc.SpanResult, result.Results)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(testCases)
		a.NoError(err)
		err = os.WriteFile(testDataPath, byteValue, 0644)
		a.NoError(err)
	}
}

func buildMockDatabaseMetadataGetter(databaseMetadata []*storepb.DatabaseSchemaMetadata) base.GetDatabaseMetadataFunc {
	return func(_ context.Context, databaseName string) (*model.DatabaseMetadata, error) {
		m := make(map[string]*model.DatabaseMetadata)
		for _, metadata := range databaseMetadata {
			m[metadata.Name] = model.NewDatabaseMetadata(metadata)
		}

		if databaseMetadata, ok := m[databaseName]; ok {
			return databaseMetadata, nil
		}

		return nil, errors.Errorf("database %q not found", databaseName)
	}
}

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
