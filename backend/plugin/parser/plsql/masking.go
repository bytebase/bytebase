package plsql

import (
	"github.com/antlr4-go/antlr/v4"
	plsql "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGetMaskedFieldsFunc(storepb.Engine_ORACLE, GetMaskedFields)
	base.RegisterGetMaskedFieldsFunc(storepb.Engine_DM, GetMaskedFields)
	base.RegisterGetMaskedFieldsFunc(storepb.Engine_OCEANBASE_ORACLE, GetMaskedFields)
}

func GetMaskedFields(statement, currentDatabase string, schemaInfo *base.SensitiveSchemaInfo) ([]base.SensitiveField, error) {
	extractor := &fieldExtractor{
		currentDatabase: currentDatabase,
		schemaInfo:      schemaInfo,
	}
	result, err := extractor.extractSensitiveField(statement)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// fieldExtractor is the extractor for plsql.
type fieldExtractor struct {
	// For Oracle, we need to know the current database to determine if the table is in the current schema.
	currentDatabase    string
	schemaInfo         *base.SensitiveSchemaInfo
	outerSchemaInfo    []base.FieldInfo
	cteOuterSchemaInfo []base.TableSchema

	// SELECT statement specific field.
	fromFieldList []base.FieldInfo
}

func (extractor *fieldExtractor) extractSensitiveField(statement string) ([]base.SensitiveField, error) {
	tree, _, err := ParsePLSQL(statement)
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

	extractor *fieldExtractor
	result    []base.SensitiveField
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
				l.result = append(l.result, base.SensitiveField{
					Name:              field.Name,
					MaskingAttributes: field.MaskingAttributes,
				})
			}
		}
	}
}

