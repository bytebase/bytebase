package mysql

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLTableLimitSize, &MaximumTableSizeAdvisor{})
}

type MaximumTableSizeAdvisor struct {
}

type MaximumTableSizeChecker struct {
	*mysql.BaseMySQLParserListener
	affectedTabNames []string
	baseLine         int
}

var (
	_ advisor.Advisor = &MaximumTableSizeAdvisor{}
)

// If table size > xx bytes, then warning/error.
func (*MaximumTableSizeAdvisor) Check(ctx advisor.Context) ([]*storepb.Advice, error) {
	var adviceList []*storepb.Advice

	payload, err := advisor.UnmarshalNumberTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	statParsedResults, ok := ctx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	// User defined rule level.
	status, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	for _, parsedResult := range statParsedResults {
		statTypeChecker := &mysqlparser.StatementTypeChecker{}
		antlr.ParseTreeWalkerDefault.Walk(statTypeChecker, parsedResult.Tree)

		tableSizeChecker := &MaximumTableSizeChecker{}
		statementBaseLine := parsedResult.BaseLine
		if statTypeChecker.IsDDL {
			// Get table name.
			antlr.ParseTreeWalkerDefault.Walk(tableSizeChecker, parsedResult.Tree)
			if ctx.DBSchema != nil && len(ctx.DBSchema.Schemas) != 0 {
				// Check all table size.
				for _, tabName := range tableSizeChecker.affectedTabNames {
					tableRows := getTabRowsByName(tabName, ctx.DBSchema.Schemas[0].Tables)
					if tableRows >= int64(payload.Number) {
						adviceList = append(adviceList, &storepb.Advice{
							Status:  status,
							Code:    advisor.TableExceedLimitSize.Int32(),
							Title:   ctx.Rule.Type,
							Content: fmt.Sprintf("Apply DDL on large table '%s' ( %d rows ) will lock table for a long time", tabName, tableRows),
							StartPosition: &storepb.Position{
								Line: int32(statementBaseLine + tableSizeChecker.baseLine),
							},
						})
					}
				}
			}
		}
	}

	return adviceList, nil
}

func (checker *MaximumTableSizeChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	checker.baseLine = ctx.GetStart().GetLine()
	checker.affectedTabNames = append(checker.affectedTabNames, tableName)
}

func (checker *MaximumTableSizeChecker) EnterTruncateTableStatement(ctx *mysql.TruncateTableStatementContext) {
	_, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	checker.baseLine = ctx.GetStart().GetLine()
	checker.affectedTabNames = append(checker.affectedTabNames, tableName)
}

func (checker *MaximumTableSizeChecker) EnterDropTable(ctx *mysql.DropTableContext) {
	checker.baseLine = ctx.GetStart().GetLine()
	for _, tabRef := range ctx.TableRefList().AllTableRef() {
		_, tableName := mysqlparser.NormalizeMySQLTableRef(tabRef)
		checker.affectedTabNames = append(checker.affectedTabNames, tableName)
	}
}

func getTabRowsByName(targetTabName string, tables []*storepb.TableMetadata) int64 {
	for _, table := range tables {
		if table.Name == targetTabName {
			return table.RowCount
		}
	}
	return 0
}
