package mysql

import (
	"context"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*RequireAlgorithmOrLockOptionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION, &RequireAlgorithmOrLockOptionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION, &RequireAlgorithmOrLockOptionAdvisor{})

	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION, &RequireAlgorithmOrLockOptionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION, &RequireAlgorithmOrLockOptionAdvisor{})
}

// RequireAlgorithmOrLockOptionAdvisor is the advisor checking for the max execution time.
type RequireAlgorithmOrLockOptionAdvisor struct {
}

func (*RequireAlgorithmOrLockOptionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	requiredOption, errorCode := "ALGORITHM", code.StatementNoAlgorithmOption
	if checkCtx.Rule.Type == storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION {
		requiredOption, errorCode = "LOCK", code.StatementNoLockOption
	}

	// Create the rule
	rule := NewRequireAlgorithmOrLockOptionRule(level, checkCtx.Rule.Type.String(), requiredOption, errorCode)

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

// RequireAlgorithmOrLockOptionRule checks for required algorithm or lock options.
type RequireAlgorithmOrLockOptionRule struct {
	BaseRule
	requiredOption        string
	hasOption             bool
	inAlterTableStatement bool
	errorCode             code.Code
	text                  string
	line                  int
}

// NewRequireAlgorithmOrLockOptionRule creates a new RequireAlgorithmOrLockOptionRule.
func NewRequireAlgorithmOrLockOptionRule(level storepb.Advice_Status, title string, requiredOption string, errorCode code.Code) *RequireAlgorithmOrLockOptionRule {
	return &RequireAlgorithmOrLockOptionRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		requiredOption: requiredOption,
		errorCode:      errorCode,
	}
}

// Name returns the rule name.
func (*RequireAlgorithmOrLockOptionRule) Name() string {
	return "RequireAlgorithmOrLockOptionRule"
}

// OnEnter is called when entering a parse tree node.
func (r *RequireAlgorithmOrLockOptionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	case NodeTypeAlterTableActions:
		r.checkAlterTableActions(ctx.(*mysql.AlterTableActionsContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *RequireAlgorithmOrLockOptionRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeAlterTable {
		if !r.hasOption {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          int32(r.errorCode),
				Title:         r.title,
				Content:       "ALTER TABLE statement should include " + r.requiredOption + " option",
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + r.line),
			})
		}
		r.inAlterTableStatement = false
		r.hasOption = false
	}
	return nil
}

func (r *RequireAlgorithmOrLockOptionRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	r.inAlterTableStatement = true
	r.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
	r.line = ctx.GetStart().GetLine()
}

func (r *RequireAlgorithmOrLockOptionRule) checkAlterTableActions(ctx *mysql.AlterTableActionsContext) {
	if !r.inAlterTableStatement {
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
		if r.requiredOption == "ALGORITHM" && modifier.AlterAlgorithmOption() != nil {
			if modifier.AlterAlgorithmOption().Identifier() != nil {
				algorithmOptionValue := mysqlparser.NormalizeMySQLIdentifier(modifier.AlterAlgorithmOption().Identifier())
				// Don't need to check the value of the algorithm option right now.
				if algorithmOptionValue != "" {
					r.hasOption = true
				}
			}
		}
		if r.requiredOption == "LOCK" && modifier.AlterLockOption() != nil {
			if modifier.AlterLockOption().Identifier() != nil {
				lockOptionValue := mysqlparser.NormalizeMySQLIdentifier(modifier.AlterLockOption().Identifier())
				// Don't need to check the value of the lock option right now.
				if lockOptionValue != "" {
					r.hasOption = true
				}
			}
		}
	}
}