func (extractor *fieldExtractor) plsqlExtractContext(ctx antlr.ParserRuleContext) ([]base.FieldInfo, error) {
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

func (extractor *fieldExtractor) plsqlExtractFactoringElement(ctx plsql.IFactoring_elementContext) (base.TableSchema, error) {
	// Deal with recursive CTE first.
	tableName := NormalizeIdentifierContext(ctx.Query_name().Identifier())

	if yes, lastPart := extractor.plsqlIsRecursiveCTE(ctx); yes {
		subquery := ctx.Subquery()
		initialField, err := extractor.plsqlExtractSubqueryExceptLastPart(subquery)
		if err != nil {
			return base.TableSchema{}, err
		}

		if ctx.Paren_column_list() != nil {
			var columnNameList []string
			for _, column := range ctx.Paren_column_list().Column_list().AllColumn_name() {
				_, _, columnName, err := plsqlNormalizeColumnName("", column)
				if err != nil {
					return base.TableSchema{}, err
				}
				columnNameList = append(columnNameList, columnName)
			}
			if len(columnNameList) != len(initialField) {
				return base.TableSchema{}, errors.Errorf("column list and subquery must have the same number of columns")
			}
			for i, columnName := range columnNameList {
				initialField[i].Name = columnName
			}
		}

		cteInfo := base.TableSchema{
			Name:       tableName,
			ColumnList: []base.ColumnInfo{},
		}
		for _, field := range initialField {
			cteInfo.ColumnList = append(cteInfo.ColumnList, base.ColumnInfo{
				Name:              field.Name,
				MaskingAttributes: field.MaskingAttributes,
			})
		}

		// Compute dependent closures.
		// There are two ways to compute dependent closures:
		//   1. find the all dependent edges, then use graph theory traversal to find the closure.
		//   2. Iterate to simulate the CTE recursive process, each turn check whether the Sensitive state has changed, and stop if no change.
		//
		// Consider the option 2 can easy to implementation, because the simulate process has been written.
		// On the other hand, the number of iterations of the entire algorithm will not exceed the length of fields.
		// In actual use, the length of fields will not be more than 20 generally.
		// So I think it's OK for now.
		// If any performance issues in use, optimize here.
		extractor.cteOuterSchemaInfo = append(extractor.cteOuterSchemaInfo, cteInfo)
		defer func() {
			extractor.cteOuterSchemaInfo = extractor.cteOuterSchemaInfo[:len(extractor.cteOuterSchemaInfo)-1]
		}()
		for {
			fieldList, err := extractor.plsqlExtractSubqueryBasicElements(lastPart.Subquery_basic_elements())
			if err != nil {
				return base.TableSchema{}, err
			}
			if len(fieldList) != len(cteInfo.ColumnList) {
				return base.TableSchema{}, errors.Errorf("recursive WITH clause members must have the same number of columns")
			}

			changed := false
			for i, field := range fieldList {
				changed = changed || cteInfo.ColumnList[i].MaskingAttributes.TransmittedBy(field.MaskingAttributes)
			}

			if !changed {
				break
			}
			extractor.cteOuterSchemaInfo[len(extractor.cteOuterSchemaInfo)-1] = cteInfo
		}
		return cteInfo, nil
	}

	return extractor.plsqlExtractNonRecursiveCTE(ctx)
}

func (*fieldExtractor) plsqlIsRecursiveCTE(ctx plsql.IFactoring_elementContext) (bool, plsql.ISubquery_operation_partContext) {
	subquery := ctx.Subquery()
	allParts := subquery.AllSubquery_operation_part()
	if len(allParts) == 0 {
		return false, nil
	}
	lastPart := allParts[len(allParts)-1]
	return lastPart.ALL() != nil, lastPart
}

func (extractor *fieldExtractor) plsqlExtractSubqueryExceptLastPart(ctx plsql.ISubqueryContext) ([]base.FieldInfo, error) {
	subqueryBasicElements := ctx.Subquery_basic_elements()
	if subqueryBasicElements == nil {
		return nil, nil
	}

	leftField, err := extractor.plsqlExtractSubqueryBasicElements(subqueryBasicElements)
	if err != nil {
		return nil, err
	}

	allParts := ctx.AllSubquery_operation_part()
	for _, part := range allParts[:len(allParts)-1] {
		leftField, err = extractor.plsqlExtractSubqueryOperationPart(part, leftField)
		if err != nil {
			return nil, err
		}
	}

	return leftField, nil
}

func (extractor *fieldExtractor) plsqlExtractNonRecursiveCTE(ctx plsql.IFactoring_elementContext) (base.TableSchema, error) {
	fieldList, err := extractor.plsqlExtractSubquery(ctx.Subquery())
	if err != nil {
		return base.TableSchema{}, err
	}

	if ctx.Paren_column_list() != nil {
		var columnNameList []string
		for _, column := range ctx.Paren_column_list().Column_list().AllColumn_name() {
			_, _, columnName, err := plsqlNormalizeColumnName("", column)
			if err != nil {
				return base.TableSchema{}, err
			}
			columnNameList = append(columnNameList, columnName)
		}
		if len(columnNameList) != len(fieldList) {
			return base.TableSchema{}, errors.Errorf("column list and subquery must have the same number of columns")
		}
		for i, columnName := range columnNameList {
			fieldList[i].Name = columnName
		}
	}

	tableName := NormalizeIdentifierContext(ctx.Query_name().Identifier())

	result := base.TableSchema{
		Name:       tableName,
		ColumnList: []base.ColumnInfo{},
	}
	for _, field := range fieldList {
		result.ColumnList = append(result.ColumnList, base.ColumnInfo{
			Name:              field.Name,
			MaskingAttributes: field.MaskingAttributes,
		})
	}
	return result, nil
}

func (extractor *fieldExtractor) plsqlExtractSelectOnlyStatement(ctx plsql.ISelect_only_statementContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	subquery := ctx.Subquery()
	if subquery == nil {
		return nil, nil
	}

	return extractor.plsqlExtractSubquery(subquery)
}

func (extractor *fieldExtractor) plsqlExtractSelect(ctx plsql.ISelect_statementContext) ([]base.FieldInfo, error) {
	selectOnlyStatement := ctx.Select_only_statement()
	if selectOnlyStatement == nil {
		return nil, nil
	}

	return extractor.plsqlExtractSelectOnlyStatement(selectOnlyStatement)
}

func (extractor *fieldExtractor) plsqlExtractSubquery(ctx plsql.ISubqueryContext) ([]base.FieldInfo, error) {
	subqueryBasicElements := ctx.Subquery_basic_elements()
	if subqueryBasicElements == nil {
		return nil, nil
	}

	leftField, err := extractor.plsqlExtractSubqueryBasicElements(subqueryBasicElements)
	if err != nil {
		return nil, err
	}

	for _, part := range ctx.AllSubquery_operation_part() {
		leftField, err = extractor.plsqlExtractSubqueryOperationPart(part, leftField)
		if err != nil {
			return nil, err
		}
	}

	return leftField, nil
}

func (extractor *fieldExtractor) plsqlExtractSubqueryOperationPart(ctx plsql.ISubquery_operation_partContext, leftField []base.FieldInfo) ([]base.FieldInfo, error) {
	rightField, err := extractor.plsqlExtractSubqueryBasicElements(ctx.Subquery_basic_elements())
	if err != nil {
		return nil, err
	}

	if len(leftField) != len(rightField) {
		return nil, errors.Errorf("each UNION/INTERSECT/EXCEPT query must have the same number of columns")
	}

	var result []base.FieldInfo
	for i, field := range rightField {
		finalAttributes := leftField[i].MaskingAttributes
		finalAttributes.TransmittedBy(field.MaskingAttributes)
		result = append(result, base.FieldInfo{
			Name:              leftField[i].Name,
			Table:             leftField[i].Table,
			Database:          leftField[i].Database,
			MaskingAttributes: finalAttributes,
		})
	}

	return result, nil
}

func (extractor *fieldExtractor) plsqlExtractSubqueryBasicElements(ctx plsql.ISubquery_basic_elementsContext) ([]base.FieldInfo, error) {
	if ctx.Query_block() != nil {
		return extractor.plsqlExtractQueryBlock(ctx.Query_block())
	}

	if ctx.Subquery() != nil {
		return extractor.plsqlExtractSubquery(ctx.Subquery())
	}

	return nil, nil
}

func (extractor *fieldExtractor) plsqlExtractQueryBlock(ctx plsql.IQuery_blockContext) (result []base.FieldInfo, err error) {
	withClause := ctx.Subquery_factoring_clause()
	if withClause != nil {
		cteOuterLength := len(extractor.cteOuterSchemaInfo)
		defer func() {
			extractor.cteOuterSchemaInfo = extractor.cteOuterSchemaInfo[:cteOuterLength]
		}()
		for _, cte := range withClause.AllFactoring_element() {
			cteTable, err := extractor.plsqlExtractFactoringElement(cte)
			if err != nil {
				return nil, err
			}
			extractor.cteOuterSchemaInfo = append(extractor.cteOuterSchemaInfo, cteTable)
		}
	}

	var fromFieldList []base.FieldInfo
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
					if schemaName == field.Database && field.Table == tableName {
						result = append(result, field)
					}
				}
			} else {
				fieldName, maskingLevel, err := extractor.plsqlEvalMaskingLevelInExpression(element.Expression())
				if err != nil {
					return nil, err
				}
				if element.Column_alias() != nil {
					fieldName = normalizeColumnAlias(element.Column_alias())
				} else if fieldName == "" {
					fieldName = element.Expression().GetText()
				}
				result = append(result, base.FieldInfo{
					Database:          extractor.currentDatabase,
					Name:              fieldName,
					MaskingAttributes: maskingLevel,
				})
			}
		}
	}

	return result, nil
}

