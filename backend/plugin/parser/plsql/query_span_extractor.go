package plsql

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"unicode"

	oracleast "github.com/bytebase/omni/oracle/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type querySpanExtractor struct {
	ctx             context.Context
	gCtx            base.GetQuerySpanContext
	defaultDatabase string

	ctes []*base.PseudoTable

	outerTableSources []base.TableSource
	tableSourcesFrom  []base.TableSource
	linkedDatabases   map[string]linkedDatabaseResolution

	// source is the original SQL text, used to slice expression names via Loc.
	source                   string
	topLevelTableSourcesFrom []base.TableSource
}

func newQuerySpanExtractor(connectionDatabase string, gCtx base.GetQuerySpanContext) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase: connectionDatabase,
		gCtx:            gCtx,
		linkedDatabases: make(map[string]linkedDatabaseResolution),
	}
}

// linkedDatabaseResolution is what a database link resolved to; a nil meta means it
// resolved to nothing.
type linkedDatabaseResolution struct {
	instanceID   string
	databaseName string
	meta         *model.DatabaseMetadata
}

// getLinkedDatabaseMetadata resolves a database link once per (link, schema) for the
// statement; the access-table pass and the column pass both ask for the same links.
func (q *querySpanExtractor) getLinkedDatabaseMetadata(linkName string, schema string) (string, string, *model.DatabaseMetadata, error) {
	if q.gCtx.GetLinkedDatabaseMetadataFunc == nil {
		return "", "", nil, nil
	}
	key := linkName + "\x00" + schema
	if r, ok := q.linkedDatabases[key]; ok {
		return r.instanceID, r.databaseName, r.meta, nil
	}
	linkedInstanceID, databaseName, meta, err := q.gCtx.GetLinkedDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, linkName, schema)
	if err != nil {
		return "", "", nil, errors.Wrapf(err, "failed to get linked database metadata for schema: %s", schema)
	}
	q.linkedDatabases[key] = linkedDatabaseResolution{instanceID: linkedInstanceID, databaseName: databaseName, meta: meta}
	return linkedInstanceID, databaseName, meta, nil
}

func (q *querySpanExtractor) getDatabaseMetadata(schema string) (*model.DatabaseMetadata, error) {
	_, meta, err := q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, schema)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for schema: %s", schema)
	}
	return meta, nil
}

func (q *querySpanExtractor) existsTableMetadata(resource base.SchemaResource) bool {
	database := resource.Database
	if database == "" {
		database = q.defaultDatabase
	}
	meta, err := q.getDatabaseMetadata(database)
	if err != nil || meta == nil {
		return false
	}
	schema := meta.GetSchemaMetadata("")
	if schema == nil {
		return false
	}

	return schema.GetTable(resource.Table) != nil ||
		schema.GetView(resource.Table) != nil ||
		schema.GetMaterializedView(resource.Table) != nil ||
		schema.GetExternalTable(resource.Table) != nil
}

// isMixedQuery classifies the tables a statement reads. A linked table is a user table:
// a remote schema name says nothing about the connected instance's system schemas, and an
// info-schema verdict would skip the per-table check for it.
func isMixedQuery(local, linked []base.SchemaResource) (allSystem, mixed bool) {
	hasSystem, hasUser := false, len(linked) > 0
	for _, item := range local {
		if systemSchemaMap[item.Database] {
			hasSystem = true
		} else {
			hasUser = true
		}
	}

	if hasSystem && hasUser {
		return false, true
	}

	return !hasUser && hasSystem, false
}

var systemSchemaMap = map[string]bool{
	"ANONYMOUS":              true,
	"APPQOSSYS":              true,
	"AUDSYS":                 true,
	"CTXSYS":                 true,
	"DBSFWUSER":              true,
	"DBSNMP":                 true,
	"DGPDB_INT":              true,
	"DIP":                    true,
	"DVF":                    true,
	"DVSYS":                  true,
	"GGSYS":                  true,
	"GSMADMIN_INTERNAL":      true,
	"GSMCATUSER":             true,
	"GSMROOTUSER":            true,
	"GSMUSER":                true,
	"LBACSYS":                true,
	"MDDATA":                 true,
	"MDSYS":                  true,
	"OPS$ORACLE":             true,
	"ORACLE_OCM":             true,
	"OUTLN":                  true,
	"REMOTE_SCHEDULER_AGENT": true,
	"SYS":                    true,
	"SYS$UMF":                true,
	"SYSBACKUP":              true,
	"SYSDG":                  true,
	"SYSKM":                  true,
	"SYSRAC":                 true,
	"SYSTEM":                 true,
	"XDB":                    true,
	"XS$NULL":                true,
	"XS$$NULL":               true,
	"FLOWS_FILES":            true,
	"HR":                     true,
	"EXFSYS":                 true,
	"MGMT_VIEW":              true,
	"OLAPSYS":                true,
	"ORDDATA":                true,
	"ORDPLUGINS":             true,
	"ORDSYS":                 true,
	"OWBSYS":                 true,
	"OWBSYS_AUDIT":           true,
	"SCOTT":                  true,
	"SI_INFORMTN_SCHEMA":     true,
	"SPATIAL_CSW_ADMIN_USR":  true,
	"SPATIAL_WFS_ADMIN_USR":  true,
	"SYSMAN":                 true,
	"WMSYS":                  true,
	"OJVMSYS":                true,
}

func (q *querySpanExtractor) plsqlFindTableSchema(dbLink []string, schemaName, tableName string) (base.TableSource, error) {
	if len(dbLink) > 0 {
		linkName := strings.Join(dbLink, ".")
		linkedInstanceID, linkedDatabaseName, linkedMeta, err := q.getLinkedDatabaseMetadata(linkName, schemaName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get linked database metadata for: %s", dbLink)
		}
		if linkedMeta == nil {
			return nil, &base.ResourceNotFoundError{
				DatabaseLink: &linkName,
			}
		}
		// Lineage carries the resolved database, as the access set does, so the masker
		// finds the policy of the database the link reaches rather than of "" for `t@link`.
		return q.findTableSchemaInMetadata(linkedInstanceID, linkedMeta, linkedDatabaseName, tableName)
	}

	if schemaName == q.defaultDatabase {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			table := q.ctes[i]
			if table.Name == tableName {
				return table, nil
			}
		}
	}

	dbMetadata, err := q.getDatabaseMetadata(schemaName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for: %s", schemaName)
	}
	if dbMetadata == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &schemaName,
		}
	}

	return q.findTableSchemaInMetadata(q.gCtx.InstanceID, dbMetadata, schemaName, tableName)
}

func (q *querySpanExtractor) findTableSchemaInMetadata(instanceID string, dbMetadata *model.DatabaseMetadata, databaseName, tableName string) (base.TableSource, error) {
	schema := dbMetadata.GetSchemaMetadata("")
	if schema == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &databaseName,
		}
	}
	table := schema.GetTable(tableName)
	view := schema.GetView(tableName)
	materializedView := schema.GetMaterializedView(tableName)
	foreignTable := schema.GetExternalTable(tableName)
	if table == nil && view == nil && materializedView == nil && foreignTable == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &databaseName,
			Table:    &tableName,
		}
	}

	if table != nil {
		var columns []string
		invisibleColumns := make(map[string]bool)
		for _, column := range table.GetProto().GetColumns() {
			columns = append(columns, column.Name)
			if column.IsInvisible {
				invisibleColumns[column.Name] = true
			}
		}
		return &base.PhysicalTable{
			Server:           "",
			Database:         databaseName,
			Name:             table.GetProto().Name,
			Columns:          columns,
			InvisibleColumns: invisibleColumns,
		}, nil
	}

	if foreignTable != nil {
		var columns []string
		for _, column := range foreignTable.GetProto().GetColumns() {
			columns = append(columns, column.Name)
		}
		return &base.PhysicalTable{
			Server:   "",
			Database: databaseName,
			Name:     foreignTable.GetProto().Name,
			Columns:  columns,
		}, nil
	}

	if view != nil && view.Definition != "" {
		columns, err := q.getColumnsForView(instanceID, databaseName, view.Definition)
		if err != nil {
			return nil, err
		}
		return &base.PseudoTable{
			Name:    view.Name,
			Columns: columns,
		}, nil
	}

	if materializedView != nil && materializedView.Definition != "" {
		columns, err := q.getColumnsForMaterializedView(instanceID, databaseName, materializedView.Definition)
		if err != nil {
			return nil, err
		}
		return &base.PseudoTable{
			Name:    materializedView.Name,
			Columns: columns,
		}, nil
	}
	return nil, nil
}

