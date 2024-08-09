package plsql

import (
	"sort"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_ORACLE, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_DM, extractChangedResources)
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_OCEANBASE_ORACLE, extractChangedResources)
}

func extractChangedResources(currentDatabase string, currentSchema string, asts any, _ string) (*base.ChangeSummary, error) {
	tree, ok := asts.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert ast to antlr.Tree")
	}

	l := &plsqlChangedResourceExtractListener{
		currentDatabase: currentDatabase,
		currentSchema:   currentSchema,
		resourceMap:     make(map[string]base.SchemaResource),
	}

	antlr.ParseTreeWalkerDefault.Walk(l, tree)

	var resources []base.SchemaResource
	for _, resource := range l.resourceMap {
		resources = append(resources, resource)
	}
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].String() < resources[j].String()
	})
	var resourceChanges []*base.ResourceChange
	for _, resource := range resources {
		resourceChanges = append(resourceChanges, &base.ResourceChange{
			Resource: resource,
		})
	}

	return &base.ChangeSummary{
		ResourceChanges: resourceChanges,
	}, nil
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
