package tsql

import (
	"context"
	"strings"
	"unicode"

	"github.com/bytebase/omni/mssql/ast"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// querySpanExtractor holds the shared identifier-resolution and scoping state
// used by the omni-based query span extractor. Only string-based helpers live
// on this type; AST-walking logic lives on *omniQuerySpanExtractor.
type querySpanExtractor struct {
	ctx context.Context

	defaultDatabase     string
	defaultSchema       string
	ignoreCaseSensitive bool

	gCtx base.GetQuerySpanContext
	// ctes records the common table expressions visible in the current scope.
	// Pushed when entering a WITH clause; truncated on exit.
	ctes []*base.PseudoTable

	// outerTableSources lets correlated subqueries resolve columns against the
	// enclosing query's FROM clause.
	outerTableSources []base.TableSource

	// tableSourcesFrom is this scope's FROM clause.
	tableSourcesFrom []base.TableSource

	predicateColumns base.SourceColumnSet

	viewResolutionStack map[string]bool
}

func newQuerySpanExtractor(defaultDatabase string, defaultSchema string, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) *querySpanExtractor {
	if defaultSchema == "" {
		// Fall back to the default schema `dbo`.
		// Reference: https://learn.microsoft.com/en-us/sql/relational-databases/security/authentication-access/ownership-and-user-schema-separation#the-dbo-schema
		defaultSchema = "dbo"
	}
	return &querySpanExtractor{
		defaultDatabase:     defaultDatabase,
		defaultSchema:       defaultSchema,
		gCtx:                gCtx,
		ignoreCaseSensitive: ignoreCaseSensitive,
		predicateColumns:    make(base.SourceColumnSet),
	}
}

func (q *querySpanExtractor) findTempTable(rawName string) (*base.PhysicalTable, bool) {
	for name, tempTable := range q.gCtx.TempTables {
		if q.isIdentifierEqual(rawName, name) {
			return tempTable, true
		}
	}
	return nil, false
}

// tsqlFindTableSchemaByParts resolves a (linkedServer, database, schema, table)
// tuple to a TableSource. Empty strings for database/schema mean "not specified
// by the user"; defaults are applied here when looking up metadata. CTEs
// registered in q.ctes shadow physical tables when no database/schema was
// specified.
func (q *querySpanExtractor) tsqlFindTableSchemaByParts(linkedServer, rawDatabase, rawSchema, rawTable string) (base.TableSource, error) {
	if linkedServer != "" {
		// TODO(zp): How do we handle the linked server?
		return nil, errors.Errorf("linked server is not supported yet, but found %q", linkedServer)
	}
	if strings.HasPrefix(rawTable, "#") {
		if tempTable, ok := q.findTempTable(rawTable); ok {
			return tempTable, nil
		}
		// TODO(masking): Considering SELECT * INTO #temp FROM dbo.t1; SELECT * FROM #temp. We should mask the #temp.
		return &base.PseudoTable{}, nil
	}

	// SQL Server: CTEs shadow physical tables — check first, nearest match wins.
	if rawDatabase == "" && rawSchema == "" {
		for _, cte := range q.ctes {
			if q.isIdentifierEqual(rawTable, cte.Name) {
				return cte, nil
			}
		}
	}

	database := q.defaultDatabase
	if rawDatabase != "" {
		database = rawDatabase
	}
	schema := q.defaultSchema
	if rawSchema != "" {
		schema = rawSchema
	}

	allDatabases, err := q.gCtx.ListDatabaseNamesFunc(q.ctx, q.gCtx.InstanceID)
	if err != nil {
		return nil, errors.Errorf("failed to list databases: %v", err)
	}

	for _, databaseName := range allDatabases {
		if database != "" && !q.isIdentifierEqual(database, databaseName) {
			continue
		}
		_, databaseMeta, err := q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, databaseName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database %s metadata", databaseName)
		}

		for _, schemaName := range databaseMeta.ListSchemaNames() {
			if schema != "" && !q.isIdentifierEqual(schema, schemaName) {
				continue
			}
			schemaSchema := databaseMeta.GetSchemaMetadata(schemaName)
			for _, tableName := range schemaSchema.ListTableNames() {
				if !q.isIdentifierEqual(rawTable, tableName) {
					continue
				}
				table := schemaSchema.GetTable(tableName)
				columns := make([]string, 0, len(table.GetProto().GetColumns()))
				for _, c := range table.GetProto().GetColumns() {
					columns = append(columns, c.Name)
				}
				return &base.PhysicalTable{
					Database: databaseName,
					Schema:   schemaName,
					Name:     table.GetProto().Name,
					Columns:  columns,
				}, nil
			}
			for _, viewName := range schemaSchema.ListViewNames() {
				if !q.isIdentifierEqual(rawTable, viewName) {
					continue
				}
				view := schemaSchema.GetView(viewName)
				viewColumns, err := q.getColumnsFromCreateView(view.Definition, databaseName, schemaName, viewName)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get columns for view %s.%s.%s", databaseName, schemaName, viewName)
				}
				return &base.PhysicalView{
					Database: databaseName,
					Schema:   schemaName,
					Name:     view.Name,
					Columns:  viewColumns,
				}, nil
			}
		}
	}
	return nil, &base.ResourceNotFoundError{
		Database: &database,
		Schema:   &schema,
		Table:    &rawTable,
	}
}