func (q *querySpanExtractor) getColumnsForView(instanceID, defaultDatabase, definition string) ([]base.QuerySpanResult, error) {
	newContext := base.GetQuerySpanContext{
		InstanceID:                    instanceID,
		GetDatabaseMetadataFunc:       q.gCtx.GetDatabaseMetadataFunc,
		ListDatabaseNamesFunc:         q.gCtx.ListDatabaseNamesFunc,
		GetLinkedDatabaseMetadataFunc: q.gCtx.GetLinkedDatabaseMetadataFunc,
	}
	newQ := newQuerySpanExtractor(defaultDatabase, newContext)
	span, err := newQ.getQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span for view definition: %s", definition)
	}
	if span.NotFoundError != nil {
		return nil, span.NotFoundError
	}
	return span.Results, nil
}

func (q *querySpanExtractor) getColumnsForMaterializedView(instanceID, defaultDatabase, definition string) ([]base.QuerySpanResult, error) {
	newContext := base.GetQuerySpanContext{
		InstanceID:                    instanceID,
		GetDatabaseMetadataFunc:       q.gCtx.GetDatabaseMetadataFunc,
		ListDatabaseNamesFunc:         q.gCtx.ListDatabaseNamesFunc,
		GetLinkedDatabaseMetadataFunc: q.gCtx.GetLinkedDatabaseMetadataFunc,
	}
	newQ := newQuerySpanExtractor(defaultDatabase, newContext)
	span, err := newQ.getQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span for materialized view definition: %s", definition)
	}
	if span.NotFoundError != nil {
		return nil, span.NotFoundError
	}
	return span.Results, nil
}

