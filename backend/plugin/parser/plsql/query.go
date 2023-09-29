package plsql

import (
	"sort"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func ExtractOracleChangedResources(currentDatabase string, currentSchema string, statement string) ([]base.SchemaResource, error) {
	tree, _, err := ParsePLSQL(statement)
	if err != nil {
		return nil, err
	}

	l := &plsqlChangedResourceExtractListener{
		currentDatabase: currentDatabase,
		currentSchema:   currentSchema,
		resourceMap:     make(map[string]base.SchemaResource),
	}

	var result []base.SchemaResource
	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	for _, resource := range l.resourceMap {
		result = append(result, resource)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})

	return result, nil
}

type plsqlChangedResourceExtractListener struct {
	*parser.BasePlSqlParserListener

	currentDatabase string
	currentSchema   string
	resourceMap     map[string]base.SchemaResource
}

// EnterCreate_table is called when production create_table is entered.
func (l *plsqlChangedResourceExtractListener) EnterCreate_table(ctx *parser.Create_tableContext) {
	resource := base.SchemaResource{
		Database: l.currentDatabase,
		Schema:   l.currentSchema,
		Table:    NormalizeIdentifierContext(ctx.Table_name().Identifier()),
	}

	if ctx.Schema_name() != nil {
		resource.Schema = NormalizeIdentifierContext(ctx.Schema_name().Identifier())
	}
	l.resourceMap[resource.String()] = resource
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

	resource := base.SchemaResource{
		Database: l.currentDatabase,
		Schema:   result[0],
		Table:    result[1],
	}
	l.resourceMap[resource.String()] = resource
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

	resource := base.SchemaResource{
		Database: l.currentDatabase,
		Schema:   result[0],
		Table:    result[1],
	}
	l.resourceMap[resource.String()] = resource
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

	resource := base.SchemaResource{
		Database: l.currentDatabase,
		Schema:   result[0],
		Table:    result[1],
	}
	l.resourceMap[resource.String()] = resource
}

func ExtractOracleResourceList(currentDatabase string, currentSchema string, statement string) ([]base.SchemaResource, error) {
	tree, _, err := ParsePLSQL(statement)
	if err != nil {
		return nil, err
	}

	l := &plsqlResourceExtractListener{
		currentDatabase: currentDatabase,
		currentSchema:   currentSchema,
		resourceMap:     make(map[string]base.SchemaResource),
	}

	var result []base.SchemaResource
	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	for _, resource := range l.resourceMap {
		result = append(result, resource)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})

	return result, nil
}

type plsqlResourceExtractListener struct {
	*parser.BasePlSqlParserListener

	currentDatabase string
	currentSchema   string
	resourceMap     map[string]base.SchemaResource
}

func (l *plsqlResourceExtractListener) EnterTableview_name(ctx *parser.Tableview_nameContext) {
	if ctx.Identifier() == nil {
		return
	}

	result := []string{NormalizeIdentifierContext(ctx.Identifier())}
	if ctx.Id_expression() != nil {
		result = append(result, NormalizeIDExpression(ctx.Id_expression()))
	}
	if len(result) == 1 {
		result = []string{l.currentSchema, result[0]}
	}

	resource := base.SchemaResource{
		Database: l.currentDatabase,
		Schema:   result[0],
		Table:    result[1],
	}
	l.resourceMap[resource.String()] = resource
}
