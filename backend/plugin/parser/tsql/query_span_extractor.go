package tsql

import (
	"context"
	"fmt"
	"sort"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"

	parsererror "github.com/bytebase/bytebase/backend/plugin/parser/errors"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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

	accessTables, err := getAccessTables(q.connectedDB, q.connectedSchema, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get access tables")
	}
	// We do not support simultaneous access to the system table and the user table
	// because we do not synchronize the schema of the system table.
	// This causes an error (NOT_FOUND) when using querySpanExtractor.findTableSchema.
	// As a result, we exclude getting query span results for accessing only the system table.
	allSystems, mixed := isMixedQuery(accessTables, q.ignoreCaseSensitive)
	if mixed != nil {
		return nil, mixed
	}
	if allSystems {
		return &base.QuerySpan{
			Results:       []base.QuerySpanResult{},
			SourceColumns: base.SourceColumnSet{},
		}, nil
	}

	result, err := ParseTSQL(statement)
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
	// In order to decrease the maintenance cost, we use listener
	// to handlet the select statement precisely.
	listener := &tsqlSelectOnlyListener{
		extractor: q,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, result.Tree)
	if listener.err != nil {
		return nil, errors.Wrapf(listener.err, "failed to extract sensitive fields from select statement")
	}

	return &base.QuerySpan{
		SourceColumns: accessTables,
		Results:       listener.result,
	}, nil
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
		listener.err = err
		return
	}

	listener.result = result.GetQuerySpanResult()
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
			normalizedCTEName := NormalizeTSQLIdentifier(commonTableExpression.GetExpression_name())
			var columns []base.QuerySpanResult
			// If statement has more than one UNION, the first one is the anchor, and the rest are recursive.
			recursiveCTE := false
			queryExpression := commonTableExpression.Select_statement().Query_expression()
			if queryExpression.Query_specification() != nil {
				anchorTable, err := q.extractTSqlSensitiveFieldsFromQuerySpecification(queryExpression.Query_specification())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields from `query_specification` in `query_expression`")
				}
				columns = anchorTable.GetQuerySpanResult()
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
						if len(columns) != len(recursiveTableSourceQuerySpanResult) {
							return nil, errors.Wrapf(err, "the number of columns in the query statement nearly line %d returns %d fields, but %d set operator near line %d returns %d fields", ctx.GetStart().GetLine(), len(columns), i+1, allSQLUnions[i].GetStart().GetLine(), len(recursiveTableSourceQuerySpanResult))
						}
						for j := range recursiveTableSourceQuerySpanResult {
							columns[j].SourceColumns, _ = base.MergeSourceColumnSet(columns[j].SourceColumns, recursiveTableSourceQuerySpanResult[j].SourceColumns)
						}
					}
				}
			} else if allQueryExpression := queryExpression.AllQuery_expression(); len(allQueryExpression) > 0 {
				if len(allQueryExpression) > 1 {
					recursiveCTE = true
				}
				anchorTable, err := q.extractTSqlSensitiveFieldsFromQueryExpression(allQueryExpression[0])
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields from `query_specification` in `query_expression`")
				}
				columns = anchorTable.GetQuerySpanResult()
			}
			if recursiveCTE {
				tempCte := &base.PseudoTable{
					Name:    normalizedCTEName,
					Columns: columns,
				}
				originalSize := len(q.ctes)
				for {
					q.ctes = q.ctes[:originalSize]
					q.ctes = append(q.ctes, tempCte)
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
							var anyChange bool
							tempCte.Columns[i].SourceColumns, anyChange = base.MergeSourceColumnSet(tempCte.Columns[i].SourceColumns, recursiveTableSourceQuerySpanResult[i].SourceColumns)
							change = change || anyChange
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
							var anyChange bool
							tempCte.Columns[i].SourceColumns, change = base.MergeSourceColumnSet(tempCte.Columns[i].SourceColumns, recursiveTableSourceQuerySpanResult[i].SourceColumns)
							change = change || anyChange
						}
					}
					if !change {
						break
					}
				}
				q.ctes = q.ctes[:originalSize]
				columns = tempCte.Columns
			}
			if v := commonTableExpression.Column_name_list(); v != nil {
				if len(columns) != len(v.AllId_()) {
					return nil, errors.Errorf("the number of column name list %d does not match the number of columns %d", len(v.AllId_()), len(columns))
				}
				for i, columnName := range v.AllId_() {
					normalizedColumnName := NormalizeTSQLIdentifier(columnName)
					columns[i].Name = normalizedColumnName
				}
			}
			// Append to the extractor.schemaInfo.DatabaseList
			q.ctes = append(q.ctes, &base.PseudoTable{
				Name:    normalizedCTEName,
				Columns: columns,
			})
		}
	}

	return q.extractTSqlSensitiveFieldsFromSelectStatement(ctx.Select_statement())
}

// extractTSqlSensitiveFieldsFromSelectStatement extracts sensitive fields from select_statement.
func (q *querySpanExtractor) extractTSqlSensitiveFieldsFromSelectStatement(ctx parser.ISelect_statementContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	queryResult, err := q.extractTSqlSensitiveFieldsFromQueryExpression(ctx.Query_expression())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields from `query_expression` in `select_statement`")
	}

	return queryResult, nil
}

