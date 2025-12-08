package advisor

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"

	// Register walk-through implementations
	_ "github.com/bytebase/bytebase/backend/plugin/schema/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/pg"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/tidb"
)

// How to add a SQL review rule:
//   1. Implement an advisor.(plugin/advisor/mysql or plugin/advisor/pg)
//   2. Register this advisor in map[storepb.Engine][storepb.SQLReviewRule_Type].(plugin/advisor.go)
//   3. Add advisor error code if needed(plugin/advisor/code.go).

const (
	// TableNameTemplateToken is the token for table name.
	TableNameTemplateToken = "{{table}}"
	// ColumnListTemplateToken is the token for column name list.
	ColumnListTemplateToken = "{{column_list}}"
	// ReferencingTableNameTemplateToken is the token for referencing table name.
	ReferencingTableNameTemplateToken = "{{referencing_table}}"
	// ReferencingColumnNameTemplateToken is the token for referencing column name.
	ReferencingColumnNameTemplateToken = "{{referencing_column}}"
	// ReferencedTableNameTemplateToken is the token for referenced table name.
	ReferencedTableNameTemplateToken = "{{referenced_table}}"
	// ReferencedColumnNameTemplateToken is the token for referenced column name.
	ReferencedColumnNameTemplateToken = "{{referenced_column}}"

	// DefaultNameLengthLimit is the default length limit for naming rules.
	// PostgreSQL has it's own naming length limit, will auto slice the name to make sure its length <= 63
	// https://www.postgresql.org/docs/current/limits.html.
	// While MySQL does not enforce the limit, thus we use PostgreSQL's 63 as the default limit.
	DefaultNameLengthLimit = 63
)

var (
	// TemplateNamingTokens is the mapping for rule type to template token.
	TemplateNamingTokens = map[storepb.SQLReviewRule_Type]map[string]bool{
		storepb.SQLReviewRule_NAMING_INDEX_IDX: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		storepb.SQLReviewRule_NAMING_INDEX_PK: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		storepb.SQLReviewRule_NAMING_INDEX_UK: {
			TableNameTemplateToken:  true,
			ColumnListTemplateToken: true,
		},
		storepb.SQLReviewRule_NAMING_INDEX_FK: {
			ReferencingTableNameTemplateToken:  true,
			ReferencingColumnNameTemplateToken: true,
			ReferencedTableNameTemplateToken:   true,
			ReferencedColumnNameTemplateToken:  true,
		},
	}
)

// NamingRulePayload is the payload for naming rule.
//
// Deprecated: Used only for JSON conversion. Use storepb.NamingRulePayload instead.
type NamingRulePayload struct {
	MaxLength int    `json:"maxLength"`
	Format    string `json:"format"`
}

// StringArrayTypeRulePayload is the payload for rules with string array value.
//
// Deprecated: Used only for JSON conversion. Use storepb.StringArrayRulePayload instead.
type StringArrayTypeRulePayload struct {
	List []string `json:"list"`
}

// RequiredColumnRulePayload is the payload for required column rule.
//
// Deprecated: Used only for JSON conversion. Use storepb.RequiredColumnRulePayload instead.
type RequiredColumnRulePayload struct {
	ColumnList []string `json:"columnList"`
}

// CommentConventionRulePayload is the payload for comment convention rule.
//
// Deprecated: Used only for JSON conversion. Use storepb.CommentConventionRulePayload instead.
type CommentConventionRulePayload struct {
	Required  bool `json:"required"`
	MaxLength int  `json:"maxLength"`
}

// NumberTypeRulePayload is the number type payload.
//
// Deprecated: Used only for JSON conversion. Use storepb.NumberRulePayload instead.
type NumberTypeRulePayload struct {
	Number int `json:"number"`
}

// StringTypeRulePayload is the string type payload.
//
// Deprecated: Used only for JSON conversion. Use storepb.StringRulePayload instead.
type StringTypeRulePayload struct {
	String string `json:"string"`
}

// NamingCaseRulePayload is the payload for naming case rule.
//
// Deprecated: Used only for JSON conversion. Use storepb.NamingCaseRulePayload instead.
type NamingCaseRulePayload struct {
	// Upper is true means the case should be upper case, otherwise lower case.
	Upper bool `json:"upper"`
}

