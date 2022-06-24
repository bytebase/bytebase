package advisor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"

	"github.com/bytebase/bytebase/plugin/catalog"
	"github.com/bytebase/bytebase/plugin/db"
)

// How to add a schema review rule:
//   1. Implement an advisor.(plugin/advisor/mysql or plugin/advisor/pg)
//   2. Register this advisor in map[db.Type][AdvisorType].(plugin/advisor.go)
//   3. Add advisor error code if needed. After new code added, we also need to update the ConvertToErrorCode in api/task_check_run.go
//   4. Map SchemaReviewRuleType to advisor.Type in getAdvisorTypeByRule(current file).

// SchemaReviewRuleLevel is the error level for schema review rule.
type SchemaReviewRuleLevel string

// SchemaReviewRuleType is the type of schema rule.
type SchemaReviewRuleType string

const (
	// SchemaRuleLevelError is the error level of SchemaReviewRuleLevel.
	SchemaRuleLevelError SchemaReviewRuleLevel = "ERROR"
	// SchemaRuleLevelWarning is the warning level of SchemaReviewRuleLevel.
	SchemaRuleLevelWarning SchemaReviewRuleLevel = "WARNING"
	// SchemaRuleLevelDisabled is the disabled level of SchemaReviewRuleLevel.
	SchemaRuleLevelDisabled SchemaReviewRuleLevel = "DISABLED"

	// SchemaRuleMySQLEngine require InnoDB as the storage engine.
	SchemaRuleMySQLEngine SchemaReviewRuleType = "engine.mysql.use-innodb"

	// SchemaRuleTableNaming enforce the table name format.
	SchemaRuleTableNaming SchemaReviewRuleType = "naming.table"
	// SchemaRuleColumnNaming enforce the column name format
	SchemaRuleColumnNaming SchemaReviewRuleType = "naming.column"
	// SchemaRuleUKNaming enforce the unique key name format.
	SchemaRuleUKNaming SchemaReviewRuleType = "naming.index.uk"
	// SchemaRuleFKNaming enforce the foreign key name format.
	SchemaRuleFKNaming SchemaReviewRuleType = "naming.index.fk"
	// SchemaRuleIDXNaming enforce the index name format.
	SchemaRuleIDXNaming SchemaReviewRuleType = "naming.index.idx"

	// SchemaRuleStatementNoSelectAll disallow 'SELECT *'.
	SchemaRuleStatementNoSelectAll SchemaReviewRuleType = "statement.select.no-select-all"
	// SchemaRuleStatementRequireWhere require 'WHERE' clause.
	SchemaRuleStatementRequireWhere SchemaReviewRuleType = "statement.where.require"
	// SchemaRuleStatementNoLeadingWildcardLike disallow leading '%' in LIKE, e.g. LIKE foo = '%x' is not allowed.
	SchemaRuleStatementNoLeadingWildcardLike SchemaReviewRuleType = "statement.where.no-leading-wildcard-like"

	// SchemaRuleTableRequirePK require the table to have a primary key.
	SchemaRuleTableRequirePK SchemaReviewRuleType = "table.require-pk"

	// SchemaRuleRequiredColumn enforce the required columns in each table.
	SchemaRuleRequiredColumn SchemaReviewRuleType = "column.required"
	// SchemaRuleColumnNotNull enforce the columns cannot have NULL value.
	SchemaRuleColumnNotNull SchemaReviewRuleType = "column.no-null"

	// SchemaRuleSchemaBackwardCompatibility enforce the MySQL and TiDB support check whether the schema change is backward compatible.
	SchemaRuleSchemaBackwardCompatibility SchemaReviewRuleType = "schema.backward-compatibility"

	// TableNameTemplateToken is the token for table name
	TableNameTemplateToken = "{{table}}"
	// ColumnListTemplateToken is the token for column name list
	ColumnListTemplateToken = "{{column_list}}"
	// ReferencingTableNameTemplateToken is the token for referencing table name
	ReferencingTableNameTemplateToken = "{{referencing_table}}"
	// ReferencingColumnNameTemplateToken is the token for referencing column name
	ReferencingColumnNameTemplateToken = "{{referencing_column}}"
	// ReferencedTableNameTemplateToken is the token for referenced table name
	ReferencedTableNameTemplateToken = "{{referenced_table}}"
	// ReferencedColumnNameTemplateToken is the token for referenced column name
	ReferencedColumnNameTemplateToken = "{{referenced_column}}"
)

