package oracle

import (
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGetDatabaseDefinition(storepb.Engine_ORACLE, GetDatabaseDefinition)
	schema.RegisterGetTableDefinition(storepb.Engine_ORACLE, GetTableDefinition)
}

func GetDatabaseDefinition(_ schema.GetDefinitionContext, to *storepb.DatabaseSchemaMetadata) (string, error) {
	if len(to.Schemas) == 0 {
		return "", nil
	}

	var buf strings.Builder
	schema := to.Schemas[0]
	// For Oracle, the database name (user/schema name) is stored in to.Name
	databaseName := to.Name

	// First generate sequences (they need to exist before tables that reference them)
	for _, sequence := range schema.Sequences {
		// Skip system-generated sequences
		if !strings.HasPrefix(sequence.Name, "ISEQ$$_") {
			if err := writeSequence(&buf, sequence); err != nil {
				return "", err
			}
		}
	}

	// Build dependency graph for topological sorting
	graph := base.NewGraph()
	tableMap := make(map[string]*storepb.TableMetadata)
	viewMap := make(map[string]*storepb.ViewMetadata)
	materializedViewMap := make(map[string]*storepb.MaterializedViewMetadata)
	functionMap := make(map[string]*storepb.FunctionMetadata)
	procedureMap := make(map[string]*storepb.ProcedureMetadata)

	// Add tables to graph
	for _, table := range schema.Tables {
		tableID := getObjectID(databaseName, table.Name)
		tableMap[tableID] = table
		graph.AddNode(tableID)
	}

	// Add views to graph
	for _, view := range schema.Views {
		viewID := getObjectID(databaseName, view.Name)
		viewMap[viewID] = view
		graph.AddNode(viewID)
		// Add dependencies from view to tables/views it references
		for _, dependency := range view.DependencyColumns {
			dependencyID := getObjectID(dependency.Schema, dependency.Table)
			graph.AddEdge(dependencyID, viewID)
		}
	}

	// Add materialized views to graph
	for _, view := range schema.MaterializedViews {
		viewID := getObjectID(databaseName, view.Name)
		materializedViewMap[viewID] = view
		graph.AddNode(viewID)
		// Add dependencies from materialized view to tables/views it references
		for _, dependency := range view.DependencyColumns {
			dependencyID := getObjectID(dependency.Schema, dependency.Table)
			graph.AddEdge(dependencyID, viewID)
		}
	}

	// Add functions to graph
	for _, function := range schema.Functions {
		functionID := getObjectID(databaseName, function.Name)
		functionMap[functionID] = function
		graph.AddNode(functionID)
		// Add dependencies from function to tables it references
		for _, dependency := range function.DependencyTables {
			dependencyID := getObjectID(dependency.Schema, dependency.Table)
			graph.AddEdge(dependencyID, functionID)
		}
	}

	// Add procedures to graph
	for _, procedure := range schema.Procedures {
		procedureID := getObjectID(databaseName, procedure.Name)
		procedureMap[procedureID] = procedure
		graph.AddNode(procedureID)
		// Note: ProcedureMetadata doesn't have DependencyTables field
		// Procedures will be created after tables by default order
	}

	// Add foreign key dependencies between tables
	for _, table := range schema.Tables {
		tableID := getObjectID(databaseName, table.Name)
		for _, fk := range table.ForeignKeys {
			if fk.ReferencedTable != table.Name { // Avoid self-references
				referencedTableID := getObjectID(databaseName, fk.ReferencedTable)
				graph.AddEdge(referencedTableID, tableID)
			}
		}
	}

	// Perform topological sort
	orderedList, err := graph.TopologicalSort()
	if err != nil {
		// If there are cycles, fall back to original order
		for _, table := range schema.Tables {
			if err := writeTable(&buf, table); err != nil {
				return "", err
			}
		}
		for _, view := range schema.Views {
			if err := writeView(&buf, view); err != nil {
				return "", err
			}
		}
		for _, view := range schema.MaterializedViews {
			if err := writeMaterializedView(&buf, view); err != nil {
				return "", err
			}
		}
		for _, function := range schema.Functions {
			if err := writeFunction(&buf, function); err != nil {
				return "", err
			}
		}
		for _, procedure := range schema.Procedures {
			if err := writeProcedure(&buf, procedure); err != nil {
				return "", err
			}
		}
		return buf.String(), nil
	}

	// Generate objects in dependency order
	for _, objectID := range orderedList {
		if table, ok := tableMap[objectID]; ok {
			if err := writeTable(&buf, table); err != nil {
				return "", err
			}
		}
		if view, ok := viewMap[objectID]; ok {
			if err := writeView(&buf, view); err != nil {
				return "", err
			}
		}
		if view, ok := materializedViewMap[objectID]; ok {
			if err := writeMaterializedView(&buf, view); err != nil {
				return "", err
			}
		}
		if function, ok := functionMap[objectID]; ok {
			if err := writeFunction(&buf, function); err != nil {
				return "", err
			}
		}
		if procedure, ok := procedureMap[objectID]; ok {
			if err := writeProcedure(&buf, procedure); err != nil {
				return "", err
			}
		}
	}

	return buf.String(), nil
}