// UnmarshalNamingRulePayloadAsRegexp will unmarshal payload to NamingRulePayload and compile it as regular expression.
//
// Deprecated: Use rule.GetNamingPayload() instead.
func UnmarshalNamingRulePayloadAsRegexp(payload string) (*regexp.Regexp, int, error) {
	var nr NamingRulePayload
	if err := json.Unmarshal([]byte(payload), &nr); err != nil {
		return nil, 0, errors.Wrapf(err, "failed to unmarshal naming rule payload %q", payload)
	}

	format, err := regexp.Compile(nr.Format)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "failed to compile regular expression \"%s\"", nr.Format)
	}

	// We need to be compatible with existed naming rules in the database. 0 means using the default length limit.
	maxLength := nr.MaxLength
	if maxLength == 0 {
		maxLength = DefaultNameLengthLimit
	}

	return format, maxLength, nil
}

// UnmarshalNamingRulePayloadAsTemplate will unmarshal payload to NamingRulePayload and extract all the template keys.
// For example, "hard_code_{{table}}_{{column}}_end" will return
// "hard_code_{{table}}_{{column}}_end", ["{{table}}", "{{column}}"].
//
// Deprecated: Use rule.GetNamingPayload() instead.
func UnmarshalNamingRulePayloadAsTemplate(ruleType storepb.SQLReviewRule_Type, payload string) (string, []string, int, error) {
	var nr NamingRulePayload
	if err := json.Unmarshal([]byte(payload), &nr); err != nil {
		return "", nil, 0, errors.Wrapf(err, "failed to unmarshal naming rule payload %q", payload)
	}

	template := nr.Format
	keys, _ := ParseTemplateTokens(template)

	for _, key := range keys {
		if _, ok := TemplateNamingTokens[ruleType][key]; !ok {
			return "", nil, 0, errors.Errorf("invalid template %s for rule %s", key, ruleType)
		}
	}

	// We need to be compatible with existed naming rules in the database. 0 means using the default length limit.
	maxLength := nr.MaxLength
	if maxLength == 0 {
		maxLength = DefaultNameLengthLimit
	}

	return template, keys, maxLength, nil
}