func (q *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx
	q.source = statement

	list, err := ParsePLSQL(statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse statement: %s", statement)
	}
	if list == nil || len(list.Items) == 0 {
		return nil, nil
	}
	if len(list.Items) > 1 {
		return nil, errors.Errorf("query span extraction only supports single statement, got %d statements", len(list.Items))
	}
	raw, ok := list.Items[0].(*oracleast.RawStmt)
	if !ok || raw.Stmt == nil {
		return nil, nil
	}

	accessTables, linkedTables := collectAccessTables(q.defaultDatabase, raw.Stmt)
	allSystem, mixed := isMixedQuery(accessTables, linkedTables)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	queryType := queryType(raw.Stmt, allSystem)
	if queryType != base.Select {
		sourceColumns := make(base.SourceColumnSet)
		for _, resource := range accessTables {
			sourceColumns[base.ColumnResource{Database: resource.Database, Table: resource.Table}] = true
		}
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: sourceColumns,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	columnSet := make(base.SourceColumnSet)
	for _, resource := range accessTables {
		if !q.existsTableMetadata(resource) {
			continue
		}
		columnSet[base.ColumnResource{
			Database: resource.Database,
			Table:    resource.Table,
		}] = true
	}
	for _, resource := range linkedTables {
		column, err := q.linkedAccessColumn(resource)
		if err != nil {
			return nil, err
		}
		columnSet[column] = true
	}
	selectStmt, ok := raw.Stmt.(*oracleast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("expected SELECT statement, got %T", raw.Stmt)
	}
	tableSource, err := q.extractSelect(selectStmt)
	if err != nil {
		var resourceNotFound *base.ResourceNotFoundError
		if errors.As(err, &resourceNotFound) {
			if len(columnSet) == 0 {
				columnSet[base.ColumnResource{
					Database: q.defaultDatabase,
				}] = true
			}
			return &base.QuerySpan{
				Type:          base.Select,
				SourceColumns: columnSet,
				Results:       []base.QuerySpanResult{},
				NotFoundError: resourceNotFound,
			}, nil
		}
		return nil, err
	}
	var results []base.QuerySpanResult
	if tableSource != nil {
		results = tableSource.GetQuerySpanResult()
	}
	if resourceNotFound := q.findMissingAccessTable(raw.Stmt, accessTables); resourceNotFound != nil {
		if len(columnSet) == 0 {
			columnSet[base.ColumnResource{
				Database: q.defaultDatabase,
			}] = true
		}
		return &base.QuerySpan{
			Type:          base.Select,
			SourceColumns: columnSet,
			Results:       []base.QuerySpanResult{},
			NotFoundError: resourceNotFound,
		}, nil
	}
	return &base.QuerySpan{
		Type:          base.Select,
		SourceColumns: columnSet,
		Results:       results,
	}, nil
}

func queryType(stmt oracleast.StmtNode, allSystem bool) base.QueryType {
	if stmt == nil {
		return base.QueryTypeUnknown
	}
	switch stmt.(type) {
	case *oracleast.ExplainPlanStmt:
		return base.Explain
	case *oracleast.SelectStmt:
		if allSystem {
			return base.SelectInfoSchema
		}
		return base.Select
	case *oracleast.InsertStmt, *oracleast.UpdateStmt, *oracleast.DeleteStmt, *oracleast.MergeStmt,
		*oracleast.CallStmt, *oracleast.LockTableStmt:
		return base.DML
	case *oracleast.CreateTableStmt, *oracleast.AlterTableStmt, *oracleast.DropStmt,
		*oracleast.CreateIndexStmt, *oracleast.CreateViewStmt,
		*oracleast.CreateSequenceStmt, *oracleast.CreateSynonymStmt,
		*oracleast.CreateDatabaseLinkStmt, *oracleast.CreateTypeStmt,
		*oracleast.CreatePackageStmt, *oracleast.CreateProcedureStmt,
		*oracleast.CreateFunctionStmt, *oracleast.CreateTriggerStmt,
		*oracleast.TruncateStmt, *oracleast.AlterSessionStmt,
		*oracleast.AlterSystemStmt, *oracleast.CreateUserStmt,
		*oracleast.AlterUserStmt, *oracleast.CreateRoleStmt,
		*oracleast.AlterRoleStmt, *oracleast.CreateProfileStmt,
		*oracleast.AlterProfileStmt, *oracleast.AlterResourceCostStmt,
		*oracleast.AdminDDLStmt, *oracleast.CreateSchemaStmt,
		*oracleast.AlterDatabaseLinkStmt, *oracleast.AlterSynonymStmt,
		*oracleast.AlterMaterializedViewStmt,
		*oracleast.CreateAuditPolicyStmt, *oracleast.AlterAuditPolicyStmt,
		*oracleast.DropAuditPolicyStmt, *oracleast.CreateTablespaceStmt,
		*oracleast.AlterTablespaceStmt, *oracleast.CreateTablespaceSetStmt,
		*oracleast.DropTablespaceStmt, *oracleast.CreateClusterStmt,
		*oracleast.CreateDimensionStmt, *oracleast.AlterClusterStmt,
		*oracleast.AlterDimensionStmt, *oracleast.CreateMaterializedZonemapStmt,
		*oracleast.AlterMaterializedZonemapStmt, *oracleast.CreateInmemoryJoinGroupStmt,
		*oracleast.AlterInmemoryJoinGroupStmt, *oracleast.AlterIndexStmt,
		*oracleast.AlterViewStmt, *oracleast.AlterSequenceStmt,
		*oracleast.AlterProcedureStmt, *oracleast.AlterFunctionStmt,
		*oracleast.AlterPackageStmt, *oracleast.AlterTriggerStmt,
		*oracleast.AlterTypeStmt, *oracleast.CreateIndextypeStmt,
		*oracleast.AlterIndextypeStmt, *oracleast.CreateOperatorStmt,
		*oracleast.AlterOperatorStmt, *oracleast.CreateMviewLogStmt,
		*oracleast.AlterMviewLogStmt, *oracleast.CreateAnalyticViewStmt,
		*oracleast.AlterAnalyticViewStmt, *oracleast.CreateJsonDualityViewStmt,
		*oracleast.AlterJsonDualityViewStmt, *oracleast.CreateAttributeDimensionStmt,
		*oracleast.AlterAttributeDimensionStmt, *oracleast.CreateHierarchyStmt,
		*oracleast.AlterHierarchyStmt, *oracleast.CreateDomainStmt,
		*oracleast.AlterDomainStmt, *oracleast.CreatePropertyGraphStmt,
		*oracleast.CreateVectorIndexStmt, *oracleast.CreateLockdownProfileStmt,
		*oracleast.AlterLockdownProfileStmt, *oracleast.CreateOutlineStmt,
		*oracleast.AlterOutlineStmt:
		return base.DDL
	}
	return base.QueryTypeUnknown
}

// linkedAccessColumn authorizes the name as written, not looked up in connected-instance
// metadata, which would drop a remote table or keep a same-named local one as the wrong
// resource. A link that resolves to nothing yields Server set and Instance empty.
func (q *querySpanExtractor) linkedAccessColumn(resource base.SchemaResource) (base.ColumnResource, error) {
	column := base.ColumnResource{
		Server:   resource.LinkedServer,
		Database: resource.Database,
		Table:    resource.Table,
	}
	linkedInstanceID, linkedDatabaseName, linkedMeta, err := q.getLinkedDatabaseMetadata(resource.LinkedServer, resource.Database)
	if err != nil {
		return base.ColumnResource{}, err
	}
	if linkedMeta == nil || linkedInstanceID == "" || linkedDatabaseName == "" {
		return column, nil
	}
	column.Instance = linkedInstanceID
	column.Database = linkedDatabaseName
	return column, nil
}

func (q *querySpanExtractor) findMissingAccessTable(stmt oracleast.StmtNode, accessTables []base.SchemaResource) *base.ResourceNotFoundError {
	cteNames := collectCTENames(stmt)
	for _, resource := range accessTables {
		if cteNames[resource.Table] || q.existsTableMetadata(resource) {
			continue
		}
		database := resource.Database
		if database == "" {
			database = q.defaultDatabase
		}
		table := resource.Table
		return &base.ResourceNotFoundError{
			Database: &database,
			Table:    &table,
		}
	}
	return nil
}

func collectCTENames(stmt oracleast.StmtNode) map[string]bool {
	result := make(map[string]bool)
	oracleast.Inspect(stmt, func(node oracleast.Node) bool {
		cte, ok := node.(*oracleast.CTE)
		if ok {
			result[cte.Name] = true
		}
		return true
	})
	return result
}

// isDual reports whether a name is Oracle's DUAL pseudo-table: SYS.DUAL, "PUBLIC".DUAL,
// or an unqualified local DUAL. A schema may own a real table named DUAL, which a qualified
// name resolves to; an unqualified name through a link resolves in the link user's schema
// first, so it is a table there. The unqualified local case assumes the session schema
// owns no DUAL table, as the extractor always has.
func isDual(name *oracleast.ObjectName) bool {
	if name.Name != "DUAL" {
		return false
	}
	return (name.Schema == "" && name.DBLink == "") || name.Schema == "SYS" || name.Schema == "PUBLIC"
}

func dualTable() base.TableSource {
	return &base.PseudoTable{
		Name:    "DUAL",
		Columns: []base.QuerySpanResult{},
	}
}

// tableLink is the database link a table reference names, as Oracle identifies it. A
// connection qualifier is part of the name: `t@link@q` reaches the link LINK@Q, which omni
// splits between ObjectName.DBLink and TableRef.Dblink. TableRef.Dblink alone names no
// link: omni fills it for a link after a partition clause, which Oracle rejects (ORA-03048).
func tableLink(ref *oracleast.TableRef) string {
	if ref.Name == nil {
		return ""
	}
	if ref.Name.DBLink != "" && ref.Dblink != "" {
		return ref.Name.DBLink + "@" + ref.Dblink
	}
	return ref.Name.DBLink
}

// A linked table keeps the schema as written; an unqualified one lives in the link user's
// schema, which the resolver supplies.
func collectAccessTables(defaultDatabase string, stmt oracleast.StmtNode) (local, linked []base.SchemaResource) {
	seen := make(map[base.SchemaResource]bool)
	addResource := func(name *oracleast.ObjectName, link string) {
		if name == nil || isDual(name) {
			return
		}
		database := name.Schema
		if database == "" && link == "" {
			database = defaultDatabase
		}
		resource := base.SchemaResource{
			Database:     database,
			Table:        name.Name,
			LinkedServer: link,
		}
		if seen[resource] {
			return
		}
		seen[resource] = true
		if link != "" {
			linked = append(linked, resource)
			return
		}
		local = append(local, resource)
	}
	addTarget := func(name *oracleast.ObjectName) {
		if name != nil {
			addResource(name, name.DBLink)
		}
	}
	oracleast.Inspect(stmt, func(node oracleast.Node) bool {
		switch node := node.(type) {
		case *oracleast.TableRef:
			addResource(node.Name, tableLink(node))
		case *oracleast.ContainersExpr:
			addTarget(node.Name)
		// A DML target is an ObjectName, not a TableRef, so it is named here.
		case *oracleast.InsertStmt:
			addTarget(node.Table)
		case *oracleast.InsertIntoClause:
			addTarget(node.Table)
		case *oracleast.UpdateStmt:
			addTarget(node.Table)
		case *oracleast.DeleteStmt:
			addTarget(node.Table)
		case *oracleast.MergeStmt:
			addTarget(node.Target)
		default:
		}
		return true
	})
	return local, linked
}

func (q *querySpanExtractor) clone() *querySpanExtractor {
	copyExtractor := *q
	copyExtractor.topLevelTableSourcesFrom = cloneTableSourceSlice(q.topLevelTableSourcesFrom)
	return &copyExtractor
}

func (q *querySpanExtractor) extractSelect(stmt *oracleast.SelectStmt) (base.TableSource, error) {
	if stmt == nil {
		return nil, nil
	}

	oldCTEs := q.ctes
	if stmt.WithClause != nil {
		for _, node := range listItems(stmt.WithClause.CTEs) {
			cte, ok := node.(*oracleast.CTE)
			if !ok {
				continue
			}
			table, err := q.extractCTE(cte)
			if err != nil {
				q.ctes = oldCTEs
				return nil, err
			}
			q.ctes = append(q.ctes, table)
		}
	}
	defer func() {
		q.ctes = oldCTEs
	}()

	if stmt.Op != 0 {
		return q.extractSetSelect(stmt)
	}
	// omni wraps a parenthesized query that ORDER BY or a row limit follows,
	// or a WITH clause precedes; the wrapper has no select list of its own.
	if stmt.Larg != nil {
		return q.extractSelect(stmt.Larg)
	}
	if stmt.ModelClause != nil {
		source, err := q.extractModelSelect(stmt)
		if err != nil {
			return nil, err
		}
		return q.projectTransformedSelect(stmt, source)
	}

	oldFrom := q.tableSourcesFrom
	oldTopLevelFrom := q.topLevelTableSourcesFrom
	q.tableSourcesFrom = nil
	q.topLevelTableSourcesFrom = nil
	defer func() {
		q.tableSourcesFrom = oldFrom
		q.topLevelTableSourcesFrom = oldTopLevelFrom
	}()

	for _, node := range listItems(stmt.FromClause) {
		tableExpr, ok := node.(oracleast.TableExpr)
		if !ok {
			continue
		}
		tableSource, err := q.extractTableExpr(tableExpr)
		if err != nil {
			return nil, err
		}
		if tableSource != nil {
			q.tableSourcesFrom = append(q.tableSourcesFrom, tableSource)
			q.topLevelTableSourcesFrom = append(q.topLevelTableSourcesFrom, tableSource)
		}
	}

	results, err := q.extractTargetList(stmt.TargetList)
	if err != nil {
		return nil, err
	}
	return &base.PseudoTable{
		Columns: results,
	}, nil
}

func (q *querySpanExtractor) projectTransformedSelect(stmt *oracleast.SelectStmt, source base.TableSource) (base.TableSource, error) {
	oldFrom := q.tableSourcesFrom
	oldTopLevelFrom := q.topLevelTableSourcesFrom
	q.tableSourcesFrom = nil
	q.topLevelTableSourcesFrom = nil
	if source != nil {
		q.tableSourcesFrom = append(q.tableSourcesFrom, source)
		q.topLevelTableSourcesFrom = append(q.topLevelTableSourcesFrom, source)
	}
	defer func() {
		q.tableSourcesFrom = oldFrom
		q.topLevelTableSourcesFrom = oldTopLevelFrom
	}()

	results, err := q.extractTargetList(stmt.TargetList)
	if err != nil {
		return nil, err
	}
	return &base.PseudoTable{Columns: results}, nil
}

func (q *querySpanExtractor) extractSetSelect(stmt *oracleast.SelectStmt) (base.TableSource, error) {
	left, err := q.extractSelect(setLeftSelect(stmt))
	if err != nil {
		return nil, err
	}
	right, err := q.extractSelect(stmt.Rarg)
	if err != nil {
		return nil, err
	}
	return mergeSetTableSources(left, right)
}

func mergeSetTableSources(left, right base.TableSource) (base.TableSource, error) {
	if left == nil {
		return right, nil
	}
	if right == nil {
		return left, nil
	}
	leftResults := left.GetQuerySpanResult()
	rightResults := right.GetQuerySpanResult()
	if len(leftResults) != len(rightResults) {
		return nil, errors.Errorf("left and right query span result length mismatch: %d != %d", len(leftResults), len(rightResults))
	}
	result := make([]base.QuerySpanResult, 0, len(leftResults))
	for i, leftResult := range leftResults {
		sourceColumns, _ := base.MergeSourceColumnSet(leftResult.SourceColumns, rightResults[i].SourceColumns)
		result = append(result, base.QuerySpanResult{
			Name:          leftResult.Name,
			SourceColumns: sourceColumns,
			IsPlainField:  false,
		})
	}
	return &base.PseudoTable{Columns: result}, nil
}

func (q *querySpanExtractor) extractCTE(cte *oracleast.CTE) (*base.PseudoTable, error) {
	if cte == nil {
		return nil, nil
	}
	name := cte.Name
	columnNames := stringList(cte.Columns)

	selectStmt, ok := cte.Query.(*oracleast.SelectStmt)
	if ok && selectStmt.Op != 0 && len(columnNames) > 0 {
		initialSource, err := q.extractSelect(setLeftSelect(selectStmt))
		if err != nil {
			return nil, err
		}
		initial := cloneQuerySpanResults(initialSource.GetQuerySpanResult())
		applyColumnAliases(initial, columnNames)

		placeholder := &base.PseudoTable{Name: name, Columns: initial}
		q.ctes = append(q.ctes, placeholder)
		defer func() {
			q.ctes = q.ctes[:len(q.ctes)-1]
		}()

		columns := initial
		for range 16 {
			placeholder.Columns = columns
			rightSource, err := q.extractSelect(selectStmt.Rarg)
			if err != nil {
				return nil, err
			}
			mergedSource, err := mergeSetTableSources(&base.PseudoTable{Columns: initial}, rightSource)
			if err != nil {
				return nil, err
			}
			next := cloneQuerySpanResults(mergedSource.GetQuerySpanResult())
			applyColumnAliases(next, columnNames)
			if querySpanResultsEqual(columns, next) {
				return &base.PseudoTable{Name: name, Columns: next}, nil
			}
			columns = next
		}
		return &base.PseudoTable{Name: name, Columns: columns}, nil
	}

	child := q.clone()
	tableSource, err := child.extractStmt(cte.Query)
	if err != nil {
		return nil, err
	}
	columns := cloneQuerySpanResults(tableSource.GetQuerySpanResult())
	applyColumnAliases(columns, columnNames)
	return &base.PseudoTable{Name: name, Columns: columns}, nil
}

func (q *querySpanExtractor) extractStmt(stmt oracleast.StmtNode) (base.TableSource, error) {
	selectStmt, ok := stmt.(*oracleast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("unsupported statement in query span extractor: %T", stmt)
	}
	return q.extractSelect(selectStmt)
}

func (q *querySpanExtractor) extractTargetList(list *oracleast.List) ([]base.QuerySpanResult, error) {
	if list == nil || list.Len() == 0 {
		return q.expandAsterisk("", "")
	}

	var results []base.QuerySpanResult
	for _, node := range listItems(list) {
		target, ok := node.(*oracleast.ResTarget)
		if !ok || target.Expr == nil {
			continue
		}
		if isStar(target.Expr) {
			expanded, err := q.expandAsterisk("", "")
			if err != nil {
				return nil, err
			}
			results = append(results, expanded...)
			continue
		}
		if ref, ok := target.Expr.(*oracleast.ColumnRef); ok && ref.Column == "*" {
			expanded, err := q.expandAsterisk(ref.Schema, ref.Table)
			if err != nil {
				return nil, err
			}
			results = append(results, expanded...)
			continue
		}

		name, sourceColumns, err := q.extractExpr(target.Expr)
		if err != nil {
			return nil, err
		}
		if target.Name != "" {
			name = target.Name
		}
		results = append(results, base.QuerySpanResult{
			Name:          name,
			SourceColumns: sourceColumns,
			IsPlainField:  false,
		})
	}
	return results, nil
}

func (q *querySpanExtractor) extractTableExpr(expr oracleast.TableExpr) (base.TableSource, error) {
	switch expr := expr.(type) {
	case *oracleast.TableRef:
		if expr.Name == nil {
			return nil, nil
		}
		if isDual(expr.Name) {
			return aliasTableSource(dualTable(), expr.Alias), nil
		}
		dbLink := tableLink(expr)
		database := expr.Name.Schema
		if database == "" && dbLink == "" {
			database = q.defaultDatabase
		}
		tableSource, err := q.plsqlFindTableSchema(splitDBLink(dbLink), database, expr.Name.Name)
		if err != nil {
			return nil, err
		}
		return aliasTableSource(tableSource, expr.Alias), nil
	case *oracleast.SubqueryRef:
		child := q.clone()
		tableSource, err := child.extractStmt(expr.Subquery)
		if err != nil {
			return nil, err
		}
		return aliasTableSource(tableSource, expr.Alias), nil
	case *oracleast.SubqueryExpr:
		child := q.clone()
		tableSource, err := child.extractStmt(expr.Subquery)
		if err != nil {
			return nil, err
		}
		return tableSource, nil
	case *oracleast.LateralRef:
		child := q.clone()
		child.outerTableSources = append(cloneTableSourceSlice(child.outerTableSources), q.tableSourcesFrom...)
		tableSource, err := child.extractStmt(expr.Subquery)
		if err != nil {
			return nil, err
		}
		return aliasTableSource(tableSource, expr.Alias), nil
	case *oracleast.JoinClause:
		left, err := q.extractTableExpr(expr.Left)
		if err != nil {
			return nil, err
		}
		if left != nil {
			q.tableSourcesFrom = append(q.tableSourcesFrom, left)
		}
		right, err := q.extractTableExpr(expr.Right)
		if err != nil {
			return nil, err
		}
		if right != nil {
			q.tableSourcesFrom = append(q.tableSourcesFrom, right)
		}
		return mergeJoinTableSource(expr, left, right)
	case *oracleast.XmlTableRef:
		return q.extractXMLTable(expr)
	case *oracleast.JsonTableRef:
		return q.extractJSONTable(expr)
	case *oracleast.ContainersExpr:
		if expr.Name == nil {
			return nil, nil
		}
		if isDual(expr.Name) {
			return aliasTableSource(dualTable(), expr.Alias), nil
		}
		database := expr.Name.Schema
		if database == "" {
			database = q.defaultDatabase
		}
		tableSource, err := q.plsqlFindTableSchema(nil, database, expr.Name.Name)
		if err != nil {
			return nil, err
		}
		return aliasTableSource(tableSource, expr.Alias), nil
	case *oracleast.InlineExternalTable:
		return extractInlineExternalTable(expr), nil
	case *oracleast.TableCollectionExpr:
		return q.extractTableCollection(expr)
	case *oracleast.PivotClause:
		return q.extractPivot(expr)
	case *oracleast.UnpivotClause:
		return q.extractUnpivot(expr)
	case *oracleast.MatchRecognizeClause:
		return q.extractMatchRecognize(expr)
	default:
		return nil, errors.Errorf("unsupported oracle table source: %T", expr)
	}
}

func (q *querySpanExtractor) extractPivot(pivot *oracleast.PivotClause) (base.TableSource, error) {
	if pivot == nil {
		return nil, nil
	}
	// Capture the FROM scope before extracting the source: a join source
	// appends its operands to tableSourcesFrom, and they must not leak past
	// this transform — the clause replaces them with its own output.
	oldFrom := q.tableSourcesFrom
	defer func() {
		q.tableSourcesFrom = oldFrom
	}()
	source, err := q.extractTableExpr(pivot.Source)
	if err != nil {
		return nil, err
	}
	// While deriving the transform's output, the scope is exactly the
	// extracted source: the clause's inputs resolve against it alone, never
	// against enclosing join operands or the source's own leaked operands
	// (a USING-merged column must attribute to both sides, not to the first
	// operand in the slice).
	q.tableSourcesFrom = nil
	if source != nil {
		q.tableSourcesFrom = []base.TableSource{source}
	}

	excluded := make(map[string]bool)
	pivotSources := make(base.SourceColumnSet)
	for _, expr := range exprList(pivot.ForColumns) {
		name, sourceColumns, err := q.extractExpr(expr)
		if err != nil {
			return nil, err
		}
		excluded[name] = true
		pivotSources, _ = base.MergeSourceColumnSet(pivotSources, sourceColumns)
	}

	var aggregates []*oracleast.PivotAggregate
	for _, item := range listItems(pivot.Aggregates) {
		aggregate, ok := item.(*oracleast.PivotAggregate)
		if ok {
			aggregates = append(aggregates, aggregate)
		}
	}
	var inItems []*oracleast.PivotInItem
	for _, item := range listItems(pivot.InItems) {
		inItem, ok := item.(*oracleast.PivotInItem)
		if ok {
			inItems = append(inItems, inItem)
		}
	}

	aggregateSources := make([]base.SourceColumnSet, len(aggregates))
	for i, aggregate := range aggregates {
		_, sourceColumns, err := q.extractExpr(aggregate.Expr)
		if err != nil {
			return nil, err
		}
		aggregateSources[i] = sourceColumns
		excludeColumnRefNames(excluded, aggregate.Expr)
		excludeSourceColumnNames(excluded, sourceColumns)
	}

	var results []base.QuerySpanResult
	if source != nil {
		for _, result := range source.GetQuerySpanResult() {
			if !excluded[result.Name] {
				results = append(results, cloneQuerySpanResult(result))
			}
		}
	}
	for _, inItem := range inItems {
		inName := pivotInItemName(q, inItem)
		for i, aggregate := range aggregates {
			sourceColumns, _ := base.MergeSourceColumnSet(pivotSources, aggregateSources[i])
			results = append(results, base.QuerySpanResult{
				Name:          pivotColumnName(inName, aggregate.Alias, len(aggregates)),
				SourceColumns: sourceColumns,
				IsPlainField:  false,
			})
		}
	}
	return aliasTableSource(&base.PseudoTable{Columns: results}, pivot.Alias), nil
}

func (q *querySpanExtractor) extractUnpivot(unpivot *oracleast.UnpivotClause) (base.TableSource, error) {
	if unpivot == nil {
		return nil, nil
	}
	// Capture the FROM scope before extracting the source: a join source
	// appends its operands to tableSourcesFrom, and they must not leak past
	// this transform — the clause replaces them with its own output.
	oldFrom := q.tableSourcesFrom
	defer func() {
		q.tableSourcesFrom = oldFrom
	}()
	source, err := q.extractTableExpr(unpivot.Source)
	if err != nil {
		return nil, err
	}
	// While deriving the transform's output, the scope is exactly the
	// extracted source: the clause's inputs resolve against it alone, never
	// against enclosing join operands or the source's own leaked operands
	// (a USING-merged column must attribute to both sides, not to the first
	// operand in the slice).
	q.tableSourcesFrom = nil
	if source != nil {
		q.tableSourcesFrom = []base.TableSource{source}
	}

	valueColumns := exprList(unpivot.ValueColumns)
	inputSources := make([]base.SourceColumnSet, len(valueColumns))
	excluded := make(map[string]bool)
	for _, item := range listItems(unpivot.InputMappings) {
		mapping, ok := item.(*oracleast.UnpivotInItem)
		if !ok {
			continue
		}
		for i, input := range exprList(mapping.InputColumns) {
			name, sourceColumns, err := q.extractExpr(input)
			if err != nil {
				return nil, err
			}
			excluded[name] = true
			if i < len(inputSources) {
				inputSources[i], _ = base.MergeSourceColumnSet(inputSources[i], sourceColumns)
			}
		}
	}

	var results []base.QuerySpanResult
	if source != nil {
		for _, result := range source.GetQuerySpanResult() {
			if !excluded[result.Name] {
				results = append(results, cloneQuerySpanResult(result))
			}
		}
	}
	for i, expr := range valueColumns {
		name, _, err := q.extractExpr(expr)
		if err != nil {
			return nil, err
		}
		results = append(results, base.QuerySpanResult{
			Name:          name,
			SourceColumns: inputSources[i],
			IsPlainField:  false,
		})
	}
	if unpivot.PivotColumn != nil {
		name, _, err := q.extractExpr(unpivot.PivotColumn)
		if err != nil {
			return nil, err
		}
		results = append(results, base.QuerySpanResult{
			Name:          name,
			SourceColumns: base.SourceColumnSet{},
			IsPlainField:  false,
		})
	}
	return aliasTableSource(&base.PseudoTable{Columns: results}, unpivot.Alias), nil
}

func (q *querySpanExtractor) extractModelSelect(stmt *oracleast.SelectStmt) (base.TableSource, error) {
	oldFrom := q.tableSourcesFrom
	oldTopLevelFrom := q.topLevelTableSourcesFrom
	q.tableSourcesFrom = nil
	q.topLevelTableSourcesFrom = nil
	defer func() {
		q.tableSourcesFrom = oldFrom
		q.topLevelTableSourcesFrom = oldTopLevelFrom
	}()

	for _, node := range listItems(stmt.FromClause) {
		tableExpr, ok := node.(oracleast.TableExpr)
		if !ok {
			continue
		}
		tableSource, err := q.extractTableExpr(tableExpr)
		if err != nil {
			return nil, err
		}
		if tableSource != nil {
			q.tableSourcesFrom = append(q.tableSourcesFrom, tableSource)
			q.topLevelTableSourcesFrom = append(q.topLevelTableSourcesFrom, tableSource)
		}
	}

	var results []base.QuerySpanResult
	if stmt.ModelClause != nil && stmt.ModelClause.MainModel != nil && stmt.ModelClause.MainModel.ColumnClauses != nil {
		columns := stmt.ModelClause.MainModel.ColumnClauses
		for _, list := range []*oracleast.List{columns.PartitionBy, columns.DimensionBy, columns.Measures} {
			if list == nil {
				continue
			}
			extracted, err := q.extractTargetList(list)
			if err != nil {
				return nil, err
			}
			results = append(results, extracted...)
		}
	}
	return &base.PseudoTable{Columns: results}, nil
}

func (*querySpanExtractor) extractTableCollection(ref *oracleast.TableCollectionExpr) (base.TableSource, error) {
	if ref == nil {
		return nil, nil
	}
	var columns []base.QuerySpanResult
	for _, name := range stringList(ref.ColumnAliases) {
		columns = append(columns, base.QuerySpanResult{
			Name:          name,
			SourceColumns: base.SourceColumnSet{},
			IsPlainField:  false,
		})
	}
	if len(columns) == 0 {
		return nil, errors.Errorf("unsupported oracle table collection without column aliases")
	}
	return aliasTableSource(&base.PseudoTable{Columns: columns}, ref.Alias), nil
}

func (q *querySpanExtractor) extractMatchRecognize(ref *oracleast.MatchRecognizeClause) (base.TableSource, error) {
	if ref == nil {
		return nil, nil
	}
	// Capture the FROM scope before extracting the source: a join source
	// appends its operands to tableSourcesFrom, and they must not leak past
	// this transform — the clause replaces them with its own output.
	oldFrom := q.tableSourcesFrom
	defer func() {
		q.tableSourcesFrom = oldFrom
	}()
	source, err := q.extractTableExpr(ref.Source)
	if err != nil {
		return nil, err
	}
	// While deriving the transform's output, the scope is exactly the
	// extracted source: the clause's inputs resolve against it alone, never
	// against enclosing join operands or the source's own leaked operands
	// (a USING-merged column must attribute to both sides, not to the first
	// operand in the slice).
	q.tableSourcesFrom = nil
	if source != nil {
		q.tableSourcesFrom = []base.TableSource{source}
	}

	partitionResults, err := q.extractResultList(ref.PartitionBy)
	if err != nil {
		return nil, err
	}
	measureResults, err := q.extractMatchRecognizeMeasureList(ref.Measures)
	if err != nil {
		return nil, err
	}
	var results []base.QuerySpanResult
	if strings.HasPrefix(strings.ToUpper(ref.RowsPerMatch), "ALL ROWS PER MATCH") && source != nil {
		results = cloneQuerySpanResults(source.GetQuerySpanResult())
	} else {
		results = append(results, partitionResults...)
	}
	results = append(results, measureResults...)
	return aliasTableSource(&base.PseudoTable{Columns: results}, ref.Alias), nil
}

func (q *querySpanExtractor) extractResultList(list *oracleast.List) ([]base.QuerySpanResult, error) {
	return q.extractResultListWithSourceColumns(list, nil)
}

func (q *querySpanExtractor) extractMatchRecognizeMeasureList(list *oracleast.List) ([]base.QuerySpanResult, error) {
	return q.extractResultListWithSourceColumns(list, q.extractMatchRecognizeMeasureSourceColumns)
}

func (q *querySpanExtractor) extractResultListWithSourceColumns(list *oracleast.List, sourceColumnsExtractor func(oracleast.ExprNode) (base.SourceColumnSet, error)) ([]base.QuerySpanResult, error) {
	var results []base.QuerySpanResult
	for _, node := range listItems(list) {
		var expr oracleast.ExprNode
		name := ""
		switch node := node.(type) {
		case *oracleast.ResTarget:
			expr = node.Expr
			name = node.Name
		case oracleast.ExprNode:
			expr = node
		default:
			continue
		}
		if expr == nil {
			continue
		}

		extractedName, sourceColumns, err := q.extractExpr(expr)
		if err != nil {
			return nil, err
		}
		if sourceColumnsExtractor != nil {
			sourceColumns, err = sourceColumnsExtractor(expr)
			if err != nil {
				return nil, err
			}
		}
		if name == "" {
			name = extractedName
		}
		results = append(results, base.QuerySpanResult{
			Name:          name,
			SourceColumns: sourceColumns,
			IsPlainField:  false,
		})
	}
	return results, nil
}

func (q *querySpanExtractor) extractMatchRecognizeMeasureSourceColumns(expr oracleast.ExprNode) (base.SourceColumnSet, error) {
	return q.extractExprSourceColumnsWithResolver(expr, func(node *oracleast.ColumnRef) base.SourceColumnSet {
		sourceColumns := q.getFieldColumnSource(node.Schema, node.Table, node.Column)
		if len(sourceColumns) == 0 && node.Schema == "" && node.Table != "" {
			sourceColumns = q.getFieldColumnSource("", "", node.Column)
		}
		return sourceColumns
	})
}

func setLeftSelect(stmt *oracleast.SelectStmt) *oracleast.SelectStmt {
	if stmt == nil {
		return nil
	}
	if stmt.Larg != nil {
		return stmt.Larg
	}
	body := *stmt
	body.Op = 0
	body.SetAll = false
	body.Larg = nil
	body.Rarg = nil
	return &body
}

func mergeJoinTableSource(join *oracleast.JoinClause, left, right base.TableSource) (base.TableSource, error) {
	if left == nil {
		return right, nil
	}
	if right == nil {
		return left, nil
	}
	leftResults := left.GetQuerySpanResult()
	rightResults := right.GetQuerySpanResult()
	result := new(base.PseudoTable)

	leftIndex := make(map[string]int)
	rightIndex := make(map[string]int)
	for i, field := range leftResults {
		leftIndex[field.Name] = i
	}
	for i, field := range rightResults {
		rightIndex[field.Name] = i
	}

	if isNaturalJoin(join.Type) {
		for _, field := range leftResults {
			if rightIdx, ok := rightIndex[field.Name]; ok {
				field.SourceColumns, _ = base.MergeSourceColumnSet(field.SourceColumns, rightResults[rightIdx].SourceColumns)
			}
			result.Columns = append(result.Columns, field)
		}
		for _, field := range rightResults {
			if _, ok := leftIndex[field.Name]; !ok {
				result.Columns = append(result.Columns, field)
			}
		}
		return result, nil
	}

	if len(listItems(join.Using)) != 0 {
		usingMap := make(map[string]bool)
		for _, node := range listItems(join.Using) {
			switch node := node.(type) {
			case *oracleast.String:
				usingMap[node.Str] = true
			case *oracleast.ColumnRef:
				usingMap[node.Column] = true
			default:
			}
		}
		for _, field := range leftResults {
			if usingMap[field.Name] {
				if rightIdx, ok := rightIndex[field.Name]; ok {
					field.SourceColumns, _ = base.MergeSourceColumnSet(field.SourceColumns, rightResults[rightIdx].SourceColumns)
				}
			}
			result.Columns = append(result.Columns, field)
		}
		for _, field := range rightResults {
			if !usingMap[field.Name] {
				result.Columns = append(result.Columns, field)
			}
		}
		return result, nil
	}

	result.Columns = append(result.Columns, leftResults...)
	result.Columns = append(result.Columns, rightResults...)
	return result, nil
}

func isNaturalJoin(joinType oracleast.JoinType) bool {
	switch joinType {
	case oracleast.JOIN_NATURAL_INNER, oracleast.JOIN_NATURAL_LEFT, oracleast.JOIN_NATURAL_RIGHT, oracleast.JOIN_NATURAL_FULL:
		return true
	default:
		return false
	}
}

func (q *querySpanExtractor) extractExpr(expr oracleast.ExprNode) (string, base.SourceColumnSet, error) {
	if expr == nil {
		return "", base.SourceColumnSet{}, nil
	}

	switch expr := expr.(type) {
	case *oracleast.ColumnRef:
		if expr.Column == "*" {
			return "*", base.SourceColumnSet{}, nil
		}
		return expr.Column, q.getFieldColumnSource(expr.Schema, expr.Table, expr.Column), nil
	case *oracleast.Star:
		return "*", base.SourceColumnSet{}, nil
	case *oracleast.NumberLiteral:
		return q.exprName(expr.Loc, expr.Val), base.SourceColumnSet{}, nil
	case *oracleast.StringLiteral:
		return q.exprName(expr.Loc, expr.Val), base.SourceColumnSet{}, nil
	case *oracleast.NullLiteral:
		return q.exprName(expr.Loc, "NULL"), base.SourceColumnSet{}, nil
	case *oracleast.DateTimeLiteral:
		return q.exprName(expr.Loc, expr.TypeName+" "+expr.Val), base.SourceColumnSet{}, nil
	case *oracleast.SubqueryExpr:
		child := q.clone()
		child.outerTableSources = append(cloneTableSourceSlice(q.outerTableSources), q.tableSourcesFrom...)
		tableSource, err := child.extractStmt(expr.Subquery)
		if err != nil {
			return "", nil, err
		}
		return q.exprName(expr.Loc, ""), mergeSourceColumnsFromResults(tableSource.GetQuerySpanResult()), nil
	case *oracleast.ExistsExpr:
		child := q.clone()
		child.outerTableSources = append(cloneTableSourceSlice(q.outerTableSources), q.tableSourcesFrom...)
		tableSource, err := child.extractStmt(expr.Subquery)
		if err != nil {
			return "", nil, err
		}
		return q.exprName(expr.Loc, ""), mergeSourceColumnsFromResults(tableSource.GetQuerySpanResult()), nil
	case *oracleast.CursorExpr:
		child := q.clone()
		child.outerTableSources = append(cloneTableSourceSlice(q.outerTableSources), q.tableSourcesFrom...)
		tableSource, err := child.extractStmt(expr.Subquery)
		if err != nil {
			return "", nil, err
		}
		return q.exprName(expr.Loc, ""), mergeSourceColumnsFromResults(tableSource.GetQuerySpanResult()), nil
	default:
		name := q.exprName(getExprFullLoc(expr), oracleast.NodeToString(expr))
		sourceColumns, err := q.extractExprSourceColumns(expr)
		return name, sourceColumns, err
	}
}

func (q *querySpanExtractor) extractExprSourceColumns(expr oracleast.ExprNode) (base.SourceColumnSet, error) {
	return q.extractExprSourceColumnsWithResolver(expr, func(node *oracleast.ColumnRef) base.SourceColumnSet {
		return q.getFieldColumnSource(node.Schema, node.Table, node.Column)
	})
}

func (q *querySpanExtractor) extractExprSourceColumnsWithResolver(expr oracleast.ExprNode, resolveColumn func(*oracleast.ColumnRef) base.SourceColumnSet) (base.SourceColumnSet, error) {
	result := make(base.SourceColumnSet)
	var walkErr error
	oracleast.Inspect(expr, func(node oracleast.Node) bool {
		if walkErr != nil || node == nil {
			return false
		}
		switch node := node.(type) {
		case *oracleast.ColumnRef:
			if node.Column != "*" {
				result, _ = base.MergeSourceColumnSet(result, resolveColumn(node))
			}
			return false
		case *oracleast.FuncCallExpr:
			return true
		case *oracleast.SubqueryExpr:
			child := q.clone()
			child.outerTableSources = append(cloneTableSourceSlice(q.outerTableSources), q.tableSourcesFrom...)
			tableSource, err := child.extractStmt(node.Subquery)
			if err != nil {
				walkErr = err
				return false
			}
			result, _ = base.MergeSourceColumnSet(result, mergeSourceColumnsFromResults(tableSource.GetQuerySpanResult()))
			return false
		case *oracleast.ExistsExpr:
			child := q.clone()
			child.outerTableSources = append(cloneTableSourceSlice(q.outerTableSources), q.tableSourcesFrom...)
			tableSource, err := child.extractStmt(node.Subquery)
			if err != nil {
				walkErr = err
				return false
			}
			result, _ = base.MergeSourceColumnSet(result, mergeSourceColumnsFromResults(tableSource.GetQuerySpanResult()))
			return false
		case *oracleast.CursorExpr:
			child := q.clone()
			child.outerTableSources = append(cloneTableSourceSlice(q.outerTableSources), q.tableSourcesFrom...)
			tableSource, err := child.extractStmt(node.Subquery)
			if err != nil {
				walkErr = err
				return false
			}
			result, _ = base.MergeSourceColumnSet(result, mergeSourceColumnsFromResults(tableSource.GetQuerySpanResult()))
			return false
		default:
			return true
		}
	})
	return result, walkErr
}

func (q *querySpanExtractor) getFieldColumnSource(schemaName, tableName, columnName string) base.SourceColumnSet {
	findInTableSource := func(tableSource base.TableSource) (base.SourceColumnSet, bool) {
		if schemaName != "" && schemaName != tableSource.GetDatabaseName() {
			return nil, false
		}
		if tableName != "" && tableName != tableSource.GetTableName() {
			return nil, false
		}
		for _, field := range tableSource.GetQuerySpanResult() {
			if field.Name == columnName {
				return field.SourceColumns, true
			}
		}
		return nil, false
	}

	for _, tableSource := range q.tableSourcesFrom {
		if sourceColumnSet, ok := findInTableSource(tableSource); ok {
			return sourceColumnSet
		}
	}
	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		if sourceColumnSet, ok := findInTableSource(q.outerTableSources[i]); ok {
			return sourceColumnSet
		}
	}
	return base.SourceColumnSet{}
}

