package pg

import (
	"context"
	"fmt"
	"regexp"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*NamingTableConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_TABLE, &NamingTableConventionAdvisor{})
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

	rule := &namingTableConventionRule{
		BaseRule: BaseRule{
			level: level,
			title: checkCtx.Rule.Type.String(),
		},
		format:    format,
		maxLength: maxLength,
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

type namingTableConventionRule struct {
	BaseRule

	format    *regexp.Regexp
	maxLength int
}

func (*namingTableConventionRule) Name() string {
	return string(storepb.SQLReviewRule_NAMING_TABLE)
}

func (r *namingTableConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		r.handleCreatestmt(ctx.(*parser.CreatestmtContext))
	case "Renamestmt":
		r.handleRenamestmt(ctx.(*parser.RenamestmtContext))
	default:
	}
	return nil
}

func (*namingTableConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleCreatestmt handles CREATE TABLE
func (r *namingTableConventionRule) handleCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	allQualifiedNames := ctx.AllQualified_name()
	if len(allQualifiedNames) > 0 {
		tableName := extractTableName(allQualifiedNames[0])
		r.checkTableName(tableName, ctx)
	}
}

// handleRenamestmt handles ALTER TABLE RENAME TO
func (r *namingTableConventionRule) handleRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check for ALTER TABLE ... RENAME TO new_name
	if ctx.TABLE() != nil && ctx.TO() != nil {
		allNames := ctx.AllName()
		if len(allNames) > 0 {
			// The new table name is the last Name() in RENAME TO new_name
			newTableName := pgparser.NormalizePostgreSQLName(allNames[len(allNames)-1])
			r.checkTableName(newTableName, ctx)
		}
	}
}

func (r *namingTableConventionRule) checkTableName(tableName string, ctx antlr.ParserRuleContext) {
	if !r.format.MatchString(tableName) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NamingTableConventionMismatch.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, r.format),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
	if r.maxLength > 0 && len(tableName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NamingTableConventionMismatch.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, r.maxLength),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}
