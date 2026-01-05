// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"regexp"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*NamingTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_NAMING_TABLE, &NamingTableAdvisor{})
}

// NamingTableAdvisor is the advisor checking for table naming convention.
type NamingTableAdvisor struct {
}

// Check checks for table naming convention.
func (*NamingTableAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for this rule")
	}

	format, err := regexp.Compile(namingPayload.Format)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}

	rule := NewNamingTableRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase, format, maxLength)
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

	return checker.GetAdviceList()
}

// NamingTableRule is the rule implementation for table naming convention.
type NamingTableRule struct {
	BaseRule

	currentDatabase string
	format          *regexp.Regexp
	maxLength       int
}

// NewNamingTableRule creates a new NamingTableRule.
func NewNamingTableRule(level storepb.Advice_Status, title string, currentDatabase string, format *regexp.Regexp, maxLength int) *NamingTableRule {
	return &NamingTableRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		format:          format,
		maxLength:       maxLength,
	}
}

// Name returns the rule name.
func (*NamingTableRule) Name() string {
	return "naming.table"
}

// OnEnter is called when the parser enters a rule context.
func (r *NamingTableRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.handleCreateTable(ctx.(*parser.Create_tableContext))
	case "Alter_table_properties":
		r.handleAlterTableProperties(ctx.(*parser.Alter_table_propertiesContext))
	default:
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*NamingTableRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *NamingTableRule) handleCreateTable(ctx *parser.Create_tableContext) {
	tableName := normalizeIdentifier(ctx.Table_name(), r.currentDatabase)
	if !r.format.MatchString(tableName) {
		r.AddAdvice(
			r.level,
			code.NamingTableConventionMismatch.Int32(),
			fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, r.format),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	}
	if r.maxLength > 0 && len(tableName) > r.maxLength {
		r.AddAdvice(
			r.level,
			code.NamingTableConventionMismatch.Int32(),
			fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, r.maxLength),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	}
}

func (r *NamingTableRule) handleAlterTableProperties(ctx *parser.Alter_table_propertiesContext) {
	if ctx.Tableview_name() == nil {
		return
	}
	tableName := lastIdentifier(normalizeIdentifier(ctx.Tableview_name(), r.currentDatabase))
	if tableName == "" {
		return
	}
	if !r.format.MatchString(tableName) {
		r.AddAdvice(
			r.level,
			code.NamingTableConventionMismatch.Int32(),
			fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, r.format),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	}
	if r.maxLength > 0 && len(tableName) > r.maxLength {
		r.AddAdvice(
			r.level,
			code.NamingTableConventionMismatch.Int32(),
			fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, r.maxLength),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	}
}
