// Package util implements the util functions.
package util

import (
	"cmp"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/snowsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	snowparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type SnowSensitiveFieldExtractor struct {
	// For Oracle, we need to know the current database to determine if the table is in the current schema.
	currentDatabase    string
	schemaInfo         *db.SensitiveSchemaInfo
	cteOuterSchemaInfo []db.TableSchema

	// SELECT statement specific field.
	fromFieldList []base.FieldInfo
}

func (extractor *SnowSensitiveFieldExtractor) extractSnowsqlSensitiveFields(sql string) ([]db.SensitiveField, error) {
	tree, err := snowparser.ParseSnowSQL(sql)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse snowsql")
	}
	if tree == nil {
		return nil, nil
	}

	listener := &snowsqlSnowSensitiveFieldExtractorListener{
		extractor: extractor,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.result, listener.err
}

type snowsqlSnowSensitiveFieldExtractorListener struct {
	*parser.BaseSnowflakeParserListener

	extractor *SnowSensitiveFieldExtractor
	result    []db.SensitiveField
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
		l.result = append(l.result, db.SensitiveField{
			Name:         field.Name,
			MaskingLevel: field.MaskingLevel,
		})
	}
}

func (extractor *SnowSensitiveFieldExtractor) extractSnowsqlSensitiveFieldsQueryStatement(ctx parser.IQuery_statementContext) ([]base.FieldInfo, error) {
	if ctx.With_expression() != nil {
		allCommandTableExpression := ctx.With_expression().AllCommon_table_expression()

		for _, commandTableExpression := range allCommandTableExpression {
			var result []base.FieldInfo
			var err error
			normalizedCTEName := snowparser.NormalizeSnowSQLObjectNamePart(commandTableExpression.Id_())

			if commandTableExpression.RECURSIVE() != nil || commandTableExpression.UNION() != nil {
				// TODO(zp): refactor code
				fieldsInAnchorClause, err := extractor.extractSnowsqlSensitiveFieldsQueryStatement(commandTableExpression.Anchor_clause().Query_statement())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields of the anchor clause of recursive CTE %q near line %d", normalizedCTEName, commandTableExpression.GetStart().GetLine())
				}

				tempCTEOuterSchemaInfo := db.TableSchema{
					Name: normalizedCTEName,
				}
				for i := 0; i < len(fieldsInAnchorClause); i++ {
					tempCTEOuterSchemaInfo.ColumnList = append(tempCTEOuterSchemaInfo.ColumnList, db.ColumnInfo{
						Name:         fieldsInAnchorClause[i].Name,
						MaskingLevel: fieldsInAnchorClause[i].MaskingLevel,
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
						if cmp.Less[storepb.MaskingLevel](tempCTEOuterSchemaInfo.ColumnList[i].MaskingLevel, fieldsInRecursiveClause[i].MaskingLevel) {
							change = true
							tempCTEOuterSchemaInfo.ColumnList[i].MaskingLevel = fieldsInRecursiveClause[i].MaskingLevel
							result[i].MaskingLevel = fieldsInRecursiveClause[i].MaskingLevel
						}
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
					normalizedColumnName := snowparser.NormalizeSnowSQLObjectNamePart(columnName.Id_())
					result[i].Name = normalizedColumnName
				}
			}
			// Append to the extractor.schemaInfo.DatabaseList
			columnList := make([]db.ColumnInfo, 0, len(result))
			for _, field := range result {
				columnList = append(columnList, db.ColumnInfo{
					Name:         field.Name,
					MaskingLevel: field.MaskingLevel,
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
			finalLevel := result[i].MaskingLevel
			if cmp.Less[storepb.MaskingLevel](finalLevel, right[i].MaskingLevel) {
				finalLevel = right[i].MaskingLevel
			}
			result[i].MaskingLevel = finalLevel
		}
	}
	return result, nil
}

func (extractor *SnowSensitiveFieldExtractor) extractSnowsqlSensitiveFieldSetOperator(ctx parser.ISet_operatorsContext) ([]base.FieldInfo, error) {
	return extractor.extractSnowsqlSensitiveFieldsSelectStatement(ctx.Select_statement())
}

func (extractor *SnowSensitiveFieldExtractor) extractSnowsqlSensitiveFieldsSelectStatement(ctx parser.ISelect_statementContext) ([]base.FieldInfo, error) {
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
				normalizedTableName = snowparser.NormalizeSnowSQLObjectNamePart(v.Id_())
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
				normalizedColumnName = snowparser.NormalizeSnowSQLObjectNamePart(columnElem.Column_name().Id_())
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
				result[len(result)-1].Name = snowparser.NormalizeSnowSQLObjectNamePart(asAlias.Alias().Id_())
			}
		} else if expressionElem := iSelectListElem.Expression_elem(); expressionElem != nil {
			if v := expressionElem.Expr(); v != nil {
				columnName, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
				}
				result = append(result, base.FieldInfo{
					Name:         columnName,
					MaskingLevel: maskingLevel,
				})
			} else if v := expressionElem.Predicate(); v != nil {
				columnName, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
				}
				result = append(result, base.FieldInfo{
					Name:         columnName,
					MaskingLevel: maskingLevel,
				})
			}

			if asAlias := expressionElem.As_alias(); asAlias != nil {
				result[len(result)-1].Name = snowparser.NormalizeSnowSQLObjectNamePart(asAlias.Alias().Id_())
			}
		}
	}

	return result, nil
}

