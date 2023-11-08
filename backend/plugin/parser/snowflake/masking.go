package snowflake

import (
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/snowsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterGetMaskedFieldsFunc(storepb.Engine_SNOWFLAKE, GetMaskedFields)
}

func GetMaskedFields(statement, currentDatabase string, schemaInfo *base.SensitiveSchemaInfo) ([]base.SensitiveField, error) {
	extractor := &fieldExtractor{
		currentDatabase: currentDatabase,
		schemaInfo:      schemaInfo,
	}
	result, err := extractor.extractSensitiveFields(statement)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type fieldExtractor struct {
	// For Oracle, we need to know the current database to determine if the table is in the current schema.
	currentDatabase    string
	schemaInfo         *base.SensitiveSchemaInfo
	cteOuterSchemaInfo []base.TableSchema

	// SELECT statement specific field.
	fromFieldList []base.FieldInfo
}

func (extractor *fieldExtractor) extractSensitiveFields(sql string) ([]base.SensitiveField, error) {
	result, err := ParseSnowSQL(sql)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse snowsql")
	}
	if result == nil {
		return nil, nil
	}

	listener := &snowsqlSnowSensitiveFieldExtractorListener{
		extractor: extractor,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, result.Tree)

	return listener.result, listener.err
}

type snowsqlSnowSensitiveFieldExtractorListener struct {
	*parser.BaseSnowflakeParserListener

	extractor *fieldExtractor
	result    []base.SensitiveField
	err       error
}

func (l *snowsqlSnowSensitiveFieldExtractorListener) EnterDml_command(ctx *parser.Dml_commandContext) {
	if l.err != nil {
		return
	}

	result, err := l.extractor.extractSnowsqlSensitiveFieldsQueryStatement(ctx.Query_statement())
	if err != nil {
		l.err = err
		return
	}
	for _, field := range result {
		l.result = append(l.result, base.SensitiveField{
			Name:              field.Name,
			MaskingAttributes: field.MaskingAttributes,
		})
	}
}