// expandAsterisk returns the columns * stands for. It leaves out invisible
// columns, as Oracle does; including one would shift every result after it,
// and the masking applied to each.
func (q *querySpanExtractor) expandAsterisk(schemaName, tableName string) ([]base.QuerySpanResult, error) {
	results, err := q.expandAsteriskWithInvisible(schemaName, tableName)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(results, func(result base.QuerySpanResult) bool { return result.Invisible }), nil
}

func (q *querySpanExtractor) expandAsteriskWithInvisible(schemaName, tableName string) ([]base.QuerySpanResult, error) {
	findInTableSource := func(tableSource base.TableSource) ([]base.QuerySpanResult, bool) {
		if schemaName != "" && schemaName != tableSource.GetDatabaseName() {
			return nil, false
		}
		if tableName != "" && tableName != tableSource.GetTableName() {
			return nil, false
		}
		return tableSource.GetQuerySpanResult(), true
	}

	if tableName == "" && schemaName == "" {
		if len(q.topLevelTableSourcesFrom) > 0 {
			return cloneQuerySpanResultsFromTableSources(q.topLevelTableSourcesFrom), nil
		}
		if len(q.tableSourcesFrom) > 0 {
			return cloneQuerySpanResultsFromTableSources(q.tableSourcesFrom), nil
		}
		if len(q.outerTableSources) > 0 {
			return cloneQuerySpanResults(q.outerTableSources[len(q.outerTableSources)-1].GetQuerySpanResult()), nil
		}
		return []base.QuerySpanResult{}, nil
	}

	for i := len(q.tableSourcesFrom) - 1; i >= 0; i-- {
		if results, ok := findInTableSource(q.tableSourcesFrom[i]); ok {
			return cloneQuerySpanResults(results), nil
		}
	}
	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		if results, ok := findInTableSource(q.outerTableSources[i]); ok {
			return cloneQuerySpanResults(results), nil
		}
	}
	return nil, errors.Errorf("failed to resolve asterisk for table %q", tableName)
}

