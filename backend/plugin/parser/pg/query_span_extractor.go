package pg

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	pgquery "github.com/pganalyze/pg_query_go/v4"

	parsererror "github.com/bytebase/bytebase/backend/plugin/parser/errors"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	pgUnknownFieldName = "?column?"
	generateSeries     = "generate_series"
	generateSubscripts = "generate_subscripts"
)

// querySpanExtractor is the extractor to extract the query span from the given pgquery.RawStmt.
type querySpanExtractor struct {
	ctx         context.Context
	connectedDB string
	// The metaCache serves as a lazy-load cache for the database metadata and should not be accessed directly.
	// Instead, use querySpanExtractor.getDatabaseMetadata to access it.
	metaCache map[string]*model.DatabaseMetadata
	f         base.GetDatabaseMetadataFunc

	// Private fields.

	ctes []*base.PseudoTable

	// outerTableSource is the list of table sources from the outer query,
	// it's used to resolve the column name in the correlated subquery.
	outerTableSources []base.TableSource

	// tableSourcesFrom is the list of table sources from the FROM clause.
	tableSourcesFrom []base.TableSource

	// sourceColumnsInFunction is the source columns in the function.
	// It's used to resolve defined functions body as a table source.
	sourceColumnsInFunction base.SourceColumnSet
}

// newQuerySpanExtractor creates a new query span extractor, the databaseMetadata and the ast are in the read guard.
func newQuerySpanExtractor(connectedDB string, getDatabaseMetadata base.GetDatabaseMetadataFunc) *querySpanExtractor {
	return &querySpanExtractor{
		connectedDB:             connectedDB,
		metaCache:               make(map[string]*model.DatabaseMetadata),
		f:                       getDatabaseMetadata,
		sourceColumnsInFunction: make(base.SourceColumnSet),
	}
}

// nolint: unused
func (q *querySpanExtractor) getDatabaseMetadata(database string) (*model.DatabaseMetadata, error) {
	if meta, ok := q.metaCache[database]; ok {
		return meta, nil
	}
	_, meta, err := q.f(q.ctx, database)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", database)
	}
	q.metaCache[database] = meta
	return meta, nil
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	q.ctx = ctx

	// Our querySpanExtractor is based on the pg_query_go library, which does not support listening to or walking the AST.
	// We separate the logic for querying spans and accessing data.
	// The second one is achieved using ParseToJson, which is simpler.
	accessTables, err := q.getAccessTables(stmt)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get access columns from statement: %s", stmt)
	}
	// We do not support simultaneous access to the system table and the user table
	// because we do not synchronize the schema of the system table.
	// This causes an error (NOT_FOUND) when using querySpanExtractor.findTableSchema.
	// As a result, we exclude getting query span results for accessing only the system table.
	allSystems, mixed := isMixedQuery(accessTables)
	if mixed != nil {
		return nil, mixed
	}
	if allSystems {
		return &base.QuerySpan{
			Results:       []base.QuerySpanResult{},
			SourceColumns: base.SourceColumnSet{},
		}, nil
	}

	res, err := pgquery.Parse(stmt)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse statement: %s", stmt)
	}
	if len(res.Stmts) != 1 {
		return nil, errors.Errorf("expecting 1 statement, but got %d", len(res.Stmts))
	}
	ast := res.Stmts[0]

	switch ast.Stmt.Node.(type) {
	case *pgquery.Node_SelectStmt:
	case *pgquery.Node_ExplainStmt:
		// Skip the EXPLAIN statement.
		return &base.QuerySpan{
			Results:       []base.QuerySpanResult{},
			SourceColumns: base.SourceColumnSet{},
		}, nil
	case *pgquery.Node_VariableSetStmt:
		// Skip the SET statement.
		return &base.QuerySpan{
			Results:       []base.QuerySpanResult{},
			SourceColumns: base.SourceColumnSet{},
		}, nil
	default:
		return nil, errors.Errorf("expect a query statement but found %T", ast.Stmt.Node)
	}

	tableSource, err := q.extractTableSourceFromNode(ast.Stmt)
	if err != nil {
		tableNotFound := regexp.MustCompile("^Table \"(.*)\\.(.*)\" not found$")
		content := tableNotFound.FindStringSubmatch(err.Error())
		if len(content) == 3 && IsSystemSchema(content[1]) {
			// skip for system schema
			return &base.QuerySpan{
				Results:       []base.QuerySpanResult{},
				SourceColumns: base.SourceColumnSet{},
			}, nil
		}
		return nil, err
	}

	// merge the source columns in the function to the access tables.
	for source := range q.sourceColumnsInFunction {
		accessTables[source] = true
	}

	return &base.QuerySpan{
		Results:       tableSource.GetQuerySpanResult(),
		SourceColumns: accessTables,
	}, nil
}

// extractTableSourceFromNode is the entry for recursively extracting the span sources from the given node.
// It returns the table source for the given node, which can be a physical table or a down cast temporary table losing the original schema info.
func (q *querySpanExtractor) extractTableSourceFromNode(node *pgquery.Node) (base.TableSource, error) {
	if node == nil {
		return nil, nil
	}

	switch node := node.Node.(type) {
	case *pgquery.Node_SelectStmt:
		return q.extractTableSourceFromSelect(node)
	case *pgquery.Node_RangeVar:
		return q.extractTableSourceFromRangeVar(node)
	case *pgquery.Node_RangeSubselect:
		return q.extractTableSourceFromSubselect(node)
	case *pgquery.Node_JoinExpr:
		return q.extractTableSourceFromJoin(node)
	case *pgquery.Node_RangeFunction:
		return q.extractTableSourceFromRangeFunction(node)
	}
	return nil, newTypeNotSupportedErrorByNode(node)
}

func (q *querySpanExtractor) extractTableSourceFromRangeFunction(node *pgquery.Node_RangeFunction) (base.TableSource, error) {
	schemaName, funcName := extractFunctionNameInRangeFunction(node.RangeFunction)
	if schemaName == "" || funcName == "" {
		return nil, &parsererror.TypeNotSupportedError{
			Type: "function",
			Err:  errors.Errorf("node: %+v", node),
		}
	}

	tableSource, err := q.findFunctionDefine(schemaName, funcName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find function: %s.%s", schemaName, funcName)
	}
	if node.RangeFunction.Alias == nil {
		return tableSource, nil
	}
	querySpanResult := tableSource.GetQuerySpanResult()
	if len(node.RangeFunction.Alias.Colnames) == 0 {
		return base.NewPseudoTable(node.RangeFunction.Alias.Aliasname, querySpanResult), nil
	}

	if len(node.RangeFunction.Alias.Colnames) != len(querySpanResult) {
		return nil, errors.Errorf("expect equal length but found %d and %d", len(node.RangeFunction.Alias.Colnames), len(querySpanResult))
	}

	var columns []base.QuerySpanResult
	for i, columnName := range node.RangeFunction.Alias.Colnames {
		name := columnName.GetString_().Sval
		columns = append(columns, base.QuerySpanResult{
			Name:          name,
			SourceColumns: querySpanResult[i].SourceColumns,
		})
	}

	return base.NewPseudoTable(node.RangeFunction.Alias.Aliasname, columns), nil
}

