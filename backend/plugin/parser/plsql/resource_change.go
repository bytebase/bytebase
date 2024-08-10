package plsql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_ORACLE, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_DM, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_OCEANBASE_ORACLE, extractChangedResources)
}

func extractChangedResources(currentDatabase string, _ string, dbSchema *model.DBSchema, asts any, _ string) (*base.ChangeSummary, error) {
	// currentDatabase is the same as currentSchema for Oracle.
	tree, ok := asts.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert ast to antlr.Tree")
	}

	changedResources := model.NewChangedResources(dbSchema)
	l := &plsqlChangedResourceExtractListener{
		currentSchema:    currentDatabase,
		changedResources: changedResources,
	}

	antlr.ParseTreeWalkerDefault.Walk(l, tree)

	return &base.ChangeSummary{
		ChangedResources: changedResources,
	}, nil
}

type plsqlChangedResourceExtractListener struct {
	*parser.BasePlSqlParserListener

	currentSchema    string
	changedResources *model.ChangedResources
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
		schema,
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
		schema,
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
		schema,
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
		schema,
		&storepb.ChangedResourceTable{
			Name: table,
		},
		false)
}

// EnterAlter_table is called when production alter_table is entered.
func (l *plsqlChangedResourceExtractListener) EnterCreate_index(ctx *parser.Create_indexContext) {
	if ctx.Table_index_clause() == nil {
		return
	}

	tableIndexClause := ctx.Table_index_clause()
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
		schema,
		&storepb.ChangedResourceTable{
			Name: table,
		},
		false)
}