// parseTemplateTokens parses the template and returns template tokens and their delimiters.
// For example, if the template is "{{DB_NAME}}_hello_{{LOCATION}}", then the tokens will be ["{{DB_NAME}}", "{{LOCATION}}"],
// and the delimiters will be ["_hello_"].
// The caller will usually replace the tokens with a normal string, or a regexp. In the latter case, it will be a problem
// if there are special regexp characters such as "$" in the delimiters. The caller should escape the delimiters in such cases.
// ParseTemplateTokens parses the template and returns template tokens and their delimiters.
func ParseTemplateTokens(template string) ([]string, []string) {
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

// UnmarshalRequiredColumnList will unmarshal payload and parse the required column list.
//
// Deprecated: Use rule.GetStringArrayPayload() instead.
func UnmarshalRequiredColumnList(payload string) ([]string, error) {
	stringArrayRulePayload, err := UnmarshalStringArrayTypeRulePayload(payload)
	if err != nil {
		return nil, err
	}
	if len(stringArrayRulePayload.List) != 0 {
		return stringArrayRulePayload.List, nil
	}

	// The RequiredColumnRulePayload is deprecated.
	// Just keep it to compatible with old data, and we can remove this later.
	columnRulePayload, err := unmarshalRequiredColumnRulePayload(payload)
	if err != nil {
		return nil, err
	}

	return columnRulePayload.ColumnList, nil
}

// unmarshalRequiredColumnRulePayload will unmarshal payload to RequiredColumnRulePayload.
func unmarshalRequiredColumnRulePayload(payload string) (*RequiredColumnRulePayload, error) {
	var rcr RequiredColumnRulePayload
	if err := json.Unmarshal([]byte(payload), &rcr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal required column rule payload %q", payload)
	}
	if len(rcr.ColumnList) == 0 {
		return nil, errors.Errorf("invalid required column rule payload, column list cannot be empty")
	}
	return &rcr, nil
}

// UnmarshalCommentConventionRulePayload will unmarshal payload to CommentConventionRulePayload.
//
// Deprecated: Use rule.GetCommentConventionPayload() instead.
func UnmarshalCommentConventionRulePayload(payload string) (*CommentConventionRulePayload, error) {
	var ccr CommentConventionRulePayload
	if err := json.Unmarshal([]byte(payload), &ccr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal comment convention rule payload %q", payload)
	}
	return &ccr, nil
}

// UnmarshalNumberTypeRulePayload will unmarshal payload to NumberTypeRulePayload.
//
// Deprecated: Use rule.GetNumberPayload() instead.
func UnmarshalNumberTypeRulePayload(payload string) (*NumberTypeRulePayload, error) {
	var nlr NumberTypeRulePayload
	if err := json.Unmarshal([]byte(payload), &nlr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal number type rule payload %q", payload)
	}
	return &nlr, nil
}

// UnmarshalStringTypeRulePayload will unmarshal payload to StringTypeRulePayload.
//
// Deprecated: Use rule.GetStringPayload() instead.
func UnmarshalStringTypeRulePayload(payload string) (*StringTypeRulePayload, error) {
	var slr StringTypeRulePayload
	if err := json.Unmarshal([]byte(payload), &slr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal string type rule payload %q", payload)
	}
	return &slr, nil
}

// UnmarshalStringArrayTypeRulePayload will unmarshal payload to StringArrayTypeRulePayload.
//
// Deprecated: Use rule.GetStringArrayPayload() instead.
func UnmarshalStringArrayTypeRulePayload(payload string) (*StringArrayTypeRulePayload, error) {
	var trr StringArrayTypeRulePayload
	if err := json.Unmarshal([]byte(payload), &trr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal string array rule payload %q", payload)
	}
	return &trr, nil
}

// UnmarshalNamingCaseRulePayload will unmarshal payload to NamingCaseRulePayload.
//
// Deprecated: Use rule.GetNamingCasePayload() instead.
func UnmarshalNamingCaseRulePayload(payload string) (*NamingCaseRulePayload, error) {
	var ncr NamingCaseRulePayload
	if err := json.Unmarshal([]byte(payload), &ncr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal naming case rule payload %q", payload)
	}
	return &ncr, nil
}

// GetPayloadTypeForRule returns the expected payload type for a given rule type.
// This is used for validation and conversion between JSON and proto formats.
//
// Deprecated: Used only for test backward compatibility.
func GetPayloadTypeForRule(ruleType storepb.SQLReviewRule_Type) string {
	switch ruleType {
	// Naming rules use NamingRulePayload
	case storepb.SQLReviewRule_NAMING_TABLE,
		storepb.SQLReviewRule_NAMING_COLUMN,
		storepb.SQLReviewRule_NAMING_INDEX_PK,
		storepb.SQLReviewRule_NAMING_INDEX_UK,
		storepb.SQLReviewRule_NAMING_INDEX_FK,
		storepb.SQLReviewRule_NAMING_INDEX_IDX,
		storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT,
		storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION:
		return "naming"

	// Number rules use NumberRulePayload
	case storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
		storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
		storepb.SQLReviewRule_STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT,
		storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE,
		storepb.SQLReviewRule_STATEMENT_MAXIMUM_JOIN_TABLE_COUNT,
		storepb.SQLReviewRule_STATEMENT_MAXIMUM_STATEMENTS_IN_TRANSACTION,
		storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME,
		storepb.SQLReviewRule_STATEMENT_QUERY_MINIMUM_PLAN_LEVEL,
		storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH,
		storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH,
		storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE,
		storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT,
		storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT,
		storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT,
		storepb.SQLReviewRule_TABLE_TEXT_FIELDS_TOTAL_LENGTH,
		storepb.SQLReviewRule_TABLE_LIMIT_SIZE,
		storepb.SQLReviewRule_SYSTEM_COMMENT_LENGTH:
		return "number"

	// String array rules use StringArrayRulePayload
	case storepb.SQLReviewRule_COLUMN_REQUIRED,
		storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST,
		storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST,
		storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST,
		storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST,
		storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST,
		storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOWED_LIST,
		storepb.SQLReviewRule_TABLE_DISALLOW_DDL,
		storepb.SQLReviewRule_TABLE_DISALLOW_DML:
		return "string_array"

	// Comment convention rules use CommentConventionRulePayload
	case storepb.SQLReviewRule_TABLE_COMMENT,
		storepb.SQLReviewRule_COLUMN_COMMENT:
		return "comment_convention"

	// Naming case rule uses NamingCaseRulePayload
	case storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE:
		return "naming_case"

	// String rules use StringRulePayload
	case storepb.SQLReviewRule_NAMING_FULLY_QUALIFIED:
		return "string"

	// Rules with no payload
	default:
		return "none"
	}
}

// ConvertJSONPayloadToProto converts a JSON string payload to the appropriate proto payload type.
//
// Deprecated: Used only for test backward compatibility. Use typed proto payloads directly.
func ConvertJSONPayloadToProto(rule *storepb.SQLReviewRule, jsonPayload string) error {
	if jsonPayload == "" || jsonPayload == "{}" {
		return nil
	}

	payloadType := GetPayloadTypeForRule(rule.Type)

	switch payloadType {
	case "naming":
		var payload NamingRulePayload
		if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
			return errors.Wrapf(err, "failed to unmarshal naming payload for rule %s", rule.Type)
		}
		rule.Payload = &storepb.SQLReviewRule_NamingPayload{
			NamingPayload: &storepb.NamingRulePayload{
				MaxLength: int32(payload.MaxLength),
				Format:    payload.Format,
			},
		}

	case "number":
		var payload NumberTypeRulePayload
		if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
			return errors.Wrapf(err, "failed to unmarshal number payload for rule %s", rule.Type)
		}
		rule.Payload = &storepb.SQLReviewRule_NumberPayload{
			NumberPayload: &storepb.NumberRulePayload{
				Number: int32(payload.Number),
			},
		}

	case "string_array":
		var payload StringArrayTypeRulePayload
		if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
			return errors.Wrapf(err, "failed to unmarshal string array payload for rule %s", rule.Type)
		}
		rule.Payload = &storepb.SQLReviewRule_StringArrayPayload{
			StringArrayPayload: &storepb.StringArrayRulePayload{
				List: payload.List,
			},
		}

	case "comment_convention":
		var payload CommentConventionRulePayload
		if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
			return errors.Wrapf(err, "failed to unmarshal comment convention payload for rule %s", rule.Type)
		}
		rule.Payload = &storepb.SQLReviewRule_CommentConventionPayload{
			CommentConventionPayload: &storepb.CommentConventionRulePayload{
				Required:  payload.Required,
				MaxLength: int32(payload.MaxLength),
			},
		}

	case "naming_case":
		var payload NamingCaseRulePayload
		if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
			return errors.Wrapf(err, "failed to unmarshal naming case payload for rule %s", rule.Type)
		}
		rule.Payload = &storepb.SQLReviewRule_NamingCasePayload{
			NamingCasePayload: &storepb.NamingCaseRulePayload{
				Upper: payload.Upper,
			},
		}

	case "string":
		var payload StringTypeRulePayload
		if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
			return errors.Wrapf(err, "failed to unmarshal string payload for rule %s", rule.Type)
		}
		rule.Payload = &storepb.SQLReviewRule_StringPayload{
			StringPayload: &storepb.StringRulePayload{
				Value: payload.String,
			},
		}

	case "none":
		// No payload for this rule type
		return nil

	default:
		return errors.Errorf("unknown payload type %s for rule %s", payloadType, rule.Type)
	}

	return nil
}