func (extractor *fieldExtractor) extractSnowsqlSensitiveFieldsQueryStatement(ctx parser.IQuery_statementContext) ([]base.FieldInfo, error) {
	if ctx.With_expression() != nil {
		allCommandTableExpression := ctx.With_expression().AllCommon_table_expression()

		for _, commandTableExpression := range allCommandTableExpression {
			var result []base.FieldInfo
			var err error
			normalizedCTEName := NormalizeSnowSQLObjectNamePart(commandTableExpression.Id_())

			if commandTableExpression.RECURSIVE() != nil || commandTableExpression.UNION() != nil {
				// TODO(zp): refactor code
				fieldsInAnchorClause, err := extractor.extractSnowsqlSensitiveFieldsQueryStatement(commandTableExpression.Anchor_clause().Query_statement())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields of the anchor clause of recursive CTE %q near line %d", normalizedCTEName, commandTableExpression.GetStart().GetLine())
				}

				tempCTEOuterSchemaInfo := base.TableSchema{
					Name: normalizedCTEName,
				}
				for i := 0; i < len(fieldsInAnchorClause); i++ {
					tempCTEOuterSchemaInfo.ColumnList = append(tempCTEOuterSchemaInfo.ColumnList, base.ColumnInfo{
						Name:              fieldsInAnchorClause[i].Name,
						MaskingAttributes: fieldsInAnchorClause[i].MaskingAttributes,
					})
					result = append(result, fieldsInAnchorClause[i])
				}
				originalSize := len(extractor.cteOuterSchemaInfo)
				extractor.cteOuterSchemaInfo = append(extractor.cteOuterSchemaInfo, tempCTEOuterSchemaInfo)
				for {
					change := false
					fieldsInRecursiveClause, err := extractor.extractSnowsqlSensitiveFieldsQueryStatement(commandTableExpression.Recursive_clause().Query_statement())
					if err != nil {
						return nil, errors.Wrapf(err, "failed to extract sensitive fields of the recursive clause of recursive CTE %q near line %d", normalizedCTEName, commandTableExpression.Recursive_clause().GetStart().GetLine())
					}
					if len(fieldsInRecursiveClause) != len(tempCTEOuterSchemaInfo.ColumnList) {
						return nil, errors.Wrapf(err, "recursive clause returns %d fields, but anchor clause returns %d fields in recursive CTE %q near line %d", len(fieldsInRecursiveClause), len(tempCTEOuterSchemaInfo.ColumnList), normalizedCTEName, commandTableExpression.GetStart().GetLine())
					}
					extractor.cteOuterSchemaInfo = extractor.cteOuterSchemaInfo[:originalSize]
					for i := 0; i < len(fieldsInRecursiveClause); i++ {
						change = change || tempCTEOuterSchemaInfo.ColumnList[i].MaskingAttributes.TransmittedBy(fieldsInRecursiveClause[i].MaskingAttributes)
						result[i].MaskingAttributes = tempCTEOuterSchemaInfo.ColumnList[i].MaskingAttributes.Clone()
					}
					if !change {
						break
					}
					originalSize = len(extractor.cteOuterSchemaInfo)
					extractor.cteOuterSchemaInfo = append(extractor.cteOuterSchemaInfo, tempCTEOuterSchemaInfo)
				}
				extractor.cteOuterSchemaInfo = extractor.cteOuterSchemaInfo[:originalSize]
			} else {
				result, err = extractor.extractSnowsqlSensitiveFieldsQueryStatement(commandTableExpression.Query_statement())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields of the CTE %q near line %d", normalizedCTEName, commandTableExpression.GetStart().GetLine())
				}
			}

			if commandTableExpression.Column_list() != nil {
				if len(result) != len(commandTableExpression.Column_list().AllColumn_name()) {
					return nil, errors.Wrapf(err, "the length of the column list in cte is %d, but body returns %d fields", len(commandTableExpression.Column_list().AllColumn_name()), len(result))
				}
				for i, columnName := range commandTableExpression.Column_list().AllColumn_name() {
					normalizedColumnName := NormalizeSnowSQLObjectNamePart(columnName.Id_())
					result[i].Name = normalizedColumnName
				}
			}
			// Append to the extractor.schemaInfo.DatabaseList
			columnList := make([]base.ColumnInfo, 0, len(result))
			for _, field := range result {
				columnList = append(columnList, base.ColumnInfo{
					Name:              field.Name,
					MaskingAttributes: field.MaskingAttributes,
				})
			}
			extractor.cteOuterSchemaInfo = append(extractor.cteOuterSchemaInfo, base.TableSchema{
				Name:       normalizedCTEName,
				ColumnList: columnList,
			})
		}
	}

	selectStatement := ctx.Select_statement()
	result, err := extractor.extractSnowsqlSensitiveFieldsSelectStatement(selectStatement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields of the query statement near line %d", selectStatement.GetStart().GetLine())
	}

	allSetOperators := ctx.AllSet_operators()
	for i, setOperator := range allSetOperators {
		// For UNION operator, the number of the columns in the result set is the same, and will use the left part's column name.
		// So we only need to extract the sensitive fields of the right part.
		right, err := extractor.extractSnowsqlSensitiveFieldSetOperator(setOperator)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract the %d set operator near line %d", i+1, setOperator.GetStart().GetLine())
		}
		if len(result) != len(right) {
			return nil, errors.Wrapf(err, "the number of columns in the query statement nearly line %d returns %d fields, but %d set operator near line %d returns %d fields", selectStatement.GetStart().GetLine(), len(result), i+1, setOperator.GetStart().GetLine(), len(right))
		}
		for i := range right {
			finalAttributes := result[i].MaskingAttributes
			finalAttributes.TransmittedBy(right[i].MaskingAttributes)
			result[i].MaskingAttributes = finalAttributes
		}
	}
	return result, nil
}

func (extractor *fieldExtractor) extractSnowsqlSensitiveFieldSetOperator(ctx parser.ISet_operatorsContext) ([]base.FieldInfo, error) {
	return extractor.extractSnowsqlSensitiveFieldsSelectStatement(ctx.Select_statement())
}

