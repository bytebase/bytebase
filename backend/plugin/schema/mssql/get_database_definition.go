package mssql

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	defaultSchema = "dbo"
)

func init() {
	schema.RegisterGetDatabaseDefinition(storepb.Engine_MSSQL, GetDatabaseDefinition)
	schema.RegisterGetTableDefinition(storepb.Engine_MSSQL, GetTableDefinition)
	schema.RegisterGetViewDefinition(storepb.Engine_MSSQL, GetViewDefinition)
	schema.RegisterGetFunctionDefinition(storepb.Engine_MSSQL, GetFunctionDefinition)
	schema.RegisterGetProcedureDefinition(storepb.Engine_MSSQL, GetProcedureDefinition)
}

func GetDatabaseDefinition(_ schema.GetDefinitionContext, to *storepb.DatabaseSchemaMetadata) (string, error) {
	if to == nil {
		return "", nil
	}

	var buf strings.Builder

	// First, write all schemas
	for _, schema := range to.Schemas {
		if schema.Name != defaultSchema {
			_, _ = fmt.Fprintf(&buf, "CREATE SCHEMA [%s];\nGO\n\n", schema.Name)
		}
	}

	// Then, write all tables in dependency order
	if hasTables(to.Schemas) {
		writeAllTablesInOrder(&buf, to.Schemas)
	}

	// Then, write all views in dependency order across all schemas
	if hasViews(to.Schemas) {
		// Add GO separator between tables and views if we have tables
		hasTables := false
		for _, schema := range to.Schemas {
			if len(schema.Tables) > 0 {
				hasTables = true
				break
			}
		}
		if hasTables {
			_, _ = buf.WriteString("GO\n\n")
		}
		writeAllViewsInOrder(&buf, to.Schemas)
	}

	// Finally, write functions and procedures
	for _, schema := range to.Schemas {
		writeFunctionsAndProcedures(&buf, schema)
	}

	return buf.String(), nil
}

func GetTableDefinition(schemaName string, table *storepb.TableMetadata, _ []*storepb.SequenceMetadata) (string, error) {
	var buf strings.Builder
	writeTable(&buf, schemaName, table)
	return buf.String(), nil
}

func writeFunctionsAndProcedures(out *strings.Builder, schema *storepb.SchemaMetadata) {
	for _, function := range schema.Functions {
		writeFunction(out, schema.Name, function)
	}

	for _, procedure := range schema.Procedures {
		writeProcedure(out, schema.Name, procedure)
	}
}

func hasViews(schemas []*storepb.SchemaMetadata) bool {
	for _, schema := range schemas {
		if len(schema.Views) > 0 {
			return true
		}
	}
	return false
}

func hasTables(schemas []*storepb.SchemaMetadata) bool {
	for _, schema := range schemas {
		if len(schema.Tables) > 0 {
			return true
		}
	}
	return false
}

func writeAllTablesInOrder(out *strings.Builder, schemas []*storepb.SchemaMetadata) {
	// Collect all tables from all schemas
	var allTables []*tableWithSchema
	for _, schema := range schemas {
		for _, table := range schema.Tables {
			allTables = append(allTables, &tableWithSchema{
				schema: schema.Name,
				table:  table,
			})
		}
	}

	if len(allTables) == 0 {
		return
	}

	// Sort tables by foreign key dependencies across all schemas
	sortedTables := sortTablesByDependenciesAcrossSchemas(allTables)

	// Write sorted tables
	for _, tws := range sortedTables {
		writeTable(out, tws.schema, tws.table)
	}
}

type tableWithSchema struct {
	schema string
	table  *storepb.TableMetadata
}

