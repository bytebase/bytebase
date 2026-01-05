package pg

import (
	"context"
	"fmt"
	"regexp"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*TableDropNamingConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION, &TableDropNamingConventionAdvisor{})
}

// TableDropNamingConventionAdvisor is the advisor checking for table drop with naming convention.
type TableDropNamingConventionAdvisor struct {
}

// Check checks for table drop with naming convention.
func (*TableDropNamingConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for table drop naming convention rule")
	}

	format, err := regexp.Compile(namingPayload.Format)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
	}

	rule := &tableDropNamingConventionRule{
		BaseRule: BaseRule{
			level: level,
			title: checkCtx.Rule.Type.String(),
		},
		format: format,
	}

	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

type tableDropNamingConventionRule struct {
	BaseRule
	format *regexp.Regexp
}

// Name returns the rule name.
func (*tableDropNamingConventionRule) Name() string {
	return "table.drop-naming-convention"
}

// OnEnter is called when the parser enters a rule context.
func (r *tableDropNamingConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Dropstmt":
		r.handleDropstmt(ctx.(*parser.DropstmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*tableDropNamingConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *tableDropNamingConventionRule) handleDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is a DROP TABLE statement
	if ctx.Object_type_any_name() == nil || ctx.Object_type_any_name().TABLE() == nil {
		return
	}

	// Check all tables being dropped
	if ctx.Any_name_list() != nil {
		allNames := ctx.Any_name_list().AllAny_name()
		for _, nameCtx := range allNames {
			tableName := r.extractTableNameFromAnyName(nameCtx)
			if tableName != "" && !r.format.MatchString(tableName) {
				r.AddAdvice(&storepb.Advice{
					Status:  r.level,
					Code:    code.TableDropNamingConventionMismatch.Int32(),
					Title:   r.title,
					Content: fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", tableName, r.format),
					StartPosition: &storepb.Position{
						Line:   int32(ctx.GetStart().GetLine()),
						Column: 0,
					},
				})
			}
		}
	}
}

// extractTableNameFromAnyName extracts the table name from Any_name context.
// For schema.table, returns "table". For just "table", returns "table".
func (*tableDropNamingConventionRule) extractTableNameFromAnyName(ctx parser.IAny_nameContext) string {
	parts := pg.NormalizePostgreSQLAnyName(ctx)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
