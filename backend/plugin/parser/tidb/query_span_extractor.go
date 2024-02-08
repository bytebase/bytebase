package tidb

import (
	"context"
	"strings"

	"github.com/pingcap/tidb/pkg/parser"
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	parsererror "github.com/bytebase/bytebase/backend/plugin/parser/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type querySpanExtractor struct {
	ctx               context.Context
	connectedDB       string
	metaCache         map[string]*model.DatabaseMetadata
	f                 base.GetDatabaseMetadataFunc
	lowerTableViewMap map[string]map[string]bool

	ctes              []*base.PseudoTable
	outerTableSources []base.TableSource
	tableSourcesFrom  []base.TableSource
}

func newQuerySpanExtractor(connectedDB string, f base.GetDatabaseMetadataFunc) *querySpanExtractor {
	return &querySpanExtractor{
		connectedDB:       connectedDB,
		metaCache:         make(map[string]*model.DatabaseMetadata),
		lowerTableViewMap: make(map[string]map[string]bool),
		f:                 f,
	}
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx

	// TODO: check for query all system tables.

	p := parser.New()
	p.EnableWindowFunc(true)
	nodeList, _, err := p.Parse(statement, "", "")
	if err != nil {
		return nil, err
	}
	if len(nodeList) != 1 {
		return nil, errors.Errorf("expecting 1 statement, but got %d", len(nodeList))
	}

	node := nodeList[0]

	switch node.(type) {
	case *tidbast.SelectStmt:
	case *tidbast.SetOprStmt:
	case *tidbast.CreateViewStmt:
	case *tidbast.ExplainStmt:
		// Skip the EXPLAIN statement.
		return &base.QuerySpan{
			Results:       []base.QuerySpanResult{},
			SourceColumns: base.SourceColumnSet{},
		}, nil
	default:
		return nil, errors.Errorf("expect a query statement but found %T", node)
	}

	tableSource, err := q.extractTableSourceFromNode(node)
	if err != nil {
		// TODO: check for query all system tables.
		return nil, err
	}

	return &base.QuerySpan{
		Results:       tableSource.GetQuerySpanResult(),
		SourceColumns: q.getAccessTables(node),
	}, nil
}

func (q *querySpanExtractor) getAccessTables(node tidbast.Node) base.SourceColumnSet {
	accessesMap := make(base.SourceColumnSet)
	tables := ExtractMySQLTableList(node, false /* asName */)
	for _, table := range tables {
		databaseName := table.Schema.O
		if databaseName == "" {
			databaseName = q.connectedDB
		}

		// This is a false-positive behavior, the table we found may not be the table the query actually accesses.
		// For example, the query is `WITH t1 AS (SELECT 1) SELECT * FROM t1` and we have a physical table `t1` in the database exactly,
		// what we found is the physical table `t1`, but the query actually accesses the CTE `t1`.
		// We do this because we do not have too much time to implement the real behavior.
		// XXX(rebelice/zp): Can we pass more information here to make this function know the context and then
		// figure out whether the table is the table the query actually accesses
		if !q.existsTableMetadata(databaseName, table.Name.O) {
			continue
		}

		resource := base.ColumnResource{
			Database: databaseName,
			Table:    table.Name.O,
		}
		accessesMap[resource] = true
	}
	return accessesMap
}

func (q *querySpanExtractor) existsTableMetadata(databaseName string, tableName string) bool {
	if databaseName == "" {
		databaseName = q.connectedDB
	}

	if q.lowerTableViewMap[databaseName] != nil {
		return q.lowerTableViewMap[databaseName][strings.ToLower(tableName)]
	}

	databaseMetadata, err := q.getDatabaseMetadata(databaseName)
	if err != nil {
		return false
	}

	if databaseMetadata == nil {
		return false
	}
	schemaMetadata := databaseMetadata.GetSchema("")
	if schemaMetadata == nil {
		return false
	}

	tableViewMap := make(map[string]bool)
	for _, table := range schemaMetadata.ListTableNames() {
		tableViewMap[strings.ToLower(table)] = true
	}
	for _, view := range schemaMetadata.ListViewNames() {
		tableViewMap[strings.ToLower(view)] = true
	}
	q.lowerTableViewMap[databaseName] = tableViewMap

	return q.lowerTableViewMap[databaseName][strings.ToLower(tableName)]
}

