package mysql

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*StatementJoinStrictColumnAttrsAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLStatementJoinStrictColumnAttrs, &StatementJoinStrictColumnAttrsAdvisor{})
}

type StatementJoinStrictColumnAttrsAdvisor struct {
}

func (*StatementJoinStrictColumnAttrsAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &statementJoinStrictColumnAttrsChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}
	if ctx.DBSchema != nil {
		dbSchema := model.NewDBSchema(ctx.DBSchema, nil, nil)
		checker.dbSchema = dbSchema
	}

	for _, stmt := range stmtList {
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

type statementJoinStrictColumnAttrsChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine       int
	adviceList     []advisor.Advice
	level          advisor.Status
	title          string
	text           string
	dbSchema       *model.DBSchema
	isSelect       bool
	isInFromClause bool
}

type SourceTable struct {
	Name  string
	Alias string
}

type ColumnAttr struct {
	Table     string
	Column    string
	Type      string
	Charset   string
	Collation string
}

func (checker *statementJoinStrictColumnAttrsChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (checker *statementJoinStrictColumnAttrsChecker) EnterSelectStatement(ctx *mysql.SelectStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	checker.isSelect = true
}

func (checker *statementJoinStrictColumnAttrsChecker) ExitSelectStatement(ctx *mysql.SelectStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	checker.isSelect = false
}

func (checker *statementJoinStrictColumnAttrsChecker) EnterFromClause(ctx *mysql.FromClauseContext) {
	if !checker.isSelect {
		return
	}
	if ctx.TableReferenceList() == nil {
		return
	}

	checker.isInFromClause = true
	sourceTables := []SourceTable{}
	tableRefs := ctx.TableReferenceList().AllTableReference()
	for _, tableRef := range tableRefs {
		if tableRef.TableFactor() == nil {
			continue
		}

		tableFactor := tableRef.TableFactor()
		if tableFactor.SingleTable() != nil && tableFactor.SingleTable().TableRef() != nil {
			sourceTable := SourceTable{
				Name: tableFactor.SingleTable().TableRef().GetText(),
			}
			if tableFactor.SingleTable().TableAlias() != nil {
				sourceTable.Alias = tableFactor.SingleTable().TableAlias().Identifier().GetText()
			}
			sourceTables = append(sourceTables, sourceTable)
		}

		if tableRef.AllJoinedTable() == nil {
			continue
		}
		for _, joinedTable := range tableRef.AllJoinedTable() {
			if joinedTable.TableReference() == nil {
				continue
			}

			referencedTable := joinedTable.TableReference().GetText()
			if joinedTable.USING_SYMBOL() != nil {
				if joinedTable.IdentifierListWithParentheses() == nil || joinedTable.IdentifierListWithParentheses().IdentifierList() == nil {
					continue
				}
				identifierList := joinedTable.IdentifierListWithParentheses().IdentifierList().AllIdentifier()
				for _, identifier := range identifierList {
					column := identifier.GetText()
					rightColumn := &ColumnAttr{
						Table:  referencedTable,
						Column: column,
					}
					for _, sourceTable := range sourceTables {
						checker.checkColumnAttrs(&ColumnAttr{
							Table:  sourceTable.Name,
							Column: column,
						}, rightColumn)
					}
				}
			}
		}
	}
}

func (checker *statementJoinStrictColumnAttrsChecker) ExitFromClause(_ *mysql.FromClauseContext) {
	checker.isInFromClause = false
}

func (checker *statementJoinStrictColumnAttrsChecker) EnterPrimaryExprCompare(ctx *mysql.PrimaryExprCompareContext) {
	if !checker.isSelect || !checker.isInFromClause {
		return
	}

	compOp := ctx.CompOp()
	// We only check for equal for now.
	if compOp == nil || compOp.EQUAL_OPERATOR() == nil {
		return
	}
	if ctx.BoolPri() == nil || ctx.Predicate() == nil {
		return
	}
	leftColumnAttr := extractJoinInfoFromText(ctx.BoolPri().GetText())
	rightColumnAttr := extractJoinInfoFromText(ctx.Predicate().GetText())
	checker.checkColumnAttrs(leftColumnAttr, rightColumnAttr)
}

func (checker *statementJoinStrictColumnAttrsChecker) checkColumnAttrs(leftColumnAttr *ColumnAttr, rightColumnAttr *ColumnAttr) {
	if !checker.isSelect || !checker.isInFromClause {
		return
	}
	if leftColumnAttr == nil || rightColumnAttr == nil {
		return
	}

	// TODO(steven): dynamic match table alias later.
	leftTable := checker.findTable(leftColumnAttr.Table)
	rightTable := checker.findTable(rightColumnAttr.Table)
	if leftTable == nil || rightTable == nil {
		return
	}
	leftColumn := leftTable.GetColumn(leftColumnAttr.Column)
	rightColumn := rightTable.GetColumn(rightColumnAttr.Column)
	if leftColumn == nil || rightColumn == nil {
		return
	}
	if leftColumn.Type != rightColumn.Type || leftColumn.CharacterSet != rightColumn.CharacterSet || leftColumn.Collation != rightColumn.Collation {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    advisor.StatementJoinColumnAttrsNotMatch,
			Title:   checker.title,
			Content: fmt.Sprintf("%s.%s and %s.%s column fields do not match", leftColumnAttr.Table, leftColumnAttr.Column, rightColumnAttr.Table, rightColumnAttr.Column),
			Line:    checker.baseLine,
		})
	}
}

func (checker *statementJoinStrictColumnAttrsChecker) findTable(tableName string) *model.TableMetadata {
	if checker.dbSchema == nil {
		return nil
	}
	return checker.dbSchema.GetDatabaseMetadata().GetSchema("").GetTable(tableName)
}

func extractJoinInfoFromText(text string) *ColumnAttr {
	elements := strings.Split(text, ".")
	if len(elements) != 2 {
		return nil
	}
	return &ColumnAttr{
		Table:  elements[0],
		Column: elements[1],
	}
}
