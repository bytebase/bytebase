package mysql

import (
	"context"
	"fmt"
	"regexp"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*NamingTableConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_TABLE, &NamingTableConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_NAMING_TABLE, &NamingTableConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_NAMING_TABLE, &NamingTableConventionAdvisor{})
}

// NamingTableConventionAdvisor is the advisor checking for table naming convention.
type NamingTableConventionAdvisor struct {
}

// Check checks for table naming convention.
func (*NamingTableConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for naming table rule")
	}

	format, err := regexp.Compile(namingPayload.Format)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}

	// Create the rule
	rule := NewNamingTableRule(level, checkCtx.Rule.Type.String(), format, maxLength)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}
	return checker.GetAdviceList(), nil
}

// NamingTableRule checks for table naming conventions.
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
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	case NodeTypeRenameTableStatement:
		r.checkRenameTableStatement(ctx.(*mysql.RenameTableStatementContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*NamingTableRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	// This rule doesn't need exit processing
	return nil
}

func (r *NamingTableRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	r.handleTableName(tableName, ctx.GetStart().GetLine())
}

func (r *NamingTableRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.AlterTableActions() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item.RENAME_SYMBOL() == nil {
			continue
		}
		if item.TableName() == nil {
			continue
		}
		_, tableName := mysqlparser.NormalizeMySQLTableName(item.TableName())
		r.handleTableName(tableName, ctx.GetStart().GetLine())
	}
}

func (r *NamingTableRule) checkRenameTableStatement(ctx *mysql.RenameTableStatementContext) {
	for _, pair := range ctx.AllRenamePair() {
		if pair.TableName() == nil {
			continue
		}
		_, tableName := mysqlparser.NormalizeMySQLTableName(pair.TableName())
		r.handleTableName(tableName, ctx.GetStart().GetLine())
	}
}

func (r *NamingTableRule) handleTableName(tableName string, lineNumber int) {
	lineNumber += r.baseLine
	if !r.format.MatchString(tableName) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("`%s` mismatches table naming convention, naming format should be %q", tableName, r.format),
			StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
		})
	}
	if r.maxLength > 0 && len(tableName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("`%s` mismatches table naming convention, its length should be within %d characters", tableName, r.maxLength),
			StartPosition: common.ConvertANTLRLineToPosition(lineNumber),
		})
	}
}
