package advisor_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"

	_ "github.com/bytebase/bytebase/backend/plugin/advisor/bigquery"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mariadb"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mssql"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/oceanbase"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/oracle"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/pg"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/redshift"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/snowflake"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/tidb"
)

// TestTemplateAdvisorRegistrations validates that all SQL review rules in frontend templates
// have corresponding advisor registrations in the Go code.
// This test prevents issues like BYT-8388 where templates contained rules that were never implemented.
func TestTemplateAdvisorRegistrations(t *testing.T) {
	repoRoot, err := getRepoRoot()
	require.NoError(t, err, "Failed to find repository root")

	templateFiles := []string{
		"frontend/src/types/sql-review.prod.yaml",
		"frontend/src/types/sql-review.dev.yaml",
		"frontend/src/types/sql-review-schema.yaml",
	}

	for _, templateFile := range templateFiles {
		t.Run(templateFile, func(t *testing.T) {
			templatePath := filepath.Join(repoRoot, templateFile)
			mismatches, err := validateTemplate(templatePath)
			require.NoError(t, err, "Failed to validate template")

			if len(mismatches) > 0 {
				var errMsg strings.Builder
				fmt.Fprintf(&errMsg, "\nFound %d advisor rule(s) in template that are NOT registered in Go code:\n", len(mismatches))
				for _, mismatch := range mismatches {
					fmt.Fprintf(&errMsg, "  - Engine: %s, Rule: %s\n", mismatch.Engine, mismatch.RuleType)
				}
				errMsg.WriteString("\nTo fix this issue:\n")
				errMsg.WriteString("1. Remove the invalid rule from the template file, OR\n")
				errMsg.WriteString("2. Implement the advisor in backend/plugin/advisor/<engine>/\n")
				errMsg.WriteString("3. Add a database migration to clean up existing entries (see migration 3.12.5)\n")
				t.Fatal(errMsg.String())
			}
		})
	}
}

// Mismatch represents a rule in the template without a corresponding Go registration.
type Mismatch struct {
	Engine   string
	RuleType string
}

// TemplateRule represents a rule from the YAML template.
type TemplateRule struct {
	Type   string `yaml:"type"`
	Engine string `yaml:"engine"`
}

// SQLReviewTemplate represents the structure of the SQL review YAML files.
type SQLReviewTemplate struct {
	RuleList []TemplateRule `yaml:"ruleList"`
}

// SQLReviewSchemaTemplate represents the structure of the SQL review schema YAML file.
// The schema file is a flat array, not wrapped in an object.
type SQLReviewSchemaTemplate []TemplateRule

// getRepoRoot finds the repository root by looking for go.mod.
func getRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not find repository root (go.mod not found)")
		}
		dir = parent
	}
}

// validateTemplate validates a template file against the registered advisors.
func validateTemplate(templatePath string) ([]Mismatch, error) {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read template file")
	}

	var templateRules []TemplateRule
	if strings.Contains(templatePath, "sql-review-schema.yaml") {
		var schema SQLReviewSchemaTemplate
		if err := yaml.Unmarshal(data, &schema); err != nil {
			return nil, errors.Wrap(err, "failed to parse schema YAML")
		}
		templateRules = schema
	} else {
		var template SQLReviewTemplate
		if err := yaml.Unmarshal(data, &template); err != nil {
			return nil, errors.Wrap(err, "failed to parse template YAML")
		}
		templateRules = template.RuleList
	}

	var mismatches []Mismatch
	seen := make(map[string]bool)

	for _, rule := range templateRules {
		if rule.Type == "" || rule.Engine == "" {
			continue
		}

		ruleType, ok := parseRuleType(rule.Type)
		if !ok {
			mismatches = append(mismatches, Mismatch{
				Engine:   rule.Engine,
				RuleType: rule.Type + " (unknown rule type)",
			})
			continue
		}

		key := fmt.Sprintf("%s|%s", rule.Engine, rule.Type)
		if seen[key] {
			continue
		}
		seen[key] = true

		if isPreParseSQLReviewRule(rule.Type) {
			continue
		}
		engineVal, ok := storepb.Engine_value[rule.Engine]
		if !ok || !advisor.IsRegistered(storepb.Engine(engineVal), ruleType) {
			mismatches = append(mismatches, Mismatch{
				Engine:   rule.Engine,
				RuleType: rule.Type,
			})
		}
	}

	slices.SortFunc(mismatches, func(a, b Mismatch) int {
		if a.Engine != b.Engine {
			return strings.Compare(a.Engine, b.Engine)
		}
		return strings.Compare(a.RuleType, b.RuleType)
	})

	return mismatches, nil
}

func isPreParseSQLReviewRule(ruleType string) bool {
	return ruleType == "BUILTIN_STATEMENT_MAXIMUM_SQL_SIZE"
}

func parseRuleType(ruleType string) (storepb.SQLReviewRule_Type, bool) {
	name := strings.TrimPrefix(ruleType, "storepb.SQLReviewRule_")
	value, ok := storepb.SQLReviewRule_Type_value[name]
	if !ok {
		return storepb.SQLReviewRule_TYPE_UNSPECIFIED, false
	}
	return storepb.SQLReviewRule_Type(value), true
}