var (
	// TemplateNamingTokens is the mapping for rule type to template token
	TemplateNamingTokens = map[SchemaReviewRuleType]map[string]bool{
		SchemaRuleIDXNaming: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		SchemaRuleUKNaming: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		SchemaRuleFKNaming: {
			ReferencingTableNameTemplateToken:  true,
			ReferencingColumnNameTemplateToken: true,
			ReferencedTableNameTemplateToken:   true,
			ReferencedColumnNameTemplateToken:  true,
		},
	}
)

// SchemaReviewPolicy is the policy configuration for schema review.
type SchemaReviewPolicy struct {
	Name     string              `json:"name"`
	RuleList []*SchemaReviewRule `json:"ruleList"`
}

// Validate validates the SchemaReviewPolicy. It also validates the each review rule.
func (policy *SchemaReviewPolicy) Validate() error {
	if policy.Name == "" || len(policy.RuleList) == 0 {
		return fmt.Errorf("invalid payload, name or rule list cannot be empty")
	}
	for _, rule := range policy.RuleList {
		if err := rule.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// SchemaReviewRule is the rule for schema review policy.
type SchemaReviewRule struct {
	Type  SchemaReviewRuleType  `json:"type"`
	Level SchemaReviewRuleLevel `json:"level"`
	// Payload is the stringify value for XXXRulePayload (e.g. NamingRulePayload, RequiredColumnRulePayload)
	// If the rule doesn't have any payload configuration, the payload would be "{}"
	Payload string `json:"payload"`
}

// Validate validates the schema review rule.
func (rule *SchemaReviewRule) Validate() error {
	// TODO(rebelice): add other schema review rule validation.
	switch rule.Type {
	case SchemaRuleTableNaming, SchemaRuleColumnNaming:
		if _, err := UnamrshalNamingRulePayloadAsRegexp(rule.Payload); err != nil {
			return err
		}
	case SchemaRuleFKNaming, SchemaRuleIDXNaming, SchemaRuleUKNaming:
		if _, _, err := UnmarshalNamingRulePayloadAsTemplate(rule.Type, rule.Payload); err != nil {
			return err
		}
	case SchemaRuleRequiredColumn:
		if _, err := UnmarshalRequiredColumnRulePayload(rule.Payload); err != nil {
			return err
		}
	}
	return nil
}

// NamingRulePayload is the payload for naming rule.
type NamingRulePayload struct {
	Format string `json:"format"`
}

// RequiredColumnRulePayload is the payload for required column rule.
type RequiredColumnRulePayload struct {
	ColumnList []string `json:"columnList"`
}

// UnamrshalNamingRulePayloadAsRegexp will unmarshal payload to NamingRulePayload and compile it as regular expression.
func UnamrshalNamingRulePayloadAsRegexp(payload string) (*regexp.Regexp, error) {
	var nr NamingRulePayload
	if err := json.Unmarshal([]byte(payload), &nr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal naming rule payload %q: %q", payload, err)
	}
	format, err := regexp.Compile(nr.Format)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regular expression: %v, err: %v", nr.Format, err)
	}
	return format, nil
}

// UnmarshalNamingRulePayloadAsTemplate will unmarshal payload to NamingRulePayload and extract all the template keys.
// For example, "hard_code_{{table}}_{{column}}_end" will return
// "hard_code_{{table}}_{{column}}_end", ["{{table}}", "{{column}}"]
func UnmarshalNamingRulePayloadAsTemplate(ruleType SchemaReviewRuleType, payload string) (string, []string, error) {
	var nr NamingRulePayload
	if err := json.Unmarshal([]byte(payload), &nr); err != nil {
		return "", nil, fmt.Errorf("failed to unmarshal naming rule payload %q: %q", payload, err)
	}

	template := nr.Format
	keys, _ := parseTemplateTokens(template)

	for _, key := range keys {
		if _, ok := TemplateNamingTokens[ruleType][key]; !ok {
			return "", nil, fmt.Errorf("invalid template %s for rule %s", key, ruleType)
		}
	}

	return template, keys, nil
}

// parseTemplateTokens parses the template and returns template tokens and their delimiters.
// For example, if the template is "{{DB_NAME}}_hello_{{LOCATION}}", then the tokens will be ["{{DB_NAME}}", "{{LOCATION}}"],
// and the delimiters will be ["_hello_"].
// The caller will usually replace the tokens with a normal string, or a regexp. In the latter case, it will be a problem
// if there are special regexp characters like "$" in the delimiters. The caller should escape the delimiters in such cases.
func parseTemplateTokens(template string) ([]string, []string) {
	r := regexp.MustCompile(`{{[^{}]+}}`)
	tokens := r.FindAllString(template, -1)
	if len(tokens) > 0 {
		split := r.Split(template, -1)
		var delimiters []string
		for _, s := range split {
			if s != "" {
				delimiters = append(delimiters, s)
			}
		}
		return tokens, delimiters
	}
	return nil, nil
}

// UnmarshalRequiredColumnRulePayload will unmarshal payload to RequiredColumnRulePayload.
func UnmarshalRequiredColumnRulePayload(payload string) (*RequiredColumnRulePayload, error) {
	var rcr RequiredColumnRulePayload
	if err := json.Unmarshal([]byte(payload), &rcr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal required column rule payload %q: %q", payload, err)
	}
	if len(rcr.ColumnList) == 0 {
		return nil, fmt.Errorf("invalid required column rule payload, column list cannot be empty")
	}
	return &rcr, nil
}

// SchemaReviewCheckContext is the context for schema review check.
type SchemaReviewCheckContext struct {
	Charset   string
	Collation string
	DbType    db.Type
	Catalog   catalog.Catalog
}

// SchemaReviewCheck checks the statments with schema review policy.
func SchemaReviewCheck(ctx context.Context, statements string, policy *SchemaReviewPolicy, context SchemaReviewCheckContext) ([]Advice, error) {
	var result []Advice
	for _, rule := range policy.RuleList {
		if rule.Level == SchemaRuleLevelDisabled {
			continue
		}

		advisorType, err := getAdvisorTypeByRule(rule.Type, context.DbType)
		if err != nil {
			log.Printf("not supported rule: %v. error:  %v\n", rule.Type, err)
			continue
		}

		adviceList, err := Check(
			context.DbType,
			advisorType,
			Context{
				Charset:   context.Charset,
				Collation: context.Collation,
				Rule:      rule,
				Catalog:   context.Catalog,
			},
			statements,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to check statement: %w", err)
		}

		result = append(result, adviceList...)
	}

	// There may be multiple syntax errors, return one only.
	if len(result) > 0 && result[0].Title == SyntaxErrorTitle {
		return result[:1], nil
	}
	if len(result) == 0 {
		result = append(result, Advice{
			Status:  Success,
			Code:    Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return result, nil
}

func getAdvisorTypeByRule(ruleType SchemaReviewRuleType, engine db.Type) (Type, error) {
	switch ruleType {
	case SchemaRuleStatementRequireWhere:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLWhereRequirement, nil
		}
	case SchemaRuleStatementNoLeadingWildcardLike:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLNoLeadingWildcardLike, nil
		}
	case SchemaRuleStatementNoSelectAll:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLNoSelectAll, nil
		}
	case SchemaRuleSchemaBackwardCompatibility:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLMigrationCompatibility, nil
		}
	case SchemaRuleTableNaming:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLNamingTableConvention, nil
		case db.Postgres:
			return PostgreSQLNamingTableConvention, nil
		}
	case SchemaRuleIDXNaming:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLNamingIndexConvention, nil
		}
	case SchemaRuleUKNaming:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLNamingUKConvention, nil
		}
	case SchemaRuleFKNaming:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLNamingFKConvention, nil
		}
	case SchemaRuleColumnNaming:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLNamingColumnConvention, nil
		case db.Postgres:
			return PostgreSQLNamingColumnConvention, nil
		}
	case SchemaRuleRequiredColumn:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLColumnRequirement, nil
		}
	case SchemaRuleColumnNotNull:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLColumnNoNull, nil
		}
	case SchemaRuleTableRequirePK:
		switch engine {
		case db.MySQL, db.TiDB:
			return MySQLTableRequirePK, nil
		}
	case SchemaRuleMySQLEngine:
		if engine == db.MySQL {
			return MySQLUseInnoDB, nil
		}
	}
	return Fake, fmt.Errorf("unknown schema review rule type %v for %v", ruleType, engine)
}
