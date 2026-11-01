package mariadb

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// querySpanGoldenFixturePaths is the YAML corpus the harness below compares
// Bytebase's query span against.
var querySpanGoldenFixturePaths = []string{
	"test-data/query-span/query_type.yaml",
	"test-data/query-span/standard.yaml",
	"test-data/query-span/case_insensitive.yaml",
}

// firstLineOf shortens a statement for a failure message.
func firstLineOf(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + " ..."
	}
	return s
}

// TestMySQLOmniQuerySpanGoldenHarness holds this package's query span to the
// shared YAML corpus. It drives the extractor directly rather than through
// GetQuerySpan, so a span is compared against the fixture's expectation and not
// against itself.
func TestMySQLOmniQuerySpanGoldenHarness(t *testing.T) {
	type testCase struct {
		Description        string              `yaml:"description,omitempty"`
		Statement          string              `yaml:"statement,omitempty"`
		DefaultDatabase    string              `yaml:"defaultDatabase,omitempty"`
		IgnoreCaseSensitve bool                `yaml:"ignoreCaseSensitive,omitempty"`
		Metadata           string              `yaml:"metadata,omitempty"`
		QuerySpan          *base.YamlQuerySpan `yaml:"querySpan,omitempty"`
	}

	type diff struct {
		fixture     string
		index       int
		description string
		statement   string
		details     string
	}

	var diffs []diff
	total := 0
	for _, testDataPath := range querySpanGoldenFixturePaths {
		engine := storepb.Engine_MARIADB

		yamlFile, err := os.Open(testDataPath)
		if err != nil {
			t.Fatalf("open %s: %v", testDataPath, err)
		}
		byteValue, err := io.ReadAll(yamlFile)
		if closeErr := yamlFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			t.Fatalf("read %s: %v", testDataPath, err)
		}

		var testCases []testCase
		if err := yaml.Unmarshal(byteValue, &testCases); err != nil {
			t.Fatalf("yaml %s: %v", testDataPath, err)
		}

		for i, tc := range testCases {
			if strings.TrimSpace(tc.Statement) == "" {
				continue
			}
			total++

			omni, err := runMySQLOmniGoldenCase(context.Background(), tc.Statement, tc.Metadata, tc.DefaultDatabase, engine, tc.IgnoreCaseSensitve)
			if err != nil {
				diffs = append(diffs, diff{
					fixture:     testDataPath,
					index:       i,
					description: tc.Description,
					statement:   tc.Statement,
					details:     err.Error(),
				})
				continue
			}
			if !reflect.DeepEqual(tc.QuerySpan, omni) {
				diffs = append(diffs, diff{
					fixture:     testDataPath,
					index:       i,
					description: tc.Description,
					statement:   tc.Statement,
					details:     fmt.Sprintf("want=%+v omni=%+v", tc.QuerySpan, omni),
				})
			}
		}
	}

	t.Logf("MySQL omni query-span golden: %d/%d matched, %d diffs", total-len(diffs), total, len(diffs))
	for i, d := range diffs {
		if i >= 10 {
			t.Logf("... %d more diffs omitted", len(diffs)-i)
			break
		}
		t.Logf("[%s case %d %q] %s\n  SQL: %s", d.fixture, d.index, d.description, d.details, firstLineOf(d.statement))
	}
	require.Empty(t, diffs)
}

func runMySQLOmniGoldenCase(
	ctx context.Context,
	statement string,
	metadataText string,
	defaultDatabase string,
	engine storepb.Engine,
	ignoreCaseSensitive bool,
) (*base.YamlQuerySpan, error) {
	metadata := &metadatapb.DatabaseSchemaMetadata{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(metadataText), metadata); err != nil {
		return nil, err
	}
	databaseMetadataGetter, databaseNameLister := buildMockDatabaseMetadataGetter([]*metadatapb.DatabaseSchemaMetadata{metadata})
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: databaseMetadataGetter,
		ListDatabaseNamesFunc:   databaseNameLister,
		Engine:                  engine,
	}

	omni, err := newOmniQuerySpanExtractor(defaultDatabase, gCtx, ignoreCaseSensitive).getOmniQuerySpan(ctx, statement)
	if err != nil {
		return nil, err
	}
	return omni.ToYaml(), nil
}
