package v2

import (
	"context"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/snowsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

type querySpanExtractor struct {
	ctx context.Context

	connectedDB     string
	connectedSchema string
	// https://docs.snowflake.com/en/sql-reference/identifiers-syntax
	ignoreCaseSensitive bool

	f base.GetDatabaseMetadataFunc
	l base.ListDatabaseNamesFunc

	// Private fields.
	// ctes is used to record the common table expressions (CTEs) in the query.
	// It should be shrunk to 0 after each query span extraction.
	ctes []*base.PseudoTable

	// tableSourcesFrom is used to record the table sources from the query.
	tableSourcesFrom []base.TableSource
}

func newQuerySpanExtractor(connectedDB, connectedSchema string, f base.GetDatabaseMetadataFunc, l base.ListDatabaseNamesFunc, ignoreCaseSensitive bool) *querySpanExtractor {
	return &querySpanExtractor{
		connectedDB:         connectedDB,
		connectedSchema:     connectedSchema,
		ignoreCaseSensitive: ignoreCaseSensitive,
		f:                   f,
		l:                   l,
	}
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx

	return nil, nil
}

type selectOnlyListener struct {
	*parser.BaseSnowflakeParserListener

	q      *querySpanExtractor
	result *base.QuerySpan
	err    error
}

func (l *selectOnlyListener) EnterSelect_stmt(ctx *parser.Dml_commandContext) {
	if l.err != nil {
		return
	}

	if ctx.Query_statement() == nil {
		return
	}

	parent := ctx.GetParent()
	if parent == nil {
		return
	}

	if _, ok := parent.(*parser.Sql_commandContext); !ok {
		return
	}

	result, err := l.q.extractSnowsqlSensitiveFieldsQueryStatement(ctx.Query_statement())
	if err != nil {
		l.err = err
	}
	l.result = &base.QuerySpan{
		Results: result.GetQuerySpanResult(),
	}
}

func (q *querySpanExtractor) extractSnowsqlSensitiveFieldsQueryStatement(ctx parser.IQuery_statementContext) (*base.PseudoTable, error) {
	if ctx.With_expression() != nil {
		allCommandTableExpression := ctx.With_expression().AllCommon_table_expression()
		for _, commandTableExpression := range allCommandTableExpression {
			normalizedCTEName := snowflake.NormalizeSnowSQLObjectNamePart(commandTableExpression.Id_())
			var err error
			pseudoTable := &base.PseudoTable{
				Name:    normalizedCTEName,
				Columns: make([]base.QuerySpanResult, 0),
			}
			if commandTableExpression.RECURSIVE() != nil || commandTableExpression.UNION() != nil {
				// TODO(zp): refactor code
				anchorTableSource, err := q.extractSnowsqlSensitiveFieldsQueryStatement(commandTableExpression.Anchor_clause().Query_statement())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields of the anchor clause of recursive CTE %q near line %d", normalizedCTEName, commandTableExpression.GetStart().GetLine())
				}
				tempCte := &base.PseudoTable{
					Name:    normalizedCTEName,
					Columns: anchorTableSource.GetQuerySpanResult(),
				}
				q.ctes = append(q.ctes, tempCte)
				originalSize := len(q.ctes)
				for {
					originalSize := len(q.ctes)
					recursivePartTableSource, err := q.extractSnowsqlSensitiveFieldsQueryStatement(commandTableExpression.Recursive_clause().Query_statement())
					if err != nil {
						return nil, errors.Wrapf(err, "failed to extract sensitive fields of the recursive clause of recursive CTE %q near line %d", normalizedCTEName, commandTableExpression.Recursive_clause().GetStart().GetLine())
					}
					anchorQuerySpanResults := q.ctes[originalSize].GetQuerySpanResult()
					recursivePartQuerySpanResults := recursivePartTableSource.GetQuerySpanResult()
					if len(anchorQuerySpanResults) != len(recursivePartQuerySpanResults) {
						return nil, errors.Errorf("recursive clause returns %d fields, but anchor clause returns %d fields in recursive CTE %q near line %d", len(anchorQuerySpanResults), len(recursivePartQuerySpanResults), normalizedCTEName, commandTableExpression.GetStart().GetLine())
					}
					changed := false
					for i := range anchorQuerySpanResults {
						var hasChange bool
						anchorQuerySpanResults[i].SourceColumns, hasChange = base.MergeSourceColumnSet(anchorQuerySpanResults[i].SourceColumns, recursivePartQuerySpanResults[i].SourceColumns)
						changed = changed || hasChange
					}
					tempCte := &base.PseudoTable{
						Name:    normalizedCTEName,
						Columns: anchorQuerySpanResults,
					}
					q.ctes = q.ctes[:originalSize-1]
					if !changed {
						break
					}
					q.ctes = append(q.ctes, tempCte)
				}
				q.ctes = q.ctes[:originalSize-1]
				pseudoTable = tempCte
			} else {
				pseudoTable, err = q.extractSnowsqlSensitiveFieldsQueryStatement(commandTableExpression.Query_statement())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields of the CTE %q near line %d", normalizedCTEName, commandTableExpression.GetStart().GetLine())
				}
			}

			if commandTableExpression.Column_list() != nil {
				if len(commandTableExpression.Column_list().AllColumn_name()) != len(pseudoTable.GetQuerySpanResult()) {
					return nil, errors.Errorf("the number of columns in the CTE %q near line %d returns %d fields, but the column list returns %d fields", normalizedCTEName, commandTableExpression.GetStart().GetLine(), len(pseudoTable.GetQuerySpanResult()), len(commandTableExpression.Column_list().AllColumn_name()))
				}
				for i, columnName := range commandTableExpression.Column_list().AllColumn_name() {
					newPseudoTable := &base.PseudoTable{
						Name:    normalizedCTEName,
						Columns: make([]base.QuerySpanResult, 0),
					}
					newPseudoTable.Columns = append(newPseudoTable.Columns, pseudoTable.GetQuerySpanResult()[:i]...)
					newPseudoTable.Columns = append(newPseudoTable.Columns, base.QuerySpanResult{
						Name:          snowflake.NormalizeSnowSQLObjectNamePart(columnName.Id_()),
						SourceColumns: pseudoTable.GetQuerySpanResult()[i].SourceColumns,
					})
					newPseudoTable.Columns = append(newPseudoTable.Columns, pseudoTable.GetQuerySpanResult()[i+1:]...)
				}
			}
			q.ctes = append(q.ctes, pseudoTable)
		}
	}

	selectStatement := ctx.Select_statement()
	result, err := q.extractSnowsqlSensitiveFieldsSelectStatement(selectStatement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields of the query statement near line %d", selectStatement.GetStart().GetLine())
	}

	allSetOperators := ctx.AllSet_operators()
	for i, setOperator := range allSetOperators {
		// For UNION operator, the number of the columns in the result set is the same, and will use the left part's column name.
		// So we only need to extract the sensitive fields of the right part.
		right, err := q.extractSnowsqlSensitiveFieldSetOperator(setOperator)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract the %d set operator near line %d", i+1, setOperator.GetStart().GetLine())
		}
		resultQuerySpanResults := result.GetQuerySpanResult()
		rightQuerySpanResults := right.GetQuerySpanResult()
		if len(resultQuerySpanResults) != len(rightQuerySpanResults) {
			return nil, errors.Errorf("the number of columns in the query statement nearly line %d returns %d fields, but %d set operator near line %d returns %d fields", selectStatement.GetStart().GetLine(), len(resultQuerySpanResults), i+1, setOperator.GetStart().GetLine(), len(rightQuerySpanResults))
		}
		for i := range rightQuerySpanResults {
			result.Columns[i].SourceColumns, _ = base.MergeSourceColumnSet(resultQuerySpanResults[i].SourceColumns, rightQuerySpanResults[i].SourceColumns)
		}
	}
	return result, nil
}