func (q *querySpanExtractor) findFunctionDefine(schemaName, funcName string) (base.TableSource, error) {
	dbSchema, err := q.getDatabaseMetadata(q.connectedDB)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", q.connectedDB)
	}
	if dbSchema == nil {
		return nil, &parsererror.ResourceNotFoundError{
			Database: &q.connectedDB,
		}
	}
	if schemaName == "" {
		schemaName = "public"
	}
	schema := dbSchema.GetSchema(schemaName)
	if schema == nil {
		return nil, &parsererror.ResourceNotFoundError{
			Database: &q.connectedDB,
			Schema:   &schemaName,
		}
	}
	// find user defined function.
	function := schema.GetFunction(funcName)
	if function != nil {
		columns, err := q.getColumnsForFunction(fmt.Sprintf("%s.%s", schemaName, funcName), function.Definition)
		if err != nil {
			return nil, err
		}
		return &base.PseudoTable{
			Columns: columns,
		}, nil
	}
	// find system function.
	// https://www.postgresql.org/docs/current/functions-srf.html.
	switch funcName {
	case generateSeries:
		return &base.PseudoTable{
			Columns: []base.QuerySpanResult{
				{
					Name:          generateSeries,
					SourceColumns: make(base.SourceColumnSet),
				},
			},
		}, nil
	case generateSubscripts:
		return &base.PseudoTable{
			Columns: []base.QuerySpanResult{
				{
					Name:          generateSubscripts,
					SourceColumns: make(base.SourceColumnSet),
				},
			},
		}, nil
	}

	return nil, &parsererror.ResourceNotFoundError{
		Database: &q.connectedDB,
		Schema:   &schemaName,
		Function: &funcName,
	}
}

type languageType int

const (
	languageTypeSQL languageType = iota
	languageTypePLPGSQL
)

func (q *querySpanExtractor) getColumnsForFunction(name, definition string) ([]base.QuerySpanResult, error) {
	res, err := pgquery.Parse(definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse function definition: %s", definition)
	}
	if len(res.Stmts) != 1 {
		return nil, errors.Errorf("expecting 1 statement, but got %d", len(res.Stmts))
	}
	createFunc, ok := res.Stmts[0].Stmt.Node.(*pgquery.Node_CreateFunctionStmt)
	if !ok {
		return nil, errors.Errorf("expecting CreateFunctionStmt but got %T", res.Stmts[0].Stmt.Node)
	}
	var asBody string
	var language languageType
	for _, option := range createFunc.CreateFunctionStmt.Options {
		defElem := option.GetDefElem()
		if defElem == nil {
			continue
		}
		if strings.EqualFold(defElem.Defname, "as") {
			argList := defElem.Arg.GetList()
			if argList == nil {
				continue
			}
			for _, arg := range argList.Items {
				if arg == nil {
					continue
				}
				asBody = arg.GetString_().Sval
				break
			}
			continue
		}
		if strings.EqualFold(defElem.Defname, "language") {
			arg := defElem.Arg.GetString_()
			if arg == nil {
				continue
			}
			switch strings.ToLower(arg.Sval) {
			case "sql":
				language = languageTypeSQL
			case "plpgsql":
				language = languageTypePLPGSQL
			default:
				return nil, &parsererror.TypeNotSupportedError{
					Type:  "function language",
					Name:  arg.Sval,
					Extra: fmt.Sprintf("function: %s", name),
				}
			}
			continue
		}
	}
	if asBody == "" {
		return nil, errors.Errorf("expecting AS body but got empty for function: %s", name)
	}
	switch language {
	case languageTypeSQL:
		return q.extractTableSourceFromSQLFunction(createFunc, name, asBody)
	case languageTypePLPGSQL:
		return q.extractTableSourceFromPLPGSQLFunction(createFunc, name, definition)
	}
	return nil, errors.Errorf("unsupported language type: %d", language)
}

func (q *querySpanExtractor) extractTableSourceFromPLPGSQLFunction(createFunc *pgquery.Node_CreateFunctionStmt, name, definition string) ([]base.QuerySpanResult, error) {
	var columnNames []string
	for _, parameter := range createFunc.CreateFunctionStmt.Parameters {
		funcPara := parameter.GetFunctionParameter()
		if funcPara == nil {
			continue
		}
		switch funcPara.Mode {
		case pgquery.FunctionParameterMode_FUNC_PARAM_OUT, pgquery.FunctionParameterMode_FUNC_PARAM_TABLE:
			columnNames = append(columnNames, funcPara.Name)
		}
	}

	res, err := pgquery.ParsePlPgSqlToJSON(definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse PLPGSQL function definition: %s", definition)
	}

	var jsonData []any
	if err := json.Unmarshal([]byte(res), &jsonData); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal JSON")
	}

	var sqlList []string
	for _, value := range jsonData {
		switch value := value.(type) {
		case map[string]any:
			sqlList = append(sqlList, extractSQLListFromJSONData(value)...)
		case []any:
			for _, v := range value {
				if m, ok := v.(map[string]any); ok {
					sqlList = append(sqlList, extractSQLListFromJSONData(m)...)
				}
			}
		}
	}
	var leftQuerySpanResult []base.QuerySpanResult
	for _, columnName := range columnNames {
		leftQuerySpanResult = append(leftQuerySpanResult, base.QuerySpanResult{
			Name:          columnName,
			SourceColumns: base.SourceColumnSet{},
		})
	}

	for _, sql := range sqlList {
		newQ := newQuerySpanExtractor(q.connectedDB, q.f)
		span, err := newQ.getQuerySpan(q.ctx, sql)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get query span for function: %s", name)
		}
		for source := range span.SourceColumns {
			q.sourceColumnsInFunction[source] = true
		}

		rightQuerySpanResult := span.Results
		if len(leftQuerySpanResult) != len(rightQuerySpanResult) {
			return nil, errors.Errorf("expecting %d columns but got %d for function: %s", len(leftQuerySpanResult), len(rightQuerySpanResult), name)
		}
		var result []base.QuerySpanResult
		for i, leftSpanResult := range leftQuerySpanResult {
			rightSpanResult := rightQuerySpanResult[i]
			newResourceColumns, _ := base.MergeSourceColumnSet(leftSpanResult.SourceColumns, rightSpanResult.SourceColumns)
			result = append(result, base.QuerySpanResult{
				Name:          leftSpanResult.Name,
				SourceColumns: newResourceColumns,
			})
		}
		leftQuerySpanResult = result
	}

	return leftQuerySpanResult, nil
}

func extractSQLListFromJSONData(jsonData map[string]any) []string {
	var sqlList []string
	if jsonData["PLpgSQL_stmt_return_query"] != nil {
		sqlList = append(sqlList, extractSQL(jsonData["PLpgSQL_stmt_return_query"]))
	}

	for _, value := range jsonData {
		switch value := value.(type) {
		case map[string]any:
			sqlList = append(sqlList, extractSQLListFromJSONData(value)...)
		case []any:
			for _, v := range value {
				if m, ok := v.(map[string]any); ok {
					sqlList = append(sqlList, extractSQLListFromJSONData(m)...)
				}
			}
		}
	}

	return sqlList
}

func extractSQL(data any) string {
	if data == nil {
		return ""
	}
	switch data := data.(type) {
	case string:
		return data
	case map[string]any:
		switch {
		case data["query"] != nil:
			return extractSQL(data["query"])
		case data["PLpgSQL_expr"] != nil:
			return extractSQL(data["PLpgSQL_expr"])
		}
	}
	return ""
}

