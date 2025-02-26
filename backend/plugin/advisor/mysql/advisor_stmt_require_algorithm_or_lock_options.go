package mysql

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*RequireAlgorithmOrLockOptionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementRequireAlgorithmOption, &RequireAlgorithmOrLockOptionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLStatementRequireAlgorithmOption, &RequireAlgorithmOrLockOptionAdvisor{})

	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementRequireLockOption, &RequireAlgorithmOrLockOptionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLStatementRequireLockOption, &RequireAlgorithmOrLockOptionAdvisor{})
}

// RequireAlgorithmOrLockOptionAdvisor is the advisor checking for the max execution time.
type RequireAlgorithmOrLockOptionAdvisor struct {
}

func (*RequireAlgorithmOrLockOptionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to stmt list")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	requiredOption, errorCode := "ALGORITHM", advisor.StatementNoAlgorithmOption
	if checkCtx.Rule.Type == string(advisor.SchemaRuleStatementRequireLockOption) {
		requiredOption, errorCode = "LOCK", advisor.StatementNoLockOption
	}
	checker := &RequireAlgorithmOptionChecker{
		requiredOption: requiredOption,
		level:          level,
		title:          string(checkCtx.Rule.Type),
		adviceList:     []*storepb.Advice{},
		errorCode:      errorCode,
	}
	for _, stmt := range stmtList {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}
	return checker.adviceList, nil
}

type RequireAlgorithmOptionChecker struct {
	*mysql.BaseMySQLParserListener

	requiredOption        string
	hasOption             bool
	inAlterTableStatement bool

	errorCode  advisor.Code
	baseLine   int
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	text       string
	line       int
}

func (checker *RequireAlgorithmOptionChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	checker.inAlterTableStatement = true
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
	checker.line = ctx.GetStart().GetLine()
}

func (checker *RequireAlgorithmOptionChecker) ExitAlterTable(*mysql.AlterTableContext) {
	if !checker.hasOption {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:  checker.level,
			Code:    int32(checker.errorCode),
			Title:   checker.title,
			Content: "ALTER TABLE statement should include " + checker.requiredOption + " option",
			StartPosition: &storepb.Position{
				Line: int32(checker.baseLine + checker.line),
			},
		})
	}
	checker.inAlterTableStatement = false
	checker.hasOption = false
}

func (checker *RequireAlgorithmOptionChecker) EnterAlterTableActions(ctx *mysql.AlterTableActionsContext) {
	if !checker.inAlterTableStatement {
		return
	}

	modifierList := []mysql.IAlterCommandsModifierContext{}
	if ctx.AlterCommandsModifierList() != nil {
		modifierList = append(modifierList, ctx.AlterCommandsModifierList().AllAlterCommandsModifier()...)
	}
	if ctx.AlterCommandList() != nil {
		if ctx.AlterCommandList().AlterCommandsModifierList() != nil {
			modifierList = append(modifierList, ctx.AlterCommandList().AlterCommandsModifierList().AllAlterCommandsModifier()...)
		}
		if ctx.AlterCommandList().AlterList() != nil {
			modifierList = append(modifierList, ctx.AlterCommandList().AlterList().AllAlterCommandsModifier()...)
		}
	}
	for _, modifier := range modifierList {
		if checker.requiredOption == "ALGORITHM" && modifier.AlterAlgorithmOption() != nil {
			if modifier.AlterAlgorithmOption().Identifier() != nil {
				algorithmOptionValue := mysqlparser.NormalizeMySQLIdentifier(modifier.AlterAlgorithmOption().Identifier())
				// Don't need to check the value of the algorithm option right now.
				if algorithmOptionValue != "" {
					checker.hasOption = true
				}
			}
		}
		if checker.requiredOption == "LOCK" && modifier.AlterLockOption() != nil {
			if modifier.AlterLockOption().Identifier() != nil {
				lockOptionValue := mysqlparser.NormalizeMySQLIdentifier(modifier.AlterLockOption().Identifier())
				// Don't need to check the value of the lock option right now.
				if lockOptionValue != "" {
					checker.hasOption = true
				}
			}
		}
	}
}