func (q *querySpanExtractor) extractSnowsqlSensitiveFieldSetOperator(ctx parser.ISet_operatorsContext) (*base.PseudoTable, error) {
	return q.extractSnowsqlSensitiveFieldsSelectStatement(ctx.Select_statement())
}

func (q *querySpanExtractor) extractSnowsqlSensitiveFieldsSelectStatement(ctx parser.ISelect_statementContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	if ctx.Select_optional_clauses().From_clause() != nil {
		tableSourcesFrom, err := q.extractSnowsqlSensitiveFieldsFromClause(ctx.Select_optional_clauses().From_clause())
		if err != nil {
			return nil, err
		}
		originalFromFieldsLength := len(q.tableSourcesFrom)
		q.tableSourcesFrom = append(q.tableSourcesFrom, tableSourcesFrom)
		defer func() {
			q.tableSourcesFrom = q.tableSourcesFrom[:originalFromFieldsLength]
		}()
	}

	result := &base.PseudoTable{
		Name:    "",
		Columns: make([]base.QuerySpanResult, 0),
	}

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
				normalizedTableName = snowflake.NormalizeSnowSQLObjectNamePart(v.Id_())
			} else if v := columnElem.Object_name(); v != nil {
				normalizedDatabaseName, normalizedSchemaName, normalizedTableName = normalizedObjectName(v, "", "")
			}
			if columnElem.STAR() != nil {
				left, err := q.getAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields of the query statement near line %d", ctx.GetStart().GetLine())
				}
				result.Columns = append(result.Columns, left...)
			} else if columnElem.Column_name() != nil {
				normalizedColumnName = snowflake.NormalizeSnowSQLObjectNamePart(columnElem.Column_name().Id_())
				querySpanResult, err := q.snowflakeGetField(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to check whether the column %q is sensitive near line %d", normalizedColumnName, columnElem.Column_name().GetStart().GetLine())
				}
				result.Columns = append(result.Columns, querySpanResult)
			} else if columnElem.DOLLAR() != nil {
				columnPosition, err := strconv.Atoi(columnElem.Column_position().Num().GetText())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse column position %q to integer near line %d", columnElem.Column_position().Num().GetText(), columnElem.Column_position().Num().GetStart().GetLine())
				}
				if columnPosition < 1 {
					return nil, errors.Wrapf(err, "column position %d is invalid because it is less than 1 near line %d", columnPosition, columnElem.Column_position().Num().GetStart().GetLine())
				}
				left, err := q.getAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields of the query statement near line %d", ctx.GetStart().GetLine())
				}
				if columnPosition > len(left) {
					return nil, errors.Wrapf(err, "column position is invalid because want to try get the %d column near line %d, but FROM clause only returns %d columns for %q.%q.%q", columnPosition, columnElem.Column_position().Num().GetStart().GetLine(), len(left), normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
				}
				result.Columns = append(result.Columns, left[columnPosition-1])
			}
			if asAlias := columnElem.As_alias(); asAlias != nil {
				result.Columns[len(result.Columns)-1].Name = snowflake.NormalizeSnowSQLObjectNamePart(asAlias.Alias().Id_())
			}
		} else if expressionElem := iSelectListElem.Expression_elem(); expressionElem != nil {
			if v := expressionElem.Expr(); v != nil {
				columnName, querySpanResult, err := q.evalSnowSQLExprMaskingAttributes(v)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
				}
				result.Columns = append(result.Columns, base.QuerySpanResult{
					Name:          columnName,
					SourceColumns: querySpanResult.SourceColumns,
				})
			} else if v := expressionElem.Predicate(); v != nil {
				columnName, querySpanResult, err := q.evalSnowSQLExprMaskingAttributes(v)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
				}
				result.Columns = append(result.Columns, base.QuerySpanResult{
					Name:          columnName,
					SourceColumns: querySpanResult.SourceColumns,
				})
			}

			if asAlias := expressionElem.As_alias(); asAlias != nil {
				result.Columns[len(result.Columns)-1].Name = snowflake.NormalizeSnowSQLObjectNamePart(asAlias.Alias().Id_())
			}
		}
	}

	return result, nil
}