func (q *querySpanExtractor) extractTSqlSensitiveFieldsFromQueryExpression(ctx parser.IQuery_expressionContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	if ctx.Query_specification() != nil {
		anchor, err := q.extractTSqlSensitiveFieldsFromQuerySpecification(ctx.Query_specification())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields from `query_specification` in `query_expression`")
		}
		if allSQLUnions := ctx.AllSql_union(); len(allSQLUnions) > 0 {
			for i, sqlUnion := range allSQLUnions {
				// For UNION operator, the number of the columns in the result set is the same, and will use the left part's column name.
				// So we only need to extract the sensitive fields of the right part.
				right, err := q.extractTSqlSensitiveFieldsFromQuerySpecification(sqlUnion.Query_specification())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract the %d set operator near line %d", i+1, sqlUnion.GetStart().GetLine())
				}
				querySpanResult, err := unionTableSources(anchor, right)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to union the %d set operator near line %d", i+1, sqlUnion.GetStart().GetLine())
				}
				anchor.Columns = querySpanResult
			}
		}
		return anchor, nil
	}

	if allQueryExpressions := ctx.AllQuery_expression(); len(allQueryExpressions) > 0 {
		anchor, err := q.extractTSqlSensitiveFieldsFromQueryExpression(allQueryExpressions[0])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields from `query_specification` in `query_expression`")
		}
		for i := 1; i < len(allQueryExpressions); i++ {
			// For UNION operator, the number of the columns in the result set is the same, and will use the left part's column name.
			// So we only need to extract the sensitive fields of the right part.
			right, err := q.extractTSqlSensitiveFieldsFromQueryExpression(allQueryExpressions[i])
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract the %d set operator near line %d", i+1, allQueryExpressions[i].GetStart().GetLine())
			}
			querySpanResult, err := unionTableSources(anchor, right)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to union the %d set operator near line %d", i+1, allQueryExpressions[i].GetStart().GetLine())
			}
			anchor.Columns = querySpanResult
		}
		return anchor, nil
	}

	panic("never reach here")
}

func (q *querySpanExtractor) extractTSqlSensitiveFieldsFromQuerySpecification(ctx parser.IQuery_specificationContext) (*base.PseudoTable, error) {
	if ctx == nil {
		return nil, nil
	}

	if from := ctx.GetFrom(); from != nil {
		fromFieldList, err := q.extractTSqlSensitiveFieldsFromTableSources(ctx.Table_sources())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields from `table_sources` in `query_specification`")
		}
		originalFromFieldList := len(q.tableSourcesFrom)
		q.tableSourcesFrom = append(q.tableSourcesFrom, fromFieldList...)
		defer func() {
			q.tableSourcesFrom = q.tableSourcesFrom[:originalFromFieldList]
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
			left, err := q.tsqlGetAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
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
			querySpanResult, err := q.getQuerySpanResultFromExpr(expressionElem)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to check if the expression element is sensitive")
			}
			result.Columns = append(result.Columns, querySpanResult)
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

func (q *querySpanExtractor) extractTSqlSensitiveFieldsFromTableSource(ctx parser.ITable_sourceContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	var columns []base.QuerySpanResult
	anchor, err := q.extractTSqlSensitiveFieldsFromTableSourceItem(ctx.Table_source_item())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract sensitive fields from `table_source_item` in `table_source`")
	}
	name := anchor.GetTableName()
	columns = append(columns, anchor.GetQuerySpanResult()...)

	if allJoinParts := ctx.AllJoin_part(); len(allJoinParts) > 0 {
		name = ""
		// https://learn.microsoft.com/en-us/sql/relational-databases/performance/joins?view=sql-server-ver16
		for _, joinPart := range allJoinParts {
			if joinOn := joinPart.Join_on(); joinOn != nil {
				right, err := q.extractTSqlSensitiveFieldsFromTableSource(joinOn.Table_source())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields from `table_source` in `join_on`")
				}
				columns = append(columns, right.GetQuerySpanResult()...)
			}
			if crossJoin := joinPart.Cross_join(); crossJoin != nil {
				right, err := q.extractTSqlSensitiveFieldsFromTableSourceItem(crossJoin.Table_source_item())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract sensitive fields from `table_source` in `cross_join`")
				}
				columns = append(columns, right.GetQuerySpanResult()...)
			}
			if apply := joinPart.Apply_(); apply != nil {
				right, err := q.extractTSqlSensitiveFieldsFromTableSourceItem(apply.Table_source_item())
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
		Name:    name,
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
		asName := NormalizeTSQLIdentifier(asTableAlias.Table_alias().Id_())
		result = &base.PseudoTable{
			Name:    asName,
			Columns: result.GetQuerySpanResult(),
		}
	}

	if columnAliasList := ctx.Column_alias_list(); columnAliasList != nil {
		allColumnAlias := columnAliasList.AllColumn_alias()
		if len(allColumnAlias) != len(result.GetQuerySpanResult()) {
			return nil, errors.Errorf("the number of column alias %d does not match the number of columns %d", len(allColumnAlias), len(result.GetQuerySpanResult()))
		}
		for i := 0; i < len(allColumnAlias); i++ {
			if allColumnAlias[i].Id_() != nil {
				name := NormalizeTSQLIdentifier(allColumnAlias[i].Id_())
				result = &base.PseudoTable{
					Name:    result.GetTableName(),
					Columns: result.GetQuerySpanResult(),
				}
				result.GetQuerySpanResult()[i].Name = name
				continue
			} else if allColumnAlias[i].STRING() != nil {
				name := allColumnAlias[i].STRING().GetText()
				result = &base.PseudoTable{
					Name:    result.GetTableName(),
					Columns: result.GetQuerySpanResult(),
				}
				result.GetQuerySpanResult()[i].Name = name
				continue
			}
			panic("never reach here")
		}
	}

	return result, nil
}

func (q *querySpanExtractor) extractTSqlSensitiveFieldsFromDerivedTable(ctx parser.IDerived_tableContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, nil
	}

	allSubquery := ctx.AllSubquery()
	if len(allSubquery) > 0 {
		anchor, err := q.extractTSqlSensitiveFieldsFromSubquery(allSubquery[0])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract sensitive fields from `subquery` in `derived_table`")
		}
		for i := 1; i < len(allSubquery); i++ {
			// For UNION operator, the number of the columns in the result set is the same, and will use the left part's column name.
			// So we only need to extract the sensitive fields of the right part.
			right, err := q.extractTSqlSensitiveFieldsFromSubquery(allSubquery[i])
			rightQuerySpanResult := right.GetQuerySpanResult()
			anchorQuerySpanResult := anchor.GetQuerySpanResult()
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract the %d set operator near line %d", i+1, allSubquery[i].GetStart().GetLine())
			}
			if len(anchorQuerySpanResult) != len(rightQuerySpanResult) {
				return nil, errors.Wrapf(err, "the number of columns in the derived table statement nearly line %d returns %d fields, but %d set operator near line %d returns %d fields", ctx.GetStart().GetLine(), len(anchorQuerySpanResult), i+1, allSubquery[i].GetStart().GetLine(), len(rightQuerySpanResult))
			}
			for i := range rightQuerySpanResult {
				anchorQuerySpanResult[i].SourceColumns, _ = base.MergeSourceColumnSet(anchorQuerySpanResult[i].SourceColumns, rightQuerySpanResult[i].SourceColumns)
			}
		}
		return anchor, nil
	}

	if tableValueConstructor := ctx.Table_value_constructor(); tableValueConstructor != nil {
		return q.extractTSqlSensitiveFieldsFromTableValueConstructor(tableValueConstructor)
	}

	panic("never reach here")
}

