package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleSchemaBackwardCompatibility, &CompatibilityAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleSchemaBackwardCompatibility, &CompatibilityAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleSchemaBackwardCompatibility, &CompatibilityAdvisor{})
}

// CompatibilityAdvisor is the advisor checking for schema backward compatibility.
type CompatibilityAdvisor struct {
}

// Check checks schema backward compatibility.
func (*CompatibilityAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewCompatibilityRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// CompatibilityRule checks for schema backward compatibility.
type CompatibilityRule struct {
	BaseRule
	text            string
	lastCreateTable string
	code            code.Code
}

// NewCompatibilityRule creates a new CompatibilityRule.
func NewCompatibilityRule(level storepb.Advice_Status, title string) *CompatibilityRule {
	return &CompatibilityRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*CompatibilityRule) Name() string {
	return "CompatibilityRule"
}

// OnEnter is called when entering a parse tree node.
func (r *CompatibilityRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
		r.code = code.Ok
	case NodeTypeCreateTable:
		r.checkCreateTable(ctx.(*mysql.CreateTableContext))
	case NodeTypeDropDatabase:
		r.checkDropDatabase(ctx.(*mysql.DropDatabaseContext))
	case NodeTypeRenameTableStatement:
		r.checkRenameTableStatement(ctx.(*mysql.RenameTableStatementContext))
	case NodeTypeDropTable:
		r.checkDropTable(ctx.(*mysql.DropTableContext))
	case NodeTypeAlterTable:
		r.checkAlterTable(ctx.(*mysql.AlterTableContext))
	case NodeTypeCreateIndex:
		r.checkCreateIndex(ctx.(*mysql.CreateIndexContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *CompatibilityRule) OnExit(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeQuery && r.code != code.Ok {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          r.code.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("\"%s\" may cause incompatibility with the existing data and code", r.text),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
	return nil
}

func (r *CompatibilityRule) checkCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableName() == nil {
		return
	}
	if ctx.TableElementList() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	r.lastCreateTable = tableName
}

func (r *CompatibilityRule) checkDropDatabase(ctx *mysql.DropDatabaseContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	r.code = code.CompatibilityDropDatabase
}

func (r *CompatibilityRule) checkRenameTableStatement(ctx *mysql.RenameTableStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	r.code = code.CompatibilityRenameTable
}

func (r *CompatibilityRule) checkDropTable(ctx *mysql.DropTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	r.code = code.CompatibilityDropTable
}

func (r *CompatibilityRule) checkAlterTable(ctx *mysql.AlterTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.AlterTableActions() == nil || ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}

	if ctx.TableRef() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	if tableName == r.lastCreateTable {
		return
	}

	// alter table add column, change column, modify column.
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil {
			continue
		}

		if item.RENAME_SYMBOL() != nil && item.COLUMN_SYMBOL() != nil {
			r.code = code.CompatibilityRenameColumn
			return
		}

		if item.DROP_SYMBOL() != nil && item.ColumnInternalRef() != nil {
			r.code = code.CompatibilityDropColumn
			return
		}
		if item.DROP_SYMBOL() != nil && item.TableName() != nil {
			r.code = code.CompatibilityRenameTable
			return
		}

		if item.ADD_SYMBOL() != nil {
			if item.TableConstraintDef() != nil {
				if item.TableConstraintDef().GetType_() == nil {
					continue
				}
				switch item.TableConstraintDef().GetType_().GetTokenType() {
				// add primary key.
				case mysql.MySQLParserPRIMARY_SYMBOL:
					r.code = code.CompatibilityAddPrimaryKey
					return
				// add unique key.
				case mysql.MySQLParserUNIQUE_SYMBOL:
					r.code = code.CompatibilityAddUniqueKey
					return
				// add foreign key.
				case mysql.MySQLParserFOREIGN_SYMBOL:
					r.code = code.CompatibilityAddForeignKey
					return
				default:
				}
			}

			// add check enforced.
			// Check is only supported after 8.0.16 https://dev.mysql.com/doc/refman/8.0/en/create-table-check-constraints.html
			if item.TableConstraintDef() != nil && item.TableConstraintDef().CheckConstraint() != nil && item.TableConstraintDef().ConstraintEnforcement() != nil {
				r.code = code.CompatibilityAddCheck
				return
			}
		}

		// add check enforced.
		// Check is only supported after 8.0.16 https://dev.mysql.com/doc/refman/8.0/en/create-table-check-constraints.html
		if item.ALTER_SYMBOL() != nil && item.CHECK_SYMBOL() != nil {
			if item.ConstraintEnforcement() != nil {
				r.code = code.CompatibilityAlterCheck
				return
			}
		}

		// MODIFY COLUMN / CHANGE COLUMN
		// Due to the limitation that we don't know the current data type of the column before the change,
		// so we treat all as incompatible. This generates false positive when:
		// 1. Change to a compatible data type such as INT to BIGINT
		// 2. Change properties such as comment, change it to NULL
		// modify column
		if item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil {
			r.code = code.CompatibilityAlterColumn
			return
		}
		// change column
		if item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil {
			r.code = code.CompatibilityAlterColumn
			return
		}
	}
}

func (r *CompatibilityRule) checkCreateIndex(ctx *mysql.CreateIndexContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.GetType_() == nil {
		return
	}
	if ctx.GetType_().GetTokenType() != mysql.MySQLParserUNIQUE_SYMBOL {
		return
	}
	if ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().TableRef() == nil {
		return
	}

	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.CreateIndexTarget().TableRef())
	if r.lastCreateTable != tableName {
		r.code = code.CompatibilityAddUniqueKey
	}
}
