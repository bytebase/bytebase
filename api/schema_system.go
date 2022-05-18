package api

import (
	"encoding/json"
	"fmt"
	"regexp"
)

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
	if policy.Empty() {
		return fmt.Errorf("invalid payload, name or rule list cannot be empty")
	}
	for _, rule := range policy.RuleList {
		if err := rule.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Empty will check if the SchemaReviewPolicy is empty.
func (policy *SchemaReviewPolicy) Empty() bool {
	return policy.Name == "" || len(policy.RuleList) == 0
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