// The closure of the IExprContext.
func (q *querySpanExtractor) evalSnowSQLExprMaskingAttributes(ctx antlr.RuleContext) (string, base.QuerySpanResult, error) {
	switch ctx := ctx.(type) {
	case *parser.ExprContext:
		if v := ctx.Primitive_expression(); v != nil {
			return q.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Function_call(); v != nil {
			return q.evalSnowSQLExprMaskingAttributes(v)
		}

		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expr := range ctx.AllExpr() {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(expr)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
		}
		if v := ctx.Subquery(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
		}

		if v := ctx.Case_expression(); v != nil {
			return q.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Iff_expr(); v != nil {
			return q.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Full_column_name(); v != nil {
			return q.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Bracket_expression(); v != nil {
			return q.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Arr_literal(); v != nil {
			return q.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Json_literal(); v != nil {
			return q.evalSnowSQLExprMaskingAttributes(v)
		}

		if v := ctx.Try_cast_expr(); v != nil {
			return q.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Object_name(); v != nil {
			return q.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Trim_expression(); v != nil {
			return q.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Expr_list(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
		}
		return ctx.GetText(), querySpanResult, nil
	case *parser.Full_column_nameContext:
		normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName := normalizedFullColumnName(ctx)
		querySpanResult, err := q.snowflakeGetField(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
		if err != nil {
			return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the column %q is sensitive near line %d", normalizedColumnName, ctx.GetStart().GetLine())
		}
		return querySpanResult.Name, querySpanResult, nil
	case *parser.Object_nameContext:
		normalizedDatabaseName, normalizedSchemaName, normalizedTableName := normalizedObjectName(ctx, q.connectedDB, "PUBLIC")
		fieldInfo, err := q.snowflakeGetField(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, "")
		if err != nil {
			return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the object %q is sensitive near line %d", normalizedTableName, ctx.GetStart().GetLine())
		}
		return fieldInfo.Name, fieldInfo, nil
	case *parser.Trim_expressionContext:
		if v := ctx.Expr(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		panic("never reach here")
	case *parser.Try_cast_exprContext:
		if v := ctx.Expr(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		panic("never reach here")
	case *parser.Json_literalContext:
		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if v := ctx.AllKv_pair(); len(v) > 0 {
			for _, kvPair := range v {
				_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(kvPair)
				if err != nil {
					return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", kvPair.GetText(), kvPair.GetStart().GetLine())
				}
				querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			}
		}
		return ctx.GetText(), querySpanResult, nil
	case *parser.Kv_pairContext:
		if v := ctx.Value(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		panic("never reach here")
	case *parser.Arr_literalContext:
		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if v := ctx.AllValue(); len(v) > 0 {
			for _, value := range v {
				_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(value)
				if err != nil {
					return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", value.GetText(), value.GetStart().GetLine())
				}
				querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			}
		}
		return ctx.GetText(), querySpanResult, nil
	case *parser.ValueContext:
		return q.evalSnowSQLExprMaskingAttributes(ctx.Expr())
	case *parser.Bracket_expressionContext:
		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if v := ctx.Expr(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
		}
		if v := ctx.Subquery(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			return ctx.GetText(), querySpanResult, nil
		}
		panic("never reach here")
	case *parser.Iff_exprContext:
		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if v := ctx.Search_condition(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(ctx.Search_condition())
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			for _, expr := range ctx.AllExpr() {
				_, finalAttributes, err := q.evalSnowSQLExprMaskingAttributes(expr)
				if err != nil {
					return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
				}
				querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, finalAttributes.SourceColumns)
			}
			return ctx.GetText(), querySpanResult, nil
		}
		panic("never reach here")
	case *parser.Case_expressionContext:
		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expr := range ctx.AllExpr() {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(expr)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
		}
		if v := ctx.AllSwitch_section(); len(v) > 0 {
			for _, switchSection := range v {
				_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(switchSection)
				if err != nil {
					return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", switchSection.GetText(), switchSection.GetStart().GetLine())
				}
				querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			}
			return ctx.GetText(), querySpanResult, nil
		}
		if v := ctx.AllSwitch_search_condition_section(); len(v) > 0 {
			for _, switchSearchConditionSection := range v {
				_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(switchSearchConditionSection)
				if err != nil {
					return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", switchSearchConditionSection.GetText(), switchSearchConditionSection.GetStart().GetLine())
				}
				querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			}
			return ctx.GetText(), querySpanResult, nil
		}
		panic("never reach here")
	case *parser.Switch_sectionContext:
		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if v := ctx.AllExpr(); len(v) > 0 {
			for _, expr := range v {
				_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(expr)
				if err != nil {
					return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
				}
				querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			}
			return ctx.GetText(), querySpanResult, nil
		}
		panic("never reach here")
	case *parser.Switch_search_condition_sectionContext:
		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if v := ctx.Search_condition(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			_, maskingAttributes, err = q.evalSnowSQLExprMaskingAttributes(ctx.Expr())
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Expr().GetText(), ctx.Expr().GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			return ctx.GetText(), querySpanResult, nil
		}
		panic("never reach here")
	case *parser.Search_conditionContext:
		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if v := ctx.Predicate(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Predicate().GetText(), ctx.Predicate().GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
		}
		if v := ctx.AllSearch_condition(); len(v) > 0 {
			for _, searchCondition := range v {
				_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(searchCondition)
				if err != nil {
					return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", searchCondition.GetText(), searchCondition.GetStart().GetLine())
				}
				querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			}
			return ctx.GetText(), querySpanResult, nil
		}
		panic("never reach here")
	case *parser.PredicateContext:
		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if v := ctx.AllExpr(); len(v) > 0 {
			for _, expr := range v {
				_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(expr)
				if err != nil {
					return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
				}
				querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			}
		}
		if v := ctx.Subquery(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
		}
		if v := ctx.Expr_list(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
		}
		return ctx.GetText(), querySpanResult, nil
	case *parser.SubqueryContext:
		fields, err := q.extractSnowsqlSensitiveFieldsQueryStatement(ctx.Query_statement())
		if err != nil {
			return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.GetText(), ctx.GetStart().GetLine())
		}
		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, querySpanResult := range fields.GetQuerySpanResult() {
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, querySpanResult.SourceColumns)
		}
		return ctx.GetText(), querySpanResult, nil
	case *parser.Primitive_expressionContext:
		if v := ctx.Id_(); v != nil {
			return q.evalSnowSQLExprMaskingAttributes(v)
		}
		return ctx.GetText(), base.QuerySpanResult{
			Name: ctx.GetText(),
		}, nil
	case *parser.Function_callContext:
		if v := ctx.Ranking_windowed_function(); v != nil {
			return q.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Aggregate_function(); v != nil {
			return q.evalSnowSQLExprMaskingAttributes(v)
		}
		if v := ctx.Object_name(); v != nil {
			return v.GetText(), base.QuerySpanResult{
				Name: ctx.GetText(),
			}, nil
		}
		if v := ctx.Expr_list(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		if v := ctx.Expr(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		panic("never reach here")
	case *parser.Aggregate_functionContext:
		if v := ctx.Expr_list(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		if ctx.STAR() != nil {
			return ctx.GetText(), base.QuerySpanResult{}, nil
		}
		if v := ctx.Expr(); v != nil {
			querySpanResult := base.QuerySpanResult{
				Name:          ctx.GetText(),
				SourceColumns: make(base.SourceColumnSet),
			}
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			_, maskingAttributes, err = q.evalSnowSQLExprMaskingAttributes(ctx.Order_by_clause())
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Order_by_clause().GetText(), ctx.Order_by_clause().GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			return ctx.GetText(), querySpanResult, nil
		}
		panic("never reach here")
	case *parser.Ranking_windowed_functionContext:
		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if v := ctx.Expr(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
		}
		if v := ctx.Over_clause(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			return ctx.GetText(), querySpanResult, nil
		}
		panic("never reach here")
	case *parser.Over_clauseContext:
		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if v := ctx.Partition_by(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			return ctx.GetText(), querySpanResult, nil
		}
		if v := ctx.Order_by_expr(); v != nil {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(v)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", v.GetText(), v.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
			return ctx.GetText(), querySpanResult, nil
		}
		panic("never reach here")
	case *parser.Partition_byContext:
		_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(ctx.Expr_list())
		if err != nil {
			return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Expr_list().GetText(), ctx.Expr_list().GetStart().GetLine())
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.Order_by_exprContext:
		_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(ctx.Expr_list_sorted())
		if err != nil {
			return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", ctx.Expr_list_sorted().GetText(), ctx.Expr_list_sorted().GetStart().GetLine())
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.Expr_listContext:
		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		allExpr := ctx.AllExpr()
		for _, expr := range allExpr {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(expr)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
		}
		return ctx.GetText(), querySpanResult, nil
	case *parser.Expr_list_sortedContext:
		querySpanResult := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		allExpr := ctx.AllExpr()
		for _, expr := range allExpr {
			_, maskingAttributes, err := q.evalSnowSQLExprMaskingAttributes(expr)
			if err != nil {
				return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the expression %q is sensitive near line %d", expr.GetText(), expr.GetStart().GetLine())
			}
			querySpanResult.SourceColumns, _ = base.MergeSourceColumnSet(querySpanResult.SourceColumns, maskingAttributes.SourceColumns)
		}
		return ctx.GetText(), querySpanResult, nil
	case *parser.Id_Context:
		normalizedColumnName := snowflake.NormalizeSnowSQLObjectNamePart(ctx)
		fieldInfo, err := q.snowflakeGetField("", "", "", normalizedColumnName)
		if err != nil {
			return "", base.QuerySpanResult{}, errors.Wrapf(err, "failed to check whether the column %q is sensitive near line %d", normalizedColumnName, ctx.GetStart().GetLine())
		}
		return fieldInfo.Name, fieldInfo, nil
	}
	panic("never reach here")
}

func (q *querySpanExtractor) extractSnowsqlSensitiveFieldsFromClause(ctx parser.IFrom_clauseContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	return q.extractSnowsqlSensitiveFieldsTableSources(ctx.Table_sources())
}

func (q *querySpanExtractor) extractSnowsqlSensitiveFieldsTableSources(ctx parser.ITable_sourcesContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}
	allTableSources := ctx.AllTable_source()

	var result base.TableSource
	// If there are multiple table sources, the default join type is CROSS JOIN.
	for _, tableSource := range allTableSources {
		candidatesTableSource, err := q.extractSnowsqlSensitiveFieldsTableSource(tableSource)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = candidatesTableSource
		} else {
			pseudoTable := &base.PseudoTable{
				Name:    "",
				Columns: append(result.GetQuerySpanResult(), candidatesTableSource.GetQuerySpanResult()...),
			}
			for _, column := range result.GetQuerySpanResult() {
				pseudoTable.Columns = append(pseudoTable.Columns, column)
			}
			result = pseudoTable
		}
	}
	return result, nil
}

func (q *querySpanExtractor) extractSnowsqlSensitiveFieldsTableSource(ctx parser.ITable_sourceContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}
	return q.extractSnowsqlSensitiveFieldsTableSourceItemJoined(ctx.Table_source_item_joined())
}

func (q *querySpanExtractor) extractSnowsqlSensitiveFieldsTableSourceItemJoined(ctx parser.ITable_source_item_joinedContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	var left base.TableSource
	var err error
	if ctx.Object_ref() != nil {
		left, err = q.extractSnowsqlSensitiveFieldsObjectRef(ctx.Object_ref())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields of the left part of the object ref near line %d", ctx.Object_ref().GetStart().GetLine())
		}
	}

	if ctx.Table_source_item_joined() != nil {
		left, err = q.extractSnowsqlSensitiveFieldsTableSourceItemJoined(ctx.Table_source_item_joined())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields of the left part of the table source item joined near line %d", ctx.Table_source_item_joined().GetStart().GetLine())
		}
	}

	for i, joinClause := range ctx.AllJoin_clause() {
		left, err = q.extractSnowsqlSensitiveFieldsJoinClause(joinClause, left)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields of the left part of the #%d join clause near line %d", i+1, joinClause.GetStart().GetLine())
		}
	}

	return left, nil
}

func (q *querySpanExtractor) extractSnowsqlSensitiveFieldsJoinClause(ctx parser.IJoin_clauseContext, left base.TableSource) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	// Snowflake has 6 types of join:
	// INNER JOIN, LEFT OUTER JOIN, RIGHT OUTER JOIN, FULL OUTER JOIN, CROSS JOIN, and NATURAL JOIN.
	// Only the result(column num) of NATURAL JOIN may be reduced.
	right, err := q.extractSnowsqlSensitiveFieldsObjectRef(ctx.Object_ref())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields of the right part of the JOIN near line %d", ctx.Object_ref().GetStart().GetLine())
	}
	if ctx.NATURAL() != nil {
		// We should remove all the duplicate columns in the result set.
		// For example, if the left part has columns [a, b, c], and the right part has columns [a, b, d],
		// then the result set of NATURAL JOIN should be [a, b, c, d].
		rightMap := make(map[string]bool)
		for _, rightColumn := range right.GetQuerySpanResult() {
			rightMap[rightColumn.Name] = true
		}
		var result []base.QuerySpanResult
		for _, leftColumn := range left.GetQuerySpanResult() {
			if _, ok := rightMap[leftColumn.Name]; ok {
				delete(rightMap, leftColumn.Name)
			}
			result = append(result, leftColumn)
		}
		for _, rightColumn := range right.GetQuerySpanResult() {
			if _, ok := rightMap[rightColumn.Name]; ok {
				result = append(result, rightColumn)
			}
		}
		return &base.PseudoTable{
			Name:    "",
			Columns: result,
		}, nil
	}

	// For other types of join, we should keep all the columns for the left part and the right part.
	var result []base.QuerySpanResult
	result = append(result, left.GetQuerySpanResult()...)
	result = append(result, right.GetQuerySpanResult()...)
	return &base.PseudoTable{
		Name:    "",
		Columns: result,
	}, nil
}

func (q *querySpanExtractor) extractSnowsqlSensitiveFieldsObjectRef(ctx parser.IObject_refContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	var result []base.QuerySpanResult

	if objectName := ctx.Object_name(); objectName != nil {
		_, tableSource, err := q.snowsqlFindTableSchema(objectName, q.connectedDB, "PUBLIC")
		if err != nil {
			return nil, err
		}
		return tableSource, nil
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
		tableSource, err := q.extractSnowsqlSensitiveFieldsQueryStatement(ctx.Subquery().Query_statement())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields of subquery near line %d", ctx.Subquery().GetStart().GetLine())
		}
		return tableSource, nil
	}

	// TODO(zp): Handle the flatten table.
	if ctx.Flatten_table() != nil {
		return nil, nil
	}

	if ctx.Pivot_unpivot() != nil {
		if v := ctx.Pivot_unpivot(); v.PIVOT() != nil {
			pivotColumnName := v.AllId_()[1]
			normalizedPivotColumnName := snowflake.NormalizeSnowSQLObjectNamePart(pivotColumnName)
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
			normalizedValueColumnName := snowflake.NormalizeSnowSQLObjectNamePart(valueColumnName)
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
				result = append(result, base.QuerySpanResult{
					Name:          literal.GetText(),
					SourceColumns: pivotColumnInOriginalResult.SourceColumns,
				})
			}
		} else if v := ctx.Pivot_unpivot(); v.UNPIVOT() != nil {
			var strippedColumnIndices []int
			var strippedColumnInOriginalResult []base.QuerySpanResult
			for idx, columnName := range v.Column_list().AllColumn_name() {
				normalizedColumnName := snowflake.NormalizeSnowSQLObjectNamePart(columnName.Id_())
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

			sourceColumns := make(base.SourceColumnSet)
			for _, field := range strippedColumnInOriginalResult {
				sourceColumns, _ = base.MergeSourceColumnSet(sourceColumns, field.SourceColumns)
			}

			valueColumnName := v.Id_(0)
			normalizedValueColumnName := snowflake.NormalizeSnowSQLObjectNamePart(valueColumnName)

			nameColumnName := v.Column_name().Id_()
			normalizedNameColumnName := snowflake.NormalizeSnowSQLObjectNamePart(nameColumnName)

			result = append(result, base.QuerySpanResult{
				Name:          normalizedNameColumnName,
				SourceColumns: make(base.SourceColumnSet),
			}, base.QuerySpanResult{
				Name:          normalizedValueColumnName,
				SourceColumns: sourceColumns,
			})
		}
	}

	// If the as alias is not nil, we should use the alias name to replace the original table name.
	if ctx.As_alias() != nil {
		id := ctx.As_alias().Alias().Id_()
		aliasName := snowflake.NormalizeSnowSQLObjectNamePart(id)
		return &base.PseudoTable{
			Name:    aliasName,
			Columns: result,
		}, nil
	}

	return &base.PseudoTable{
		Name:    "",
		Columns: result,
	}, nil
}

func (q *querySpanExtractor) snowsqlFindTableSchema(objectName parser.IObject_nameContext, normalizedFallbackDatabaseName, normalizedFallbackSchemaName string) (string, base.TableSource, error) {
	normalizedDatabaseName, normalizedSchemaName, normalizedTableName := normalizedObjectName(objectName, "", "")
	// For snowflake, we should find the table schema in ctes by ascending order.
	if normalizedDatabaseName == "" && normalizedSchemaName == "" {
		for _, cte := range q.ctes {
			if normalizedTableName == cte.Name {
				return normalizedDatabaseName, cte, nil
			}
		}
	}
	normalizedDatabaseName, normalizedSchemaName, normalizedTableName = normalizedObjectName(objectName, normalizedFallbackDatabaseName, normalizedFallbackSchemaName)
	allDatabases, err := q.l(q.ctx)
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to get all databases")
	}

	for _, databaseName := range allDatabases {
		if normalizedDatabaseName != "" && normalizedDatabaseName != databaseName {
			continue
		}
		_, database, err := q.f(q.ctx, normalizedDatabaseName)
		if err != nil {
			return "", nil, errors.Wrapf(err, "failed to get database %s", normalizedDatabaseName)
		}
		allSchemaNames := database.ListSchemaNames()
		for _, schemaSchema := range allSchemaNames {
			if normalizedSchemaName != "" && normalizedSchemaName != schemaSchema {
				continue
			}
			schema := database.GetSchema(normalizedSchemaName)
			if schema == nil {
				return "", nil, errors.Errorf(`schema %s.%s is not found`, normalizedDatabaseName, normalizedSchemaName)
			}
			allTableNames := schema.ListTableNames()
			for _, tableName := range allTableNames {
				if normalizedTableName != tableName {
					continue
				}
				tableSchema := schema.GetTable(normalizedTableName)
				if tableSchema == nil {
					return "", nil, errors.Errorf(`table %s.%s.%s is not found`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
				}
				columns := tableSchema.GetColumns()
				return normalizedDatabaseName, &base.PhysicalTable{
					Name: normalizedTableName,
					Columns: func() []string {
						var result []string
						for _, column := range columns {
							result = append(result, column.Name)
						}
						return result
					}(),
				}, nil
			}
		}
	}
	return "", nil, errors.Errorf(`table %s.%s.%s is not found`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
}

func (q *querySpanExtractor) getAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName string) ([]base.QuerySpanResult, error) {
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

	var result []base.QuerySpanResult

	if mask&maskDatabaseName == 0 && mask&maskSchemaName == 0 && mask&maskTableName != 0 {
		for _, cte := range q.ctes {
			if normalizedTableName == cte.Name {
				result = append(result, cte.GetQuerySpanResult()...)
				return result, nil
			}
		}
	}

	for _, tableSource := range q.tableSourcesFrom {
		if mask&maskDatabaseName != 0 && normalizedDatabaseName != tableSource.GetDatabaseName() {
			continue
		}
		if mask&maskSchemaName != 0 && normalizedSchemaName != tableSource.GetSchemaName() {
			continue
		}
		if mask&maskTableName != 0 && normalizedTableName != tableSource.GetTableName() {
			continue
		}
		result = append(result, tableSource.GetQuerySpanResult()...)
		return result, nil
	}

	return nil, errors.Errorf(`no matching table %q.%q.%q`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
}

// snowflakeGetField iterates through the tableSourcesFrom sequentially until we find the first matching object and return the column name, and returns the fieldInfo.
func (q *querySpanExtractor) snowflakeGetField(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName string) (base.QuerySpanResult, error) {
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
			return base.QuerySpanResult{}, errors.Errorf(`table name %s is specified without column name`, normalizedTableName)
		}
		mask |= maskTableName
	}
	if normalizedSchemaName != "" {
		if mask&maskTableName == 0 {
			return base.QuerySpanResult{}, errors.Errorf(`schema name %s is specified without table name`, normalizedSchemaName)
		}
		mask |= maskSchemaName
	}
	if normalizedDatabaseName != "" {
		if mask&maskSchemaName == 0 {
			return base.QuerySpanResult{}, errors.Errorf(`database name %s is specified without schema name`, normalizedDatabaseName)
		}
		mask |= maskDatabaseName
	}

	if mask == maskNone {
		return base.QuerySpanResult{}, errors.Errorf(`no object name is specified`)
	}

	// We just need to iterate through the tableSourcesFrom sequentially until we find the first matching object.

	// It is safe if there are two or more objects in the tableSourcesFrom have the same column name, because the executor
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

	for _, tableSource := range q.tableSourcesFrom {
		if mask&maskDatabaseName != 0 && normalizedDatabaseName != tableSource.GetDatabaseName() {
			continue
		}
		if mask&maskSchemaName != 0 && normalizedSchemaName != tableSource.GetSchemaName() {
			continue
		}
		if mask&maskTableName != 0 && normalizedTableName != tableSource.GetTableName() {
			continue
		}
		for _, field := range tableSource.GetQuerySpanResult() {
			if mask&maskColumnName != 0 && normalizedColumnName != field.Name {
				continue
			}
			return field, nil
		}
	}
	return base.QuerySpanResult{}, errors.Errorf(`no matching column %q.%q.%q.%q`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
}

func normalizedFullColumnName(ctx parser.IFull_column_nameContext) (normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName string) {
	if ctx.GetDb_name() != nil {
		normalizedDatabaseName = snowflake.NormalizeSnowSQLObjectNamePart(ctx.GetDb_name())
	}
	if ctx.GetSchema() != nil {
		normalizedSchemaName = snowflake.NormalizeSnowSQLObjectNamePart(ctx.GetSchema())
	}
	if ctx.GetTab_name() != nil {
		normalizedTableName = snowflake.NormalizeSnowSQLObjectNamePart(ctx.GetTab_name())
	}
	if ctx.GetCol_name() != nil {
		normalizedColumnName = snowflake.NormalizeSnowSQLObjectNamePart(ctx.GetCol_name())
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
		normalizedD := snowflake.NormalizeSnowSQLObjectNamePart(d)
		if normalizedD != "" {
			database = normalizedD
		}
	}
	parts = append(parts, database)

	schema := normalizedFallbackSchemaName
	if s := objectName.GetS(); s != nil {
		normalizedS := snowflake.NormalizeSnowSQLObjectNamePart(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}
	parts = append(parts, schema)

	normalizedO := snowflake.NormalizeSnowSQLObjectNamePart(objectName.GetO())
	parts = append(parts, normalizedO)

	return parts[0], parts[1], parts[2]
}