func (q *querySpanExtractor) extractTableSourceFromNode(node tidbast.Node) (base.TableSource, error) {
	if node == nil {
		return nil, nil
	}
	switch node := node.(type) {
	case *tidbast.SelectStmt:
		return q.extractSelect(node)
	case *tidbast.Join:
		return q.extractJoin(node)
	case *tidbast.TableSource:
		return q.extractTableSource(node)
	case *tidbast.TableName:
		return q.extractTableName(node)
	case *tidbast.SetOprStmt:
		return q.extractSetOpr(node)
	case *tidbast.SetOprSelectList:
		return q.extractSetOprSelectList(node)
	case *tidbast.CreateViewStmt:
		tableSource, err := q.extractTableSourceFromNode(node.Select)
		if err != nil {
			return nil, err
		}
		querySpanResult := tableSource.GetQuerySpanResult()

		if len(node.Cols) == 0 {
			return base.NewPseudoTable(node.ViewName.Name.O, querySpanResult), nil
		}

		if len(node.Cols) != len(querySpanResult) {
			return nil, errors.Errorf("The used SELECT statements have a different number of columns for view %s", node.ViewName.Name.O)
		}

		var columns []base.QuerySpanResult
		for i, item := range querySpanResult {
			columns = append(columns, base.QuerySpanResult{
				Name:          node.Cols[i].L,
				SourceColumns: item.SourceColumns,
			})
		}
		return base.NewPseudoTable(node.ViewName.Name.O, columns), nil
	}
	return nil, nil
}

func (q *querySpanExtractor) extractNonRecursiveCTE(node *tidbast.CommonTableExpression) (*base.PseudoTable, error) {
	tableSource, err := q.extractTableSourceFromNode(node.Query.Query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract table source from CTE query")
	}
	querySpanResults := tableSource.GetQuerySpanResult()
	if len(node.ColNameList) > 0 {
		if len(node.ColNameList) != len(querySpanResults) {
			return nil, errors.Errorf("cte table expr has %d columns, but alias has %d columns", len(querySpanResults), len(node.ColNameList))
		}
		for i, name := range node.ColNameList {
			// The column name for MySQL is case insensitive.
			querySpanResults[i].Name = name.L
		}
	}

	return &base.PseudoTable{
		Name:    node.Name.O,
		Columns: querySpanResults,
	}, nil
}

