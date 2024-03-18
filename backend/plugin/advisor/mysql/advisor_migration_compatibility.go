package mysql

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLMigrationCompatibility, &CompatibilityAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLMigrationCompatibility, &CompatibilityAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLMigrationCompatibility, &CompatibilityAdvisor{})
}

// CompatibilityAdvisor is the advisor checking for schema backward compatibility.
type CompatibilityAdvisor struct {
}

// Check checks schema backward compatibility.
func (*CompatibilityAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &compatibilityChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	for _, stmt := range root {
		checker.baseLine = stmt.BaseLine
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type compatibilityChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine        int
	adviceList      []advisor.Advice
	level           advisor.Status
	title           string
	lastCreateTable string
	code            advisor.Code
}

// EnterQuery is called when production query is entered.
func (checker *compatibilityChecker) EnterQuery(_ *mysql.QueryContext) {
	checker.code = advisor.Ok
}

// ExitQuery is called when production query is exited.
func (checker *compatibilityChecker) ExitQuery(ctx *mysql.QueryContext) {
	if checker.code != advisor.Ok {
		text := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    checker.code,
			Title:   checker.title,
			Content: fmt.Sprintf("\"%s\" may cause incompatibility with the existing data and code", text),
			Line:    checker.baseLine + ctx.GetStart().GetLine(),
		})
	}
}

// EnterCreateTable is called when production createTable is entered.
func (checker *compatibilityChecker) EnterCreateTable(ctx *mysql.CreateTableContext) {
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
	checker.lastCreateTable = tableName
}

// EnterDropDatabase is called when production dropDatabase is entered.
func (checker *compatibilityChecker) EnterDropDatabase(ctx *mysql.DropDatabaseContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	checker.code = advisor.CompatibilityDropDatabase
}

// EnterRenameTableStatement is called when production renameTableStatement is entered.
func (checker *compatibilityChecker) EnterRenameTableStatement(ctx *mysql.RenameTableStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	checker.code = advisor.CompatibilityRenameTable
}

// EnterDropTable is called when production dropTable is entered.
func (checker *compatibilityChecker) EnterDropTable(ctx *mysql.DropTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	checker.code = advisor.CompatibilityDropTable
}

func (checker *compatibilityChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
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
	if tableName == checker.lastCreateTable {
		return
	}

	// alter table add column, change column, modify column.
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil {
			continue
		}

		if item.RENAME_SYMBOL() != nil && item.COLUMN_SYMBOL() != nil {
			checker.code = advisor.CompatibilityRenameColumn
			return
		}

		if item.DROP_SYMBOL() != nil && item.ColumnInternalRef() != nil {
			checker.code = advisor.CompatibilityDropColumn
			return
		}
		if item.DROP_SYMBOL() != nil && item.TableName() != nil {
			checker.code = advisor.CompatibilityRenameTable
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
					checker.code = advisor.CompatibilityAddPrimaryKey
					return
				// add unique key.
				case mysql.MySQLParserUNIQUE_SYMBOL:
					checker.code = advisor.CompatibilityAddUniqueKey
					return
				// add foreign key.
				case mysql.MySQLParserFOREIGN_SYMBOL:
					checker.code = advisor.CompatibilityAddForeignKey
					return
				}
			}

			// add check enforced.
			// Check is only supported after 8.0.16 https://dev.mysql.com/doc/refman/8.0/en/create-table-check-constraints.html
			if item.TableConstraintDef() != nil && item.TableConstraintDef().CheckConstraint() != nil && item.TableConstraintDef().ConstraintEnforcement() != nil {
				checker.code = advisor.CompatibilityAddCheck
				return
			}
		}

		// add check enforced.
		// Check is only supported after 8.0.16 https://dev.mysql.com/doc/refman/8.0/en/create-table-check-constraints.html
		if item.ALTER_SYMBOL() != nil && item.CHECK_SYMBOL() != nil {
			if item.ConstraintEnforcement() != nil {
				checker.code = advisor.CompatibilityAlterCheck
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
			checker.code = advisor.CompatibilityAlterColumn
			return
		}
		// change column
		if item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil {
			checker.code = advisor.CompatibilityAlterColumn
			return
		}
	}
}

// EnterCreateIndex is called when production createIndex is entered.
func (checker *compatibilityChecker) EnterCreateIndex(ctx *mysql.CreateIndexContext) {
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
	if checker.lastCreateTable != tableName {
		checker.code = advisor.CompatibilityAddUniqueKey
	}
}