func (q *querySpanExtractor) extractTableSourceFromSQLFunction(createFunc *pgquery.Node_CreateFunctionStmt, name string, asBody string) ([]base.QuerySpanResult, error) {
	newQ := newQuerySpanExtractor(q.connectedDB, q.f)
	span, err := newQ.getQuerySpan(q.ctx, asBody)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span for function: %s", name)
	}
	for source := range span.SourceColumns {
		q.sourceColumnsInFunction[source] = true
	}

	var columnNames []string
	for _, parameter := range createFunc.CreateFunctionStmt.Parameters {
		funcPara := parameter.GetFunctionParameter()
		if funcPara == nil {
			continue
		}
		switch funcPara.Mode {
		case pgquery.FunctionParameterMode_FUNC_PARAM_OUT, pgquery.FunctionParameterMode_FUNC_PARAM_TABLE:
			columnNames = append(columnNames, funcPara.Name)
		}
	}
	if len(columnNames) != len(span.Results) {
		return nil, errors.Errorf("expecting %d columns but got %d for function: %s", len(columnNames), len(span.Results), name)
	}
	for i, columnName := range columnNames {
		span.Results[i].Name = columnName
	}
	return span.Results, nil
}

func (q *querySpanExtractor) extractTableSourceFromSelect(node *pgquery.Node_SelectStmt) (base.TableSource, error) {
	// We should reset the table sources from the FROM clause after exit the SELECT statement.
	previousTableSourcesFromLength := len(q.tableSourcesFrom)
	defer func() {
		q.tableSourcesFrom = q.tableSourcesFrom[:previousTableSourcesFromLength]
	}()

	// The WITH clause.
	if node.SelectStmt.WithClause != nil {
		previousCteOuterLength := len(q.ctes)
		defer func() {
			q.ctes = q.ctes[:previousCteOuterLength]
		}()

		for _, cte := range node.SelectStmt.WithClause.Ctes {
			cteExpr, ok := cte.Node.(*pgquery.Node_CommonTableExpr)
			if !ok {
				return nil, errors.Errorf("expect CommonTableExpr for CTE, but got %T", cte.Node)
			}
			var cteTableSource *base.PseudoTable
			var err error
			if node.SelectStmt.WithClause.Recursive {
				cteTableSource, err = q.extractTemporaryTableResourceFromRecursiveCTE(cteExpr)
			} else {
				cteTableSource, err = q.extractTemporaryTableResourceFromNonRecursiveCTE(cteExpr)
			}
			if err != nil {
				return nil, err
			}
			q.ctes = append(q.ctes, cteTableSource)
		}
	}

	// The VALUES case.
	// https://www.postgresql.org/docs/current/queries-values.html
	if len(node.SelectStmt.ValuesLists) > 0 {
		var columnSourceSets []base.SourceColumnSet
		for _, row := range node.SelectStmt.ValuesLists {
			list, ok := row.Node.(*pgquery.Node_List)
			if !ok {
				return nil, errors.Errorf("expect List for VALUES list, but got %T", row.Node)
			}
			for i, value := range list.List.Items {
				sourceColumnSet, err := q.extractSourceColumnSetFromExpressionNode(value)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract source column set from VALUES expression: %+v", value)
				}
				if i >= len(columnSourceSets) {
					columnSourceSets = append(columnSourceSets, sourceColumnSet)
				} else {
					columnSourceSets[i], _ = base.MergeSourceColumnSet(columnSourceSets[i], sourceColumnSet)
				}
			}
		}

		var querySpanResults []base.QuerySpanResult
		for i, columnSourceSet := range columnSourceSets {
			querySpanResults = append(querySpanResults, base.QuerySpanResult{
				Name:          fmt.Sprintf("column%d", i+1),
				SourceColumns: columnSourceSet,
			})
		}
		// FIXME(zp): Consider the alias case to give a name to table.
		// => SELECT * FROM (VALUES (1, 'one'), (2, 'two'), (3, 'three')) AS t (num,letter);
		return &base.PseudoTable{
			Name:    "",
			Columns: querySpanResults,
		}, nil
	}

	// UNION/INTERSECT/EXCEPT case.
	switch node.SelectStmt.Op {
	case pgquery.SetOperation_SETOP_UNION, pgquery.SetOperation_SETOP_INTERSECT, pgquery.SetOperation_SETOP_EXCEPT:
		leftSpanResults, err := q.extractTableSourceFromSelect(&pgquery.Node_SelectStmt{SelectStmt: node.SelectStmt.Larg})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract span result from left select: %+v", node.SelectStmt.Larg)
		}
		rightSpanResults, err := q.extractTableSourceFromSelect(&pgquery.Node_SelectStmt{SelectStmt: node.SelectStmt.Rarg})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract span result from right select: %+v", node.SelectStmt.Rarg)
		}
		leftQuerySpanResult, rightQuerySpanResult := leftSpanResults.GetQuerySpanResult(), rightSpanResults.GetQuerySpanResult()
		if len(leftQuerySpanResult) != len(rightQuerySpanResult) {
			return nil, errors.Errorf("left select has %d columns, but right select has %d columns", len(leftQuerySpanResult), len(rightQuerySpanResult))
		}
		var result []base.QuerySpanResult
		for i, leftSpanResult := range leftQuerySpanResult {
			rightSpanResult := rightQuerySpanResult[i]
			newResourceColumns, _ := base.MergeSourceColumnSet(leftSpanResult.SourceColumns, rightSpanResult.SourceColumns)
			result = append(result, base.QuerySpanResult{
				Name:          leftSpanResult.Name,
				SourceColumns: newResourceColumns,
			})
		}
		// FIXME(zp): Consider UNION alias.
		return &base.PseudoTable{
			Name:    "",
			Columns: result,
		}, nil
	case pgquery.SetOperation_SETOP_NONE:
	default:
		return nil, errors.Errorf("unsupported set operation: %s", node.SelectStmt.Op)
	}

	// The FROM clause.
	var fromFieldList []base.TableSource
	for _, item := range node.SelectStmt.FromClause {
		tableSource, err := q.extractTableSourceFromNode(item)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract span result from FROM item: %+v", item)
		}
		q.tableSourcesFrom = append(q.tableSourcesFrom, tableSource)
		fromFieldList = append(fromFieldList, tableSource)
	}

	// The TARGET field list.
	// The SELECT statement is really create a pseudo table, and this pseudo table will be shown as the result.
	result := new(base.PseudoTable)
	for _, field := range node.SelectStmt.TargetList {
		resTarget, ok := field.Node.(*pgquery.Node_ResTarget)
		if !ok {
			return nil, errors.Errorf("expect ResTarget for SELECT target, but got %T", field.Node)
		}
		switch fieldNode := resTarget.ResTarget.Val.Node.(type) {
		case *pgquery.Node_ColumnRef:
			columnRef, err := pgrawparser.ConvertNodeListToColumnNameDef(fieldNode.ColumnRef.Fields)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to convert column ref to column name def: %+v", fieldNode.ColumnRef.Fields)
			}
			if columnRef.ColumnName == "*" {
				// SELECT [x].* FROM ... case
				if columnRef.Table.Name == "" {
					var columns []base.QuerySpanResult
					for _, tableSource := range fromFieldList {
						columns = append(columns, tableSource.GetQuerySpanResult()...)
					}
					result.Columns = append(result.Columns, columns...)
				} else {
					schemaName, tableName, _ := extractSchemaTableColumnName(columnRef)
					find := false
					// TODO(zp): remove iterate fromFieldList because we had append it to q.tableSourcesFrom.
					for _, tableSource := range fromFieldList {
						if schemaName == "" || schemaName == tableSource.GetSchemaName() {
							if tableName == tableSource.GetTableName() {
								find = true
								result.Columns = append(result.Columns, tableSource.GetQuerySpanResult()...)
								break
							}
						}
					}
					if !find {
						sources, ok := q.getAllTableColumnSources(schemaName, tableName)
						if ok {
							result.Columns = append(result.Columns, sources...)
							find = true
						}
					}
					if !find {
						return nil, &parsererror.ResourceNotFoundError{
							Err:    errors.New("failed to find table to calculate asterisk"),
							Schema: &schemaName,
							Table:  &tableName,
						}
					}
				}
			} else {
				sourceColumnSet, err := q.extractSourceColumnSetFromExpressionNode(resTarget.ResTarget.Val)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract source column set from expression: %+v", resTarget.ResTarget.Val)
				}
				// XXX(zp): Should we handle the AS case?
				columnName := columnRef.ColumnName
				if resTarget.ResTarget.Name != "" {
					columnName = resTarget.ResTarget.Name
				}
				result.Columns = append(result.Columns, base.QuerySpanResult{
					Name:          columnName,
					SourceColumns: sourceColumnSet,
				})
			}
		default:
			sourceColumnSet, err := q.extractSourceColumnSetFromExpressionNode(resTarget.ResTarget.Val)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract source column set from expression: %+v", resTarget.ResTarget.Val)
			}
			// XXX(zp): Should we handle the AS case?
			columnName := resTarget.ResTarget.Name
			if columnName == "" {
				if columnName, err = pgExtractFieldName(resTarget.ResTarget.Val); err != nil {
					return nil, errors.Wrapf(err, "failed to extract field name from expression: %+v", resTarget.ResTarget.Val)
				}
			}
			result.Columns = append(result.Columns, base.QuerySpanResult{
				Name:          columnName,
				SourceColumns: sourceColumnSet,
			})
		}
	}

	return result, nil
}