func (q *querySpanExtractor) extractRecursiveCTE(node *tidbast.CommonTableExpression) (*base.PseudoTable, error) {
	switch x := node.Query.Query.(type) {
	case *tidbast.SetOprStmt:
		if x.With != nil {
			previousCteOuterLength := len(q.ctes)
			defer func() {
				q.ctes = q.ctes[:previousCteOuterLength]
			}()
			for _, cte := range x.With.CTEs {
				cteTableSource, err := q.extractCTE(cte)
				if err != nil {
					return nil, err
				}
				q.ctes = append(q.ctes, cteTableSource)
			}

			for i := previousCteOuterLength; i < len(q.ctes); i++ {
				cteTableSource := q.ctes[i]
				if cteTableSource.Name == node.Name.O {
					// It means this recursive CTE will be hidden by the inner CTE with the same name.
					// In other words, this recursive CTE will be not references by itself sub-query.
					// So, we can build it as non-recursive CTE
					return q.extractNonRecursiveCTE(node)
				}
			}
		}

		initialPart, recursivePart := splitInitialAndRecursivePart(x, node.Name.O)
		if len(initialPart) == 0 {
			return nil, errors.Errorf("Failed to find initial part for recursive common table expression")
		}
		if len(recursivePart) == 0 {
			return q.extractNonRecursiveCTE(node)
		}

		initialTableSource, err := q.extractTableSourceFromNode(&tidbast.SetOprStmt{
			SelectList: &tidbast.SetOprSelectList{
				Selects: initialPart,
			},
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract table source from initial part of recursive CTE")
		}
		initialQuerySpanResult := initialTableSource.GetQuerySpanResult()
		if len(node.ColNameList) > 0 {
			if len(node.ColNameList) != len(initialQuerySpanResult) {
				return nil, errors.Errorf("cte table expr has %d columns, but alias has %d columns", len(initialQuerySpanResult), len(node.ColNameList))
			}
			for i, name := range node.ColNameList {
				// The column name for MySQL is case insensitive.
				initialQuerySpanResult[i].Name = name.L
			}
		}

		cteTableResource := &base.PseudoTable{
			Name:    node.Name.O,
			Columns: initialQuerySpanResult,
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
		q.ctes = append(q.ctes, cteTableResource)
		defer func() {
			q.ctes = q.ctes[:len(q.ctes)-1]
		}()
		for {
			recursiveTableSource, err := q.extractTableSourceFromNode(&tidbast.SetOprStmt{
				SelectList: &tidbast.SetOprSelectList{
					Selects: recursivePart,
				},
			})
			if err != nil {
				return nil, errors.Wrap(err, "failed to extract table source from recursive part of recursive CTE")
			}
			recursiveQuerySpanResult := recursiveTableSource.GetQuerySpanResult()
			if len(recursiveQuerySpanResult) != len(initialQuerySpanResult) {
				return nil, errors.Errorf("cte table expr has %d columns, but recursive part has %d columns", len(initialQuerySpanResult), len(recursiveQuerySpanResult))
			}

			changed := false
			for i, spanQueryResult := range recursiveQuerySpanResult {
				newResourceColumns, hasDiff := base.MergeSourceColumnSet(initialQuerySpanResult[i].SourceColumns, spanQueryResult.SourceColumns)
				if hasDiff {
					changed = true
					initialQuerySpanResult[i].SourceColumns = newResourceColumns
				}
			}

			if !changed {
				break
			}
			q.ctes[len(q.ctes)-1].Columns = initialQuerySpanResult
		}
		return cteTableResource, nil
	default:
		return q.extractNonRecursiveCTE(node)
	}
}

func (q *querySpanExtractor) extractCTE(node *tidbast.CommonTableExpression) (*base.PseudoTable, error) {
	if node.IsRecursive {
		return q.extractRecursiveCTE(node)
	}
	return q.extractNonRecursiveCTE(node)
}

func (q *querySpanExtractor) extractSelect(node *tidbast.SelectStmt) (base.TableSource, error) {
	if node.With != nil {
		previousCteOuterLength := len(q.ctes)
		defer func() {
			q.ctes = q.ctes[:previousCteOuterLength]
		}()
		for _, cte := range node.With.CTEs {
			cteTableSource, err := q.extractCTE(cte)
			if err != nil {
				return nil, err
			}
			q.ctes = append(q.ctes, cteTableSource)
		}
	}

	var fromFieldList []base.TableSource
	if node.From != nil {
		tableSource, err := q.extractTableSourceFromNode(node.From.TableRefs)
		if err != nil {
			return nil, errors.Wrap(err, "failed to extract table source from FROM clause")
		}
		q.tableSourcesFrom = append(q.tableSourcesFrom, tableSource)
		fromFieldList = append(fromFieldList, tableSource)
		defer func() {
			q.tableSourcesFrom = nil
		}()
	}

	result := new(base.PseudoTable)
	if node.Fields != nil {
		for _, field := range node.Fields.Fields {
			if field.WildCard != nil {
				if field.WildCard.Table.O == "" {
					var columns []base.QuerySpanResult
					for _, tableSource := range fromFieldList {
						columns = append(columns, tableSource.GetQuerySpanResult()...)
					}
					result.Columns = append(result.Columns, columns...)
				} else {
					for _, tableSource := range fromFieldList {
						sameDatabase := (field.WildCard.Schema.O == tableSource.GetDatabaseName() || (field.WildCard.Schema.O == "" && tableSource.GetDatabaseName() == q.connectedDB))
						sameTable := field.WildCard.Table.O == tableSource.GetTableName()
						find := false
						if sameDatabase && sameTable {
							result.Columns = append(result.Columns, tableSource.GetQuerySpanResult()...)
							find = true
							break
						}
						if !find {
							sources, ok := q.getAllTableColumnSources(field.WildCard.Schema.O, field.WildCard.Table.O)
							if ok {
								result.Columns = append(result.Columns, sources...)
								find = true
							}
						}
						if !find {
							return nil, &parsererror.ResourceNotFoundError{
								Err:      errors.New("failed to find table to calculate asterisk"),
								Database: &field.WildCard.Schema.O,
								Table:    &field.WildCard.Table.O,
							}
						}
					}
				}
			} else {
				sourceColumnSet, err := q.extractSourceColumnSetFromExpression(field.Expr)
				if err != nil {
					return nil, err
				}
				fieldName := extractFieldName(field)
				result.Columns = append(result.Columns, base.QuerySpanResult{
					Name:          fieldName,
					SourceColumns: sourceColumnSet,
				})
			}
		}
	}

	return result, nil
}

func (q *querySpanExtractor) extractSourceColumnSetFromExpressionList(list []tidbast.ExprNode) (base.SourceColumnSet, error) {
	result := make(base.SourceColumnSet)
	for _, node := range list {
		sourceColumnSet, err := q.extractSourceColumnSetFromExpression(node)
		if err != nil {
			return nil, err
		}
		result, _ = base.MergeSourceColumnSet(result, sourceColumnSet)
	}
	return result, nil
}

func (q *querySpanExtractor) extractSourceColumnSetFromExpression(in tidbast.ExprNode) (base.SourceColumnSet, error) {
	if in == nil {
		return base.SourceColumnSet{}, nil
	}

	switch node := in.(type) {
	case *tidbast.ColumnNameExpr:
		database, table, column := node.Name.Schema.O, node.Name.Table.O, node.Name.Name.O
		sources, ok := q.getFieldColumnSource(database, table, column)
		if !ok {
			return base.SourceColumnSet{}, &parsererror.ResourceNotFoundError{
				Err:      errors.New("cannot find the column ref"),
				Database: &database,
				Table:    &table,
				Column:   &column,
			}
		}
		return sources, nil
	case *tidbast.BinaryOperationExpr:
		return q.extractSourceColumnSetFromExpressionList([]tidbast.ExprNode{node.L, node.R})
	case *tidbast.UnaryOperationExpr:
		return q.extractSourceColumnSetFromExpression(node.V)
	case *tidbast.FuncCallExpr:
		return q.extractSourceColumnSetFromExpressionList(node.Args)
	case *tidbast.FuncCastExpr:
		return q.extractSourceColumnSetFromExpression(node.Expr)
	case *tidbast.AggregateFuncExpr:
		return q.extractSourceColumnSetFromExpressionList(node.Args)
	case *tidbast.PatternInExpr:
		nodeList := []tidbast.ExprNode{}
		nodeList = append(nodeList, node.Expr)
		nodeList = append(nodeList, node.List...)
		nodeList = append(nodeList, node.Sel)
		return q.extractSourceColumnSetFromExpressionList(nodeList)
	case *tidbast.PatternLikeOrIlikeExpr:
		return q.extractSourceColumnSetFromExpressionList([]tidbast.ExprNode{node.Expr, node.Pattern})
	case *tidbast.PatternRegexpExpr:
		return q.extractSourceColumnSetFromExpressionList([]tidbast.ExprNode{node.Expr, node.Pattern})
	case *tidbast.SubqueryExpr:
		// Subquery in SELECT fields is special.
		// It can be the non-associated or associated subquery.
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new q is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &querySpanExtractor{
			ctx:               q.ctx,
			connectedDB:       q.connectedDB,
			metaCache:         q.metaCache,
			f:                 q.f,
			ctes:              q.ctes,
			outerTableSources: append(q.outerTableSources, q.tableSourcesFrom...),
			tableSourcesFrom:  []base.TableSource{},
		}
		tableSource, err := subqueryExtractor.extractTableSourceFromNode(node.Query)
		if err != nil {
			return base.SourceColumnSet{}, errors.Wrap(err, "failed to extract table source from subquery")
		}
		spanResult := tableSource.GetQuerySpanResult()
		sourceColumnSet := base.SourceColumnSet{}

		for _, field := range spanResult {
			sourceColumnSet, _ = base.MergeSourceColumnSet(sourceColumnSet, field.SourceColumns)
		}
		return sourceColumnSet, nil
	case *tidbast.CompareSubqueryExpr:
		return q.extractSourceColumnSetFromExpressionList([]tidbast.ExprNode{node.L, node.R})
	case *tidbast.ExistsSubqueryExpr:
		return q.extractSourceColumnSetFromExpression(node.Sel)
	case *tidbast.IsNullExpr:
		return q.extractSourceColumnSetFromExpression(node.Expr)
	case *tidbast.IsTruthExpr:
		return q.extractSourceColumnSetFromExpression(node.Expr)
	case *tidbast.BetweenExpr:
		return q.extractSourceColumnSetFromExpressionList([]tidbast.ExprNode{node.Expr, node.Left, node.Right})
	case *tidbast.CaseExpr:
		nodeList := []tidbast.ExprNode{}
		nodeList = append(nodeList, node.Value)
		nodeList = append(nodeList, node.ElseClause)
		for _, whenClause := range node.WhenClauses {
			nodeList = append(nodeList, whenClause.Expr)
			nodeList = append(nodeList, whenClause.Result)
		}
		return q.extractSourceColumnSetFromExpressionList(nodeList)
	case *tidbast.ParenthesesExpr:
		return q.extractSourceColumnSetFromExpression(node.Expr)
	case *tidbast.RowExpr:
		return q.extractSourceColumnSetFromExpressionList(node.Values)
	case *tidbast.VariableExpr:
		return q.extractSourceColumnSetFromExpression(node.Value)
	case *tidbast.PositionExpr:
		return q.extractSourceColumnSetFromExpression(node.P)
	case *tidbast.MatchAgainst:
		return q.extractSourceColumnSetFromExpression(node.Against)
	case *tidbast.WindowFuncExpr:
		return q.extractSourceColumnSetFromExpressionList(node.Args)
	case *tidbast.ValuesExpr,
		*tidbast.TableNameExpr,
		*tidbast.MaxValueExpr,
		*tidbast.SetCollationExpr,
		*tidbast.TrimDirectionExpr,
		*tidbast.TimeUnitExpr,
		*tidbast.GetFormatSelectorExpr,
		*tidbast.DefaultExpr:
		// No expression need to extract.
	}
	return base.SourceColumnSet{}, nil
}

func (q *querySpanExtractor) getAllTableColumnSources(databaseName, tableName string) ([]base.QuerySpanResult, bool) {
	findInTableSource := func(tableSource base.TableSource) ([]base.QuerySpanResult, bool) {
		if databaseName != "" && databaseName != tableSource.GetDatabaseName() {
			return nil, false
		}
		if databaseName == "" && tableSource.GetDatabaseName() != "" && tableSource.GetDatabaseName() != q.connectedDB {
			return nil, false
		}
		if tableName != "" && tableName != tableSource.GetTableName() {
			return nil, false
		}
		// If the table name is empty, we should check if there are ambiguous fields,
		// but we delegate this responsibility to the db-server, we do the fail-open strategy here.

		return tableSource.GetQuerySpanResult(), true
	}

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

	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		tableSource := q.outerTableSources[i]
		if sourceColumnSet, ok := findInTableSource(tableSource); ok {
			return sourceColumnSet, true
		}
	}

	for _, tableSource := range q.tableSourcesFrom {
		if sourceColumnSet, ok := findInTableSource(tableSource); ok {
			return sourceColumnSet, true
		}
	}

	return nil, false
}

func (q *querySpanExtractor) getFieldColumnSource(databaseName, tableName, fieldName string) (base.SourceColumnSet, bool) {
	findInTableSource := func(tableSource base.TableSource) (base.SourceColumnSet, bool) {
		if databaseName != "" && databaseName != tableSource.GetDatabaseName() {
			return nil, false
		}
		if databaseName == "" && tableSource.GetDatabaseName() != "" && tableSource.GetDatabaseName() != q.connectedDB {
			return nil, false
		}
		if tableName != "" && tableName != tableSource.GetTableName() {
			return nil, false
		}
		// If the table name is empty, we should check if there are ambiguous fields,
		// but we delegate this responsibility to the db-server, we do the fail-open strategy here.

		querySpanResult := tableSource.GetQuerySpanResult()
		for _, field := range querySpanResult {
			if strings.EqualFold(field.Name, fieldName) {
				return field.SourceColumns, true
			}
		}
		return nil, false
	}

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

	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		tableSource := q.outerTableSources[i]
		if sourceColumnSet, ok := findInTableSource(tableSource); ok {
			return sourceColumnSet, true
		}
	}

	for _, tableSource := range q.tableSourcesFrom {
		if sourceColumnSet, ok := findInTableSource(tableSource); ok {
			return sourceColumnSet, true
		}
	}

	return base.SourceColumnSet{}, false
}

