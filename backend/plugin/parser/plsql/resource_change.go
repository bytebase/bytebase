package plsql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_ORACLE, extractChangedResources)
}

func extractChangedResources(currentDatabase string, _ string, dbSchema *model.DatabaseSchema, asts any, statement string) (*base.ChangeSummary, error) {
	// currentDatabase is the same as currentSchema for Oracle.
	tree, ok := asts.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert ast to antlr.Tree")
	}

	changedResources := model.NewChangedResources(dbSchema)
	l := &plsqlChangedResourceExtractListener{
		currentSchema:    currentDatabase,
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

type plsqlChangedResourceExtractListener struct {
	*parser.BasePlSqlParserListener

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

func (l *plsqlChangedResourceExtractListener) EnterUnit_statement(ctx *parser.Unit_statementContext) {
	l.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterCreate_table is called when production create_table is entered.
func (l *plsqlChangedResourceExtractListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	var schema string
	if ctx.Schema_name() != nil {
		schema = NormalizeIdentifierContext(ctx.Schema_name().Identifier())
	}
	if schema == "" {
		schema = l.currentSchema
	}

	tableName := NormalizeIdentifierContext(ctx.Table_name().Identifier())
	l.changedResources.AddTable(
		schema,
		"",
		&storepb.ChangedResourceTable{
			Name:   tableName,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
		false)
}

// EnterDrop_table is called when production drop_table is entered.
func (l *plsqlChangedResourceExtractListener) EnterDrop_table(ctx *parser.Drop_tableContext) {
	if ctx.Tableview_name() == nil {
		return
	}

	var schema, table string
	if ctx.Tableview_name().Id_expression() == nil {
		table = NormalizeIdentifierContext(ctx.Tableview_name().Identifier())
	} else {
		schema = NormalizeIdentifierContext(ctx.Tableview_name().Identifier())
		table = NormalizeIDExpression(ctx.Tableview_name().Id_expression())
	}
	if schema == "" {
		schema = l.currentSchema
	}

	l.changedResources.AddTable(
		schema,
		"",
		&storepb.ChangedResourceTable{
			Name:   table,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
		true)
}

// EnterAlter_table is called when production alter_table is entered.
func (l *plsqlChangedResourceExtractListener) EnterAlter_table(ctx *parser.Alter_tableContext) {
	if ctx.Tableview_name() == nil {
		return
	}

	var schema, table string
	if ctx.Tableview_name().Id_expression() == nil {
		table = NormalizeIdentifierContext(ctx.Tableview_name().Identifier())
	} else {
		schema = NormalizeIdentifierContext(ctx.Tableview_name().Identifier())
		table = NormalizeIDExpression(ctx.Tableview_name().Id_expression())
	}
	if schema == "" {
		schema = l.currentSchema
	}

	l.changedResources.AddTable(
		schema,
		"",
		&storepb.ChangedResourceTable{
			Name:   table,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
		true)
}

// EnterAlter_table_properties is called when production alter_table_properties is entered.
func (l *plsqlChangedResourceExtractListener) EnterAlter_table_properties(ctx *parser.Alter_table_propertiesContext) {
	if ctx.RENAME() == nil {
		return
	}

	// Rename table.
	var schema, table string
	if ctx.Tableview_name().Id_expression() == nil {
		table = NormalizeIdentifierContext(ctx.Tableview_name().Identifier())
	} else {
		schema = NormalizeIdentifierContext(ctx.Tableview_name().Identifier())
		table = NormalizeIDExpression(ctx.Tableview_name().Id_expression())
	}
	if schema == "" {
		schema = l.currentSchema
	}

	l.changedResources.AddTable(
		schema,
		"",
		&storepb.ChangedResourceTable{
			Name:   table,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
		false)
}

// EnterAlter_table is called when production create_index is entered.
func (l *plsqlChangedResourceExtractListener) EnterCreate_index(ctx *parser.Create_indexContext) {
	tableIndexClause := ctx.Table_index_clause()
	if tableIndexClause == nil {
		return
	}

	var schema, table string
	if tableIndexClause.Tableview_name().Id_expression() == nil {
		table = NormalizeIdentifierContext(tableIndexClause.Tableview_name().Identifier())
	} else {
		schema = NormalizeIdentifierContext(tableIndexClause.Tableview_name().Identifier())
		table = NormalizeIDExpression(tableIndexClause.Tableview_name().Id_expression())
	}
	if schema == "" {
		schema = l.currentSchema
	}

	l.changedResources.AddTable(
		schema,
		"",
		&storepb.ChangedResourceTable{
			Name:   table,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
		false)
}

// EnterDrop_index is called when production drop_index is entered.
func (l *plsqlChangedResourceExtractListener) EnterDrop_index(ctx *parser.Drop_indexContext) {
	schema, index := NormalizeIndexName(ctx.Index_name())
	if schema == "" {
		schema = l.currentSchema
	}
	foundSchema := l.dbSchema.GetDatabaseMetadata().GetSchema(schema)
	if foundSchema == nil {
		return
	}
	indexes := foundSchema.GetIndexes(index)
	if len(indexes) == 0 {
		return
	}
	foundTable := indexes[0].GetTableProto().GetName()

	l.changedResources.AddTable(
		schema,
		"",
		&storepb.ChangedResourceTable{
			Name:   foundTable,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
		false)
}

// EnterCreate_view is called when production create_view is entered.
func (l *plsqlChangedResourceExtractListener) EnterCreate_view(ctx *parser.Create_viewContext) {
	var schema, view string
	if ctx.Schema_name() != nil {
		schema = NormalizeIdentifierContext(ctx.Schema_name().Identifier())
	}
	if len(ctx.AllId_expression()) > 0 {
		view = NormalizeIDExpression(ctx.AllId_expression()[0])
	}
	if schema == "" {
		schema = l.currentSchema
	}

	l.changedResources.AddView(
		schema,
		"",
		&storepb.ChangedResourceView{
			Name:   view,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
	)
}

func (l *plsqlChangedResourceExtractListener) EnterDrop_view(ctx *parser.Drop_viewContext) {
	var schema, view string
	tableViewName := ctx.Tableview_name()
	if tableViewName.Id_expression() == nil {
		view = NormalizeIdentifierContext(tableViewName.Identifier())
	} else {
		schema = NormalizeIdentifierContext(tableViewName.Identifier())
		view = NormalizeIDExpression(tableViewName.Id_expression())
	}
	if schema == "" {
		schema = l.currentSchema
	}

	l.changedResources.AddView(
		schema,
		"",
		&storepb.ChangedResourceView{
			Name:   view,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
	)
}

func (l *plsqlChangedResourceExtractListener) EnterAlter_view(ctx *parser.Alter_viewContext) {
	var schema, view string
	tableViewName := ctx.Tableview_name()
	if tableViewName.Id_expression() == nil {
		view = NormalizeIdentifierContext(tableViewName.Identifier())
	} else {
		schema = NormalizeIdentifierContext(tableViewName.Identifier())
		view = NormalizeIDExpression(tableViewName.Id_expression())
	}
	if schema == "" {
		schema = l.currentSchema
	}

	l.changedResources.AddView(
		schema,
		"",
		&storepb.ChangedResourceView{
			Name:   view,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
	)
}

func (l *plsqlChangedResourceExtractListener) EnterCreate_procedure_body(ctx *parser.Create_procedure_bodyContext) {
	var schema, procedure string
	procedureName := ctx.Procedure_name()
	if procedureName.Id_expression() == nil {
		procedure = NormalizeIdentifierContext(procedureName.Identifier())
	} else {
		schema = NormalizeIdentifierContext(procedureName.Identifier())
		procedure = NormalizeIDExpression(procedureName.Id_expression())
	}
	if schema == "" {
		schema = l.currentSchema
	}

	l.changedResources.AddProcedure(
		schema,
		"",
		&storepb.ChangedResourceProcedure{
			Name:   procedure,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
	)
}

func (l *plsqlChangedResourceExtractListener) EnterDrop_procedure(ctx *parser.Drop_procedureContext) {
	var schema, procedure string
	procedureName := ctx.Procedure_name()
	if procedureName.Id_expression() == nil {
		procedure = NormalizeIdentifierContext(procedureName.Identifier())
	} else {
		schema = NormalizeIdentifierContext(procedureName.Identifier())
		procedure = NormalizeIDExpression(procedureName.Id_expression())
	}
	if schema == "" {
		schema = l.currentSchema
	}

	l.changedResources.AddProcedure(
		schema,
		"",
		&storepb.ChangedResourceProcedure{
			Name:   procedure,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
	)
}

func (l *plsqlChangedResourceExtractListener) EnterAlter_procedure(ctx *parser.Alter_procedureContext) {
	var schema, procedure string
	procedureName := ctx.Procedure_name()
	if procedureName.Id_expression() == nil {
		procedure = NormalizeIdentifierContext(procedureName.Identifier())
	} else {
		schema = NormalizeIdentifierContext(procedureName.Identifier())
		procedure = NormalizeIDExpression(procedureName.Id_expression())
	}
	if schema == "" {
		schema = l.currentSchema
	}

	l.changedResources.AddProcedure(
		schema,
		"",
		&storepb.ChangedResourceProcedure{
			Name:   procedure,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
	)
}

func (l *plsqlChangedResourceExtractListener) EnterCreate_function_body(ctx *parser.Create_function_bodyContext) {
	var schema, function string
	functionName := ctx.Function_name()
	if functionName.Id_expression() == nil {
		function = NormalizeIdentifierContext(functionName.Identifier())
	} else {
		schema = NormalizeIdentifierContext(functionName.Identifier())
		function = NormalizeIDExpression(functionName.Id_expression())
	}
	if schema == "" {
		schema = l.currentSchema
	}

	l.changedResources.AddFunction(
		schema,
		"",
		&storepb.ChangedResourceFunction{
			Name:   function,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
	)
}

func (l *plsqlChangedResourceExtractListener) EnterDrop_function(ctx *parser.Drop_functionContext) {
	var schema, function string
	functionName := ctx.Function_name()
	if functionName.Id_expression() == nil {
		function = NormalizeIdentifierContext(functionName.Identifier())
	} else {
		schema = NormalizeIdentifierContext(functionName.Identifier())
		function = NormalizeIDExpression(functionName.Id_expression())
	}
	if schema == "" {
		schema = l.currentSchema
	}

	l.changedResources.AddFunction(
		schema,
		"",
		&storepb.ChangedResourceFunction{
			Name:   function,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
	)
}

func (l *plsqlChangedResourceExtractListener) EnterAlter_function(ctx *parser.Alter_functionContext) {
	var schema, function string
	functionName := ctx.Function_name()
	if functionName.Id_expression() == nil {
		function = NormalizeIdentifierContext(functionName.Identifier())
	} else {
		schema = NormalizeIdentifierContext(functionName.Identifier())
		function = NormalizeIDExpression(functionName.Id_expression())
	}
	if schema == "" {
		schema = l.currentSchema
	}

	l.changedResources.AddFunction(
		schema,
		"",
		&storepb.ChangedResourceFunction{
			Name:   function,
			Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
		},
	)
}

func (l *plsqlChangedResourceExtractListener) EnterInsert_statement(ctx *parser.Insert_statementContext) {
	var resources []base.SchemaResource
	if ctx.Single_table_insert() != nil {
		resources = append(resources, l.extractTableReference(ctx.Single_table_insert().Insert_into_clause().General_table_ref())...)
	}
	if ctx.Multi_table_insert() != nil {
		for _, item := range ctx.Multi_table_insert().AllMulti_table_element() {
			resources = append(resources, l.extractTableReference(item.Insert_into_clause().General_table_ref())...)
		}
		if ctx.Multi_table_insert().Conditional_insert_clause() != nil {
			conditionCtx := ctx.Multi_table_insert().Conditional_insert_clause()
			for _, item := range conditionCtx.AllConditional_insert_when_part() {
				for _, multiItem := range item.AllMulti_table_element() {
					resources = append(resources, l.extractTableReference(multiItem.Insert_into_clause().General_table_ref())...)
				}
			}
			if conditionCtx.Conditional_insert_else_part() != nil {
				for _, item := range conditionCtx.Conditional_insert_else_part().AllMulti_table_element() {
					resources = append(resources, l.extractTableReference(item.Insert_into_clause().General_table_ref())...)
				}
			}
		}
	}
	for _, resource := range resources {
		l.changedResources.AddTable(
			resource.Database,
			"",
			&storepb.ChangedResourceTable{
				Name:   resource.Table,
				Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
			},
			false,
		)
	}

	if ctx.Single_table_insert() != nil && ctx.Single_table_insert().Values_clause() != nil {
		// Oracle allows only one value.
		// https://docs.oracle.com/en/database/other-databases/nosql-database/22.1/sqlreferencefornosql/insert-statement.html
		l.insertCount++
		return
	}
	// Track DMLs.
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, l.text)
	}
}

func (l *plsqlChangedResourceExtractListener) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	// Track DMLs.
	resources := l.extractTableReference(ctx.General_table_ref())
	for _, resource := range resources {
		l.changedResources.AddTable(
			resource.Database,
			"",
			&storepb.ChangedResourceTable{
				Name:   resource.Table,
				Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
			},
			false,
		)
	}
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, l.text)
	}
}

func (l *plsqlChangedResourceExtractListener) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	// Track DMLs.
	resources := l.extractTableReference(ctx.General_table_ref())
	for _, resource := range resources {
		l.changedResources.AddTable(
			resource.Database,
			"",
			&storepb.ChangedResourceTable{
				Name:   resource.Table,
				Ranges: []*storepb.Range{base.NewRange(l.statement, l.text)},
			},
			false,
		)
	}
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, l.text)
	}
}

func (l *plsqlChangedResourceExtractListener) extractTableReference(ctx parser.IGeneral_table_refContext) []base.SchemaResource {
	resources := make([]base.SchemaResource, 0)
	if ctx == nil {
		return resources
	}

	if ctx.Dml_table_expression_clause() != nil && ctx.Dml_table_expression_clause().Tableview_name() != nil {
		_, schema, table := NormalizeTableViewName(l.currentSchema, ctx.Dml_table_expression_clause().Tableview_name())
		resources = append(resources, base.SchemaResource{
			Database: schema,
			Table:    table,
		})
	}

	return resources
}
