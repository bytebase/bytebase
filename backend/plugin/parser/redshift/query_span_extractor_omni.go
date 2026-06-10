package redshift

import (
	"context"
	"slices"
	"strings"

	redshiftanalysis "github.com/bytebase/omni/redshift/analysis"
	redshiftast "github.com/bytebase/omni/redshift/ast"
	redshiftcatalog "github.com/bytebase/omni/redshift/catalog"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

const redshiftGetDatabaseMetadataError = "failed to get database metadata for database: %s"

type omniQuerySpanExtractor struct {
	defaultDatabase string
	searchPath      []string
	gCtx            base.GetQuerySpanContext
}

func newOmniQuerySpanExtractor(defaultDatabase string, searchPath []string, gCtx base.GetQuerySpanContext) *omniQuerySpanExtractor {
	return &omniQuerySpanExtractor{
		defaultDatabase: defaultDatabase,
		searchPath:      normalizeOmniSearchPath(searchPath),
		gCtx:            gCtx,
	}
}

func normalizeOmniSearchPath(searchPath []string) []string {
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

func (q *omniQuerySpanExtractor) getOmniQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	return q.getOmniQuerySpanWithFunctionStack(ctx, statement, map[string]bool{})
}

func (q *omniQuerySpanExtractor) getOmniQuerySpanWithFunctionStack(ctx context.Context, statement string, functionStack map[string]bool) (*base.QuerySpan, error) {
	stmts, err := ParseRedshiftOmni(statement)
	if err != nil {
		return nil, err
	}
	if len(stmts) == 0 || stmts[0].Empty() {
		return emptyOmniQuerySpan(base.Select), nil
	}
	if len(stmts) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", len(stmts))
	}

	node := stmts[0].AST
	queryType := omniQuerySpanType(node, false /* allSystems */)
	if explain, ok := node.(*redshiftast.ExplainStmt); ok && hasOmniExplainAnalyze(explain) {
		return q.getOmniExplainAnalyzeQuerySpan(ctx, explain.Query, queryType)
	}
	if unload, ok := node.(*redshiftast.UnloadStmt); ok {
		return q.getOmniUnloadQuerySpan(ctx, unload, functionStack)
	}
	if _, ok := node.(*redshiftast.VariableSetStmt); ok {
		return emptyOmniQuerySpan(queryType), nil
	}
	if queryType != base.Select {
		return emptyOmniQuerySpan(queryType), nil
	}

	accessTables, earlySpan, err := q.collectOmniSelectAccessTables(ctx, node)
	if err != nil {
		return nil, err
	}
	if earlySpan != nil {
		return earlySpan, nil
	}

	cat, err := q.newOmniQuerySpanCatalog(ctx)
	if err != nil {
		return nil, err
	}
	if err := q.installOmniQuerySpanFunctions(ctx, cat); err != nil {
		return nil, err
	}
	omniSpan, err := redshiftanalysis.GetQuerySpan(cat, statement)
	if err != nil {
		if isOmniQuerySpanNotFound(err) {
			return q.notFoundOmniQuerySpan(err), nil
		}
		return nil, err
	}

	results := q.convertOmniResults(omniSpan.Results)
	functionAccessTables, functionResultSources, functionIssues, err := q.collectOmniFunctionBodySources(ctx, node, functionStack, make(map[string]omniFunctionBodySources))
	if err != nil {
		return nil, err
	}
	for source := range functionAccessTables {
		accessTables[source] = true
	}
	if functionIssues.functionNotSupportedError != nil && len(accessTables) == 0 {
		accessTables[base.ColumnResource{Database: q.defaultDatabase}] = true
	}
	mergeOmniFunctionResultSources(results, functionResultSources)
	if hasOmniRangeVar(node) && hasResultWithoutSource(results) && !q.allOmniAccessTablesExist(ctx, accessTables) {
		return q.notFoundOmniQuerySpan(&base.ResourceNotFoundError{Database: &q.defaultDatabase}), nil
	}

	return &base.QuerySpan{
		Type:                      base.Select,
		SourceColumns:             accessTables,
		PredicateColumns:          q.convertOmniColumnList(omniSpan.PredicateColumns),
		Results:                   results,
		NotFoundError:             functionIssues.notFoundError,
		FunctionNotSupportedError: functionIssues.functionNotSupportedError,
	}, nil
}