func (q *querySpanExtractor) extractJoin(node *tidbast.Join) (base.TableSource, error) {
	if node.Right == nil {
		// This case is no Join.
		return q.extractTableSourceFromNode(node.Left)
	}
	leftTableSource, err := q.extractTableSourceFromNode(node.Left)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract table source from left table")
	}
	q.tableSourcesFrom = append(q.tableSourcesFrom, leftTableSource)
	rightTableSource, err := q.extractTableSourceFromNode(node.Right)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract table source from right table")
	}
	q.tableSourcesFrom = append(q.tableSourcesFrom, rightTableSource)
	q.tableSourcesFrom = append(q.tableSourcesFrom, rightTableSource)
	return q.mergeJoinTableSource(node, leftTableSource, rightTableSource)
}

func (*querySpanExtractor) mergeJoinTableSource(node *tidbast.Join, leftTableSource, rightTableSource base.TableSource) (*base.PseudoTable, error) {
	leftSpanResult, rightSpanResult := leftTableSource.GetQuerySpanResult(), rightTableSource.GetQuerySpanResult()

	result := new(base.PseudoTable)

	leftSpanResultIdx, rightSpanResultIdx := make(map[string]int), make(map[string]int)
	for i, spanResult := range leftSpanResult {
		// Column name in TiDB is case insensitive.
		leftSpanResultIdx[strings.ToLower(spanResult.Name)] = i
	}
	for i, spanResult := range rightSpanResult {
		// Column name in TiDB is case insensitive.
		rightSpanResultIdx[strings.ToLower(spanResult.Name)] = i
	}

	if node.NaturalJoin {
		// Natural Join will merge the same column name field.
		for _, spanResult := range leftSpanResult {
			if rightIdx, exists := rightSpanResultIdx[strings.ToLower(spanResult.Name)]; exists {
				spanResult.SourceColumns, _ = base.MergeSourceColumnSet(spanResult.SourceColumns, rightSpanResult[rightIdx].SourceColumns)
			}
			result.Columns = append(result.Columns, spanResult)
		}
		for _, spanResult := range rightSpanResult {
			if _, exists := leftSpanResultIdx[strings.ToLower(spanResult.Name)]; !exists {
				result.Columns = append(result.Columns, spanResult)
			}
		}
	} else {
		if len(node.Using) != 0 {
			// ... JOIN ... USING (...) will merge the column in USING.
			usingMap := make(map[string]bool)
			for _, column := range node.Using {
				// Column name in TiDB is case insensitive.
				usingMap[column.Name.L] = true
			}

			for _, spanResult := range leftSpanResult {
				_, existsInUsingMap := usingMap[strings.ToLower(spanResult.Name)]
				rightIdx, existsInRightField := rightSpanResultIdx[strings.ToLower(spanResult.Name)]
				// Merge column in USING.
				if existsInUsingMap && existsInRightField {
					rightSpanResult[rightIdx].SourceColumns, _ = base.MergeSourceColumnSet(spanResult.SourceColumns, rightSpanResult[rightIdx].SourceColumns)
				}
				result.Columns = append(result.Columns, spanResult)
			}

			for _, spanResult := range rightSpanResult {
				_, existsInUsing := usingMap[strings.ToLower(spanResult.Name)]
				_, existsInLeftField := leftSpanResultIdx[strings.ToLower(spanResult.Name)]
				if existsInUsing && existsInLeftField {
					continue
				}
				result.Columns = append(result.Columns, spanResult)
			}
		} else {
			result.Columns = append(result.Columns, leftSpanResult...)
			result.Columns = append(result.Columns, rightSpanResult...)
		}
	}

	return result, nil
}

