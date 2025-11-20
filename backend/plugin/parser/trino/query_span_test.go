package trino

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// CustomQueryType allows for both string and int representations of query types in YAML
type CustomQueryType struct {
	base.QueryType
}

// NestedQueryType is a helper type for the nested structure in the YAML
type NestedQueryType struct {
	QueryType int `yaml:"querytype"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (c *CustomQueryType) UnmarshalYAML(value *yaml.Node) error {
	// Check for scalar first
	if value.Kind == yaml.ScalarNode {
		var strValue string
		if err := value.Decode(&strValue); err == nil {
			// Convert string QueryType to int
			switch strings.ToUpper(strValue) {
			case "SELECT":
				c.QueryType = base.Select
			case "EXPLAIN":
				c.QueryType = base.Explain
			case "SELECT_INFO_SCHEMA":
				c.QueryType = base.SelectInfoSchema
			case "DDL":
				c.QueryType = base.DDL
			case "DML":
				c.QueryType = base.DML
			default:
				c.QueryType = base.QueryTypeUnknown
			}
			return nil
		}

		// Try to unmarshal as int
		var intValue int
		if err := value.Decode(&intValue); err == nil {
			c.QueryType = base.QueryType(intValue)
			return nil
		}
	}

	// Handle nested mapping node
	if value.Kind == yaml.MappingNode {
		var nested NestedQueryType
		if err := value.Decode(&nested); err == nil {
			c.QueryType = base.QueryType(nested.QueryType)
			return nil
		}
	}

	return errors.New("unable to decode QueryType")
}

// CustomYamlQuerySpan mimics base.YamlQuerySpan but with custom QueryType
type CustomYamlQuerySpan struct {
	Type             CustomQueryType             `yaml:"type"`
	Results          []CustomYamlQuerySpanResult `yaml:"results"`
	SourceColumns    map[string]bool             `yaml:"sourceColumns"`
	PredicateColumns map[string]bool             `yaml:"predicateColumns"`
}

// CustomYamlQuerySpanResult mimics base.YamlQuerySpanResult but with maps for source columns
type CustomYamlQuerySpanResult struct {
	Name             string          `yaml:"name"`
	SourceColumns    map[string]bool `yaml:"sourceColumns"`
	IsPlainField     bool            `yaml:"isPlainField"`
	SourceFieldPaths []any           `yaml:"sourceFieldPaths"`
	SelectAsterisk   bool            `yaml:"selectAsterisk"`
}

// ToBaseYamlQuerySpan converts CustomYamlQuerySpan to base.YamlQuerySpan
func (c *CustomYamlQuerySpan) ToBaseYamlQuerySpan() *base.YamlQuerySpan {
	result := &base.YamlQuerySpan{
		Type:             c.Type.QueryType,
		Results:          []base.YamlQuerySpanResult{},
		SourceColumns:    []base.ColumnResource{},
		PredicateColumns: []base.ColumnResource{},
	}

	for _, r := range c.Results {
		baseResult := base.YamlQuerySpanResult{
			Name:             r.Name,
			SourceColumns:    []base.ColumnResource{},
			IsPlainField:     r.IsPlainField,
			SelectAsterisk:   r.SelectAsterisk,
			SourceFieldPaths: []base.YamlQuerySpanResultSourceFieldPaths{},
		}
		result.Results = append(result.Results, baseResult)
	}

	return result
}

func TestGetQuerySpanFromYAML(t *testing.T) {
	type testCase struct {
		Description        string `yaml:"description,omitempty"`
		Statement          string `yaml:"statement,omitempty"`
		DefaultDatabase    string `yaml:"defaultDatabase,omitempty"`
		IgnoreCaseSensitve bool   `yaml:"ignoreCaseSensitive,omitempty"`
		// Metadata is the protojson encoded storepb.DatabaseSchemaMetadata,
		// if it's empty, we will use the defaultDatabaseMetadata.
		Metadata  string               `yaml:"metadata,omitempty"`
		QuerySpan *CustomYamlQuerySpan `yaml:"querySpan,omitempty"`
	}

	var (
		record        = false
		testDataPaths = []string{
			"test-data/query-span/query_type.yaml",
			"test-data/query-span/standard.yaml",
			"test-data/query-span/case-sensitivity.yaml",
			"test-data/query-span/join.yaml",
			"test-data/query-span/trino-specific.yaml",
			"test-data/query-span/masking.yaml",
		}
	)

	a := require.New(t)
	for _, testDataPath := range testDataPaths {
		testDataPath := testDataPath

		yamlFile, err := os.Open(testDataPath)
		a.NoError(err)

		var testCases []testCase
		byteValue, err := io.ReadAll(yamlFile)
		a.NoError(err)
		a.NoError(yamlFile.Close())
		a.NoError(yaml.Unmarshal(byteValue, &testCases))

		for i, tc := range testCases {
			metadata := &storepb.DatabaseSchemaMetadata{}
			a.NoErrorf(common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Metadata), metadata), "cases %d", i+1)
			databaseMetadataGetter, databaseNameLister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
			result, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
				GetDatabaseMetadataFunc: databaseMetadataGetter,
				ListDatabaseNamesFunc:   databaseNameLister,
				Engine:                  storepb.Engine_TRINO,
			}, tc.Statement, tc.DefaultDatabase, "", tc.IgnoreCaseSensitve)
			a.NoErrorf(err, "statement: %s", tc.Statement)

			// Verify the query type is correct
			if tc.QuerySpan != nil {
				a.Equal(tc.QuerySpan.Type.QueryType, result.Type,
					"Query type mismatch for statement: %s", tc.Statement)
			}

			if record {
				// When recording, use the standard ToYaml result
				resultYaml := result.ToYaml()

				// Convert to our custom type for YAML serialization
				custom := &CustomYamlQuerySpan{
					Type:          CustomQueryType{QueryType: resultYaml.Type},
					SourceColumns: make(map[string]bool),
					Results:       []CustomYamlQuerySpanResult{},
				}

				// Convert source columns to map for YAML
				for _, col := range resultYaml.SourceColumns {
					custom.SourceColumns[col.String()] = true
				}

				// Convert results
				for _, r := range resultYaml.Results {
					customResult := CustomYamlQuerySpanResult{
						Name:           r.Name,
						IsPlainField:   r.IsPlainField,
						SelectAsterisk: r.SelectAsterisk,
						SourceColumns:  make(map[string]bool),
					}
					for _, col := range r.SourceColumns {
						customResult.SourceColumns[col.String()] = true
					}
					custom.Results = append(custom.Results, customResult)
				}

				testCases[i].QuerySpan = custom
			}
		}

		if record {
			byteValue, err := yaml.Marshal(testCases)
			a.NoError(err)
			err = os.WriteFile(testDataPath, byteValue, 0644)
			a.NoError(err)
		}
	}
}

func buildMockDatabaseMetadataGetter(databaseMetadata []*storepb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			m := make(map[string]*model.DatabaseMetadata)
			for _, metadata := range databaseMetadata {
				m[metadata.Name] = model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true /* isObjectCaseSensitive */)
			}

			if databaseMetadata, ok := m[databaseName]; ok {
				return "", databaseMetadata, nil
			}

			return "", nil, errors.Errorf("database %q not found", databaseName)
		}, func(_ context.Context, _ string) ([]string, error) {
			var names []string
			for _, metadata := range databaseMetadata {
				names = append(names, metadata.Name)
			}
			return names, nil
		}
}
