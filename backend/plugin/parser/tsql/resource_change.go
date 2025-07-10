package tsql

import (
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_MSSQL, extractChangedResources)
}

func extractChangedResources(currentDatabase string, currentSchema string, dbSchema *model.DatabaseSchema, asts any, statement string) (*base.ChangeSummary, error) {
	tree, ok := asts.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert ast to antlr.Tree")
	}

	changedResources := model.NewChangedResources(dbSchema)
	l := &tsqlChangedResourceExtractListener{
		currentDatabase:  currentDatabase,
		currentSchema:    currentSchema,
		dbSchema:         dbSchema,
		changedResources: changedResources,
		statement:        statement,
	}

	antlr.ParseTreeWalkerDefault.Walk(l, tree)

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS:       l.sampleDMLs,
		DMLCount:         l.dmlCount,
		InsertCount:      l.insertCount,
	}, nil
}

type tsqlChangedResourceExtractListener struct {
	*parser.BaseTSqlParserListener

	currentDatabase  string
	currentSchema    string
	dbSchema         *model.DatabaseSchema
	changedResources *model.ChangedResources
	statement        string
	sampleDMLs       []string
	dmlCount         int
	insertCount      int

	// Internal data structure used temporarily.
	text string
}

func (l *tsqlChangedResourceExtractListener) EnterSql_clauses(ctx *parser.Sql_clausesContext) {
	l.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

func (l *tsqlChangedResourceExtractListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	tableName := ctx.Table_name()
	if tableName == nil {
		return
	}
	d, s, t := normalizeTableNameSeparated(tableName, l.currentDatabase, l.currentSchema, false /* caseSensitive */)

	l.changedResources.AddTable(
		d,
		s,
		&storepb.ChangedResourceTable{
			Name:   t,
			Ranges: []*storepb.Range{base.NewRange(l.statement, trimStatement(l.text))},
		},
		false)
}

func (l *tsqlChangedResourceExtractListener) EnterDrop_table(ctx *parser.Drop_tableContext) {
	if ctx.AllTable_name() == nil {
		return
	}

	for _, tableName := range ctx.AllTable_name() {
		d, s, t := normalizeTableNameSeparated(tableName, l.currentDatabase, l.currentSchema, false /* caseSensitive */)

		l.changedResources.AddTable(
			d,
			s,
			&storepb.ChangedResourceTable{
				Name:   t,
				Ranges: []*storepb.Range{base.NewRange(l.statement, trimStatement(l.text))},
			},
			true)
	}
}

func (l *tsqlChangedResourceExtractListener) EnterAlter_table(ctx *parser.Alter_tableContext) {
	if ctx.AllTable_name() == nil {
		return
	}

	for _, tableName := range ctx.AllTable_name() {
		d, s, t := normalizeTableNameSeparated(tableName, l.currentDatabase, l.currentSchema, false /* caseSensitive */)

		l.changedResources.AddTable(
			d,
			s,
			&storepb.ChangedResourceTable{
				Name:   t,
				Ranges: []*storepb.Range{base.NewRange(l.statement, trimStatement(l.text))},
			},
			true)
	}
}

func (l *tsqlChangedResourceExtractListener) EnterCreate_index(ctx *parser.Create_indexContext) {
	tableName := ctx.Table_name()
	if tableName == nil {
		return
	}
	d, s, t := normalizeTableNameSeparated(tableName, l.currentDatabase, l.currentSchema, false /* caseSensitive */)

	l.changedResources.AddTable(
		d,
		s,
		&storepb.ChangedResourceTable{
			Name:   t,
			Ranges: []*storepb.Range{base.NewRange(l.statement, trimStatement(l.text))},
		},
		false)
}

// EnterDrop_index is called when production drop_index is entered.
func (l *tsqlChangedResourceExtractListener) EnterDrop_index(ctx *parser.Drop_indexContext) {
	for _, index := range ctx.AllDrop_relational_or_xml_or_spatial_index() {
		fullTable, err := NormalizeFullTableName(index.Full_table_name())
		if err != nil {
			continue
		}
		d, _ := NormalizeTSQLIdentifierText(l.currentDatabase)
		s, _ := NormalizeTSQLIdentifierText(l.currentSchema)
		if fullTable.Database != "" {
			d = fullTable.Database
		}
		if fullTable.Schema != "" {
			s = fullTable.Schema
		}

		l.changedResources.AddTable(
			d,
			s,
			&storepb.ChangedResourceTable{
				Name:   fullTable.Table,
				Ranges: []*storepb.Range{base.NewRange(l.statement, trimStatement(l.text))},
			},
			false)
	}
}

// EnterCreate_view is called when production create_view is entered.
func (l *tsqlChangedResourceExtractListener) EnterCreate_view(ctx *parser.Create_viewContext) {
	d, _ := NormalizeTSQLIdentifierText(l.currentDatabase)
	schema, name := normalizeSimpleNameSeparated(ctx.Simple_name(), l.currentSchema, false)
	l.changedResources.AddView(
		d,
		schema,
		&storepb.ChangedResourceView{
			Name:   name,
			Ranges: []*storepb.Range{base.NewRange(l.statement, trimStatement(l.text))},
		},
	)
}

func (l *tsqlChangedResourceExtractListener) EnterDrop_view(ctx *parser.Drop_viewContext) {
	d, _ := NormalizeTSQLIdentifierText(l.currentDatabase)
	for _, simpleName := range ctx.AllSimple_name() {
		schema, name := normalizeSimpleNameSeparated(simpleName, l.currentSchema, false)
		l.changedResources.AddView(
			d,
			schema,
			&storepb.ChangedResourceView{
				Name:   name,
				Ranges: []*storepb.Range{base.NewRange(l.statement, trimStatement(l.text))},
			},
		)
	}
}

func (l *tsqlChangedResourceExtractListener) EnterCreate_or_alter_procedure(ctx *parser.Create_or_alter_procedureContext) {
	d, _ := NormalizeTSQLIdentifierText(l.currentDatabase)
	schema, procedure := normalizeProcedureSeparated(ctx.GetProcName(), l.currentSchema, false)

	l.changedResources.AddProcedure(
		d,
		schema,
		&storepb.ChangedResourceProcedure{
			Name:   procedure,
			Ranges: []*storepb.Range{base.NewRange(l.statement, trimStatement(l.text))},
		},
	)
}

func (l *tsqlChangedResourceExtractListener) EnterDrop_procedure(ctx *parser.Drop_procedureContext) {
	d, _ := NormalizeTSQLIdentifierText(l.currentDatabase)
	for _, functionName := range ctx.AllFunc_proc_name_schema() {
		schema, function := normalizeProcedureSeparated(functionName, l.currentSchema, false)

		l.changedResources.AddFunction(
			d,
			schema,
			&storepb.ChangedResourceFunction{
				Name:   function,
				Ranges: []*storepb.Range{base.NewRange(l.statement, trimStatement(l.text))},
			},
		)
	}
}

func (l *tsqlChangedResourceExtractListener) EnterCreate_or_alter_function(ctx *parser.Create_or_alter_functionContext) {
	d, _ := NormalizeTSQLIdentifierText(l.currentDatabase)
	schema, function := normalizeProcedureSeparated(ctx.GetFuncName(), l.currentSchema, false)

	l.changedResources.AddFunction(
		d,
		schema,
		&storepb.ChangedResourceFunction{
			Name:   function,
			Ranges: []*storepb.Range{base.NewRange(l.statement, trimStatement(l.text))},
		},
	)
}

func (l *tsqlChangedResourceExtractListener) EnterDrop_function(ctx *parser.Drop_functionContext) {
	d, _ := NormalizeTSQLIdentifierText(l.currentDatabase)
	for _, functionName := range ctx.AllFunc_proc_name_schema() {
		schema, function := normalizeProcedureSeparated(functionName, l.currentSchema, false)

		l.changedResources.AddFunction(
			d,
			schema,
			&storepb.ChangedResourceFunction{
				Name:   function,
				Ranges: []*storepb.Range{base.NewRange(l.statement, trimStatement(l.text))},
			},
		)
	}
}

func (l *tsqlChangedResourceExtractListener) EnterInsert_statement(ctx *parser.Insert_statementContext) {
	if ctx.Ddl_object() != nil && ctx.Ddl_object().Full_table_name() != nil {
		table, err := NormalizeFullTableName(ctx.Ddl_object().Full_table_name())
		if err == nil && table != nil && table.Table != "" {
			d := table.Database
			if d == "" {
				d = l.currentDatabase
			}
			s := table.Schema
			if s == "" {
				s = l.currentSchema
			}
			l.changedResources.AddTable(
				d,
				s,
				&storepb.ChangedResourceTable{
					Name:   table.Table,
					Ranges: []*storepb.Range{base.NewRange(l.statement, trimStatement(l.text))},
				},
				false,
			)
		}
	}

	if ctx.Insert_statement_value() != nil && ctx.Insert_statement_value().Derived_table() != nil && ctx.Insert_statement_value().Derived_table().Table_value_constructor() != nil {
		tvc := ctx.Insert_statement_value().Derived_table().Table_value_constructor()
		l.insertCount += len(tvc.AllExpression_list_())
		return
	}

	// Track DMLs.
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, trimStatement(l.text))
	}
}

