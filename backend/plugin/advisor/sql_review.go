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

	// defaultNameLengthLimit is the default length limit for naming rules.
	// PostgreSQL has it's own naming length limit, will auto slice the name to make sure its length <= 63
	// https://www.postgresql.org/docs/current/limits.html.
	// While MySQL does not enforce the limit, thus we use PostgreSQL's 63 as the default limit.
	defaultNameLengthLimit = 63
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
type NamingRulePayload struct {
	MaxLength int    `json:"maxLength"`
	Format    string `json:"format"`
}

// StringArrayTypeRulePayload is the payload for rules with string array value.
type StringArrayTypeRulePayload struct {
	List []string `json:"list"`
}

// RequiredColumnRulePayload is the payload for required column rule.
type RequiredColumnRulePayload struct {
	ColumnList []string `json:"columnList"`
}

// CommentConventionRulePayload is the payload for comment convention rule.
type CommentConventionRulePayload struct {
	Required  bool `json:"required"`
	MaxLength int  `json:"maxLength"`
}

// NumberTypeRulePayload is the number type payload.
type NumberTypeRulePayload struct {
	Number int `json:"number"`
}

// StringTypeRulePayload is the string type payload.
type StringTypeRulePayload struct {
	String string `json:"string"`
}

// NamingCaseRulePayload is the payload for naming case rule.
type NamingCaseRulePayload struct {
	// Upper is true means the case should be upper case, otherwise lower case.
	Upper bool `json:"upper"`
}

// UnmarshalNamingRulePayloadAsRegexp will unmarshal payload to NamingRulePayload and compile it as regular expression.
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
		maxLength = defaultNameLengthLimit
	}

	return format, maxLength, nil
}

// UnmarshalNamingRulePayloadAsTemplate will unmarshal payload to NamingRulePayload and extract all the template keys.
// For example, "hard_code_{{table}}_{{column}}_end" will return
// "hard_code_{{table}}_{{column}}_end", ["{{table}}", "{{column}}"].
func UnmarshalNamingRulePayloadAsTemplate(ruleType storepb.SQLReviewRule_Type, payload string) (string, []string, int, error) {
	var nr NamingRulePayload
	if err := json.Unmarshal([]byte(payload), &nr); err != nil {
		return "", nil, 0, errors.Wrapf(err, "failed to unmarshal naming rule payload %q", payload)
	}

	template := nr.Format
	keys, _ := parseTemplateTokens(template)

	for _, key := range keys {
		if _, ok := TemplateNamingTokens[ruleType][key]; !ok {
			return "", nil, 0, errors.Errorf("invalid template %s for rule %s", key, ruleType)
		}
	}

	// We need to be compatible with existed naming rules in the database. 0 means using the default length limit.
	maxLength := nr.MaxLength
	if maxLength == 0 {
		maxLength = defaultNameLengthLimit
	}

	return template, keys, maxLength, nil
}

// parseTemplateTokens parses the template and returns template tokens and their delimiters.
// For example, if the template is "{{DB_NAME}}_hello_{{LOCATION}}", then the tokens will be ["{{DB_NAME}}", "{{LOCATION}}"],
// and the delimiters will be ["_hello_"].
// The caller will usually replace the tokens with a normal string, or a regexp. In the latter case, it will be a problem
// if there are special regexp characters such as "$" in the delimiters. The caller should escape the delimiters in such cases.
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

// UnmarshalRequiredColumnList will unmarshal payload and parse the required column list.
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
func UnmarshalCommentConventionRulePayload(payload string) (*CommentConventionRulePayload, error) {
	var ccr CommentConventionRulePayload
	if err := json.Unmarshal([]byte(payload), &ccr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal comment convention rule payload %q", payload)
	}
	return &ccr, nil
}

// UnmarshalNumberTypeRulePayload will unmarshal payload to NumberTypeRulePayload.
func UnmarshalNumberTypeRulePayload(payload string) (*NumberTypeRulePayload, error) {
	var nlr NumberTypeRulePayload
	if err := json.Unmarshal([]byte(payload), &nlr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal number type rule payload %q", payload)
	}
	return &nlr, nil
}

// UnmarshalStringTypeRulePayload will unmarshal payload to StringTypeRulePayload.
func UnmarshalStringTypeRulePayload(payload string) (*StringTypeRulePayload, error) {
	var slr StringTypeRulePayload
	if err := json.Unmarshal([]byte(payload), &slr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal string type rule payload %q", payload)
	}
	return &slr, nil
}

// UnmarshalStringArrayTypeRulePayload will unmarshal payload to StringArrayTypeRulePayload.
func UnmarshalStringArrayTypeRulePayload(payload string) (*StringArrayTypeRulePayload, error) {
	var trr StringArrayTypeRulePayload
	if err := json.Unmarshal([]byte(payload), &trr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal string array rule payload %q", payload)
	}
	return &trr, nil
}

// UnmarshalNamingCaseRulePayload will unmarshal payload to NamingCaseRulePayload.
func UnmarshalNamingCaseRulePayload(payload string) (*NamingCaseRulePayload, error) {
	var ncr NamingCaseRulePayload
	if err := json.Unmarshal([]byte(payload), &ncr); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal naming case rule payload %q", payload)
	}
	return &ncr, nil
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