func (q *querySpanExtractor) extractTSqlSensitiveFieldsFromTableValueConstructor(ctx parser.ITable_value_constructorContext) (base.TableSource, error) {
	if allExpressionList := ctx.AllExpression_list_(); len(allExpressionList) > 0 {
		// The number of expression in each expression list should be the same.
		// But we do not check, just use the first one, and engine will throw a compilation error if the number of expressions are not the same.
		expressionList := allExpressionList[0]
		pseudoTable := &base.PseudoTable{
			Columns: make([]base.QuerySpanResult, 0, len(expressionList.AllExpression())),
		}
		for _, expression := range expressionList.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract sensitive fields from `expression` in `table_value_constructor`")
			}
			pseudoTable.Columns = append(pseudoTable.Columns, querySpanResult)
		}
		return pseudoTable, nil
	}
	panic("never reach here")
}

func (q *querySpanExtractor) extractTSqlSensitiveFieldsFromSubquery(ctx parser.ISubqueryContext) (*base.PseudoTable, error) {
	return q.extractTSqlSensitiveFieldsFromSelectStatement(ctx.Select_statement())
}

func (q *querySpanExtractor) tsqlFindTableSchema(fullTableName parser.IFull_table_nameContext) (base.TableSource, error) {
	normalizedLinkedServer, normalizedDatabaseName, normalizedSchemaName, normalizedTableName := normalizeFullTableName(fullTableName, "" /* Linked Server Name */, "", "")
	if normalizedLinkedServer != "" {
		// TODO(zp): How do we handle the linked server?
		return nil, errors.Errorf("linked server is not supported yet, but found %q", fullTableName.GetText())
	}

	// For SQL Server, the cte will shadow the physical tables, so we check the ctes first,
	// also, we record the cte in ascending order, so we should check the ctes in descending order
	// to find the nearest match.
	if normalizedDatabaseName == "" && normalizedSchemaName == "" {
		for _, cte := range q.ctes {
			if q.isIdentifierEqual(normalizedTableName, cte.Name) {
				return cte, nil
			}
		}
	}

	normalizedLinkedServer, normalizedDatabaseName, normalizedSchemaName, normalizedTableName = normalizeFullTableName(fullTableName, "" /* Linked Server Name */, q.connectedDB, q.connectedSchema)
	if normalizedLinkedServer != "" {
		// TODO(zp): How do we handle the linked server?
		return nil, errors.Errorf("linked server is not supported yet, but found %q", fullTableName.GetText())
	}
	allDatabases, err := q.l(q.ctx)
	if err != nil {
		return nil, errors.Errorf("failed to list databases: %v", err)
	}

	for _, databaseName := range allDatabases {
		if normalizedDatabaseName != "" && !q.isIdentifierEqual(normalizedDatabaseName, databaseName) {
			continue
		}
		_, database, err := q.f(q.ctx, databaseName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database %s metadata", databaseName)
		}

		allSchemaNames := database.ListSchemaNames()
		for _, schemaName := range allSchemaNames {
			if normalizedSchemaName != "" && !q.isIdentifierEqual(normalizedSchemaName, schemaName) {
				continue
			}
			schemaSchema := database.GetSchema(schemaName)
			allTableNames := schemaSchema.ListTableNames()
			for _, tableName := range allTableNames {
				if !q.isIdentifierEqual(normalizedTableName, tableName) {
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
				return physicalTableSource, nil
			}
		}
	}
	return nil, &parsererror.ResourceNotFoundError{
		Database: &normalizedDatabaseName,
		Schema:   &normalizedSchemaName,
		Table:    &normalizedTableName,
	}
}

func (q *querySpanExtractor) tsqlGetAllFieldsOfTableInFromOrOuterCTE(normalizedDatabaseName, normalizedSchemaName, normalizedTableName string) ([]base.QuerySpanResult, error) {
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

	for _, tableSource := range q.tableSourcesFrom {
		if mask&maskDatabaseName != 0 && !q.isIdentifierEqual(normalizedDatabaseName, tableSource.GetDatabaseName()) {
			continue
		}
		if mask&maskSchemaName != 0 && !q.isIdentifierEqual(normalizedSchemaName, tableSource.GetSchemaName()) {
			continue
		}
		if mask&maskTableName != 0 && !q.isIdentifierEqual(normalizedTableName, tableSource.GetTableName()) {
			continue
		}
		return tableSource.GetQuerySpanResult(), nil
	}

	return nil, errors.Errorf(`no matching table %q.%q.%q`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
}

func (q *querySpanExtractor) tsqlIsFullColumnNameSensitive(ctx parser.IFull_column_nameContext) (base.QuerySpanResult, error) {
	normalizedLinkedServer, normalizedDatabaseName, normalizedSchemaName, normalizedTableName := normalizeFullTableName(ctx.Full_table_name(), "", "", "")
	if normalizedLinkedServer != "" {
		return base.QuerySpanResult{}, errors.Errorf("linked server is not supported yet, but found %q", ctx.GetText())
	}
	normalizedColumnName := NormalizeTSQLIdentifier(ctx.Id_())

	return q.tsqlIsFieldSensitive(normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
}

func (q *querySpanExtractor) tsqlIsFieldSensitive(normalizedDatabaseName string, normalizedSchemaName string, normalizedTableName string, normalizedColumnName string) (base.QuerySpanResult, error) {
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
	for _, tableSource := range q.tableSourcesFrom {
		if mask&maskDatabaseName != 0 && !q.isIdentifierEqual(normalizedDatabaseName, tableSource.GetDatabaseName()) {
			continue
		}
		if mask&maskSchemaName != 0 && !q.isIdentifierEqual(normalizedSchemaName, tableSource.GetSchemaName()) {
			continue
		}
		if mask&maskTableName != 0 && !q.isIdentifierEqual(normalizedTableName, tableSource.GetTableName()) {
			continue
		}
		for _, column := range tableSource.GetQuerySpanResult() {
			if mask&maskColumnName != 0 && !q.isIdentifierEqual(normalizedColumnName, column.Name) {
				continue
			}
			return column, nil
		}
	}
	return base.QuerySpanResult{}, errors.Errorf(`no matching column %q.%q.%q.%q`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName, normalizedColumnName)
}

// isIdentifierEqual compares the identifier with the given normalized parts, returns true if they are equal.
// It will consider the case sensitivity based on the current database.
func (q *querySpanExtractor) isIdentifierEqual(a, b string) bool {
	if !q.ignoreCaseSensitive {
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

// getQuerySpanResultFromExpr returns true if the expression element is sensitive, and returns the column name.
// It is the closure of the expression_elemContext, it will recursively check the sub expression element.
func (q *querySpanExtractor) getQuerySpanResultFromExpr(ctx antlr.RuleContext) (base.QuerySpanResult, error) {
	if ctx == nil {
		return base.QuerySpanResult{
			Name:          "",
			SourceColumns: make(base.SourceColumnSet, 0),
		}, nil
	}
	switch ctx := ctx.(type) {
	case *parser.Expression_elemContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the expression element is sensitive")
		}
		if columnAlias := ctx.Column_alias(); columnAlias != nil {
			querySpanResult.Name = NormalizeTSQLIdentifier(columnAlias.Id_())
		} else if asColumnAlias := ctx.As_column_alias(); asColumnAlias != nil {
			querySpanResult.Name = NormalizeTSQLIdentifier(asColumnAlias.Column_alias().Id_())
		}
		return querySpanResult, nil
	case *parser.ExpressionContext:
		if ctx.Primitive_expression() != nil {
			return q.getQuerySpanResultFromExpr(ctx.Primitive_expression())
		}
		if ctx.Function_call() != nil {
			return q.getQuerySpanResultFromExpr(ctx.Function_call())
		}
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
				if err != nil {
					return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the expression is sensitive")
				}
				anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			}
		}
		if valueCall := ctx.Value_call(); valueCall != nil {
			return q.getQuerySpanResultFromExpr(valueCall)
		}
		if queryCall := ctx.Query_call(); queryCall != nil {
			return q.getQuerySpanResultFromExpr(queryCall)
		}
		if existCall := ctx.Exist_call(); existCall != nil {
			return q.getQuerySpanResultFromExpr(existCall)
		}
		if modifyCall := ctx.Modify_call(); modifyCall != nil {
			return q.getQuerySpanResultFromExpr(modifyCall)
		}
		if hierarchyIDCall := ctx.Hierarchyid_call(); hierarchyIDCall != nil {
			return q.getQuerySpanResultFromExpr(hierarchyIDCall)
		}
		if caseExpression := ctx.Case_expression(); caseExpression != nil {
			return q.getQuerySpanResultFromExpr(caseExpression)
		}
		if fullColumnName := ctx.Full_column_name(); fullColumnName != nil {
			return q.getQuerySpanResultFromExpr(fullColumnName)
		}
		if bracketExpression := ctx.Bracket_expression(); bracketExpression != nil {
			return q.getQuerySpanResultFromExpr(bracketExpression)
		}
		if unaryOperationExpression := ctx.Unary_operator_expression(); unaryOperationExpression != nil {
			return q.getQuerySpanResultFromExpr(unaryOperationExpression)
		}
		if overClause := ctx.Over_clause(); overClause != nil {
			return q.getQuerySpanResultFromExpr(overClause)
		}
		return anchor, nil
	case *parser.Unary_operator_expressionContext:
		if expression := ctx.Expression(); expression != nil {
			return q.getQuerySpanResultFromExpr(expression)
		}
		return base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}, nil
	case *parser.Bracket_expressionContext:
		if expression := ctx.Expression(); expression != nil {
			return q.getQuerySpanResultFromExpr(expression)
		}
		if subquery := ctx.Subquery(); subquery != nil {
			return q.getQuerySpanResultFromExpr(subquery)
		}
		return base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}, nil
	case *parser.Case_expressionContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
				if err != nil {
					return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the case_expression is sensitive")
				}
				anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			}
		}
		if allSwitchSections := ctx.AllSwitch_section(); len(allSwitchSections) > 0 {
			for _, switchSection := range allSwitchSections {
				querySpanExtractor, err := q.getQuerySpanResultFromExpr(switchSection)
				if err != nil {
					return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the case_expression is sensitive")
				}
				anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanExtractor.SourceColumns)
			}
		}
		if allSwitchSearchConditionSections := ctx.AllSwitch_search_condition_section(); len(allSwitchSearchConditionSections) > 0 {
			for _, switchSearchConditionSection := range allSwitchSearchConditionSections {
				querySpanExtractor, err := q.getQuerySpanResultFromExpr(switchSearchConditionSection)
				if err != nil {
					return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the case_expression is sensitive")
				}
				anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanExtractor.SourceColumns)
			}
		}
		return anchor, nil
	case *parser.Switch_sectionContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
				if err != nil {
					return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the switch_setion is sensitive")
				}
				anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			}
		}
		return anchor, nil
	case *parser.Switch_search_condition_sectionContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if searchCondition := ctx.Search_condition(); searchCondition != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(searchCondition)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the switch_search_condition_section is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		if expression := ctx.Expression(); expression != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the switch_search_condition_section is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.Search_conditionContext:
		if predicate := ctx.Predicate(); predicate != nil {
			return q.getQuerySpanResultFromExpr(predicate)
		}
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if allSearchConditions := ctx.AllSearch_condition(); len(allSearchConditions) > 0 {
			for _, searchCondition := range allSearchConditions {
				querySpanResult, err := q.getQuerySpanResultFromExpr(searchCondition)
				if err != nil {
					return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the search_condition is sensitive")
				}
				anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			}
		}
		return anchor, nil
	case *parser.PredicateContext:
		if subquery := ctx.Subquery(); subquery != nil {
			return q.getQuerySpanResultFromExpr(subquery)
		}
		if freeTextPredicate := ctx.Freetext_predicate(); freeTextPredicate != nil {
			return q.getQuerySpanResultFromExpr(freeTextPredicate)
		}

		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
				if err != nil {
					return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the predicate is sensitive")
				}
				anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			}
		}
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expressionList)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the predicate is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.Freetext_predicateContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
				if err != nil {
					return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the freetext_predicate is sensitive")
				}
				anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			}
		}
		if allCullColumnName := ctx.AllFull_column_name(); len(allCullColumnName) > 0 {
			for _, fullColumnName := range allCullColumnName {
				querySpanResult, err := q.getQuerySpanResultFromExpr(fullColumnName)
				if err != nil {
					return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the freetext_predicate is sensitive")
				}
				anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			}
		}
		return anchor, nil
	case *parser.SubqueryContext:
		// For subquery, we clone the current extractor, reset the from list, but keep the cte, and then extract the sensitive fields from the subquery
		cloneExtractor := &querySpanExtractor{
			connectedDB:     q.connectedDB,
			connectedSchema: q.connectedSchema,
			f:               q.f,
			l:               q.l,
			// outerTableSources: extractor.outerTableSources,
			ctes:                q.ctes,
			ignoreCaseSensitive: q.ignoreCaseSensitive,
		}
		tableSource, err := cloneExtractor.extractTSqlSensitiveFieldsFromSubquery(ctx)
		// The expect behavior is the fieldInfo contains only one field, which is the column name,
		// but in order to do not block user, we just return isSensitive if there is any sensitive field.
		// return fieldInfo[0].sensitive, err
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the subquery is sensitive")
		}
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, field := range tableSource.GetQuerySpanResult() {
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, field.SourceColumns)
		}
		return anchor, nil
	case *parser.Hierarchyid_callContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
				if err != nil {
					return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the hierarchyid_call is sensitive")
				}
				anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			}
		}
		return anchor, nil
	case *parser.Query_callContext:
		return base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}, nil
	case *parser.Exist_callContext:
		return base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}, nil
	case *parser.Modify_callContext:
		return base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}, nil
	case *parser.Value_callContext:
		return base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}, nil
	case *parser.Primitive_expressionContext:
		if ctx.Primitive_constant() != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Primitive_constant())
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the primitive constant is sensitive")
			}
			return querySpanResult, nil
		}
		panic("never reach here")
	case *parser.Primitive_constantContext:
		return base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}, nil
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
		return q.getQuerySpanResultFromExpr(ctx.Ranking_windowed_function())
	case *parser.Ranking_windowed_functionContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if overClause := ctx.Over_clause(); overClause != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(overClause)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the ranking_windowed_function is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		if expression := ctx.Expression(); expression != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the ranking_windowed_function is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.Over_clauseContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression_list_())
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the over_clause is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		if orderByClause := ctx.Order_by_clause(); orderByClause != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(orderByClause)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the over_clause is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		if rowOrRangeClause := ctx.Row_or_range_clause(); rowOrRangeClause != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(rowOrRangeClause)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the over_clause is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.Expression_list_Context:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the expression_list is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.Order_by_clauseContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, orderByExpression := range ctx.GetOrder_bys() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(orderByExpression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the order_by_clause is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.Order_by_expressionContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the order_by_expression is sensitive")
		}
		return querySpanResult, nil
	case *parser.Row_or_range_clauseContext:
		if windowFrameExtent := ctx.Window_frame_extent(); windowFrameExtent != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(windowFrameExtent)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the row_or_range_clause is sensitive")
			}
			return querySpanResult, nil
		}
		panic("never reach here")
	case *parser.Window_frame_extentContext:
		if windowFramePreceding := ctx.Window_frame_preceding(); windowFramePreceding != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(windowFramePreceding)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the window_frame_extent is sensitive")
			}
			return querySpanResult, nil
		}
		if windowFrameBounds := ctx.AllWindow_frame_bound(); len(windowFrameBounds) > 0 {
			anchor := base.QuerySpanResult{
				Name:          ctx.GetText(),
				SourceColumns: make(base.SourceColumnSet),
			}
			for _, windowFrameBound := range windowFrameBounds {
				querySpanResult, err := q.getQuerySpanResultFromExpr(windowFrameBound)
				if err != nil {
					return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the window_frame_extent is sensitive")
				}
				anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			}
			return anchor, nil
		}
		panic("never reach here")
	case *parser.Window_frame_boundContext:
		if preceding := ctx.Window_frame_preceding(); preceding != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(preceding)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the window_frame_bound is sensitive")
			}
			return querySpanResult, nil
		} else if following := ctx.Window_frame_following(); following != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(following)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the window_frame_bound is sensitive")
			}
			return querySpanResult, nil
		}
		panic("never reach here")
	case *parser.Window_frame_precedingContext:
		return base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}, nil
	case *parser.Window_frame_followingContext:
		return base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}, nil
	case *parser.AGGREGATE_WINDOWED_FUNCContext:
		return q.getQuerySpanResultFromExpr(ctx.Aggregate_windowed_function())
	case *parser.Aggregate_windowed_functionContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if allDistinctExpression := ctx.All_distinct_expression(); allDistinctExpression != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(allDistinctExpression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the aggregate_windowed_function is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		if overClause := ctx.Over_clause(); overClause != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(overClause)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the aggregate_windowed_function is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		if expression := ctx.Expression(); expression != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the aggregate_windowed_function is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expressionList)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the aggregate_windowed_function is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.All_distinct_expressionContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the all_distinct_expression is sensitive")
		}
		return querySpanResult, nil
	case *parser.ANALYTIC_WINDOWED_FUNCContext:
		return q.getQuerySpanResultFromExpr(ctx.Analytic_windowed_function())
	case *parser.Analytic_windowed_functionContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
				if err != nil {
					return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the analytic_windowed_function is sensitive")
				}
				anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			}
		}
		if overClause := ctx.Over_clause(); overClause != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(overClause)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the analytic_windowed_function is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expressionList)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the analytic_windowed_function is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		if orderByClause := ctx.Order_by_clause(); orderByClause != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(orderByClause)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the analytic_windowed_function is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.BUILT_IN_FUNCContext:
		return q.getQuerySpanResultFromExpr(ctx.Built_in_functions())
	case *parser.APP_NAMEContext:
		return base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}, nil
	case *parser.APPLOCK_MODEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the applock_mode is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.APPLOCK_TESTContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the applock_test is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.ASSEMBLYPROPERTYContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the assemblyproperty is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.COL_LENGTHContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the col_length is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.COL_NAMEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the col_name is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.COLUMNPROPERTYContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the columnproperty is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.DATABASEPROPERTYEXContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the databasepropertyex is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.DB_IDContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the db_id is sensitive")
		}
		return querySpanResult, nil
	case *parser.DB_NAMEContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the db_name is sensitive")
		}
		return querySpanResult, nil
	case *parser.FILE_IDContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the file_id is sensitive")
		}
		return querySpanResult, nil
	case *parser.FILE_IDEXContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the file_idex is sensitive")
		}
		return querySpanResult, nil
	case *parser.FILE_NAMEContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the file_name is sensitive")
		}
		return querySpanResult, nil
	case *parser.FILEGROUP_IDContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the filegroup_id is sensitive")
		}
		return querySpanResult, nil
	case *parser.FILEGROUP_NAMEContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the filegroup_name is sensitive")
		}
		return querySpanResult, nil
	case *parser.FILEGROUPPROPERTYContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the filegroupproperty is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.FILEPROPERTYContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the fileproperty is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.FILEPROPERTYEXContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the filepropertyex is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.FULLTEXTCATALOGPROPERTYContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the fulltextcatalogproperty is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.FULLTEXTSERVICEPROPERTYContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the fulltextserviceproperty is sensitive")
		}
		return querySpanResult, nil
	case *parser.INDEX_COLContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the index_col is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.INDEXKEY_PROPERTYContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the indexkey_property is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.INDEXPROPERTYContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the indexproperty is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.OBJECT_DEFINITIONContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the object_definition is sensitive")
		}
		return querySpanResult, nil
	case *parser.OBJECT_IDContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the object_id is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.OBJECT_NAMEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the object_name is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.OBJECT_SCHEMA_NAMEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the object_schema_name is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.OBJECTPROPERTYContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the objectproperty is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.OBJECTPROPERTYEXContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the objectpropertyex is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.PARSENAMEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the parsename is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.SCHEMA_IDContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the schema_id is sensitive")
		}
		return querySpanResult, nil
	case *parser.SCHEMA_NAMEContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the schema_name is sensitive")
		}
		return querySpanResult, nil
	case *parser.SERVERPROPERTYContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the serverproperty is sensitive")
		}
		return querySpanResult, nil
	case *parser.STATS_DATEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the stats_date is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.TYPE_IDContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the type_id is sensitive")
		}
		return querySpanResult, nil
	case *parser.TYPE_NAMEContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the type_name is sensitive")
		}
		return querySpanResult, nil
	case *parser.TYPEPROPERTYContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the typeproperty is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.ASCIIContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the ascii is sensitive")
		}
		return querySpanResult, nil
	case *parser.CHARContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the char is sensitive")
		}
		return querySpanResult, nil
	case *parser.CHARINDEXContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the charindex is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.CONCATContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the concat is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.CONCAT_WSContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the concat_ws is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.DIFFERENCEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the difference is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.FORMATContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the format is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.LEFTContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the left is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.LENContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the len is sensitive")
		}
		return querySpanResult, nil
	case *parser.LOWERContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the lower is sensitive")
		}
		return querySpanResult, nil
	case *parser.LTRIMContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the ltrim is sensitive")
		}
		return querySpanResult, nil
	case *parser.NCHARContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the nchar is sensitive")
		}
		return querySpanResult, nil
	case *parser.PATINDEXContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the patindex is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.QUOTENAMEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the quotename is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.REPLACEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the replace is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.REPLICATEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the replicate is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.REVERSEContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the reverse is sensitive")
		}
		return querySpanResult, nil
	case *parser.RIGHTContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the right is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.RTRIMContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the rtrim is sensitive")
		}
		return querySpanResult, nil
	case *parser.SOUNDEXContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the soundex is sensitive")
		}
		return querySpanResult, nil
	case *parser.SPACEContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the space is sensitive")
		}
		return querySpanResult, nil
	case *parser.STRContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the str is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.STRINGAGGContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the stringagg is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.STRING_ESCAPEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the string_escape is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.STUFFContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the stuff is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.SUBSTRINGContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the substring is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.TRANSLATEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the translate is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.TRIMContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the trim is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.UNICODEContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the unicode is sensitive")
		}
		return querySpanResult, nil
	case *parser.UPPERContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the upper is sensitive")
		}
		return querySpanResult, nil
	case *parser.BINARY_CHECKSUMContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the binary_checksum is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.CHECKSUMContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the checksum is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}, nil
	case *parser.COMPRESSContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the compress is sensitive")
		}
		return querySpanResult, nil
	case *parser.DECOMPRESSContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the decompress is sensitive")
		}
		return querySpanResult, nil
	case *parser.FORMATMESSAGEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the formatmessage is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.ISNULLContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the isnull is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.ISNUMERICContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the isnumeric is sensitive")
		}
		return querySpanResult, nil
	case *parser.CASTContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the cast is sensitive")
		}
		return querySpanResult, nil
	case *parser.TRY_CASTContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the try_cast is sensitive")
		}
		return querySpanResult, nil
	case *parser.CONVERTContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the convert is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.COALESCEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression_list_())
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the coalesce is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.CURSOR_STATUSContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the cursor_status is sensitive")
		}
		return querySpanResult, nil
	case *parser.CERT_IDContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the cert_id is sensitive")
		}
		return querySpanResult, nil
	case *parser.DATALENGTHContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the datalength is sensitive")
		}
		return querySpanResult, nil
	case *parser.IDENT_CURRENTContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the  ident_current is sensitive")
		}
		return querySpanResult, nil
	case *parser.IDENT_INCRContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the  ident_incr is sensitive")
		}
		return querySpanResult, nil
	case *parser.IDENT_SEEDContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the  ident_seed is sensitive")
		}
		return querySpanResult, nil
	case *parser.SQL_VARIANT_PROPERTYContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the sql_variant_property is sensitive")
		}
		return querySpanResult, nil
	case *parser.DATE_BUCKETContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the date_bucket is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.DATEADDContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the dateadd is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.DATEDIFFContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the datediff is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.DATEDIFF_BIGContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the datediff_big is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.DATEFROMPARTSContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the datefromparts is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.DATENAMEContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the datename is sensitive")
		}
		return querySpanResult, nil
	case *parser.DATEPARTContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the datepart is sensitive")
		}
		return querySpanResult, nil
	case *parser.DATETIME2FROMPARTSContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the datetime2fromparts is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.DATETIMEFROMPARTSContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the datetimefromparts is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.DATETIMEOFFSETFROMPARTSContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the datetimeoffsetfromparts is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.DATETRUNCContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the datetrunc is sensitive")
		}
		return querySpanResult, nil
	case *parser.DAYContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the day is sensitive")
		}
		return querySpanResult, nil
	case *parser.EOMONTHContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the eomonth is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.ISDATEContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the isdate is sensitive")
		}
		return querySpanResult, nil
	case *parser.MONTHContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the month is sensitive")
		}
		return querySpanResult, nil
	case *parser.SMALLDATETIMEFROMPARTSContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the smalldatetimefromparts is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.SWITCHOFFSETContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the switchoffset is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.TIMEFROMPARTSContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the timefromparts is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.TODATETIMEOFFSETContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the todatetimeoffset is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.YEARContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the year is sensitive")
		}
		return querySpanResult, nil
	case *parser.NULLIFContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the nullif is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.PARSEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the parse is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.IIFContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the iif is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.ISJSONContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the isjson is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.JSON_ARRAYContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression_list_())
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the json_array is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.JSON_VALUEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the json_value is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.JSON_QUERYContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the json_query is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.JSON_MODIFYContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the json_modify is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.JSON_PATH_EXISTSContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the json_path_exists is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.ABSContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the abs is sensitive")
		}
		return querySpanResult, nil
	case *parser.ACOSContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the acos is sensitive")
		}
		return querySpanResult, nil
	case *parser.ASINContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the asin is sensitive")
		}
		return querySpanResult, nil
	case *parser.ATANContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the atan is sensitive")
		}
		return querySpanResult, nil
	case *parser.ATN2Context:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the atn2 is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.CEILINGContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the ceiling is sensitive")
		}
		return querySpanResult, nil
	case *parser.COSContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the cos is sensitive")
		}
		return querySpanResult, nil
	case *parser.COTContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the cot is sensitive")
		}
		return querySpanResult, nil
	case *parser.DEGREESContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the degrees is sensitive")
		}
		return querySpanResult, nil
	case *parser.EXPContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the exp is sensitive")
		}
		return querySpanResult, nil
	case *parser.FLOORContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the floor is sensitive")
		}
		return querySpanResult, nil
	case *parser.LOGContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the log is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.LOG10Context:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the log10 is sensitive")
		}
		return querySpanResult, nil
	case *parser.POWERContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the power is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.RADIANSContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the radians is sensitive")
		}
		return querySpanResult, nil
	case *parser.RANDContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the rand is sensitive")
		}
		return querySpanResult, nil
	case *parser.ROUNDContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the round is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.MATH_SIGNContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the math_sign is sensitive")
		}
		return querySpanResult, nil
	case *parser.SINContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the sin is sensitive")
		}
		return querySpanResult, nil
	case *parser.SQRTContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the sqrt is sensitive")
		}
		return querySpanResult, nil
	case *parser.SQUAREContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the square is sensitive")
		}
		return querySpanResult, nil
	case *parser.TANContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the tan is sensitive")
		}
		return querySpanResult, nil
	case *parser.GREATESTContext:
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression_list_())
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the greatest is sensitive")
			}
			return querySpanResult, nil
		}
		panic("never reach here")
	case *parser.LEASTContext:
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression_list_())
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the least is sensitive")
			}
			return querySpanResult, nil
		}
		panic("never reach here")
	case *parser.CERTENCODEDContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the certencoded is sensitive")
		}
		return querySpanResult, nil
	case *parser.CERTPRIVATEKEYContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the certprivatekey is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.DATABASE_PRINCIPAL_IDContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the database_principal_id is sensitive")
		}
		return querySpanResult, nil
	case *parser.HAS_DBACCESSContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the has_dbaccess is sensitive")
		}
		return querySpanResult, nil
	case *parser.HAS_PERMS_BY_NAMEContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the has_perms_by_name is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.IS_MEMBERContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the is_member is sensitive")
		}
		return querySpanResult, nil
	case *parser.IS_ROLEMEMBERContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the is_rolemember is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.IS_SRVROLEMEMBERContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the is_srvrolemember is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.LOGINPROPERTYContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the loginproperty is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.PERMISSIONSContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the permissions is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.PWDENCRYPTContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the pwdencrypt is sensitive")
		}
		return querySpanResult, nil
	case *parser.PWDCOMPAREContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the pwdcompare is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.SESSIONPROPERTYContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the sessionproperty is sensitive")
		}
		return querySpanResult, nil
	case *parser.SUSER_IDContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the suser_id is sensitive")
		}
		return querySpanResult, nil
	case *parser.SUSER_SNAMEContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the suser_sname is sensitive")
		}
		return querySpanResult, nil
	case *parser.SUSER_SIDContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		for _, expression := range ctx.AllExpression() {
			querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the suser_sid is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.USER_IDContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the user_id is sensitive")
		}
		return querySpanResult, nil
	case *parser.USER_NAMEContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the user_name is sensitive")
		}
		return querySpanResult, nil
	case *parser.SCALAR_FUNCTIONContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if expressionList := ctx.Expression_list_(); expressionList != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Expression_list_())
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the scalar_function is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		if scalarFunctionName := ctx.Scalar_function_name(); scalarFunctionName != nil {
			querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Scalar_function_name())
			if err != nil {
				return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the scalar_function is sensitive")
			}
			var change bool
			anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			if !change {
				return anchor, nil
			}
		}
		return anchor, nil
	case *parser.Scalar_function_nameContext:
		return base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}, nil
	case *parser.Freetext_functionContext:
		anchor := base.QuerySpanResult{
			Name:          ctx.GetText(),
			SourceColumns: make(base.SourceColumnSet),
		}
		if allFullColumnName := ctx.AllFull_column_name(); len(allFullColumnName) > 0 {
			for _, fullColumnName := range allFullColumnName {
				querySpanResult, err := q.getQuerySpanResultFromExpr(fullColumnName)
				if err != nil {
					return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the freetext_function is sensitive")
				}
				anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			}
		}
		if allExpressions := ctx.AllExpression(); len(allExpressions) > 0 {
			for _, expression := range allExpressions {
				querySpanResult, err := q.getQuerySpanResultFromExpr(expression)
				if err != nil {
					return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the freetext_function is sensitive")
				}
				anchor.SourceColumns, _ = base.MergeSourceColumnSet(anchor.SourceColumns, querySpanResult.SourceColumns)
			}
		}
		return anchor, nil
	case *parser.Full_column_nameContext:
		querySpanResult, err := q.tsqlIsFullColumnNameSensitive(ctx)
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the full_column_name is sensitive")
		}
		return querySpanResult, nil
	case *parser.PARTITION_FUNCContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Partition_function().Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the partition_function is sensitive")
		}
		return querySpanResult, nil
	case *parser.HIERARCHYID_METHODContext:
		querySpanResult, err := q.getQuerySpanResultFromExpr(ctx.Hierarchyid_static_method().Expression())
		if err != nil {
			return base.QuerySpanResult{}, errors.Wrapf(err, "failed to check if the hierarchyid_method is sensitive")
		}
		return querySpanResult, nil
	}
	panic("never reach here")
}