func cloneQuerySpanResultsFromTableSources(tableSources []base.TableSource) []base.QuerySpanResult {
	var result []base.QuerySpanResult
	for _, tableSource := range tableSources {
		if tableSource == nil {
			continue
		}
		result = append(result, cloneQuerySpanResults(tableSource.GetQuerySpanResult())...)
	}
	return result
}

func (q *querySpanExtractor) extractXMLTable(ref *oracleast.XmlTableRef) (base.TableSource, error) {
	_, sourceColumns, err := q.extractExpr(ref.Passing)
	if err != nil {
		return nil, err
	}
	var columns []base.QuerySpanResult
	for _, node := range listItems(ref.Columns) {
		column, ok := node.(*oracleast.XmlTableColumn)
		if !ok {
			continue
		}
		columnSource := base.SourceColumnSet{}
		if !column.ForOrdinality {
			columnSource = sourceColumns
		}
		columns = append(columns, base.QuerySpanResult{
			Name:          column.Name,
			SourceColumns: columnSource,
			IsPlainField:  false,
		})
	}
	name := ""
	if ref.Alias != nil {
		name = ref.Alias.Name
	}
	return &base.PseudoTable{Name: name, Columns: columns}, nil
}

func (q *querySpanExtractor) extractJSONTable(ref *oracleast.JsonTableRef) (base.TableSource, error) {
	_, sourceColumns, err := q.extractExpr(ref.Expr)
	if err != nil {
		return nil, err
	}
	columns := extractJSONTableColumns(ref.Columns, sourceColumns)
	name := ""
	if ref.Alias != nil {
		name = ref.Alias.Name
	}
	return &base.PseudoTable{Name: name, Columns: columns}, nil
}