func (q *querySpanExtractor) extractTableSourceFromJoin(node *pgquery.Node_JoinExpr) (*base.PseudoTable, error) {
	leftTableSource, err := q.extractTableSourceFromNode(node.JoinExpr.Larg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract span result from left join: %+v", node.JoinExpr.Larg)
	}
	q.tableSourcesFrom = append(q.tableSourcesFrom, leftTableSource)
	rightTableSource, err := q.extractTableSourceFromNode(node.JoinExpr.Rarg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract span result from right join: %+v", node.JoinExpr.Rarg)
	}
	q.tableSourcesFrom = append(q.tableSourcesFrom, rightTableSource)
	return q.mergeJoinTableSource(node, leftTableSource, rightTableSource)
}

func (*querySpanExtractor) mergeJoinTableSource(node *pgquery.Node_JoinExpr, left base.TableSource, right base.TableSource) (*base.PseudoTable, error) {
	leftSpanResult, rightSpanResult := left.GetQuerySpanResult(), right.GetQuerySpanResult()

	result := new(base.PseudoTable)

	if node.JoinExpr.IsNatural {
		leftSpanResultIdx, rightSpanResultIdx := make(map[string]int, len(leftSpanResult)), make(map[string]int, len(rightSpanResult))
		for i, spanResult := range leftSpanResult {
			leftSpanResultIdx[spanResult.Name] = i
		}
		for i, spanResult := range rightSpanResult {
			rightSpanResultIdx[spanResult.Name] = i
		}

		// NaturalJoin will merge the same column name field.
		for idx, spanResult := range leftSpanResult {
			if _, ok := rightSpanResultIdx[spanResult.Name]; ok {
				spanResult.SourceColumns, _ = base.MergeSourceColumnSet(spanResult.SourceColumns, rightSpanResult[idx].SourceColumns)
			}
			result.Columns = append(result.Columns, spanResult)
		}
		for _, spanResult := range rightSpanResult {
			if _, ok := leftSpanResultIdx[spanResult.Name]; !ok {
				result.Columns = append(result.Columns, spanResult)
			}
		}
	} else {
		if len(node.JoinExpr.UsingClause) > 0 {
			// ... JOIN ... USING (...) will merge the column in USING.
			usingMap := make(map[string]bool, len(node.JoinExpr.UsingClause))
			for _, using := range node.JoinExpr.UsingClause {
				name, ok := using.Node.(*pgquery.Node_String_)
				if !ok {
					return nil, errors.Errorf("expect Node_String_ for using clause, but got %T", using.Node)
				}
				usingMap[name.String_.Sval] = true
			}

			result.Columns = append(result.Columns, leftSpanResult...)
			for _, spanResult := range rightSpanResult {
				if _, ok := usingMap[spanResult.Name]; !ok {
					result.Columns = append(result.Columns, spanResult)
				}
			}
		} else {
			result.Columns = append(result.Columns, leftSpanResult...)
			result.Columns = append(result.Columns, rightSpanResult...)
		}
	}

	if node.JoinExpr.Alias != nil {
		// The AS case
		aliasName, columnNameList, err := pgExtractAlias(node.JoinExpr.Alias)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract alias from join: %+v", node)
		}
		if len(columnNameList) != 0 && len(columnNameList) != len(result.Columns) {
			return nil, errors.Errorf("expect equal length but found %d and %d", len(node.JoinExpr.Alias.Colnames), len(result.Columns))
		}

		result.Name = aliasName
		if len(columnNameList) == 0 {
			return result, nil
		}

		var columns []base.QuerySpanResult
		for i, columnName := range columnNameList {
			columns = append(columns, base.QuerySpanResult{
				Name:          columnName,
				SourceColumns: result.Columns[i].SourceColumns,
			})
		}
		result.Columns = columns
	}
	return result, nil
}

func (q *querySpanExtractor) extractTableSourceFromRangeVar(node *pgquery.Node_RangeVar) (base.TableSource, error) {
	tableSource, err := q.findTableSchema(node.RangeVar.Schemaname, node.RangeVar.Relname)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find table schema for range var: %+v", node)
	}

	if node.RangeVar.Alias == nil {
		return tableSource, nil
	}

	// The AS case
	aliasName, columnNameList, err := pgExtractAlias(node.RangeVar.Alias)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract alias from range var: %+v", node)
	}

	querySpanResult := tableSource.GetQuerySpanResult()
	if len(columnNameList) != 0 && len(columnNameList) != len(querySpanResult) {
		return nil, errors.Errorf("expect equal length but found %d and %d", len(node.RangeVar.Alias.Colnames), len(tableSource.GetQuerySpanResult()))
	}

	tableName := aliasName
	if len(columnNameList) == 0 {
		return base.NewPseudoTable(tableName, querySpanResult), nil
	}

	var columns []base.QuerySpanResult
	if len(columnNameList) > 0 {
		for i, columnName := range columnNameList {
			columns = append(columns, base.QuerySpanResult{
				Name:          columnName,
				SourceColumns: querySpanResult[i].SourceColumns,
			})
		}
	}
	return base.NewPseudoTable(tableName, columns), nil
}

func (q *querySpanExtractor) extractTableSourceFromSubselect(node *pgquery.Node_RangeSubselect) (base.TableSource, error) {
	tableSource, err := q.extractTableSourceFromNode(node.RangeSubselect.Subquery)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract span result from subselect: %+v", node)
	}
	if node.RangeSubselect.Alias == nil {
		return tableSource, nil
	}

	// The AS case
	aliasName, columnNameList, err := pgExtractAlias(node.RangeSubselect.Alias)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract alias from range var: %+v", node)
	}
	querySpanResult := tableSource.GetQuerySpanResult()
	if len(columnNameList) != 0 && len(columnNameList) != len(querySpanResult) {
		return nil, errors.Errorf("expect equal length but found %d and %d", len(columnNameList), len(tableSource.GetQuerySpanResult()))
	}

	tableName := aliasName
	if len(columnNameList) == 0 {
		return base.NewPseudoTable(tableName, querySpanResult), nil
	}

	var columns []base.QuerySpanResult
	for i, columnName := range columnNameList {
		columns = append(columns, base.QuerySpanResult{
			Name:          columnName,
			SourceColumns: querySpanResult[i].SourceColumns,
		})
	}
	return base.NewPseudoTable(tableName, columns), nil
}

