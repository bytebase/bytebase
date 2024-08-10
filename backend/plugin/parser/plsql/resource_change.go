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

func extractChangedResources(currentDatabase string, currentSchema string, dbSchema *model.DBSchema, asts any, _ string) (*base.ChangeSummary, error) {
	tree, ok := asts.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert ast to antlr.Tree")
	}

	changedResources := model.NewChangedResources(dbSchema)
	l := &plsqlChangedResourceExtractListener{
		currentDatabase:  currentDatabase,
		currentSchema:    currentSchema,
		changedResources: changedResources,
	}

	antlr.ParseTreeWalkerDefault.Walk(l, tree)

	return &base.ChangeSummary{
		ChangedResources: changedResources,
	}, nil
}

type plsqlChangedResourceExtractListener struct {
	*parser.BasePlSqlParserListener

	currentDatabase  string
	currentSchema    string
	changedResources *model.ChangedResources
}

// EnterCreate_table is called when production create_table is entered.
func (l *plsqlChangedResourceExtractListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	schemaName := l.currentSchema
	if ctx.Schema_name() != nil {
		schemaName = NormalizeIdentifierContext(ctx.Schema_name().Identifier())
	}
	tableName := NormalizeIdentifierContext(ctx.Table_name().Identifier())
	l.changedResources.AddTable(
		l.currentDatabase,
		schemaName,
		&storepb.ChangedResourceTable{
			Name: tableName,
		},
		false)
}

// EnterDrop_table is called when production drop_table is entered.
func (l *plsqlChangedResourceExtractListener) EnterDrop_table(ctx *parser.Drop_tableContext) {
	result := []string{NormalizeIdentifierContext(ctx.Tableview_name().Identifier())}
	if ctx.Tableview_name().Id_expression() != nil {
		result = append(result, NormalizeIDExpression(ctx.Tableview_name().Id_expression()))
	}
	if len(result) == 1 {
		result = []string{l.currentSchema, result[0]}
	}

	schemaName := result[0]
	tableName := result[1]
	l.changedResources.AddTable(
		l.currentDatabase,
		schemaName,
		&storepb.ChangedResourceTable{
			Name: tableName,
		},
		false)
}

// EnterAlter_table is called when production alter_table is entered.
func (l *plsqlChangedResourceExtractListener) EnterAlter_table(ctx *parser.Alter_tableContext) {
	result := []string{NormalizeIdentifierContext(ctx.Tableview_name().Identifier())}
	if ctx.Tableview_name().Id_expression() != nil {
		result = append(result, NormalizeIDExpression(ctx.Tableview_name().Id_expression()))
	}
	if len(result) == 1 {
		result = []string{l.currentSchema, result[0]}
	}

	schemaName := result[0]
	tableName := result[1]
	l.changedResources.AddTable(
		l.currentDatabase,
		schemaName,
		&storepb.ChangedResourceTable{
			Name: tableName,
		},
		false)
}

// EnterAlter_table_properties is called when production alter_table_properties is entered.
func (l *plsqlChangedResourceExtractListener) EnterAlter_table_properties(ctx *parser.Alter_table_propertiesContext) {
	if ctx.RENAME() == nil {
		return
	}
	result := []string{NormalizeIdentifierContext(ctx.Tableview_name().Identifier())}
	if ctx.Tableview_name().Id_expression() != nil {
		result = append(result, NormalizeIDExpression(ctx.Tableview_name().Id_expression()))
	}
	if len(result) == 1 {
		result = []string{l.currentSchema, result[0]}
	}

	schemaName := result[0]
	tableName := result[1]
	l.changedResources.AddTable(
		l.currentDatabase,
		schemaName,
		&storepb.ChangedResourceTable{
			Name: tableName,
		},
		false)
}
