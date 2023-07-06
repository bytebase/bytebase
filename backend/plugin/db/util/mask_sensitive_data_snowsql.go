// Package util implements the util functions.
package util

import (
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	snowparser "github.com/bytebase/snowsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFields(sql string) ([]db.SensitiveField, error) {
	tree, err := parser.ParseSnowSQL(sql)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse snowsql")
	}
	if tree == nil {
		return nil, nil
	}

	listener := &snowsqlSensitiveFieldExtractorListener{
		extractor: extractor,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.result, listener.err
}

type snowsqlSensitiveFieldExtractorListener struct {
	*snowparser.BaseSnowflakeParserListener

	extractor *sensitiveFieldExtractor
	result    []db.SensitiveField
	err       error
}

func (l *snowsqlSensitiveFieldExtractorListener) EnterDml_command(ctx *snowparser.Dml_commandContext) {
	if l.err != nil {
		return
	}

	result, err := l.extractor.extractSnowsqlSensitiveFieldsQueryStatement(ctx.Query_statement())
	if err != nil {
		l.err = err
		return
	}
	for _, field := range result {
		l.result = append(l.result, db.SensitiveField{
			Name:      field.name,
			Sensitive: field.sensitive,
		})
	}
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsQueryStatement(ctx snowparser.IQuery_statementContext) ([]fieldInfo, error) {
	if ctx.With_expression() != nil {
		// TODO(zp): handle recursive CTE
		allCommandTableExpression := ctx.With_expression().AllCommon_table_expression()
		originalDatabaseSchema := extractor.schemaInfo.DatabaseList
		defer func() {
			extractor.schemaInfo.DatabaseList = originalDatabaseSchema
		}()

		for _, commandTableExpression := range allCommandTableExpression {
			normalizedCTEName := parser.NormalizeObjectNamePart(commandTableExpression.Id_())
			result, err := extractor.extractSnowsqlSensitiveFieldsSelectStatement(commandTableExpression.Select_statement())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract sensitive fields of the CTE %q near line %d", normalizedCTEName, commandTableExpression.GetStart().GetLine())
			}

			if commandTableExpression.Column_list() != nil {
				if len(result) != len(commandTableExpression.Column_list().AllColumn_name()) {
					return nil, errors.Wrapf(err, "the length of the column list in cte is %d, but body returns %d fields", len(commandTableExpression.Column_list().AllColumn_name()), len(result))
				}
				for i, columnName := range commandTableExpression.Column_list().AllColumn_name() {
					normalizedColumnName := parser.NormalizeObjectNamePart(columnName.Id_())
					result[i].name = normalizedColumnName
				}
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
					return nil, errors.Wrapf(err, "the number of columns in the select statement nearly line %d returns %d fields, but %d set operator near line %d returns %d fields", commandTableExpression.Select_statement().GetStart().GetLine(), len(result), i+1, setOperator.GetStart().GetLine(), len(right))
				}
				for i := range right {
					if !result[i].sensitive {
						result[i].sensitive = right[i].sensitive
					}
				}
			}
			// Append to the extractor.schemaInfo.DatabaseList
			columnList := make([]db.ColumnInfo, 0, len(result))
			for _, field := range result {
				columnList = append(columnList, db.ColumnInfo{
					Name:      field.name,
					Sensitive: field.sensitive,
				})
			}
			extractor.cteOuterSchemaInfo = append(extractor.cteOuterSchemaInfo, db.TableSchema{
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
			if !result[i].sensitive {
				result[i].sensitive = right[i].sensitive
			}
		}
	}
	return result, nil
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldSetOperator(ctx snowparser.ISet_operatorsContext) ([]fieldInfo, error) {
	return extractor.extractSnowsqlSensitiveFieldsSelectStatement(ctx.Select_statement())
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsSelectStatement(ctx snowparser.ISelect_statementContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var fromFieldList []fieldInfo
	var err error
	if ctx.Select_optional_clauses().From_clause() != nil {
		fromFieldList, err = extractor.extractSnowsqlSensitiveFieldsFromClause(ctx.Select_optional_clauses().From_clause())
		if err != nil {
			return nil, err
		}
		originalFromFields := extractor.fromFieldList
		extractor.fromFieldList = fromFieldList
		defer func() {
			extractor.fromFieldList = originalFromFields
		}()
	}

	var result []fieldInfo

	var selectList snowparser.ISelect_listContext
	if ctx.Select_clause() != nil {
		selectList = ctx.Select_clause().Select_list_no_top().Select_list()
	} else if ctx.Select_top_clause() != nil {
		selectList = ctx.Select_clause().Select_list_no_top().Select_list()
	}
	for _, iSelectListElem := range selectList.AllSelect_list_elem() {
		if columnElem := iSelectListElem.Column_elem(); columnElem != nil {
			var normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName string
			if v := columnElem.Alias(); v != nil {
				normalizedTableName = parser.NormalizeObjectNamePart(v.Id_())
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
				normalizedColumnName = parser.NormalizeObjectNamePart(columnElem.Column_name().Id_())
				left, err := extractor.snowflakeIsFieldSensitive(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
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
				result[len(result)-1].name = parser.NormalizeObjectNamePart(asAlias.Alias().Id_())
			}
		} else if expressionElem := iSelectListElem.Expression_elem(); expressionElem != nil {
			if v := expressionElem.Expr(); v != nil {
				columnName, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
				}
				result = append(result, fieldInfo{
					name:      columnName,
					sensitive: isSensitive,
				})
			} else if v := expressionElem.Predicate(); v != nil {
				columnName, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
				}
				result = append(result, fieldInfo{
					name:      columnName,
					sensitive: isSensitive,
				})
			}

			if asAlias := expressionElem.As_alias(); asAlias != nil {
				result[len(result)-1].name = parser.NormalizeObjectNamePart(asAlias.Alias().Id_())
			}
		}
	}

	return result, nil
}

// The closure of the IExprContext.
func (extractor *sensitiveFieldExtractor) isSnowSQLExprSensitive(ctx antlr.RuleContext) (string, bool, error) {
	switch ctx := ctx.(type) {
	case *snowparser.ExprContext:
		if v := ctx.Primitive_expression(); v != nil {
			return extractor.isSnowSQLExprSensitive(v)
		}
		if v := ctx.Function_call(); v != nil {
			return extractor.isSnowSQLExprSensitive(v)
		}

		for _, expr := range ctx.AllExpr() {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(expr)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
		}
		if v := ctx.Subquery(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
		}

		if v := ctx.Case_expression(); v != nil {
			return extractor.isSnowSQLExprSensitive(v)
		}
		if v := ctx.Iff_expr(); v != nil {
			return extractor.isSnowSQLExprSensitive(v)
		}
		if v := ctx.Full_column_name(); v != nil {
			return extractor.isSnowSQLExprSensitive(v)
		}
		if v := ctx.Bracket_expression(); v != nil {
			return extractor.isSnowSQLExprSensitive(v)
		}
		if v := ctx.Arr_literal(); v != nil {
			return extractor.isSnowSQLExprSensitive(v)
		}
		if v := ctx.Json_literal(); v != nil {
			return extractor.isSnowSQLExprSensitive(v)
		}

		if v := ctx.Try_cast_expr(); v != nil {
			return extractor.isSnowSQLExprSensitive(v)
		}
		if v := ctx.Object_name(); v != nil {
			return extractor.isSnowSQLExprSensitive(v)
		}
		if v := ctx.Trim_expression(); v != nil {
			return extractor.isSnowSQLExprSensitive(v)
		}
		if v := ctx.Expr_list(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
		}
		return ctx.GetText(), false, nil
	case *snowparser.Full_column_nameContext:
		normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName := normalizedFullColumnName(ctx)
		fieldInfo, err := extractor.snowflakeIsFieldSensitive(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
		if err != nil {
			return "", false, errors.Wrapf(err, "failed to check whether the column %q is sensitive near line %d", normalizedColumnName, ctx.GetStart().GetLine())
		}
		return fieldInfo.name, fieldInfo.sensitive, nil
	case *snowparser.Object_nameContext:
		return ctx.GetText(), false, nil
	case *snowparser.Trim_expressionContext:
		if v := ctx.Expr(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), isSensitive, nil
		}
		panic("never reach here")
	case *snowparser.Try_cast_exprContext:
		if v := ctx.Expr(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), isSensitive, nil
		}
		panic("never reach here")
	case *snowparser.Json_literalContext:
		if v := ctx.AllKv_pair(); len(v) > 0 {
			for _, kvPair := range v {
				_, isSensitive, err := extractor.isSnowSQLExprSensitive(kvPair)
				if err != nil {
					return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", kvPair.GetText(), kvPair.GetStart().GetLine())
				}
				if isSensitive {
					return ctx.GetText(), true, nil
				}
			}
		}
		return ctx.GetText(), false, nil
	case *snowparser.Kv_pairContext:
		if v := ctx.Value(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), isSensitive, nil
		}
		panic("never reach here")
	case *snowparser.Arr_literalContext:
		if v := ctx.AllValue(); len(v) > 0 {
			for _, value := range v {
				_, isSensitive, err := extractor.isSnowSQLExprSensitive(value)
				if err != nil {
					return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", value.GetText(), value.GetStart().GetLine())
				}
				if isSensitive {
					return ctx.GetText(), true, nil
				}
			}
		}
		return ctx.GetText(), false, nil
	case *snowparser.ValueContext:
		return extractor.isSnowSQLExprSensitive(ctx.Expr())
	case *snowparser.Bracket_expressionContext:
		if v := ctx.Expr(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
		}
		if v := ctx.Subquery(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
		}
		panic("never reach here")
	case *snowparser.Iff_exprContext:
		if v := ctx.Search_condition(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(ctx.Search_condition())
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
			for _, expr := range ctx.AllExpr() {
				_, isSensitive, err := extractor.isSnowSQLExprSensitive(expr)
				if err != nil {
					return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
				}
				if isSensitive {
					return ctx.GetText(), true, nil
				}
			}
			return ctx.GetText(), false, nil
		}
		panic("never reach here")
	case *snowparser.Case_expressionContext:
		for _, expr := range ctx.AllExpr() {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(expr)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			if !isSensitive {
				return ctx.GetText(), false, nil
			}
		}
		if v := ctx.AllSwitch_section(); len(v) > 0 {
			for _, switchSection := range v {
				_, isSensitive, err := extractor.isSnowSQLExprSensitive(switchSection)
				if err != nil {
					return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", switchSection.GetText(), switchSection.GetStart().GetLine())
				}
				if isSensitive {
					return ctx.GetText(), true, nil
				}
			}
			return ctx.GetText(), false, nil
		}
		if v := ctx.AllSwitch_search_condition_section(); len(v) > 0 {
			for _, switchSearchConditionSection := range v {
				_, isSensitive, err := extractor.isSnowSQLExprSensitive(switchSearchConditionSection)
				if err != nil {
					return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", switchSearchConditionSection.GetText(), switchSearchConditionSection.GetStart().GetLine())
				}
				if isSensitive {
					return ctx.GetText(), true, nil
				}
			}
			return ctx.GetText(), false, nil
		}
		panic("never reach here")
	case *snowparser.Switch_sectionContext:
		if v := ctx.AllExpr(); len(v) > 0 {
			for _, expr := range v {
				_, isSensitive, err := extractor.isSnowSQLExprSensitive(expr)
				if err != nil {
					return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
				}
				if isSensitive {
					return ctx.GetText(), true, nil
				}
			}
			return ctx.GetText(), false, nil
		}
		panic("never reach here")
	case *snowparser.Switch_search_condition_sectionContext:
		if v := ctx.Search_condition(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
			_, isSensitive, err = extractor.isSnowSQLExprSensitive(ctx.Expr())
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Expr().GetText(), ctx.Expr().GetStart().GetLine())
			}
			return ctx.GetText(), isSensitive, nil
		}
		panic("never reach here")
	case *snowparser.Search_conditionContext:
		if v := ctx.Predicate(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
		}
		if v := ctx.AllSearch_condition(); len(v) > 0 {
			for _, searchCondition := range v {
				_, isSensitive, err := extractor.isSnowSQLExprSensitive(searchCondition)
				if err != nil {
					return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", searchCondition.GetText(), searchCondition.GetStart().GetLine())
				}
				if isSensitive {
					return ctx.GetText(), true, nil
				}
			}
			return ctx.GetText(), false, nil
		}
		panic("never reach here")
	case *snowparser.PredicateContext:
		if v := ctx.AllExpr(); len(v) > 0 {
			for _, expr := range v {
				_, isSensitive, err := extractor.isSnowSQLExprSensitive(expr)
				if err != nil {
					return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
				}
				if isSensitive {
					return ctx.GetText(), true, nil
				}
			}
		}
		if v := ctx.Subquery(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
		}
		if v := ctx.Expr_list(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
		}
		return ctx.GetText(), false, nil
	case *snowparser.SubqueryContext:
		fields, err := extractor.extractSnowsqlSensitiveFieldsQueryStatement(ctx.Query_statement())
		if err != nil {
			return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.GetText(), ctx.GetStart().GetLine())
		}
		var sensitive bool
		for _, field := range fields {
			if field.sensitive {
				sensitive = true
				break
			}
		}
		return ctx.GetText(), sensitive, nil
	case *snowparser.Primitive_expressionContext:
		if v := ctx.Id_(); v != nil {
			return extractor.isSnowSQLExprSensitive(v)
		}
		return ctx.GetText(), false, nil
	case *snowparser.Function_callContext:
		if v := ctx.Ranking_windowed_function(); v != nil {
			return extractor.isSnowSQLExprSensitive(v)
		}
		if v := ctx.Aggregate_function(); v != nil {
			return extractor.isSnowSQLExprSensitive(v)
		}
		if v := ctx.Object_name(); v != nil {
			return v.GetText(), false, nil
		}
		if v := ctx.Expr_list(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), isSensitive, nil
		}
		if v := ctx.Expr(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), isSensitive, nil
		}
		panic("never reach here")
	case *snowparser.Aggregate_functionContext:
		if v := ctx.Expr_list(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), isSensitive, nil
		}
		if ctx.STAR() != nil {
			return ctx.GetText(), false, nil
		}
		if v := ctx.Expr(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
			_, isSensitive, err = extractor.isSnowSQLExprSensitive(ctx.Order_by_clause())
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Order_by_clause().GetText(), ctx.Order_by_clause().GetStart().GetLine())
			}
			return ctx.GetText(), isSensitive, nil
		}
		panic("never reach here")
	case *snowparser.Ranking_windowed_functionContext:
		if v := ctx.Expr(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
		}
		if v := ctx.Over_clause(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), isSensitive, nil
		}
		panic("never reach here")
	case *snowparser.Over_clauseContext:
		if v := ctx.Partition_by(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
		}
		if v := ctx.Order_by_expr(); v != nil {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(v)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), isSensitive, nil
		}
		panic("never reach here")
	case *snowparser.Partition_byContext:
		_, isSensitive, err := extractor.isSnowSQLExprSensitive(ctx.Expr_list())
		if err != nil {
			return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Expr_list().GetText(), ctx.Expr_list().GetStart().GetLine())
		}
		return ctx.GetText(), isSensitive, nil
	case *snowparser.Order_by_exprContext:
		_, isSensitive, err := extractor.isSnowSQLExprSensitive(ctx.Expr_list_sorted())
		if err != nil {
			return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Expr_list_sorted().GetText(), ctx.Expr_list_sorted().GetStart().GetLine())
		}
		return ctx.GetText(), isSensitive, nil
	case *snowparser.Expr_listContext:
		allExpr := ctx.AllExpr()
		for _, expr := range allExpr {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(expr)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
		}
		return ctx.GetText(), false, nil
	case *snowparser.Expr_list_sortedContext:
		allExpr := ctx.AllExpr()
		for _, expr := range allExpr {
			_, isSensitive, err := extractor.isSnowSQLExprSensitive(expr)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			if isSensitive {
				return ctx.GetText(), true, nil
			}
		}
		return ctx.GetText(), false, nil
	case *snowparser.Id_Context:
		normalizedColumnName := parser.NormalizeObjectNamePart(ctx)
		fieldInfo, err := extractor.snowflakeIsFieldSensitive("", "", "", normalizedColumnName)
		if err != nil {
			return "", false, errors.Wrapf(err, "failed to check whether the column %q is sensitive near line %d", normalizedColumnName, ctx.GetStart().GetLine())
		}
		return fieldInfo.name, fieldInfo.sensitive, nil
	}
	panic("never reach here")
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsFromClause(ctx snowparser.IFrom_clauseContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	return extractor.extractSnowsqlSensitiveFieldsTableSources(ctx.Table_sources())
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsTableSources(ctx snowparser.ITable_sourcesContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}
	allTableSources := ctx.AllTable_source()
	var result []fieldInfo
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

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsTableSource(ctx snowparser.ITable_sourceContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}
	return extractor.extractSnowsqlSensitiveFieldsTableSourceItemJoined(ctx.Table_source_item_joined())
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsTableSourceItemJoined(ctx snowparser.ITable_source_item_joinedContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var left []fieldInfo
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

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsJoinClause(ctx snowparser.IJoin_clauseContext, left []fieldInfo) ([]fieldInfo, error) {
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
			leftMap[field.name] = i
		}

		var result []fieldInfo
		result = append(result, left...)
		for _, field := range right {
			if _, ok := leftMap[field.name]; !ok {
				result = append(result, field)
			} else if field.sensitive {
				// If the field is in the left part and the right part, we should keep the field in the left part,
				// and set the sensitive flag to true if the field in the right part is sensitive.
				result[leftMap[field.name]].sensitive = true
			}
		}
		return result, nil
	}

	// For other types of join, we should keep all the columns for the left part and the right part.
	var result []fieldInfo
	result = append(result, left...)
	result = append(result, right...)
	return result, nil
}