// sortTablesByDependenciesAcrossSchemas sorts tables using topological sort considering cross-schema foreign key dependencies
func sortTablesByDependenciesAcrossSchemas(tables []*tableWithSchema) []*tableWithSchema {
	if len(tables) <= 1 {
		return tables
	}

	// Create a map for quick lookup
	tableMap := make(map[string]*tableWithSchema)
	for _, tws := range tables {
		tableID := getObjectID(tws.schema, tws.table.Name)
		tableMap[tableID] = tws
	}

	// Build dependency graph
	graph := base.NewGraph()

	// Add all table nodes
	for _, tws := range tables {
		tableID := getObjectID(tws.schema, tws.table.Name)
		graph.AddNode(tableID)
	}

	// Add edges based on foreign key dependencies
	for _, tws := range tables {
		tableID := getObjectID(tws.schema, tws.table.Name)

		// For each foreign key in this table
		for _, fk := range tws.table.ForeignKeys {
			// Get the referenced table ID
			referencedTableID := getObjectID(fk.ReferencedSchema, fk.ReferencedTable)

			// If the referenced table exists in our set, add a dependency edge
			// The edge goes from referenced table to current table (referenced table must be created first)
			if _, exists := tableMap[referencedTableID]; exists {
				graph.AddEdge(referencedTableID, tableID)
			}
		}
	}

	// Perform topological sort
	sortedIDs, err := graph.TopologicalSort()
	if err != nil {
		// If there's a cycle (circular foreign key references), fall back to original order
		// This shouldn't happen with well-designed schemas, but we handle it gracefully
		return tables
	}

	// Build result in topologically sorted order
	var result []*tableWithSchema
	for _, tableID := range sortedIDs {
		if tws, exists := tableMap[tableID]; exists {
			result = append(result, tws)
		}
	}

	// Add any tables that weren't in the dependency graph (shouldn't happen)
	for _, tws := range tables {
		tableID := getObjectID(tws.schema, tws.table.Name)
		found := false
		for _, resultTws := range result {
			resultTableID := getObjectID(resultTws.schema, resultTws.table.Name)
			if tableID == resultTableID {
				found = true
				break
			}
		}
		if !found {
			result = append(result, tws)
		}
	}

	return result
}

func writeAllViewsInOrder(out *strings.Builder, schemas []*storepb.SchemaMetadata) {
	// Collect all views from all schemas
	var allViews []*viewWithSchema
	for _, schema := range schemas {
		for _, view := range schema.Views {
			allViews = append(allViews, &viewWithSchema{
				schema: schema.Name,
				view:   view,
			})
		}
	}

	if len(allViews) == 0 {
		return
	}

	// Sort views by dependencies across all schemas
	sortedViews := sortViewsByDependenciesAcrossSchemas(allViews)

	// Write sorted views
	for _, vws := range sortedViews {
		writeView(out, vws.schema, vws.view)
	}
}

type viewWithSchema struct {
	schema string
	view   *storepb.ViewMetadata
}

// sortViewsByDependenciesAcrossSchemas sorts views using topological sort considering cross-schema dependencies
func sortViewsByDependenciesAcrossSchemas(views []*viewWithSchema) []*viewWithSchema {
	if len(views) <= 1 {
		return views
	}

	// Create a map for quick lookup
	viewMap := make(map[string]*viewWithSchema)
	for _, vws := range views {
		viewID := getObjectID(vws.schema, vws.view.Name)
		viewMap[viewID] = vws
	}

	// Build dependency graph
	graph := base.NewGraph()

	// Add all view nodes
	for _, vws := range views {
		viewID := getObjectID(vws.schema, vws.view.Name)
		graph.AddNode(viewID)
	}

	// Add edges based on dependencies
	for _, vws := range views {
		viewID := getObjectID(vws.schema, vws.view.Name)

		// Get dependencies from the view definition
		deps, err := getViewDependencies(vws.view.Definition, vws.schema)
		if err != nil {
			// If we can't parse dependencies, continue without adding edges
			continue
		}

		// For each dependency, check if it's a view (in any schema)
		for _, dep := range deps {
			// dep is already in format schema.table
			if _, isView := viewMap[dep]; isView {
				// The dependency view must come before this view
				graph.AddEdge(dep, viewID)
			}
		}
	}

	// Perform topological sort
	sortedIDs, err := graph.TopologicalSort()
	if err != nil {
		// If there's a cycle or error, return views in original order
		// Sort by schema then by name for deterministic output
		slices.SortFunc(views, func(a, b *viewWithSchema) int {
			if a.schema != b.schema {
				if a.schema < b.schema {
					return -1
				}
				return 1
			}
			if a.view.Name < b.view.Name {
				return -1
			}
			if a.view.Name > b.view.Name {
				return 1
			}
			return 0
		})
		return views
	}

	// Build the result in sorted order
	var result []*viewWithSchema
	for _, id := range sortedIDs {
		if vws, ok := viewMap[id]; ok {
			result = append(result, vws)
		}
	}

	return result
}

