package plsql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_ORACLE, extractChangedResources)
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, statement string) (*base.ChangeSummary, error) {
	// currentDatabase is the same as currentSchema for Oracle.
	changedResources := model.NewChangedResources(dbMetadata)
	l := &plsqlChangedResourceExtractListener{
		currentSchema:    currentDatabase,
		dbMetadata:       dbMetadata,
		changedResources: changedResources,
		statement:        statement,
	}

	for _, ast := range asts {
		antlrAST, ok := base.GetANTLRAST(ast)
		if !ok {
			return nil, errors.New("expected ANTLR AST for Oracle")
		}
		antlr.ParseTreeWalkerDefault.Walk(l, antlrAST.Tree)
	}

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
	dbMetadata       *model.DatabaseMetadata
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
			Name: tableName,
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
			Name: table,
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
			Name: table,
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
			Name: table,
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
			Name: table,
		},
		false)
}

// EnterDrop_index is called when production drop_index is entered.
func (l *plsqlChangedResourceExtractListener) EnterDrop_index(ctx *parser.Drop_indexContext) {
	schema, index := NormalizeIndexName(ctx.Index_name())
	if schema == "" {
		schema = l.currentSchema
	}
	foundSchema := l.dbMetadata.GetSchemaMetadata(schema)
	if foundSchema == nil {
		return
	}
	foundIndex := foundSchema.GetIndex(index)
	if foundIndex == nil {
		return
	}
	foundTable := foundIndex.GetTableProto().GetName()

	l.changedResources.AddTable(
		schema,
		"",
		&storepb.ChangedResourceTable{
			Name: foundTable,
		},
		false)
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
				Name: resource.Table,
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
				Name: resource.Table,
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
				Name: resource.Table,
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