func GetTableDefinition(_ string, table *storepb.TableMetadata, _ []*storepb.SequenceMetadata) (string, error) {
	var buf strings.Builder
	if err := writeTable(&buf, table); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func writeTable(buf *strings.Builder, table *storepb.TableMetadata) error {
	if _, err := buf.WriteString(`CREATE TABLE "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(table.Name); err != nil {
		return err
	}
	if _, err := buf.WriteString("\" (\n"); err != nil {
		return err
	}
	for i, column := range table.Columns {
		if i > 0 {
			if _, err := buf.WriteString(",\n"); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`  `); err != nil {
			return err
		}
		if err := writeColumn(buf, column); err != nil {
			return err
		}
	}

	constraints := []*storepb.IndexMetadata{}
	for _, constraint := range table.Indexes {
		if constraint.IsConstraint {
			constraints = append(constraints, constraint)
		}
	}

	for i, constraint := range constraints {
		if i+len(table.Indexes) > 0 {
			if _, err := buf.WriteString(",\n"); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`  `); err != nil {
			return err
		}
		if err := writeConstraint(buf, constraint); err != nil {
			return err
		}
	}

	for i, check := range table.CheckConstraints {
		if i+len(table.Indexes)+len(constraints) > 0 {
			if _, err := buf.WriteString(",\n"); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`  `); err != nil {
			return err
		}
		if err := writeCheckConstraint(buf, check); err != nil {
			return err
		}
	}

	for i, fk := range table.ForeignKeys {
		if i+len(table.Indexes)+len(constraints)+len(table.CheckConstraints) > 0 {
			if _, err := buf.WriteString(",\n"); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`  `); err != nil {
			return err
		}
		if err := writeForeignKey(buf, fk); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n);\n\n"); err != nil {
		return err
	}

	for _, index := range table.Indexes {
		if index.IsConstraint {
			continue
		}
		if err := writeIndex(buf, table.Name, index); err != nil {
			return err
		}
	}

	// Write triggers for this table
	for _, trigger := range table.Triggers {
		if err := writeTrigger(buf, trigger); err != nil {
			return err
		}
	}

	return nil
}

func writeIndex(buf *strings.Builder, table string, index *storepb.IndexMetadata) error {
	if _, err := buf.WriteString(`CREATE`); err != nil {
		return err
	}

	switch index.Type {
	case "BITMAP", "FUNCTION-BASED BITMAP":
		if _, err := buf.WriteString(` BITMAP`); err != nil {
			return err
		}
	default:
		// Other index types don't need special syntax
	}

	if index.Unique {
		if _, err := buf.WriteString(` UNIQUE`); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString(` INDEX "`); err != nil {
		return err
	}

	if _, err := buf.WriteString(index.Name); err != nil {
		return err
	}

	if _, err := buf.WriteString(`" ON "`); err != nil {
		return err
	}

	if _, err := buf.WriteString(table); err != nil {
		return err
	}

	if _, err := buf.WriteString(`" (`); err != nil {
		return err
	}

	if strings.Contains(index.Type, "FUNCTION-BASED") {
		for i, expression := range index.Expressions {
			if i != 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString(expression); err != nil {
				return err
			}

			if i < len(index.Descending) && index.Descending[i] {
				if _, err := buf.WriteString(` DESC`); err != nil {
					return err
				}
			} else {
				if _, err := buf.WriteString(` ASC`); err != nil {
					return err
				}
			}
		}
	} else {
		for i, column := range index.Expressions {
			if i != 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}

			// Check if column is already quoted to avoid double quoting
			if strings.HasPrefix(column, `"`) && strings.HasSuffix(column, `"`) {
				// Column is already quoted, use as-is
				if _, err := buf.WriteString(column); err != nil {
					return err
				}
			} else {
				// Column is not quoted, add quotes
				if _, err := buf.WriteString(`"`); err != nil {
					return err
				}
				if _, err := buf.WriteString(column); err != nil {
					return err
				}
				if _, err := buf.WriteString(`"`); err != nil {
					return err
				}
			}

			if i < len(index.Descending) && index.Descending[i] {
				if _, err := buf.WriteString(` DESC`); err != nil {
					return err
				}
			} else {
				if _, err := buf.WriteString(` ASC`); err != nil {
					return err
				}
			}
		}
	}

	if _, err := buf.WriteString(`);`); err != nil {
		return err
	}
	if _, err := buf.WriteString("\n\n"); err != nil {
		return err
	}
	return nil
}

