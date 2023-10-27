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
	base.RegisterExtractResourceListFunc(storepb.Engine_ORACLE, ExtractResourceList)
	base.RegisterExtractResourceListFunc(storepb.Engine_DM, ExtractResourceList)
	base.RegisterExtractResourceListFunc(storepb.Engine_OCEANBASE_ORACLE, ExtractResourceList)
}

// validateQuery validates the SQL statement for SQL editor.
func validateQuery(statement string) (bool, error) {
	tree, _, err := ParsePLSQL(statement)
	if err != nil {
		return false, err
	}
	l := &queryValidateListener{
		validate: true,
	}
	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	if !l.validate {
		return false, nil
	}
	return true, nil
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

func ExtractResourceList(currentDatabase string, currentSchema string, statement string) ([]base.SchemaResource, error) {
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