func extractJSONTableColumns(list *oracleast.List, sourceColumns base.SourceColumnSet) []base.QuerySpanResult {
	var columns []base.QuerySpanResult
	for _, node := range listItems(list) {
		column, ok := node.(*oracleast.JsonTableColumn)
		if !ok {
			continue
		}
		if column.Nested != nil {
			columns = append(columns, extractJSONTableColumns(column.Nested.Columns, sourceColumns)...)
			continue
		}
		columnSource := sourceColumns
		if column.ForOrdinality {
			columnSource = base.SourceColumnSet{}
		}
		columns = append(columns, base.QuerySpanResult{
			Name:          column.Name,
			SourceColumns: columnSource,
			IsPlainField:  false,
		})
	}
	return columns
}

func extractInlineExternalTable(ref *oracleast.InlineExternalTable) base.TableSource {
	var columns []base.QuerySpanResult
	for _, node := range listItems(ref.Columns) {
		column, ok := node.(*oracleast.ColumnDef)
		if !ok {
			continue
		}
		columns = append(columns, base.QuerySpanResult{
			Name:          column.Name,
			SourceColumns: base.SourceColumnSet{},
			IsPlainField:  false,
		})
	}
	name := ""
	if ref.Alias != nil {
		name = ref.Alias.Name
	}
	return &base.PseudoTable{Name: name, Columns: columns}
}

