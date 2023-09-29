package plsql

import (
	"errors"
	"fmt"
	"sort"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func ExtractChangedResources(currentDatabase string, currentSchema string, statement string) ([]base.SchemaResource, error) {
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

// ValidateForEditor validates the given PLSQL for editor.
func ValidateForEditor(tree antlr.Tree) error {
	l := &plsqlValidateForEditorListener{
		validate: true,
	}
	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	if !l.validate {
		return errors.New("only support SELECT sql statement")
	}
	return nil
}

type plsqlValidateForEditorListener struct {
	*parser.BasePlSqlParserListener

	validate bool
}

// EnterSql_script is called when production sql_script is entered.
func (l *plsqlValidateForEditorListener) EnterSql_script(ctx *parser.Sql_scriptContext) {
	if len(ctx.AllSql_plus_command()) > 0 {
		l.validate = false
	}
}

// EnterUnit_statement is called when production unit_statement is entered.
func (l *plsqlValidateForEditorListener) EnterUnit_statement(ctx *parser.Unit_statementContext) {
	if ctx.Data_manipulation_language_statements() == nil {
		l.validate = false
	}
}

// EnterData_manipulation_language_statements is called when production data_manipulation_language_statements is entered.
func (l *plsqlValidateForEditorListener) EnterData_manipulation_language_statements(ctx *parser.Data_manipulation_language_statementsContext) {
	if ctx.Select_statement() == nil && ctx.Explain_statement() == nil {
		l.validate = false
	}
}

// EquivalentType returns true if the given type is equivalent to the given text.
func EquivalentType(tp parser.IDatatypeContext, text string) (bool, error) {
	tree, _, err := ParsePLSQL(fmt.Sprintf(`CREATE TABLE t(a %s);`, text))
	if err != nil {
		return false, err
	}

	listener := &typeEquivalentListener{tp: tp, equivalent: false}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	return listener.equivalent, nil
}

type typeEquivalentListener struct {
	*parser.BasePlSqlParserListener

	tp         parser.IDatatypeContext
	equivalent bool
}

// EnterColumn_definition is called when production column_definition is entered.
func (l *typeEquivalentListener) EnterColumn_definition(ctx *parser.Column_definitionContext) {
	if ctx.Datatype() != nil {
		l.equivalent = equalDataType(l.tp, ctx.Datatype())
	}
}

func equalDataType(lType parser.IDatatypeContext, rType parser.IDatatypeContext) bool {
	if lType == nil || rType == nil {
		return false
	}
	lNative := lType.Native_datatype_element()
	rNative := rType.Native_datatype_element()

	if lNative != nil && rNative != nil {
		switch {
		case lNative.BINARY_INTEGER() != nil:
			return rNative.BINARY_INTEGER() != nil
		case lNative.PLS_INTEGER() != nil:
			return rNative.PLS_INTEGER() != nil
		case lNative.NATURAL() != nil:
			return rNative.NATURAL() != nil
		case lNative.BINARY_FLOAT() != nil:
			return rNative.BINARY_FLOAT() != nil
		case lNative.BINARY_DOUBLE() != nil:
			return rNative.BINARY_DOUBLE() != nil
		case lNative.NATURALN() != nil:
			return rNative.NATURALN() != nil
		case lNative.POSITIVE() != nil:
			return rNative.POSITIVE() != nil
		case lNative.POSITIVEN() != nil:
			return rNative.POSITIVEN() != nil
		case lNative.SIGNTYPE() != nil:
			return rNative.SIGNTYPE() != nil
		case lNative.SIMPLE_INTEGER() != nil:
			return rNative.SIMPLE_INTEGER() != nil
		case lNative.NVARCHAR2() != nil:
			return rNative.NVARCHAR2() != nil
		case lNative.DEC() != nil:
			return rNative.DEC() != nil
		case lNative.INTEGER() != nil:
			return rNative.INTEGER() != nil
		case lNative.INT() != nil:
			return rNative.INT() != nil
		case lNative.NUMERIC() != nil:
			return rNative.NUMERIC() != nil
		case lNative.SMALLINT() != nil:
			return rNative.SMALLINT() != nil
		case lNative.NUMBER() != nil:
			return rNative.NUMBER() != nil
		case lNative.DECIMAL() != nil:
			return rNative.DECIMAL() != nil
		case lNative.DOUBLE() != nil:
			return rNative.DOUBLE() != nil
		case lNative.FLOAT() != nil:
			return rNative.FLOAT() != nil
		case lNative.REAL() != nil:
			return rNative.REAL() != nil
		case lNative.NCHAR() != nil:
			return rNative.NCHAR() != nil
		case lNative.LONG() != nil:
			return rNative.LONG() != nil
		case lNative.CHAR() != nil:
			return rNative.CHAR() != nil
		case lNative.CHARACTER() != nil:
			return rNative.CHARACTER() != nil
		case lNative.VARCHAR2() != nil:
			return rNative.VARCHAR2() != nil
		case lNative.VARCHAR() != nil:
			return rNative.VARCHAR() != nil
		case lNative.STRING() != nil:
			return rNative.STRING() != nil
		case lNative.RAW() != nil:
			return rNative.RAW() != nil
		case lNative.BOOLEAN() != nil:
			return rNative.BOOLEAN() != nil
		case lNative.DATE() != nil:
			return rNative.DATE() != nil
		case lNative.ROWID() != nil:
			return rNative.ROWID() != nil
		case lNative.UROWID() != nil:
			return rNative.UROWID() != nil
		case lNative.YEAR() != nil:
			return rNative.YEAR() != nil
		case lNative.MONTH() != nil:
			return rNative.MONTH() != nil
		case lNative.DAY() != nil:
			return rNative.DAY() != nil
		case lNative.HOUR() != nil:
			return rNative.HOUR() != nil
		case lNative.MINUTE() != nil:
			return rNative.MINUTE() != nil
		case lNative.SECOND() != nil:
			return rNative.SECOND() != nil
		case lNative.TIMEZONE_HOUR() != nil:
			return rNative.TIMEZONE_HOUR() != nil
		case lNative.TIMEZONE_MINUTE() != nil:
			return rNative.TIMEZONE_MINUTE() != nil
		case lNative.TIMEZONE_REGION() != nil:
			return rNative.TIMEZONE_REGION() != nil
		case lNative.TIMEZONE_ABBR() != nil:
			return rNative.TIMEZONE_ABBR() != nil
		case lNative.TIMESTAMP() != nil:
			return rNative.TIMESTAMP() != nil
		case lNative.TIMESTAMP_UNCONSTRAINED() != nil:
			return rNative.TIMESTAMP_UNCONSTRAINED() != nil
		case lNative.TIMESTAMP_TZ_UNCONSTRAINED() != nil:
			return rNative.TIMESTAMP_TZ_UNCONSTRAINED() != nil
		case lNative.TIMESTAMP_LTZ_UNCONSTRAINED() != nil:
			return rNative.TIMESTAMP_LTZ_UNCONSTRAINED() != nil
		case lNative.YMINTERVAL_UNCONSTRAINED() != nil:
			return rNative.YMINTERVAL_UNCONSTRAINED() != nil
		case lNative.DSINTERVAL_UNCONSTRAINED() != nil:
			return rNative.DSINTERVAL_UNCONSTRAINED() != nil
		case lNative.BFILE() != nil:
			return rNative.BFILE() != nil
		case lNative.BLOB() != nil:
			return rNative.BLOB() != nil
		case lNative.CLOB() != nil:
			return rNative.CLOB() != nil
		case lNative.NCLOB() != nil:
			return rNative.NCLOB() != nil
		case lNative.MLSLABEL() != nil:
			return rNative.MLSLABEL() != nil
		default:
			return false
		}
	}

	if lNative != nil || rNative != nil {
		return false
	}

	return lType.GetText() == rType.GetText()
}
