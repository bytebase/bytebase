// Package googlesqltest holds the test harness shared by the bigquery and
// spanner engine packages: the query-span differential-corpus runner (the
// corpora are recorded FROM the legacy ANTLR resolvers and must be reproduced,
// never re-recorded against the new implementation) and the mock metadata
// plumbing the leak-pin tests use.
package googlesqltest

import (
	"context"
	"io"
	"os"
	"slices"
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

// RunQuerySpanCorpus runs the legacy-recorded query-span differential corpus at
// testDataPath against getQuerySpan. With record=true it re-records the corpus
// from the CURRENT implementation — only ever do that from a legacy worktree;
// the goldens are the legacy resolvers' outputs and are the masking-parity bar.
func RunQuerySpanCorpus(t *testing.T, engine storepb.Engine, getQuerySpan base.GetQuerySpanFunc, testDataPath string, record bool) {
	type testCase struct {
		Description        string `yaml:"description,omitempty"`
		Statement          string `yaml:"statement,omitempty"`
		DefaultDatabase    string `yaml:"defaultDatabase,omitempty"`
		IgnoreCaseSensitve bool   `yaml:"ignoreCaseSensitive,omitempty"`
		// Metadata is the protojson encoded storepb.DatabaseSchemaMetadata.
		Metadata  string              `yaml:"metadata,omitempty"`
		QuerySpan *base.YamlQuerySpan `yaml:"querySpan,omitempty"`
	}

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
		a.NoErrorf(common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Metadata), metadata), "cases %d", i+1)
		getter, lister := BuildMockDatabaseMetadataGetter(engine, []*storepb.DatabaseSchemaMetadata{metadata})
		result, err := getQuerySpan(context.TODO(), base.GetQuerySpanContext{
			GetDatabaseMetadataFunc: getter,
			ListDatabaseNamesFunc:   lister,
		}, base.Statement{Text: tc.Statement}, tc.DefaultDatabase, "dbo", tc.IgnoreCaseSensitve)
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

// BuildMockDatabaseMetadataGetter builds the metadata getter/lister pair the
// query-span tests use, keyed by database name.
func BuildMockDatabaseMetadataGetter(engine storepb.Engine, databaseMetadata []*storepb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			m := make(map[string]*model.DatabaseMetadata)
			for _, metadata := range databaseMetadata {
				m[metadata.Name] = model.NewDatabaseMetadata(metadata, nil, nil, engine, true /* isObjectCaseSensitive */)
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

// GetSpan evaluates one statement's query span against the given metadata —
// the leak-pin tests' entry point.
func GetSpan(t *testing.T, engine storepb.Engine, getQuerySpan base.GetQuerySpanFunc, statement, defaultDatabase string, schemas []*storepb.SchemaMetadata) (*base.QuerySpan, error) {
	t.Helper()
	meta := &storepb.DatabaseSchemaMetadata{Name: defaultDatabase, Schemas: schemas}
	getter, lister := BuildMockDatabaseMetadataGetter(engine, []*storepb.DatabaseSchemaMetadata{meta})
	return getQuerySpan(context.TODO(), base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
	}, base.Statement{Text: statement}, defaultDatabase, "", false)
}

// DefaultSchemaTables wraps tables into a single default ("") schema.
func DefaultSchemaTables(tables ...*storepb.TableMetadata) []*storepb.SchemaMetadata {
	return []*storepb.SchemaMetadata{{Name: "", Tables: tables}}
}

// SourcesOf renders a result's source columns as sorted "[schema.]table.column"
// strings for compact assertions.
func SourcesOf(r base.QuerySpanResult) []string {
	out := make([]string, 0, len(r.SourceColumns))
	for c := range r.SourceColumns {
		name := c.Table + "." + c.Column
		if c.Schema != "" {
			name = c.Schema + "." + name
		}
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}
