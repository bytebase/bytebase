package advisor

import (
	"context"
	"fmt"
	"regexp"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
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

// ParseTemplateTokens parses the template and returns template tokens and their delimiters.
// For example, if the template is "{{DB_NAME}}_hello_{{LOCATION}}", then the tokens will be ["{{DB_NAME}}", "{{LOCATION}}"],
// and the delimiters will be ["_hello_"].
// The caller will usually replace the tokens with a normal string, or a regexp. In the latter case, it will be a problem
// if there are special regexp characters such as "$" in the delimiters. The caller should escape the delimiters in such cases.
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
		checkContext.StatementsTotalSize = len(statements)
		checkContext.Rule = rule
		checkContext.ParsedStatements = stmts

		adviceList, err := Check(
			ctx,
			checkContext.DBType,
			ruleType,
			checkContext,
		)
		if err != nil {
			errorAdvices = append(errorAdvices, &storepb.Advice{
				Status:  storepb.Advice_ERROR,
				Code:    code.Internal.Int32(),
				Title:   ruleType.String(),
				Content: fmt.Sprintf("Rule check failed: %v", err),
			})
			continue
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