func (q *querySpanExtractor) extractTemporaryTableResourceFromNonRecursiveCTE(cteExpr *pgquery.Node_CommonTableExpr) (*base.PseudoTable, error) {
	tableSource, err := q.extractTableSourceFromNode(cteExpr.CommonTableExpr.Ctequery)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract span result from CTE query: %+v", cteExpr.CommonTableExpr.Ctequery)
	}

	querySpanResults := tableSource.GetQuerySpanResult()
	if len(cteExpr.CommonTableExpr.Aliascolnames) > 0 {
		if len(cteExpr.CommonTableExpr.Aliascolnames) != len(querySpanResults) {
			return nil, errors.Errorf("cte table expr has %d columns, but alias has %d columns", len(querySpanResults), len(cteExpr.CommonTableExpr.Aliascolnames))
		}
		for i, name := range cteExpr.CommonTableExpr.Aliascolnames {
			stringNode, ok := name.Node.(*pgquery.Node_String_)
			if !ok {
				return nil, errors.Errorf("expect string node for alias column name, but got %T", name.Node)
			}
			querySpanResults[i].Name = stringNode.String_.Sval
		}
	}

	return &base.PseudoTable{
		Name:    cteExpr.CommonTableExpr.Ctename,
		Columns: querySpanResults,
	}, nil
}

func (q *querySpanExtractor) extractTemporaryTableResourceFromRecursiveCTE(cteExpr *pgquery.Node_CommonTableExpr) (*base.PseudoTable, error) {
	switch selectNode := cteExpr.CommonTableExpr.Ctequery.Node.(type) {
	case *pgquery.Node_SelectStmt:
		if selectNode.SelectStmt.Op != pgquery.SetOperation_SETOP_UNION {
			return q.extractTemporaryTableResourceFromNonRecursiveCTE(cteExpr)
		}
		// For PostgreSQL, recursive CTE would be a UNION statement, and the left node is the initial part,
		// the right node is the recursive part.
		// https://www.postgresql.org/docs/15/queries-with.html#QUERIES-WITH-RECURSIVE
		initialTableSource, err := q.extractTableSourceFromSelect(&pgquery.Node_SelectStmt{SelectStmt: selectNode.SelectStmt.Larg})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract span result from CTE initial query: %+v", selectNode.SelectStmt.Larg)
		}
		initialQuerySpanResult := initialTableSource.GetQuerySpanResult()
		if len(cteExpr.CommonTableExpr.Aliascolnames) > 0 {
			if len(cteExpr.CommonTableExpr.Aliascolnames) != len(initialQuerySpanResult) {
				return nil, errors.Errorf("cte table expr has %d columns, but alias has %d columns", len(initialQuerySpanResult), len(cteExpr.CommonTableExpr.Aliascolnames))
			}
			for i, name := range cteExpr.CommonTableExpr.Aliascolnames {
				stringNode, ok := name.Node.(*pgquery.Node_String_)
				if !ok {
					return nil, errors.Errorf("expect string node for alias column name, but got %T", name.Node)
				}
				initialQuerySpanResult[i].Name = stringNode.String_.Sval
			}
		}

		cteTableResource := &base.PseudoTable{Name: cteExpr.CommonTableExpr.Ctename, Columns: initialQuerySpanResult}

		// Compute dependent closures.
		// There are two ways to compute dependent closures:
		//   1. find the all dependent edges, then use graph theory traversal to find the closure.
		//   2. Iterate to simulate the CTE recursive process, each turn check whether the columns have changed, and stop if not change.
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
			recursiveTableSource, err := q.extractTableSourceFromSelect(&pgquery.Node_SelectStmt{SelectStmt: selectNode.SelectStmt.Rarg})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract span result from CTE recursive query: %+v", selectNode.SelectStmt.Rarg)
			}
			recursiveQuerySpanResult := recursiveTableSource.GetQuerySpanResult()
			if len(recursiveQuerySpanResult) != len(initialQuerySpanResult) {
				return nil, errors.Errorf("cte table expr has %d columns, but recursive query has %d columns", len(initialQuerySpanResult), len(recursiveQuerySpanResult))
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
		return q.extractTemporaryTableResourceFromNonRecursiveCTE(cteExpr)
	}
}

func (q *querySpanExtractor) extractSourceColumnSetFromExpressionNode(node *pgquery.Node) (base.SourceColumnSet, error) {
	if node == nil {
		return base.SourceColumnSet{}, nil
	}

	switch node := node.Node.(type) {
	case *pgquery.Node_List:
		return q.extractSourceColumnSetFromExpressionNodeList(node.List.Items)
	case *pgquery.Node_FuncCall:
		var nodeList []*pgquery.Node
		nodeList = append(nodeList, node.FuncCall.Args...)
		nodeList = append(nodeList, node.FuncCall.AggOrder...)
		nodeList = append(nodeList, node.FuncCall.AggFilter)
		return q.extractSourceColumnSetFromExpressionNodeList(nodeList)
	case *pgquery.Node_SortBy:
		return q.extractSourceColumnSetFromExpressionNode(node.SortBy.Node)
	case *pgquery.Node_XmlExpr:
		var nodeList []*pgquery.Node
		nodeList = append(nodeList, node.XmlExpr.Args...)
		nodeList = append(nodeList, node.XmlExpr.NamedArgs...)
		return q.extractSourceColumnSetFromExpressionNodeList(nodeList)
	case *pgquery.Node_ResTarget:
		return q.extractSourceColumnSetFromExpressionNode(node.ResTarget.Val)
	case *pgquery.Node_TypeCast:
		return q.extractSourceColumnSetFromExpressionNode(node.TypeCast.Arg)
	case *pgquery.Node_AConst:
		return base.SourceColumnSet{}, nil
	case *pgquery.Node_ColumnRef:
		columnNameDef, err := pgrawparser.ConvertNodeListToColumnNameDef(node.ColumnRef.Fields)
		if err != nil {
			return base.SourceColumnSet{}, err
		}
		schema, table, column := extractSchemaTableColumnName(columnNameDef)
		sources, ok := q.getFieldColumnSource(schema, table, column)
		if !ok {
			return base.SourceColumnSet{}, &parsererror.ResourceNotFoundError{
				Err:      errors.New("cannot find the column ref"),
				Database: &q.connectedDB,
				Schema:   &schema,
				Table:    &table,
				Column:   &column,
			}
		}
		return sources, nil
	case *pgquery.Node_AExpr:
		var nodeList []*pgquery.Node
		nodeList = append(nodeList, node.AExpr.Lexpr)
		nodeList = append(nodeList, node.AExpr.Rexpr)
		return q.extractSourceColumnSetFromExpressionNodeList(nodeList)
	case *pgquery.Node_CaseExpr:
		var nodeList []*pgquery.Node
		nodeList = append(nodeList, node.CaseExpr.Arg)
		nodeList = append(nodeList, node.CaseExpr.Args...)
		nodeList = append(nodeList, node.CaseExpr.Defresult)
		return q.extractSourceColumnSetFromExpressionNodeList(nodeList)
	case *pgquery.Node_CaseWhen:
		var nodeList []*pgquery.Node
		nodeList = append(nodeList, node.CaseWhen.Expr)
		nodeList = append(nodeList, node.CaseWhen.Result)
		return q.extractSourceColumnSetFromExpressionNodeList(nodeList)
	case *pgquery.Node_AArrayExpr:
		return q.extractSourceColumnSetFromExpressionNodeList(node.AArrayExpr.Elements)
	case *pgquery.Node_NullTest:
		return q.extractSourceColumnSetFromExpressionNode(node.NullTest.Arg)
	case *pgquery.Node_XmlSerialize:
		return q.extractSourceColumnSetFromExpressionNode(node.XmlSerialize.Expr)
	case *pgquery.Node_ParamRef:
		return base.SourceColumnSet{}, nil
	case *pgquery.Node_BoolExpr:
		return q.extractSourceColumnSetFromExpressionNodeList(node.BoolExpr.Args)
	case *pgquery.Node_SubLink:
		sourceColumnSet, err := q.extractSourceColumnSetFromExpressionNode(node.SubLink.Testexpr)
		if err != nil {
			return base.SourceColumnSet{}, err
		}
		// Subquery in SELECT fields is special.
		// It can be the non-associated or associated subquery.
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &querySpanExtractor{
			ctx:               q.ctx,
			connectedDB:       q.connectedDB,
			metaCache:         q.metaCache,
			f:                 q.f,
			ctes:              q.ctes,
			outerTableSources: append(q.outerTableSources, q.tableSourcesFrom...),
			tableSourcesFrom:  []base.TableSource{},
		}
		tableSource, err := subqueryExtractor.extractTableSourceFromNode(node.SubLink.Subselect)
		if err != nil {
			return base.SourceColumnSet{}, errors.Wrapf(err, "failed to extract span result from sublink: %+v", node.SubLink.Subselect)
		}
		spanResult := tableSource.GetQuerySpanResult()

		for _, field := range spanResult {
			sourceColumnSet, _ = base.MergeSourceColumnSet(sourceColumnSet, field.SourceColumns)
		}
		return sourceColumnSet, nil
	case *pgquery.Node_RowExpr:
		return q.extractSourceColumnSetFromExpressionNodeList(node.RowExpr.Args)
	case *pgquery.Node_CoalesceExpr:
		return q.extractSourceColumnSetFromExpressionNodeList(node.CoalesceExpr.Args)
	case *pgquery.Node_SetToDefault:
		return base.SourceColumnSet{}, nil
	case *pgquery.Node_AIndirection:
		return q.extractSourceColumnSetFromExpressionNode(node.AIndirection.Arg)
	case *pgquery.Node_CollateClause:
		return q.extractSourceColumnSetFromExpressionNode(node.CollateClause.Arg)
	case *pgquery.Node_CurrentOfExpr:
		return base.SourceColumnSet{}, nil
	case *pgquery.Node_SqlvalueFunction:
		return base.SourceColumnSet{}, nil
	case *pgquery.Node_MinMaxExpr:
		return q.extractSourceColumnSetFromExpressionNodeList(node.MinMaxExpr.Args)
	case *pgquery.Node_BooleanTest:
		return q.extractSourceColumnSetFromExpressionNode(node.BooleanTest.Arg)
	case *pgquery.Node_GroupingFunc:
		return q.extractSourceColumnSetFromExpressionNodeList(node.GroupingFunc.Args)
	}
	return base.SourceColumnSet{}, nil
}

