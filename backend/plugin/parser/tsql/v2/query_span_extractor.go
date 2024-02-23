package v2

import (
	"context"
	"fmt"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"

	parsererror "github.com/bytebase/bytebase/backend/plugin/parser/errors"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

type querySpanExtractor struct {
	ctx context.Context

	connectedDB         string
	connectedSchema     string
	ignoreCaseSensitive bool

	f base.GetDatabaseMetadataFunc
	l base.ListDatabaseNamesFunc

	// Private fields.
	// ctes is used to record the common table expressions (CTEs) in the query.
	// It should be shrunk to the privious length while exiting the query scope.
	ctes []*base.PseudoTable

	// outerTableSources is used to record the outer table sources in the query.
	// It's used to resolve the column name in the correlated subquery.
	// outerTableSources []base.TableSource

	// tableSourcesFrom is the list of table sources from the FROM clause.
	tableSourcesFrom []base.TableSource
}

func newQuerySpanExtractor(connectedDB string, connectedSchema string, f base.GetDatabaseMetadataFunc, l base.ListDatabaseNamesFunc, ignoreCaseSensitive bool) *querySpanExtractor {
	return &querySpanExtractor{
		connectedDB:         connectedDB,
		connectedSchema:     connectedSchema,
		f:                   f,
		l:                   l,
		ignoreCaseSensitive: ignoreCaseSensitive,
	}
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx

	return nil, nil
}

func (extractor *querySpanExtractor) extractSensitiveFields(sql string) ([]base.QuerySpanResult, error) {
	result, err := tsql.ParseTSQL(sql)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse tsql")
	}
	if result == nil {
		return nil, nil
	}
	if result.Tree == nil {
		return nil, nil
	}

	// We assumes the caller had handled the statement type case,
	// so we only need to handle the determined statement type here.
	// In order to decrease the maintainance cost, we use listener
	// to handlet the select statement precisely.
	listener := &tsqlSelectOnlyListener{
		extractor: extractor,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, result.Tree)

	return listener.result, listener.err
}

type tsqlSelectOnlyListener struct {
	*parser.BaseTSqlParserListener

	extractor *querySpanExtractor
	result    []base.QuerySpanResult
	err       error
}

// EnterSelect_statement_standalone is called when production select_statement_standalone is entered.
func (listener *tsqlSelectOnlyListener) EnterDml_clause(ctx *parser.Dml_clauseContext) {
	if ctx.Select_statement_standalone() == nil {
		return
	}

	result, err := listener.extractor.extractTSqlSensitiveFieldsFromSelectStatementStandalone(ctx.Select_statement_standalone())
	if err != nil {
		l.err = err
		return
	}

	for _, field := range result {
		listener.result = append(listener.result, base.QuerySpanResult{
			Name:              field.Name,
			MaskingAttributes: field.MaskingAttributes,
		})
	}
}

// extractTSqlSensitiveFieldsFromSelectStatementStandalone extracts sensitive fields from select_statement_standalone.
func (q *querySpanExtractor) extractTSqlSensitiveFieldsFromSelectStatementStandalone(ctx parser.ISelect_statement_standaloneContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	if ctx.With_expression() != nil {
		allCommonTableExpression := ctx.With_expression().AllCommon_table_expression()
		// TSQL do not have `RECURSIVE` keyword, if we detect `UNION`, we will treat it as `RECURSIVE`.
		for _, commonTableExpression := range allCommonTableExpression {
			var result *base.PseudoTable
			var err error
			normalizedCTEName := tsql.NormalizeTSQLIdentifier(commonTableExpression.GetExpression_name())

			var anchorTable *base.PseudoTable
			// If statement has more than one UNION, the first one is the anchor, and the rest are recursive.
			recursiveCTE := false
			queryExpression := commonTableExpression.Select_statement().Query_expression()
			if queryExpression.Query_specification() != nil {
				anchorTable, err = q.extractTSqlSensitiveFieldsFromQuerySpecification(queryExpression.Query_specification())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields from `query_specification` in `query_expression`")
				}
				if allSQLUnions := queryExpression.AllSql_union(); len(allSQLUnions) > 0 {
					recursiveCTE = true
					for i := 0; i < len(allSQLUnions)-1; i++ {
						// For UNION operator, the number of the columns in the result set is the same, and will use the left part's column name.
						// So we only need to extract the sensitive fields of the recursiveTableSource part.
						recursiveTableSource, err := q.extractTSqlSensitiveFieldsFromQuerySpecification(allSQLUnions[i].Query_specification())
						if err != nil {
							return nil, errors.Wrapf(err, "failed to extract the %d set operator near line %d", i+1, allSQLUnions[i].GetStart().GetLine())
						}
						recursiveTableSourceQuerySpanResult := recursiveTableSource.GetQuerySpanResult()
						if len(anchorTable.GetQuerySpanResult()) != len(recursiveTableSourceQuerySpanResult) {
							return nil, errors.Wrapf(err, "the number of columns in the query statement nearly line %d returns %d fields, but %d set operator near line %d returns %d fields", ctx.GetStart().GetLine(), len(anchorTable.GetQuerySpanResult()), i+1, allSQLUnions[i].GetStart().GetLine(), len(recursiveTableSourceQuerySpanResult))
						}
						for j := range recursiveTableSourceQuerySpanResult {
							anchorTable.Columns[j].SourceColumns, _ = base.MergeSourceColumnSet(anchorTable.Columns[j].SourceColumns, recursiveTableSourceQuerySpanResult[j].SourceColumns)
						}
					}
				}
			} else if allQueryExpression := queryExpression.AllQuery_expression(); len(allQueryExpression) > 0 {
				if len(allQueryExpression) > 1 {
					recursiveCTE = true
				}
				anchorTable, err = q.extractTSqlSensitiveFieldsFromQueryExpression(allQueryExpression[0])
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields from `query_specification` in `query_expression`")
				}
			}
			if !recursiveCTE {
				result = anchorTable
			} else {
				tempCte := &base.PseudoTable{
					Name:    normalizedCTEName,
					Columns: anchorTable.GetQuerySpanResult(),
				}
				originalSize := len(q.ctes)
				q.ctes = append(q.ctes, tempCte)
				for {
					change := false
					if queryExpression.Query_specification() != nil && len(queryExpression.AllSql_union()) > 0 {
						recursiveTableSource, err := q.extractTSqlSensitiveFieldsFromQuerySpecification(queryExpression.AllSql_union()[len(queryExpression.AllSql_union())-1].Query_specification())
						if err != nil {
							return nil, errors.Wrapf(err, "failed to extract sensitive fields of the recursive clause of recursive CTE %q near line %d", normalizedCTEName, queryExpression.AllSql_union()[len(queryExpression.AllSql_union())-1].Query_specification().GetStart().GetLine())
						}
						recursiveTableSourceQuerySpanResult := recursiveTableSource.GetQuerySpanResult()
						if len(recursiveTableSourceQuerySpanResult) != len(tempCte.GetQuerySpanResult()) {
							return nil, errors.Wrapf(err, "recursive clause returns %d fields, but anchor clause returns %d fields in recursive CTE %q near line %d", len(recursiveTableSourceQuerySpanResult), len(tempCte.GetQuerySpanResult()), normalizedCTEName, queryExpression.AllSql_union()[len(queryExpression.AllSql_union())-1].Query_specification().GetStart().GetLine())
						}
						for i := 0; i < len(recursiveTableSourceQuerySpanResult); i++ {
							tempCte.Columns[i].SourceColumns, change = base.MergeSourceColumnSet(tempCte.Columns[i].SourceColumns, recursiveTableSourceQuerySpanResult[i].SourceColumns)
						}
					} else if allQueryExpression := queryExpression.AllQuery_expression(); len(allQueryExpression) > 1 {
						recursiveTableSource, err := q.extractTSqlSensitiveFieldsFromQueryExpression(allQueryExpression[len(allQueryExpression)-1])
						if err != nil {
							return nil, errors.Wrapf(err, "failed to extract sensitive fields of the recursive clause of recursive CTE %q near line %d", normalizedCTEName, allQueryExpression[len(allQueryExpression)-1].GetStart().GetLine())
						}
						recursiveTableSourceQuerySpanResult := recursiveTableSource.GetQuerySpanResult()
						if len(recursiveTableSourceQuerySpanResult) != len(tempCte.GetQuerySpanResult()) {
							return nil, errors.Wrapf(err, "recursive clause returns %d fields, but anchor clause returns %d fields in recursive CTE %q near line %d", len(recursiveTableSourceQuerySpanResult), len(tempCte.GetQuerySpanResult()), normalizedCTEName, allQueryExpression[len(allQueryExpression)-1].GetStart().GetLine())
						}
						for i := 0; i < len(recursiveTableSourceQuerySpanResult); i++ {
							tempCte.Columns[i].SourceColumns, change = base.MergeSourceColumnSet(tempCte.Columns[i].SourceColumns, recursiveTableSourceQuerySpanResult[i].SourceColumns)
						}
					}
					q.ctes = q.ctes[:originalSize]
					originalSize = len(q.ctes)
					if !change {
						break
					}
				}
				q.ctes = q.ctes[:originalSize]
				result = tempCte
			}
			if v := commonTableExpression.Column_name_list(); v != nil {
				if len(result.GetQuerySpanResult()) != len(v.AllId_()) {
					return nil, errors.Errorf("the number of column name list %d does not match the number of columns %d", len(v.AllId_()), len(result.GetQuerySpanResult()))
				}
				for i, columnName := range v.AllId_() {
					normalizedColumnName := tsql.NormalizeTSQLIdentifier(columnName)
					result.Columns[i].Name = normalizedColumnName
				}
			}
			// Append to the extractor.schemaInfo.DatabaseList
			q.ctes = append(q.ctes, result)
		}
	}

	return q.extractTSqlSensitiveFieldsFromSelectStatement(ctx.Select_statement())
}

