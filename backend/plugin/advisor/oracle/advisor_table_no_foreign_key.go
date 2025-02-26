// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*TableNoForeignKeyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleTableNoFK, &TableNoForeignKeyAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleTableNoFK, &TableNoForeignKeyAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleTableNoFK, &TableNoForeignKeyAdvisor{})
}

// TableNoForeignKeyAdvisor is the advisor checking for table disallow foreign key.
type TableNoForeignKeyAdvisor struct {
}

// Check checks for table disallow foreign key.
func (*TableNoForeignKeyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &tableNoForeignKeyListener{
		level:           level,
		title:           string(checkCtx.Rule.Type),
		currentDatabase: checkCtx.CurrentDatabase,
		tableWithFK:     make(map[string]bool),
		tableLine:       make(map[string]int),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// tableNoForeignKeyListener is the listener for table disallow foreign key.
type tableNoForeignKeyListener struct {
	*parser.BasePlSqlParserListener

	level           storepb.Advice_Status
	title           string
	currentDatabase string
	tableName       string
	tableWithFK     map[string]bool
	tableLine       map[string]int
}

func (l *tableNoForeignKeyListener) generateAdvice() ([]*storepb.Advice, error) {
	advice := []*storepb.Advice{}
	for tableName, hasFK := range l.tableWithFK {
		if hasFK {
			advice = append(advice, &storepb.Advice{
				Status:  l.level,
				Code:    advisor.TableHasFK.Int32(),
				Title:   l.title,
				Content: fmt.Sprintf("Foreign key is not allowed in the table %s.", normalizeIdentifierName(tableName)),
				StartPosition: &storepb.Position{
					Line: int32(l.tableLine[tableName]),
				},
			})
		}
	}

	return advice, nil
}

// EnterCreate_table is called when production create_table is entered.
func (l *tableNoForeignKeyListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	schemaName := l.currentDatabase
	if ctx.Schema_name() != nil {
		schemaName = normalizeIdentifier(ctx.Schema_name(), l.currentDatabase)
	}

	l.tableName = fmt.Sprintf("%s.%s", schemaName, normalizeIdentifier(ctx.Table_name(), l.currentDatabase))
}

// ExitCreate_table is called when production create_table is exited.
func (l *tableNoForeignKeyListener) ExitCreate_table(_ *parser.Create_tableContext) {
	l.tableName = ""
}

// EnterReferences_clause is called when production references_clause is entered.
func (l *tableNoForeignKeyListener) EnterReferences_clause(ctx *parser.References_clauseContext) {
	l.tableWithFK[l.tableName] = true
	l.tableLine[l.tableName] = ctx.GetStop().GetLine()
}

// EnterAlter_table is called when production alter_table is entered.
func (l *tableNoForeignKeyListener) EnterAlter_table(ctx *parser.Alter_tableContext) {
	l.tableName = normalizeIdentifier(ctx.Tableview_name(), l.currentDatabase)
}

// ExitAlter_table is called when production alter_table is exited.
func (l *tableNoForeignKeyListener) ExitAlter_table(_ *parser.Alter_tableContext) {
	l.tableName = ""
}