func (extractor *fieldExtractor) extractSnowsqlSensitiveFieldsSelectStatement(ctx parser.ISelect_statementContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	if ctx.Select_optional_clauses().From_clause() != nil {
		fromFieldList, err := extractor.extractSnowsqlSensitiveFieldsFromClause(ctx.Select_optional_clauses().From_clause())
		if err != nil {
			return nil, err
		}
		originalFromFieldsLength := len(extractor.fromFieldList)
		extractor.fromFieldList = append(extractor.fromFieldList, fromFieldList...)
		defer func() {
			extractor.fromFieldList = extractor.fromFieldList[:originalFromFieldsLength]
		}()
	}

	var result []base.FieldInfo

	var selectList parser.ISelect_listContext
	if ctx.Select_clause() != nil {
		selectList = ctx.Select_clause().Select_list_no_top().Select_list()
	} else if ctx.Select_top_clause() != nil {
		selectList = ctx.Select_clause().Select_list_no_top().Select_list()
	}
	for _, iSelectListElem := range selectList.AllSelect_list_elem() {
		if columnElem := iSelectListElem.Column_elem(); columnElem != nil {
			var normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName string
			if v := columnElem.Alias(); v != nil {
				normalizedTableName = NormalizeSnowSQLObjectNamePart(v.Id_())
			} else if v := columnElem.Object_name(); v != nil {
				normalizedDatabaseName, normalizedSchemaName, normalizedTableName = normalizedObjectName(v, "", "")
			}
			if columnElem.STAR() != nil {
				left, err := extractor.getAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields of the query statement near line %d", ctx.GetStart().GetLine())
				}
				result = append(result, left...)
			} else if columnElem.Column_name() != nil {
				normalizedColumnName = NormalizeSnowSQLObjectNamePart(columnElem.Column_name().Id_())
				left, err := extractor.snowflakeGetField(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to check whether the column %q is sensitive near line %d", normalizedColumnName, columnElem.Column_name().GetStart().GetLine())
				}
				result = append(result, left)
			} else if columnElem.DOLLAR() != nil {
				columnPosition, err := strconv.Atoi(columnElem.Column_position().Num().GetText())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse column position %q to integer near line %d", columnElem.Column_position().Num().GetText(), columnElem.Column_position().Num().GetStart().GetLine())
				}
				if columnPosition < 1 {
					return nil, errors.Wrapf(err, "column position %d is invalid because it is less than 1 near line %d", columnPosition, columnElem.Column_position().Num().GetStart().GetLine())
				}
				left, err := extractor.getAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields of the query statement near line %d", ctx.GetStart().GetLine())
				}
				if columnPosition > len(left) {
					return nil, errors.Wrapf(err, "column position is invalid because want to try get the %d column near line %d, but FROM clause only returns %d columns for %q.%q.%q", columnPosition, columnElem.Column_position().Num().GetStart().GetLine(), len(left), normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
				}
				result = append(result, left[columnPosition-1])
			}
			if asAlias := columnElem.As_alias(); asAlias != nil {
				result[len(result)-1].Name = NormalizeSnowSQLObjectNamePart(asAlias.Alias().Id_())
			}
		} else if expressionElem := iSelectListElem.Expression_elem(); expressionElem != nil {
			if v := expressionElem.Expr(); v != nil {
				columnName, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
				}
				result = append(result, base.FieldInfo{
					Name:              columnName,
					MaskingAttributes: maskingAttributes,
				})
			} else if v := expressionElem.Predicate(); v != nil {
				columnName, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
				}
				result = append(result, base.FieldInfo{
					Name:              columnName,
					MaskingAttributes: maskingAttributes,
				})
			}

			if asAlias := expressionElem.As_alias(); asAlias != nil {
				result[len(result)-1].Name = NormalizeSnowSQLObjectNamePart(asAlias.Alias().Id_())
			}
		}
	}

	return result, nil
}

