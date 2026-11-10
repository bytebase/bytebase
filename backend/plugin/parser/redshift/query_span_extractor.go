package redshift

import (
	"context"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	redshiftanalysis "github.com/bytebase/omni/redshift/analysis"
	redshiftast "github.com/bytebase/omni/redshift/ast"
	redshiftcatalog "github.com/bytebase/omni/redshift/catalog"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

const redshiftGetDatabaseMetadataError = "failed to get database metadata for database: %s"

type querySpanExtractor struct {
	defaultDatabase string
	searchPath      []string
	gCtx            base.GetQuerySpanContext
}

func newQuerySpanExtractor(defaultDatabase string, searchPath []string, gCtx base.GetQuerySpanContext) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase: defaultDatabase,
		searchPath:      normalizeSearchPath(searchPath),
		gCtx:            gCtx,
	}
}

func normalizeSearchPath(searchPath []string) []string {
	result := make([]string, 0, len(searchPath))
	for _, schema := range searchPath {
		if schema != "" {
			result = append(result, schema)
		}
	}
	if len(result) == 0 {
		return []string{"public"}
	}
	return result
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	return q.getQuerySpanWithFunctionStack(ctx, statement, map[string]bool{})
}

func (q *querySpanExtractor) getQuerySpanWithFunctionStack(ctx context.Context, statement string, functionStack map[string]bool) (*base.QuerySpan, error) {
	stmts, err := ParseRedshift(statement)
	if err != nil {
		return nil, err
	}
	if len(stmts) == 0 || stmts[0].Empty() {
		return emptyQuerySpan(base.Select), nil
	}
	if len(stmts) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", len(stmts))
	}

	node := stmts[0].AST
	queryType := querySpanType(node, false /* allSystems */)
	if explain, ok := node.(*redshiftast.ExplainStmt); ok && hasExplainAnalyze(explain) {
		return q.getExplainAnalyzeQuerySpan(ctx, explain.Query, queryType)
	}
	if unload, ok := node.(*redshiftast.UnloadStmt); ok {
		return q.getUnloadQuerySpan(ctx, unload, functionStack)
	}
	if _, ok := node.(*redshiftast.VariableSetStmt); ok {
		return emptyQuerySpan(queryType), nil
	}
	if queryType != base.Select {
		return emptyQuerySpan(queryType), nil
	}

	accessTables, earlySpan, err := q.collectSelectAccessTables(ctx, node)
	if err != nil {
		return nil, err
	}
	if earlySpan != nil {
		return earlySpan, nil
	}

	cat, err := q.newQuerySpanCatalog(ctx)
	if err != nil {
		return nil, err
	}
	span, err := redshiftanalysis.GetQuerySpan(cat, statement)
	if err != nil {
		if isQuerySpanNotFound(err) {
			return q.notFoundQuerySpan(err), nil
		}
		return nil, err
	}

	results := q.convertResults(span.Results)
	functionAccessTables, functionResultSources, functionIssues, err := q.collectFunctionBodySources(ctx, node, functionStack, make(map[string]functionBodySources))
	if err != nil {
		return nil, err
	}
	for source := range functionAccessTables {
		accessTables[source] = true
	}
	if functionIssues.functionNotSupportedError != nil && len(accessTables) == 0 {
		accessTables[base.ColumnResource{Database: q.defaultDatabase}] = true
	}
	mergeFunctionResultSources(results, functionResultSources)
	if hasRangeVar(node) && hasResultWithoutSource(results) && !q.allAccessTablesExist(ctx, accessTables) {
		return q.notFoundQuerySpan(&base.ResourceNotFoundError{Database: &q.defaultDatabase}), nil
	}

	return &base.QuerySpan{
		Type:                      base.Select,
		SourceColumns:             accessTables,
		PredicateColumns:          q.convertColumnList(span.PredicateColumns),
		Results:                   results,
		NotFoundError:             functionIssues.notFoundError,
		FunctionNotSupportedError: functionIssues.functionNotSupportedError,
	}, nil
}

func (q *querySpanExtractor) getExplainAnalyzeQuerySpan(ctx context.Context, query redshiftast.Node, queryType base.QueryType) (*base.QuerySpan, error) {
	if queryType != base.Select {
		return emptyQuerySpan(queryType), nil
	}

	accessTables, earlySpan, err := q.collectSelectAccessTables(ctx, query)
	if err != nil {
		return nil, err
	}
	if earlySpan != nil {
		return earlySpan, nil
	}
	return &base.QuerySpan{
		Type:             base.Select,
		SourceColumns:    accessTables,
		PredicateColumns: base.SourceColumnSet{},
		Results:          []base.QuerySpanResult{},
	}, nil
}

func (q *querySpanExtractor) getUnloadQuerySpan(ctx context.Context, unload *redshiftast.UnloadStmt, functionStack map[string]bool) (*base.QuerySpan, error) {
	span, err := q.getQuerySpanWithFunctionStack(ctx, unload.Query, functionStack)
	if err != nil {
		return nil, err
	}
	return &base.QuerySpan{
		Type:                      base.DML,
		SourceColumns:             span.SourceColumns,
		PredicateColumns:          span.PredicateColumns,
		Results:                   []base.QuerySpanResult{},
		NotFoundError:             span.NotFoundError,
		FunctionNotSupportedError: span.FunctionNotSupportedError,
	}, nil
}

