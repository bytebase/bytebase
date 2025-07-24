// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"regexp"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*NamingTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleNamingTableConvention, &NamingTableAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleNamingTableConvention, &NamingTableAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleNamingTableConvention, &NamingTableAdvisor{})
}

// NamingTableAdvisor is the advisor checking for table naming convention.
type NamingTableAdvisor struct {
}

// Check checks for table naming convention.
func (*NamingTableAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, maxLength, err := advisor.UnmarshalNamingRulePayloadAsRegexp(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	rule := NewNamingTableRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase, format, maxLength)
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

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
			advisor.NamingTableConventionMismatch.Int32(),
			fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, r.format),
			common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		)
	}
	if r.maxLength > 0 && len(tableName) > r.maxLength {
		r.AddAdvice(
			r.level,
			advisor.NamingTableConventionMismatch.Int32(),
			fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, r.maxLength),
			common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
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
			advisor.NamingTableConventionMismatch.Int32(),
			fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, r.format),
			common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		)
	}
	if r.maxLength > 0 && len(tableName) > r.maxLength {
		r.AddAdvice(
			r.level,
			advisor.NamingTableConventionMismatch.Int32(),
			fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, r.maxLength),
			common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		)
	}
}
