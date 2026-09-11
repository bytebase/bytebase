package tidb

import (
	"context"
	"io"
	"os"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
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
		Description     string `yaml:"description,omitempty"`
		Statement       string `yaml:"statement,omitempty"`
		DefaultDatabase string `yaml:"defaultDatabase,omitempty"`
		// Metadata is the protojson encoded metadatapb.DatabaseSchemaMetadata,
		// if it's empty, we will use the defaultDatabaseMetadata.
		Metadata  string              `yaml:"metadata,omitempty"`
		QuerySpan *base.YamlQuerySpan `yaml:"querySpan,omitempty"`
	}

	const (
		record = false
	)

	var (
		testDataPaths = []string{
			"test-data/query_span.yaml",
			"test-data/query_type.yaml",
		}
	)

	a := require.New(t)

	for _, testDataPath := range testDataPaths {
		yamlFile, err := os.Open(testDataPath)
		a.NoError(err)

		var testCases []testCase
		byteValue, err := io.ReadAll(yamlFile)
		a.NoError(err)
		a.NoError(yamlFile.Close())
		a.NoError(yaml.Unmarshal(byteValue, &testCases))

		for i, tc := range testCases {
			metadata := &metadatapb.DatabaseSchemaMetadata{}
			a.NoError(common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Metadata), metadata))
			databaseMetadataGetter, databaseNamesLister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{metadata})
			result, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{
				GetDatabaseMetadataFunc: databaseMetadataGetter,
				ListDatabaseNamesFunc:   databaseNamesLister,
			}, base.Statement{Text: tc.Statement}, tc.DefaultDatabase, "", false)
			a.NoError(err)
			resultYaml := result.ToYaml()
			if record {
				testCases[i].QuerySpan = resultYaml
			} else {
				a.Equal(tc.QuerySpan, resultYaml, "statement: %s", tc.Statement)
			}
		}

		if record {
			yamltest.Record(t, testDataPath, testCases)
		}
	}
}

// When a referenced column is missing from cached metadata, the extractor must
// surface ResourceNotFoundError on the span (not as a top-level error) so the
// SQL service's resync+retry path can recover.
func TestGetQuerySpanStaleMetadataReturnsNotFoundError(t *testing.T) {
	a := require.New(t)

	// Metadata omits distribute_level to mimic a stale cache after out-of-band ALTER TABLE ADD COLUMN.
	staleMetadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "cif",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "byt9385_repro",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id"},
							{Name: "existing_col"},
							{Name: "create_time"},
						},
					},
				},
			},
		},
	}
	databaseMetadataGetter, databaseNamesLister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{staleMetadata})

	span, err := GetQuerySpan(
		context.TODO(),
		base.GetQuerySpanContext{
			GetDatabaseMetadataFunc: databaseMetadataGetter,
			ListDatabaseNamesFunc:   databaseNamesLister,
		},
		base.Statement{Text: "SELECT distribute_level FROM byt9385_repro ORDER BY create_time DESC"},
		"cif",
		"",
		false,
	)
	a.NoError(err, "expected stale-metadata case to return a span, not a wrapped error")
	a.NotNil(span)
	a.NotNil(span.NotFoundError, "stale-metadata case must populate span.NotFoundError so sql_service can resync+retry")

	var resourceNotFound *base.ResourceNotFoundError
	a.True(errors.As(span.NotFoundError, &resourceNotFound))
	a.NotNil(resourceNotFound.Column)
	a.Equal("distribute_level", *resourceNotFound.Column)

	// SourceColumns must reference the FROM table so sql_service knows which database to resync.
	foundTable := false
	for k := range span.SourceColumns {
		if k.Database == "cif" && k.Table == "byt9385_repro" {
			foundTable = true
		}
		// optionalAccessCheck validates every SourceColumns entry against table-scoped
		// CEL policies; a synthesized table-less entry would deny least-privilege users
		// on the recovery path even when the FROM table is already authorized.
		a.NotEmpty(k.Table, "SourceColumns must not contain table-less entries when the FROM table is known: got %+v", k)
	}
	a.True(foundTable, "span.SourceColumns must reference the FROM table for resync to target the right database")
}