func writeForeignKey(buf *strings.Builder, fk *storepb.ForeignKeyMetadata) error {
	if _, err := buf.WriteString(`CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(fk.Name); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(` FOREIGN KEY (`); err != nil {
		return err
	}
	for i, column := range fk.Columns {
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`"`); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"`); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(`) REFERENCES "`); err != nil {
		return err
	}
	// Only include schema prefix for cross-schema foreign key references
	// Same-schema references don't need prefix for portability
	if fk.ReferencedSchema != "" {
		if _, err := buf.WriteString(fk.ReferencedSchema); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"."`); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(fk.ReferencedTable); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" (`); err != nil {
		return err
	}
	for i, column := range fk.ReferencedColumns {
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`"`); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"`); err != nil {
			return err
		}
	}
	_, err := buf.WriteString(`)`)
	return err
}

func writeCheckConstraint(buf *strings.Builder, check *storepb.CheckConstraintMetadata) error {
	if _, err := buf.WriteString(`CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(check.Name); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(` CHECK (`); err != nil {
		return err
	}
	if _, err := buf.WriteString(check.Expression); err != nil {
		return err
	}
	_, err := buf.WriteString(`)`)
	return err
}

func writeConstraint(buf *strings.Builder, constraint *storepb.IndexMetadata) error {
	if _, err := buf.WriteString(`CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(constraint.Name); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"`); err != nil {
		return err
	}

	switch {
	case constraint.Primary:
		if _, err := buf.WriteString(` PRIMARY KEY (`); err != nil {
			return err
		}
		for i, column := range constraint.Expressions {
			if i != 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString(`"`); err != nil {
				return err
			}
			if _, err := buf.WriteString(column); err != nil {
				return err
			}
			if _, err := buf.WriteString(`"`); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`)`); err != nil {
			return err
		}
	case !constraint.Primary && constraint.Unique:
		if _, err := buf.WriteString(` UNIQUE (`); err != nil {
			return err
		}
		for i, column := range constraint.Expressions {
			if i != 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString(`"`); err != nil {
				return err
			}
			if _, err := buf.WriteString(column); err != nil {
				return err
			}
			if _, err := buf.WriteString(`"`); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`)`); err != nil {
			return err
		}
	default:
		// Other constraint types (e.g. CHECK, FOREIGN KEY)
	}
	return nil
}