// SQLReviewCheck checks the statements with sql review rules.
func SQLReviewCheck(
	ctx context.Context,
	sm *sheet.Manager,
	statements string,
	ruleList []*storepb.SQLReviewRule,
	checkContext Context,
) ([]*storepb.Advice, error) {
	stmts, parseResult := sm.GetStatementsForChecks(checkContext.DBType, statements)
	asts := base.ExtractASTs(stmts)

	builtinOnly := len(ruleList) == 0

	if !checkContext.NoAppendBuiltin {
		// Append builtin rules to the rule list.
		ruleList = append(ruleList, GetBuiltinRules(checkContext.DBType)...)
	}

	if asts == nil || len(ruleList) == 0 {
		return parseResult, nil
	}

	if !builtinOnly && checkContext.FinalMetadata != nil {
		switch checkContext.DBType {
		case storepb.Engine_TIDB, storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_POSTGRES, storepb.Engine_OCEANBASE:
			if advice := schema.WalkThrough(checkContext.DBType, checkContext.FinalMetadata, asts); advice != nil {
				return []*storepb.Advice{advice}, nil
			}
		default:
			// Other database types don't need walkthrough
		}
	}

	var errorAdvices, warningAdvices []*storepb.Advice
	for _, rule := range ruleList {
		if rule.Engine != storepb.Engine_ENGINE_UNSPECIFIED && rule.Engine != checkContext.DBType {
			continue
		}

		ruleType := rule.Type

		// Set per-rule fields
		checkContext.AST = asts
		checkContext.Statements = statements
		checkContext.Rule = rule

		adviceList, err := Check(
			ctx,
			checkContext.DBType,
			ruleType,
			checkContext,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check statement")
		}

		for _, advice := range adviceList {
			switch advice.Status {
			case storepb.Advice_ERROR:
				if len(errorAdvices) < common.MaximumAdvicePerStatus {
					errorAdvices = append(errorAdvices, advice)
				}
			case storepb.Advice_WARNING:
				if len(warningAdvices) < common.MaximumAdvicePerStatus {
					warningAdvices = append(warningAdvices, advice)
				}
			default:
			}
		}
		// Skip remaining rules if we have enough error and warning advices.
		if len(errorAdvices) >= common.MaximumAdvicePerStatus && len(warningAdvices) >= common.MaximumAdvicePerStatus {
			break
		}
	}

	var advices []*storepb.Advice
	advices = append(advices, errorAdvices...)
	advices = append(advices, warningAdvices...)
	return advices, nil
}
