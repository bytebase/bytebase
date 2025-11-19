package pg

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestCase is the data struct for test.
type TestCase struct {
	Statement  string                                        `yaml:"statement"`
	ChangeType storepb.PlanCheckRunConfig_ChangeDatabaseType `yaml:"changeType"`
	Want       []*storepb.Advice                             `yaml:"want,omitempty"`
}

// RunANTLRAdvisorRuleTest helps to test ANTLR-based SQL review rules.
// This is similar to advisor.RunSQLReviewRuleTest but for ANTLR advisors.
func RunANTLRAdvisorRuleTest(t *testing.T, rule advisor.SQLReviewRuleType, dbType storepb.Engine, record bool) {
	var tests []TestCase

	fileName := strings.Map(func(r rune) rune {
		switch r {
		case '.', '-':
			return '_'
		default:
			return r
		}
	}, string(rule))
	filepath := filepath.Join("test", fileName+".yaml")
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err, rule)

	for i, tc := range tests {
		// Add engine types here for mocked database metadata.
		var schemaMetadata *storepb.DatabaseSchemaMetadata
		switch dbType {
		case storepb.Engine_POSTGRES:
			schemaMetadata = advisor.MockPostgreSQLDatabase
		default:
			t.Fatalf("%s doesn't have mocked metadata support", storepb.Engine_name[int32(dbType)])
		}

		// Create OriginalMetadata as DatabaseMetadata (read-only)
		// Clone to avoid mutations affecting future test cases
		originalCatalogClone, ok := proto.Clone(schemaMetadata).(*storepb.DatabaseSchemaMetadata)
		require.True(t, ok, "failed to clone schema metadata")
		originalSchema := model.NewDatabaseSchema(originalCatalogClone, nil, nil, dbType, true /* isCaseSensitive for PostgreSQL */)
		originalMetadata := originalSchema.GetDatabaseMetadata()

		// Create FinalMetadata as DatabaseMetadata (mutable for walk-through)
		// Clone to avoid mutations affecting future test cases
		finalCatalogClone, ok := proto.Clone(schemaMetadata).(*storepb.DatabaseSchemaMetadata)
		require.True(t, ok, "failed to clone schema metadata")
		finalMetadata := model.NewDatabaseMetadata(finalCatalogClone, true /* isCaseSensitive for PostgreSQL */, true /* isDetailCaseSensitive */)

		// Get default payload, or use empty string for test-only rules
		payload, err := advisor.SetDefaultSQLReviewRulePayload(rule, dbType)
		if err != nil {
			// For test-only rules (like "hello-world"), use empty payload
			payload = ""
		}

		// Parse SQL using ANTLR for pg advisors
		tree, err := pg.ParsePostgreSQL(tc.Statement)
		require.NoError(t, err, "Failed to parse SQL: %s", tc.Statement)

		// Always walk through the catalog to build metadata.
		advice := schema.WalkThrough(dbType, finalMetadata, tree)
		require.Nil(t, advice, "Failed to walk through final catalog: %s", tc.Statement)

		ruleList := []*storepb.SQLReviewRule{
			{
				Type:    string(rule),
				Level:   storepb.SQLReviewRuleLevel_WARNING,
				Payload: payload,
			},
		}

		// Call the advisor directly with ANTLR AST
		adviceList, err := advisor.Check(
			context.Background(),
			dbType,
			rule,
			advisor.Context{
				DBSchema:                 schemaMetadata,
				ChangeType:               tc.ChangeType,
				EnablePriorBackup:        true,
				AST:                      tree, // Pass ANTLR parse result
				Statements:               tc.Statement,
				Rule:                     ruleList[0],
				OriginalMetadata:         originalMetadata,
				FinalMetadata:            finalMetadata,
				Driver:                   nil,
				CurrentDatabase:          "TEST_DB",
				UsePostgresDatabaseOwner: true,
			},
		)

		// Sort adviceList by (line, content) for consistent comparison
		slices.SortFunc(adviceList, func(x, y *storepb.Advice) int {
			if x.GetStartPosition() == nil || y.GetStartPosition() == nil {
				if x.GetStartPosition() == nil && y.GetStartPosition() == nil {
					return 0
				} else if x.GetStartPosition() == nil {
					return -1
				}
				return 1
			}
			if x.GetStartPosition().Line != y.GetStartPosition().Line {
				if x.GetStartPosition().Line < y.GetStartPosition().Line {
					return -1
				}
				return 1
			}
			if x.Content < y.Content {
				return -1
			} else if x.Content > y.Content {
				return 1
			}
			return 0
		})

		require.NoError(t, err)
		if record {
			tests[i].Want = adviceList
		} else {
			require.Equalf(t, tc.Want, adviceList, "rule: %s, statements: %s", rule, tc.Statement)
		}
	}

	if record {
		err := yamlFile.Close()
		require.NoError(t, err)
		byteValue, err := yaml.Marshal(tests)
		require.NoError(t, err)
		err = os.WriteFile(filepath, byteValue, 0644)
		require.NoError(t, err)
	}
}
