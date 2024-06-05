// Package oracle is the advisor for oracle database.
package oracle

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NamingTableNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleTableNamingNoKeyword, &NamingTableNoKeywordAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleTableNamingNoKeyword, &NamingTableNoKeywordAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleTableNamingNoKeyword, &NamingTableNoKeywordAdvisor{})
}

// NamingTableNoKeywordAdvisor is the advisor checking for table naming convention without keyword.
type NamingTableNoKeywordAdvisor struct {
}

// Check checks for table naming convention without keyword.
func (*NamingTableNoKeywordAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &namingTableNoKeywordListener{
		level:           level,
		title:           string(ctx.Rule.Type),
		currentDatabase: ctx.CurrentDatabase,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// namingTableNoKeywordListener is the listener for table naming convention without keyword.
type namingTableNoKeywordListener struct {
	*parser.BasePlSqlParserListener

	level           storepb.Advice_Status
	title           string
	currentDatabase string
	adviceList      []*storepb.Advice
}

func (l *namingTableNoKeywordListener) generateAdvice() ([]*storepb.Advice, error) {
	if len(l.adviceList) == 0 {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:  storepb.Advice_SUCCESS,
			Code:    advisor.Ok.Int32(),
			Title:   "OK",
			Content: "",
		})
	}
	return l.adviceList, nil
}

// EnterCreate_table is called when production create_table is entered.
func (l *namingTableNoKeywordListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	tableName := normalizeIdentifier(ctx.Table_name(), l.currentDatabase)
	if plsqlparser.IsOracleKeyword(tableName) {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:  l.level,
			Code:    advisor.NameIsKeywordIdentifier.Int32(),
			Title:   l.title,
			Content: fmt.Sprintf("Table name %q is a keyword identifier and should be avoided.", tableName),
			StartPosition: &storepb.Position{
				Line: int32(ctx.GetStart().GetLine()),
			},
		})
	}
}

// EnterAlter_table_properties is called when production alter_table_properties is entered.
func (l *namingTableNoKeywordListener) EnterAlter_table_properties(ctx *parser.Alter_table_propertiesContext) {
	if ctx.Tableview_name() == nil {
		return
	}
	tableName := lastIdentifier(normalizeIdentifier(ctx.Tableview_name(), l.currentDatabase))
	if plsqlparser.IsOracleKeyword(tableName) {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:  l.level,
			Code:    advisor.NameIsKeywordIdentifier.Int32(),
			Title:   l.title,
			Content: fmt.Sprintf("Table name %q is a keyword identifier and should be avoided.", tableName),
			StartPosition: &storepb.Position{
				Line: int32(ctx.GetStart().GetLine()),
			},
		})
	}
}