func aliasTableSource(tableSource base.TableSource, alias *oracleast.Alias) base.TableSource {
	if tableSource == nil || alias == nil || alias.Name == "" {
		return tableSource
	}
	columns := cloneQuerySpanResults(tableSource.GetQuerySpanResult())
	applyColumnAliases(columns, stringList(alias.Cols))
	return &base.PseudoTable{
		Name:    alias.Name,
		Columns: columns,
	}
}

func applyColumnAliases(columns []base.QuerySpanResult, names []string) {
	for i, name := range names {
		if i >= len(columns) {
			return
		}
		columns[i].Name = name
	}
}

func cloneQuerySpanResult(result base.QuerySpanResult) base.QuerySpanResult {
	return base.QuerySpanResult{
		Name:          result.Name,
		SourceColumns: cloneSourceColumnSet(result.SourceColumns),
		IsPlainField:  result.IsPlainField,
		Invisible:     result.Invisible,
	}
}

func cloneQuerySpanResults(results []base.QuerySpanResult) []base.QuerySpanResult {
	cloned := make([]base.QuerySpanResult, 0, len(results))
	for _, result := range results {
		cloned = append(cloned, cloneQuerySpanResult(result))
	}
	return cloned
}

func cloneSourceColumnSet(source base.SourceColumnSet) base.SourceColumnSet {
	cloned := make(base.SourceColumnSet, len(source))
	for column := range source {
		cloned[column] = true
	}
	return cloned
}