func (extractor *fieldExtractor) plsqlCheckFieldMaskingLevel(schemaName string, tableName string, columnName string) base.MaskingAttributes {
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
		sameSchema := (schemaName == field.Database)
		sameTable := (tableName == field.Table || tableName == "")
		sameColumn := (columnName == field.Name)
		if sameSchema && sameTable && sameColumn {
			return field.MaskingAttributes
		}
	}

	for _, field := range extractor.fromFieldList {
		sameSchema := (schemaName == field.Database)
		sameTable := (tableName == field.Table || tableName == "")
		sameColumn := (columnName == field.Name)
		if sameSchema && sameTable && sameColumn {
			return field.MaskingAttributes
		}
	}

	return base.NewDefaultMaskingAttributes()
}

func (extractor *fieldExtractor) plsqlEvalMaskingLevelInExpression(ctx antlr.ParserRuleContext) (string, base.MaskingAttributes, error) {
	if ctx == nil {
		return "", base.NewDefaultMaskingAttributes(), nil
	}

	switch rule := ctx.(type) {
	case plsql.IColumn_nameContext:
		schemaName, tableName, columnName, err := plsqlNormalizeColumnName(extractor.currentDatabase, rule)
		if err != nil {
			return "", base.NewDefaultMaskingAttributes(), err
		}
		return columnName, extractor.plsqlCheckFieldMaskingLevel(schemaName, tableName, columnName), nil
	case plsql.IIdentifierContext:
		id := NormalizeIdentifierContext(rule)
		return id, extractor.plsqlCheckFieldMaskingLevel("", "", id), nil
	case plsql.IConstantContext:
		list := rule.AllQuoted_string()
		if len(list) == 1 && rule.DATE() == nil && rule.TIMESTAMP() == nil && rule.INTERVAL() == nil {
			// This case may be a column name...
			return extractor.plsqlEvalMaskingLevelInExpression(list[0])
		}
	case plsql.IQuoted_stringContext:
		if rule.Variable_name() != nil {
			return extractor.plsqlEvalMaskingLevelInExpression(rule.Variable_name())
		}
		return "", base.NewDefaultMaskingAttributes(), nil
	case plsql.IVariable_nameContext:
		if rule.Bind_variable() != nil {
			// TODO: handle bind variable
			return "", base.NewDefaultMaskingAttributes(), nil
		}
		var list []string
		for _, item := range rule.AllId_expression() {
			list = append(list, NormalizeIDExpression(item))
		}
		switch len(list) {
		case 1:
			return list[0], extractor.plsqlCheckFieldMaskingLevel(extractor.currentDatabase, "", list[0]), nil
		case 2:
			return list[1], extractor.plsqlCheckFieldMaskingLevel(extractor.currentDatabase, list[0], list[1]), nil
		case 3:
			return list[2], extractor.plsqlCheckFieldMaskingLevel(list[0], list[1], list[2]), nil
		default:
			return "", base.NewDefaultMaskingAttributes(), nil
		}
	case plsql.IGeneral_elementContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllGeneral_element_part() {
			list = append(list, item)
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.IGeneral_element_partContext:
		// This case is for functions, such as CONCAT(a, b)
		if rule.Function_argument() != nil {
			_, maskingLevel, err := extractor.plsqlEvalMaskingLevelInExpression(rule.Function_argument())
			return "", maskingLevel, err
		}

		// This case is for column names, such as root.a.b
		var list []string
		for _, item := range rule.AllId_expression() {
			list = append(list, NormalizeIDExpression(item))
		}
		switch len(list) {
		case 1:
			return list[0], extractor.plsqlCheckFieldMaskingLevel(extractor.currentDatabase, "", list[0]), nil
		case 2:
			return list[1], extractor.plsqlCheckFieldMaskingLevel(extractor.currentDatabase, list[0], list[1]), nil
		case 3:
			return list[2], extractor.plsqlCheckFieldMaskingLevel(list[0], list[1], list[2]), nil
		default:
			return "", base.NewDefaultMaskingAttributes(), nil
		}
	case plsql.IExpressionContext:
		if rule.Logical_expression() != nil {
			return extractor.plsqlEvalMaskingLevelInExpression(rule.Logical_expression())
		}

		return extractor.plsqlEvalMaskingLevelInExpression(rule.Cursor_expression())
	case plsql.ICursor_expressionContext:
		return extractor.plsqlEvalMaskingLevelInExpression(rule.Subquery())
	case plsql.IQuery_blockContext:
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &fieldExtractor{
			currentDatabase: extractor.currentDatabase,
			schemaInfo:      extractor.schemaInfo,
			outerSchemaInfo: append(extractor.outerSchemaInfo, extractor.fromFieldList...),
		}
		fieldList, err := subqueryExtractor.plsqlExtractQueryBlock(rule)
		if err != nil {
			return "", base.NewDefaultMaskingAttributes(), err
		}
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, field := range fieldList {
			finalAttributes.TransmittedBy(field.MaskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return "", finalAttributes, nil
			}
		}
		return "", finalAttributes, nil
	case plsql.ISubqueryContext:
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &fieldExtractor{
			currentDatabase: extractor.currentDatabase,
			schemaInfo:      extractor.schemaInfo,
			outerSchemaInfo: append(extractor.outerSchemaInfo, extractor.fromFieldList...),
		}
		fieldList, err := subqueryExtractor.plsqlExtractSubquery(rule)
		if err != nil {
			return "", base.NewDefaultMaskingAttributes(), err
		}
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, field := range fieldList {
			finalAttributes.TransmittedBy(field.MaskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return "", finalAttributes, nil
			}
		}
		return "", finalAttributes, nil
	case plsql.ILogical_expressionContext:
		if rule.Unary_logical_expression() != nil {
			return extractor.plsqlEvalMaskingLevelInExpression(rule.Unary_logical_expression())
		}
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllLogical_expression() {
			list = append(list, item)
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.IUnary_logical_expressionContext:
		return extractor.plsqlEvalMaskingLevelInExpression(rule.Multiset_expression())
	case plsql.IMultiset_expressionContext:
		var list []antlr.ParserRuleContext
		if rule.Relational_expression() != nil {
			list = append(list, rule.Relational_expression())
		}
		if rule.Concatenation() != nil {
			list = append(list, rule.Concatenation())
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.IRelational_expressionContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllRelational_expression() {
			list = append(list, item)
		}
		if rule.Compound_expression() != nil {
			list = append(list, rule.Compound_expression())
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
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
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.IIn_elementsContext:
		var list []antlr.ParserRuleContext
		if rule.Subquery() != nil {
			list = append(list, rule.Subquery())
		}
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.IBetween_elementsContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
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
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.IModel_expressionContext:
		var list []antlr.ParserRuleContext
		if rule.Unary_expression() != nil {
			list = append(list, rule.Unary_expression())
		}
		if rule.Model_expression_element() != nil {
			list = append(list, rule.Model_expression_element())
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.IInterval_expressionContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllConcatenation() {
			list = append(list, item)
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
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
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.ICase_statementContext:
		var list []antlr.ParserRuleContext
		if rule.Simple_case_statement() != nil {
			list = append(list, rule.Simple_case_statement())
		}
		if rule.Searched_case_statement() != nil {
			list = append(list, rule.Searched_case_statement())
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
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
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.ISimple_case_when_partContext:
		// not handle seq_of_statements
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.ICase_else_partContext:
		// not handle seq_of_statements
		return extractor.plsqlEvalMaskingLevelInExpressionList([]antlr.ParserRuleContext{rule.Expression()})
	case plsql.ISearched_case_statementContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllSearched_case_when_part() {
			list = append(list, item)
		}
		if rule.Case_else_part() != nil {
			list = append(list, rule.Case_else_part())
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.ISearched_case_when_partContext:
		// not handle seq_of_statements
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.IQuantified_expressionContext:
		var list []antlr.ParserRuleContext
		if rule.Expression() != nil {
			list = append(list, rule.Expression())
		}
		if rule.Select_only_statement() != nil {
			list = append(list, rule.Select_only_statement())
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.ISelect_only_statementContext:
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &fieldExtractor{
			currentDatabase: extractor.currentDatabase,
			schemaInfo:      extractor.schemaInfo,
			outerSchemaInfo: append(extractor.outerSchemaInfo, extractor.fromFieldList...),
		}
		fieldList, err := subqueryExtractor.plsqlExtractSelectOnlyStatement(rule)
		if err != nil {
			return "", base.NewDefaultMaskingAttributes(), err
		}
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, field := range fieldList {
			finalAttributes.TransmittedBy(field.MaskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return "", finalAttributes, nil
			}
		}
		return "", finalAttributes, nil
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
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.IString_functionContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		if rule.Table_element() != nil {
			list = append(list, rule.Table_element())
		}
		if rule.Standard_function() != nil {
			list = append(list, rule.Standard_function())
		}
		if rule.Concatenation() != nil {
			list = append(list, rule.Concatenation())
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
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
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
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
		_, sensitive, err := extractor.plsqlEvalMaskingLevelInExpressionList(list)
		return "", sensitive, err
	case plsql.IExpressionsContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.ISingle_column_for_loopContext:
		var list []antlr.ParserRuleContext
		if rule.Column_name() != nil {
			list = append(list, rule.Column_name())
		}
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
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
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
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
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.IJson_array_elementContext:
		var list []antlr.ParserRuleContext
		if rule.Expression() != nil {
			list = append(list, rule.Expression())
		}
		if rule.Json_function() != nil {
			list = append(list, rule.Json_function())
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.IJson_object_contentContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllJson_object_entry() {
			list = append(list, item)
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.IJson_object_entryContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllExpression() {
			list = append(list, item)
		}
		if rule.Identifier() != nil {
			list = append(list, rule.Identifier())
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
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
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.IFunction_argument_analyticContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllArgument() {
			list = append(list, item)
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.IArgumentContext:
		return extractor.plsqlEvalMaskingLevelInExpression(rule.Expression())
	case plsql.IFunction_argument_modelingContext:
		// TODO(rebelice): implement standard function with USING
		return "", base.NewDefaultMaskingAttributes(), nil
	case plsql.ITable_elementContext:
		// handled as column name
		var str []string
		for _, item := range rule.AllId_expression() {
			str = append(str, NormalizeIDExpression(item))
		}
		switch len(str) {
		case 1:
			return str[0], extractor.plsqlCheckFieldMaskingLevel(extractor.currentDatabase, "", str[0]), nil
		case 2:
			return str[1], extractor.plsqlCheckFieldMaskingLevel(extractor.currentDatabase, str[0], str[1]), nil
		case 3:
			return str[2], extractor.plsqlCheckFieldMaskingLevel(str[0], str[1], str[2]), nil
		default:
			return "", base.NewDefaultMaskingAttributes(), nil
		}
	case plsql.IFunction_argumentContext:
		var list []antlr.ParserRuleContext
		for _, item := range rule.AllArgument() {
			list = append(list, item)
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
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
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	case plsql.ISubquery_operation_partContext:
		return extractor.plsqlEvalMaskingLevelInExpression(rule.Subquery_basic_elements())
	case plsql.ISubquery_basic_elementsContext:
		var list []antlr.ParserRuleContext
		if rule.Query_block() != nil {
			list = append(list, rule.Query_block())
		}
		if rule.Subquery() != nil {
			list = append(list, rule.Subquery())
		}
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
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
		return extractor.plsqlEvalMaskingLevelInExpressionList(list)
	}

	return "", base.NewDefaultMaskingAttributes(), nil
}

func (extractor *fieldExtractor) plsqlEvalMaskingLevelInExpressionList(list []antlr.ParserRuleContext) (string, base.MaskingAttributes, error) {
	var fieldName string
	var err error
	var attributes base.MaskingAttributes
	finalAttributes := base.NewDefaultMaskingAttributes()
	for _, ctx := range list {
		fieldName, attributes, err = extractor.plsqlEvalMaskingLevelInExpression(ctx)
		if err != nil {
			return "", base.NewDefaultMaskingAttributes(), err
		}
		if len(list) != 1 {
			fieldName = ""
		}
		finalAttributes.TransmittedBy(attributes)
		if finalAttributes.IsNeverChangeInTransmission() {
			return fieldName, finalAttributes, nil
		}
	}
	return fieldName, finalAttributes, nil
}

func (extractor *fieldExtractor) plsqlExtractFromClause(ctx plsql.IFrom_clauseContext) ([]base.FieldInfo, error) {
	tableReferenceList := ctx.Table_ref_list()
	if tableReferenceList == nil {
		return nil, nil
	}

	var result []base.FieldInfo
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

func (extractor *fieldExtractor) plsqlExtractTableRef(ctx plsql.ITable_refContext) ([]base.FieldInfo, error) {
	tableRefAux := ctx.Table_ref_aux()
	if tableRefAux == nil {
		return nil, nil
	}

	leftField, err := extractor.plsqlExtractTableRefAux(tableRefAux)
	if err != nil {
		return nil, err
	}

	joins := ctx.AllJoin_clause()
	if len(joins) == 0 {
		return leftField, nil
	}

	for _, join := range joins {
		leftField, err = extractor.plsqlMergeJoin(leftField, join)
		if err != nil {
			return nil, err
		}
	}

	return leftField, nil
}

func (extractor *fieldExtractor) plsqlMergeJoin(leftField []base.FieldInfo, ctx plsql.IJoin_clauseContext) ([]base.FieldInfo, error) {
	rightField, err := extractor.plsqlExtractTableRefAux(ctx.Table_ref_aux())
	if err != nil {
		return nil, err
	}

	var result []base.FieldInfo
	leftFieldMap := make(map[string]base.FieldInfo)
	rightFieldMap := make(map[string]base.FieldInfo)
	for _, field := range leftField {
		leftFieldMap[field.Name] = field
	}
	for _, field := range rightField {
		rightFieldMap[field.Name] = field
	}

	if ctx.NATURAL() != nil {
		// Natural Join will merge the same column name field.
		for _, field := range leftField {
			if rField, exists := rightFieldMap[field.Name]; exists {
				finalAttributes := field.MaskingAttributes
				finalAttributes.TransmittedBy(rField.MaskingAttributes)
				result = append(result, base.FieldInfo{
					Database:          field.Database,
					Table:             field.Table,
					Name:              field.Name,
					MaskingAttributes: finalAttributes,
				})
			} else {
				result = append(result, field)
			}

			for _, field := range rightField {
				if _, exists := leftFieldMap[field.Name]; !exists {
					result = append(result, field)
				}
			}
		}
		return result, nil
	}

	// Why multi-USING part will be here?
	if len(ctx.AllJoin_using_part()) != 0 {
		usingMap := make(map[string]bool)
		for _, part := range ctx.AllJoin_using_part() {
			for _, column := range part.Paren_column_list().Column_list().AllColumn_name() {
				_, _, name, err := plsqlNormalizeColumnName(extractor.currentDatabase, column)
				if err != nil {
					return nil, err
				}
				usingMap[name] = true
			}
		}

		for _, field := range leftField {
			_, existsInUsingMap := usingMap[field.Name]
			rField, existsInRightFieldMap := rightFieldMap[field.Name]
			if existsInUsingMap && existsInRightFieldMap {
				finalAttributes := field.MaskingAttributes
				finalAttributes.TransmittedBy(rField.MaskingAttributes)
				result = append(result, base.FieldInfo{
					Database:          field.Database,
					Table:             field.Table,
					Name:              field.Name,
					MaskingAttributes: finalAttributes,
				})
			} else {
				result = append(result, field)
			}
		}

		for _, field := range rightField {
			_, existsInUsingMap := usingMap[field.Name]
			_, existsInLeftFieldMap := leftFieldMap[field.Name]
			if existsInUsingMap && existsInLeftFieldMap {
				continue
			}
			result = append(result, field)
		}
		return result, nil
	}

	result = append(result, leftField...)
	result = append(result, rightField...)
	return result, nil
}

func (extractor *fieldExtractor) plsqlExtractTableRefAux(ctx plsql.ITable_ref_auxContext) ([]base.FieldInfo, error) {
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

	var result []base.FieldInfo
	for _, field := range list {
		result = append(result, base.FieldInfo{
			Database:          field.Database,
			Table:             alias,
			Name:              field.Name,
			MaskingAttributes: field.MaskingAttributes,
		})
	}

	return result, nil
}

func (extractor *fieldExtractor) plsqlExtractTableRefAuxInternal(ctx plsql.ITable_ref_aux_internalContext) ([]base.FieldInfo, error) {
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

func (extractor *fieldExtractor) plsqlExtractDmlTableExpressionClause(ctx plsql.IDml_table_expression_clauseContext) ([]base.FieldInfo, error) {
	tableViewName := ctx.Tableview_name()
	if tableViewName != nil {
		schema, table := normalizeTableViewName(extractor.currentDatabase, tableViewName)
		tableSchema, err := extractor.plsqlFindTableSchema(schema, table)
		if err != nil {
			return nil, err
		}

		var result []base.FieldInfo
		for _, column := range tableSchema.ColumnList {
			result = append(result, base.FieldInfo{
				Database:          schema,
				Table:             table,
				Name:              column.Name,
				MaskingAttributes: column.MaskingAttributes,
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

func (extractor *fieldExtractor) plsqlFindTableSchema(schemaName, tableName string) (base.TableSchema, error) {
	if tableName == "DUAL" {
		return base.TableSchema{
			Name:       "DUAL",
			ColumnList: []base.ColumnInfo{},
		}, nil
	}

	// Each CTE name in one WITH clause must be unique, but we can use the same name in the different level CTE, such as:
	//
	//  with tt2 as (
	//    with tt2 as (select * from t)
	//    select max(a) from tt2)
	//  select * from tt2
	//
	// This query has two CTE can be called `tt2`, and the FROM clause 'from tt2' uses the closer tt2 CTE.
	// This is the reason we loop the slice in reversed order.
	for i := len(extractor.cteOuterSchemaInfo) - 1; i >= 0; i-- {
		table := extractor.cteOuterSchemaInfo[i]
		if table.Name == tableName && schemaName == extractor.currentDatabase {
			return table, nil
		}
	}

	for _, schema := range extractor.schemaInfo.DatabaseList {
		if schema.Name != schemaName {
			continue
		}
		if len(schema.SchemaList) == 0 {
			continue
		}
		tableList := schema.SchemaList[0].TableList
		for _, table := range tableList {
			if table.Name == tableName {
				return table, nil
			}
		}
	}

	return base.TableSchema{}, errors.Errorf("table %s.%s not found", schemaName, tableName)
}

func plsqlNormalizeColumnName(currentSchema string, ctx plsql.IColumn_nameContext) (string, string, string, error) {
	var buf []string
	buf = append(buf, NormalizeIdentifierContext(ctx.Identifier()))
	for _, idExpression := range ctx.AllId_expression() {
		buf = append(buf, NormalizeIDExpression(idExpression))
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
		return NormalizeIdentifierContext(ctx.Identifier())
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
		return NormalizeIdentifierContext(ctx.Identifier())
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

	identifier := NormalizeIdentifierContext(ctx.Identifier())

	if ctx.Id_expression() == nil {
		return currentSchema, identifier
	}

	idExpression := NormalizeIDExpression(ctx.Id_expression())

	return identifier, idExpression
}