func writeColumn(buf *strings.Builder, column *storepb.ColumnMetadata) error {
	if _, err := buf.WriteString(`"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(column.Name); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" `); err != nil {
		return err
	}
	if _, err := buf.WriteString(column.Type); err != nil {
		return err
	}
	// Skip collation for Oracle as it causes issues with standard string size settings
	// if column.Collation != "" {
	//	if _, err := buf.WriteString(` COLLATE "`); err != nil {
	//		return err
	//	}
	//	if _, err := buf.WriteString(column.Collation); err != nil {
	//		return err
	//	}
	//	if _, err := buf.WriteString(`"`); err != nil {
	//		return err
	//	}
	// }
	// Handle default values
	hasDefault := column.Default != ""
	if hasDefault {
		defaultExpr := column.Default

		// Skip system-generated sequence references as they can't be manually created
		if defaultExpr != "" && !strings.Contains(defaultExpr, "ISEQ$$_") {
			if _, err := buf.WriteString(` DEFAULT `); err != nil {
				return err
			}
			if column.DefaultOnNull {
				if _, err := buf.WriteString(`ON NULL `); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString(defaultExpr); err != nil {
				return err
			}
		}
	}
	if !column.Nullable {
		if _, err := buf.WriteString(` NOT NULL`); err != nil {
			return err
		}
	}
	return nil
}

func writeSequence(buf *strings.Builder, sequence *storepb.SequenceMetadata) error {
	if _, err := buf.WriteString("CREATE SEQUENCE \""); err != nil {
		return err
	}
	if _, err := buf.WriteString(sequence.Name); err != nil {
		return err
	}
	if _, err := buf.WriteString("\""); err != nil {
		return err
	}

	if sequence.Start != "" && sequence.Start != "0" {
		if _, err := buf.WriteString(" START WITH "); err != nil {
			return err
		}
		if _, err := buf.WriteString(sequence.Start); err != nil {
			return err
		}
	}

	if sequence.Increment != "" && sequence.Increment != "0" && sequence.Increment != "1" {
		if _, err := buf.WriteString(" INCREMENT BY "); err != nil {
			return err
		}
		if _, err := buf.WriteString(sequence.Increment); err != nil {
			return err
		}
	}

	if sequence.MaxValue != "" && sequence.MaxValue != "0" {
		if _, err := buf.WriteString(" MAXVALUE "); err != nil {
			return err
		}
		if _, err := buf.WriteString(sequence.MaxValue); err != nil {
			return err
		}
	}

	if sequence.Cycle {
		if _, err := buf.WriteString(" CYCLE"); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString(";\n\n"); err != nil {
		return err
	}

	return nil
}

// getObjectID returns a unique identifier for database objects
func getObjectID(schema, objectName string) string {
	if schema == "" {
		return objectName
	}
	return schema + "." + objectName
}

// writeView writes a CREATE VIEW statement
func writeView(buf *strings.Builder, view *storepb.ViewMetadata) error {
	if _, err := buf.WriteString(`CREATE VIEW "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(view.Name); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" AS `); err != nil {
		return err
	}
	if _, err := buf.WriteString(view.Definition); err != nil {
		return err
	}
	if !strings.HasSuffix(strings.TrimSpace(view.Definition), ";") {
		if _, err := buf.WriteString(`;`); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n\n"); err != nil {
		return err
	}
	return nil
}

// writeMaterializedView writes a CREATE MATERIALIZED VIEW statement
func writeMaterializedView(buf *strings.Builder, view *storepb.MaterializedViewMetadata) error {
	if _, err := buf.WriteString(`CREATE MATERIALIZED VIEW "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(view.Name); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" AS `); err != nil {
		return err
	}
	if _, err := buf.WriteString(view.Definition); err != nil {
		return err
	}
	if !strings.HasSuffix(strings.TrimSpace(view.Definition), ";") {
		if _, err := buf.WriteString(`;`); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n\n"); err != nil {
		return err
	}
	return nil
}

// writeFunction writes a CREATE FUNCTION statement
func writeFunction(buf *strings.Builder, function *storepb.FunctionMetadata) error {
	definition := function.Definition
	// If the definition doesn't start with CREATE, add the CREATE OR REPLACE prefix
	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(definition)), "CREATE") {
		if _, err := buf.WriteString("CREATE OR REPLACE "); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(definition); err != nil {
		return err
	}
	if !strings.HasSuffix(strings.TrimSpace(definition), ";") {
		if _, err := buf.WriteString(";"); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n\n"); err != nil {
		return err
	}
	return nil
}

// writeProcedure writes a CREATE PROCEDURE statement
func writeProcedure(buf *strings.Builder, procedure *storepb.ProcedureMetadata) error {
	definition := procedure.Definition
	// If the definition doesn't start with CREATE, add the CREATE OR REPLACE prefix
	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(definition)), "CREATE") {
		if _, err := buf.WriteString("CREATE OR REPLACE "); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(definition); err != nil {
		return err
	}
	if !strings.HasSuffix(strings.TrimSpace(definition), ";") {
		if _, err := buf.WriteString(";"); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n\n"); err != nil {
		return err
	}
	return nil
}

// writeTrigger writes a CREATE TRIGGER statement
func writeTrigger(buf *strings.Builder, trigger *storepb.TriggerMetadata) error {
	// The trigger body should already contain the full CREATE TRIGGER statement
	if _, err := buf.WriteString(trigger.Body); err != nil {
		return err
	}
	if !strings.HasSuffix(strings.TrimSpace(trigger.Body), ";") {
		if _, err := buf.WriteString(";"); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n\n"); err != nil {
		return err
	}
	return nil
}
