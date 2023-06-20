// Package util implements the util functions.
package util

import (
	"github.com/antlr4-go/antlr/v4"
	plsql "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

func (extractor *sensitiveFieldExtractor) extractOracleSensitiveField(statement string) ([]db.SensitiveField, error) {
	tree, err := parser.ParsePLSQL(statement)
	if err != nil {
		return nil, err
	}
	if tree == nil {
		return nil, nil
	}

	listener := &selectStatementListener{
		extractor: extractor,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	return listener.result, listener.err
}

type selectStatementListener struct {
	*plsql.BasePlSqlParserListener

	extractor *sensitiveFieldExtractor
	result    []db.SensitiveField
	err       error
}

// EnterSelect_statement is called when production select_statement is entered.
func (l *selectStatementListener) EnterSelect_statement(ctx *plsql.Select_statementContext) {
	parent := ctx.GetParent()
	if parent == nil {
		return
	}

	if _, ok := parent.(*plsql.Data_manipulation_language_statementsContext); ok {
		if _, ok := parent.GetParent().(*plsql.Unit_statementContext); ok {
			fieldList, err := l.extractor.plsqlExtractContext(ctx)
			if err != nil {
				l.err = err
				return
			}

			for _, field := range fieldList {
				l.result = append(l.result, db.SensitiveField{
					Name:      field.name,
					Sensitive: field.sensitive,
				})
			}
		}
	}
}

func (extractor *sensitiveFieldExtractor) plsqlExtractContext(ctx antlr.ParserRuleContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	switch ctx := ctx.(type) {
	case plsql.ISelect_statementContext:
		return extractor.plsqlExtractSelect(ctx)
	default:
		return nil, nil
	}
}

func (extractor *sensitiveFieldExtractor) plsqlExtractSelect(ctx plsql.ISelect_statementContext) ([]fieldInfo, error) {
	selectOnlyStatement := ctx.Select_only_statement()
	if selectOnlyStatement == nil {
		return nil, nil
	}

	// TODO(rebelice): handle CTE

	subquery := selectOnlyStatement.Subquery()
	if subquery == nil {
		return nil, nil
	}

	return extractor.plsqlExtractSubquery(subquery)
}

func (extractor *sensitiveFieldExtractor) plsqlExtractSubquery(ctx plsql.ISubqueryContext) ([]fieldInfo, error) {
	subqueryBasicElements := ctx.Subquery_basic_elements()
	if subqueryBasicElements == nil {
		return nil, nil
	}

	// TODO(rebelice): handle SET OPERATORS

	return extractor.plsqlExtractSubqueryBasicElements(subqueryBasicElements)
}

func (extractor *sensitiveFieldExtractor) plsqlExtractSubqueryBasicElements(ctx plsql.ISubquery_basic_elementsContext) ([]fieldInfo, error) {
	if ctx.Query_block() != nil {
		return extractor.plsqlExtractQueryBlock(ctx.Query_block())
	}

	if ctx.Subquery() != nil {
		return extractor.plsqlExtractSubquery(ctx.Subquery())
	}

	return nil, nil
}

func (extractor *sensitiveFieldExtractor) plsqlExtractQueryBlock(ctx plsql.IQuery_blockContext) ([]fieldInfo, error) {
	var fromFieldList []fieldInfo
	var err error
	fromClause := ctx.From_clause()
	if fromClause != nil {
		fromFieldList, err = extractor.plsqlExtractFromClause(fromClause)
		if err != nil {
			return nil, err
		}
		extractor.fromFieldList = fromFieldList
	}
	defer func() {
		extractor.fromFieldList = nil
	}()

	var result []fieldInfo

	// Extract selected fields
	selectedList := ctx.Selected_list()
	if selectedList != nil {
		if selectedList.ASTERISK() != nil {
			return fromFieldList, nil
		}

		selectListElements := selectedList.AllSelect_list_elements()
		for _, element := range selectListElements {
			if element.ASTERISK() != nil {
				schemaName, tableName := normalizeTableViewName(extractor.currentDatabase, element.Tableview_name())
				for _, field := range fromFieldList {
					if schemaName == field.database && field.table == tableName {
						result = append(result, field)
					}
				}
			} else {
				fieldName, sensitive, err := extractor.plsqlIsSensitiveExpression(element.Expression())
				if err != nil {
					return nil, err
				}
				if element.Column_alias() != nil {
					fieldName = normalizeColumnAlias(element.Column_alias())
				} else if fieldName == "" {
					fieldName = element.Expression().GetText()
				}
				result = append(result, fieldInfo{
					database:  extractor.currentDatabase,
					name:      fieldName,
					sensitive: sensitive,
				})
			}
		}
	}

	return result, nil
}

func (extractor *sensitiveFieldExtractor) plsqlCheckFieldSensitive(schemaName string, tableName string, columnName string) bool {
	// One sub-query may have multi-outer schemas and the multi-outer schemas can use the same name, such as:
	//
	//  select (
	//    select (
	//      select max(a) > x1.a from t
	//    )
	//    from t1 as x1
	//    limit 1
	//  )
	//  from t as x1;
	//
	// This query has two tables can be called `x1`, and the expression x1.a uses the closer x1 table.
	// This is the reason we loop the slice in reversed order.
	for i := len(extractor.outerSchemaInfo) - 1; i >= 0; i-- {
		field := extractor.outerSchemaInfo[i]
		sameSchema := (schemaName == field.database)
		sameTable := (tableName == field.table || tableName == "")
		sameColumn := (columnName == field.name)
		if sameSchema && sameTable && sameColumn {
			return field.sensitive
		}
	}

	for _, field := range extractor.fromFieldList {
		sameSchema := (schemaName == field.database)
		sameTable := (tableName == field.table || tableName == "")
		sameColumn := (columnName == field.name)
		if sameSchema && sameTable && sameColumn {
			return field.sensitive
		}

	}

	return false
}

func (extractor *sensitiveFieldExtractor) plsqlIsSensitiveExpression(ctx antlr.ParserRuleContext) (string, bool, error) {
	if ctx == nil {
		return "", false, nil
	}

	switch rule := ctx.(type) {
	case plsql.IColumn_nameContext:
		schemaName, tableName, columnName, err := plsqlNormalizeColumnName(extractor.currentDatabase, rule)
		if err != nil {
			return "", false, err
		}
		return columnName, extractor.plsqlCheckFieldSensitive(schemaName, tableName, columnName), nil
	case plsql.IIdentifierContext:
		id := parser.PLSQLNormalizeIdentifierContext(rule)
		return id, extractor.plsqlCheckFieldSensitive("", "", id), nil
	case plsql.IConstantContext:
		list := rule.AllQuoted_string()
		if len(list) == 1 && rule.DATE() == nil && rule.TIMESTAMP() == nil && rule.INTERVAL() == nil {
			// This case may be a column name...
			return extractor.plsqlIsSensitiveExpression(list[0])
		}
	case plsql.IQuoted_stringContext:
		if rule.Variable_name() != nil {
			return extractor.plsqlIsSensitiveExpression(rule.Variable_name())
		}
		return "", false, nil
	case plsql.IVariable_nameContext:
		if rule.Bind_variable() != nil {
			// TODO: handle bind variable
			return "", false, nil
		}
		var list []string
		for _, item := range rule.AllId_expression() {
			list = append(list, parser.PLSQLNormalizeIDExpression(item))
		}
		switch len(list) {
		case 1:
			return list[0], extractor.plsqlCheckFieldSensitive(extractor.currentDatabase, "", list[0]), nil
		case 2:
			return list[1], extractor.plsqlCheckFieldSensitive(extractor.currentDatabase, list[0], list[1]), nil
		case 3:
			return list[2], extractor.plsqlCheckFieldSensitive(list[0], list[1], list[2]), nil
		default:
			return "", false, nil
		}
	case plsql.IGeneral_elementContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllGeneral_element_part() {
			list = append(list, item)
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IGeneral_element_partContext:
		var list []string
		for _, item := range rule.AllId_expression() {
			list = append(list, parser.PLSQLNormalizeIDExpression(item))
		}
		switch len(list) {
		case 1:
			return list[0], extractor.plsqlCheckFieldSensitive(extractor.currentDatabase, "", list[0]), nil
		case 2:
			return list[1], extractor.plsqlCheckFieldSensitive(extractor.currentDatabase, list[0], list[1]), nil
		case 3:
			return list[2], extractor.plsqlCheckFieldSensitive(list[0], list[1], list[2]), nil
		default:
			return "", false, nil
		}
	case plsql.IExpressionContext:
		if rule.Logical_expression() != nil {
			return extractor.plsqlIsSensitiveExpression(rule.Logical_expression())
		}

		return extractor.plsqlIsSensitiveExpression(rule.Cursor_expression())
	case plsql.ICursor_expressionContext:
		return extractor.plsqlIsSensitiveExpression(rule.Subquery())
	case plsql.IQuery_blockContext:
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &sensitiveFieldExtractor{
			currentDatabase: extractor.currentDatabase,
			schemaInfo:      extractor.schemaInfo,
			outerSchemaInfo: append(extractor.outerSchemaInfo, extractor.fromFieldList...),
		}
		fieldList, err := subqueryExtractor.plsqlExtractQueryBlock(rule)
		if err != nil {
			return "", false, err
		}
		for _, field := range fieldList {
			if field.sensitive {
				return "", true, nil
			}
		}
		return "", false, nil
	case plsql.ISubqueryContext:
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &sensitiveFieldExtractor{
			currentDatabase: extractor.currentDatabase,
			schemaInfo:      extractor.schemaInfo,
			outerSchemaInfo: append(extractor.outerSchemaInfo, extractor.fromFieldList...),
		}
		fieldList, err := subqueryExtractor.plsqlExtractSubquery(rule)
		if err != nil {
			return "", false, err
		}
		for _, field := range fieldList {
			if field.sensitive {
				return "", true, nil
			}
		}
		return "", false, nil
	case plsql.ILogical_expressionContext:
		if rule.Unary_logical_expression() != nil {
			return extractor.plsqlIsSensitiveExpression(rule.Unary_logical_expression())
		}
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllLogical_expression() {
			list = append(list, item)
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IUnary_logical_expressionContext:
		return extractor.plsqlIsSensitiveExpression(rule.Multiset_expression())
	case plsql.IMultiset_expressionContext:
		var list []antlr.ParserRuleContext
		if rule.Relational_expression() != nil {
			list = append(list, rule.Relational_expression())
		}
		if rule.Concatenation() != nil {
			list = append(list, rule.Concatenation())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IRelational_expressionContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllRelational_expression() {
			list = append(list, item)
		}
		if rule.Compound_expression() != nil {
			list = append(list, rule.Compound_expression())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.ICompound_expressionContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		if rule.In_elements() != nil {
			list = append(list, rule.In_elements())
		}
		if rule.Between_elements() != nil {
			list = append(list, rule.Between_elements())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IIn_elementsContext:
		var list []antlr.ParserRuleContext
		if rule.Subquery() != nil {
			list = append(list, rule.Subquery())
		}
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IBetween_elementsContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IConcatenationContext:
		var list []antlr.ParserRuleContext
		if rule.Model_expression() != nil {
			list = append(list, rule.Model_expression())
		}
		if rule.Interval_expression() != nil {
			list = append(list, rule.Interval_expression())
		}
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IModel_expressionContext:
		var list []antlr.ParserRuleContext
		if rule.Unary_expression() != nil {
			list = append(list, rule.Unary_expression())
		}
		if rule.Model_expression_element() != nil {
			list = append(list, rule.Model_expression_element())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IInterval_expressionContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IUnary_expressionContext:
		var list []antlr.ParserRuleContext
		if rule.Unary_expression() != nil {
			list = append(list, rule.Unary_expression())
		}
		if rule.Case_statement() != nil {
			list = append(list, rule.Case_statement())
		}
		if rule.Quantified_expression() != nil {
			list = append(list, rule.Quantified_expression())
		}
		if rule.Standard_function() != nil {
			list = append(list, rule.Standard_function())
		}
		if rule.Atom() != nil {
			list = append(list, rule.Atom())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.ICase_statementContext:
		var list []antlr.ParserRuleContext
		if rule.Simple_case_statement() != nil {
			list = append(list, rule.Simple_case_statement())
		}
		if rule.Searched_case_statement() != nil {
			list = append(list, rule.Searched_case_statement())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.ISimple_case_statementContext:
		var list []antlr.ParserRuleContext
		if rule.Expression() != nil {
			list = append(list, rule.Expression())
		}
		for _, item := range rule.AllSimple_case_when_part() {
			list = append(list, item)
		}
		if rule.Case_else_part() != nil {
			list = append(list, rule.Case_else_part())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.ISimple_case_when_partContext:
		// not handle seq_of_statements
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.ICase_else_partContext:
		// not handle seq_of_statements
		return extractor.plsqlExistSensitiveExpression([]antlr.ParserRuleContext{rule.Expression()})
	case plsql.ISearched_case_statementContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllSearched_case_when_part() {
			list = append(list, item)
		}
		if rule.Case_else_part() != nil {
			list = append(list, rule.Case_else_part())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.ISearched_case_when_partContext:
		// not handle seq_of_statements
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IQuantified_expressionContext:
		var list []antlr.ParserRuleContext
		if rule.Expression() != nil {
			list = append(list, rule.Expression())
		}
		if rule.Select_only_statement() != nil {
			list = append(list, rule.Select_only_statement())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.ISelect_only_statementContext:
		// TODO(rebelice): handle CTE
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &sensitiveFieldExtractor{
			currentDatabase: extractor.currentDatabase,
			schemaInfo:      extractor.schemaInfo,
			outerSchemaInfo: append(extractor.outerSchemaInfo, extractor.fromFieldList...),
		}
		fieldList, err := subqueryExtractor.plsqlExtractSubquery(rule.Subquery())
		if err != nil {
			return "", false, err
		}
		for _, field := range fieldList {
			if field.sensitive {
				return "", true, nil
			}
		}
		return "", false, nil
	case plsql.IStandard_functionContext:
		var list []antlr.ParserRuleContext
		if rule.String_function() != nil {
			list = append(list, rule.String_function())
		}
		if rule.Numeric_function_wrapper() != nil {
			list = append(list, rule.Numeric_function_wrapper())
		}
		if rule.Json_function() != nil {
			list = append(list, rule.Json_function())
		}
		if rule.Other_function() != nil {
			list = append(list, rule.Other_function())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IString_functionContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		// TODO(rebelice): handle table_element
		if rule.Table_element() != nil {
			list = append(list, rule.Table_element())
		}
		if rule.Standard_function() != nil {
			list = append(list, rule.Standard_function())
		}
		if rule.Concatenation() != nil {
			list = append(list, rule.Concatenation())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.INumeric_function_wrapperContext:
		var list []antlr.ParserRuleContext
		if rule.Numeric_function() != nil {
			list = append(list, rule.Numeric_function())
		}
		if rule.Single_column_for_loop() != nil {
			list = append(list, rule.Single_column_for_loop())
		}
		if rule.Multi_column_for_loop() != nil {
			list = append(list, rule.Multi_column_for_loop())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.INumeric_functionContext:
		var list []antlr.ParserRuleContext
		if rule.Expression() != nil {
			list = append(list, rule.Expression())
		}
		if rule.Expressions() != nil {
			list = append(list, rule.Expressions())
		}
		if rule.Concatenation() != nil {
			list = append(list, rule.Concatenation())
		}
		// TODO(rebelice): handle over_clause
		_, sensitive, err := extractor.plsqlExistSensitiveExpression(list)
		return "", sensitive, err
	case plsql.IExpressionsContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.ISingle_column_for_loopContext:
		var list []antlr.ParserRuleContext
		if rule.Column_name() != nil {
			list = append(list, rule.Column_name())
		}
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IMulti_column_for_loopContext:
		var list []antlr.ParserRuleContext
		if rule.Paren_column_list() != nil {
			list = append(list, rule.Paren_column_list())
		}
		if rule.Subquery() != nil {
			list = append(list, rule.Subquery())
		}
		if rule.Expressions() != nil {
			list = append(list, rule.Expressions())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IJson_functionContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllJson_array_element() {
			list = append(list, item)
		}
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		if rule.Json_object_content() != nil {
			list = append(list, rule.Json_object_content())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IJson_array_elementContext:
		var list []antlr.ParserRuleContext
		if rule.Expression() != nil {
			list = append(list, rule.Expression())
		}
		if rule.Json_function() != nil {
			list = append(list, rule.Json_function())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IJson_object_contentContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllJson_object_entry() {
			list = append(list, item)
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IJson_object_entryContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		if rule.Identifier() != nil {
			list = append(list, rule.Identifier())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IOther_functionContext:
		var list []antlr.ParserRuleContext
		if rule.Function_argument_analytic() != nil {
			list = append(list, rule.Function_argument_analytic())
		}
		if rule.Function_argument_modeling() != nil {
			list = append(list, rule.Function_argument_modeling())
		}
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		if rule.Subquery() != nil {
			list = append(list, rule.Subquery())
		}
		if rule.Table_element() != nil {
			list = append(list, rule.Table_element())
		}
		if rule.Function_argument() != nil {
			list = append(list, rule.Function_argument())
		}
		if rule.Argument() != nil {
			list = append(list, rule.Argument())
		}
		if rule.Expressions() != nil {
			list = append(list, rule.Expressions())
		}
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		// TODO: handle xmltable
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IFunction_argument_analyticContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllArgument() {
			list = append(list, item)
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IArgumentContext:
		return extractor.plsqlIsSensitiveExpression(rule.Expression())
	case plsql.IFunction_argument_modelingContext:
		// TODO(rebelice): implement standard function with USING
		return "", false, nil
	case plsql.ITable_elementContext:
		// handled as column name
		var str []string
		for _, item := range rule.AllId_expression() {
			str = append(str, parser.PLSQLNormalizeIDExpression(item))
		}
		switch len(str) {
		case 1:
			return str[0], extractor.plsqlCheckFieldSensitive(extractor.currentDatabase, "", str[0]), nil
		case 2:
			return str[1], extractor.plsqlCheckFieldSensitive(extractor.currentDatabase, str[0], str[1]), nil
		case 3:
			return str[2], extractor.plsqlCheckFieldSensitive(str[0], str[1], str[2]), nil
		default:
			return "", false, nil
		}
	case plsql.IFunction_argumentContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllArgument() {
			list = append(list, item)
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IAtomContext:
		var list []antlr.ParserRuleContext
		if rule.Table_element() != nil {
			list = append(list, rule.Table_element())
		}
		if rule.Subquery() != nil {
			list = append(list, rule.Subquery())
		}
		for _, item := range rule.AllSubquery_operation_part() {
			list = append(list, item)
		}
		if rule.Expressions() != nil {
			list = append(list, rule.Expressions())
		}
		if rule.Constant() != nil {
			list = append(list, rule.Constant())
		}
		if rule.General_element() != nil {
			list = append(list, rule.General_element())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.ISubquery_operation_partContext:
		return extractor.plsqlIsSensitiveExpression(rule.Subquery_basic_elements())
	case plsql.ISubquery_basic_elementsContext:
		var list []antlr.ParserRuleContext
		if rule.Query_block() != nil {
			list = append(list, rule.Query_block())
		}
		if rule.Subquery() != nil {
			list = append(list, rule.Subquery())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	case plsql.IModel_expression_elementContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		for _, item := range rule.AllSingle_column_for_loop() {
			list = append(list, item)
		}
		if rule.Multi_column_for_loop() != nil {
			list = append(list, rule.Multi_column_for_loop())
		}
		return extractor.plsqlExistSensitiveExpression(list)
	}

	return "", false, nil
}

func (extractor *sensitiveFieldExtractor) plsqlExistSensitiveExpression(list []antlr.ParserRuleContext) (string, bool, error) {
	var fieldName string
	var sensitive bool
	var err error
	for _, ctx := range list {
		fieldName, sensitive, err = extractor.plsqlIsSensitiveExpression(ctx)
		if err != nil {
			return "", false, err
		}
		if len(list) != 1 {
			fieldName = ""
		}
		if sensitive {
			return fieldName, true, nil
		}
	}
	return fieldName, false, nil
}

func (extractor *sensitiveFieldExtractor) plsqlExtractFromClause(ctx plsql.IFrom_clauseContext) ([]fieldInfo, error) {
	tableReferenceList := ctx.Table_ref_list()
	if tableReferenceList == nil {
		return nil, nil
	}

	var result []fieldInfo
	tableRefs := tableReferenceList.AllTable_ref()
	for _, tableRef := range tableRefs {
		list, err := extractor.plsqlExtractTableRef(tableRef)
		if err != nil {
			return nil, err
		}
		result = append(result, list...)
	}

	return result, nil
}

func (extractor *sensitiveFieldExtractor) plsqlExtractTableRef(ctx plsql.ITable_refContext) ([]fieldInfo, error) {
	// TODO(rebelice): handle JOIN

	tableRefAux := ctx.Table_ref_aux()
	if tableRefAux == nil {
		return nil, nil
	}

	return extractor.plsqlExtractTableRefAux(tableRefAux)
}

func (extractor *sensitiveFieldExtractor) plsqlExtractTableRefAux(ctx plsql.ITable_ref_auxContext) ([]fieldInfo, error) {
	tableRefAuxInternal := ctx.Table_ref_aux_internal()

	list, err := extractor.plsqlExtractTableRefAuxInternal(tableRefAuxInternal)
	if err != nil {
		return nil, err
	}

	tableAlias := ctx.Table_alias()
	if tableAlias == nil {
		return list, nil
	}

	alias := normalizeTableAlias(tableAlias)

	var result []fieldInfo
	for _, field := range list {
		result = append(result, fieldInfo{
			database:  field.database,
			table:     alias,
			name:      field.name,
			sensitive: field.sensitive,
		})
	}

	return result, nil
}

func (extractor *sensitiveFieldExtractor) plsqlExtractTableRefAuxInternal(ctx plsql.ITable_ref_aux_internalContext) ([]fieldInfo, error) {
	switch rule := ctx.(type) {
	case *plsql.Table_ref_aux_internal_oneContext:
		return extractor.plsqlExtractDmlTableExpressionClause(rule.Dml_table_expression_clause())
	case *plsql.Table_ref_aux_internal_twoContext:
		// TODO(rebelice): handle subquery_operation_part
		return extractor.plsqlExtractTableRef(rule.Table_ref())
	case *plsql.Table_ref_aux_internal_threeContext:
		return extractor.plsqlExtractDmlTableExpressionClause(rule.Dml_table_expression_clause())
	default:
		return nil, errors.Errorf("unknown table_ref_aux_internal rule: %T", rule)
	}
}

func (extractor *sensitiveFieldExtractor) plsqlExtractDmlTableExpressionClause(ctx plsql.IDml_table_expression_clauseContext) ([]fieldInfo, error) {
	tableViewName := ctx.Tableview_name()
	if tableViewName != nil {
		schema, table := normalizeTableViewName(extractor.currentDatabase, tableViewName)
		tableSchema, err := extractor.plsqlFindTableSchema(schema, table)
		if err != nil {
			return nil, err
		}

		var result []fieldInfo
		for _, column := range tableSchema.ColumnList {
			result = append(result, fieldInfo{
				database:  schema,
				table:     table,
				name:      column.Name,
				sensitive: column.Sensitive,
			})
		}
		return result, nil
	}

	if ctx.Select_statement() != nil {
		return extractor.plsqlExtractSelect(ctx.Select_statement())
	}

	// TODO(rebelice): handle other cases for DML_TABLE_EXPRESSION_CLAUSE
	return nil, errors.Errorf("unknown DML_TABLE_EXPRESSION_CLAUSE rule: %T", ctx)
}

func (extractor *sensitiveFieldExtractor) plsqlFindTableSchema(schemaName, tableName string) (db.TableSchema, error) {
	if tableName == "DUAL" {
		return db.TableSchema{
			Name:       "DUAL",
			ColumnList: []db.ColumnInfo{},
		}, nil
	}
	// TODO(rebelice): handle CTE tables

	for _, schema := range extractor.schemaInfo.DatabaseList {
		if schema.Name != schemaName {
			continue
		}
		for _, table := range schema.TableList {
			if table.Name == tableName {
				return table, nil
			}
		}
	}

	return db.TableSchema{}, errors.Errorf("table %s.%s not found", schemaName, tableName)
}

func plsqlNormalizeColumnName(currentSchema string, ctx plsql.IColumn_nameContext) (string, string, string, error) {
	var buf []string
	buf = append(buf, parser.PLSQLNormalizeIdentifierContext(ctx.Identifier()))
	for _, idExpression := range ctx.AllId_expression() {
		buf = append(buf, parser.PLSQLNormalizeIDExpression(idExpression))
	}
	switch len(buf) {
	case 1:
		return currentSchema, "", buf[0], nil
	case 2:
		return currentSchema, buf[0], buf[1], nil
	case 3:
		return buf[0], buf[1], buf[2], nil
	default:
		return "", "", "", errors.Errorf("invalid column name: %s", ctx.GetText())
	}
}

func normalizeColumnAlias(ctx plsql.IColumn_aliasContext) string {
	if ctx == nil {
		return ""
	}

	if ctx.Identifier() != nil {
		return parser.PLSQLNormalizeIdentifierContext(ctx.Identifier())
	}

	if ctx.Quoted_string() != nil {
		return ctx.Quoted_string().GetText()
	}

	return ""
}

func normalizeTableAlias(ctx plsql.ITable_aliasContext) string {
	if ctx == nil {
		return ""
	}

	if ctx.Identifier() != nil {
		return parser.PLSQLNormalizeIdentifierContext(ctx.Identifier())
	}

	if ctx.Quoted_string() != nil {
		return ctx.Quoted_string().GetText()
	}

	return ""
}

// normalizeTableViewName normalizes the table name and schema name.
// Return empty string if it's xml table.
func normalizeTableViewName(currentSchema string, ctx plsql.ITableview_nameContext) (string, string) {
	if ctx.Identifier() == nil {
		return "", ""
	}

	identifier := parser.PLSQLNormalizeIdentifierContext(ctx.Identifier())

	if ctx.Id_expression() == nil {
		return currentSchema, identifier
	}

	idExpression := parser.PLSQLNormalizeIDExpression(ctx.Id_expression())

	return identifier, idExpression
}