// extractSourceColumnSetFromExpressionNodeList is the helper function to extract the source column set from the given expression node list,
// which iterates the list and merge each set.
func (q *querySpanExtractor) extractSourceColumnSetFromExpressionNodeList(list []*pgquery.Node) (base.SourceColumnSet, error) {
	result := make(base.SourceColumnSet)
	for _, node := range list {
		sourceColumnSet, err := q.extractSourceColumnSetFromExpressionNode(node)
		if err != nil {
			return nil, err
		}
		result, _ = base.MergeSourceColumnSet(result, sourceColumnSet)
	}
	return result, nil
}

func (q *querySpanExtractor) getAllTableColumnSources(schemaName, tableName string) ([]base.QuerySpanResult, bool) {
	findInTableSource := func(tableSource base.TableSource) ([]base.QuerySpanResult, bool) {
		if schemaName != "" && schemaName != tableSource.GetSchemaName() {
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
		if sourceColumnSet, ok := findInTableSource(q.outerTableSources[i]); ok {
			return sourceColumnSet, true
		}
	}

	for _, tableSource := range q.tableSourcesFrom {
		if sourceColumnSet, ok := findInTableSource(tableSource); ok {
			return sourceColumnSet, true
		}
	}

	return []base.QuerySpanResult{}, false
}

func (q *querySpanExtractor) getFieldColumnSource(schemaName, tableName, fieldName string) (base.SourceColumnSet, bool) {
	findInTableSource := func(tableSource base.TableSource) (base.SourceColumnSet, bool) {
		if schemaName != "" && schemaName != tableSource.GetSchemaName() {
			return nil, false
		}
		if tableName != "" && tableName != tableSource.GetTableName() {
			return nil, false
		}
		// If the table name is empty, we should check if there are ambiguous fields,
		// but we delegate this responsibility to the db-server, we do the fail-open strategy here.

		querySpanResult := tableSource.GetQuerySpanResult()
		for _, field := range querySpanResult {
			if field.Name == fieldName {
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
		if sourceColumnSet, ok := findInTableSource(q.outerTableSources[i]); ok {
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

func (q *querySpanExtractor) findTableSchema(schemaName string, tableName string) (base.TableSource, error) {
	// Each CTE name in one WITH clause must be unique, but we can use the same name in the different level CTE, such as:
	//
	//  with tt2 as (
	//    with tt2 as (select * from t)
	//    select max(a) from tt2)
	//  select * from tt2
	//
	// This query has two CTE can be called `tt2`, and the FROM clause 'from tt2' uses the closer tt2 CTE.
	// This is the reason we loop the slice in reversed order.
	if schemaName == "" {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			table := q.ctes[i]
			if table.Name == tableName {
				return table, nil
			}
		}
	}

	// FIXME: consider cross database query which is supported in Redshift.
	dbSchema, err := q.getDatabaseMetadata(q.connectedDB)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", q.connectedDB)
	}
	if dbSchema == nil {
		return nil, &parsererror.ResourceNotFoundError{
			Database: &q.connectedDB,
		}
	}
	if schemaName == "" {
		schemaName = "public"
	}
	schema := dbSchema.GetSchema(schemaName)
	if schema == nil {
		return nil, &parsererror.ResourceNotFoundError{
			Database: &q.connectedDB,
			Schema:   &schemaName,
		}
	}
	table := schema.GetTable(tableName)
	view := schema.GetView(tableName)
	materializedView := schema.GetMaterializedView(tableName)
	foreignTable := schema.GetExternalTable(tableName)
	if table == nil && view == nil && foreignTable == nil && materializedView == nil {
		return nil, &parsererror.ResourceNotFoundError{
			Database: &q.connectedDB,
			Schema:   &schemaName,
			Table:    &tableName,
		}
	}

	if table != nil {
		var columns []string
		for _, column := range table.GetColumns() {
			columns = append(columns, column.Name)
		}
		return &base.PhysicalTable{
			Server:   "",
			Database: q.connectedDB,
			Schema:   schemaName,
			Name:     tableName,
			Columns:  columns,
		}, nil
	}

	if foreignTable != nil {
		var columns []string
		for _, column := range foreignTable.GetColumns() {
			columns = append(columns, column.Name)
		}
		return &base.PhysicalTable{
			Server:   "",
			Database: q.connectedDB,
			Schema:   schemaName,
			Name:     tableName,
			Columns:  columns,
		}, nil
	}

	if view != nil && view.Definition != "" {
		columns, err := q.getColumnsForView(view.Definition)
		if err != nil {
			return nil, err
		}
		return &base.PhysicalView{
			Server:   "",
			Database: q.connectedDB,
			Schema:   schemaName,
			Name:     tableName,
			Columns:  columns,
		}, nil
	}

	if materializedView != nil && materializedView.Definition != "" {
		columns, err := q.getColumnsForMaterializedView(materializedView.Definition)
		if err != nil {
			return nil, err
		}
		return &base.PhysicalView{
			Server:   "",
			Database: q.connectedDB,
			Schema:   schemaName,
			Name:     tableName,
			Columns:  columns,
		}, nil
	}
	return nil, nil
}

func extractSchemaTableColumnName(columnName *ast.ColumnNameDef) (string, string, string) {
	return columnName.Table.Schema, columnName.Table.Name, columnName.ColumnName
}

func pgExtractAlias(alias *pgquery.Alias) (string, []string, error) {
	if alias == nil {
		return "", nil, nil
	}
	var columnNameList []string
	for _, item := range alias.Colnames {
		stringNode, yes := item.Node.(*pgquery.Node_String_)
		if !yes {
			return "", nil, errors.Errorf("expect Node_String_ but found %T", item.Node)
		}
		columnNameList = append(columnNameList, stringNode.String_.Sval)
	}
	return alias.Aliasname, columnNameList, nil
}

func pgExtractFieldName(node *pgquery.Node) (string, error) {
	if node == nil || node.Node == nil {
		return pgUnknownFieldName, nil
	}
	switch node := node.Node.(type) {
	case *pgquery.Node_ResTarget:
		if node.ResTarget.Name != "" {
			return node.ResTarget.Name, nil
		}
		return pgExtractFieldName(node.ResTarget.Val)
	case *pgquery.Node_ColumnRef:
		columnRef, err := pgrawparser.ConvertNodeListToColumnNameDef(node.ColumnRef.Fields)
		if err != nil {
			return "", err
		}
		return columnRef.ColumnName, nil
	case *pgquery.Node_FuncCall:
		lastNode, yes := node.FuncCall.Funcname[len(node.FuncCall.Funcname)-1].Node.(*pgquery.Node_String_)
		if !yes {
			return "", errors.Errorf("expect Node_string_ but found %T", node.FuncCall.Funcname[len(node.FuncCall.Funcname)-1].Node)
		}
		return lastNode.String_.Sval, nil
	case *pgquery.Node_XmlExpr:
		switch node.XmlExpr.Op {
		case pgquery.XmlExprOp_IS_XMLCONCAT:
			return "xmlconcat", nil
		case pgquery.XmlExprOp_IS_XMLELEMENT:
			return "xmlelement", nil
		case pgquery.XmlExprOp_IS_XMLFOREST:
			return "xmlforest", nil
		case pgquery.XmlExprOp_IS_XMLPARSE:
			return "xmlparse", nil
		case pgquery.XmlExprOp_IS_XMLPI:
			return "xmlpi", nil
		case pgquery.XmlExprOp_IS_XMLROOT:
			return "xmlroot", nil
		case pgquery.XmlExprOp_IS_XMLSERIALIZE:
			return "xmlserialize", nil
		case pgquery.XmlExprOp_IS_DOCUMENT:
			return pgUnknownFieldName, nil
		}
	case *pgquery.Node_TypeCast:
		// return the arg name
		columnName, err := pgExtractFieldName(node.TypeCast.Arg)
		if err != nil {
			return "", err
		}
		if columnName != pgUnknownFieldName {
			return columnName, nil
		}
		// return the type name
		if node.TypeCast.TypeName != nil {
			lastName, yes := node.TypeCast.TypeName.Names[len(node.TypeCast.TypeName.Names)-1].Node.(*pgquery.Node_String_)
			if !yes {
				return "", errors.Errorf("expect Node_string_ but found %T", node.TypeCast.TypeName.Names[len(node.TypeCast.TypeName.Names)-1].Node)
			}
			return lastName.String_.Sval, nil
		}
	case *pgquery.Node_AConst:
		return pgUnknownFieldName, nil
	case *pgquery.Node_AExpr:
		return pgUnknownFieldName, nil
	case *pgquery.Node_CaseExpr:
		return "case", nil
	case *pgquery.Node_AArrayExpr:
		return "array", nil
	case *pgquery.Node_NullTest:
		return pgUnknownFieldName, nil
	case *pgquery.Node_XmlSerialize:
		return "xmlserialize", nil
	case *pgquery.Node_ParamRef:
		return pgUnknownFieldName, nil
	case *pgquery.Node_BoolExpr:
		return pgUnknownFieldName, nil
	case *pgquery.Node_SubLink:
		switch node.SubLink.SubLinkType {
		case pgquery.SubLinkType_EXISTS_SUBLINK:
			return "exists", nil
		case pgquery.SubLinkType_ARRAY_SUBLINK:
			return "array", nil
		case pgquery.SubLinkType_EXPR_SUBLINK:
			if node.SubLink.Subselect != nil {
				selectNode, yes := node.SubLink.Subselect.Node.(*pgquery.Node_SelectStmt)
				if !yes {
					return pgUnknownFieldName, nil
				}
				if len(selectNode.SelectStmt.TargetList) == 1 {
					return pgExtractFieldName(selectNode.SelectStmt.TargetList[0])
				}
				return pgUnknownFieldName, nil
			}
		default:
			return pgUnknownFieldName, nil
		}
	case *pgquery.Node_RowExpr:
		return "row", nil
	case *pgquery.Node_CoalesceExpr:
		return "coalesce", nil
	case *pgquery.Node_SetToDefault:
		return pgUnknownFieldName, nil
	case *pgquery.Node_AIndirection:
		// TODO(rebelice): we do not deal with the A_Indirection. Fix it.
		return pgUnknownFieldName, nil
	case *pgquery.Node_CollateClause:
		return pgExtractFieldName(node.CollateClause.Arg)
	case *pgquery.Node_CurrentOfExpr:
		return pgUnknownFieldName, nil
	case *pgquery.Node_SqlvalueFunction:
		switch node.SqlvalueFunction.Op {
		case pgquery.SQLValueFunctionOp_SVFOP_CURRENT_DATE:
			return "current_date", nil
		case pgquery.SQLValueFunctionOp_SVFOP_CURRENT_TIME, pgquery.SQLValueFunctionOp_SVFOP_CURRENT_TIME_N:
			return "current_time", nil
		case pgquery.SQLValueFunctionOp_SVFOP_CURRENT_TIMESTAMP, pgquery.SQLValueFunctionOp_SVFOP_CURRENT_TIMESTAMP_N:
			return "current_timestamp", nil
		case pgquery.SQLValueFunctionOp_SVFOP_LOCALTIME, pgquery.SQLValueFunctionOp_SVFOP_LOCALTIME_N:
			return "localtime", nil
		case pgquery.SQLValueFunctionOp_SVFOP_LOCALTIMESTAMP, pgquery.SQLValueFunctionOp_SVFOP_LOCALTIMESTAMP_N:
			return "localtimestamp", nil
		case pgquery.SQLValueFunctionOp_SVFOP_CURRENT_ROLE:
			return "current_role", nil
		case pgquery.SQLValueFunctionOp_SVFOP_CURRENT_USER:
			return "current_user", nil
		case pgquery.SQLValueFunctionOp_SVFOP_USER:
			return "user", nil
		case pgquery.SQLValueFunctionOp_SVFOP_SESSION_USER:
			return "session_user", nil
		case pgquery.SQLValueFunctionOp_SVFOP_CURRENT_CATALOG:
			return "current_catalog", nil
		case pgquery.SQLValueFunctionOp_SVFOP_CURRENT_SCHEMA:
			return "current_schema", nil
		default:
			return pgUnknownFieldName, nil
		}
	case *pgquery.Node_MinMaxExpr:
		switch node.MinMaxExpr.Op {
		case pgquery.MinMaxOp_IS_GREATEST:
			return "greatest", nil
		case pgquery.MinMaxOp_IS_LEAST:
			return "least", nil
		default:
			return pgUnknownFieldName, nil
		}
	case *pgquery.Node_BooleanTest:
		return pgUnknownFieldName, nil
	case *pgquery.Node_GroupingFunc:
		return "grouping", nil
	}
	return pgUnknownFieldName, nil
}

func (q *querySpanExtractor) getAccessTables(sql string) (base.SourceColumnSet, error) {
	jsonText, err := pgquery.ParseToJSON(sql)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse sql to json")
	}

	var jsonData map[string]any

	if err := json.Unmarshal([]byte(jsonText), &jsonData); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal json")
	}

	accessesMap := make(base.SourceColumnSet)

	result, err := q.getRangeVarsFromJSONRecursive(jsonData, q.connectedDB, "public")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get range vars from json")
	}
	for _, resource := range result {
		accessesMap[resource] = true
	}

	return accessesMap, nil
}

func (q *querySpanExtractor) getRangeVarsFromJSONRecursive(jsonData map[string]any, currentDatabase, currentSchema string) ([]base.ColumnResource, error) {
	var result []base.ColumnResource
	if jsonData["RangeVar"] != nil {
		resource := base.ColumnResource{
			Server:   "",
			Database: "",
			Schema:   "",
			Table:    "",
			Column:   "",
		}

		rangeVar, ok := jsonData["RangeVar"].(map[string]any)
		if !ok {
			return nil, errors.Errorf("failed to convert range var")
		}
		if rangeVar["schemaname"] != nil {
			schema, ok := rangeVar["schemaname"].(string)
			if !ok {
				return nil, errors.Errorf("failed to convert schemaname")
			}
			resource.Schema = schema
		}
		if rangeVar["relname"] != nil {
			table, ok := rangeVar["relname"].(string)
			if !ok {
				return nil, errors.Errorf("failed to convert relname")
			}
			resource.Table = table
		}

		// This is a false-positive behavior, the table we found may not be the table the query actually accesses.
		// For example, the query is `WITH t1 AS (SELECT 1) SELECT * FROM t1` and we have a physical table `t1` in the database exactly,
		// what we found is the physical table `t1`, but the query actually accesses the CTE `t1`.
		// We do this because we do not have too much time to implement the real behavior.
		// XXX(rebelice/zp): Can we pass more information here to make this function know the context and then
		// figure out whether the table is the table the query actually accesses?

		// Bytebase do not sync the system objects, so we skip finding for system objects in the metadata.
		if msg := isSystemResource(resource); msg == "" {
			// Backfill the default database/schema name.
			if resource.Database == "" {
				resource.Database = currentDatabase
			}
			if resource.Schema == "" {
				resource.Schema = currentSchema
			}
			databaseMetadata, err := q.getDatabaseMetadata(currentDatabase)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", currentDatabase)
			}
			// Access pseudo table or table/view we do not sync, return directly.
			if databaseMetadata == nil {
				return nil, nil
			}
			schema := databaseMetadata.GetSchema(resource.Schema)
			if schema == nil {
				return nil, nil
			}
			if schema.GetTable(resource.Table) == nil && schema.GetView(resource.Table) == nil && schema.GetMaterializedView(resource.Table) == nil && schema.GetExternalTable(resource.Table) == nil {
				return nil, nil
			}
		}
		result = append(result, resource)
	}

	for _, value := range jsonData {
		switch v := value.(type) {
		case map[string]any:
			resources, err := q.getRangeVarsFromJSONRecursive(v, currentDatabase, currentSchema)
			if err != nil {
				return nil, err
			}
			result = append(result, resources...)
		case []any:
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					resources, err := q.getRangeVarsFromJSONRecursive(m, currentDatabase, currentSchema)
					if err != nil {
						return nil, err
					}
					result = append(result, resources...)
				}
			}
		}
	}

	return result, nil
}