func writeTable(out *strings.Builder, schemaName string, table *storepb.TableMetadata) {
	_, _ = fmt.Fprintf(out, "CREATE TABLE [%s].[%s] (\n", schemaName, table.Name)
	for i, column := range table.Columns {
		if i != 0 {
			_, _ = out.WriteString(",\n")
		}
		writeColumn(out, column)
	}

	for _, key := range table.Indexes {
		if !key.IsConstraint {
			continue
		}

		_, _ = out.WriteString(",\n")
		writeKey(out, key)
	}

	for _, fk := range table.ForeignKeys {
		_, _ = out.WriteString(",\n")
		writeForeignKey(out, fk)
	}

	for _, check := range table.CheckConstraints {
		_, _ = out.WriteString(",\n")
		writeCheck(out, check)
	}
	_, _ = fmt.Fprintf(out, "\n);\n\n")

	for _, index := range table.Indexes {
		if index.IsConstraint {
			continue
		}
		writeIndex(out, schemaName, table.Name, index)
	}
}

func writeClusteredColumnStoreIndex(out *strings.Builder, schemaName string, tableName string, index *storepb.IndexMetadata) {
	_, _ = fmt.Fprintf(out, "CREATE CLUSTERED COLUMNSTORE INDEX [%s] ON [%s].[%s];\n\n", index.Name, schemaName, tableName)
}

func writeNonClusteredColumnStoreIndex(out *strings.Builder, schemaName string, tableName string, index *storepb.IndexMetadata) {
	_, _ = fmt.Fprintf(out, "CREATE NONCLUSTERED COLUMNSTORE INDEX [%s] ON [%s].[%s] (\n", index.Name, schemaName, tableName)
	for i, column := range index.Expressions {
		if i != 0 {
			_, _ = out.WriteString(",\n")
		}
		_, _ = fmt.Fprintf(out, "    [%s]", column)
	}
	_, _ = out.WriteString("\n);\n\n")
}

func writeNormalIndex(out *strings.Builder, schemaName string, tableName string, index *storepb.IndexMetadata) {
	_, _ = out.WriteString("CREATE")
	if index.Unique {
		_, _ = out.WriteString(" UNIQUE")
	}
	if index.Type != "" {
		_, _ = fmt.Fprintf(out, " %s", index.Type)
	}
	_, _ = fmt.Fprintf(out, " INDEX [%s] ON\n[%s].[%s] (\n", index.Name, schemaName, tableName)
	for i, column := range index.Expressions {
		if i != 0 {
			_, _ = out.WriteString(",\n")
		}
		_, _ = fmt.Fprintf(out, "    [%s]", column)
		if i < len(index.Descending) && index.Descending[i] {
			_, _ = out.WriteString(" DESC")
		} else {
			_, _ = out.WriteString(" ASC")
		}
	}
	_, _ = out.WriteString("\n);\n\n")
}

func writeIndex(out *strings.Builder, schemaName string, tableName string, index *storepb.IndexMetadata) {
	switch strings.ToUpper(index.Type) {
	case "CLUSTERED COLUMNSTORE":
		writeClusteredColumnStoreIndex(out, schemaName, tableName, index)
	case "NONCLUSTERED COLUMNSTORE":
		writeNonClusteredColumnStoreIndex(out, schemaName, tableName, index)
	default:
		writeNormalIndex(out, schemaName, tableName, index)
	}
}

func writeCheck(out *strings.Builder, check *storepb.CheckConstraintMetadata) {
	_, _ = fmt.Fprintf(out, "    CONSTRAINT [%s] CHECK %s", check.Name, check.Expression)
}

