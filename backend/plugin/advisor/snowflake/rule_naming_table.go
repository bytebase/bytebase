// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NamingTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, advisor.SchemaRuleTableNaming, &NamingTableAdvisor{})
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

	rule := NewNamingTableRule(level, string(checkCtx.Rule.Type), format, maxLength)
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// NamingTableRule checks for table naming convention.
type NamingTableRule struct {
	BaseRule
	format    *regexp.Regexp
	maxLength int
}

// NewNamingTableRule creates a new NamingTableRule.
func NewNamingTableRule(level storepb.Advice_Status, title string, format *regexp.Regexp, maxLength int) *NamingTableRule {
	return &NamingTableRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		format:    format,
		maxLength: maxLength,
	}
}

// Name returns the rule name.
func (*NamingTableRule) Name() string {
	return "NamingTableRule"
}

// OnEnter is called when entering a parse tree node.
func (r *NamingTableRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.enterCreateTable(ctx.(*parser.Create_tableContext))
	case NodeTypeAlterTable:
		r.enterAlterTable(ctx.(*parser.Alter_tableContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*NamingTableRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *NamingTableRule) enterCreateTable(ctx *parser.Create_tableContext) {
	objectName := ctx.Object_name().GetO().GetText()
	tableName := strings.TrimPrefix(strings.TrimSuffix(objectName, `"`), `"`)

	if !r.format.MatchString(tableName) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, r.format),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
	if r.maxLength > 0 && len(tableName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, r.maxLength),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}

func (r *NamingTableRule) enterAlterTable(ctx *parser.Alter_tableContext) {
	if ctx.RENAME() == nil {
		return
	}

	allObjectNames := ctx.AllObject_name()
	if len(allObjectNames) != 2 {
		slog.Warn("Unexpected number of object names in alter table rename statement", slog.Int("objectNameCount", len(allObjectNames)))
		return
	}

	newObjectName := allObjectNames[1].GetO().GetText()
	tableName := strings.TrimPrefix(strings.TrimSuffix(newObjectName, `"`), `"`)

	if !r.format.MatchString(tableName) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, r.format),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
	if r.maxLength > 0 && len(tableName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, r.maxLength),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}