// getColumnsFromCreateView parses a CREATE VIEW definition with omni, extracts
// the body's result columns, and applies the optional column alias list.
func (q *querySpanExtractor) getColumnsFromCreateView(definition string, viewDatabaseName string, viewSchemaName string, viewName string) ([]base.QuerySpanResult, error) {
	key := tsqlViewResolutionKey(viewDatabaseName, viewSchemaName, viewName)
	if q.viewResolutionStack[key] {
		return nil, errors.Errorf("cyclic view reference detected while resolving %q", viewName)
	}

	stmts, err := ParseTSQLOmni(definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse CREATE VIEW definition")
	}
	var createView *ast.CreateViewStmt
	for _, s := range stmts {
		if cv, ok := s.AST.(*ast.CreateViewStmt); ok {
			createView = cv
			break
		}
	}
	if createView == nil {
		return nil, errors.Errorf("no CREATE VIEW statement found in definition")
	}
	body, ok := createView.Query.(*ast.SelectStmt)
	if !ok || body == nil {
		return nil, errors.Errorf("CREATE VIEW body is not a SELECT")
	}
	var columnAliases []string
	if createView.Columns != nil {
		for _, it := range createView.Columns.Items {
			if s, ok := it.(*ast.String); ok {
				columnAliases = append(columnAliases, s.Str)
			}
		}
	}

	newQ := newOmniQuerySpanExtractor(viewDatabaseName, viewSchemaName, q.gCtx, q.ignoreCaseSensitive)
	newQ.ctx = q.ctx
	newQ.source = definition
	newQ.viewResolutionStack = cloneViewResolutionStack(q.viewResolutionStack)
	newQ.viewResolutionStack[key] = true
	pseudo, err := newQ.extractFromSelectStmt(body)
	if err != nil {
		var resourceNotFound *base.ResourceNotFoundError
		if errors.As(err, &resourceNotFound) {
			return nil, resourceNotFound
		}
		return nil, errors.Wrapf(err, "failed to extract view body")
	}
	results := pseudo.GetQuerySpanResult()

	if len(columnAliases) > 0 && len(columnAliases) == len(results) {
		for i, alias := range columnAliases {
			results[i].Name = normIdent(alias)
		}
	}
	return results, nil
}

func tsqlViewResolutionKey(databaseName, schemaName, viewName string) string {
	return databaseName + "\x00" + schemaName + "\x00" + viewName
}

func cloneViewResolutionStack(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// tsqlGetAllFieldsOfTableInFromOrOuterCTE expands `table.*` to the column set
// of the matching table source in scope.
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

// tsqlIsFieldSensitive resolves a (database, schema, table, column) tuple to a
// QuerySpanResult by scanning tableSourcesFrom first, then outerTableSources
// (nearest-first) for correlated-subquery resolution.
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
	//
	// It is safe if there are two or more objects in the fromFieldList have the same column name, because the executor
	// will throw a compilation error if the column name is ambiguous.
	// For example, there are two tables T1 and T2, and both of them have a column named "C1". The following query will throw
	// a compilation error:
	//   SELECT C1 FROM T1, T2;
	// Users can specify the table name to disambiguate:
	//   SELECT T1.C1 FROM T1, T2;
	// If an alias is set the original table name is shadowed:
	//   SELECT T1.C1 FROM T1 AS T3, T2;  -- invalid
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

	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		tableSource := q.outerTableSources[i]
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

// isIdentifierEqual compares identifiers using the extractor's case sensitivity flag.
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

// unionTableSources unions two or more table sources column-wise.
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

// isMixedQuery reports (allSystems, mixed): mixed=true means the query touches
// both user tables and system tables simultaneously, which we treat as an
// error. allSystems=true means every accessed table is a system table.
func isMixedQuery(m base.SourceColumnSet, ignoreCaseSensitive bool) (bool, bool) {
	hasSystem, hasUser := false, false
	for table := range m {
		if isSystemResource(table, ignoreCaseSensitive) {
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

func isSystemResource(resource base.ColumnResource, ignoreCaseSensitive bool) bool {
	if IsSystemDatabase(resource.Database, !ignoreCaseSensitive) {
		return true
	}
	if IsSystemSchema(resource.Schema, !ignoreCaseSensitive) {
		return true
	}
	return false
}