func (q *querySpanExtractor) collectSelectAccessTables(ctx context.Context, node redshiftast.Node) (base.SourceColumnSet, *base.QuerySpan, error) {
	accessTables, err := q.collectAccessTables(ctx, node)
	if err != nil {
		var resourceNotFound *base.ResourceNotFoundError
		if errors.As(err, &resourceNotFound) {
			return nil, q.notFoundQuerySpan(resourceNotFound), nil
		}
		return nil, nil, err
	}
	allSystems, mixed := isMixedQuery(accessTables)
	if mixed {
		return nil, nil, base.MixUserSystemTablesError
	}
	if allSystems {
		return nil, emptyQuerySpan(base.SelectInfoSchema), nil
	}
	return accessTables, nil, nil
}

func (q *querySpanExtractor) notFoundQuerySpan(err error) *base.QuerySpan {
	return &base.QuerySpan{
		Type: base.Select,
		SourceColumns: base.SourceColumnSet{
			{Database: q.defaultDatabase}: true,
		},
		PredicateColumns: base.SourceColumnSet{},
		Results:          []base.QuerySpanResult{},
		NotFoundError:    err,
	}
}

func emptyQuerySpan(queryType base.QueryType) *base.QuerySpan {
	return &base.QuerySpan{
		Type:          queryType,
		SourceColumns: base.SourceColumnSet{},
		Results:       []base.QuerySpanResult{},
	}
}

func (q *querySpanExtractor) newQuerySpanCatalog(ctx context.Context) (*redshiftcatalog.Catalog, error) {
	if q.gCtx.GetDatabaseMetadataFunc == nil {
		return nil, errors.New("GetDatabaseMetadataFunc is not set in GetQuerySpanContext")
	}
	_, metadata, err := q.gCtx.GetDatabaseMetadataFunc(ctx, q.gCtx.InstanceID, q.defaultDatabase)
	if err != nil {
		return nil, errors.Wrapf(err, redshiftGetDatabaseMetadataError, q.defaultDatabase)
	}
	if metadata == nil {
		return nil, &base.ResourceNotFoundError{Database: &q.defaultDatabase}
	}
	cat := redshiftcatalog.New()
	cat.SetSearchPath(q.searchPath)
	cat.UseMetadata(metadata.GetProto())
	return cat, nil
}

type functionBodySources struct {
	accessTables base.SourceColumnSet
	results      []base.QuerySpanResult
	issues       functionBodyIssues
}

type functionBodyIssues struct {
	notFoundError             error
	functionNotSupportedError error
}

func (i *functionBodyIssues) merge(other functionBodyIssues) {
	if i.notFoundError == nil {
		i.notFoundError = other.notFoundError
	}
	if i.functionNotSupportedError == nil {
		i.functionNotSupportedError = other.functionNotSupportedError
	}
}

type functionSourceProjection struct {
	names   []string
	sources []base.SourceColumnSet
}

type functionSourceScope map[string]functionSourceProjection