func (l *tsqlChangedResourceExtractListener) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if ctx.Ddl_object() != nil && ctx.Ddl_object().Full_table_name() != nil {
		table, err := NormalizeFullTableName(ctx.Ddl_object().Full_table_name())
		if err == nil && table != nil && table.Table != "" {
			d := table.Database
			if d == "" {
				d = l.currentDatabase
			}
			s := table.Schema
			if s == "" {
				s = l.currentSchema
			}
			l.changedResources.AddTable(
				d,
				s,
				&storepb.ChangedResourceTable{
					Name:   table.Table,
					Ranges: []*storepb.Range{base.NewRange(l.statement, trimStatement(l.text))},
				},
				false,
			)
		}
	}

	// Track DMLs.
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, trimStatement(l.text))
	}
}

func (l *tsqlChangedResourceExtractListener) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	from := ctx.Delete_statement_from()
	if from.Ddl_object() != nil && from.Ddl_object().Full_table_name() != nil {
		table, err := NormalizeFullTableName(from.Ddl_object().Full_table_name())
		if err == nil && table != nil && table.Table != "" {
			d := table.Database
			if d == "" {
				d = l.currentDatabase
			}
			s := table.Schema
			if s == "" {
				s = l.currentSchema
			}
			l.changedResources.AddTable(
				d,
				s,
				&storepb.ChangedResourceTable{
					Name:   table.Table,
					Ranges: []*storepb.Range{base.NewRange(l.statement, trimStatement(l.text))},
				},
				false,
			)
		}
	}

	// Track DMLs.
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, trimStatement(l.text))
	}
}

func trimStatement(statement string) string {
	// TODO(d): why test is "UPDATE t1 SET c1 = 5\n;".
	return strings.TrimLeftFunc(strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon), unicode.IsSpace) + ";"
}