// extractTSqlSensitiveFieldsFromSelectStatement extracts sensitive fields from select_statement.
func (extractor *querySpanExtractor) extractTSqlSensitiveFieldsFromSelectStatement(ctx parser.ISelect_statementContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	queryResult, err := extractor.extractTSqlSensitiveFieldsFromQueryExpression(ctx.Query_expression())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields from `query_expression` in `select_statement`")
	}

	return queryResult, nil
}

func (extractor *querySpanExtractor) extractTSqlSensitiveFieldsFromQueryExpression(ctx parser.IQuery_expressionContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	if ctx.Query_specification() != nil {
		anchor, err := extractor.extractTSqlSensitiveFieldsFromQuerySpecification(ctx.Query_specification())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields from `query_specification` in `query_expression`")
		}
		querySpanResult := anchor.GetQuerySpanResult()
		if allSQLUnions := ctx.AllSql_union(); len(allSQLUnions) > 0 {
			for i, sqlUnion := range allSQLUnions {
				// For UNION operator, the number of the columns in the result set is the same, and will use the left part's column name.
				// So we only need to extract the sensitive fields of the right part.
				right, err := extractor.extractTSqlSensitiveFieldsFromQuerySpecification(sqlUnion.Query_specification())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract the %d set operator near line %d", i+1, sqlUnion.GetStart().GetLine())
				}
				querySpanResult, err = unionTableSources(anchor, right)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to union the %d set operator near line %d", i+1, sqlUnion.GetStart().GetLine())
				}
				anchor.Columns = querySpanResult
			}
		}
		return anchor, nil
	}

	if allQueryExpressions := ctx.AllQuery_expression(); len(allQueryExpressions) > 0 {
		anchor, err := extractor.extractTSqlSensitiveFieldsFromQueryExpression(allQueryExpressions[0])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields from `query_specification` in `query_expression`")
		}
		querySpanResult := anchor.GetQuerySpanResult()
		for i := 1; i < len(allQueryExpressions); i++ {
			// For UNION operator, the number of the columns in the result set is the same, and will use the left part's column name.
			// So we only need to extract the sensitive fields of the right part.
			right, err := extractor.extractTSqlSensitiveFieldsFromQueryExpression(allQueryExpressions[i])
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract the %d set operator near line %d", i+1, allQueryExpressions[i].GetStart().GetLine())
			}
			querySpanResult, err = unionTableSources(anchor, right)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to union the %d set operator near line %d", i+1, allQueryExpressions[i].GetStart().GetLine())
			}
			anchor.Columns = querySpanResult
		}
		return anchor, nil
	}

	panic("never reach here")
}