func (q *querySpanExtractor) collectAccessTables(ctx context.Context, node redshiftast.Node) (base.SourceColumnSet, error) {
	result := make(base.SourceColumnSet)
	if err := q.collectAccessTablesFromNode(ctx, node, map[string]bool{}, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (q *querySpanExtractor) collectAccessTablesFromNode(ctx context.Context, node redshiftast.Node, ctes map[string]bool, result base.SourceColumnSet) error {
	if node == nil {
		return nil
	}
	if selectStmt, ok := node.(*redshiftast.SelectStmt); ok {
		return q.collectAccessTablesFromSelect(ctx, selectStmt, ctes, result)
	}

	var err error
	redshiftast.Inspect(node, func(n redshiftast.Node) bool {
		if err != nil {
			return false
		}
		if n != node {
			if selectStmt, ok := n.(*redshiftast.SelectStmt); ok {
				err = q.collectAccessTablesFromSelect(ctx, selectStmt, ctes, result)
				return false
			}
		}
		rangeVar, ok := n.(*redshiftast.RangeVar)
		if !ok || rangeVar == nil || rangeVar.Relname == "" {
			return true
		}
		if isCTEReference(rangeVar, ctes) {
			return true
		}

		resource, resolveErr := q.resolveRangeVar(ctx, rangeVar)
		if resolveErr != nil {
			err = resolveErr
			return false
		}
		if resource.Table != "" {
			result[resource] = true
		}
		return true
	})
	if err != nil {
		return err
	}
	return nil
}

func (q *querySpanExtractor) collectAccessTablesFromSelect(ctx context.Context, selectStmt *redshiftast.SelectStmt, ctes map[string]bool, result base.SourceColumnSet) error {
	localCTEs := cloneCTENameSet(ctes)
	if selectStmt.WithClause != nil && selectStmt.WithClause.Ctes != nil {
		cteScope := cloneCTENameSet(ctes)
		if selectStmt.WithClause.Recursive {
			for _, item := range selectStmt.WithClause.Ctes.Items {
				cte, ok := item.(*redshiftast.CommonTableExpr)
				if ok && cte.Ctename != "" {
					cteScope[strings.ToLower(cte.Ctename)] = true
				}
			}
		}
		for _, item := range selectStmt.WithClause.Ctes.Items {
			cte, ok := item.(*redshiftast.CommonTableExpr)
			if !ok || cte.Ctename == "" {
				continue
			}
			if err := q.collectAccessTablesFromNode(ctx, cte.Ctequery, cteScope, result); err != nil {
				return err
			}
			name := strings.ToLower(cte.Ctename)
			cteScope[name] = true
			localCTEs[name] = true
		}
	}

	for _, node := range []redshiftast.Node{
		selectStmt.WhereClause,
		selectStmt.HavingClause,
		selectStmt.QualifyClause,
		selectStmt.LimitOffset,
		selectStmt.LimitCount,
	} {
		if err := q.collectAccessTablesFromNode(ctx, node, localCTEs, result); err != nil {
			return err
		}
	}
	if selectStmt.IntoClause != nil {
		if err := q.collectAccessTablesFromNode(ctx, selectStmt.IntoClause, localCTEs, result); err != nil {
			return err
		}
	}
	if selectStmt.Larg != nil {
		if err := q.collectAccessTablesFromNode(ctx, selectStmt.Larg, localCTEs, result); err != nil {
			return err
		}
	}
	if selectStmt.Rarg != nil {
		if err := q.collectAccessTablesFromNode(ctx, selectStmt.Rarg, localCTEs, result); err != nil {
			return err
		}
	}
	for _, list := range []*redshiftast.List{
		selectStmt.DistinctClause,
		selectStmt.TargetList,
		selectStmt.FromClause,
		selectStmt.GroupClause,
		selectStmt.WindowClause,
		selectStmt.ExcludeList,
		selectStmt.ValuesLists,
		selectStmt.SortClause,
		selectStmt.LockingClause,
	} {
		if err := q.collectAccessTablesFromList(ctx, list, localCTEs, result); err != nil {
			return err
		}
	}
	return nil
}

func (q *querySpanExtractor) collectAccessTablesFromList(ctx context.Context, list *redshiftast.List, ctes map[string]bool, result base.SourceColumnSet) error {
	if list == nil {
		return nil
	}
	for _, item := range list.Items {
		if err := q.collectAccessTablesFromNode(ctx, item, ctes, result); err != nil {
			return err
		}
	}
	return nil
}

func (q *querySpanExtractor) collectFunctionBodySources(ctx context.Context, node redshiftast.Node, functionStack map[string]bool, cache map[string]functionBodySources) (base.SourceColumnSet, []base.SourceColumnSet, functionBodyIssues, error) {
	accessTables, issues, err := q.collectFunctionBodyAccessTables(ctx, node, functionStack, cache)
	if err != nil {
		return nil, nil, functionBodyIssues{}, err
	}

	var resultSources []base.SourceColumnSet
	if selectStmt, ok := node.(*redshiftast.SelectStmt); ok {
		var sourceIssues functionBodyIssues
		resultSources, sourceIssues, err = q.collectSelectFunctionResultSources(ctx, selectStmt, functionStack, cache, nil)
		if err != nil {
			return nil, nil, functionBodyIssues{}, err
		}
		issues.merge(sourceIssues)
	}
	return accessTables, resultSources, issues, nil
}

func (q *querySpanExtractor) collectFunctionBodyAccessTables(ctx context.Context, node redshiftast.Node, functionStack map[string]bool, cache map[string]functionBodySources) (base.SourceColumnSet, functionBodyIssues, error) {
	result := make(base.SourceColumnSet)
	if node == nil {
		return result, functionBodyIssues{}, nil
	}

	var err error
	issues := functionBodyIssues{}
	redshiftast.Inspect(node, func(n redshiftast.Node) bool {
		if err != nil {
			return false
		}
		call, ok := n.(*redshiftast.FuncCall)
		if !ok {
			return true
		}
		sources, sourceErr := q.functionBodySourcesForCall(ctx, call, functionStack, cache)
		if sourceErr != nil {
			err = sourceErr
			return false
		}
		issues.merge(sources.issues)
		for source := range sources.accessTables {
			result[source] = true
		}
		return true
	})
	return result, issues, err
}

func (q *querySpanExtractor) collectSelectFunctionResultSources(ctx context.Context, selectStmt *redshiftast.SelectStmt, functionStack map[string]bool, cache map[string]functionBodySources, outerScope functionSourceScope) ([]base.SourceColumnSet, functionBodyIssues, error) {
	if selectStmt == nil || selectStmt.TargetList == nil {
		return nil, functionBodyIssues{}, nil
	}

	scope, issues, err := q.functionSourceScopeForSelect(ctx, selectStmt, functionStack, cache, outerScope)
	if err != nil {
		return nil, functionBodyIssues{}, err
	}

	result := make([]base.SourceColumnSet, 0, selectStmt.TargetList.Len())
	for _, item := range selectStmt.TargetList.Items {
		target, ok := item.(*redshiftast.ResTarget)
		if !ok {
			result = append(result, nil)
			continue
		}
		sources, sourceIssues, err := q.collectFunctionBodyResultSources(ctx, target.Val, functionStack, cache, scope)
		if err != nil {
			return nil, functionBodyIssues{}, err
		}
		issues.merge(sourceIssues)
		result = append(result, sources)
	}
	return result, issues, nil
}

func (q *querySpanExtractor) functionSourceScopeForSelect(ctx context.Context, selectStmt *redshiftast.SelectStmt, functionStack map[string]bool, cache map[string]functionBodySources, outerScope functionSourceScope) (functionSourceScope, functionBodyIssues, error) {
	scope := cloneFunctionSourceScope(outerScope)
	issues := functionBodyIssues{}

	if selectStmt.WithClause != nil && selectStmt.WithClause.Ctes != nil {
		cteScope := cloneFunctionSourceScope(scope)
		for _, item := range selectStmt.WithClause.Ctes.Items {
			cte, ok := item.(*redshiftast.CommonTableExpr)
			if !ok || cte.Ctename == "" {
				continue
			}
			cteSelect, ok := cte.Ctequery.(*redshiftast.SelectStmt)
			if !ok {
				continue
			}
			sources, sourceIssues, err := q.collectSelectFunctionResultSources(ctx, cteSelect, functionStack, cache, cteScope)
			if err != nil {
				return nil, functionBodyIssues{}, err
			}
			issues.merge(sourceIssues)
			projection := functionSourceProjection{
				names:   redshiftSelectOutputNames(cteSelect),
				sources: sources,
			}
			if aliasNames := redshiftStringList(cte.Aliascolnames); len(aliasNames) > 0 {
				projection.names = aliasNames
			}
			name := strings.ToLower(cte.Ctename)
			scope[name] = projection
			cteScope[name] = projection
		}
	}

	sourceIssues, err := q.collectFunctionSourceScopeFromList(ctx, selectStmt.FromClause, functionStack, cache, scope)
	if err != nil {
		return nil, functionBodyIssues{}, err
	}
	issues.merge(sourceIssues)
	return scope, issues, nil
}

func (q *querySpanExtractor) collectFunctionSourceScopeFromList(ctx context.Context, list *redshiftast.List, functionStack map[string]bool, cache map[string]functionBodySources, scope functionSourceScope) (functionBodyIssues, error) {
	if list == nil {
		return functionBodyIssues{}, nil
	}
	issues := functionBodyIssues{}
	for _, item := range list.Items {
		sourceIssues, err := q.collectFunctionSourceScopeFromNode(ctx, item, functionStack, cache, scope)
		if err != nil {
			return functionBodyIssues{}, err
		}
		issues.merge(sourceIssues)
	}
	return issues, nil
}

func (q *querySpanExtractor) collectFunctionSourceScopeFromNode(ctx context.Context, node redshiftast.Node, functionStack map[string]bool, cache map[string]functionBodySources, scope functionSourceScope) (functionBodyIssues, error) {
	switch n := node.(type) {
	case *redshiftast.RangeVar:
		if n == nil || n.Alias == nil || n.Alias.Aliasname == "" {
			return functionBodyIssues{}, nil
		}
		projection, ok := scope[strings.ToLower(n.Relname)]
		if !ok {
			return functionBodyIssues{}, nil
		}
		if aliasNames := redshiftStringList(n.Alias.Colnames); len(aliasNames) > 0 {
			projection.names = aliasNames
		}
		scope[strings.ToLower(n.Alias.Aliasname)] = projection
		return functionBodyIssues{}, nil
	case *redshiftast.RangeSubselect:
		if n == nil || n.Alias == nil || n.Alias.Aliasname == "" {
			return functionBodyIssues{}, nil
		}
		selectStmt, ok := n.Subquery.(*redshiftast.SelectStmt)
		if !ok {
			return functionBodyIssues{}, nil
		}
		sources, issues, err := q.collectSelectFunctionResultSources(ctx, selectStmt, functionStack, cache, scope)
		if err != nil {
			return functionBodyIssues{}, err
		}
		projection := functionSourceProjection{
			names:   redshiftSelectOutputNames(selectStmt),
			sources: sources,
		}
		if aliasNames := redshiftStringList(n.Alias.Colnames); len(aliasNames) > 0 {
			projection.names = aliasNames
		}
		scope[strings.ToLower(n.Alias.Aliasname)] = projection
		return issues, nil
	case *redshiftast.JoinExpr:
		if n == nil {
			return functionBodyIssues{}, nil
		}
		leftIssues, err := q.collectFunctionSourceScopeFromNode(ctx, n.Larg, functionStack, cache, scope)
		if err != nil {
			return functionBodyIssues{}, err
		}
		rightIssues, err := q.collectFunctionSourceScopeFromNode(ctx, n.Rarg, functionStack, cache, scope)
		if err != nil {
			return functionBodyIssues{}, err
		}
		leftIssues.merge(rightIssues)
		return leftIssues, nil
	default:
		return functionBodyIssues{}, nil
	}
}

func (q *querySpanExtractor) collectFunctionBodyResultSources(ctx context.Context, node redshiftast.Node, functionStack map[string]bool, cache map[string]functionBodySources, scope functionSourceScope) (base.SourceColumnSet, functionBodyIssues, error) {
	result := make(base.SourceColumnSet)
	if node == nil {
		return result, functionBodyIssues{}, nil
	}

	var err error
	issues := functionBodyIssues{}
	redshiftast.Inspect(node, func(n redshiftast.Node) bool {
		if err != nil {
			return false
		}
		if columnRef, ok := n.(*redshiftast.ColumnRef); ok {
			mergeSourceColumnSet(result, resolveFunctionColumnRef(scope, columnRef))
			return true
		}
		call, ok := n.(*redshiftast.FuncCall)
		if !ok {
			return true
		}
		sources, sourceErr := q.functionBodySourcesForCall(ctx, call, functionStack, cache)
		if sourceErr != nil {
			err = sourceErr
			return false
		}
		issues.merge(sources.issues)
		for _, resultColumn := range sources.results {
			for source := range resultColumn.SourceColumns {
				result[source] = true
			}
		}
		return true
	})
	return result, issues, err
}

func (q *querySpanExtractor) functionBodySourcesForCall(ctx context.Context, call *redshiftast.FuncCall, functionStack map[string]bool, cache map[string]functionBodySources) (functionBodySources, error) {
	schemaName, functionName := functionCallName(call.Funcname)
	if functionName == "" || IsSystemFunction(functionName, "") {
		return functionBodySources{accessTables: base.SourceColumnSet{}}, nil
	}

	functionSchema, function, findErr := q.findFunctionMetadata(ctx, schemaName, functionName, call.Args.Len())
	if findErr != nil || function == nil {
		return functionBodySources{accessTables: base.SourceColumnSet{}}, findErr
	}

	cacheKey := strings.ToLower(functionSchema + "." + function.GetName() + "/" + function.GetSignature())
	if sources, ok := cache[cacheKey]; ok {
		return sources, nil
	}
	if functionStack[cacheKey] {
		return functionBodySources{accessTables: base.SourceColumnSet{}}, nil
	}

	body, ok, bodyErr := extractSQLFunctionBody(function.GetDefinition())
	if bodyErr != nil {
		// Body extraction failures should fail closed through QuerySpan.FunctionNotSupportedError.
		//nolint:nilerr
		return functionBodySources{
			accessTables: base.SourceColumnSet{},
			issues: functionBodyIssues{
				functionNotSupportedError: &base.FunctionNotSupportedError{
					Err:        bodyErr,
					Function:   functionSchema + "." + function.GetName(),
					Definition: function.GetDefinition(),
				},
			},
		}, nil
	}
	if !ok {
		return functionBodySources{
			accessTables: base.SourceColumnSet{},
			issues: functionBodyIssues{
				functionNotSupportedError: &base.FunctionNotSupportedError{
					Err:        errors.New("unsupported or empty SQL function body"),
					Function:   functionSchema + "." + function.GetName(),
					Definition: function.GetDefinition(),
				},
			},
		}, nil
	}

	functionStack[cacheKey] = true
	span, querySpanErr := q.getQuerySpanWithFunctionStack(ctx, body, functionStack)
	delete(functionStack, cacheKey)
	if querySpanErr != nil {
		return functionBodySources{}, errors.Wrapf(querySpanErr, "failed to get query span for function: %s.%s", functionSchema, functionName)
	}
	sources := functionBodySources{
		accessTables: span.SourceColumns,
		results:      span.Results,
		issues: functionBodyIssues{
			notFoundError:             span.NotFoundError,
			functionNotSupportedError: span.FunctionNotSupportedError,
		},
	}
	if sources.issues.notFoundError == nil && sources.issues.functionNotSupportedError == nil {
		cache[cacheKey] = sources
	}
	return sources, nil
}

func (q *querySpanExtractor) findFunctionMetadata(ctx context.Context, schemaName, functionName string, argCount int) (string, *metadatapb.FunctionMetadata, error) {
	if q.gCtx.GetDatabaseMetadataFunc == nil {
		return "", nil, nil
	}
	_, metadata, err := q.gCtx.GetDatabaseMetadataFunc(ctx, q.gCtx.InstanceID, q.defaultDatabase)
	if err != nil {
		return "", nil, errors.Wrapf(err, redshiftGetDatabaseMetadataError, q.defaultDatabase)
	}
	if metadata == nil {
		return "", nil, nil
	}

	searchPath := q.searchPath
	if schemaName != "" {
		searchPath = []string{schemaName}
	}
	schemas, functions := metadata.SearchFunctions(searchPath, functionName)
	if len(functions) == 0 {
		return "", nil, nil
	}
	if len(functions) == 1 {
		return schemas[0], functions[0], nil
	}
	for i, function := range functions {
		count, ok := functionParameterCount(function.GetDefinition())
		if ok && count == argCount {
			return schemas[i], function, nil
		}
	}
	return "", nil, nil
}

func functionCallName(list *redshiftast.List) (string, string) {
	if list == nil {
		return "", ""
	}
	parts := make([]string, 0, list.Len())
	for _, item := range list.Items {
		if value, ok := item.(*redshiftast.String); ok {
			parts = append(parts, value.Str)
		}
	}
	switch len(parts) {
	case 1:
		return "", parts[0]
	case 2:
		return parts[0], parts[1]
	case 3:
		return parts[1], parts[2]
	default:
		return "", ""
	}
}

func functionParameterCount(definition string) (int, bool) {
	stmts, err := ParseRedshift(definition)
	if err != nil || len(stmts) != 1 {
		return 0, false
	}
	stmt, ok := stmts[0].AST.(*redshiftast.CreateFunctionStmt)
	if !ok || stmt.Parameters == nil {
		return 0, ok
	}
	return stmt.Parameters.Len(), true
}

func extractSQLFunctionBody(definition string) (string, bool, error) {
	stmts, err := ParseRedshift(definition)
	if err != nil {
		return "", false, errors.Wrapf(err, "failed to parse function definition")
	}
	if len(stmts) != 1 {
		return "", false, nil
	}
	stmt, ok := stmts[0].AST.(*redshiftast.CreateFunctionStmt)
	if !ok {
		return "", false, nil
	}

	language := ""
	body := ""
	if stmt.Options != nil {
		for _, item := range stmt.Options.Items {
			elem, ok := item.(*redshiftast.DefElem)
			if !ok {
				continue
			}
			switch strings.ToLower(elem.Defname) {
			case "language":
				language = defElemString(elem.Arg)
			case "as":
				body = defElemString(elem.Arg)
			default:
			}
		}
	}
	if language != "" && !strings.EqualFold(language, "sql") {
		return "", false, nil
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return "", false, nil
	}
	return body, true, nil
}

func defElemString(node redshiftast.Node) string {
	switch value := node.(type) {
	case *redshiftast.String:
		return value.Str
	case *redshiftast.List:
		if len(value.Items) == 0 {
			return ""
		}
		return defElemString(value.Items[0])
	default:
		return ""
	}
}

func mergeFunctionResultSources(results []base.QuerySpanResult, functionResultSources []base.SourceColumnSet) {
	if len(functionResultSources) == len(results) {
		for i, sources := range functionResultSources {
			for source := range sources {
				results[i].SourceColumns[source] = true
			}
		}
		return
	}
	for _, sources := range functionResultSources {
		if len(sources) == 0 {
			continue
		}
		for i := range results {
			if len(results[i].SourceColumns) != 0 {
				continue
			}
			for source := range sources {
				results[i].SourceColumns[source] = true
			}
		}
	}
}

func cloneFunctionSourceScope(scope functionSourceScope) functionSourceScope {
	result := make(functionSourceScope, len(scope))
	for name, projection := range scope {
		result[name] = projection
	}
	return result
}

func mergeSourceColumnSet(dst, src base.SourceColumnSet) {
	for source := range src {
		dst[source] = true
	}
}

func resolveFunctionColumnRef(scope functionSourceScope, columnRef *redshiftast.ColumnRef) base.SourceColumnSet {
	parts := redshiftColumnRefParts(columnRef)
	switch len(parts) {
	case 1:
		return resolveUnqualifiedFunctionColumn(scope, parts[0])
	case 2:
		if projection, ok := scope[strings.ToLower(parts[0])]; ok {
			return projection.functionSourcesForColumn(parts[1])
		}
		return base.SourceColumnSet{}
	default:
		return base.SourceColumnSet{}
	}
}

func resolveUnqualifiedFunctionColumn(scope functionSourceScope, columnName string) base.SourceColumnSet {
	var result base.SourceColumnSet
	for _, projection := range scope {
		sources := projection.functionSourcesForColumn(columnName)
		if len(sources) == 0 {
			continue
		}
		if result != nil {
			return base.SourceColumnSet{}
		}
		result = sources
	}
	if result == nil {
		return base.SourceColumnSet{}
	}
	return result
}

func (p functionSourceProjection) functionSourcesForColumn(columnName string) base.SourceColumnSet {
	for i, name := range p.names {
		if !strings.EqualFold(name, columnName) || i >= len(p.sources) {
			continue
		}
		return p.sources[i]
	}
	return base.SourceColumnSet{}
}

func redshiftSelectOutputNames(selectStmt *redshiftast.SelectStmt) []string {
	if selectStmt == nil || selectStmt.TargetList == nil {
		return nil
	}
	result := make([]string, 0, selectStmt.TargetList.Len())
	for _, item := range selectStmt.TargetList.Items {
		target, ok := item.(*redshiftast.ResTarget)
		if !ok {
			result = append(result, "")
			continue
		}
		result = append(result, redshiftTargetOutputName(target))
	}
	return result
}

func redshiftTargetOutputName(target *redshiftast.ResTarget) string {
	if target == nil {
		return ""
	}
	if target.Name != "" {
		return target.Name
	}
	if parts := redshiftColumnRefParts(asColumnRef(target.Val)); len(parts) > 0 {
		return parts[len(parts)-1]
	}
	if call, ok := target.Val.(*redshiftast.FuncCall); ok {
		_, functionName := functionCallName(call.Funcname)
		return functionName
	}
	return ""
}

func asColumnRef(node redshiftast.Node) *redshiftast.ColumnRef {
	columnRef, ok := node.(*redshiftast.ColumnRef)
	if !ok {
		return nil
	}
	return columnRef
}

func redshiftColumnRefParts(columnRef *redshiftast.ColumnRef) []string {
	if columnRef == nil || columnRef.Fields == nil {
		return nil
	}
	result := make([]string, 0, columnRef.Fields.Len())
	for _, item := range columnRef.Fields.Items {
		value, ok := item.(*redshiftast.String)
		if !ok {
			return nil
		}
		result = append(result, value.Str)
	}
	return result
}

func redshiftStringList(list *redshiftast.List) []string {
	if list == nil {
		return nil
	}
	result := make([]string, 0, list.Len())
	for _, item := range list.Items {
		value, ok := item.(*redshiftast.String)
		if !ok {
			continue
		}
		result = append(result, value.Str)
	}
	return result
}

func cloneCTENameSet(ctes map[string]bool) map[string]bool {
	result := make(map[string]bool, len(ctes))
	for name, ok := range ctes {
		result[name] = ok
	}
	return result
}

func isCTEReference(rangeVar *redshiftast.RangeVar, ctes map[string]bool) bool {
	if rangeVar == nil || rangeVar.Schemaname != "" || rangeVar.Catalogname != "" {
		return false
	}
	return ctes[strings.ToLower(rangeVar.Relname)]
}

func hasResultWithoutSource(results []base.QuerySpanResult) bool {
	for _, result := range results {
		if len(result.SourceColumns) == 0 {
			return true
		}
	}
	return false
}

func hasRangeVar(node redshiftast.Node) bool {
	found := false
	redshiftast.Inspect(node, func(n redshiftast.Node) bool {
		if _, ok := n.(*redshiftast.RangeVar); ok {
			found = true
			return false
		}
		return true
	})
	return found
}

func (q *querySpanExtractor) allAccessTablesExist(ctx context.Context, accessTables base.SourceColumnSet) bool {
	if len(accessTables) == 0 || q.gCtx.GetDatabaseMetadataFunc == nil {
		return false
	}
	for resource := range accessTables {
		if !q.accessTableExists(ctx, resource) {
			return false
		}
	}
	return true
}

func (q *querySpanExtractor) accessTableExists(ctx context.Context, resource base.ColumnResource) bool {
	if isSystemResource(resource) {
		return true
	}
	database := resource.Database
	if database == "" {
		database = q.defaultDatabase
	}
	_, metadata, err := q.gCtx.GetDatabaseMetadataFunc(ctx, q.gCtx.InstanceID, database)
	if err != nil || metadata == nil {
		return false
	}
	if resource.Schema == "" {
		schemaName, objectName := metadata.SearchRelation(q.searchPath, resource.Table)
		return schemaName != "" || objectName != ""
	}
	schema := metadata.GetSchemaMetadata(resource.Schema)
	if schema == nil {
		return false
	}
	// Match the relation kinds used by SearchRelation above.
	return schema.GetTable(resource.Table) != nil ||
		schema.GetView(resource.Table) != nil ||
		schema.GetMaterializedView(resource.Table) != nil ||
		schema.GetExternalTable(resource.Table) != nil ||
		schema.GetSequence(resource.Table) != nil
}

func (q *querySpanExtractor) resolveRangeVar(ctx context.Context, rangeVar *redshiftast.RangeVar) (base.ColumnResource, error) {
	resource := base.ColumnResource{
		Database: q.defaultDatabase,
		Schema:   rangeVar.Schemaname,
		Table:    rangeVar.Relname,
	}
	if rangeVar.Catalogname != "" {
		resource.Database = rangeVar.Catalogname
	}
	if resource.Schema != "" || isSystemResource(resource) || q.gCtx.GetDatabaseMetadataFunc == nil {
		return resource, nil
	}

	_, metadata, err := q.gCtx.GetDatabaseMetadataFunc(ctx, q.gCtx.InstanceID, resource.Database)
	if err != nil {
		return base.ColumnResource{}, errors.Wrapf(err, redshiftGetDatabaseMetadataError, resource.Database)
	}
	if metadata == nil {
		return resource, nil
	}
	schemaName, objectName := metadata.SearchRelation(q.searchPath, resource.Table)
	if schemaName == "" && objectName == "" {
		return base.ColumnResource{}, &base.ResourceNotFoundError{
			Database: &resource.Database,
			Table:    &resource.Table,
		}
	}
	resource.Schema = schemaName
	resource.Table = objectName
	return resource, nil
}

func (q *querySpanExtractor) convertResults(results []redshiftanalysis.QuerySpanResult) []base.QuerySpanResult {
	converted := make([]base.QuerySpanResult, 0, len(results))
	for _, result := range results {
		converted = append(converted, base.QuerySpanResult{
			Name:          result.Name,
			SourceColumns: q.convertColumnList(result.SourceColumns),
			IsPlainField:  result.IsPlainField,
		})
	}
	return converted
}

func (q *querySpanExtractor) convertColumnList(columns []redshiftanalysis.ColumnResource) base.SourceColumnSet {
	result := make(base.SourceColumnSet)
	for _, column := range columns {
		result[base.ColumnResource{
			Database: q.defaultDatabase,
			Schema:   column.Schema,
			Table:    column.Table,
			Column:   column.Column,
		}] = true
	}
	return result
}

func querySpanType(node redshiftast.Node, allSystems bool) base.QueryType {
	switch n := node.(type) {
	case *redshiftast.SelectStmt:
		if hasSelectIntoClause(n) {
			return base.DDL
		}
		if allSystems {
			return base.SelectInfoSchema
		}
		return base.Select
	case *redshiftast.ExplainStmt:
		if hasExplainAnalyze(n) {
			return querySpanType(n.Query, allSystems)
		}
		return base.Explain
	case *redshiftast.RedshiftShowStmt, *redshiftast.RedshiftDescStmt, *redshiftast.VariableShowStmt:
		return base.SelectInfoSchema
	case *redshiftast.VariableSetStmt:
		return base.Select
	case *redshiftast.InsertStmt, *redshiftast.UpdateStmt, *redshiftast.DeleteStmt, *redshiftast.MergeStmt,
		*redshiftast.CopyStmt, *redshiftast.UnloadStmt, *redshiftast.RefreshMatViewStmt, *redshiftast.CallStmt:
		return base.DML
	case *redshiftast.CreateStmt, *redshiftast.CreateTableAsStmt, *redshiftast.ViewStmt, *redshiftast.IndexStmt, *redshiftast.CreateSeqStmt,
		*redshiftast.CreateSchemaStmt, *redshiftast.CreateFunctionStmt, *redshiftast.CreatedbStmt, *redshiftast.CreateTrigStmt,
		*redshiftast.CreateEnumStmt, *redshiftast.CreateDomainStmt, *redshiftast.CreateEventTrigStmt, *redshiftast.CreatePLangStmt,
		*redshiftast.CreateFdwStmt, *redshiftast.CreateForeignServerStmt, *redshiftast.CreateForeignTableStmt,
		*redshiftast.CreateUserMappingStmt, *redshiftast.CreateExtensionStmt, *redshiftast.CreateTableSpaceStmt,
		*redshiftast.CreateAmStmt, *redshiftast.CreatePolicyStmt, *redshiftast.CreatePublicationStmt,
		*redshiftast.CreateSubscriptionStmt, *redshiftast.CreateStatsStmt, *redshiftast.CreateOpClassStmt,
		*redshiftast.CreateOpFamilyStmt, *redshiftast.CreateCastStmt, *redshiftast.CreateTransformStmt,
		*redshiftast.CreateConversionStmt, *redshiftast.CreateRangeStmt, *redshiftast.CreateRoleStmt,
		*redshiftast.DropStmt, *redshiftast.DropdbStmt, *redshiftast.DropRoleStmt, *redshiftast.DropUserMappingStmt,
		*redshiftast.DropSubscriptionStmt, *redshiftast.DropTableSpaceStmt, *redshiftast.DropOwnedStmt,
		*redshiftast.AlterTableStmt, *redshiftast.AlterTableMoveAllStmt, *redshiftast.AlterSeqStmt,
		*redshiftast.AlterEnumStmt, *redshiftast.AlterDomainStmt, *redshiftast.AlterObjectSchemaStmt,
		*redshiftast.AlterOwnerStmt, *redshiftast.AlterDatabaseStmt, *redshiftast.AlterDatabaseSetStmt,
		*redshiftast.AlterDatabaseRefreshCollStmt, *redshiftast.AlterSystemStmt, *redshiftast.AlterCollationStmt,
		*redshiftast.AlterFunctionStmt, *redshiftast.AlterEventTrigStmt, *redshiftast.AlterFdwStmt,
		*redshiftast.AlterForeignServerStmt, *redshiftast.AlterUserMappingStmt, *redshiftast.AlterExtensionStmt,
		*redshiftast.AlterExtensionContentsStmt, *redshiftast.AlterTableSpaceOptionsStmt, *redshiftast.AlterPolicyStmt,
		*redshiftast.AlterPublicationStmt, *redshiftast.AlterSubscriptionStmt, *redshiftast.AlterObjectDependsStmt,
		*redshiftast.AlterOperatorStmt, *redshiftast.AlterTypeStmt, *redshiftast.AlterDefaultPrivilegesStmt,
		*redshiftast.AlterTSDictionaryStmt, *redshiftast.AlterTSConfigurationStmt, *redshiftast.AlterStatsStmt,
		*redshiftast.AlterOpFamilyStmt, *redshiftast.AlterRoleStmt, *redshiftast.AlterRoleSetStmt,
		*redshiftast.RenameStmt, *redshiftast.TruncateStmt, *redshiftast.VacuumStmt, *redshiftast.GrantStmt,
		*redshiftast.GrantRoleStmt, *redshiftast.CommentStmt, *redshiftast.LockStmt, *redshiftast.ClusterStmt,
		*redshiftast.ReindexStmt, *redshiftast.RuleStmt, *redshiftast.CheckPointStmt, *redshiftast.DiscardStmt,
		*redshiftast.LoadStmt, *redshiftast.ConstraintsSetStmt, *redshiftast.FetchStmt, *redshiftast.SecLabelStmt,
		*redshiftast.DoStmt, *redshiftast.ImportForeignSchemaStmt, *redshiftast.ReassignOwnedStmt,
		*redshiftast.RedshiftObjectStmt:
		return base.DDL
	default:
		return base.QueryTypeUnknown
	}
}

func hasSelectIntoClause(n *redshiftast.SelectStmt) bool {
	if n == nil {
		return false
	}
	if n.IntoClause != nil {
		return true
	}
	if hasSelectIntoClause(n.Larg) {
		return true
	}
	if hasSelectIntoClause(n.Rarg) {
		return true
	}
	return false
}

func isQuerySpanNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "not found")
}
