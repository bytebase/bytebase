// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"slices"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*ColumnRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleRequireColumnDefault, &ColumnRequireDefaultAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleRequireColumnDefault, &ColumnRequireDefaultAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleRequireColumnDefault, &ColumnRequireDefaultAdvisor{})
}

// ColumnRequireDefaultAdvisor is the advisor checking for column default requirement.
type ColumnRequireDefaultAdvisor struct {
}

// Check checks for column default requirement.
func (*ColumnRequireDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &columnRequireDefaultListener{
		level:            level,
		title:            string(checkCtx.Rule.Type),
		currentDatabase:  checkCtx.CurrentDatabase,
		noDefaultColumns: make(columnMap),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// columnRequireDefaultListener is the listener for column default requirement.
type columnRequireDefaultListener struct {
	*parser.BasePlSqlParserListener

	level            storepb.Advice_Status
	title            string
	currentDatabase  string
	noDefaultColumns columnMap
	tableName        string
}

func (l *columnRequireDefaultListener) generateAdvice() ([]*storepb.Advice, error) {
	advice := []*storepb.Advice{}

	var columnIDs []string
	for columnID := range l.noDefaultColumns {
		columnIDs = append(columnIDs, columnID)
	}
	slices.Sort(columnIDs)
	for _, columnID := range columnIDs {
		line := l.noDefaultColumns[columnID]
		advice = append(advice, &storepb.Advice{
			Status:        l.level,
			Code:          advisor.NoDefault.Int32(),
			Title:         l.title,
			Content:       fmt.Sprintf("Column %q doesn't have default value", lastIdentifier(columnID)),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
	}

	return advice, nil
}

// EnterCreate_table is called when production create_table is entered.
func (l *columnRequireDefaultListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	schemaName := l.currentDatabase
	if ctx.Schema_name() != nil {
		schemaName = normalizeIdentifier(ctx.Schema_name(), l.currentDatabase)
	}
	l.tableName = fmt.Sprintf("%s.%s", schemaName, normalizeIdentifier(ctx.Table_name(), schemaName))
}

// ExitCreate_table is called when production create_table is exited.
func (l *columnRequireDefaultListener) ExitCreate_table(_ *parser.Create_tableContext) {
	l.tableName = ""
}

// EnterColumn_definition is called when production column_definition is entered.
func (l *columnRequireDefaultListener) EnterColumn_definition(ctx *parser.Column_definitionContext) {
	if l.tableName == "" {
		return
	}
	columnName := normalizeIdentifier(ctx.Column_name(), l.currentDatabase)
	columnID := fmt.Sprintf(`%s.%s`, l.tableName, columnName)
	if ctx.DEFAULT() == nil {
		l.noDefaultColumns[columnID] = ctx.GetStart().GetLine()
	} else {
		delete(l.noDefaultColumns, columnID)
	}
}

// EnterAlter_table is called when production alter_table is entered.
func (l *columnRequireDefaultListener) EnterAlter_table(ctx *parser.Alter_tableContext) {
	l.tableName = normalizeIdentifier(ctx.Tableview_name(), l.currentDatabase)
}

// ExitAlter_table is called when production alter_table is exited.
func (l *columnRequireDefaultListener) ExitAlter_table(_ *parser.Alter_tableContext) {
	l.tableName = ""
}

// EnterModify_col_properties is called when production modify_col_properties is entered.
func (l *columnRequireDefaultListener) EnterModify_col_properties(ctx *parser.Modify_col_propertiesContext) {
	if l.tableName == "" {
		return
	}
	columnName := normalizeIdentifier(ctx.Column_name(), l.currentDatabase)
	if ctx.DEFAULT() != nil {
		columnID := fmt.Sprintf(`%s.%s`, l.tableName, columnName)
		delete(l.noDefaultColumns, columnID)
	}
}