func (extractor *querySpanExtractor) extractTSqlSensitiveFieldsFromQuerySpecification(ctx parser.IQuery_specificationContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	if from := ctx.GetFrom(); from != nil {
		fromFieldList, err := extractor.extractTSqlSensitiveFieldsFromTableSources(ctx.Table_sources())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields from `table_sources` in `query_specification`")
		}
		originalFromFieldList := len(extractor.tableSourcesFrom)
		extractor.tableSourcesFrom = append(extractor.tableSourcesFrom, fromFieldList...)
		defer func() {
			extractor.tableSourcesFrom = extractor.tableSourcesFrom[:originalFromFieldList]
		}()
	}

	result := &base.PseudoTable{}

	selectList := ctx.Select_list()
	for _, selectListElem := range selectList.AllSelect_list_elem() {
		if asterisk := selectListElem.Asterisk(); asterisk != nil {
			var normalizedDatabaseName, normalizedSchemaName, normalizedTableName string
			if tableName := asterisk.Table_name(); tableName != nil {
				normalizedDatabaseName, normalizedSchemaName, normalizedTableName = splitTableNameIntoNormalizedParts(tableName)
			}
			left, err := extractor.tsqlGetAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get all fields of table %s.%s.%s", normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
			}
			result.Columns = append(result.Columns, left...)
		} else if selectListElem.Udt_elem() != nil {
			// TODO(zp): handle the UDT.
			result.Columns = append(result.Columns, base.QuerySpanResult{
				Name:          fmt.Sprintf("UNSUPPORTED UDT %s", selectListElem.GetText()),
				SourceColumns: make(base.SourceColumnSet, 0),
			})
		} else if selectListElem.LOCAL_ID() != nil {
			// TODO(zp): handle the local variable, SELECT @a=id FROM blog.dbo.t1;
			result.Columns = append(result.Columns, base.QuerySpanResult{
				Name:          fmt.Sprintf("UNSUPPORTED LOCALID %s", selectListElem.GetText()),
				SourceColumns: make(base.SourceColumnSet, 0),
			})
		} else if expressionElem := selectListElem.Expression_elem(); expressionElem != nil {
			columnName, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expressionElem)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to check if the expression element is sensitive")
			}
			result = append(result, base.FieldInfo{
				Name:              columnName,
				MaskingAttributes: maskingAttributes,
			})
		}
	}

	return result, nil
}

func (q *querySpanExtractor) extractTSqlSensitiveFieldsFromTableSources(ctx parser.ITable_sourcesContext) ([]base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	var allTableSources []parser.ITable_sourceContext
	if v := ctx.Non_ansi_join(); v != nil {
		allTableSources = v.GetSource()
	} else if len(ctx.AllTable_source()) != 0 {
		allTableSources = ctx.GetSource()
	}

	var result []base.TableSource
	// If there are multiple table sources, the default join type is CROSS JOIN.
	for _, tableSource := range allTableSources {
		ts, err := q.extractTSqlSensitiveFieldsFromTableSource(tableSource)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields from `table_source` in `table_sources`")
		}
		result = append(result, ts)
	}
	return result, nil
}

func (extractor *querySpanExtractor) extractTSqlSensitiveFieldsFromTableSource(ctx parser.ITable_sourceContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	var columns []base.QuerySpanResult
	anchor, err := extractor.extractTSqlSensitiveFieldsFromTableSourceItem(ctx.Table_source_item())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields from `table_source_item` in `table_source`")
	}
	columns = append(columns, anchor.GetQuerySpanResult()...)

	if allJoinParts := ctx.AllJoin_part(); len(allJoinParts) > 0 {
		// https://learn.microsoft.com/en-us/sql/relational-databases/performance/joins?view=sql-server-ver16
		for _, joinPart := range allJoinParts {
			if joinOn := joinPart.Join_on(); joinOn != nil {
				right, err := extractor.extractTSqlSensitiveFieldsFromTableSource(joinOn.Table_source())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields from `table_source` in `join_on`")
				}
				columns = append(columns, right.GetQuerySpanResult()...)
			}
			if crossJoin := joinPart.Cross_join(); crossJoin != nil {
				right, err := extractor.extractTSqlSensitiveFieldsFromTableSourceItem(crossJoin.Table_source_item())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields from `table_source` in `cross_join`")
				}
				columns = append(columns, right.GetQuerySpanResult()...)
			}
			if apply := joinPart.Apply_(); apply != nil {
				right, err := extractor.extractTSqlSensitiveFieldsFromTableSourceItem(apply.Table_source_item())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields from `table_source` in `apply`")
				}
				columns = append(columns, right.GetQuerySpanResult()...)
			}
			// TODO(zp): handle pivot and unpivot.
			if pivot := joinPart.Pivot(); pivot != nil {
				return nil, errors.New("pivot is not supported yet")
			}
			if unpivot := joinPart.Unpivot(); unpivot != nil {
				return nil, errors.New("unpivot is not supported yet")
			}
		}
	}

	return &base.PseudoTable{
		Columns: columns,
	}, nil
}

// extractTSqlSensitiveFieldsFromTableSourceItem extracts sensitive fields from table source item.
func (q *querySpanExtractor) extractTSqlSensitiveFieldsFromTableSourceItem(ctx parser.ITable_source_itemContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	var result base.TableSource
	var err error
	// TODO(zp): handle other cases likes ROWSET_FUNCTION.
	if ctx.Full_table_name() != nil {
		result, err = q.tsqlFindTableSchema(ctx.Full_table_name())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find table schema for full table name")
		}
	} else if ctx.Derived_table() != nil {
		result, err = q.extractTSqlSensitiveFieldsFromDerivedTable(ctx.Derived_table())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract table source from derived table")
		}
	} else if ctx.Table_source() != nil {
		return q.extractTSqlSensitiveFieldsFromTableSource(ctx.Table_source())
	} else {
		return nil, &parsererror.TypeNotSupportedError{
			Err:  errors.Errorf("only full table name in table source item is supported"),
			Type: fmt.Sprintf("%v", ctx),
		}
	}

	// If there are as_table_alias, we should patch the table name to the alias name, and reset the schema and database.
	// For example:
	// SELECT t1.id FROM blog.dbo.t1 AS TT1; -- The multi-part identifier "t1.id" could not be bound.
	// SELECT TT1.id FROM blog.dbo.t1 AS TT1; -- OK
	if asTableAlias := ctx.As_table_alias(); asTableAlias != nil {
		asName := tsql.NormalizeTSQLIdentifier(asTableAlias.Table_alias().Id_())

		for i := 0; i < len(result); i++ {
			result[i].Table = asName
			result[i].Schema = ""
			result[i].Database = ""
		}
	}

	if columnAliasList := ctx.Column_alias_list(); columnAliasList != nil {
		allColumnAlias := columnAliasList.AllColumn_alias()
		if len(allColumnAlias) != len(result) {
			return nil, errors.Errorf("the number of column alias %d does not match the number of columns %d", len(allColumnAlias), len(result))
		}
		for i := 0; i < len(result); i++ {
			if allColumnAlias[i].Id_() != nil {
				result[i].Name = NormalizeTSQLIdentifier(allColumnAlias[i].Id_())
				continue
			} else if allColumnAlias[i].STRING() != nil {
				result[i].Name = allColumnAlias[i].STRING().GetText()
				continue
			}
			panic("never reach here")
		}
	}

	return result, nil
}