// When the referenced table is missing from cached metadata, accessTables is
// empty; SourceColumns must still expose a sync target so resync+retry can run.
func TestGetQuerySpanMissingTableFallsBackToNotFoundDatabase(t *testing.T) {
	a := require.New(t)

	// Metadata is missing the table entirely.
	staleMetadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "cif",
		Schemas: []*metadatapb.SchemaMetadata{
			{Name: "", Tables: []*metadatapb.TableMetadata{}},
		},
	}
	databaseMetadataGetter, databaseNamesLister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{staleMetadata})

	span, err := GetQuerySpan(
		context.TODO(),
		base.GetQuerySpanContext{
			GetDatabaseMetadataFunc: databaseMetadataGetter,
			ListDatabaseNamesFunc:   databaseNamesLister,
		},
		base.Statement{Text: "SELECT name FROM byt9385_caseB"},
		"cif",
		"",
		false,
	)
	a.NoError(err)
	a.NotNil(span)
	a.NotNil(span.NotFoundError, "missing-table case must populate span.NotFoundError")

	var resourceNotFound *base.ResourceNotFoundError
	a.True(errors.As(span.NotFoundError, &resourceNotFound))
	a.NotNil(resourceNotFound.Table)
	a.Equal("byt9385_caseB", *resourceNotFound.Table)

	a.NotEmpty(span.SourceColumns, "SourceColumns must not be empty — sql_service iterates it to build the resync target list")
	foundDatabase := false
	for k := range span.SourceColumns {
		if k.Database == "cif" {
			foundDatabase = true
			break
		}
	}
	a.True(foundDatabase, "resync target must reference the database where the missing table lives")
}

// SourceColumns must include the not-found target alongside known tables — the
// pre-execute ACL check isn't rerun after resync+retry, so dropping the missing
// resource would let an out-of-band CREATE TABLE bypass table-level ACL.
func TestGetQuerySpanMissingTableUnionedWithAccessTables(t *testing.T) {
	a := require.New(t)

	// Metadata has the "known" table but not the "unknown" one.
	staleMetadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "cif",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "",
				Tables: []*metadatapb.TableMetadata{
					{
						Name:    "byt9385_known",
						Columns: []*metadatapb.ColumnMetadata{{Name: "a"}},
					},
				},
			},
		},
	}
	databaseMetadataGetter, databaseNamesLister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{staleMetadata})

	span, err := GetQuerySpan(
		context.TODO(),
		base.GetQuerySpanContext{
			GetDatabaseMetadataFunc: databaseMetadataGetter,
			ListDatabaseNamesFunc:   databaseNamesLister,
		},
		base.Statement{Text: "SELECT a FROM byt9385_known UNION SELECT a FROM byt9385_unknown"},
		"cif",
		"",
		false,
	)
	a.NoError(err)
	a.NotNil(span)
	a.NotNil(span.NotFoundError)

	foundKnown, foundUnknown := false, false
	for k := range span.SourceColumns {
		if k.Database == "cif" && k.Table == "byt9385_known" {
			foundKnown = true
		}
		if k.Database == "cif" && k.Table == "byt9385_unknown" {
			foundUnknown = true
		}
	}
	a.True(foundKnown, "known table must remain in SourceColumns")
	a.True(foundUnknown, "not-found table must be unioned into SourceColumns so the pre-execute ACL check sees it")
}

// TestGetQuerySpanCyclicViewReference pins that a cyclic view reference is
// reported as an error rather than recursing until the stack overflows.
// Ported from the MySQL guard (#20153).
func TestGetQuerySpanCyclicViewReference(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "",
				Views: []*metadatapb.ViewMetadata{
					{Name: "v1", Definition: "SELECT * FROM v2"},
					{Name: "v2", Definition: "SELECT * FROM v1"},
				},
			},
		},
	}
	databaseMetadataGetter, databaseNamesLister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{metadata})

	_, err := GetQuerySpan(
		context.TODO(),
		base.GetQuerySpanContext{
			GetDatabaseMetadataFunc: databaseMetadataGetter,
			ListDatabaseNamesFunc:   databaseNamesLister,
		},
		base.Statement{Text: "SELECT * FROM v1"},
		"db",
		"",
		false,
	)
	require.Error(t, err)
	require.ErrorContains(t, err, "cyclic view reference")
}

func buildMockDatabaseMetadataGetter(databaseMetadata []*metadatapb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			m := make(map[string]*model.DatabaseMetadata)
			for _, metadata := range databaseMetadata {
				m[metadata.Name] = model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_TIDB, false /* isObjectCaseSensitive */)
			}

			if databaseMetadata, ok := m[databaseName]; ok {
				return databaseName, databaseMetadata, nil
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