func (extractor *sensitiveFieldExtractor) extractSnowsqlSensitiveFieldsObjectRef(ctx snowparser.IObject_refContext) ([]fieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	var result []fieldInfo

	if objectName := ctx.Object_name(); objectName != nil {
		normalizedDatabaseName, tableSchema, err := extractor.snowsqlFindTableSchema(objectName, extractor.currentDatabase, "PUBLIC")
		if err != nil {
			return nil, err
		}
		for _, column := range tableSchema.ColumnList {
			result = append(result, fieldInfo{
				database:  normalizedDatabaseName,
				table:     tableSchema.Name,
				name:      column.Name,
				sensitive: column.Sensitive,
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

	// If the as alias is not nil, we should use the alias name to replace the original table name.
	if ctx.As_alias() != nil {
		id := ctx.As_alias().Alias().Id_()
		aliasName := parser.NormalizeObjectNamePart(id)
		for i := 0; i < len(result); i++ {
			result[i].table = aliasName
		}
	}

	return result, nil
}

func (extractor *sensitiveFieldExtractor) snowsqlFindTableSchema(objectName snowparser.IObject_nameContext, normalizedFallbackDatabaseName, normalizedFallbackSchemaName string) (string, db.TableSchema, error) {
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
	return "", db.TableSchema{}, errors.Errorf(`table %s.%s.%s is not found`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
}

func (extractor *sensitiveFieldExtractor) getAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName string) ([]fieldInfo, error) {
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

	var result []fieldInfo
	for _, field := range extractor.fromFieldList {
		if mask&maskDatabaseName != 0 && normalizedDatabaseName != field.database {
			continue
		}
		if mask&maskSchemaName != 0 && normalizedSchemaName != field.schema {
			continue
		}
		if mask&maskTableName != 0 && normalizedTableName != field.table {
			continue
		}
		result = append(result, field)
	}
	return result, nil
}

// snowflakeIsFieldSensitive iterates through the fromFieldList sequentially until we find the first matching object and return the column name, and whether the column is sensitive.
func (extractor *sensitiveFieldExtractor) snowflakeIsFieldSensitive(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName string) (fieldInfo, error) {
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
			return fieldInfo{}, errors.Errorf(`table name %s is specified without column name`, normalizedTableName)
		}
		mask |= maskTableName
	}
	if normalizedSchemaName != "" {
		if mask&maskTableName == 0 {
			return fieldInfo{}, errors.Errorf(`schema name %s is specified without table name`, normalizedSchemaName)
		}
		mask |= maskSchemaName
	}
	if normalizedDatabaseName != "" {
		if mask&maskSchemaName == 0 {
			return fieldInfo{}, errors.Errorf(`database name %s is specified without schema name`, normalizedDatabaseName)
		}
		mask |= maskDatabaseName
	}

	if mask == maskNone {
		return fieldInfo{}, errors.Errorf(`no object name is specified`)
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
		if mask&maskDatabaseName != 0 && normalizedDatabaseName != field.database {
			continue
		}
		if mask&maskSchemaName != 0 && normalizedSchemaName != field.schema {
			continue
		}
		if mask&maskTableName != 0 && normalizedTableName != field.table {
			continue
		}
		if mask&maskColumnName != 0 && normalizedColumnName != field.name {
			continue
		}
		return field, nil
	}
	return fieldInfo{}, errors.Errorf(`no matching column %q.%q.%q.%q`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
}

func normalizedFullColumnName(ctx snowparser.IFull_column_nameContext) (normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName string) {
	if ctx.GetDb_name() != nil {
		normalizedDatabaseName = parser.NormalizeObjectNamePart(ctx.GetDb_name())
	}
	if ctx.GetSchema() != nil {
		normalizedSchemaName = parser.NormalizeObjectNamePart(ctx.GetSchema())
	}
	if ctx.GetTab_name() != nil {
		normalizedTableName = parser.NormalizeObjectNamePart(ctx.GetTab_name())
	}
	if ctx.GetCol_name() != nil {
		normalizedColumnName = parser.NormalizeObjectNamePart(ctx.GetCol_name())
	}
	return normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName
}

func normalizedObjectName(objectName snowparser.IObject_nameContext, normalizedFallbackDatabaseName, normalizedFallbackSchemaName string) (string, string, string) {
	// TODO(zp): unify here with NormalizeObjectName in backend/plugin/parser/sql/snowsql.go
	var parts []string
	if objectName == nil {
		return "", "", ""
	}
	database := normalizedFallbackDatabaseName
	if d := objectName.GetD(); d != nil {
		normalizedD := parser.NormalizeObjectNamePart(d)
		if normalizedD != "" {
			database = normalizedD
		}
	}
	parts = append(parts, database)

	schema := normalizedFallbackSchemaName
	if s := objectName.GetS(); s != nil {
		normalizedS := parser.NormalizeObjectNamePart(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}
	parts = append(parts, schema)

	normalizedO := parser.NormalizeObjectNamePart(objectName.GetO())
	parts = append(parts, normalizedO)

	return parts[0], parts[1], parts[2]
}