// unionTableSources union two or more table sources, return the original one if there is only one table source.
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

// getAccessTables extracts the list of resources from the SELECT statement, and normalizes the object names with the NON-EMPTY currentNormalizedDatabase and currentNormalizedSchema.
func getAccessTables(currentNormalizedDatabase string, currentNormalizedSchema string, selectStatement string) (base.SourceColumnSet, error) {
	parseResult, err := ParseTSQL(selectStatement)
	if err != nil {
		return nil, err
	}
	if parseResult == nil {
		return nil, nil
	}

	l := &accessTableListener{
		currentDatabase: currentNormalizedDatabase,
		currentSchema:   currentNormalizedSchema,
		resourceMap:     make(base.SourceColumnSet),
	}

	var result []base.SchemaResource
	antlr.ParseTreeWalkerDefault.Walk(l, parseResult.Tree)
	sort.Slice(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})

	return l.resourceMap, nil
}

type accessTableListener struct {
	*parser.BaseTSqlParserListener

	currentDatabase string
	currentSchema   string
	resourceMap     base.SourceColumnSet
}

// EnterTable_source_item is called when the parser enters the table_source_item production.
func (l *accessTableListener) EnterTable_source_item(ctx *parser.Table_source_itemContext) {
	if fullTableName := ctx.Full_table_name(); fullTableName != nil {
		var linkedServer string
		if server := fullTableName.GetLinkedServer(); server != nil {
			linkedServer = NormalizeTSQLIdentifier(server)
		}

		database := l.currentDatabase
		if d := fullTableName.GetDatabase(); d != nil {
			normalizedD := NormalizeTSQLIdentifier(d)
			if normalizedD != "" {
				database = normalizedD
			}
		}

		schema := l.currentSchema
		if s := fullTableName.GetSchema(); s != nil {
			normalizedS := NormalizeTSQLIdentifier(s)
			if normalizedS != "" {
				schema = normalizedS
			}
		}

		var table string
		if t := fullTableName.GetTable(); t != nil {
			normalizedT := NormalizeTSQLIdentifier(t)
			if normalizedT != "" {
				table = normalizedT
			}
		}

		l.resourceMap[base.ColumnResource{
			Server:   linkedServer,
			Database: database,
			Schema:   schema,
			Table:    table,
		}] = true
	}

	if rowsetFunction := ctx.Rowset_function(); rowsetFunction != nil {
		return
	}

	// https://simonlearningsqlserver.wordpress.com/tag/changetable/
	// It seems that the CHANGETABLE is only return some statistics, so we ignore it.
	if changeTable := ctx.Change_table(); changeTable != nil {
		return
	}

	// other...
}