// The closure of the IExprContext.
func (extractor *fieldExtractor) evalSnowSQLExprMaskingAttributes(ctx antlr.RuleContext) (string, base.MaskingAttributes, error) {
	switch ctx := ctx.(type) {
	case *parser.ExprContext:
		if v := ctx.Primitive_expression(); v != nil {
			return extractor.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Function_call(); v != nil {
			return extractor.evalSnowSQLExprMaskingAttributes(v)
		}

		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expr := range ctx.AllExpr() {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(expr)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if v := ctx.Subquery(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}

		if v := ctx.Case_expression(); v != nil {
			return extractor.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Iff_expr(); v != nil {
			return extractor.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Full_column_name(); v != nil {
			return extractor.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Bracket_expression(); v != nil {
			return extractor.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Arr_literal(); v != nil {
			return extractor.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Json_literal(); v != nil {
			return extractor.evalSnowSQLExprMaskingAttributes(v)
		}

		if v := ctx.Try_cast_expr(); v != nil {
			return extractor.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Object_name(); v != nil {
			return extractor.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Trim_expression(); v != nil {
			return extractor.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Expr_list(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Full_column_nameContext:
		normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName := normalizedFullColumnName(ctx)
		fieldInfo, err := extractor.snowflakeGetField(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the column %q is sensitive near line %d", normalizedColumnName, ctx.GetStart().GetLine())
		}
		return fieldInfo.Name, fieldInfo.MaskingAttributes, nil
	case *parser.Object_nameContext:
		normalizedDatabaseName, normalizedSchemaName, normalizedTableName := normalizedObjectName(ctx, extractor.currentDatabase, "PUBLIC")
		fieldInfo, err := extractor.snowflakeGetField(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, "")
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the object %q is sensitive near line %d", normalizedTableName, ctx.GetStart().GetLine())
		}
		return fieldInfo.Name, fieldInfo.MaskingAttributes, nil
	case *parser.Trim_expressionContext:
		if v := ctx.Expr(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		panic("never reach here")
	case *parser.Try_cast_exprContext:
		if v := ctx.Expr(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		panic("never reach here")
	case *parser.Json_literalContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if v := ctx.AllKv_pair(); len(v) > 0 {
			for _, kvPair := range v {
				_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(kvPair)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", kvPair.GetText(), kvPair.GetStart().GetLine())
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Kv_pairContext:
		if v := ctx.Value(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		panic("never reach here")
	case *parser.Arr_literalContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if v := ctx.AllValue(); len(v) > 0 {
			for _, value := range v {
				_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(value)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", value.GetText(), value.GetStart().GetLine())
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.ValueContext:
		return extractor.evalSnowSQLExprMaskingAttributes(ctx.Expr())
	case *parser.Bracket_expressionContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if v := ctx.Expr(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if v := ctx.Subquery(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			return ctx.GetText(), finalAttributes, nil
		}
		panic("never reach here")
	case *parser.Iff_exprContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if v := ctx.Search_condition(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(ctx.Search_condition())
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
			for _, expr := range ctx.AllExpr() {
				_, finalAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(expr)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
			return ctx.GetText(), finalAttributes, nil
		}
		panic("never reach here")
	case *parser.Case_expressionContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expr := range ctx.AllExpr() {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(expr)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if v := ctx.AllSwitch_section(); len(v) > 0 {
			for _, switchSection := range v {
				_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(switchSection)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", switchSection.GetText(), switchSection.GetStart().GetLine())
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
			return ctx.GetText(), finalAttributes, nil
		}
		if v := ctx.AllSwitch_search_condition_section(); len(v) > 0 {
			for _, switchSearchConditionSection := range v {
				_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(switchSearchConditionSection)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", switchSearchConditionSection.GetText(), switchSearchConditionSection.GetStart().GetLine())
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
			return ctx.GetText(), finalAttributes, nil
		}
		panic("never reach here")
	case *parser.Switch_sectionContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if v := ctx.AllExpr(); len(v) > 0 {
			for _, expr := range v {
				_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(expr)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
			return ctx.GetText(), finalAttributes, nil
		}
		panic("never reach here")
	case *parser.Switch_search_condition_sectionContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if v := ctx.Search_condition(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
			_, maskingAttributes, err = extractor.evalSnowSQLExprMaskingAttributes(ctx.Expr())
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Expr().GetText(), ctx.Expr().GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			return ctx.GetText(), finalAttributes, nil
		}
		panic("never reach here")
	case *parser.Search_conditionContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if v := ctx.Predicate(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Predicate().GetText(), ctx.Predicate().GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
		}
		if v := ctx.AllSearch_condition(); len(v) > 0 {
			for _, searchCondition := range v {
				_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(searchCondition)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", searchCondition.GetText(), searchCondition.GetStart().GetLine())
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
			return ctx.GetText(), finalAttributes, nil
		}
		panic("never reach here")
	case *parser.PredicateContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if v := ctx.AllExpr(); len(v) > 0 {
			for _, expr := range v {
				_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(expr)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		if v := ctx.Subquery(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
		}
		if v := ctx.Expr_list(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.SubqueryContext:
		fields, err := extractor.extractSnowsqlSensitiveFieldsQueryStatement(ctx.Query_statement())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.GetText(), ctx.GetStart().GetLine())
		}
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, field := range fields {
			finalAttributes.TransmittedByInExpression(field.MaskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Primitive_expressionContext:
		if v := ctx.Id_(); v != nil {
			return extractor.evalSnowSQLExprMaskingAttributes(v)
		}
		return ctx.GetText(), base.NewDefaultMaskingAttributes(), nil
	case *parser.Function_callContext:
		if v := ctx.Ranking_windowed_function(); v != nil {
			return extractor.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Aggregate_function(); v != nil {
			return extractor.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Object_name(); v != nil {
			return v.GetText(), base.NewDefaultMaskingAttributes(), nil
		}
		if v := ctx.Expr_list(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		if v := ctx.Expr(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		panic("never reach here")
	case *parser.Aggregate_functionContext:
		if v := ctx.Expr_list(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		if ctx.STAR() != nil {
			return ctx.GetText(), base.NewDefaultMaskingAttributes(), nil
		}
		if v := ctx.Expr(); v != nil {
			finalAttributes := base.NewDefaultMaskingAttributes()
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
			_, maskingAttributes, err = extractor.evalSnowSQLExprMaskingAttributes(ctx.Order_by_clause())
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Order_by_clause().GetText(), ctx.Order_by_clause().GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			return ctx.GetText(), finalAttributes, nil
		}
		panic("never reach here")
	case *parser.Ranking_windowed_functionContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if v := ctx.Expr(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if v := ctx.Over_clause(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			return ctx.GetText(), finalAttributes, nil
		}
		panic("never reach here")
	case *parser.Over_clauseContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if v := ctx.Partition_by(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if v := ctx.Order_by_expr(); v != nil {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			return ctx.GetText(), finalAttributes, nil
		}
		panic("never reach here")
	case *parser.Partition_byContext:
		_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(ctx.Expr_list())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Expr_list().GetText(), ctx.Expr_list().GetStart().GetLine())
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.Order_by_exprContext:
		_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(ctx.Expr_list_sorted())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Expr_list_sorted().GetText(), ctx.Expr_list_sorted().GetStart().GetLine())
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.Expr_listContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		allExpr := ctx.AllExpr()
		for _, expr := range allExpr {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(expr)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Expr_list_sortedContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		allExpr := ctx.AllExpr()
		for _, expr := range allExpr {
			_, maskingAttributes, err := extractor.evalSnowSQLExprMaskingAttributes(expr)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Id_Context:
		normalizedColumnName := NormalizeSnowSQLObjectNamePart(ctx)
		fieldInfo, err := extractor.snowflakeGetField("", "", "", normalizedColumnName)
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check whether the column %q is sensitive near line %d", normalizedColumnName, ctx.GetStart().GetLine())
		}
		return fieldInfo.Name, fieldInfo.MaskingAttributes, nil
	}
	panic("never reach here")
}

func (extractor *fieldExtractor) extractSnowsqlSensitiveFieldsFromClause(ctx parser.IFrom_clauseContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	return extractor.extractSnowsqlSensitiveFieldsTableSources(ctx.Table_sources())
}

func (extractor *fieldExtractor) extractSnowsqlSensitiveFieldsTableSources(ctx parser.ITable_sourcesContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}
	allTableSources := ctx.AllTable_source()
	var result []base.FieldInfo
	// If there are multiple table sources, the default join type is CROSS JOIN.
	for _, tableSource := range allTableSources {
		tableSourceResult, err := extractor.extractSnowsqlSensitiveFieldsTableSource(tableSource)
		if err != nil {
			return nil, err
		}
		result = append(result, tableSourceResult...)
	}
	return result, nil
}

func (extractor *fieldExtractor) extractSnowsqlSensitiveFieldsTableSource(ctx parser.ITable_sourceContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}
	return extractor.extractSnowsqlSensitiveFieldsTableSourceItemJoined(ctx.Table_source_item_joined())
}

func (extractor *fieldExtractor) extractSnowsqlSensitiveFieldsTableSourceItemJoined(ctx parser.ITable_source_item_joinedContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var left []base.FieldInfo
	var err error
	if ctx.Object_ref() != nil {
		left, err = extractor.extractSnowsqlSensitiveFieldsObjectRef(ctx.Object_ref())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields of the left part of the object ref near line %d", ctx.Object_ref().GetStart().GetLine())
		}
	}

	if ctx.Table_source_item_joined() != nil {
		left, err = extractor.extractSnowsqlSensitiveFieldsTableSourceItemJoined(ctx.Table_source_item_joined())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields of the left part of the table source item joined near line %d", ctx.Table_source_item_joined().GetStart().GetLine())
		}
	}

	for i, joinClause := range ctx.AllJoin_clause() {
		left, err = extractor.extractSnowsqlSensitiveFieldsJoinClause(joinClause, left)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields of the left part of the #%d join clause near line %d", i+1, joinClause.GetStart().GetLine())
		}
	}

	return left, nil
}

func (extractor *fieldExtractor) extractSnowsqlSensitiveFieldsJoinClause(ctx parser.IJoin_clauseContext, left []base.FieldInfo) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	// Snowflake has 6 types of join:
	// INNER JOIN, LEFT OUTER JOIN, RIGHT OUTER JOIN, FULL OUTER JOIN, CROSS JOIN, and NATURAL JOIN.
	// Only the result(column num) of NATURAL JOIN may be reduced.
	right, err := extractor.extractSnowsqlSensitiveFieldsObjectRef(ctx.Object_ref())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields of the right part of the JOIN near line %d", ctx.Object_ref().GetStart().GetLine())
	}
	if ctx.NATURAL() != nil {
		// We should remove all the duplicate columns in the result set.
		// For example, if the left part has columns [a, b, c], and the right part has columns [a, b, d],
		// then the result set of NATURAL JOIN should be [a, b, c, d].
		leftMap := make(map[string]int)
		for i, field := range left {
			leftMap[field.Name] = i
		}

		var result []base.FieldInfo
		result = append(result, left...)
		for _, field := range right {
			if idx, ok := leftMap[field.Name]; !ok {
				result = append(result, field)
			} else {
				result[idx].MaskingAttributes.TransmittedBy(field.MaskingAttributes)
			}
		}
		return result, nil
	}

	// For other types of join, we should keep all the columns for the left part and the right part.
	var result []base.FieldInfo
	result = append(result, left...)
	result = append(result, right...)
	return result, nil
}

func (extractor *fieldExtractor) extractSnowsqlSensitiveFieldsObjectRef(ctx parser.IObject_refContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var result []base.FieldInfo

	if objectName := ctx.Object_name(); objectName != nil {
		normalizedDatabaseName, tableSchema, err := extractor.snowsqlFindTableSchema(objectName, extractor.currentDatabase, "PUBLIC")
		if err != nil {
			return nil, err
		}
		for _, column := range tableSchema.ColumnList {
			result = append(result, base.FieldInfo{
				Database:          normalizedDatabaseName,
				Table:             tableSchema.Name,
				Name:              column.Name,
				MaskingAttributes: column.MaskingAttributes,
			})
		}
	}

	// TODO(zp): Handle the value clause.
	if ctx.Values() != nil {
		return nil, nil
	}

	// TODO(zp): In data-warehouse, define a function to return multiple rows is widespread, we should parse the
	// function definition to extract the sensitive fields.
	if ctx.TABLE() != nil {
		return nil, nil
	}

	if ctx.Subquery() != nil {
		// TODO(zp): handle recursive cte.
		subqueryResult, err := extractor.extractSnowsqlSensitiveFieldsQueryStatement(ctx.Subquery().Query_statement())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields of subquery near line %d", ctx.Subquery().GetStart().GetLine())
		}
		result = append(result, subqueryResult...)
	}

	// TODO(zp): Handle the flatten table.
	if ctx.Flatten_table() != nil {
		return nil, nil
	}

	if ctx.Pivot_unpivot() != nil {
		if v := ctx.Pivot_unpivot(); v.PIVOT() != nil {
			pivotColumnName := v.AllId_()[1]
			normalizedPivotColumnName := NormalizeSnowSQLObjectNamePart(pivotColumnName)
			pivotColumnIndex := -1
			for i, field := range result {
				if field.Name == normalizedPivotColumnName {
					pivotColumnIndex = i
					break
				}
			}
			if pivotColumnIndex == -1 {
				return nil, errors.Errorf(`pivot column %s is not found from field list %+v`, normalizedPivotColumnName, result)
			}
			pivotColumnInOriginalResult := result[pivotColumnIndex]
			result = append(result[:pivotColumnIndex], result[pivotColumnIndex+1:]...)

			valueColumnName := v.AllId_()[2]
			normalizedValueColumnName := NormalizeSnowSQLObjectNamePart(valueColumnName)
			valueColumnIndex := -1
			for i, field := range result {
				if field.Name == normalizedValueColumnName {
					valueColumnIndex = i
					break
				}
			}
			if valueColumnIndex == -1 {
				return nil, errors.Errorf(`value column %s is not found from field list %+v`, normalizedValueColumnName, result)
			}
			result = append(result[:valueColumnIndex], result[valueColumnIndex+1:]...)

			for _, literal := range v.AllLiteral() {
				result = append(result, base.FieldInfo{
					Name:              literal.GetText(),
					MaskingAttributes: pivotColumnInOriginalResult.MaskingAttributes,
				})
			}
		} else if v := ctx.Pivot_unpivot(); v.UNPIVOT() != nil {
			var strippedColumnIndices []int
			var strippedColumnInOriginalResult []base.FieldInfo
			for idx, columnName := range v.Column_list().AllColumn_name() {
				normalizedColumnName := NormalizeSnowSQLObjectNamePart(columnName.Id_())
				for i, field := range result {
					if field.Name == normalizedColumnName {
						strippedColumnIndices = append(strippedColumnIndices, i)
						strippedColumnInOriginalResult = append(strippedColumnInOriginalResult, field)
						break
					}
				}
				if len(strippedColumnIndices) != idx+1 {
					return nil, errors.Errorf(`column %s is not found from field list %+v`, normalizedColumnName, result)
				}
				result = append(result[:strippedColumnIndices[idx]], result[strippedColumnIndices[idx]+1:]...)
			}

			finalAttributes := base.NewDefaultMaskingAttributes()
			for _, field := range strippedColumnInOriginalResult {
				finalAttributes.TransmittedBy(field.MaskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					break
				}
			}

			valueColumnName := v.Id_(0)
			normalizedValueColumnName := NormalizeSnowSQLObjectNamePart(valueColumnName)

			nameColumnName := v.Column_name().Id_()
			normalizedNameColumnName := NormalizeSnowSQLObjectNamePart(nameColumnName)

			result = append(result, base.FieldInfo{
				Name:              normalizedNameColumnName,
				MaskingAttributes: base.NewDefaultMaskingAttributes(),
			}, base.FieldInfo{
				Name:              normalizedValueColumnName,
				MaskingAttributes: finalAttributes,
			})
		}
	}

	// If the as alias is not nil, we should use the alias name to replace the original table name.
	if ctx.As_alias() != nil {
		id := ctx.As_alias().Alias().Id_()
		aliasName := NormalizeSnowSQLObjectNamePart(id)
		for i := 0; i < len(result); i++ {
			result[i].Table = aliasName
			// We can safely set the database and schema to empty string because the
			// user cannot use the original table name to access the column.
			// For example, the following query is illegal:
			// SELECT T1.A FROM T1 AS T2;
			result[i].Schema = ""
			result[i].Database = ""
		}
	}

	return result, nil
}

func (extractor *fieldExtractor) snowsqlFindTableSchema(objectName parser.IObject_nameContext, normalizedFallbackDatabaseName, normalizedFallbackSchemaName string) (string, base.TableSchema, error) {
	normalizedDatabaseName, normalizedSchemaName, normalizedTableName := normalizedObjectName(objectName, "", "")
	// For snowflake, we should find the table schema in cteOuterSchemaInfo by ascending order.
	if normalizedDatabaseName == "" && normalizedSchemaName == "" {
		for _, tableSchema := range extractor.cteOuterSchemaInfo {
			if normalizedTableName == tableSchema.Name {
				return normalizedDatabaseName, tableSchema, nil
			}
		}
	}
	normalizedDatabaseName, normalizedSchemaName, normalizedTableName = normalizedObjectName(objectName, normalizedFallbackDatabaseName, normalizedFallbackSchemaName)
	for _, databaseSchema := range extractor.schemaInfo.DatabaseList {
		if normalizedDatabaseName != "" && normalizedDatabaseName != databaseSchema.Name {
			continue
		}
		for _, schemaSchema := range databaseSchema.SchemaList {
			if normalizedSchemaName != "" && normalizedSchemaName != schemaSchema.Name {
				continue
			}
			for _, tableSchema := range schemaSchema.TableList {
				if normalizedTableName != tableSchema.Name {
					continue
				}
				return normalizedDatabaseName, tableSchema, nil
			}
		}
	}
	return "", base.TableSchema{}, errors.Errorf(`table %s.%s.%s is not found`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
}

func (extractor *fieldExtractor) getAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName string) ([]base.FieldInfo, error) {
	type maskType = uint8
	const (
		maskNone         maskType = 0
		maskDatabaseName maskType = 1 << iota
		maskSchemaName
		maskTableName
	)
	mask := maskNone
	if normalizedTableName != "" {
		mask |= maskTableName
	}
	if normalizedSchemaName != "" {
		if mask&maskTableName == 0 {
			return nil, errors.Errorf(`table name %s is specified without column name`, normalizedTableName)
		}
		mask |= maskSchemaName
	}
	if normalizedDatabaseName != "" {
		if mask&maskSchemaName == 0 {
			return nil, errors.Errorf(`database name %s is specified without schema name`, normalizedDatabaseName)
		}
		mask |= maskDatabaseName
	}

	var result []base.FieldInfo
	for _, field := range extractor.fromFieldList {
		if mask&maskDatabaseName != 0 && normalizedDatabaseName != field.Database {
			continue
		}
		if mask&maskSchemaName != 0 && normalizedSchemaName != field.Schema {
			continue
		}
		if mask&maskTableName != 0 && normalizedTableName != field.Table {
			continue
		}
		result = append(result, field)
	}
	return result, nil
}

// snowflakeGetField iterates through the fromFieldList sequentially until we find the first matching object and return the column name, and returns the fieldInfo.
func (extractor *fieldExtractor) snowflakeGetField(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName string) (base.FieldInfo, error) {
	type maskType = uint8
	const (
		maskNone         maskType = 0
		maskDatabaseName maskType = 1 << iota
		maskSchemaName
		maskTableName
		maskColumnName
	)
	mask := maskNone
	if normalizedColumnName != "" {
		mask |= maskColumnName
	}
	if normalizedTableName != "" {
		if mask&maskColumnName == 0 {
			return base.FieldInfo{}, errors.Errorf(`table name %s is specified without column name`, normalizedTableName)
		}
		mask |= maskTableName
	}
	if normalizedSchemaName != "" {
		if mask&maskTableName == 0 {
			return base.FieldInfo{}, errors.Errorf(`schema name %s is specified without table name`, normalizedSchemaName)
		}
		mask |= maskSchemaName
	}
	if normalizedDatabaseName != "" {
		if mask&maskSchemaName == 0 {
			return base.FieldInfo{}, errors.Errorf(`database name %s is specified without schema name`, normalizedDatabaseName)
		}
		mask |= maskDatabaseName
	}

	if mask == maskNone {
		return base.FieldInfo{}, errors.Errorf(`no object name is specified`)
	}

	// We just need to iterate through the fromFieldList sequentially until we find the first matching object.

	// It is safe if there are two or more objects in the fromFieldList have the same column name, because the executor
	// will throw a compilation error if the column name is ambiguous.
	// For example, there are two tables T1 and T2, and both of them have a column named "C1". The following query will throw
	// a compilation error:
	// SELECT C1 FROM T1, T2;
	//
	// But users can specify the table name to avoid the compilation error:
	// SELECT T1.C1 FROM T1, T2;
	//
	// Further more, users can not use the original table name if they specify the alias name:
	// SELECT T1.C1 FROM T1 AS T3, T2; -- invalid identifier 'ADDRESS.ID'

	for _, field := range extractor.fromFieldList {
		if mask&maskDatabaseName != 0 && normalizedDatabaseName != field.Database {
			continue
		}
		if mask&maskSchemaName != 0 && normalizedSchemaName != field.Schema {
			continue
		}
		if mask&maskTableName != 0 && normalizedTableName != field.Table {
			continue
		}
		if mask&maskColumnName != 0 && normalizedColumnName != field.Name {
			continue
		}
		return field, nil
	}
	return base.FieldInfo{}, errors.Errorf(`no matching column %q.%q.%q.%q`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
}

func normalizedFullColumnName(ctx parser.IFull_column_nameContext) (normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName string) {
	if ctx.GetDb_name() != nil {
		normalizedDatabaseName = NormalizeSnowSQLObjectNamePart(ctx.GetDb_name())
	}
	if ctx.GetSchema() != nil {
		normalizedSchemaName = NormalizeSnowSQLObjectNamePart(ctx.GetSchema())
	}
	if ctx.GetTab_name() != nil {
		normalizedTableName = NormalizeSnowSQLObjectNamePart(ctx.GetTab_name())
	}
	if ctx.GetCol_name() != nil {
		normalizedColumnName = NormalizeSnowSQLObjectNamePart(ctx.GetCol_name())
	}
	return normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName
}

func normalizedObjectName(objectName parser.IObject_nameContext, normalizedFallbackDatabaseName, normalizedFallbackSchemaName string) (string, string, string) {
	// TODO(zp): unify here with NormalizeObjectName in backend/plugin/parser/sql/snowsql.go
	var parts []string
	if objectName == nil {
		return "", "", ""
	}
	database := normalizedFallbackDatabaseName
	if d := objectName.GetD(); d != nil {
		normalizedD := NormalizeSnowSQLObjectNamePart(d)
		if normalizedD != "" {
			database = normalizedD
		}
	}
	parts = append(parts, database)

	schema := normalizedFallbackSchemaName
	if s := objectName.GetS(); s != nil {
		normalizedS := NormalizeSnowSQLObjectNamePart(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}
	parts = append(parts, schema)

	normalizedO := NormalizeSnowSQLObjectNamePart(objectName.GetO())
	parts = append(parts, normalizedO)

	return parts[0], parts[1], parts[2]
}