func (extractor *querySpanExtractor) extractTSqlSensitiveFieldsFromDerivedTable(ctx parser.IDerived_tableContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	allSubquery := ctx.AllSubquery()
	if len(allSubquery) > 0 {
		left, err := extractor.extractTSqlSensitiveFieldsFromSubquery(allSubquery[0])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields from `subquery` in `derived_table`")
		}
		for i := 1; i < len(allSubquery); i++ {
			// For UNION operator, the number of the columns in the result set is the same, and will use the left part's column name.
			// So we only need to extract the sensitive fields of the right part.
			right, err := extractor.extractTSqlSensitiveFieldsFromSubquery(allSubquery[i])
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract the %d set operator near line %d", i+1, allSubquery[i].GetStart().GetLine())
			}
			if len(left) != len(right) {
				return nil, errors.Wrapf(err, "the number of columns in the derived table statement nearly line %d returns %d fields, but %d set operator near line %d returns %d fields", ctx.GetStart().GetLine(), len(left), i+1, allSubquery[i].GetStart().GetLine(), len(right))
			}
			for i := range right {
				left[i].MaskingAttributes.TransmittedBy(right[i].MaskingAttributes)
			}
		}
		return left, nil
	}

	if tableValueConstructor := ctx.Table_value_constructor(); tableValueConstructor != nil {
		return extractor.extractTSqlSensitiveFieldsFromTableValueConstructor(tableValueConstructor)
	}

	panic("never reach here")
}

func (extractor *querySpanExtractor) extractTSqlSensitiveFieldsFromTableValueConstructor(ctx parser.ITable_value_constructorContext) ([]base.FieldInfo, error) {
	if allExpressionList := ctx.AllExpression_list_(); len(allExpressionList) > 0 {
		// The number of expression in each expression list should be the same.
		// But we do not check, just use the first one, and engine will throw a compilation error if the number of expressions are not the same.
		expressionList := allExpressionList[0]
		var result []base.FieldInfo
		for _, expression := range expressionList.AllExpression() {
			columnName, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to check if the expression is sensitive")
			}
			result = append(result, base.FieldInfo{
				Name:              columnName,
				MaskingAttributes: maskingAttributes,
			})
		}
		return result, nil
	}
	panic("never reach here")
}

func (extractor *querySpanExtractor) extractTSqlSensitiveFieldsFromSubquery(ctx parser.ISubqueryContext) ([]base.FieldInfo, error) {
	return extractor.extractTSqlSensitiveFieldsFromSelectStatement(ctx.Select_statement())
}

func (extractor *querySpanExtractor) tsqlFindTableSchema(fullTableName parser.IFull_table_nameContext) (base.TableSource, error) {
	normalizedLinkedServer, normalizedDatabaseName, normalizedSchemaName, normalizedTableName := normalizeFullTableName(fullTableName, "" /* Linked Server Name */, "", "")
	if normalizedLinkedServer != "" {
		// TODO(zp): How do we handle the linked server?
		return nil, errors.Errorf("linked server is not supported yet, but found %q", fullTableName.GetText())
	}

	// For SQL Server, the cte will shadow the physical tables, so we check the ctes first,
	// also, we record the cte in ascending order, so we should check the ctes in descending order
	// to find the nearest match.
	if normalizedDatabaseName == "" && normalizedSchemaName == "" {
		for _, cte := range extractor.ctes {
			if extractor.isIdentifierEqual(normalizedTableName, cte.Name) {
				return cte, nil
			}
		}
	}

	normalizedLinkedServer, normalizedDatabaseName, normalizedSchemaName, normalizedTableName = normalizeFullTableName(fullTableName, "" /* Linked Server Name */, extractor.connectedDB, extractor.connectedSchema)
	if normalizedLinkedServer != "" {
		// TODO(zp): How do we handle the linked server?
		return nil, errors.Errorf("linked server is not supported yet, but found %q", fullTableName.GetText())
	}
	allDatabases, err := extractor.l(extractor.ctx)
	if err != nil {
		return nil, errors.Errorf("failed to list databases: %w", err)
	}

	for _, databaseName := range allDatabases {
		if normalizedDatabaseName != "" && !extractor.isIdentifierEqual(normalizedDatabaseName, databaseName) {
			continue
		}
		_, database, err := extractor.f(extractor.ctx, databaseName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database %s metadata", databaseName)
		}

		allSchemaNames := database.ListSchemaNames()
		for _, schemaName := range allSchemaNames {
			if normalizedSchemaName != "" && !extractor.isIdentifierEqual(normalizedSchemaName, schemaName) {
				continue
			}
			schemaSchema := database.GetSchema(schemaName)
			allTableNames := schemaSchema.ListTableNames()
			for _, tableName := range allTableNames {
				if !extractor.isIdentifierEqual(normalizedTableName, tableName) {
					continue
				}
				table := schemaSchema.GetTable(tableName)
				physicalTableSource := &base.PhysicalTable{
					Server:   "",
					Database: databaseName,
					Schema:   schemaName,
					Name:     tableName,
					Columns: func() []string {
						var result []string
						for _, column := range table.GetColumns() {
							result = append(result, column.Name)
						}
						return result
					}(),
				}
				return normalizedDatabaseName, physicalTableSource, nil
			}
		}
	}
	return nil, &parsererror.ResourceNotFoundError{
		Database: &normalizedDatabaseName,
		Schema:   &normalizedSchemaName,
		Table:    &normalizedTableName,
	}
}

