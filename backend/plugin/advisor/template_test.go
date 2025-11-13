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

	"github.com/bytebase/bytebase/backend/plugin/advisor"
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
		// Line format: filepath:	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleColumnNotNull, ...)

		// Extract the registration part after the filename
		parts := strings.SplitN(line, "advisor.Register(storepb.Engine_", 2)
		if len(parts) != 2 {
			continue
		}

		// parts[1] looks like: "MYSQL, advisor.SchemaRuleColumnNotNull, ...)"
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
// This is extracted from advisor.SQLReviewRuleType constants.
func buildRuleTypeMapping() map[string]string {
	return map[string]string{
		string(advisor.SchemaRuleMySQLEngine):                                    "SchemaRuleMySQLEngine",
		string(advisor.SchemaRuleFullyQualifiedObjectName):                       "SchemaRuleFullyQualifiedObjectName",
		string(advisor.SchemaRuleTableNaming):                                    "SchemaRuleTableNaming",
		string(advisor.SchemaRuleColumnNaming):                                   "SchemaRuleColumnNaming",
		string(advisor.SchemaRulePKNaming):                                       "SchemaRulePKNaming",
		string(advisor.SchemaRuleUKNaming):                                       "SchemaRuleUKNaming",
		string(advisor.SchemaRuleFKNaming):                                       "SchemaRuleFKNaming",
		string(advisor.SchemaRuleIDXNaming):                                      "SchemaRuleIDXNaming",
		string(advisor.SchemaRuleAutoIncrementColumnNaming):                      "SchemaRuleAutoIncrementColumnNaming",
		string(advisor.SchemaRuleTableNameNoKeyword):                             "SchemaRuleTableNameNoKeyword",
		string(advisor.SchemaRuleIdentifierNoKeyword):                            "SchemaRuleIdentifierNoKeyword",
		string(advisor.SchemaRuleIdentifierCase):                                 "SchemaRuleIdentifierCase",
		string(advisor.SchemaRuleStatementNoSelectAll):                           "SchemaRuleStatementNoSelectAll",
		string(advisor.SchemaRuleStatementRequireWhereForSelect):                 "SchemaRuleStatementRequireWhereForSelect",
		string(advisor.SchemaRuleStatementRequireWhereForUpdateDelete):           "SchemaRuleStatementRequireWhereForUpdateDelete",
		string(advisor.SchemaRuleStatementNoLeadingWildcardLike):                 "SchemaRuleStatementNoLeadingWildcardLike",
		string(advisor.SchemaRuleStatementDisallowOnDelCascade):                  "SchemaRuleStatementDisallowOnDelCascade",
		string(advisor.SchemaRuleStatementDisallowRemoveTblCascade):              "SchemaRuleStatementDisallowRemoveTblCascade",
		string(advisor.SchemaRuleStatementDisallowCommit):                        "SchemaRuleStatementDisallowCommit",
		string(advisor.SchemaRuleStatementDisallowLimit):                         "SchemaRuleStatementDisallowLimit",
		string(advisor.SchemaRuleStatementDisallowOrderBy):                       "SchemaRuleStatementDisallowOrderBy",
		string(advisor.SchemaRuleStatementMergeAlterTable):                       "SchemaRuleStatementMergeAlterTable",
		string(advisor.SchemaRuleStatementInsertRowLimit):                        "SchemaRuleStatementInsertRowLimit",
		string(advisor.SchemaRuleStatementInsertMustSpecifyColumn):               "SchemaRuleStatementInsertMustSpecifyColumn",
		string(advisor.SchemaRuleStatementInsertDisallowOrderByRand):             "SchemaRuleStatementInsertDisallowOrderByRand",
		string(advisor.SchemaRuleStatementAffectedRowLimit):                      "SchemaRuleStatementAffectedRowLimit",
		string(advisor.SchemaRuleStatementDMLDryRun):                             "SchemaRuleStatementDMLDryRun",
		string(advisor.SchemaRuleStatementDisallowAddColumnWithDefault):          "SchemaRuleStatementDisallowAddColumnWithDefault",
		string(advisor.SchemaRuleStatementAddCheckNotValid):                      "SchemaRuleStatementAddCheckNotValid",
		string(advisor.SchemaRuleStatementAddFKNotValid):                         "SchemaRuleStatementAddFKNotValid",
		string(advisor.SchemaRuleStatementDisallowAddNotNull):                    "SchemaRuleStatementDisallowAddNotNull",
		string(advisor.SchemaRuleStatementSelectFullTableScan):                   "SchemaRuleStatementSelectFullTableScan",
		string(advisor.SchemaRuleStatementCreateSpecifySchema):                   "SchemaRuleStatementCreateSpecifySchema",
		string(advisor.SchemaRuleStatementCheckSetRoleVariable):                  "SchemaRuleStatementCheckSetRoleVariable",
		string(advisor.SchemaRuleStatementDisallowUsingFilesort):                 "SchemaRuleStatementDisallowUsingFilesort",
		string(advisor.SchemaRuleStatementDisallowUsingTemporary):                "SchemaRuleStatementDisallowUsingTemporary",
		string(advisor.SchemaRuleStatementWhereNoEqualNull):                      "SchemaRuleStatementWhereNoEqualNull",
		string(advisor.SchemaRuleStatementWhereDisallowFunctionsAndCalculations): "SchemaRuleStatementWhereDisallowFunctionsAndCalculations",
		string(advisor.SchemaRuleStatementQueryMinumumPlanLevel):                 "SchemaRuleStatementQueryMinumumPlanLevel",
		string(advisor.SchemaRuleStatementWhereMaximumLogicalOperatorCount):      "SchemaRuleStatementWhereMaximumLogicalOperatorCount",
		string(advisor.SchemaRuleStatementMaximumLimitValue):                     "SchemaRuleStatementMaximumLimitValue",
		string(advisor.SchemaRuleStatementMaximumJoinTableCount):                 "SchemaRuleStatementMaximumJoinTableCount",
		string(advisor.SchemaRuleStatementMaximumStatementsInTransaction):        "SchemaRuleStatementMaximumStatementsInTransaction",
		string(advisor.SchemaRuleStatementJoinStrictColumnAttrs):                 "SchemaRuleStatementJoinStrictColumnAttrs",
		string(advisor.SchemaRuleStatementDisallowMixInDDL):                      "SchemaRuleStatementDisallowMixInDDL",
		string(advisor.SchemaRuleStatementDisallowMixInDML):                      "SchemaRuleStatementDisallowMixInDML",
		string(advisor.SchemaRuleStatementPriorBackupCheck):                      "SchemaRuleStatementPriorBackupCheck",
		string(advisor.SchemaRuleStatementNonTransactional):                      "SchemaRuleStatementNonTransactional",
		string(advisor.SchemaRuleStatementAddColumnWithoutPosition):              "SchemaRuleStatementAddColumnWithoutPosition",
		string(advisor.SchemaRuleStatementDisallowOfflineDDL):                    "SchemaRuleStatementDisallowOfflineDDL",
		string(advisor.SchemaRuleStatementDisallowCrossDBQueries):                "SchemaRuleStatementDisallowCrossDBQueries",
		string(advisor.SchemaRuleStatementMaxExecutionTime):                      "SchemaRuleStatementMaxExecutionTime",
		string(advisor.SchemaRuleStatementObjectOwnerCheck):                      "SchemaRuleStatementObjectOwnerCheck",
		string(advisor.SchemaRuleStatementRequireAlgorithmOption):                "SchemaRuleStatementRequireAlgorithmOption",
		string(advisor.SchemaRuleStatementRequireLockOption):                     "SchemaRuleStatementRequireLockOption",
		string(advisor.SchemaRuleTableNoDuplicateIndex):                          "SchemaRuleTableNoDuplicateIndex",
		string(advisor.SchemaRuleTableTextFieldsTotalLength):                     "SchemaRuleTableTextFieldsTotalLength",
		string(advisor.SchemaRuleTableDisallowSetCharset):                        "SchemaRuleTableDisallowSetCharset",
		string(advisor.SchemaRuleTableLimitSize):                                 "SchemaRuleTableLimitSize",
		string(advisor.SchemaRuleTableRequireCharset):                            "SchemaRuleTableRequireCharset",
		string(advisor.SchemaRuleTableRequireCollation):                          "SchemaRuleTableRequireCollation",
		string(advisor.SchemaRuleTableDisallowDML):                               "SchemaRuleTableDisallowDML",
		string(advisor.SchemaRuleTableDisallowDDL):                               "SchemaRuleTableDisallowDDL",
		string(advisor.SchemaRuleTableRequirePK):                                 "SchemaRuleTableRequirePK",
		string(advisor.SchemaRuleTableNoFK):                                      "SchemaRuleTableNoFK",
		string(advisor.SchemaRuleTableDropNamingConvention):                      "SchemaRuleTableDropNamingConvention",
		string(advisor.SchemaRuleTableCommentConvention):                         "SchemaRuleTableCommentConvention",
		string(advisor.SchemaRuleTableDisallowPartition):                         "SchemaRuleTableDisallowPartition",
		string(advisor.SchemaRuleTableDisallowTrigger):                           "SchemaRuleTableDisallowTrigger",
		string(advisor.SchemaRuleSchemaBackwardCompatibility):                    "SchemaRuleSchemaBackwardCompatibility",
		string(advisor.SchemaRuleDropEmptyDatabase):                              "SchemaRuleDropEmptyDatabase",
		string(advisor.SchemaRuleIndexNoDuplicateColumn):                         "SchemaRuleIndexNoDuplicateColumn",
		string(advisor.SchemaRuleIndexPKTypeLimit):                               "SchemaRuleIndexPKTypeLimit",
		string(advisor.SchemaRuleIndexPrimaryKeyTypeAllowlist):                   "SchemaRuleIndexPrimaryKeyTypeAllowlist",
		string(advisor.SchemaRuleIndexKeyNumberLimit):                            "SchemaRuleIndexKeyNumberLimit",
		string(advisor.SchemaRuleIndexTotalNumberLimit):                          "SchemaRuleIndexTotalNumberLimit",
		string(advisor.SchemaRuleIndexTypeAllowList):                             "SchemaRuleIndexTypeAllowList",
		string(advisor.SchemaRuleIndexTypeNoBlob):                                "SchemaRuleIndexTypeNoBlob",
		string(advisor.SchemaRuleIndexNotRedundant):                              "SchemaRuleIndexNotRedundant",
		string(advisor.SchemaRuleCreateIndexConcurrently):                        "SchemaRuleCreateIndexConcurrently",
		string(advisor.SchemaRuleCharsetAllowlist):                               "SchemaRuleCharsetAllowlist",
		string(advisor.SchemaRuleCollationAllowlist):                             "SchemaRuleCollationAllowlist",
		string(advisor.SchemaRuleCommentLength):                                  "SchemaRuleCommentLength",
		string(advisor.SchemaRuleRequiredColumn):                                 "SchemaRuleRequiredColumn",
		string(advisor.SchemaRuleColumnNotNull):                                  "SchemaRuleColumnNotNull",
		string(advisor.SchemaRuleColumnDisallowChangeType):                       "SchemaRuleColumnDisallowChangeType",
		string(advisor.SchemaRuleColumnSetDefaultForNotNull):                     "SchemaRuleColumnSetDefaultForNotNull",
		string(advisor.SchemaRuleColumnDisallowChange):                           "SchemaRuleColumnDisallowChange",
		string(advisor.SchemaRuleColumnDisallowDrop):                             "SchemaRuleColumnDisallowDrop",
		string(advisor.SchemaRuleColumnDisallowDropInIndex):                      "SchemaRuleColumnDisallowDropInIndex",
		string(advisor.SchemaRuleColumnDisallowChangingOrder):                    "SchemaRuleColumnDisallowChangingOrder",
		string(advisor.SchemaRuleColumnCommentConvention):                        "SchemaRuleColumnCommentConvention",
		string(advisor.SchemaRuleColumnAutoIncrementMustInteger):                 "SchemaRuleColumnAutoIncrementMustInteger",
		string(advisor.SchemaRuleColumnTypeDisallowList):                         "SchemaRuleColumnTypeDisallowList",
		string(advisor.SchemaRuleColumnDisallowSetCharset):                       "SchemaRuleColumnDisallowSetCharset",
		string(advisor.SchemaRuleColumnMaximumCharacterLength):                   "SchemaRuleColumnMaximumCharacterLength",
		string(advisor.SchemaRuleColumnMaximumVarcharLength):                     "SchemaRuleColumnMaximumVarcharLength",
		string(advisor.SchemaRuleColumnRequireCharset):                           "SchemaRuleColumnRequireCharset",
		string(advisor.SchemaRuleColumnRequireCollation):                         "SchemaRuleColumnRequireCollation",
		string(advisor.SchemaRuleColumnAutoIncrementInitialValue):                "SchemaRuleColumnAutoIncrementInitialValue",
		string(advisor.SchemaRuleColumnAutoIncrementMustUnsigned):                "SchemaRuleColumnAutoIncrementMustUnsigned",
		string(advisor.SchemaRuleCurrentTimeColumnCountLimit):                    "SchemaRuleCurrentTimeColumnCountLimit",
		string(advisor.SchemaRuleColumnRequireDefault):                           "SchemaRuleColumnRequireDefault",
		string(advisor.SchemaRuleColumnDefaultDisallowVolatile):                  "SchemaRuleColumnDefaultDisallowVolatile",
		string(advisor.SchemaRuleAddNotNullColumnRequireDefault):                 "SchemaRuleAddNotNullColumnRequireDefault",
		string(advisor.SchemaRuleProcedureDisallowCreate):                        "SchemaRuleProcedureDisallowCreate",
		string(advisor.SchemaRuleEventDisallowCreate):                            "SchemaRuleEventDisallowCreate",
		string(advisor.SchemaRuleViewDisallowCreate):                             "SchemaRuleViewDisallowCreate",
		string(advisor.SchemaRuleFunctionDisallowCreate):                         "SchemaRuleFunctionDisallowCreate",
		string(advisor.SchemaRuleFunctionDisallowList):                           "SchemaRuleFunctionDisallowList",
		string(advisor.SchemaRuleOnlineMigration):                                "SchemaRuleOnlineMigration",
	}
}