func excludeSourceColumnNames(excluded map[string]bool, source base.SourceColumnSet) {
	for column := range source {
		if column.Column == "" {
			continue
		}
		excluded[column.Column] = true
		excluded[strings.ToUpper(column.Column)] = true
	}
}

func excludeColumnRefNames(excluded map[string]bool, expr oracleast.ExprNode) {
	oracleast.Inspect(expr, func(node oracleast.Node) bool {
		columnRef, ok := node.(*oracleast.ColumnRef)
		if ok && columnRef.Schema == "" && columnRef.Table == "" && columnRef.Column != "*" {
			excluded[columnRef.Column] = true
			excluded[strings.ToUpper(columnRef.Column)] = true
		}
		return true
	})
}

func mergeSourceColumnsFromResults(results []base.QuerySpanResult) base.SourceColumnSet {
	merged := make(base.SourceColumnSet)
	for _, result := range results {
		merged, _ = base.MergeSourceColumnSet(merged, result.SourceColumns)
	}
	return merged
}

func querySpanResultsEqual(left, right []base.QuerySpanResult) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Name != right[i].Name || left[i].IsPlainField != right[i].IsPlainField {
			return false
		}
		if !sourceColumnSetEqual(left[i].SourceColumns, right[i].SourceColumns) {
			return false
		}
	}
	return true
}

func sourceColumnSetEqual(left, right base.SourceColumnSet) bool {
	if len(left) != len(right) {
		return false
	}
	for column := range left {
		if !right[column] {
			return false
		}
	}
	return true
}

func listItems(list *oracleast.List) []oracleast.Node {
	if list == nil {
		return nil
	}
	return list.Items
}

func exprList(list *oracleast.List) []oracleast.ExprNode {
	var result []oracleast.ExprNode
	for _, node := range listItems(list) {
		expr, ok := node.(oracleast.ExprNode)
		if ok {
			result = append(result, expr)
		}
	}
	return result
}

func stringList(list *oracleast.List) []string {
	var result []string
	for _, node := range listItems(list) {
		switch node := node.(type) {
		case *oracleast.String:
			result = append(result, node.Str)
		case *oracleast.ColumnRef:
			result = append(result, node.Column)
		default:
		}
	}
	return result
}

func pivotInItemName(q *querySpanExtractor, item *oracleast.PivotInItem) string {
	if item == nil {
		return ""
	}
	if item.Alias != "" {
		return item.Alias
	}
	var parts []string
	for _, expr := range exprList(item.Values) {
		name, _, err := q.extractExpr(expr)
		if err != nil {
			continue
		}
		if name != "" {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, "_")
}

func pivotColumnName(inName, aggregateAlias string, _ int) string {
	if aggregateAlias == "" {
		return inName
	}
	if inName == "" {
		return aggregateAlias
	}
	return inName + "_" + aggregateAlias
}

func isStar(expr oracleast.ExprNode) bool {
	_, ok := expr.(*oracleast.Star)
	return ok
}

func splitDBLink(dbLink string) []string {
	if dbLink == "" {
		return nil
	}
	return strings.Split(dbLink, ".")
}

func cloneTableSourceSlice(sources []base.TableSource) []base.TableSource {
	cloned := make([]base.TableSource, len(sources))
	copy(cloned, sources)
	return cloned
}

func (q *querySpanExtractor) exprName(loc oracleast.Loc, fallback string) string {
	if !loc.IsUnknown() && loc.Start >= 0 && loc.End <= len(q.source) && loc.Start < loc.End {
		return removeWhitespace(q.source[loc.Start:loc.End])
	}
	return removeWhitespace(fallback)
}

func removeWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

func getNodeLoc(node oracleast.Node) oracleast.Loc {
	value := reflect.ValueOf(node)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return oracleast.NoLoc()
	}
	elem := value.Elem()
	field := elem.FieldByName("Loc")
	if !field.IsValid() || !field.CanInterface() {
		return oracleast.NoLoc()
	}
	loc, ok := field.Interface().(oracleast.Loc)
	if !ok {
		return oracleast.NoLoc()
	}
	return loc
}

func getExprFullLoc(expr oracleast.ExprNode) oracleast.Loc {
	switch expr := expr.(type) {
	case *oracleast.BinaryExpr:
		return mergeLoc(getExprFullLoc(expr.Left), getExprFullLoc(expr.Right), getNodeLoc(expr))
	case *oracleast.UnaryExpr:
		return mergeLoc(getNodeLoc(expr), getExprFullLoc(expr.Operand))
	case *oracleast.BoolExpr:
		return mergeListLoc(expr.Args, getNodeLoc(expr))
	case *oracleast.FuncCallExpr:
		return getNodeLoc(expr)
	case *oracleast.CaseExpr:
		return mergeLoc(getNodeLoc(expr), getListLastLoc(expr.Whens), getExprFullLoc(expr.Default))
	case *oracleast.CaseWhen:
		return mergeLoc(getNodeLoc(expr), getExprFullLoc(expr.Condition), getExprFullLoc(expr.Result))
	case *oracleast.DecodeExpr:
		return mergeLoc(getNodeLoc(expr), getListLastLoc(expr.Pairs), getExprFullLoc(expr.Default))
	case *oracleast.DecodePair:
		return mergeLoc(getExprFullLoc(expr.Search), getExprFullLoc(expr.Result), getNodeLoc(expr))
	case *oracleast.BetweenExpr:
		return mergeLoc(getExprFullLoc(expr.Expr), getExprFullLoc(expr.Low), getExprFullLoc(expr.High), getNodeLoc(expr))
	case *oracleast.InExpr:
		return mergeLoc(getExprFullLoc(expr.Expr), getNodeLoc(expr))
	case *oracleast.LikeExpr:
		return mergeLoc(getExprFullLoc(expr.Expr), getExprFullLoc(expr.Pattern), getExprFullLoc(expr.Escape), getNodeLoc(expr))
	case *oracleast.IsExpr:
		return mergeLoc(getExprFullLoc(expr.Expr), getNodeLoc(expr))
	case *oracleast.CastExpr:
		return mergeLoc(getNodeLoc(expr), getExprFullLoc(expr.Arg))
	case *oracleast.MultisetExpr:
		return mergeLoc(getExprFullLoc(expr.Left), getExprFullLoc(expr.Right), getNodeLoc(expr))
	case *oracleast.CursorExpr:
		return getNodeLoc(expr)
	case *oracleast.TreatExpr:
		return mergeLoc(getNodeLoc(expr), getExprFullLoc(expr.Expr))
	case *oracleast.ParenExpr:
		return mergeLoc(getNodeLoc(expr), getExprFullLoc(expr.Expr))
	default:
		return getNodeLoc(expr)
	}
}

func mergeListLoc(list *oracleast.List, fallback oracleast.Loc) oracleast.Loc {
	loc := fallback
	for _, node := range listItems(list) {
		loc = mergeLoc(loc, getNodeLoc(node))
	}
	return loc
}

func getListLastLoc(list *oracleast.List) oracleast.Loc {
	var loc oracleast.Loc
	for _, node := range listItems(list) {
		loc = mergeLoc(loc, getNodeLoc(node))
	}
	return loc
}

func mergeLoc(locs ...oracleast.Loc) oracleast.Loc {
	result := oracleast.NoLoc()
	for _, loc := range locs {
		if loc.IsUnknown() || loc.Start < 0 || loc.End < loc.Start {
			continue
		}
		if result.IsUnknown() || loc.Start < result.Start {
			result.Start = loc.Start
		}
		if result.IsUnknown() || loc.End > result.End {
			result.End = loc.End
		}
	}
	return result
}