// splitTableNameIntoNormalizedParts splits the table name into normalized 3 parts: database, schema, table.
func splitTableNameIntoNormalizedParts(tableName parser.ITable_nameContext) (string, string, string) {
	var database string
	if d := tableName.GetDatabase(); d != nil {
		normalizedD := NormalizeTSQLIdentifier(d)
		if normalizedD != "" {
			database = normalizedD
		}
	}

	var schema string
	if s := tableName.GetSchema(); s != nil {
		normalizedS := NormalizeTSQLIdentifier(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}

	var table string
	if t := tableName.GetTable(); t != nil {
		normalizedT := NormalizeTSQLIdentifier(t)
		if normalizedT != "" {
			table = normalizedT
		}
	}
	return database, schema, table
}

// normalizeFullTableName normalizes the each part of the full table name, returns (linkedServer, database, schema, table).
func normalizeFullTableName(fullTableName parser.IFull_table_nameContext, normalizedFallbackLinkedServerName, normalizedFallbackDatabaseName, normalizedFallbackSchemaName string) (string, string, string, string) {
	if fullTableName == nil {
		return "", "", "", ""
	}
	// TODO(zp): unify here and the related code in sql_service.go
	linkedServer := normalizedFallbackLinkedServerName
	if server := fullTableName.GetLinkedServer(); server != nil {
		linkedServer = tsql.NormalizeTSQLIdentifier(server)
	}

	database := normalizedFallbackDatabaseName
	if d := fullTableName.GetDatabase(); d != nil {
		normalizedD := tsql.NormalizeTSQLIdentifier(d)
		if normalizedD != "" {
			database = normalizedD
		}
	}

	schema := normalizedFallbackSchemaName
	if s := fullTableName.GetSchema(); s != nil {
		normalizedS := tsql.NormalizeTSQLIdentifier(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}

	var table string
	if t := fullTableName.GetTable(); t != nil {
		normalizedT := tsql.NormalizeTSQLIdentifier(t)
		if normalizedT != "" {
			table = normalizedT
		}
	}

	return linkedServer, database, schema, table
}

func (extractor *querySpanExtractor) tsqlGetAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName string) ([]base.QuerySpanResult, error) {
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

	for _, tableSource := range extractor.tableSourcesFrom {
		if mask&maskDatabaseName != 0 && !extractor.isIdentifierEqual(normalizedDatabaseName, tableSource.GetDatabaseName()) {
			continue
		}
		if mask&maskSchemaName != 0 && !extractor.isIdentifierEqual(normalizedSchemaName, tableSource.GetSchemaName()) {
			continue
		}
		if mask&maskTableName != 0 && !extractor.isIdentifierEqual(normalizedTableName, tableSource.GetTableName()) {
			continue
		}
		return tableSource.GetQuerySpanResult(), nil
	}

	return nil, errors.Errorf(`no matching table %q.%q.%q`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
}

func (extractor *querySpanExtractor) tsqlIsFullColumnNameSensitive(ctx parser.IFull_column_nameContext) (base.FieldInfo, error) {
	normalizedLinkedServer, normalizedDatabaseName, normalizedSchemaName, normalizedTableName := normalizeFullTableName(ctx.Full_table_name(), "", "", "")
	if normalizedLinkedServer != "" {
		return base.FieldInfo{}, errors.Errorf("linked server is not supported yet, but found %q", ctx.GetText())
	}
	normalizedColumnName := tsql.NormalizeTSQLIdentifier(ctx.Id_())

	return extractor.tsqlIsFieldSensitive(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
}

func (extractor *querySpanExtractor) tsqlIsFieldSensitive(normalizedDatabaseName string, normalizedSchemaName string, normalizedTableName string, normalizedColumnName string) (base.FieldInfo, error) {
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
	for _, field := range extractor.tableSourcesFrom {
		if mask&maskDatabaseName != 0 && !extractor.isIdentifierEqual(normalizedDatabaseName, field.Database) {
			continue
		}
		if mask&maskSchemaName != 0 && !extractor.isIdentifierEqual(normalizedSchemaName, field.Schema) {
			continue
		}
		if mask&maskTableName != 0 && !extractor.isIdentifierEqual(normalizedTableName, field.Table) {
			continue
		}
		if mask&maskColumnName != 0 && !extractor.isIdentifierEqual(normalizedColumnName, field.Name) {
			continue
		}
		return field, nil
	}
	return base.FieldInfo{}, errors.Errorf(`no matching column %q.%q.%q.%q`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
}

// isIdentifierEqual compares the identifier with the given normalized parts, returns true if they are equal.
// It will consider the case sensitivity based on the current database.
func (extractor *querySpanExtractor) isIdentifierEqual(a, b string) bool {
	if !extractor.ignoreCaseSensitive {
		return a == b
	}
	if len(a) != len(b) {
		return false
	}
	runeA, runeB := []rune(a), []rune(b)
	for i := 0; i < len(runeA); i++ {
		if unicode.ToLower(runeA[i]) != unicode.ToLower(runeB[i]) {
			return false
		}
	}
	return true
}

// evalExpressionElemMaskingLevel returns true if the expression element is sensitive, and returns the column name.
// It is the closure of the expression_elemContext, it will recursively check the sub expression element.
func (extractor *querySpanExtractor) evalExpressionElemMaskingLevel(ctx antlr.RuleContext) (string, base.MaskingAttributes, error) {
	if ctx == nil {
		return "", base.NewDefaultMaskingAttributes(), nil
	}
	switch ctx := ctx.(type) {
	case *parser.Expression_elemContext:
		columName, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the expression element is sensitive")
		}
		if columnAlias := ctx.Column_alias(); columnAlias != nil {
			columName = tsql.NormalizeTSQLIdentifier(columnAlias.Id_())
		} else if asColumnAlias := ctx.As_column_alias(); asColumnAlias != nil {
			columName = tsql.NormalizeTSQLIdentifier(asColumnAlias.Column_alias().Id_())
		}
		return columName, maskingAttributes, nil
	case *parser.ExpressionContext:
		if ctx.Primitive_expression() != nil {
			return extractor.evalExpressionElemMaskingLevel(ctx.Primitive_expression())
		}
		if ctx.Function_call() != nil {
			return extractor.evalExpressionElemMaskingLevel(ctx.Function_call())
		}
		finalAttributes := base.NewDefaultMaskingAttributes()
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the expression is sensitive")
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		if valueCall := ctx.Value_call(); valueCall != nil {
			return extractor.evalExpressionElemMaskingLevel(valueCall)
		}
		if queryCall := ctx.Query_call(); queryCall != nil {
			return extractor.evalExpressionElemMaskingLevel(queryCall)
		}
		if existCall := ctx.Exist_call(); existCall != nil {
			return extractor.evalExpressionElemMaskingLevel(existCall)
		}
		if modifyCall := ctx.Modify_call(); modifyCall != nil {
			return extractor.evalExpressionElemMaskingLevel(modifyCall)
		}
		if hierarchyIDCall := ctx.Hierarchyid_call(); hierarchyIDCall != nil {
			return extractor.evalExpressionElemMaskingLevel(hierarchyIDCall)
		}
		if caseExpression := ctx.Case_expression(); caseExpression != nil {
			return extractor.evalExpressionElemMaskingLevel(caseExpression)
		}
		if fullColumnName := ctx.Full_column_name(); fullColumnName != nil {
			return extractor.evalExpressionElemMaskingLevel(fullColumnName)
		}
		if bracketExpression := ctx.Bracket_expression(); bracketExpression != nil {
			return extractor.evalExpressionElemMaskingLevel(bracketExpression)
		}
		if unaryOperationExpression := ctx.Unary_operator_expression(); unaryOperationExpression != nil {
			return extractor.evalExpressionElemMaskingLevel(unaryOperationExpression)
		}
		if overClause := ctx.Over_clause(); overClause != nil {
			return extractor.evalExpressionElemMaskingLevel(overClause)
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Unary_operator_expressionContext:
		if expression := ctx.Expression(); expression != nil {
			return extractor.evalExpressionElemMaskingLevel(expression)
		}
		return ctx.GetText(), base.NewDefaultMaskingAttributes(), nil
	case *parser.Bracket_expressionContext:
		if expression := ctx.Expression(); expression != nil {
			return extractor.evalExpressionElemMaskingLevel(expression)
		}
		if subquery := ctx.Subquery(); subquery != nil {
			return extractor.evalExpressionElemMaskingLevel(subquery)
		}
		return ctx.GetText(), base.NewDefaultMaskingAttributes(), nil
	case *parser.Case_expressionContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the case_expression is sensitive")
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		if allSwitchSections := ctx.AllSwitch_section(); len(allSwitchSections) > 0 {
			for _, switchSection := range allSwitchSections {
				_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(switchSection)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the case_expression is sensitive")
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		if allSwitchSearchConditionSections := ctx.AllSwitch_search_condition_section(); len(allSwitchSearchConditionSections) > 0 {
			for _, switchSearchConditionSection := range allSwitchSearchConditionSections {
				_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(switchSearchConditionSection)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the case_expression is sensitive")
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Switch_sectionContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the switch_setion is sensitive")
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Switch_search_condition_sectionContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if searchCondition := ctx.Search_condition(); searchCondition != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(searchCondition)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the switch_search_condition_section is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if expression := ctx.Expression(); expression != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the switch_search_condition_section is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Search_conditionContext:
		if predicate := ctx.Predicate(); predicate != nil {
			return extractor.evalExpressionElemMaskingLevel(predicate)
		}
		finalAttributes := base.NewDefaultMaskingAttributes()
		if allSearchConditions := ctx.AllSearch_condition(); len(allSearchConditions) > 0 {
			for _, searchCondition := range allSearchConditions {
				_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(searchCondition)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the search_condition is sensitive")
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.PredicateContext:
		if subquery := ctx.Subquery(); subquery != nil {
			return extractor.evalExpressionElemMaskingLevel(subquery)
		}
		if freeTextPredicate := ctx.Freetext_predicate(); freeTextPredicate != nil {
			return extractor.evalExpressionElemMaskingLevel(freeTextPredicate)
		}

		finalAttributes := base.NewDefaultMaskingAttributes()
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the predicate is sensitive")
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expressionList)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the predicate is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Freetext_predicateContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the freetext_predicate is sensitive")
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		if allCullColumnName := ctx.AllFull_column_name(); len(allCullColumnName) > 0 {
			for _, fullColumnName := range allCullColumnName {
				_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(fullColumnName)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the freetext_predicate is sensitive")
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.SubqueryContext:
		// For subquery, we clone the current extractor, reset the from list, but keep the cte, and then extract the sensitive fields from the subquery
		cloneExtractor := &querySpanExtractor{
			connectedDB:     extractor.connectedDB,
			connectedSchema: extractor.connectedSchema,
			f:               extractor.f,
			l:               extractor.l,
			// outerTableSources: extractor.outerTableSources,
			ctes: extractor.ctes,
		}
		fieldInfo, err := cloneExtractor.extractTSqlSensitiveFieldsFromSubquery(ctx)
		// The expect behavior is the fieldInfo contains only one field, which is the column name,
		// but in order to do not block user, we just return isSensitive if there is any sensitive field.
		// return fieldInfo[0].sensitive, err
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the subquery is sensitive")
		}
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, field := range fieldInfo {
			finalAttributes.TransmittedByInExpression(field.MaskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Hierarchyid_callContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the hierarchyid_call is sensitive")
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Query_callContext:
		return ctx.GetText(), base.NewDefaultMaskingAttributes(), nil
	case *parser.Exist_callContext:
		return ctx.GetText(), base.NewDefaultMaskingAttributes(), nil
	case *parser.Modify_callContext:
		return ctx.GetText(), base.NewDefaultMaskingAttributes(), nil
	case *parser.Value_callContext:
		return ctx.GetText(), base.NewDefaultMaskingAttributes(), nil
	case *parser.Primitive_expressionContext:
		if ctx.Primitive_constant() != nil {
			_, sensitive, err := extractor.evalExpressionElemMaskingLevel(ctx.Primitive_constant())
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the primitive constant is sensitive")
			}
			return ctx.GetText(), sensitive, nil
		}
		panic("never reach here")
	case *parser.Primitive_constantContext:
		return ctx.GetText(), base.NewDefaultMaskingAttributes(), nil
	case *parser.Function_callContext:
		// In parser.g4, the function_callContext is defined as:
		// 	function_call
		// : ranking_windowed_function                         #RANKING_WINDOWED_FUNC
		// | aggregate_windowed_function                       #AGGREGATE_WINDOWED_FUNC
		// ...
		// ;
		// So it will be parsed as RANKING_WINDOWED_FUNC, AGGREGATE_WINDOWED_FUNC, etc.
		// We just need to check the first token to see if it is a sensitive function.
		panic("never reach here")
	case *parser.RANKING_WINDOWED_FUNCContext:
		return extractor.evalExpressionElemMaskingLevel(ctx.Ranking_windowed_function())
	case *parser.Ranking_windowed_functionContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if overClause := ctx.Over_clause(); overClause != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(overClause)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the ranking_windowed_function is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if expression := ctx.Expression(); expression != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the ranking_windowed_function is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Over_clauseContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression_list_())
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the over_clause is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if orderByClause := ctx.Order_by_clause(); orderByClause != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(orderByClause)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the over_clause is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if rowOrRangeClause := ctx.Row_or_range_clause(); rowOrRangeClause != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(rowOrRangeClause)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the over_clause is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Expression_list_Context:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the expression_list is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Order_by_clauseContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, orderByExpression := range ctx.GetOrder_bys() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(orderByExpression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the order_by_clause is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Order_by_expressionContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the order_by_expression is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.Row_or_range_clauseContext:
		if windowFrameExtent := ctx.Window_frame_extent(); windowFrameExtent != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(windowFrameExtent)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the row_or_range_clause is sensitive")
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		panic("never reach here")
	case *parser.Window_frame_extentContext:
		if windowFramePreceding := ctx.Window_frame_preceding(); windowFramePreceding != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(windowFramePreceding)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the window_frame_extent is sensitive")
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		if windowFrameBounds := ctx.AllWindow_frame_bound(); len(windowFrameBounds) > 0 {
			finalAttributes := base.NewDefaultMaskingAttributes()
			for _, windowFrameBound := range windowFrameBounds {
				_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(windowFrameBound)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the window_frame_extent is sensitive")
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		panic("never reach here")
	case *parser.Window_frame_boundContext:
		if preceding := ctx.Window_frame_preceding(); preceding != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(preceding)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the window_frame_bound is sensitive")
			}
			return ctx.GetText(), maskingAttributes, nil
		} else if following := ctx.Window_frame_following(); following != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(following)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the window_frame_bound is sensitive")
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		panic("never reach here")
	case *parser.Window_frame_precedingContext:
		return ctx.GetText(), base.NewDefaultMaskingAttributes(), nil
	case *parser.Window_frame_followingContext:
		return ctx.GetText(), base.NewDefaultMaskingAttributes(), nil
	case *parser.AGGREGATE_WINDOWED_FUNCContext:
		return extractor.evalExpressionElemMaskingLevel(ctx.Aggregate_windowed_function())
	case *parser.Aggregate_windowed_functionContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if allDistinctExpression := ctx.All_distinct_expression(); allDistinctExpression != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(allDistinctExpression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the aggregate_windowed_function is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if overClause := ctx.Over_clause(); overClause != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(overClause)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the aggregate_windowed_function is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if expression := ctx.Expression(); expression != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the aggregate_windowed_function is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expressionList)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the aggregate_windowed_function is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.All_distinct_expressionContext:
		_, sensitive, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the all_distinct_expression is sensitive")
		}
		return ctx.GetText(), sensitive, nil
	case *parser.ANALYTIC_WINDOWED_FUNCContext:
		return extractor.evalExpressionElemMaskingLevel(ctx.Analytic_windowed_function())
	case *parser.Analytic_windowed_functionContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the analytic_windowed_function is sensitive")
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		if overClause := ctx.Over_clause(); overClause != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(overClause)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the analytic_windowed_function is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expressionList)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the analytic_windowed_function is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if orderByClause := ctx.Order_by_clause(); orderByClause != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(orderByClause)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the analytic_windowed_function is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.BUILT_IN_FUNCContext:
		return extractor.evalExpressionElemMaskingLevel(ctx.Built_in_functions())
	case *parser.APP_NAMEContext:
		return ctx.GetText(), base.NewDefaultMaskingAttributes(), nil
	case *parser.APPLOCK_MODEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the applock_mode is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.APPLOCK_TESTContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the applock_test is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.ASSEMBLYPROPERTYContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the assemblyproperty is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.COL_LENGTHContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the col_length is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.COL_NAMEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the col_name is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.COLUMNPROPERTYContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the columnproperty is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.DATABASEPROPERTYEXContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the databasepropertyex is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.DB_IDContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the db_id is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.DB_NAMEContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the db_name is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.FILE_IDContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the file_id is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.FILE_IDEXContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the file_idex is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.FILE_NAMEContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the file_name is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.FILEGROUP_IDContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the filegroup_id is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.FILEGROUP_NAMEContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the filegroup_name is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.FILEGROUPPROPERTYContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the filegroupproperty is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.FILEPROPERTYContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the fileproperty is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.FILEPROPERTYEXContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the filepropertyex is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.FULLTEXTCATALOGPROPERTYContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the fulltextcatalogproperty is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.FULLTEXTSERVICEPROPERTYContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the fulltextserviceproperty is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.INDEX_COLContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the index_col is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.INDEXKEY_PROPERTYContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the indexkey_property is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.INDEXPROPERTYContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the indexproperty is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.OBJECT_DEFINITIONContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the object_definition is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.OBJECT_IDContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the object_id is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.OBJECT_NAMEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the object_name is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.OBJECT_SCHEMA_NAMEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the object_schema_name is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.OBJECTPROPERTYContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the objectproperty is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.OBJECTPROPERTYEXContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the objectpropertyex is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.PARSENAMEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the parsename is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.SCHEMA_IDContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the schema_id is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.SCHEMA_NAMEContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the schema_name is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.SERVERPROPERTYContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the serverproperty is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.STATS_DATEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the stats_date is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.TYPE_IDContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the type_id is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.TYPE_NAMEContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the type_name is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.TYPEPROPERTYContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the typeproperty is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.ASCIIContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the ascii is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.CHARContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the char is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.CHARINDEXContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the charindex is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.CONCATContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the concat is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.CONCAT_WSContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the concat_ws is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.DIFFERENCEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the difference is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.FORMATContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the format is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.LEFTContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the left is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.LENContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the len is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.LOWERContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the lower is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.LTRIMContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the ltrim is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.NCHARContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the nchar is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.PATINDEXContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the patindex is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.QUOTENAMEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the quotename is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.REPLACEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the replace is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.REPLICATEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the replicate is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.REVERSEContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the reverse is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.RIGHTContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the right is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.RTRIMContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the rtrim is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.SOUNDEXContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the soundex is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.SPACEContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the space is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.STRContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the str is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.STRINGAGGContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the stringagg is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.STRING_ESCAPEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the string_escape is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.STUFFContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the stuff is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.SUBSTRINGContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the substring is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.TRANSLATEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the translate is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.TRIMContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the trim is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.UNICODEContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the unicode is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.UPPERContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the upper is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.BINARY_CHECKSUMContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the binary_checksum is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.CHECKSUMContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the checksum is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), base.NewDefaultMaskingAttributes(), nil
	case *parser.COMPRESSContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the compress is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.DECOMPRESSContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the decompress is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.FORMATMESSAGEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the formatmessage is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.ISNULLContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the isnull is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.ISNUMERICContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the isnumeric is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.CASTContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the cast is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.TRY_CASTContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the try_cast is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.CONVERTContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the convert is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.COALESCEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression_list_())
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the coalesce is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.CURSOR_STATUSContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the cursor_status is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.CERT_IDContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the cert_id is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.DATALENGTHContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the datalength is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.IDENT_CURRENTContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the  ident_current is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.IDENT_INCRContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the  ident_incr is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.IDENT_SEEDContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the  ident_seed is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.SQL_VARIANT_PROPERTYContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the sql_variant_property is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.DATE_BUCKETContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the date_bucket is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.DATEADDContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the dateadd is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.DATEDIFFContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the datediff is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.DATEDIFF_BIGContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the datediff_big is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.DATEFROMPARTSContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the datefromparts is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.DATENAMEContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the datename is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.DATEPARTContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the datepart is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.DATETIME2FROMPARTSContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the datetime2fromparts is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.DATETIMEFROMPARTSContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the datetimefromparts is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.DATETIMEOFFSETFROMPARTSContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the datetimeoffsetfromparts is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.DATETRUNCContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the datetrunc is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.DAYContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the day is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.EOMONTHContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the eomonth is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.ISDATEContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the isdate is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.MONTHContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the month is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.SMALLDATETIMEFROMPARTSContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the smalldatetimefromparts is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.SWITCHOFFSETContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the switchoffset is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.TIMEFROMPARTSContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the timefromparts is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.TODATETIMEOFFSETContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the todatetimeoffset is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.YEARContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the year is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.NULLIFContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the nullif is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.PARSEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the parse is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.IIFContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the iif is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.ISJSONContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the isjson is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.JSON_ARRAYContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression_list_())
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the json_array is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.JSON_VALUEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the json_value is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.JSON_QUERYContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the json_query is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.JSON_MODIFYContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the json_modify is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.JSON_PATH_EXISTSContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the json_path_exists is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.ABSContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the abs is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.ACOSContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the acos is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.ASINContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the asin is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.ATANContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the atan is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.ATN2Context:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the atn2 is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.CEILINGContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the ceiling is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.COSContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the cos is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.COTContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the cot is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.DEGREESContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the degrees is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.EXPContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the exp is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.FLOORContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the floor is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.LOGContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the log is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.LOG10Context:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the log10 is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.POWERContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the power is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.RADIANSContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the radians is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.RANDContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the rand is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.ROUNDContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the round is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.MATH_SIGNContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the math_sign is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.SINContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the sin is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.SQRTContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the sqrt is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.SQUAREContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the square is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.TANContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the tan is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.GREATESTContext:
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression_list_())
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the greatest is sensitive")
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		panic("never reach here")
	case *parser.LEASTContext:
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression_list_())
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the least is sensitive")
			}
			return ctx.GetText(), maskingAttributes, nil
		}
		panic("never reach here")
	case *parser.CERTENCODEDContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the certencoded is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.CERTPRIVATEKEYContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the certprivatekey is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.DATABASE_PRINCIPAL_IDContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the database_principal_id is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.HAS_DBACCESSContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the has_dbaccess is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.HAS_PERMS_BY_NAMEContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the has_perms_by_name is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.IS_MEMBERContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the is_member is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.IS_ROLEMEMBERContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the is_rolemember is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.IS_SRVROLEMEMBERContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the is_srvrolemember is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.LOGINPROPERTYContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the loginproperty is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.PERMISSIONSContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the permissions is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.PWDENCRYPTContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the pwdencrypt is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.PWDCOMPAREContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the pwdcompare is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.SESSIONPROPERTYContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the sessionproperty is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.SUSER_IDContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the suser_id is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.SUSER_SNAMEContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the suser_sname is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.SUSER_SIDContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		for _, expression := range ctx.AllExpression() {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the suser_sid is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.USER_IDContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the user_id is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.USER_NAMEContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the user_name is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.SCALAR_FUNCTIONContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Expression_list_())
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the scalar_function is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		if scalarFunctionName := ctx.Scalar_function_name(); scalarFunctionName != nil {
			_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Scalar_function_name())
			if err != nil {
				return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the scalar_function is sensitive")
			}
			finalAttributes.TransmittedByInExpression(maskingAttributes)
			if finalAttributes.IsNeverChangeInTransmission() {
				return ctx.GetText(), finalAttributes, nil
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Scalar_function_nameContext:
		return ctx.GetText(), base.NewDefaultMaskingAttributes(), nil
	case *parser.Freetext_functionContext:
		finalAttributes := base.NewDefaultMaskingAttributes()
		if allFullColumnName := ctx.AllFull_column_name(); len(allFullColumnName) > 0 {
			for _, fullColumnName := range allFullColumnName {
				_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(fullColumnName)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the freetext_function is sensitive")
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			finalAttributes := base.NewDefaultMaskingAttributes()
			for _, expression := range allExpressions {
				_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(expression)
				if err != nil {
					return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the freetext_function is sensitive")
				}
				finalAttributes.TransmittedByInExpression(maskingAttributes)
				if finalAttributes.IsNeverChangeInTransmission() {
					return ctx.GetText(), finalAttributes, nil
				}
			}
		}
		return ctx.GetText(), finalAttributes, nil
	case *parser.Full_column_nameContext:
		fieldInfo, err := extractor.tsqlIsFullColumnNameSensitive(ctx)
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the full_column_name is sensitive")
		}
		return fieldInfo.Name, fieldInfo.MaskingAttributes, nil
	case *parser.PARTITION_FUNCContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Partition_function().Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the partition_function is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	case *parser.HIERARCHYID_METHODContext:
		_, maskingAttributes, err := extractor.evalExpressionElemMaskingLevel(ctx.Hierarchyid_static_method().Expression())
		if err != nil {
			return "", base.NewEmptyMaskingAttributes(), errors.Wrapf(err, "failed to check if the hierarchyid_method is sensitive")
		}
		return ctx.GetText(), maskingAttributes, nil
	}
	panic("never reach here")
}

// unionTableSources union two or more table sources, return the original one if there is only one table source
func unionTableSources(tableSources ...base.TableSource) ([]base.QuerySpanResult, error) {
	if len(tableSources) == 0 {
		return nil, errors.New("no table source to union")
	}

	anchor := tableSources[0].GetQuerySpanResult()

	for i := 1; i < len(tableSources); i++ {
		current := tableSources[i].GetQuerySpanResult()
		if len(current) != len(anchor) {
			return nil, errors.Errorf("the %dth table source has different column number with previous anchors, previous: %d, current %d", i+1, len(anchor), len(current))
		}

		for j := range anchor {
			anchor[j].SourceColumns, _ = base.MergeSourceColumnSet(anchor[j].SourceColumns, current[j].SourceColumns)
		}
	}

	return anchor, nil
}
