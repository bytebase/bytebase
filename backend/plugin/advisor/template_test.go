package advisor_test

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestTemplateAdvisorRegistrations validates that all SQL review rules in frontend templates
// have corresponding advisor registrations in the Go code.
// This test prevents issues like BYT-8388 where templates contained rules that were never implemented.
func TestTemplateAdvisorRegistrations(t *testing.T) {
	// Get the repository root directory
	repoRoot, err := getRepoRoot()
	require.NoError(t, err, "Failed to find repository root")

	// Collect all advisor registrations from Go code
	registrations, err := collectAdvisorRegistrations(repoRoot)
	require.NoError(t, err, "Failed to collect advisor registrations")
	require.NotEmpty(t, registrations, "No advisor registrations found")

	// Test each template file
	templateFiles := []string{
		"frontend/src/types/sql-review.prod.yaml",
		"frontend/src/types/sql-review.dev.yaml",
		"frontend/src/types/sql-review-schema.yaml",
	}

	for _, templateFile := range templateFiles {
		t.Run(templateFile, func(t *testing.T) {
			templatePath := filepath.Join(repoRoot, templateFile)
			mismatches, err := validateTemplate(templatePath, registrations)
			require.NoError(t, err, "Failed to validate template")

			if len(mismatches) > 0 {
				var errMsg strings.Builder
				errMsg.WriteString(fmt.Sprintf("\nFound %d advisor rule(s) in template that are NOT registered in Go code:\n", len(mismatches)))
				for _, mismatch := range mismatches {
					errMsg.WriteString(fmt.Sprintf("  - Engine: %s, Rule: %s\n", mismatch.Engine, mismatch.RuleType))
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

// AdvisorRegistration represents an advisor registration in Go code.
type AdvisorRegistration struct {
	Engine   string
	RuleType string
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
	// Start from the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree until we find go.mod
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

// collectAdvisorRegistrations collects all advisor registrations from Go code.
func collectAdvisorRegistrations(repoRoot string) (map[AdvisorRegistration]bool, error) {
	advisorDir := filepath.Join(repoRoot, "backend", "plugin", "advisor")

	// Use grep to find all advisor.Register calls
	// We use grep because it's much faster than parsing Go code
	cmd := exec.Command("grep", "-r", "advisor.Register(storepb.Engine_", advisorDir, "--include=*.go")
	output, err := cmd.Output()
	if err != nil {
		// Check if it's just an empty result
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, errors.New("no advisor registrations found")
		}
		return nil, errors.Wrap(err, "failed to grep advisor registrations")
	}

	registrations := make(map[AdvisorRegistration]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		// Line format: filepath:	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_NO_NULL, ...)

		// Extract the registration part after the filename
		parts := strings.SplitN(line, "advisor.Register(storepb.Engine_", 2)
		if len(parts) != 2 {
			continue
		}

		// parts[1] looks like: "MYSQL, storepb.SQLReviewRule_COLUMN_NO_NULL, ...)"
		regPart := parts[1]

		// Extract engine and rule type
		commaIdx := strings.Index(regPart, ",")
		if commaIdx == -1 {
			continue
		}

		engine := strings.TrimSpace(regPart[:commaIdx])

		// Find the rule type (between the first and second comma)
		remaining := regPart[commaIdx+1:]
		remaining = strings.TrimSpace(remaining)

		// Remove "advisor." prefix
		remaining = strings.TrimPrefix(remaining, "advisor.")

		// Extract until the next comma or closing paren
		endIdx := strings.IndexAny(remaining, ",)")
		if endIdx == -1 {
			continue
		}

		ruleType := strings.TrimSpace(remaining[:endIdx])

		registrations[AdvisorRegistration{
			Engine:   engine,
			RuleType: ruleType,
		}] = true
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "error reading grep output")
	}

	return registrations, nil
}

// validateTemplate validates a template file against the registered advisors.
func validateTemplate(templatePath string, registrations map[AdvisorRegistration]bool) ([]Mismatch, error) {
	// Read the template file
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read template file")
	}

	// Parse based on filename
	var templateRules []TemplateRule
	if strings.Contains(templatePath, "sql-review-schema.yaml") {
		// Schema file has different structure - it's a flat array
		var schema SQLReviewSchemaTemplate
		if err := yaml.Unmarshal(data, &schema); err != nil {
			return nil, errors.Wrap(err, "failed to parse schema YAML")
		}
		templateRules = schema
	} else {
		// Regular template files
		var template SQLReviewTemplate
		if err := yaml.Unmarshal(data, &template); err != nil {
			return nil, errors.Wrap(err, "failed to parse template YAML")
		}
		templateRules = template.RuleList
	}

	// Build a map of rule type to SchemaRule constant name
	ruleTypeToSchema := buildRuleTypeMapping()

	// Check each rule in the template
	var mismatches []Mismatch
	seen := make(map[string]bool)

	for _, rule := range templateRules {
		if rule.Type == "" || rule.Engine == "" {
			continue
		}

		// Convert template rule type to SchemaRule constant name
		schemaRuleName, ok := ruleTypeToSchema[rule.Type]
		if !ok {
			// Unknown rule type - this is also a problem
			mismatches = append(mismatches, Mismatch{
				Engine:   rule.Engine,
				RuleType: rule.Type + " (unknown rule type)",
			})
			continue
		}

		// Check if this engine+rule combination is registered
		key := fmt.Sprintf("%s|%s", rule.Engine, rule.Type)
		if seen[key] {
			// Already checked this combination (templates can have duplicates with different payloads)
			continue
		}
		seen[key] = true

		registration := AdvisorRegistration{
			Engine:   rule.Engine,
			RuleType: schemaRuleName,
		}

		if !registrations[registration] {
			mismatches = append(mismatches, Mismatch{
				Engine:   rule.Engine,
				RuleType: rule.Type,
			})
		}
	}

	// Sort mismatches for consistent output
	slices.SortFunc(mismatches, func(a, b Mismatch) int {
		if a.Engine != b.Engine {
			return strings.Compare(a.Engine, b.Engine)
		}
		return strings.Compare(a.RuleType, b.RuleType)
	})

	return mismatches, nil
}

// buildRuleTypeMapping creates a mapping from template rule types to SchemaRule constant names.
// This is extracted from storepb.SQLReviewRule_Type constants.
// buildRuleTypeMapping creates a mapping from template rule types (original string values) to SchemaRule constant names.
func buildRuleTypeMapping() map[string]string {
	return map[string]string{
		"ENGINE_MYSQL_USE_INNODB":                             "storepb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB",
		"NAMING_FULLY_QUALIFIED":                              "storepb.SQLReviewRule_NAMING_FULLY_QUALIFIED",
		"NAMING_TABLE":                                        "storepb.SQLReviewRule_NAMING_TABLE",
		"NAMING_COLUMN":                                       "storepb.SQLReviewRule_NAMING_COLUMN",
		"NAMING_INDEX_PK":                                     "storepb.SQLReviewRule_NAMING_INDEX_PK",
		"NAMING_INDEX_UK":                                     "storepb.SQLReviewRule_NAMING_INDEX_UK",
		"NAMING_INDEX_FK":                                     "storepb.SQLReviewRule_NAMING_INDEX_FK",
		"NAMING_INDEX_IDX":                                    "storepb.SQLReviewRule_NAMING_INDEX_IDX",
		"NAMING_COLUMN_AUTO_INCREMENT":                        "storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT",
		"NAMING_TABLE_NO_KEYWORD":                             "storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD",
		"NAMING_IDENTIFIER_NO_KEYWORD":                        "storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD",
		"NAMING_IDENTIFIER_CASE":                              "storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE",
		"STATEMENT_SELECT_NO_SELECT_ALL":                      "storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL",
		"STATEMENT_WHERE_REQUIRE_SELECT":                      "storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT",
		"STATEMENT_WHERE_REQUIRE_UPDATE_DELETE":               "storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE",
		"STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE":            "storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE",
		"STATEMENT_DISALLOW_ON_DEL_CASCADE":                   "storepb.SQLReviewRule_STATEMENT_DISALLOW_ON_DEL_CASCADE",
		"STATEMENT_DISALLOW_RM_TBL_CASCADE":                   "storepb.SQLReviewRule_STATEMENT_DISALLOW_RM_TBL_CASCADE",
		"STATEMENT_DISALLOW_COMMIT":                           "storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT",
		"STATEMENT_DISALLOW_LIMIT":                            "storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT",
		"STATEMENT_DISALLOW_ORDER_BY":                         "storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY",
		"STATEMENT_MERGE_ALTER_TABLE":                         "storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE",
		"STATEMENT_INSERT_ROW_LIMIT":                          "storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT",
		"STATEMENT_INSERT_MUST_SPECIFY_COLUMN":                "storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN",
		"STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND":             "storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND",
		"STATEMENT_AFFECTED_ROW_LIMIT":                        "storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT",
		"STATEMENT_DML_DRY_RUN":                               "storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN",
		"STATEMENT_DISALLOW_ADD_COLUMN_WITH_DEFAULT":          "storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_COLUMN_WITH_DEFAULT",
		"STATEMENT_ADD_CHECK_NOT_VALID":                       "storepb.SQLReviewRule_STATEMENT_ADD_CHECK_NOT_VALID",
		"STATEMENT_ADD_FOREIGN_KEY_NOT_VALID":                 "storepb.SQLReviewRule_STATEMENT_ADD_FOREIGN_KEY_NOT_VALID",
		"STATEMENT_DISALLOW_ADD_NOT_NULL":                     "storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_NOT_NULL",
		"STATEMENT_SELECT_FULL_TABLE_SCAN":                    "storepb.SQLReviewRule_STATEMENT_SELECT_FULL_TABLE_SCAN",
		"STATEMENT_CREATE_SPECIFY_SCHEMA":                     "storepb.SQLReviewRule_STATEMENT_CREATE_SPECIFY_SCHEMA",
		"STATEMENT_CHECK_SET_ROLE_VARIABLE":                   "storepb.SQLReviewRule_STATEMENT_CHECK_SET_ROLE_VARIABLE",
		"STATEMENT_DISALLOW_USING_FILESORT":                   "storepb.SQLReviewRule_STATEMENT_DISALLOW_USING_FILESORT",
		"STATEMENT_DISALLOW_USING_TEMPORARY":                  "storepb.SQLReviewRule_STATEMENT_DISALLOW_USING_TEMPORARY",
		"STATEMENT_WHERE_NO_EQUAL_NULL":                       "storepb.SQLReviewRule_STATEMENT_WHERE_NO_EQUAL_NULL",
		"STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS": "storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS",
		"STATEMENT_QUERY_MINIMUM_PLAN_LEVEL":                  "storepb.SQLReviewRule_STATEMENT_QUERY_MINIMUM_PLAN_LEVEL",
		"STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT":      "storepb.SQLReviewRule_STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT",
		"STATEMENT_MAXIMUM_LIMIT_VALUE":                       "storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE",
		"STATEMENT_MAXIMUM_JOIN_TABLE_COUNT":                  "storepb.SQLReviewRule_STATEMENT_MAXIMUM_JOIN_TABLE_COUNT",
		"STATEMENT_MAXIMUM_STATEMENTS_IN_TRANSACTION":         "storepb.SQLReviewRule_STATEMENT_MAXIMUM_STATEMENTS_IN_TRANSACTION",
		"STATEMENT_JOIN_STRICT_COLUMN_ATTRS":                  "storepb.SQLReviewRule_STATEMENT_JOIN_STRICT_COLUMN_ATTRS",
		"STATEMENT_NON_TRANSACTIONAL":                         "storepb.SQLReviewRule_STATEMENT_NON_TRANSACTIONAL",
		"STATEMENT_ADD_COLUMN_WITHOUT_POSITION":               "storepb.SQLReviewRule_STATEMENT_ADD_COLUMN_WITHOUT_POSITION",
		"STATEMENT_DISALLOW_OFFLINE_DDL":                      "storepb.SQLReviewRule_STATEMENT_DISALLOW_OFFLINE_DDL",
		"STATEMENT_DISALLOW_CROSS_DB_QUERIES":                 "storepb.SQLReviewRule_STATEMENT_DISALLOW_CROSS_DB_QUERIES",
		"STATEMENT_MAX_EXECUTION_TIME":                        "storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME",
		"STATEMENT_REQUIRE_ALGORITHM_OPTION":                  "storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION",
		"STATEMENT_REQUIRE_LOCK_OPTION":                       "storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION",
		"STATEMENT_OBJECT_OWNER_CHECK":                        "storepb.SQLReviewRule_STATEMENT_OBJECT_OWNER_CHECK",
		"TABLE_REQUIRE_PK":                                    "storepb.SQLReviewRule_TABLE_REQUIRE_PK",
		"TABLE_NO_FOREIGN_KEY":                                "storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY",
		"TABLE_DROP_NAMING_CONVENTION":                        "storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION",
		"TABLE_COMMENT":                                       "storepb.SQLReviewRule_TABLE_COMMENT",
		"TABLE_DISALLOW_PARTITION":                            "storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION",
		"TABLE_DISALLOW_TRIGGER":                              "storepb.SQLReviewRule_TABLE_DISALLOW_TRIGGER",
		"TABLE_NO_DUPLICATE_INDEX":                            "storepb.SQLReviewRule_TABLE_NO_DUPLICATE_INDEX",
		"TABLE_TEXT_FIELDS_TOTAL_LENGTH":                      "storepb.SQLReviewRule_TABLE_TEXT_FIELDS_TOTAL_LENGTH",
		"TABLE_DISALLOW_SET_CHARSET":                          "storepb.SQLReviewRule_TABLE_DISALLOW_SET_CHARSET",
		"TABLE_DISALLOW_DDL":                                  "storepb.SQLReviewRule_TABLE_DISALLOW_DDL",
		"TABLE_DISALLOW_DML":                                  "storepb.SQLReviewRule_TABLE_DISALLOW_DML",
		"TABLE_LIMIT_SIZE":                                    "storepb.SQLReviewRule_TABLE_LIMIT_SIZE",
		"TABLE_REQUIRE_CHARSET":                               "storepb.SQLReviewRule_TABLE_REQUIRE_CHARSET",
		"TABLE_REQUIRE_COLLATION":                             "storepb.SQLReviewRule_TABLE_REQUIRE_COLLATION",
		"COLUMN_REQUIRED":                                     "storepb.SQLReviewRule_COLUMN_REQUIRED",
		"COLUMN_NO_NULL":                                      "storepb.SQLReviewRule_COLUMN_NO_NULL",
		"COLUMN_DISALLOW_CHANGE_TYPE":                         "storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE",
		"COLUMN_SET_DEFAULT_FOR_NOT_NULL":                     "storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL",
		"COLUMN_DISALLOW_CHANGE":                              "storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE",
		"COLUMN_DISALLOW_CHANGING_ORDER":                      "storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER",
		"COLUMN_DISALLOW_DROP":                                "storepb.SQLReviewRule_COLUMN_DISALLOW_DROP",
		"COLUMN_DISALLOW_DROP_IN_INDEX":                       "storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX",
		"COLUMN_COMMENT":                                      "storepb.SQLReviewRule_COLUMN_COMMENT",
		"COLUMN_AUTO_INCREMENT_MUST_INTEGER":                  "storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER",
		"COLUMN_TYPE_DISALLOW_LIST":                           "storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST",
		"COLUMN_DISALLOW_SET_CHARSET":                         "storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET",
		"COLUMN_MAXIMUM_CHARACTER_LENGTH":                     "storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH",
		"COLUMN_MAXIMUM_VARCHAR_LENGTH":                       "storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH",
		"COLUMN_AUTO_INCREMENT_INITIAL_VALUE":                 "storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE",
		"COLUMN_AUTO_INCREMENT_MUST_UNSIGNED":                 "storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED",
		"COLUMN_CURRENT_TIME_COUNT_LIMIT":                     "storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT",
		"COLUMN_REQUIRE_DEFAULT":                              "storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT",
		"COLUMN_DEFAULT_DISALLOW_VOLATILE":                    "storepb.SQLReviewRule_COLUMN_DEFAULT_DISALLOW_VOLATILE",
		"COLUMN_ADD_NOT_NULL_REQUIRE_DEFAULT":                 "storepb.SQLReviewRule_COLUMN_ADD_NOT_NULL_REQUIRE_DEFAULT",
		"COLUMN_REQUIRE_CHARSET":                              "storepb.SQLReviewRule_COLUMN_REQUIRE_CHARSET",
		"COLUMN_REQUIRE_COLLATION":                            "storepb.SQLReviewRule_COLUMN_REQUIRE_COLLATION",
		"SCHEMA_BACKWARD_COMPATIBILITY":                       "storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY",
		"DATABASE_DROP_EMPTY_DATABASE":                        "storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE",
		"INDEX_NO_DUPLICATE_COLUMN":                           "storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN",
		"INDEX_KEY_NUMBER_LIMIT":                              "storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT",
		"INDEX_PK_TYPE_LIMIT":                                 "storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT",
		"INDEX_TYPE_NO_BLOB":                                  "storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB",
		"INDEX_TOTAL_NUMBER_LIMIT":                            "storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT",
		"INDEX_PRIMARY_KEY_TYPE_ALLOWLIST":                    "storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST",
		"INDEX_CREATE_CONCURRENTLY":                           "storepb.SQLReviewRule_INDEX_CREATE_CONCURRENTLY",
		"INDEX_TYPE_ALLOW_LIST":                               "storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST",
		"INDEX_NOT_REDUNDANT":                                 "storepb.SQLReviewRule_INDEX_NOT_REDUNDANT",
		"SYSTEM_CHARSET_ALLOWLIST":                            "storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST",
		"SYSTEM_COLLATION_ALLOWLIST":                          "storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST",
		"SYSTEM_COMMENT_LENGTH":                               "storepb.SQLReviewRule_SYSTEM_COMMENT_LENGTH",
		"SYSTEM_PROCEDURE_DISALLOW_CREATE":                    "storepb.SQLReviewRule_SYSTEM_PROCEDURE_DISALLOW_CREATE",
		"SYSTEM_EVENT_DISALLOW_CREATE":                        "storepb.SQLReviewRule_SYSTEM_EVENT_DISALLOW_CREATE",
		"SYSTEM_VIEW_DISALLOW_CREATE":                         "storepb.SQLReviewRule_SYSTEM_VIEW_DISALLOW_CREATE",
		"SYSTEM_FUNCTION_DISALLOW_CREATE":                     "storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOW_CREATE",
		"SYSTEM_FUNCTION_DISALLOWED_LIST":                     "storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOWED_LIST",
		"ADVICE_ONLINE_MIGRATION":                             "storepb.SQLReviewRule_ADVICE_ONLINE_MIGRATION",
		"BUILTIN_PRIOR_BACKUP_CHECK":                          storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK.String(),
	}
}
