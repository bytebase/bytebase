package snowflake

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestGetQuerySpan(t *testing.T) {
	type testCase struct {
		Description        string `yaml:"description,omitempty"`
		Statement          string `yaml:"statement,omitempty"`
		DefaultDatabase    string `yaml:"defaultDatabase,omitempty"`
		IgnoreCaseSensitve bool   `yaml:"ignoreCaseSensitive,omitempty"`
		// Metadata is the protojson encoded storepb.DatabaseSchemaMetadata,
		// if it's empty, we will use the defaultDatabaseMetadata.
		Metadata  string              `yaml:"metadata,omitempty"`
		QuerySpan *base.YamlQuerySpan `yaml:"querySpan,omitempty"`
	}

	var (
		record        = false
		testDataPaths = []string{
			"test-data/query-span/standard.yaml",
			"test-data/query-span/subquery.yaml",
			"test-data/query-span/set-operator.yaml",
			// TODO: re-enable once pivot lineage is implemented on the omni AST.
			// omni models PIVOT/UNPIVOT on ast.TableRef (Pivot/Unpivot/Nested), but
			// this extractor does not compute the pivoted projection yet — it FAILS
			// CLOSED instead (see extractTableSourceFromTableRef and
			// TestGetQuerySpan_PivotFailsClosed): GetQuerySpan returns an explicit
			// error for pivoted table sources rather than silently resolving the
			// bare base table with wrong lineage. The expected lineage assertions
			// are preserved in pivot.yaml for when pivot projection lands. Tracked
			// in the migration ledger.
			// "test-data/query-span/pivot.yaml",
			"test-data/query-span/cte.yaml",
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
			}, base.Statement{Text: tc.Statement}, tc.DefaultDatabase, "PUBLIC", tc.IgnoreCaseSensitve)
			a.NoErrorf(err, "statement: %s", tc.Statement)
			resultYaml := result.ToYaml()
			if record {
				testCases[i].QuerySpan = resultYaml
			} else {
				a.Equalf(tc.QuerySpan, resultYaml, "statement: %s", tc.Statement)
			}
		}

		if record {
			yamltest.Record(t, testDataPath, testCases)
		}
	}
}

func buildMockDatabaseMetadataGetter(databaseMetadata []*storepb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			m := make(map[string]*model.DatabaseMetadata)
			for _, metadata := range databaseMetadata {
				m[metadata.Name] = model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_SNOWFLAKE, true /* isObjectCaseSensitive */)
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

// TestGetQuerySpan_PivotFailsClosed locks the PIVOT/UNPIVOT fail-closed
// behavior: until pivot projection lineage is implemented, GetQuerySpan must
// return an explicit error for pivoted table sources — never silently resolve
// the bare base table (which would yield wrong lineage/positions for masking).
func TestGetQuerySpan_PivotFailsClosed(t *testing.T) {
	a := require.New(t)
	metadata := &storepb.DatabaseSchemaMetadata{Name: "DB1"}
	databaseMetadataGetter, databaseNameLister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
	for _, sql := range []string{
		`SELECT * FROM monthly_sales PIVOT (SUM(amount) FOR month IN ('JAN', 'FEB')) AS p;`,
		`SELECT * FROM monthly_sales UNPIVOT (sales FOR month IN (jan, feb)) AS u;`,
	} {
		_, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
			GetDatabaseMetadataFunc: databaseMetadataGetter,
			ListDatabaseNamesFunc:   databaseNameLister,
		}, base.Statement{Text: sql}, "DB1", "PUBLIC", false)
		a.Errorf(err, "expected fail-closed error, statement: %s", sql)
		a.Containsf(err.Error(), "PIVOT", "statement: %s", sql)
	}
}

// TestGetQuerySpan_ShowPipeFailsClosed locks the fail-closed posture for
// SHOW result-pipes: the trailing query reads $1 (unresolvable schema), so
// span extraction errors explicitly instead of resolving wrong lineage or
// passing as metadata-only.
func TestGetQuerySpan_ShowPipeFailsClosed(t *testing.T) {
	a := require.New(t)
	metadata := &storepb.DatabaseSchemaMetadata{Name: "DB1"}
	databaseMetadataGetter, databaseNameLister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
	_, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: databaseMetadataGetter,
		ListDatabaseNamesFunc:   databaseNameLister,
	}, base.Statement{Text: `SHOW TABLES ->> SELECT * FROM SENSITIVE_T;`}, "DB1", "PUBLIC", false)
	a.Error(err)
	a.Contains(err.Error(), "result-pipe")
}
