package plsql

import (
	"sort"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_ORACLE, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_DM, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_OCEANBASE_ORACLE, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
func validateQuery(statement string) (bool, bool, error) {
	tree, _, err := ParsePLSQL(statement)
	if err != nil {
		return false, false, err
	}
	l := &queryValidateListener{
		validate: true,
	}
	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	if !l.validate {
		return false, false, nil
	}
	return true, true, nil
}

type queryValidateListener struct {
	*parser.BasePlSqlParserListener

	validate bool
}

// EnterSql_script is called when production sql_script is entered.
func (l *queryValidateListener) EnterSql_script(ctx *parser.Sql_scriptContext) {
	if len(ctx.AllSql_plus_command()) > 0 {
		l.validate = false
	}
}

// EnterUnit_statement is called when production unit_statement is entered.
func (l *queryValidateListener) EnterUnit_statement(ctx *parser.Unit_statementContext) {
	if ctx.Data_manipulation_language_statements() == nil {
		l.validate = false
	}
}

// EnterData_manipulation_language_statements is called when production data_manipulation_language_statements is entered.
func (l *queryValidateListener) EnterData_manipulation_language_statements(ctx *parser.Data_manipulation_language_statementsContext) {
	if ctx.Select_statement() == nil && ctx.Explain_statement() == nil {
		l.validate = false
	}
}

func ExtractResourceList(currentDatabase string, _ string, statement string) ([]base.SchemaResource, error) {
	tree, _, err := ParsePLSQL(statement)
	if err != nil {
		return nil, err
	}

	l := &plsqlResourceExtractListener{
		currentDatabase: currentDatabase,
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
	resourceMap     map[string]base.SchemaResource
}

func (l *plsqlResourceExtractListener) EnterTableview_name(ctx *parser.Tableview_nameContext) {
	if ctx.Identifier() == nil {
		return
	}

	var schema, tableOrView string
	if ctx.Id_expression() == nil {
		tableOrView = NormalizeIdentifierContext(ctx.Identifier())
	} else {
		schema = NormalizeIdentifierContext(ctx.Identifier())
		tableOrView = NormalizeIDExpression(ctx.Id_expression())
	}
	if schema == "" {
		schema = l.currentDatabase
	}

	resource := base.SchemaResource{
		Database: schema,
		Table:    tableOrView,
	}
	l.resourceMap[resource.String()] = resource
}