// isMixedQuery checks whether the query accesses the user table and system table at the same time.
func isMixedQuery(m base.SourceColumnSet) (allSystems bool, mixed error) {
	userMsg, systemMsg := "", ""
	for table := range m {
		if msg := isSystemResource(table); msg != "" {
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

func isSystemResource(resource base.ColumnResource) string {
	// User can access the system table/view by name directly without database/schema name.
	// For example: `SELECT * FROM pg_database`, which will access the system table `pg_database`.
	// Additionally, user can create a table/view with the same name with system table/view and access them
	// by specify the schema name, for example:
	// `CREATE TABLE pg_database(id INT); SELECT * FROM public.pg_database;` which will access the user table `pg_database`.
	if IsSystemSchema(resource.Schema) {
		return fmt.Sprintf("system schema %q", resource.Schema)
	}
	if resource.Database == "" && resource.Schema == "" && IsSystemView(resource.Table) {
		return fmt.Sprintf("system view %q", resource.Table)
	}
	if resource.Database == "" && resource.Schema == "" && IsSystemTable(resource.Table) {
		return fmt.Sprintf("system table %q", resource.Table)
	}
	return ""
}

func (q *querySpanExtractor) getColumnsForView(definition string) ([]base.QuerySpanResult, error) {
	newQ := newQuerySpanExtractor(q.connectedDB, q.f)
	span, err := newQ.getQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, err
	}
	return span.Results, nil
}

func (q *querySpanExtractor) getColumnsForMaterializedView(definition string) ([]base.QuerySpanResult, error) {
	newQ := newQuerySpanExtractor(q.connectedDB, q.f)
	span, err := newQ.getQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, err
	}
	return span.Results, nil
}

func newTypeNotSupportedErrorByNode(node *pgquery.Node) *parsererror.TypeNotSupportedError {
	switch node := node.Node.(type) {
	case *pgquery.Node_RangeFunction:
		schemaName, funcName := extractFunctionNameInRangeFunction(node.RangeFunction)
		if schemaName == "" && funcName == "" {
			return &parsererror.TypeNotSupportedError{
				Type: "function",
				Err:  errors.Errorf("node: %+v", node),
			}
		}
		return &parsererror.TypeNotSupportedError{
			Type: "function",
			Name: fmt.Sprintf("%s.%s", schemaName, funcName),
		}
	default:
		return &parsererror.TypeNotSupportedError{
			Err: errors.Errorf("node: %+v", node),
		}
	}
}

func extractFunctionNameInRangeFunction(node *pgquery.RangeFunction) (string, string) {
	// Capture the function name from the range function.
	for _, f := range node.GetFunctions() {
		if listNode, ok := f.Node.(*pgquery.Node_List); ok {
			for _, item := range listNode.List.GetItems() {
				if funcCall, ok := item.Node.(*pgquery.Node_FuncCall); ok {
					var names []string
					for _, name := range funcCall.FuncCall.GetFuncname() {
						if stringNode, ok := name.Node.(*pgquery.Node_String_); ok {
							names = append(names, stringNode.String_.GetSval())
						}
					}

					switch len(names) {
					case 2:
						return names[0], names[1]
					case 1:
						return "public", names[0]
					case 0:
						return "", ""
					default:
						slog.Debug("Unknow function name", "name", strings.Join(names, "."))
						return names[0], names[1]
					}
				}
			}
		}
	}

	return "", ""
}