// The closure of the IExprContext.
func (extractor *SnowSensitiveFieldExtractor) evalSnowSQLExprMaskingLevel(ctx antlr.RuleContext) (string, storepb.MaskingLevel, error) {
	switch ctx := ctx.(type) {
	case *parser.ExprContext:
		if v := ctx.Primitive_expression(); v != nil {
			return extractor.evalSnowSQLExprMaskingLevel(v)
		}
		if v := ctx.Function_call(); v != nil {
			return extractor.evalSnowSQLExprMaskingLevel(v)
		}

		finalLevel := defaultMaskingLevel
		for _, expr := range ctx.AllExpr() {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(expr)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return ctx.GetText(), finalLevel, nil
			}
		}
		if v := ctx.Subquery(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return ctx.GetText(), finalLevel, nil
			}
		}

		if v := ctx.Case_expression(); v != nil {
			return extractor.evalSnowSQLExprMaskingLevel(v)
		}
		if v := ctx.Iff_expr(); v != nil {
			return extractor.evalSnowSQLExprMaskingLevel(v)
		}
		if v := ctx.Full_column_name(); v != nil {
			return extractor.evalSnowSQLExprMaskingLevel(v)
		}
		if v := ctx.Bracket_expression(); v != nil {
			return extractor.evalSnowSQLExprMaskingLevel(v)
		}
		if v := ctx.Arr_literal(); v != nil {
			return extractor.evalSnowSQLExprMaskingLevel(v)
		}
		if v := ctx.Json_literal(); v != nil {
			return extractor.evalSnowSQLExprMaskingLevel(v)
		}

		if v := ctx.Try_cast_expr(); v != nil {
			return extractor.evalSnowSQLExprMaskingLevel(v)
		}
		if v := ctx.Object_name(); v != nil {
			return extractor.evalSnowSQLExprMaskingLevel(v)
		}
		if v := ctx.Trim_expression(); v != nil {
			return extractor.evalSnowSQLExprMaskingLevel(v)
		}
		if v := ctx.Expr_list(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return ctx.GetText(), finalLevel, nil
			}
		}
		return ctx.GetText(), finalLevel, nil
	case *parser.Full_column_nameContext:
		normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName := normalizedFullColumnName(ctx)
		fieldInfo, err := extractor.snowflakeGetField(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
		if err != nil {
			return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the column %q is sensitive near line %d", normalizedColumnName, ctx.GetStart().GetLine())
		}
		return fieldInfo.Name, fieldInfo.MaskingLevel, nil
	case *parser.Object_nameContext:
		normalizedDatabaseName, normalizedSchemaName, normalizedTableName := normalizedObjectName(ctx, extractor.currentDatabase, "PUBLIC")
		fieldInfo, err := extractor.snowflakeGetField(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, "")
		if err != nil {
			return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the object %q is sensitive near line %d", normalizedTableName, ctx.GetStart().GetLine())
		}
		return fieldInfo.Name, fieldInfo.MaskingLevel, nil
	case *parser.Trim_expressionContext:
		if v := ctx.Expr(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingLevel, nil
		}
		panic("never reach here")
	case *parser.Try_cast_exprContext:
		if v := ctx.Expr(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingLevel, nil
		}
		panic("never reach here")
	case *parser.Json_literalContext:
		finalLevel := defaultMaskingLevel
		if v := ctx.AllKv_pair(); len(v) > 0 {
			for _, kvPair := range v {
				_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(kvPair)
				if err != nil {
					return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", kvPair.GetText(), kvPair.GetStart().GetLine())
				}
				if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
					finalLevel = maskingLevel
				}
				if finalLevel == maxMaskingLevel {
					return ctx.GetText(), finalLevel, nil
				}
			}
		}
		return ctx.GetText(), finalLevel, nil
	case *parser.Kv_pairContext:
		if v := ctx.Value(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingLevel, nil
		}
		panic("never reach here")
	case *parser.Arr_literalContext:
		finalLevel := defaultMaskingLevel
		if v := ctx.AllValue(); len(v) > 0 {
			for _, value := range v {
				_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(value)
				if err != nil {
					return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", value.GetText(), value.GetStart().GetLine())
				}
				if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
					finalLevel = maskingLevel
				}
				if finalLevel == maxMaskingLevel {
					return ctx.GetText(), finalLevel, nil
				}
			}
		}
		return ctx.GetText(), finalLevel, nil
	case *parser.ValueContext:
		return extractor.evalSnowSQLExprMaskingLevel(ctx.Expr())
	case *parser.Bracket_expressionContext:
		finalLevel := defaultMaskingLevel
		if v := ctx.Expr(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return ctx.GetText(), finalLevel, nil
			}
		}
		if v := ctx.Subquery(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			return ctx.GetText(), finalLevel, nil
		}
		panic("never reach here")
	case *parser.Iff_exprContext:
		finalLevel := defaultMaskingLevel
		if v := ctx.Search_condition(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(ctx.Search_condition())
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return ctx.GetText(), finalLevel, nil
			}
			for _, expr := range ctx.AllExpr() {
				_, finalLevel, err := extractor.evalSnowSQLExprMaskingLevel(expr)
				if err != nil {
					return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
				}
				if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
					finalLevel = maskingLevel
				}
				if finalLevel == maxMaskingLevel {
					return ctx.GetText(), finalLevel, nil
				}
			}
			return ctx.GetText(), finalLevel, nil
		}
		panic("never reach here")
	case *parser.Case_expressionContext:
		finalLevel := defaultMaskingLevel
		for _, expr := range ctx.AllExpr() {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(expr)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return ctx.GetText(), finalLevel, nil
			}
		}
		if v := ctx.AllSwitch_section(); len(v) > 0 {
			for _, switchSection := range v {
				_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(switchSection)
				if err != nil {
					return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", switchSection.GetText(), switchSection.GetStart().GetLine())
				}
				if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
					finalLevel = maskingLevel
				}
				if finalLevel == maxMaskingLevel {
					return ctx.GetText(), finalLevel, nil
				}
			}
			return ctx.GetText(), finalLevel, nil
		}
		if v := ctx.AllSwitch_search_condition_section(); len(v) > 0 {
			for _, switchSearchConditionSection := range v {
				_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(switchSearchConditionSection)
				if err != nil {
					return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", switchSearchConditionSection.GetText(), switchSearchConditionSection.GetStart().GetLine())
				}
				if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
					finalLevel = maskingLevel
				}
				if finalLevel == maxMaskingLevel {
					return ctx.GetText(), finalLevel, nil
				}
			}
			return ctx.GetText(), finalLevel, nil
		}
		panic("never reach here")
	case *parser.Switch_sectionContext:
		finalLevel := defaultMaskingLevel
		if v := ctx.AllExpr(); len(v) > 0 {
			for _, expr := range v {
				_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(expr)
				if err != nil {
					return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
				}
				if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
					finalLevel = maskingLevel
				}
				if finalLevel == maxMaskingLevel {
					return ctx.GetText(), finalLevel, nil
				}
			}
			return ctx.GetText(), finalLevel, nil
		}
		panic("never reach here")
	case *parser.Switch_search_condition_sectionContext:
		finalLevel := defaultMaskingLevel
		if v := ctx.Search_condition(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return ctx.GetText(), finalLevel, nil
			}
			_, maskingLevel, err = extractor.evalSnowSQLExprMaskingLevel(ctx.Expr())
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Expr().GetText(), ctx.Expr().GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			return ctx.GetText(), finalLevel, nil
		}
		panic("never reach here")
	case *parser.Search_conditionContext:
		finalLevel := defaultMaskingLevel
		if v := ctx.Predicate(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Predicate().GetText(), ctx.Predicate().GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
		}
		if v := ctx.AllSearch_condition(); len(v) > 0 {
			for _, searchCondition := range v {
				_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(searchCondition)
				if err != nil {
					return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", searchCondition.GetText(), searchCondition.GetStart().GetLine())
				}
				if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
					finalLevel = maskingLevel
				}
				if finalLevel == maxMaskingLevel {
					return ctx.GetText(), finalLevel, nil
				}
			}
			return ctx.GetText(), finalLevel, nil
		}
		panic("never reach here")
	case *parser.PredicateContext:
		finalLevel := defaultMaskingLevel
		if v := ctx.AllExpr(); len(v) > 0 {
			for _, expr := range v {
				_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(expr)
				if err != nil {
					return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
				}
				if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
					finalLevel = maskingLevel
				}
				if finalLevel == maxMaskingLevel {
					return ctx.GetText(), finalLevel, nil
				}
			}
		}
		if v := ctx.Subquery(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
		}
		if v := ctx.Expr_list(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return ctx.GetText(), finalLevel, nil
			}
		}
		return ctx.GetText(), finalLevel, nil
	case *parser.SubqueryContext:
		fields, err := extractor.extractSnowsqlSensitiveFieldsQueryStatement(ctx.Query_statement())
		if err != nil {
			return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.GetText(), ctx.GetStart().GetLine())
		}
		finalLevel := defaultMaskingLevel
		for _, field := range fields {
			if cmp.Less[storepb.MaskingLevel](finalLevel, field.MaskingLevel) {
				finalLevel = field.MaskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return ctx.GetText(), finalLevel, nil
			}
		}
		return ctx.GetText(), finalLevel, nil
	case *parser.Primitive_expressionContext:
		if v := ctx.Id_(); v != nil {
			return extractor.evalSnowSQLExprMaskingLevel(v)
		}
		return ctx.GetText(), defaultMaskingLevel, nil
	case *parser.Function_callContext:
		if v := ctx.Ranking_windowed_function(); v != nil {
			return extractor.evalSnowSQLExprMaskingLevel(v)
		}
		if v := ctx.Aggregate_function(); v != nil {
			return extractor.evalSnowSQLExprMaskingLevel(v)
		}
		if v := ctx.Object_name(); v != nil {
			return v.GetText(), defaultMaskingLevel, nil
		}
		if v := ctx.Expr_list(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingLevel, nil
		}
		if v := ctx.Expr(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingLevel, nil
		}
		panic("never reach here")
	case *parser.Aggregate_functionContext:
		if v := ctx.Expr_list(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingLevel, nil
		}
		if ctx.STAR() != nil {
			return ctx.GetText(), defaultMaskingLevel, nil
		}
		if v := ctx.Expr(); v != nil {
			finalLevel := defaultMaskingLevel
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return ctx.GetText(), finalLevel, nil
			}
			_, maskingLevel, err = extractor.evalSnowSQLExprMaskingLevel(ctx.Order_by_clause())
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Order_by_clause().GetText(), ctx.Order_by_clause().GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			return ctx.GetText(), finalLevel, nil
		}
		panic("never reach here")
	case *parser.Ranking_windowed_functionContext:
		finalLevel := defaultMaskingLevel
		if v := ctx.Expr(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return ctx.GetText(), finalLevel, nil
			}
		}
		if v := ctx.Over_clause(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			return ctx.GetText(), finalLevel, nil
		}
		panic("never reach here")
	case *parser.Over_clauseContext:
		finalLevel := defaultMaskingLevel
		if v := ctx.Partition_by(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return ctx.GetText(), finalLevel, nil
			}
		}
		if v := ctx.Order_by_expr(); v != nil {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(v)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			return ctx.GetText(), finalLevel, nil
		}
		panic("never reach here")
	case *parser.Partition_byContext:
		_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(ctx.Expr_list())
		if err != nil {
			return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Expr_list().GetText(), ctx.Expr_list().GetStart().GetLine())
		}
		return ctx.GetText(), maskingLevel, nil
	case *parser.Order_by_exprContext:
		_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(ctx.Expr_list_sorted())
		if err != nil {
			return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Expr_list_sorted().GetText(), ctx.Expr_list_sorted().GetStart().GetLine())
		}
		return ctx.GetText(), maskingLevel, nil
	case *parser.Expr_listContext:
		finalLevel := defaultMaskingLevel
		allExpr := ctx.AllExpr()
		for _, expr := range allExpr {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(expr)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return ctx.GetText(), finalLevel, nil
			}
		}
		return ctx.GetText(), finalLevel, nil
	case *parser.Expr_list_sortedContext:
		finalLevel := defaultMaskingLevel
		allExpr := ctx.AllExpr()
		for _, expr := range allExpr {
			_, maskingLevel, err := extractor.evalSnowSQLExprMaskingLevel(expr)
			if err != nil {
				return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			if cmp.Less[storepb.MaskingLevel](finalLevel, maskingLevel) {
				finalLevel = maskingLevel
			}
			if finalLevel == maxMaskingLevel {
				return ctx.GetText(), finalLevel, nil
			}
		}
		return ctx.GetText(), finalLevel, nil
	case *parser.Id_Context:
		normalizedColumnName := snowparser.NormalizeSnowSQLObjectNamePart(ctx)
		fieldInfo, err := extractor.snowflakeGetField("", "", "", normalizedColumnName)
		if err != nil {
			return "", storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to check whether the column %q is sensitive near line %d", normalizedColumnName, ctx.GetStart().GetLine())
		}
		return fieldInfo.Name, fieldInfo.MaskingLevel, nil
	}
	panic("never reach here")
}