func writeForeignKey(out *strings.Builder, fk *storepb.ForeignKeyMetadata) {
	_, _ = fmt.Fprintf(out, "    CONSTRAINT [%s] FOREIGN KEY (", fk.Name)
	for i, column := range fk.Columns {
		if i != 0 {
			_, _ = out.WriteString(", ")
		}
		_, _ = fmt.Fprintf(out, "[%s]", column)
	}
	_, _ = fmt.Fprintf(out, ") REFERENCES [%s].[%s] (", fk.ReferencedSchema, fk.ReferencedTable)
	for i, column := range fk.ReferencedColumns {
		if i != 0 {
			_, _ = out.WriteString(", ")
		}
		_, _ = fmt.Fprintf(out, "[%s]", column)
	}
	_, _ = out.WriteString(")")
	if fk.OnDelete != "" {
		_, _ = fmt.Fprintf(out, " ON DELETE %s", fk.OnDelete)
	}
	if fk.OnUpdate != "" {
		_, _ = fmt.Fprintf(out, " ON UPDATE %s", fk.OnUpdate)
	}
}

func writeKey(out *strings.Builder, key *storepb.IndexMetadata) {
	_, _ = fmt.Fprintf(out, "    CONSTRAINT [%s]", key.Name)
	if key.Primary {
		_, _ = out.WriteString(" PRIMARY KEY")
	} else if key.Unique {
		_, _ = out.WriteString(" UNIQUE")
	}

	if key.Type != "" {
		_, _ = fmt.Fprintf(out, " %s", key.Type)
	}
	_, _ = out.WriteString(" (")
	for i, column := range key.Expressions {
		if i != 0 {
			_, _ = out.WriteString(", ")
		}
		_, _ = fmt.Fprintf(out, "[%s]", column)
		if i < len(key.Descending) && key.Descending[i] {
			_, _ = out.WriteString(" DESC")
		} else {
			_, _ = out.WriteString(" ASC")
		}
	}
	_, _ = out.WriteString(")")
}

func writeColumn(out *strings.Builder, column *storepb.ColumnMetadata) {
	_, _ = fmt.Fprintf(out, "    [%s] %s", column.Name, column.Type)
	if column.IsIdentity {
		_, _ = fmt.Fprintf(out, " IDENTITY(%d,%d)", column.IdentitySeed, column.IdentityIncrement)
	}
	if column.Collation != "" {
		_, _ = fmt.Fprintf(out, " COLLATE %s", column.Collation)
	}
	if column.GetDefaultExpression() != "" {
		_, _ = fmt.Fprintf(out, " DEFAULT %s", column.GetDefaultExpression())
	}
	if !column.Nullable {
		_, _ = out.WriteString(" NOT NULL")
	}
}

func writeView(out *strings.Builder, _ string, view *storepb.ViewMetadata) {
	// The view definition already contains CREATE VIEW statement
	_, _ = fmt.Fprintf(out, "%s;\n\nGO\n\n", view.Definition)
}

func writeFunction(out *strings.Builder, _ string, function *storepb.FunctionMetadata) {
	_, _ = fmt.Fprintf(out, "%s\n\nGO\n\n", function.Definition)
}

func writeProcedure(out *strings.Builder, _ string, procedure *storepb.ProcedureMetadata) {
	_, _ = fmt.Fprintf(out, "%s\n\nGO\n\n", procedure.Definition)
}

func GetViewDefinition(schemaName string, view *storepb.ViewMetadata) (string, error) {
	var buf strings.Builder
	writeView(&buf, schemaName, view)
	return buf.String(), nil
}

func GetFunctionDefinition(schemaName string, function *storepb.FunctionMetadata) (string, error) {
	var buf strings.Builder
	writeFunction(&buf, schemaName, function)
	return buf.String(), nil
}

func GetProcedureDefinition(schemaName string, procedure *storepb.ProcedureMetadata) (string, error) {
	var buf strings.Builder
	writeProcedure(&buf, schemaName, procedure)
	return buf.String(), nil
}