func (q *omniQuerySpanExtractor) getOmniExplainAnalyzeQuerySpan(ctx context.Context, query redshiftast.Node, queryType base.QueryType) (*base.QuerySpan, error) {
	if queryType != base.Select {
		return emptyOmniQuerySpan(queryType), nil
	}

	accessTables, earlySpan, err := q.collectOmniSelectAccessTables(ctx, query)
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

func (q *omniQuerySpanExtractor) getOmniUnloadQuerySpan(ctx context.Context, unload *redshiftast.UnloadStmt, functionStack map[string]bool) (*base.QuerySpan, error) {
	span, err := q.getOmniQuerySpanWithFunctionStack(ctx, unload.Query, functionStack)
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

func (q *omniQuerySpanExtractor) collectOmniSelectAccessTables(ctx context.Context, node redshiftast.Node) (base.SourceColumnSet, *base.QuerySpan, error) {
	accessTables, err := q.collectOmniAccessTables(ctx, node)
	if err != nil {
		var resourceNotFound *base.ResourceNotFoundError
		if errors.As(err, &resourceNotFound) {
			return nil, q.notFoundOmniQuerySpan(resourceNotFound), nil
		}
		return nil, nil, err
	}
	allSystems, mixed := isMixedQuery(accessTables)
	if mixed {
		return nil, nil, base.MixUserSystemTablesError
	}
	if allSystems {
		return nil, emptyOmniQuerySpan(base.SelectInfoSchema), nil
	}
	return accessTables, nil, nil
}

func (q *omniQuerySpanExtractor) notFoundOmniQuerySpan(err error) *base.QuerySpan {
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

func emptyOmniQuerySpan(queryType base.QueryType) *base.QuerySpan {
	return &base.QuerySpan{
		Type:          queryType,
		SourceColumns: base.SourceColumnSet{},
		Results:       []base.QuerySpanResult{},
	}
}

func (q *omniQuerySpanExtractor) newOmniQuerySpanCatalog(ctx context.Context) (*redshiftcatalog.Catalog, error) {
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
	cat.SetRelationResolver(&redshiftQuerySpanRelationResolver{
		metadata: metadata,
	})
	return cat, nil
}

func (q *omniQuerySpanExtractor) installOmniQuerySpanFunctions(ctx context.Context, cat *redshiftcatalog.Catalog) error {
	if q.gCtx.GetDatabaseMetadataFunc == nil {
		return nil
	}
	_, metadata, err := q.gCtx.GetDatabaseMetadataFunc(ctx, q.gCtx.InstanceID, q.defaultDatabase)
	if err != nil {
		return errors.Wrapf(err, redshiftGetDatabaseMetadataError, q.defaultDatabase)
	}
	if metadata == nil {
		return nil
	}
	for _, schema := range metadata.GetProto().GetSchemas() {
		for _, function := range schema.GetFunctions() {
			definition := strings.TrimSpace(function.GetDefinition())
			if definition == "" {
				continue
			}
			if _, err := cat.Exec(definition, nil); err != nil {
				continue
			}
		}
	}
	return nil
}

type omniFunctionBodySources struct {
	accessTables base.SourceColumnSet
	results      []base.QuerySpanResult
	issues       omniFunctionBodyIssues
}

type omniFunctionBodyIssues struct {
	notFoundError             error
	functionNotSupportedError error
}

func (i *omniFunctionBodyIssues) merge(other omniFunctionBodyIssues) {
	if i.notFoundError == nil {
		i.notFoundError = other.notFoundError
	}
	if i.functionNotSupportedError == nil {
		i.functionNotSupportedError = other.functionNotSupportedError
	}
}

type omniFunctionSourceProjection struct {
	names   []string
	sources []base.SourceColumnSet
}

type omniFunctionSourceScope map[string]omniFunctionSourceProjection

type redshiftQuerySpanRelationResolver struct {
	metadata *model.DatabaseMetadata
}

func (r *redshiftQuerySpanRelationResolver) ResolveRelation(schemaName, relationName string, searchPath []string) (*redshiftcatalog.RelationSpec, error) {
	if r.metadata == nil {
		return nil, nil
	}

	resolvedSchemaName, resolvedRelationName := schemaName, relationName
	if resolvedSchemaName == "" {
		resolvedSchemaName, resolvedRelationName = r.metadata.SearchObject(searchPath, relationName)
		if resolvedSchemaName == "" && resolvedRelationName == "" {
			return nil, nil
		}
	}

	schema := r.metadata.GetSchemaMetadata(resolvedSchemaName)
	if schema == nil {
		return nil, nil
	}
	return redshiftRelationSpecFromMetadata(r.metadata, schema, resolvedSchemaName, resolvedRelationName), nil
}

func redshiftRelationSpecFromMetadata(metadata *model.DatabaseMetadata, schema *model.SchemaMetadata, schemaName, relationName string) *redshiftcatalog.RelationSpec {
	if table := schema.GetTable(relationName); table != nil {
		return redshiftTableRelationSpec(schemaName, table.GetProto().GetName(), table.GetProto().GetColumns())
	}
	if table := schema.GetExternalTable(relationName); table != nil {
		return redshiftTableRelationSpec(schemaName, table.GetProto().GetName(), table.GetProto().GetColumns())
	}
	if view := schema.GetView(relationName); view != nil {
		if definition := strings.TrimSpace(view.GetDefinition()); definition != "" {
			return &redshiftcatalog.RelationSpec{
				SchemaName: schemaName,
				Name:       view.GetName(),
				Kind:       redshiftcatalog.RelationKindView,
				Definition: qualifyRedshiftViewDefinition(definition, schemaName, metadata),
			}
		}
		return redshiftTableRelationSpec(schemaName, view.GetName(), view.GetColumns())
	}
	if view := schema.GetMaterializedView(relationName); view != nil {
		if definition := strings.TrimSpace(view.GetDefinition()); definition != "" {
			return &redshiftcatalog.RelationSpec{
				SchemaName: schemaName,
				Name:       view.GetName(),
				Kind:       redshiftcatalog.RelationKindMaterializedView,
				Definition: qualifyRedshiftViewDefinition(definition, schemaName, metadata),
			}
		}
		return redshiftTableRelationSpec(schemaName, view.GetName(), nil)
	}
	if sequence := schema.GetSequence(relationName); sequence != nil {
		return redshiftSequenceRelationSpec(schemaName, sequence.GetName())
	}
	return nil
}

func redshiftTableRelationSpec(schemaName, tableName string, columns []*storepb.ColumnMetadata) *redshiftcatalog.RelationSpec {
	return &redshiftcatalog.RelationSpec{
		SchemaName: schemaName,
		Name:       tableName,
		Kind:       redshiftcatalog.RelationKindTable,
		Columns:    redshiftRelationColumnSpecs(columns),
	}
}

func redshiftRelationColumnSpecs(columns []*storepb.ColumnMetadata) []redshiftcatalog.RelationColumnSpec {
	result := make([]redshiftcatalog.RelationColumnSpec, 0, len(columns))
	for _, column := range columns {
		if column == nil || column.GetName() == "" {
			continue
		}
		result = append(result, redshiftcatalog.RelationColumnSpec{
			Name: column.GetName(),
			Type: normalizeCompletionType(column.GetType()),
		})
	}
	if len(result) == 0 {
		result = append(result, redshiftcatalog.RelationColumnSpec{
			Name: "__bytebase_query_span_placeholder",
			Type: "text",
		})
	}
	return result
}

func redshiftSequenceRelationSpec(schemaName, sequenceName string) *redshiftcatalog.RelationSpec {
	return &redshiftcatalog.RelationSpec{
		SchemaName: schemaName,
		Name:       sequenceName,
		Kind:       redshiftcatalog.RelationKindTable,
		Columns: []redshiftcatalog.RelationColumnSpec{
			{Name: "last_value", Type: "bigint"},
			{Name: "log_cnt", Type: "bigint"},
			{Name: "is_called", Type: "boolean"},
		},
	}
}

type redshiftSchemaQualificationEdit struct {
	offset int
	schema string
}

func qualifyRedshiftViewDefinition(definition, schemaName string, metadata *model.DatabaseMetadata) string {
	stmts, err := ParseRedshiftOmni(definition)
	if err != nil || len(stmts) != 1 {
		return definition
	}

	var query redshiftast.Node
	switch stmt := stmts[0].AST.(type) {
	case *redshiftast.ViewStmt:
		query = stmt.Query
	case *redshiftast.CreateTableAsStmt:
		query = stmt.Query
	case *redshiftast.SelectStmt:
		query = stmt
	default:
		return definition
	}
	if query == nil {
		return definition
	}

	searchPath := normalizeOmniSearchPath([]string{schemaName, "public"})
	var edits []redshiftSchemaQualificationEdit
	collectRedshiftSchemaQualificationEdits(metadata, searchPath, query, map[string]bool{}, &edits)
	if len(edits) == 0 {
		return definition
	}

	slices.SortFunc(edits, func(a, b redshiftSchemaQualificationEdit) int {
		return b.offset - a.offset
	})
	result := definition
	for _, edit := range edits {
		if edit.offset < 0 || edit.offset > len(result) {
			continue
		}
		result = result[:edit.offset] + quoteIdent(edit.schema) + "." + result[edit.offset:]
	}
	return result
}

func collectRedshiftSchemaQualificationEdits(metadata *model.DatabaseMetadata, searchPath []string, node redshiftast.Node, ctes map[string]bool, edits *[]redshiftSchemaQualificationEdit) {
	if node == nil {
		return
	}
	if selectStmt, ok := node.(*redshiftast.SelectStmt); ok {
		collectRedshiftSchemaQualificationEditsFromSelect(metadata, searchPath, selectStmt, ctes, edits)
		return
	}

	redshiftast.Inspect(node, func(n redshiftast.Node) bool {
		if n != node {
			if selectStmt, ok := n.(*redshiftast.SelectStmt); ok {
				collectRedshiftSchemaQualificationEditsFromSelect(metadata, searchPath, selectStmt, ctes, edits)
				return false
			}
		}
		rangeVar, ok := n.(*redshiftast.RangeVar)
		if !ok || rangeVar == nil || rangeVar.Relname == "" || rangeVar.Schemaname != "" || rangeVar.Catalogname != "" {
			return true
		}
		if isOmniCTEReference(rangeVar, ctes) || rangeVar.Loc.Start < 0 {
			return true
		}
		schemaName, _ := metadata.SearchObject(searchPath, rangeVar.Relname)
		if schemaName == "" {
			return true
		}
		*edits = append(*edits, redshiftSchemaQualificationEdit{
			offset: rangeVar.Loc.Start,
			schema: schemaName,
		})
		return true
	})
}

func collectRedshiftSchemaQualificationEditsFromSelect(metadata *model.DatabaseMetadata, searchPath []string, selectStmt *redshiftast.SelectStmt, ctes map[string]bool, edits *[]redshiftSchemaQualificationEdit) {
	localCTEs := cloneOmniCTENameSet(ctes)
	if selectStmt.WithClause != nil && selectStmt.WithClause.Ctes != nil {
		cteScope := cloneOmniCTENameSet(ctes)
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
			collectRedshiftSchemaQualificationEdits(metadata, searchPath, cte.Ctequery, cteScope, edits)
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
		if node != nil {
			collectRedshiftSchemaQualificationEdits(metadata, searchPath, node, localCTEs, edits)
		}
	}
	if selectStmt.IntoClause != nil {
		collectRedshiftSchemaQualificationEdits(metadata, searchPath, selectStmt.IntoClause, localCTEs, edits)
	}
	if selectStmt.Larg != nil {
		collectRedshiftSchemaQualificationEdits(metadata, searchPath, selectStmt.Larg, localCTEs, edits)
	}
	if selectStmt.Rarg != nil {
		collectRedshiftSchemaQualificationEdits(metadata, searchPath, selectStmt.Rarg, localCTEs, edits)
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
		collectRedshiftSchemaQualificationEditsFromList(metadata, searchPath, list, localCTEs, edits)
	}
}

func collectRedshiftSchemaQualificationEditsFromList(metadata *model.DatabaseMetadata, searchPath []string, list *redshiftast.List, ctes map[string]bool, edits *[]redshiftSchemaQualificationEdit) {
	if list == nil {
		return
	}
	for _, item := range list.Items {
		collectRedshiftSchemaQualificationEdits(metadata, searchPath, item, ctes, edits)
	}
}

func (q *omniQuerySpanExtractor) collectOmniAccessTables(ctx context.Context, node redshiftast.Node) (base.SourceColumnSet, error) {
	result := make(base.SourceColumnSet)
	if err := q.collectOmniAccessTablesFromNode(ctx, node, map[string]bool{}, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (q *omniQuerySpanExtractor) collectOmniAccessTablesFromNode(ctx context.Context, node redshiftast.Node, ctes map[string]bool, result base.SourceColumnSet) error {
	if node == nil {
		return nil
	}
	if selectStmt, ok := node.(*redshiftast.SelectStmt); ok {
		return q.collectOmniAccessTablesFromSelect(ctx, selectStmt, ctes, result)
	}

	var err error
	redshiftast.Inspect(node, func(n redshiftast.Node) bool {
		if err != nil {
			return false
		}
		if n != node {
			if selectStmt, ok := n.(*redshiftast.SelectStmt); ok {
				err = q.collectOmniAccessTablesFromSelect(ctx, selectStmt, ctes, result)
				return false
			}
		}
		rangeVar, ok := n.(*redshiftast.RangeVar)
		if !ok || rangeVar == nil || rangeVar.Relname == "" {
			return true
		}
		if isOmniCTEReference(rangeVar, ctes) {
			return true
		}

		resource, resolveErr := q.resolveOmniRangeVar(ctx, rangeVar)
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

func (q *omniQuerySpanExtractor) collectOmniAccessTablesFromSelect(ctx context.Context, selectStmt *redshiftast.SelectStmt, ctes map[string]bool, result base.SourceColumnSet) error {
	localCTEs := cloneOmniCTENameSet(ctes)
	if selectStmt.WithClause != nil && selectStmt.WithClause.Ctes != nil {
		cteScope := cloneOmniCTENameSet(ctes)
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
			if err := q.collectOmniAccessTablesFromNode(ctx, cte.Ctequery, cteScope, result); err != nil {
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
		if err := q.collectOmniAccessTablesFromNode(ctx, node, localCTEs, result); err != nil {
			return err
		}
	}
	if selectStmt.IntoClause != nil {
		if err := q.collectOmniAccessTablesFromNode(ctx, selectStmt.IntoClause, localCTEs, result); err != nil {
			return err
		}
	}
	if selectStmt.Larg != nil {
		if err := q.collectOmniAccessTablesFromNode(ctx, selectStmt.Larg, localCTEs, result); err != nil {
			return err
		}
	}
	if selectStmt.Rarg != nil {
		if err := q.collectOmniAccessTablesFromNode(ctx, selectStmt.Rarg, localCTEs, result); err != nil {
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
		if err := q.collectOmniAccessTablesFromList(ctx, list, localCTEs, result); err != nil {
			return err
		}
	}
	return nil
}

func (q *omniQuerySpanExtractor) collectOmniAccessTablesFromList(ctx context.Context, list *redshiftast.List, ctes map[string]bool, result base.SourceColumnSet) error {
	if list == nil {
		return nil
	}
	for _, item := range list.Items {
		if err := q.collectOmniAccessTablesFromNode(ctx, item, ctes, result); err != nil {
			return err
		}
	}
	return nil
}

func (q *omniQuerySpanExtractor) collectOmniFunctionBodySources(ctx context.Context, node redshiftast.Node, functionStack map[string]bool, cache map[string]omniFunctionBodySources) (base.SourceColumnSet, []base.SourceColumnSet, omniFunctionBodyIssues, error) {
	accessTables, issues, err := q.collectOmniFunctionBodyAccessTables(ctx, node, functionStack, cache)
	if err != nil {
		return nil, nil, omniFunctionBodyIssues{}, err
	}

	var resultSources []base.SourceColumnSet
	if selectStmt, ok := node.(*redshiftast.SelectStmt); ok {
		var sourceIssues omniFunctionBodyIssues
		resultSources, sourceIssues, err = q.collectOmniSelectFunctionResultSources(ctx, selectStmt, functionStack, cache, nil)
		if err != nil {
			return nil, nil, omniFunctionBodyIssues{}, err
		}
		issues.merge(sourceIssues)
	}
	return accessTables, resultSources, issues, nil
}

func (q *omniQuerySpanExtractor) collectOmniFunctionBodyAccessTables(ctx context.Context, node redshiftast.Node, functionStack map[string]bool, cache map[string]omniFunctionBodySources) (base.SourceColumnSet, omniFunctionBodyIssues, error) {
	result := make(base.SourceColumnSet)
	if node == nil {
		return result, omniFunctionBodyIssues{}, nil
	}

	var err error
	issues := omniFunctionBodyIssues{}
	redshiftast.Inspect(node, func(n redshiftast.Node) bool {
		if err != nil {
			return false
		}
		call, ok := n.(*redshiftast.FuncCall)
		if !ok {
			return true
		}
		sources, sourceErr := q.omniFunctionBodySourcesForCall(ctx, call, functionStack, cache)
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

func (q *omniQuerySpanExtractor) collectOmniSelectFunctionResultSources(ctx context.Context, selectStmt *redshiftast.SelectStmt, functionStack map[string]bool, cache map[string]omniFunctionBodySources, outerScope omniFunctionSourceScope) ([]base.SourceColumnSet, omniFunctionBodyIssues, error) {
	if selectStmt == nil || selectStmt.TargetList == nil {
		return nil, omniFunctionBodyIssues{}, nil
	}

	scope, issues, err := q.omniFunctionSourceScopeForSelect(ctx, selectStmt, functionStack, cache, outerScope)
	if err != nil {
		return nil, omniFunctionBodyIssues{}, err
	}

	result := make([]base.SourceColumnSet, 0, selectStmt.TargetList.Len())
	for _, item := range selectStmt.TargetList.Items {
		target, ok := item.(*redshiftast.ResTarget)
		if !ok {
			result = append(result, nil)
			continue
		}
		sources, sourceIssues, err := q.collectOmniFunctionBodyResultSources(ctx, target.Val, functionStack, cache, scope)
		if err != nil {
			return nil, omniFunctionBodyIssues{}, err
		}
		issues.merge(sourceIssues)
		result = append(result, sources)
	}
	return result, issues, nil
}

func (q *omniQuerySpanExtractor) omniFunctionSourceScopeForSelect(ctx context.Context, selectStmt *redshiftast.SelectStmt, functionStack map[string]bool, cache map[string]omniFunctionBodySources, outerScope omniFunctionSourceScope) (omniFunctionSourceScope, omniFunctionBodyIssues, error) {
	scope := cloneOmniFunctionSourceScope(outerScope)
	issues := omniFunctionBodyIssues{}

	if selectStmt.WithClause != nil && selectStmt.WithClause.Ctes != nil {
		cteScope := cloneOmniFunctionSourceScope(scope)
		for _, item := range selectStmt.WithClause.Ctes.Items {
			cte, ok := item.(*redshiftast.CommonTableExpr)
			if !ok || cte.Ctename == "" {
				continue
			}
			cteSelect, ok := cte.Ctequery.(*redshiftast.SelectStmt)
			if !ok {
				continue
			}
			sources, sourceIssues, err := q.collectOmniSelectFunctionResultSources(ctx, cteSelect, functionStack, cache, cteScope)
			if err != nil {
				return nil, omniFunctionBodyIssues{}, err
			}
			issues.merge(sourceIssues)
			projection := omniFunctionSourceProjection{
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

	sourceIssues, err := q.collectOmniFunctionSourceScopeFromList(ctx, selectStmt.FromClause, functionStack, cache, scope)
	if err != nil {
		return nil, omniFunctionBodyIssues{}, err
	}
	issues.merge(sourceIssues)
	return scope, issues, nil
}

func (q *omniQuerySpanExtractor) collectOmniFunctionSourceScopeFromList(ctx context.Context, list *redshiftast.List, functionStack map[string]bool, cache map[string]omniFunctionBodySources, scope omniFunctionSourceScope) (omniFunctionBodyIssues, error) {
	if list == nil {
		return omniFunctionBodyIssues{}, nil
	}
	issues := omniFunctionBodyIssues{}
	for _, item := range list.Items {
		sourceIssues, err := q.collectOmniFunctionSourceScopeFromNode(ctx, item, functionStack, cache, scope)
		if err != nil {
			return omniFunctionBodyIssues{}, err
		}
		issues.merge(sourceIssues)
	}
	return issues, nil
}

func (q *omniQuerySpanExtractor) collectOmniFunctionSourceScopeFromNode(ctx context.Context, node redshiftast.Node, functionStack map[string]bool, cache map[string]omniFunctionBodySources, scope omniFunctionSourceScope) (omniFunctionBodyIssues, error) {
	switch n := node.(type) {
	case *redshiftast.RangeVar:
		if n == nil || n.Alias == nil || n.Alias.Aliasname == "" {
			return omniFunctionBodyIssues{}, nil
		}
		projection, ok := scope[strings.ToLower(n.Relname)]
		if !ok {
			return omniFunctionBodyIssues{}, nil
		}
		if aliasNames := redshiftStringList(n.Alias.Colnames); len(aliasNames) > 0 {
			projection.names = aliasNames
		}
		scope[strings.ToLower(n.Alias.Aliasname)] = projection
		return omniFunctionBodyIssues{}, nil
	case *redshiftast.RangeSubselect:
		if n == nil || n.Alias == nil || n.Alias.Aliasname == "" {
			return omniFunctionBodyIssues{}, nil
		}
		selectStmt, ok := n.Subquery.(*redshiftast.SelectStmt)
		if !ok {
			return omniFunctionBodyIssues{}, nil
		}
		sources, issues, err := q.collectOmniSelectFunctionResultSources(ctx, selectStmt, functionStack, cache, scope)
		if err != nil {
			return omniFunctionBodyIssues{}, err
		}
		projection := omniFunctionSourceProjection{
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
			return omniFunctionBodyIssues{}, nil
		}
		leftIssues, err := q.collectOmniFunctionSourceScopeFromNode(ctx, n.Larg, functionStack, cache, scope)
		if err != nil {
			return omniFunctionBodyIssues{}, err
		}
		rightIssues, err := q.collectOmniFunctionSourceScopeFromNode(ctx, n.Rarg, functionStack, cache, scope)
		if err != nil {
			return omniFunctionBodyIssues{}, err
		}
		leftIssues.merge(rightIssues)
		return leftIssues, nil
	default:
		return omniFunctionBodyIssues{}, nil
	}
}

func (q *omniQuerySpanExtractor) collectOmniFunctionBodyResultSources(ctx context.Context, node redshiftast.Node, functionStack map[string]bool, cache map[string]omniFunctionBodySources, scope omniFunctionSourceScope) (base.SourceColumnSet, omniFunctionBodyIssues, error) {
	result := make(base.SourceColumnSet)
	if node == nil {
		return result, omniFunctionBodyIssues{}, nil
	}

	var err error
	issues := omniFunctionBodyIssues{}
	redshiftast.Inspect(node, func(n redshiftast.Node) bool {
		if err != nil {
			return false
		}
		if columnRef, ok := n.(*redshiftast.ColumnRef); ok {
			mergeOmniSourceColumnSet(result, resolveOmniFunctionColumnRef(scope, columnRef))
			return true
		}
		call, ok := n.(*redshiftast.FuncCall)
		if !ok {
			return true
		}
		sources, sourceErr := q.omniFunctionBodySourcesForCall(ctx, call, functionStack, cache)
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

func (q *omniQuerySpanExtractor) omniFunctionBodySourcesForCall(ctx context.Context, call *redshiftast.FuncCall, functionStack map[string]bool, cache map[string]omniFunctionBodySources) (omniFunctionBodySources, error) {
	schemaName, functionName := omniFunctionCallName(call.Funcname)
	if functionName == "" || IsSystemFunction(functionName, "") {
		return omniFunctionBodySources{accessTables: base.SourceColumnSet{}}, nil
	}

	functionSchema, function, findErr := q.findOmniFunctionMetadata(ctx, schemaName, functionName, call.Args.Len())
	if findErr != nil || function == nil {
		return omniFunctionBodySources{accessTables: base.SourceColumnSet{}}, findErr
	}

	cacheKey := strings.ToLower(functionSchema + "." + function.GetName() + "/" + function.GetSignature())
	if sources, ok := cache[cacheKey]; ok {
		return sources, nil
	}
	if functionStack[cacheKey] {
		return omniFunctionBodySources{accessTables: base.SourceColumnSet{}}, nil
	}

	body, ok, bodyErr := extractOmniSQLFunctionBody(function.GetDefinition())
	if bodyErr != nil {
		// Body extraction failures should fail closed through QuerySpan.FunctionNotSupportedError.
		//nolint:nilerr
		return omniFunctionBodySources{
			accessTables: base.SourceColumnSet{},
			issues: omniFunctionBodyIssues{
				functionNotSupportedError: &base.FunctionNotSupportedError{
					Err:        bodyErr,
					Function:   functionSchema + "." + function.GetName(),
					Definition: function.GetDefinition(),
				},
			},
		}, nil
	}
	if !ok {
		return omniFunctionBodySources{
			accessTables: base.SourceColumnSet{},
			issues: omniFunctionBodyIssues{
				functionNotSupportedError: &base.FunctionNotSupportedError{
					Err:        errors.New("unsupported or empty SQL function body"),
					Function:   functionSchema + "." + function.GetName(),
					Definition: function.GetDefinition(),
				},
			},
		}, nil
	}

	functionStack[cacheKey] = true
	span, querySpanErr := q.getOmniQuerySpanWithFunctionStack(ctx, body, functionStack)
	delete(functionStack, cacheKey)
	if querySpanErr != nil {
		return omniFunctionBodySources{}, errors.Wrapf(querySpanErr, "failed to get query span for function: %s.%s", functionSchema, functionName)
	}
	sources := omniFunctionBodySources{
		accessTables: span.SourceColumns,
		results:      span.Results,
		issues: omniFunctionBodyIssues{
			notFoundError:             span.NotFoundError,
			functionNotSupportedError: span.FunctionNotSupportedError,
		},
	}
	if sources.issues.notFoundError == nil && sources.issues.functionNotSupportedError == nil {
		cache[cacheKey] = sources
	}
	return sources, nil
}

func (q *omniQuerySpanExtractor) findOmniFunctionMetadata(ctx context.Context, schemaName, functionName string, argCount int) (string, *storepb.FunctionMetadata, error) {
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
		count, ok := omniFunctionParameterCount(function.GetDefinition())
		if ok && count == argCount {
			return schemas[i], function, nil
		}
	}
	return "", nil, nil
}

func omniFunctionCallName(list *redshiftast.List) (string, string) {
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

func omniFunctionParameterCount(definition string) (int, bool) {
	stmts, err := ParseRedshiftOmni(definition)
	if err != nil || len(stmts) != 1 {
		return 0, false
	}
	stmt, ok := stmts[0].AST.(*redshiftast.CreateFunctionStmt)
	if !ok || stmt.Parameters == nil {
		return 0, ok
	}
	return stmt.Parameters.Len(), true
}

func extractOmniSQLFunctionBody(definition string) (string, bool, error) {
	stmts, err := ParseRedshiftOmni(definition)
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
				language = omniDefElemString(elem.Arg)
			case "as":
				body = omniDefElemString(elem.Arg)
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

func omniDefElemString(node redshiftast.Node) string {
	switch value := node.(type) {
	case *redshiftast.String:
		return value.Str
	case *redshiftast.List:
		if len(value.Items) == 0 {
			return ""
		}
		return omniDefElemString(value.Items[0])
	default:
		return ""
	}
}

func mergeOmniFunctionResultSources(results []base.QuerySpanResult, functionResultSources []base.SourceColumnSet) {
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

func cloneOmniFunctionSourceScope(scope omniFunctionSourceScope) omniFunctionSourceScope {
	result := make(omniFunctionSourceScope, len(scope))
	for name, projection := range scope {
		result[name] = projection
	}
	return result
}

func mergeOmniSourceColumnSet(dst, src base.SourceColumnSet) {
	for source := range src {
		dst[source] = true
	}
}

func resolveOmniFunctionColumnRef(scope omniFunctionSourceScope, columnRef *redshiftast.ColumnRef) base.SourceColumnSet {
	parts := redshiftColumnRefParts(columnRef)
	switch len(parts) {
	case 1:
		return resolveUnqualifiedOmniFunctionColumn(scope, parts[0])
	case 2:
		if projection, ok := scope[strings.ToLower(parts[0])]; ok {
			return projection.omniFunctionSourcesForColumn(parts[1])
		}
		return base.SourceColumnSet{}
	default:
		return base.SourceColumnSet{}
	}
}

func resolveUnqualifiedOmniFunctionColumn(scope omniFunctionSourceScope, columnName string) base.SourceColumnSet {
	var result base.SourceColumnSet
	for _, projection := range scope {
		sources := projection.omniFunctionSourcesForColumn(columnName)
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

func (p omniFunctionSourceProjection) omniFunctionSourcesForColumn(columnName string) base.SourceColumnSet {
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
	if parts := redshiftColumnRefParts(asOmniColumnRef(target.Val)); len(parts) > 0 {
		return parts[len(parts)-1]
	}
	if call, ok := target.Val.(*redshiftast.FuncCall); ok {
		_, functionName := omniFunctionCallName(call.Funcname)
		return functionName
	}
	return ""
}

func asOmniColumnRef(node redshiftast.Node) *redshiftast.ColumnRef {
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

func cloneOmniCTENameSet(ctes map[string]bool) map[string]bool {
	result := make(map[string]bool, len(ctes))
	for name, ok := range ctes {
		result[name] = ok
	}
	return result
}

func isOmniCTEReference(rangeVar *redshiftast.RangeVar, ctes map[string]bool) bool {
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

func hasOmniRangeVar(node redshiftast.Node) bool {
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

func (q *omniQuerySpanExtractor) allOmniAccessTablesExist(ctx context.Context, accessTables base.SourceColumnSet) bool {
	if len(accessTables) == 0 || q.gCtx.GetDatabaseMetadataFunc == nil {
		return false
	}
	for resource := range accessTables {
		if !q.omniAccessTableExists(ctx, resource) {
			return false
		}
	}
	return true
}

func (q *omniQuerySpanExtractor) omniAccessTableExists(ctx context.Context, resource base.ColumnResource) bool {
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
		schemaName, objectName := metadata.SearchObject(q.searchPath, resource.Table)
		return schemaName != "" || objectName != ""
	}
	schema := metadata.GetSchemaMetadata(resource.Schema)
	if schema == nil {
		return false
	}
	return schema.GetTable(resource.Table) != nil ||
		schema.GetView(resource.Table) != nil ||
		schema.GetMaterializedView(resource.Table) != nil ||
		schema.GetExternalTable(resource.Table) != nil ||
		schema.GetFunction(resource.Table) != nil ||
		schema.GetSequence(resource.Table) != nil
}

func (q *omniQuerySpanExtractor) resolveOmniRangeVar(ctx context.Context, rangeVar *redshiftast.RangeVar) (base.ColumnResource, error) {
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
	schemaName, objectName := metadata.SearchObject(q.searchPath, resource.Table)
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

func (q *omniQuerySpanExtractor) convertOmniResults(results []redshiftanalysis.QuerySpanResult) []base.QuerySpanResult {
	converted := make([]base.QuerySpanResult, 0, len(results))
	for _, result := range results {
		converted = append(converted, base.QuerySpanResult{
			Name:          result.Name,
			SourceColumns: q.convertOmniColumnList(result.SourceColumns),
			IsPlainField:  result.IsPlainField,
		})
	}
	return converted
}

func (q *omniQuerySpanExtractor) convertOmniColumnList(columns []redshiftanalysis.ColumnResource) base.SourceColumnSet {
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

func omniQuerySpanType(node redshiftast.Node, allSystems bool) base.QueryType {
	switch n := node.(type) {
	case *redshiftast.SelectStmt:
		if hasOmniSelectIntoClause(n) {
			return base.DDL
		}
		if allSystems {
			return base.SelectInfoSchema
		}
		return base.Select
	case *redshiftast.ExplainStmt:
		if hasOmniExplainAnalyze(n) {
			return omniQuerySpanType(n.Query, allSystems)
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

func hasOmniSelectIntoClause(n *redshiftast.SelectStmt) bool {
	if n == nil {
		return false
	}
	if n.IntoClause != nil {
		return true
	}
	if hasOmniSelectIntoClause(n.Larg) {
		return true
	}
	if hasOmniSelectIntoClause(n.Rarg) {
		return true
	}
	return false
}

func isOmniQuerySpanNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "not found")
}