func (extractor *SnowSensitiveFieldExtractor) extractSnowsqlSensitiveFieldsFromClause(ctx parser.IFrom_clauseContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}

	return extractor.extractSnowsqlSensitiveFieldsTableSources(ctx.Table_sources())
}

func (extractor *SnowSensitiveFieldExtractor) extractSnowsqlSensitiveFieldsTableSources(ctx parser.ITable_sourcesContext) ([]base.FieldInfo, error) {
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

func (extractor *SnowSensitiveFieldExtractor) extractSnowsqlSensitiveFieldsTableSource(ctx parser.ITable_sourceContext) ([]base.FieldInfo, error) {
	if ctx == nil {
		return nil, nil
	}
	return extractor.extractSnowsqlSensitiveFieldsTableSourceItemJoined(ctx.Table_source_item_joined())
}

func (extractor *SnowSensitiveFieldExtractor) extractSnowsqlSensitiveFieldsTableSourceItemJoined(ctx parser.ITable_source_item_joinedContext) ([]base.FieldInfo, error) {
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

func (extractor *SnowSensitiveFieldExtractor) extractSnowsqlSensitiveFieldsJoinClause(ctx parser.IJoin_clauseContext, left []base.FieldInfo) ([]base.FieldInfo, error) {
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
			} else if cmp.Less[storepb.MaskingLevel](left[idx].MaskingLevel, field.MaskingLevel) {
				// If the field is in the left part and the right part, we should keep the field in the left part,
				// and set the sensitive flag to true if the field in the right part is sensitive.
				result[idx].MaskingLevel = field.MaskingLevel
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

func (extractor *SnowSensitiveFieldExtractor) extractSnowsqlSensitiveFieldsObjectRef(ctx parser.IObject_refContext) ([]base.FieldInfo, error) {
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
				Database:     normalizedDatabaseName,
				Table:        tableSchema.Name,
				Name:         column.Name,
				MaskingLevel: column.MaskingLevel,
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
			normalizedPivotColumnName := snowparser.NormalizeSnowSQLObjectNamePart(pivotColumnName)
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
			normalizedValueColumnName := snowparser.NormalizeSnowSQLObjectNamePart(valueColumnName)
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
					Name:         literal.GetText(),
					MaskingLevel: pivotColumnInOriginalResult.MaskingLevel,
				})
			}
		} else if v := ctx.Pivot_unpivot(); v.UNPIVOT() != nil {
			var strippedColumnIndices []int
			var strippedColumnInOriginalResult []base.FieldInfo
			for idx, columnName := range v.Column_list().AllColumn_name() {
				normalizedColumnName := snowparser.NormalizeSnowSQLObjectNamePart(columnName.Id_())
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

			finalLevel := defaultMaskingLevel
			for _, field := range strippedColumnInOriginalResult {
				if cmp.Less[storepb.MaskingLevel](finalLevel, field.MaskingLevel) {
					finalLevel = field.MaskingLevel
				}
				if finalLevel == maxMaskingLevel {
					break
				}
			}

			valueColumnName := v.Id_(0)
			normalizedValueColumnName := snowparser.NormalizeSnowSQLObjectNamePart(valueColumnName)

			nameColumnName := v.Column_name().Id_()
			normalizedNameColumnName := snowparser.NormalizeSnowSQLObjectNamePart(nameColumnName)

			result = append(result, base.FieldInfo{
				Name:         normalizedNameColumnName,
				MaskingLevel: defaultMaskingLevel,
			}, base.FieldInfo{
				Name:         normalizedValueColumnName,
				MaskingLevel: finalLevel,
			})
		}
	}

	// If the as alias is not nil, we should use the alias name to replace the original table name.
	if ctx.As_alias() != nil {
		id := ctx.As_alias().Alias().Id_()
		aliasName := snowparser.NormalizeSnowSQLObjectNamePart(id)
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

func (extractor *SnowSensitiveFieldExtractor) snowsqlFindTableSchema(objectName parser.IObject_nameContext, normalizedFallbackDatabaseName, normalizedFallbackSchemaName string) (string, db.TableSchema, error) {
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

func (extractor *SnowSensitiveFieldExtractor) getAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName string) ([]base.FieldInfo, error) {
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
func (extractor *SnowSensitiveFieldExtractor) snowflakeGetField(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName string) (base.FieldInfo, error) {
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
		normalizedDatabaseName = snowparser.NormalizeSnowSQLObjectNamePart(ctx.GetDb_name())
	}
	if ctx.GetSchema() != nil {
		normalizedSchemaName = snowparser.NormalizeSnowSQLObjectNamePart(ctx.GetSchema())
	}
	if ctx.GetTab_name() != nil {
		normalizedTableName = snowparser.NormalizeSnowSQLObjectNamePart(ctx.GetTab_name())
	}
	if ctx.GetCol_name() != nil {
		normalizedColumnName = snowparser.NormalizeSnowSQLObjectNamePart(ctx.GetCol_name())
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
		normalizedD := snowparser.NormalizeSnowSQLObjectNamePart(d)
		if normalizedD != "" {
			database = normalizedD
		}
	}
	parts = append(parts, database)

	schema := normalizedFallbackSchemaName
	if s := objectName.GetS(); s != nil {
		normalizedS := snowparser.NormalizeSnowSQLObjectNamePart(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}
	parts = append(parts, schema)

	normalizedO := snowparser.NormalizeSnowSQLObjectNamePart(objectName.GetO())
	parts = append(parts, normalizedO)

	return parts[0], parts[1], parts[2]
}
