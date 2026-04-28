package mysql

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestMySQLOmniQuerySpanParityHarness is the strict cutover guard. It compares
// the public MySQL query-span path against the package-internal omni path across
// the existing YAML corpus.
func TestMySQLOmniQuerySpanParityHarness(t *testing.T) {
	type testCase struct {
		Description        string `yaml:"description,omitempty"`
		Statement          string `yaml:"statement,omitempty"`
		DefaultDatabase    string `yaml:"defaultDatabase,omitempty"`
		IgnoreCaseSensitve bool   `yaml:"ignoreCaseSensitive,omitempty"`
		Metadata           string `yaml:"metadata,omitempty"`
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
	for _, testDataPath := range mysqlOmniProbeFixturePaths {
		engine := storepb.Engine_MYSQL
		if strings.Contains(testDataPath, "starrocks") {
			engine = storepb.Engine_STARROCKS
		}

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

			ref, omni, err := runMySQLOmniParityCase(context.Background(), tc.Statement, tc.Metadata, tc.DefaultDatabase, engine, tc.IgnoreCaseSensitve)
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
			if !reflect.DeepEqual(ref, omni) {
				diffs = append(diffs, diff{
					fixture:     testDataPath,
					index:       i,
					description: tc.Description,
					statement:   tc.Statement,
					details:     fmt.Sprintf("ref=%+v omni=%+v", ref, omni),
				})
			}
		}
	}

	t.Logf("MySQL omni query-span parity: %d/%d matched, %d diffs", total-len(diffs), total, len(diffs))
	for i, d := range diffs {
		if i >= 10 {
			t.Logf("... %d more diffs omitted", len(diffs)-i)
			break
		}
		t.Logf("[%s case %d %q] %s\n  SQL: %s", d.fixture, d.index, d.description, d.details, firstMySQLOmniProbeLine(d.statement))
	}
	require.Empty(t, diffs)
}

func runMySQLOmniParityCase(
	ctx context.Context,
	statement string,
	metadataText string,
	defaultDatabase string,
	engine storepb.Engine,
	ignoreCaseSensitive bool,
) (*base.YamlQuerySpan, *base.YamlQuerySpan, error) {
	metadata := &storepb.DatabaseSchemaMetadata{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(metadataText), metadata); err != nil {
		return nil, nil, err
	}
	databaseMetadataGetter, databaseNameLister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: databaseMetadataGetter,
		ListDatabaseNamesFunc:   databaseNameLister,
		Engine:                  engine,
	}

	ref, err := GetQuerySpan(ctx, gCtx, base.Statement{Text: statement}, defaultDatabase, "", ignoreCaseSensitive)
	if err != nil {
		return nil, nil, err
	}
	omni, err := newOmniQuerySpanExtractor(defaultDatabase, gCtx, ignoreCaseSensitive).getOmniQuerySpan(ctx, statement)
	if err != nil {
		return ref.ToYaml(), nil, err
	}
	return ref.ToYaml(), omni.ToYaml(), nil
}