// isMixedQuery checks whether the query accesses the user table and system table at the same time.
func isMixedQuery(m base.SourceColumnSet, ignoreCaseSensitive bool) (allSystems bool, mixed error) {
	userMsg, systemMsg := "", ""
	for table := range m {
		if msg := isSystemResource(table, ignoreCaseSensitive); msg != "" {
			systemMsg = msg
			continue
		}
		userMsg = fmt.Sprintf("user table %q.%q", table.Schema, table.Table)
		if systemMsg != "" {
			return false, errors.Errorf("cannot access %s and %s at the same time", userMsg, systemMsg)
		}
	}

	if userMsg != "" && systemMsg != "" {
		return false, errors.Errorf("cannot access %s and %s at the same time", userMsg, systemMsg)
	}

	return userMsg == "" && systemMsg != "", nil
}

func isSystemResource(base.ColumnResource, bool) string {
	// TODO(zp): fix me.
	return ""
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
		linkedServer = NormalizeTSQLIdentifier(server)
	}

	database := normalizedFallbackDatabaseName
	if d := fullTableName.GetDatabase(); d != nil {
		normalizedD := NormalizeTSQLIdentifier(d)
		if normalizedD != "" {
			database = normalizedD
		}
	}

	schema := normalizedFallbackSchemaName
	if s := fullTableName.GetSchema(); s != nil {
		normalizedS := NormalizeTSQLIdentifier(s)
		if normalizedS != "" {
			schema = normalizedS
		}
	}

	var table string
	if t := fullTableName.GetTable(); t != nil {
		normalizedT := NormalizeTSQLIdentifier(t)
		if normalizedT != "" {
			table = normalizedT
		}
	}

	return linkedServer, database, schema, table
}