func (q *querySpanExtractor) extractTableSource(node *tidbast.TableSource) (base.TableSource, error) {
	tableSource, err := q.extractTableSourceFromNode(node.Source)
	if err != nil {
		return nil, err
	}
	if node.AsName.O != "" {
		return base.NewPseudoTable(node.AsName.O, tableSource.GetQuerySpanResult()), nil
	}
	return tableSource, nil
}

func (q *querySpanExtractor) findTableSchema(databaseName string, tableName string) (base.TableSource, error) {
	// Each CTE name in one WITH clause must be unique, but we can use the same name in the different level CTE, such as:
	//
	//  with tt2 as (
	//    with tt2 as (select * from t)
	//    select max(a) from tt2)
	//  select * from tt2
	//
	// This query has two CTE can be called `tt2`, and the FROM clause 'from tt2' uses the closer tt2 CTE.
	// This is the reason we loop the slice in reversed order.
	if databaseName == "" {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			cte := q.ctes[i]
			if cte.Name == tableName {
				return cte, nil
			}
		}
	}

	if databaseName == "" {
		databaseName = q.connectedDB
	}

	dbSchema, err := q.getDatabaseMetadata(databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for %q", databaseName)
	}
	if dbSchema == nil {
		return nil, &parsererror.ResourceNotFoundError{
			Database: &databaseName,
		}
	}
	emptySchema := ""
	schema := dbSchema.GetSchema("")
	if schema == nil {
		return nil, &parsererror.ResourceNotFoundError{
			Database: &databaseName,
			Schema:   &emptySchema,
		}
	}
	lowerTableName := strings.ToLower(tableName)
	for _, table := range schema.ListTableNames() {
		if lowerTableName == strings.ToLower(table) {
			var columns []string
			tableMeta := schema.GetTable(table)
			if tableMeta != nil {
				for _, column := range tableMeta.GetColumns() {
					columns = append(columns, column.Name)
				}
				return &base.PhysicalTable{
					Database: databaseName,
					Name:     table,
					Columns:  columns,
				}, nil
			}
		}
	}

	for _, view := range schema.ListViewNames() {
		if lowerTableName == strings.ToLower(view) {
			viewMeta := schema.GetView(view)
			if viewMeta != nil {
				columns, err := q.getColumnsForView(viewMeta.Definition)
				if err != nil {
					return nil, err
				}
				return &base.PhysicalView{
					Database: databaseName,
					Name:     view,
					Columns:  columns,
				}, nil
			}
		}
	}

	return nil, &parsererror.ResourceNotFoundError{
		Database: &databaseName,
		Schema:   &emptySchema,
		Table:    &tableName,
	}
}

