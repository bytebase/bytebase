package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*StatementJoinStrictColumnAttrsAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementJoinStrictColumnAttrs, &StatementJoinStrictColumnAttrsAdvisor{})
}

type StatementJoinStrictColumnAttrsAdvisor struct {
}

func (*StatementJoinStrictColumnAttrsAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewStatementJoinStrictColumnAttrsRule(level, string(checkCtx.Rule.Type))
	if checkCtx.DBSchema != nil {
		dbSchema := model.NewDatabaseSchema(checkCtx.DBSchema, nil, nil, storepb.Engine_MYSQL, checkCtx.IsObjectCaseSensitive)
		rule.dbSchema = dbSchema
	}

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// SourceTable represents a table in the FROM clause.
type SourceTable struct {
	Name  string
	Alias string
}

// ColumnAttr represents column attributes for join checking.
type ColumnAttr struct {
	Table     string
	Column    string
	Type      string
	Charset   string
	Collation string
}

// StatementJoinStrictColumnAttrsRule checks that joined columns have matching attributes.
type StatementJoinStrictColumnAttrsRule struct {
	BaseRule
	text           string
	dbSchema       *model.DatabaseSchema
	isSelect       bool
	isInFromClause bool
	sourceTables   []SourceTable
}

// NewStatementJoinStrictColumnAttrsRule creates a new StatementJoinStrictColumnAttrsRule.
func NewStatementJoinStrictColumnAttrsRule(level storepb.Advice_Status, title string) *StatementJoinStrictColumnAttrsRule {
	return &StatementJoinStrictColumnAttrsRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*StatementJoinStrictColumnAttrsRule) Name() string {
	return "StatementJoinStrictColumnAttrsRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementJoinStrictColumnAttrsRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeSelectStatement:
		if mysqlparser.IsTopMySQLRule(&ctx.(*mysql.SelectStatementContext).BaseParserRuleContext) {
			r.isSelect = true
		}
	case NodeTypeFromClause:
		r.handleFromClause(ctx.(*mysql.FromClauseContext))
	case NodeTypePrimaryExprCompare:
		r.handlePrimaryExprCompare(ctx.(*mysql.PrimaryExprCompareContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *StatementJoinStrictColumnAttrsRule) OnExit(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeSelectStatement:
		if mysqlparser.IsTopMySQLRule(&ctx.(*mysql.SelectStatementContext).BaseParserRuleContext) {
			r.isSelect = false
		}
	case NodeTypeFromClause:
		r.isInFromClause = false
	default:
	}
	return nil
}

func (r *StatementJoinStrictColumnAttrsRule) handleFromClause(ctx *mysql.FromClauseContext) {
	if !r.isSelect {
		return
	}
	if ctx.TableReferenceList() == nil {
		return
	}

	r.isInFromClause = true
	r.sourceTables = []SourceTable{}
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
			r.sourceTables = append(r.sourceTables, sourceTable)
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
					for _, sourceTable := range r.sourceTables {
						r.checkColumnAttrs(&ColumnAttr{
							Table:  sourceTable.Name,
							Column: column,
						}, rightColumn)
					}
				}
			}
		}
	}
}

func (r *StatementJoinStrictColumnAttrsRule) handlePrimaryExprCompare(ctx *mysql.PrimaryExprCompareContext) {
	if !r.isSelect || !r.isInFromClause {
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
	r.checkColumnAttrs(leftColumnAttr, rightColumnAttr)
}

func (r *StatementJoinStrictColumnAttrsRule) checkColumnAttrs(leftColumnAttr *ColumnAttr, rightColumnAttr *ColumnAttr) {
	if !r.isSelect || !r.isInFromClause {
		return
	}
	if leftColumnAttr == nil || rightColumnAttr == nil {
		return
	}

	// TODO(steven): dynamic match table alias later.
	leftTable := r.findTable(leftColumnAttr.Table)
	rightTable := r.findTable(rightColumnAttr.Table)
	if leftTable == nil || rightTable == nil {
		return
	}
	leftColumn := leftTable.GetColumn(leftColumnAttr.Column)
	rightColumn := rightTable.GetColumn(rightColumnAttr.Column)
	if leftColumn == nil || rightColumn == nil {
		return
	}
	if leftColumn.Type != rightColumn.Type || leftColumn.CharacterSet != rightColumn.CharacterSet || leftColumn.Collation != rightColumn.Collation {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.StatementJoinColumnAttrsNotMatch.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("%s.%s and %s.%s column fields do not match", leftColumnAttr.Table, leftColumnAttr.Column, rightColumnAttr.Table, rightColumnAttr.Column),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine),
		})
	}
}

func (r *StatementJoinStrictColumnAttrsRule) findTable(tableName string) *model.TableMetadata {
	if r.dbSchema == nil {
		return nil
	}
	return r.dbSchema.GetDatabaseMetadata().GetSchema("").GetTable(tableName)
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
