package oracle

import (
	"fmt"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
)

var (
	_ advisor.Advisor = (*TableRequirePKAdvisor)(nil)
)

func init() {
	advisor.Register(db.Oracle, advisor.OracleTableRequirePK, &TableRequirePKAdvisor{})
}

// TableRequirePKAdvisor is the advisor checking table requires PK.
type TableRequirePKAdvisor struct {
}

// Check checks table requires PK.
func (*TableRequirePKAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	tree, errAdvice := parseStatement(statement)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &TableRequirePKListener{
		level:         level,
		currentSchema: "SYS",
		tableWitPK:    make(map[string]bool),
		tableLine:     make(map[string]int),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

type TableRequirePKListener struct {
	*parser.BasePlSqlParserListener

	level         advisor.Status
	currentSchema string
	tableName     string
	tableWitPK    map[string]bool
	tableLine     map[string]int
}

func (l *TableRequirePKListener) generateAdvice() ([]advisor.Advice, error) {
	advice := []advisor.Advice{}
	for tableName, hasPK := range l.tableWitPK {
		if !hasPK {
			advice = append(advice, advisor.Advice{
				Status:  l.level,
				Code:    advisor.TableNoPK,
				Title:   "Table requires PK",
				Content: fmt.Sprintf("Table %q requires PRIMARY KEY.", tableName),
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

func (l *TableRequirePKListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	schemaName := l.currentSchema
	if ctx.Schema_name() != nil {
		schemaName = normalizeIdentifier(ctx.Schema_name())
	}

	l.tableName = fmt.Sprintf("%s.%s", schemaName, normalizeIdentifier(ctx.Table_name()))
	l.tableWitPK[l.tableName] = false
	l.tableLine[l.tableName] = ctx.GetStop().GetLine()
}

func (l *TableRequirePKListener) ExitCreate_table(ctx *parser.Create_tableContext) {
	l.tableName = ""
}

func (l *TableRequirePKListener) EnterInline_constraint(ctx *parser.Inline_constraintContext) {
	if ctx.PRIMARY() != nil {
		if _, exists := l.tableWitPK[l.tableName]; exists {
			l.tableWitPK[l.tableName] = true
		}
	}
}

func (l *TableRequirePKListener) EnterConstraint_clauses(ctx *parser.Constraint_clausesContext) {
	if ctx.PRIMARY() != nil {
		if _, exists := l.tableWitPK[l.tableName]; exists {
			l.tableWitPK[l.tableName] = true
		}
	}
}

func (l *TableRequirePKListener) EnterAlter_table(ctx *parser.Alter_tableContext) {
	l.tableName = ctx.Tableview_name().GetText()
}

func (l *TableRequirePKListener) ExitAlter_table(ctx *parser.Alter_tableContext) {
	l.tableName = ""
}