func (q *querySpanExtractor) getColumnsForView(definition string) ([]base.QuerySpanResult, error) {
	newQ := newQuerySpanExtractor(q.connectedDB, q.f)
	span, err := newQ.getQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, err
	}
	return span.Results, nil
}

func (q *querySpanExtractor) getDatabaseMetadata(databaseName string) (*model.DatabaseMetadata, error) {
	if meta, ok := q.metaCache[databaseName]; ok {
		return meta, nil
	}
	meta, err := q.f(q.ctx, databaseName)
	if err != nil {
		return nil, err
	}
	q.metaCache[databaseName] = meta
	return meta, nil
}

func (q *querySpanExtractor) extractTableName(node *tidbast.TableName) (base.TableSource, error) {
	return q.findTableSchema(node.Schema.O, node.Name.O)
}

func (q *querySpanExtractor) extractSetOpr(node *tidbast.SetOprStmt) (base.TableSource, error) {
	if node.With != nil {
		previousCteOuterLength := len(q.ctes)
		defer func() {
			q.ctes = q.ctes[:previousCteOuterLength]
		}()
		for _, cte := range node.With.CTEs {
			cteTableSource, err := q.extractCTE(cte)
			if err != nil {
				return nil, err
			}
			q.ctes = append(q.ctes, cteTableSource)
		}
	}

	return q.extractTableSourceFromNode(node.SelectList)
}

