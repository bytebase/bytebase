package redshift

import (
	"context"
	"fmt"
	"strings"

	redshiftanalysis "github.com/bytebase/omni/redshift/analysis"
	redshiftast "github.com/bytebase/omni/redshift/ast"
	redshiftcatalog "github.com/bytebase/omni/redshift/catalog"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

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

	cat, err := q.buildOmniQuerySpanCatalog(ctx)
	if err != nil {
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
	if hasOmniRangeVar(node) && hasResultWithoutSource(results) && !q.allOmniAccessTablesExist(ctx, accessTables) {
		return q.notFoundOmniQuerySpan(&base.ResourceNotFoundError{Database: &q.defaultDatabase}), nil
	}

	return &base.QuerySpan{
		Type:             base.Select,
		SourceColumns:    accessTables,
		PredicateColumns: q.convertOmniColumnList(omniSpan.PredicateColumns),
		Results:          results,
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

func (q *omniQuerySpanExtractor) buildOmniQuerySpanCatalog(ctx context.Context) (*redshiftcatalog.Catalog, error) {
	if q.gCtx.GetDatabaseMetadataFunc == nil {
		return nil, errors.New("GetDatabaseMetadataFunc is not set in GetQuerySpanContext")
	}
	_, metadata, err := q.gCtx.GetDatabaseMetadataFunc(ctx, q.gCtx.InstanceID, q.defaultDatabase)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", q.defaultDatabase)
	}
	if metadata == nil {
		return nil, &base.ResourceNotFoundError{Database: &q.defaultDatabase}
	}

	cat := redshiftcatalog.New()
	for _, schemaName := range metadata.ListSchemaNames() {
		schemaMeta := metadata.GetSchemaMetadata(schemaName)
		if schemaMeta == nil {
			continue
		}
		execOmniQuerySpanDDL(cat, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", quoteIdent(schemaName)))

		for _, tableName := range schemaMeta.ListTableNames() {
			tableMeta := schemaMeta.GetTable(tableName)
			if tableMeta == nil {
				continue
			}
			execOmniQuerySpanDDL(cat, createTableDDL(schemaName, tableName, tableMeta.GetProto().GetColumns()))
		}
		for _, tableName := range schemaMeta.ListForeignTableNames() {
			tableMeta := schemaMeta.GetExternalTable(tableName)
			if tableMeta == nil {
				continue
			}
			execOmniQuerySpanDDL(cat, createTableDDL(schemaName, tableName, tableMeta.GetProto().GetColumns()))
		}
		for _, viewName := range schemaMeta.ListViewNames() {
			viewMeta := schemaMeta.GetView(viewName)
			if viewMeta == nil {
				continue
			}
			cat.SetSearchPath([]string{schemaName, "public"})
			if viewMeta.GetDefinition() == "" || !execOmniQuerySpanDDL(cat, createQuerySpanViewDDL("VIEW", schemaName, viewName, viewMeta.GetDefinition())) {
				execOmniQuerySpanDDL(cat, createTableDDL(schemaName, viewName, viewMeta.GetColumns()))
			}
		}
		for _, viewName := range schemaMeta.ListMaterializedViewNames() {
			viewMeta := schemaMeta.GetMaterializedView(viewName)
			if viewMeta == nil {
				continue
			}
			cat.SetSearchPath([]string{schemaName, "public"})
			if viewMeta.GetDefinition() == "" || !execOmniQuerySpanDDL(cat, createQuerySpanViewDDL("MATERIALIZED VIEW", schemaName, viewName, viewMeta.GetDefinition())) {
				execOmniQuerySpanDDL(cat, createViewDDL("MATERIALIZED VIEW", schemaName, viewName, nil))
			}
		}
		for _, sequenceName := range schemaMeta.ListSequenceNames() {
			execOmniQuerySpanDDL(cat, fmt.Sprintf("CREATE SEQUENCE %s.%s;", quoteIdent(schemaName), quoteIdent(sequenceName)))
		}
	}

	cat.SetSearchPath(q.searchPath)
	return cat, nil
}

func execOmniQuerySpanDDL(cat *redshiftcatalog.Catalog, sql string) bool {
	_, err := cat.Exec(sql, nil)
	return err == nil
}

func createQuerySpanViewDDL(kind, schemaName, viewName, definition string) string {
	definition = strings.TrimSpace(definition)
	definition = strings.TrimSuffix(definition, ";")
	if strings.HasPrefix(strings.ToUpper(definition), "CREATE ") {
		return definition + ";"
	}
	return fmt.Sprintf(
		"CREATE %s %s.%s AS %s;",
		kind,
		quoteIdent(schemaName),
		quoteIdent(viewName),
		definition,
	)
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
		return base.ColumnResource{}, errors.Wrapf(err, "failed to get database metadata for database: %s", resource.Database)
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
	case *redshiftast.RedshiftShowStmt, *redshiftast.VariableShowStmt:
		return base.SelectInfoSchema
	case *redshiftast.VariableSetStmt:
		return base.Select
	case *redshiftast.InsertStmt, *redshiftast.UpdateStmt, *redshiftast.DeleteStmt, *redshiftast.MergeStmt,
		*redshiftast.CopyStmt, *redshiftast.RefreshMatViewStmt, *redshiftast.CallStmt:
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
