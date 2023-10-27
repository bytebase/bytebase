// Package oracle is the advisor for oracle database.
package oracle

import (
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
func (*TableNoForeignKeyAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &tableNoForeignKeyListener{
		level:         level,
		title:         string(ctx.Rule.Type),
		currentSchema: ctx.CurrentSchema,
		tableWithFK:   make(map[string]bool),
		tableLine:     make(map[string]int),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// tableNoForeignKeyListener is the listener for table disallow foreign key.
type tableNoForeignKeyListener struct {
	*parser.BasePlSqlParserListener

	level         advisor.Status
	title         string
	currentSchema string
	tableName     string
	tableWithFK   map[string]bool
	tableLine     map[string]int
}

func (l *tableNoForeignKeyListener) generateAdvice() ([]advisor.Advice, error) {
	advice := []advisor.Advice{}
	for tableName, hasFK := range l.tableWithFK {
		if hasFK {
			advice = append(advice, advisor.Advice{
				Status:  l.level,
				Code:    advisor.TableHasFK,
				Title:   l.title,
				Content: fmt.Sprintf("Foreign key is not allowed in the table %s.", normalizeTableName(tableName)),
				Line:    l.tableLine[tableName],
			})
		}
	}

	if len(advice) == 0 {
		advice = append(advice, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return advice, nil
}

// EnterCreate_table is called when production create_table is entered.
func (l *tableNoForeignKeyListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	schemaName := l.currentSchema
	if ctx.Schema_name() != nil {
		schemaName = normalizeIdentifier(ctx.Schema_name(), l.currentSchema)
	}

	l.tableName = fmt.Sprintf("%s.%s", schemaName, normalizeIdentifier(ctx.Table_name(), l.currentSchema))
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
	l.tableName = normalizeIdentifier(ctx.Tableview_name(), l.currentSchema)
}

// ExitAlter_table is called when production alter_table is exited.
func (l *tableNoForeignKeyListener) ExitAlter_table(_ *parser.Alter_tableContext) {
	l.tableName = ""
}