func mergeTableSource(left, right base.TableSource) (base.TableSource, error) {
	leftSpanResult, rightSpanResult := left.GetQuerySpanResult(), right.GetQuerySpanResult()

	if len(leftSpanResult) != len(rightSpanResult) {
		return nil, errors.Errorf("left table source has %d columns, but right table source has %d columns", len(leftSpanResult), len(rightSpanResult))
	}

	var result []base.QuerySpanResult
	for i, leftResult := range leftSpanResult {
		rightResult := rightSpanResult[i]
		newResourceColumns, _ := base.MergeSourceColumnSet(leftResult.SourceColumns, rightResult.SourceColumns)
		result = append(result, base.QuerySpanResult{
			Name:          leftResult.Name,
			SourceColumns: newResourceColumns,
		})
	}
	return &base.PseudoTable{
		Name:    "",
		Columns: result,
	}, nil
}

func (q *querySpanExtractor) extractSetOprSelectList(node *tidbast.SetOprSelectList) (base.TableSource, error) {
	if node.With != nil {
		previousCteOuterLength := len(q.ctes)
		defer func() {
			q.ctes = q.ctes[:previousCteOuterLength]
		}()
		for _, cte := range node.With.CTEs {
			cteTableSource, err := q.extractCTE(cte)
			if err != nil {
				return nil, err
			}
			q.ctes = append(q.ctes, cteTableSource)
		}
	}

	var result base.TableSource
	for i, selectStmt := range node.Selects {
		tableSource, err := q.extractTableSourceFromNode(selectStmt)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			result = tableSource
		} else {
			result, err = mergeTableSource(result, tableSource)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to merge table source for %dth select statement", i)
			}
		}
	}
	return result, nil
}
