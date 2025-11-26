package pg

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

const (
	header = `
SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

`

	setDefaultTableSpace = "SET default_tablespace = '';\n\n"
)

func init() {
	schema.RegisterGetDatabaseDefinition(storepb.Engine_POSTGRES, GetDatabaseDefinition)
	schema.RegisterGetSchemaDefinition(storepb.Engine_POSTGRES, GetSchemaDefinition)
	schema.RegisterGetTableDefinition(storepb.Engine_POSTGRES, GetTableDefinition)
	schema.RegisterGetViewDefinition(storepb.Engine_POSTGRES, GetViewDefinition)
	schema.RegisterGetMaterializedViewDefinition(storepb.Engine_POSTGRES, GetMaterializedViewDefinition)
	schema.RegisterGetFunctionDefinition(storepb.Engine_POSTGRES, GetFunctionDefinition)
	schema.RegisterGetSequenceDefinition(storepb.Engine_POSTGRES, GetSequenceDefinition)
	schema.RegisterGetMultiFileDatabaseDefinition(storepb.Engine_POSTGRES, GetMultiFileDatabaseDefinition)
}

func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *storepb.DatabaseSchemaMetadata) (string, error) {
	metadata = filterBackupSchemaIfNecessary(ctx, metadata)

	if len(metadata.Schemas) == 0 {
		return "", nil
	}

	var buf strings.Builder

	if ctx.SDLFormat {
		return getSDLFormat(metadata)
	}

	if ctx.PrintHeader {
		if _, err := buf.WriteString(header); err != nil {
			return "", err
		}
	}

	// Construct schemas.
	for _, schema := range metadata.Schemas {
		if schema.SkipDump {
			continue
		}
		if err := writeSchema(&buf, schema); err != nil {
			return "", err
		}
	}

	// Construct extensions.
	for _, extension := range metadata.Extensions {
		if err := writeExtension(&buf, extension); err != nil {
			return "", err
		}
	}

	// Construct enums.
	for _, schema := range metadata.Schemas {
		for _, enum := range schema.EnumTypes {
			if enum.SkipDump {
				continue
			}
			if err := writeEnum(&buf, schema.Name, enum); err != nil {
				return "", err
			}
			if _, err := buf.WriteString(";\n\n"); err != nil {
				return "", err
			}
			// Write enum comment if present
			if len(enum.Comment) > 0 {
				if err := writeEnumComment(&buf, schema.Name, enum); err != nil {
					return "", err
				}
			}
		}
	}

	// Build the graph for topological sort.
	graph := parserbase.NewGraph()
	functionMap := make(map[string]*storepb.FunctionMetadata)
	tableMap := make(map[string]*storepb.TableMetadata)
	viewMap := make(map[string]*storepb.ViewMetadata)
	materializedViewMap := make(map[string]*storepb.MaterializedViewMetadata)

	// Construct functions.
	for _, schema := range metadata.Schemas {
		for _, function := range schema.Functions {
			if function.SkipDump {
				continue
			}
			funcID := getObjectID(schema.Name, function.Name)
			functionMap[funcID] = function
			graph.AddNode(funcID)
			for _, dependency := range function.DependencyTables {
				dependencyID := getObjectID(dependency.Schema, dependency.Table)
				graph.AddEdge(dependencyID, funcID)
			}
		}
	}

	// Create non-identity sequences before tables to prevent errors when tables reference
	// sequences owned by other tables. Identity sequences are created via writeColumnIdentityGeneration.
	// Build identity column map once for O(m*c) instead of O(s*m*c) when checking each sequence.
	identityColumnMap := buildIdentityColumnMap(metadata)

	sequenceOwnershipMap := make(map[string][]*storepb.SequenceMetadata)
	for _, schema := range metadata.Schemas {
		for _, sequence := range schema.Sequences {
			if sequence.SkipDump {
				continue
			}
			// Check if this sequence belongs to an identity column using the prebuilt map
			isIdentity := false
			if sequence.OwnerTable != "" && sequence.OwnerColumn != "" {
				key := schema.Name + "." + sequence.OwnerTable + "." + sequence.OwnerColumn
				isIdentity = identityColumnMap[key]
			}

			if isIdentity {
				tableID := getObjectID(schema.Name, sequence.OwnerTable)
				sequenceOwnershipMap[tableID] = append(sequenceOwnershipMap[tableID], sequence)
				continue
			}
			if err := writeCreateSequence(&buf, schema.Name, sequence); err != nil {
				return "", err
			}
			if sequence.OwnerTable != "" && sequence.OwnerColumn != "" {
				tableID := getObjectID(schema.Name, sequence.OwnerTable)
				sequenceOwnershipMap[tableID] = append(sequenceOwnershipMap[tableID], sequence)
			}
		}
	}

	if ctx.PrintHeader {
		if _, err := buf.WriteString(setDefaultTableSpace); err != nil {
			return "", err
		}
	}

	// Construct tables, views and materialized views.
	for _, schema := range metadata.Schemas {
		for _, table := range schema.Tables {
			if table.SkipDump {
				continue
			}
			tableID := getObjectID(schema.Name, table.Name)
			tableMap[tableID] = table
			graph.AddNode(tableID)
		}

		for _, view := range schema.Views {
			if view.SkipDump {
				continue
			}
			viewID := getObjectID(schema.Name, view.Name)
			viewMap[viewID] = view
			graph.AddNode(viewID)
			for _, dependency := range view.DependencyColumns {
				dependencyID := getObjectID(dependency.Schema, dependency.Table)
				graph.AddEdge(dependencyID, viewID)
			}
		}

		for _, view := range schema.MaterializedViews {
			if view.SkipDump {
				continue
			}
			viewID := getObjectID(schema.Name, view.Name)
			materializedViewMap[viewID] = view
			graph.AddNode(viewID)
			for _, dependency := range view.DependencyColumns {
				dependencyID := getObjectID(dependency.Schema, dependency.Table)
				graph.AddEdge(dependencyID, viewID)
			}
		}
	}

	orderedList, err := graph.TopologicalSort()
	if err != nil {
		// If topological sort fails (e.g., due to circular dependencies),
		// fall back to a safe dependency order: tables -> views -> materialized views -> functions
		orderedList = make([]string, 0, len(functionMap)+len(tableMap)+len(viewMap)+len(materializedViewMap))

		// Add tables first (sorted alphabetically within each category)
		var tableIDs []string
		for id := range tableMap {
			tableIDs = append(tableIDs, id)
		}
		slices.Sort(tableIDs)
		orderedList = append(orderedList, tableIDs...)

		// Add views second
		var viewIDs []string
		for id := range viewMap {
			viewIDs = append(viewIDs, id)
		}
		slices.Sort(viewIDs)
		orderedList = append(orderedList, viewIDs...)

		// Add materialized views third
		var materializedViewIDs []string
		for id := range materializedViewMap {
			materializedViewIDs = append(materializedViewIDs, id)
		}
		slices.Sort(materializedViewIDs)
		orderedList = append(orderedList, materializedViewIDs...)

		// Add functions last
		var functionIDs []string
		for id := range functionMap {
			functionIDs = append(functionIDs, id)
		}
		slices.Sort(functionIDs)
		orderedList = append(orderedList, functionIDs...)
	}

	// Construct functions, tables, views and materialized views in order.
	for _, objectID := range orderedList {
		if function, ok := functionMap[objectID]; ok {
			if err := writeFunction(&buf, getSchemaNameFromID(objectID), function); err != nil {
				return "", err
			}
			continue
		}
		if table, ok := tableMap[objectID]; ok {
			if err := writeTable(&buf, getSchemaNameFromID(objectID), table, sequenceOwnershipMap[objectID]); err != nil {
				return "", err
			}
		}
		if view, ok := viewMap[objectID]; ok {
			if err := writeView(&buf, getSchemaNameFromID(objectID), view); err != nil {
				return "", err
			}
			continue
		}
		if view, ok := materializedViewMap[objectID]; ok {
			if err := writeMaterializedView(&buf, getSchemaNameFromID(objectID), view); err != nil {
				return "", err
			}
		}
	}

	// Construct triggers.
	for _, schema := range metadata.Schemas {
		for _, table := range schema.Tables {
			for _, trigger := range table.Triggers {
				if trigger.SkipDump {
					continue
				}
				if err := writeTrigger(&buf, schema.Name, table.Name, trigger); err != nil {
					return "", err
				}
			}
		}

		for _, view := range schema.Views {
			for _, trigger := range view.Triggers {
				if trigger.SkipDump {
					continue
				}
				if err := writeTrigger(&buf, schema.Name, view.Name, trigger); err != nil {
					return "", err
				}
			}
		}

		for _, materializedView := range schema.MaterializedViews {
			for _, trigger := range materializedView.Triggers {
				if trigger.SkipDump {
					continue
				}
				if err := writeTrigger(&buf, schema.Name, materializedView.Name, trigger); err != nil {
					return "", err
				}
			}
		}
	}

	// Construct rules.
	for _, schema := range metadata.Schemas {
		for _, table := range schema.Tables {
			if len(table.Rules) > 0 {
				if err := writeRules(&buf, schema.Name, table.Name, table.Rules); err != nil {
					return "", err
				}
			}
		}

		for _, view := range schema.Views {
			// Rules for views are already written in writeView
			// Only write non-SELECT rules here (they are not part of the view definition)
			var nonSelectRules []*storepb.RuleMetadata
			for _, rule := range view.Rules {
				if rule.Event != "SELECT" {
					nonSelectRules = append(nonSelectRules, rule)
				}
			}
			if len(nonSelectRules) > 0 {
				if err := writeRules(&buf, schema.Name, view.Name, nonSelectRules); err != nil {
					return "", err
				}
			}
		}
	}

	// Construct foreign keys.
	for _, schema := range metadata.Schemas {
		for _, table := range schema.Tables {
			if table.SkipDump {
				continue
			}
			for _, fk := range table.ForeignKeys {
				if err := writeForeignKey(&buf, schema.Name, table.Name, fk); err != nil {
					return "", err
				}
			}
		}
	}

	// Construct event triggers (last, as they depend on functions).
	for _, eventTrigger := range metadata.EventTriggers {
		if eventTrigger.SkipDump {
			continue
		}
		if err := writeEventTrigger(&buf, eventTrigger); err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}

func GetSchemaSDLDefinition(schema *storepb.SchemaMetadata) (string, error) {
	// Create a temporary database metadata containing just this schema
	tempMetadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{schema},
	}

	return getSDLFormat(tempMetadata)
}

func GetSchemaDefinition(schema *storepb.SchemaMetadata) (string, error) {
	var buf strings.Builder
	if err := writeSchema(&buf, schema); err != nil {
		return "", err
	}

	// Construct enums.
	for _, enum := range schema.EnumTypes {
		if enum.SkipDump {
			continue
		}
		if err := writeEnum(&buf, schema.Name, enum); err != nil {
			return "", err
		}
		if _, err := buf.WriteString(";\n\n"); err != nil {
			return "", err
		}
		// Write enum comment if present
		if len(enum.Comment) > 0 {
			if err := writeEnumComment(&buf, schema.Name, enum); err != nil {
				return "", err
			}
		}
	}

	// Build the graph for topological sort.
	graph := parserbase.NewGraph()
	functionMap := make(map[string]*storepb.FunctionMetadata)
	tableMap := make(map[string]*storepb.TableMetadata)
	viewMap := make(map[string]*storepb.ViewMetadata)
	materializedViewMap := make(map[string]*storepb.MaterializedViewMetadata)

	// Construct functions.
	for _, function := range schema.Functions {
		if function.SkipDump {
			continue
		}
		funcID := getObjectID(schema.Name, function.Name)
		functionMap[funcID] = function
		graph.AddNode(funcID)
		for _, dependency := range function.DependencyTables {
			dependencyID := getObjectID(dependency.Schema, dependency.Table)
			graph.AddEdge(dependencyID, funcID)
		}
	}

	// Create non-identity sequences before tables to prevent errors when tables reference
	// sequences owned by other tables. Identity sequences are created via writeColumnIdentityGeneration.
	// Build identity column map once for O(m*c) instead of O(s*m*c) when checking each sequence.
	identityColumnMap := buildIdentityColumnMapForSchema(schema)

	sequenceOwnershipMap := make(map[string][]*storepb.SequenceMetadata)
	for _, sequence := range schema.Sequences {
		if sequence.SkipDump {
			continue
		}
		// Check if this sequence belongs to an identity column using the prebuilt map
		isIdentity := false
		if sequence.OwnerTable != "" && sequence.OwnerColumn != "" {
			key := sequence.OwnerTable + "." + sequence.OwnerColumn
			isIdentity = identityColumnMap[key]
		}

		if isIdentity {
			tableID := getObjectID(schema.Name, sequence.OwnerTable)
			sequenceOwnershipMap[tableID] = append(sequenceOwnershipMap[tableID], sequence)
			continue
		}
		if err := writeCreateSequence(&buf, schema.Name, sequence); err != nil {
			return "", err
		}
		if sequence.OwnerTable != "" && sequence.OwnerColumn != "" {
			tableID := getObjectID(schema.Name, sequence.OwnerTable)
			sequenceOwnershipMap[tableID] = append(sequenceOwnershipMap[tableID], sequence)
		}
	}

	// Construct tables, views and materialized views.
	for _, table := range schema.Tables {
		if table.SkipDump {
			continue
		}
		tableID := getObjectID(schema.Name, table.Name)
		tableMap[tableID] = table
		graph.AddNode(tableID)
	}

	for _, view := range schema.Views {
		if view.SkipDump {
			continue
		}
		viewID := getObjectID(schema.Name, view.Name)
		viewMap[viewID] = view
		graph.AddNode(viewID)
		for _, dependency := range view.DependencyColumns {
			dependencyID := getObjectID(dependency.Schema, dependency.Table)
			graph.AddEdge(dependencyID, viewID)
		}
	}

	for _, view := range schema.MaterializedViews {
		if view.SkipDump {
			continue
		}
		viewID := getObjectID(schema.Name, view.Name)
		materializedViewMap[viewID] = view
		graph.AddNode(viewID)
		for _, dependency := range view.DependencyColumns {
			dependencyID := getObjectID(dependency.Schema, dependency.Table)
			graph.AddEdge(dependencyID, viewID)
		}
	}

	orderedList, err := graph.TopologicalSort()
	if err != nil {
		// If topological sort fails (e.g., due to circular dependencies),
		// fall back to a safe dependency order: tables -> views -> materialized views -> functions
		orderedList = make([]string, 0, len(functionMap)+len(tableMap)+len(viewMap)+len(materializedViewMap))

		// Add tables first (sorted alphabetically within each category)
		var tableIDs []string
		for id := range tableMap {
			tableIDs = append(tableIDs, id)
		}
		slices.Sort(tableIDs)
		orderedList = append(orderedList, tableIDs...)

		// Add views second
		var viewIDs []string
		for id := range viewMap {
			viewIDs = append(viewIDs, id)
		}
		slices.Sort(viewIDs)
		orderedList = append(orderedList, viewIDs...)

		// Add materialized views third
		var materializedViewIDs []string
		for id := range materializedViewMap {
			materializedViewIDs = append(materializedViewIDs, id)
		}
		slices.Sort(materializedViewIDs)
		orderedList = append(orderedList, materializedViewIDs...)

		// Add functions last
		var functionIDs []string
		for id := range functionMap {
			functionIDs = append(functionIDs, id)
		}
		slices.Sort(functionIDs)
		orderedList = append(orderedList, functionIDs...)
	}

	// Construct functions, tables, views and materialized views in order.
	for _, objectID := range orderedList {
		if function, ok := functionMap[objectID]; ok {
			if err := writeFunction(&buf, getSchemaNameFromID(objectID), function); err != nil {
				return "", err
			}
			continue
		}
		if table, ok := tableMap[objectID]; ok {
			if err := writeTable(&buf, getSchemaNameFromID(objectID), table, sequenceOwnershipMap[objectID]); err != nil {
				return "", err
			}
		}
		if view, ok := viewMap[objectID]; ok {
			if err := writeView(&buf, getSchemaNameFromID(objectID), view); err != nil {
				return "", err
			}
			continue
		}
		if view, ok := materializedViewMap[objectID]; ok {
			if err := writeMaterializedView(&buf, getSchemaNameFromID(objectID), view); err != nil {
				return "", err
			}
		}
	}

	// Construct triggers.
	for _, table := range schema.Tables {
		for _, trigger := range table.Triggers {
			if trigger.SkipDump {
				continue
			}
			if err := writeTrigger(&buf, schema.Name, table.Name, trigger); err != nil {
				return "", err
			}
		}
	}

	for _, view := range schema.Views {
		for _, trigger := range view.Triggers {
			if trigger.SkipDump {
				continue
			}
			if err := writeTrigger(&buf, schema.Name, view.Name, trigger); err != nil {
				return "", err
			}
		}
	}

	for _, materializedView := range schema.MaterializedViews {
		for _, trigger := range materializedView.Triggers {
			if trigger.SkipDump {
				continue
			}
			if err := writeTrigger(&buf, schema.Name, materializedView.Name, trigger); err != nil {
				return "", err
			}
		}
	}

	// Construct rules.
	for _, table := range schema.Tables {
		if len(table.Rules) > 0 {
			if err := writeRules(&buf, schema.Name, table.Name, table.Rules); err != nil {
				return "", err
			}
		}
	}

	for _, view := range schema.Views {
		// Only write non-SELECT rules here (SELECT rules are part of the view definition)
		var nonSelectRules []*storepb.RuleMetadata
		for _, rule := range view.Rules {
			if rule.Event != "SELECT" {
				nonSelectRules = append(nonSelectRules, rule)
			}
		}
		if len(nonSelectRules) > 0 {
			if err := writeRules(&buf, schema.Name, view.Name, nonSelectRules); err != nil {
				return "", err
			}
		}
	}

	// Construct foreign keys.
	for _, table := range schema.Tables {
		if table.SkipDump {
			continue
		}
		for _, fk := range table.ForeignKeys {
			if err := writeForeignKey(&buf, schema.Name, table.Name, fk); err != nil {
				return "", err
			}
		}
	}

	return buf.String(), nil
}

func GetTableDefinition(schema string, table *storepb.TableMetadata, sequences []*storepb.SequenceMetadata) (string, error) {
	var buf strings.Builder
	if err := writeTable(&buf, schema, table, sequences); err != nil {
		return "", err
	}
	// Construct triggers.
	for _, trigger := range table.Triggers {
		if trigger.SkipDump {
			continue
		}
		if err := writeTrigger(&buf, schema, table.Name, trigger); err != nil {
			return "", err
		}
	}
	// Construct rules.
	if len(table.Rules) > 0 {
		if err := writeRules(&buf, schema, table.Name, table.Rules); err != nil {
			return "", err
		}
	}
	// Construct foreign keys.
	for _, fk := range table.ForeignKeys {
		if err := writeForeignKey(&buf, schema, table.Name, fk); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func GetViewDefinition(schema string, view *storepb.ViewMetadata) (string, error) {
	var buf strings.Builder
	if err := writeView(&buf, schema, view); err != nil {
		return "", err
	}
	// Construct triggers.
	for _, trigger := range view.Triggers {
		if trigger.SkipDump {
			continue
		}
		if err := writeTrigger(&buf, schema, view.Name, trigger); err != nil {
			return "", err
		}
	}
	// Construct rules (non-SELECT rules only, as SELECT rules are part of the view definition).
	var nonSelectRules []*storepb.RuleMetadata
	for _, rule := range view.Rules {
		if rule.Event != "SELECT" {
			nonSelectRules = append(nonSelectRules, rule)
		}
	}
	if len(nonSelectRules) > 0 {
		if err := writeRules(&buf, schema, view.Name, nonSelectRules); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func GetMaterializedViewDefinition(schema string, view *storepb.MaterializedViewMetadata) (string, error) {
	var buf strings.Builder
	if err := writeMaterializedView(&buf, schema, view); err != nil {
		return "", err
	}
	// Construct triggers.
	for _, trigger := range view.Triggers {
		if trigger.SkipDump {
			continue
		}
		if err := writeTrigger(&buf, schema, view.Name, trigger); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func GetFunctionDefinition(schema string, function *storepb.FunctionMetadata) (string, error) {
	var buf strings.Builder
	if err := writeFunction(&buf, schema, function); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func GetSequenceDefinition(schema string, sequence *storepb.SequenceMetadata) (string, error) {
	var buf strings.Builder
	if err := writeCreateSequence(&buf, schema, sequence); err != nil {
		return "", err
	}
	if sequence.OwnerColumn != "" && sequence.OwnerTable != "" {
		if err := writeAlterSequenceOwnedBy(&buf, schema, sequence); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func filterBackupSchemaIfNecessary(ctx schema.GetDefinitionContext, metadata *storepb.DatabaseSchemaMetadata) *storepb.DatabaseSchemaMetadata {
	if !ctx.SkipBackupSchema {
		return metadata
	}

	filtered := &storepb.DatabaseSchemaMetadata{
		Extensions:    metadata.Extensions,
		EventTriggers: metadata.EventTriggers,
	}
	for _, schema := range metadata.Schemas {
		if schema.Name == common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES) {
			continue
		}
		filtered.Schemas = append(filtered.Schemas, schema)
	}
	return filtered
}

func writeTrigger(out io.Writer, schema string, table string, trigger *storepb.TriggerMetadata) error {
	if _, err := io.WriteString(out, trigger.Body); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n\n"); err != nil {
		return err
	}

	if len(trigger.Comment) > 0 {
		if err := writeTriggerComment(out, schema, table, trigger); err != nil {
			return err
		}
	}

	return nil
}

func writeTriggerComment(out io.Writer, schema string, table string, trigger *storepb.TriggerMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON TRIGGER "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, trigger.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" ON "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, table); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(trigger.Comment)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeEventTrigger(out io.Writer, eventTrigger *storepb.EventTriggerMetadata) error {
	// Use the stored definition if available
	if eventTrigger.Definition != "" {
		if _, err := io.WriteString(out, eventTrigger.Definition); err != nil {
			return err
		}
		if _, err := io.WriteString(out, ";\n\n"); err != nil {
			return err
		}
	} else {
		// Build the CREATE EVENT TRIGGER statement
		if _, err := io.WriteString(out, `CREATE EVENT TRIGGER "`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, eventTrigger.Name); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `" ON `); err != nil {
			return err
		}
		if _, err := io.WriteString(out, eventTrigger.Event); err != nil {
			return err
		}

		// Add WHEN TAG IN clause if tags are specified
		if len(eventTrigger.Tags) > 0 {
			if _, err := io.WriteString(out, "\n  WHEN TAG IN ("); err != nil {
				return err
			}
			for i, tag := range eventTrigger.Tags {
				if i > 0 {
					if _, err := io.WriteString(out, ", "); err != nil {
						return err
					}
				}
				if _, err := io.WriteString(out, "'"); err != nil {
					return err
				}
				if _, err := io.WriteString(out, tag); err != nil {
					return err
				}
				if _, err := io.WriteString(out, "'"); err != nil {
					return err
				}
			}
			if _, err := io.WriteString(out, ")"); err != nil {
				return err
			}
		}

		// Add EXECUTE FUNCTION clause
		if _, err := io.WriteString(out, "\n  EXECUTE FUNCTION "); err != nil {
			return err
		}
		if eventTrigger.FunctionSchema != "" {
			if _, err := io.WriteString(out, `"`); err != nil {
				return err
			}
			if _, err := io.WriteString(out, eventTrigger.FunctionSchema); err != nil {
				return err
			}
			if _, err := io.WriteString(out, `".`); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, eventTrigger.FunctionName); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `"();\n\n`); err != nil {
			return err
		}
	}

	// Write event trigger comment if present
	if eventTrigger.Comment != "" {
		if err := writeEventTriggerComment(out, eventTrigger); err != nil {
			return err
		}
	}

	return nil
}

func writeEventTriggerComment(out io.Writer, eventTrigger *storepb.EventTriggerMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON EVENT TRIGGER "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, eventTrigger.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(eventTrigger.Comment)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeEnum(out io.Writer, schema string, enum *storepb.EnumTypeMetadata) error {
	if _, err := io.WriteString(out, `CREATE TYPE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, enum.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" AS ENUM (\n"); err != nil {
		return err
	}
	for i, value := range enum.Values {
		if i > 0 {
			if _, err := io.WriteString(out, ",\n"); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `    '`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, escapeSingleQuote(value)); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `'`); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(out, "\n)"); err != nil {
		return err
	}

	return nil
}

func escapeSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func writeEnumComment(out io.Writer, schema string, enum *storepb.EnumTypeMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON TYPE "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, enum.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(enum.Comment)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeMaterializedView(out io.Writer, schema string, view *storepb.MaterializedViewMetadata) error {
	if _, err := io.WriteString(out, `CREATE MATERIALIZED VIEW "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" AS \n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, strings.TrimRight(view.Definition, ";")); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n  WITH NO DATA;\n\n"); err != nil {
		return err
	}

	for _, index := range view.Indexes {
		if err := writeIndex(out, schema, view.Name, index); err != nil {
			return err
		}
	}

	if len(view.Comment) > 0 {
		if err := writeMaterializedViewComment(out, schema, view); err != nil {
			return err
		}
	}

	return nil
}

func writeMaterializedViewComment(out io.Writer, schema string, view *storepb.MaterializedViewMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON MATERIALIZED VIEW "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(view.Comment)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeView(out io.Writer, schema string, view *storepb.ViewMetadata) error {
	if _, err := io.WriteString(out, `CREATE VIEW "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" AS \n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Definition); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n\n"); err != nil {
		return err
	}

	if len(view.Comment) > 0 {
		if err := writeViewComment(out, schema, view); err != nil {
			return err
		}
	}

	return nil
}

func writeViewComment(out io.Writer, schema string, view *storepb.ViewMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON VIEW "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(view.Comment)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeRules(out io.Writer, _ string, _ string, rules []*storepb.RuleMetadata) error {
	for _, rule := range rules {
		// Write the full rule definition
		if _, err := io.WriteString(out, rule.Definition); err != nil {
			return err
		}
		if _, err := io.WriteString(out, "\n\n"); err != nil {
			return err
		}
	}
	return nil
}

func getSchemaNameFromID(id string) string {
	parts := strings.Split(id, ".")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

func writeColumnIdentityGeneration(out io.Writer, schema string, generationTypes storepb.ColumnMetadata_IdentityGeneration, sequence *storepb.SequenceMetadata) error {
	if _, err := io.WriteString(out, `ALTER TABLE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.OwnerTable); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" ALTER COLUMN "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.OwnerColumn); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" ADD GENERATED `); err != nil {
		return err
	}
	if generationTypes == storepb.ColumnMetadata_ALWAYS {
		if _, err := io.WriteString(out, "ALWAYS "); err != nil {
			return err
		}
	} else {
		if _, err := io.WriteString(out, "BY DEFAULT "); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, "AS IDENTITY (\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "    SEQUENCE NAME \""); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\"\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "    START WITH "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Start); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n    INCREMENT BY "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Increment); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n    MINVALUE "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.MinValue); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n    MAXVALUE "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.MaxValue); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n    CACHE "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.CacheSize); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\n);\n\n")
	return err
}

func writeCreateSequence(out io.Writer, schema string, sequence *storepb.SequenceMetadata) error {
	if _, err := io.WriteString(out, `CREATE SEQUENCE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\"\n    "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "AS "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.DataType); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n	START WITH "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Start); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n	INCREMENT BY "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Increment); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n	MINVALUE "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.MinValue); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n	MAXVALUE "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.MaxValue); err != nil {
		return err
	}
	if sequence.Cycle {
		if _, err := io.WriteString(out, "\n	CYCLE"); err != nil {
			return err
		}
	} else {
		if _, err := io.WriteString(out, "\n	NO CYCLE"); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, ";\n\n"); err != nil {
		return err
	}

	if len(sequence.Comment) > 0 {
		if err := writeSequenceComment(out, schema, sequence); err != nil {
			return err
		}
	}

	return nil
}

func writeSequenceComment(out io.Writer, schema string, sequence *storepb.SequenceMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON SEQUENCE "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, sequence.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(sequence.Comment)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeAlterSequenceOwnedBy(out io.Writer, schema string, sequence *storepb.SequenceMetadata) error {
	if _, err := io.WriteString(out, `ALTER SEQUENCE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" OWNED BY \""); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.OwnerTable); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.OwnerColumn); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\";\n\n")
	return err
}

func getObjectID(schema string, object string) string {
	var buf strings.Builder
	_, _ = buf.WriteString(schema)
	_, _ = buf.WriteString(".")
	_, _ = buf.WriteString(object)
	return buf.String()
}

// writePrimaryKeyConstraintSDL writes a single primary key constraint SDL
func writePrimaryKeyConstraintSDL(out io.Writer, index *storepb.IndexMetadata) error {
	if index == nil || !index.Primary {
		return errors.New("invalid primary key constraint")
	}

	if _, err := io.WriteString(out, `CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" PRIMARY KEY (`); err != nil {
		return err
	}
	for i, expression := range index.Expressions {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, expression); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, ")"); err != nil {
		return err
	}
	return nil
}

// writeUniqueKeyConstraintSDL writes a single unique key constraint SDL
func writeUniqueKeyConstraintSDL(out io.Writer, index *storepb.IndexMetadata) error {
	if index == nil || !index.Unique || index.Primary || !index.IsConstraint {
		return errors.New("invalid unique key constraint")
	}

	if _, err := io.WriteString(out, `CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" UNIQUE (`); err != nil {
		return err
	}
	for i, expression := range index.Expressions {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, expression); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, ")"); err != nil {
		return err
	}
	return nil
}

func writeCreateTable(out io.Writer, schema string, tableName string, columns []*storepb.ColumnMetadata, checks []*storepb.CheckConstraintMetadata, excludes []*storepb.ExcludeConstraintMetadata) error {
	if _, err := io.WriteString(out, `CREATE TABLE "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, tableName); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" (`); err != nil {
		return err
	}

	for i, column := range columns {
		if i > 0 {
			if _, err := io.WriteString(out, ","); err != nil {
				return err
			}
		}

		if _, err := io.WriteString(out, "\n    "); err != nil {
			return err
		}

		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}

		if _, err := io.WriteString(out, column.Name); err != nil {
			return err
		}

		if _, err := io.WriteString(out, `" `); err != nil {
			return err
		}

		if _, err := io.WriteString(out, column.Type); err != nil {
			return err
		}

		// Handle default values
		if column.Default != "" {
			if _, err := io.WriteString(out, ` DEFAULT `); err != nil {
				return err
			}
			if _, err := io.WriteString(out, column.Default); err != nil {
				return err
			}
		}

		if !column.Nullable {
			if _, err := io.WriteString(out, ` NOT NULL`); err != nil {
				return err
			}
		}
	}

	for _, check := range checks {
		_, _ = io.WriteString(out, ",\n    ")
		_, _ = io.WriteString(out, `CONSTRAINT "`)
		_, _ = io.WriteString(out, check.Name)
		_, _ = io.WriteString(out, `" CHECK `)
		_, _ = io.WriteString(out, check.Expression)
	}

	for _, exclude := range excludes {
		_, _ = io.WriteString(out, ",\n    ")
		_, _ = io.WriteString(out, `CONSTRAINT "`)
		_, _ = io.WriteString(out, exclude.Name)
		_, _ = io.WriteString(out, `" `)
		_, _ = io.WriteString(out, exclude.Expression) // Already includes "EXCLUDE USING ..."
	}

	_, err := io.WriteString(out, "\n)")
	return err
}

// isIdentityColumn checks if a column is an identity column.
func isIdentityColumn(column *storepb.ColumnMetadata) bool {
	return column.IdentityGeneration == storepb.ColumnMetadata_ALWAYS ||
		column.IdentityGeneration == storepb.ColumnMetadata_BY_DEFAULT
}

// buildIdentityColumnMap builds a map of identity columns for the entire database.
// Key format: "schemaName.tableName.columnName" -> true
// This is O(m*c) which is done once, vs O(s*m*c) when checking each sequence individually.
func buildIdentityColumnMap(metadata *storepb.DatabaseSchemaMetadata) map[string]bool {
	identityMap := make(map[string]bool)
	for _, schema := range metadata.Schemas {
		for _, table := range schema.Tables {
			for _, column := range table.Columns {
				if isIdentityColumn(column) {
					key := schema.Name + "." + table.Name + "." + column.Name
					identityMap[key] = true
				}
			}
		}
	}
	return identityMap
}

// buildIdentityColumnMapForSchema builds a map of identity columns for a single schema.
// Key format: "tableName.columnName" -> true
func buildIdentityColumnMapForSchema(schema *storepb.SchemaMetadata) map[string]bool {
	identityMap := make(map[string]bool)
	for _, table := range schema.Tables {
		for _, column := range table.Columns {
			if isIdentityColumn(column) {
				key := table.Name + "." + column.Name
				identityMap[key] = true
			}
		}
	}
	return identityMap
}

func splitSequencesByIdentityOrNot(table *storepb.TableMetadata, sequences []*storepb.SequenceMetadata) ([]storepb.ColumnMetadata_IdentityGeneration, []*storepb.SequenceMetadata, []*storepb.SequenceMetadata) {
	columnMap := make(map[string]*storepb.ColumnMetadata)
	for _, column := range table.Columns {
		columnMap[column.Name] = column
	}
	var generationType []storepb.ColumnMetadata_IdentityGeneration
	var identitySequences []*storepb.SequenceMetadata
	var nonIdentitySequences []*storepb.SequenceMetadata
	for _, sequence := range sequences {
		if column, ok := columnMap[sequence.OwnerColumn]; ok {
			if isIdentityColumn(column) {
				generationType = append(generationType, column.IdentityGeneration)
				identitySequences = append(identitySequences, sequence)
				continue
			}
		}
		nonIdentitySequences = append(nonIdentitySequences, sequence)
	}
	return generationType, identitySequences, nonIdentitySequences
}

func writeTable(out io.Writer, schema string, table *storepb.TableMetadata, sequences []*storepb.SequenceMetadata) error {
	generationTypes, identitySequences, nonIdentitySequences := splitSequencesByIdentityOrNot(table, sequences)

	if err := writeCreateTable(out, schema, table.Name, table.Columns, table.CheckConstraints, table.ExcludeConstraints); err != nil {
		return err
	}

	if len(table.Partitions) > 0 {
		if err := writePartitionClause(out, table.Partitions[0]); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(out, ";\n\n"); err != nil {
		return err
	}

	for _, sequence := range nonIdentitySequences {
		if err := writeAlterSequenceOwnedBy(out, schema, sequence); err != nil {
			return err
		}
	}

	// Construct comments.
	if len(table.Comment) > 0 {
		if err := writeTableComment(out, schema, table); err != nil {
			return err
		}
	}

	for _, column := range table.Columns {
		if len(column.Comment) > 0 {
			if err := writeColumnComment(out, schema, table.Name, column); err != nil {
				return err
			}
		}
	}

	for i, sequence := range identitySequences {
		if err := writeColumnIdentityGeneration(out, schema, generationTypes[i], sequence); err != nil {
			return err
		}
	}

	// Construct partition tables.
	for _, partition := range table.Partitions {
		if err := writePartitionTable(out, schema, table.Columns, partition); err != nil {
			return err
		}
	}

	for _, partition := range table.Partitions {
		if err := writeAttachPartition(out, schema, table.Name, partition); err != nil {
			return err
		}
	}

	// Construct Primary Key.
	for _, index := range table.Indexes {
		if index.Primary {
			if err := writePrimaryKey(out, schema, table.Name, index); err != nil {
				return err
			}
		}
	}

	// Construct Partition table primary key.
	for _, partition := range table.Partitions {
		if err := writePartitionPrimaryKey(out, schema, partition); err != nil {
			return err
		}
	}

	// Construct Unique Key.
	for _, index := range table.Indexes {
		if index.Unique && !index.Primary && index.IsConstraint {
			if err := writeUniqueKey(out, schema, table.Name, index); err != nil {
				return err
			}
		}
	}

	// Construct Partition table unique key.
	for _, partition := range table.Partitions {
		if err := writePartitionUniqueKey(out, schema, partition); err != nil {
			return err
		}
	}

	// Construct Index.
	for _, index := range table.Indexes {
		if !index.Primary && !index.IsConstraint {
			if err := writeIndex(out, schema, table.Name, index); err != nil {
				return err
			}
		}
	}

	// Construct Partition table index.
	for _, partition := range table.Partitions {
		if err := writePartitionIndex(out, schema, partition); err != nil {
			return err
		}
	}

	// Construct index attach partition.
	for _, partition := range table.Partitions {
		if err := writeAttachPartitionIndex(out, schema, partition); err != nil {
			return err
		}
	}

	return nil
}

func writeForeignKey(out io.Writer, schema string, table string, fk *storepb.ForeignKeyMetadata) error {
	if _, err := io.WriteString(out, `ALTER TABLE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\"\n    ADD CONSTRAINT \""); err != nil {
		return err
	}
	if _, err := io.WriteString(out, fk.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" FOREIGN KEY ("); err != nil {
		return err
	}
	for i, column := range fk.Columns {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, column); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, ")\n    REFERENCES \""); err != nil {
		return err
	}
	if _, err := io.WriteString(out, fk.ReferencedSchema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, fk.ReferencedTable); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" ("); err != nil {
		return err
	}
	for i, column := range fk.ReferencedColumns {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, column); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, ")"); err != nil {
		return err
	}

	// Add ON DELETE clause if specified
	if fk.OnDelete != "" && fk.OnDelete != "NO ACTION" {
		if _, err := io.WriteString(out, "\n    ON DELETE "); err != nil {
			return err
		}
		if _, err := io.WriteString(out, fk.OnDelete); err != nil {
			return err
		}
	}

	// Add ON UPDATE clause if specified
	if fk.OnUpdate != "" && fk.OnUpdate != "NO ACTION" {
		if _, err := io.WriteString(out, "\n    ON UPDATE "); err != nil {
			return err
		}
		if _, err := io.WriteString(out, fk.OnUpdate); err != nil {
			return err
		}
	}

	_, err := io.WriteString(out, ";\n\n")
	return err
}

func writeAttachPartitionIndex(out io.Writer, schema string, partition *storepb.TablePartitionMetadata) error {
	for _, index := range partition.Indexes {
		if err := writeAttachIndex(out, schema, index); err != nil {
			return err
		}
	}

	for _, subpartition := range partition.Subpartitions {
		if err := writeAttachPartitionIndex(out, schema, subpartition); err != nil {
			return err
		}
	}
	return nil
}

func writeAttachIndex(out io.Writer, schema string, index *storepb.IndexMetadata) error {
	if len(index.ParentIndexName) == 0 || len(index.ParentIndexSchema) == 0 {
		return nil
	}

	if _, err := io.WriteString(out, `ALTER INDEX "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, index.ParentIndexSchema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, index.ParentIndexName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" ATTACH PARTITION "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\";\n\n")
	return err
}

func writePartitionIndex(out io.Writer, schema string, partition *storepb.TablePartitionMetadata) error {
	for _, index := range partition.Indexes {
		if !index.IsConstraint && !index.Primary {
			if err := writeIndex(out, schema, partition.Name, index); err != nil {
				return err
			}
		}
	}

	for _, subpartition := range partition.Subpartitions {
		if err := writePartitionIndex(out, schema, subpartition); err != nil {
			return err
		}
	}
	return nil
}

func writeIndex(out io.Writer, schema string, table string, index *storepb.IndexMetadata) error {
	return writeIndexInternal(out, schema, table, index, true, true)
}

func writeIndexKeyList(out io.Writer, index *storepb.IndexMetadata) error {
	if _, err := io.WriteString(out, `(`); err != nil {
		return err
	}

	for i, expression := range index.Expressions {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}

		if _, err := io.WriteString(out, expression); err != nil {
			return err
		}

		// Add opclass if it's not the default
		if i < len(index.OpclassNames) && i < len(index.OpclassDefaults) {
			if !index.OpclassDefaults[i] && index.OpclassNames[i] != "" {
				if _, err := io.WriteString(out, " "); err != nil {
					return err
				}
				if _, err := io.WriteString(out, index.OpclassNames[i]); err != nil {
					return err
				}
			}
		}

		// Add DESC if this column is marked as descending
		if i < len(index.Descending) && index.Descending[i] {
			if _, err := io.WriteString(out, " DESC"); err != nil {
				return err
			}
		}
		// Note: NULLS ordering information is not available in IndexMetadata
		// so we omit it in the generated DDL
	}

	_, err := io.WriteString(out, ")")
	return err
}

func writeIndexComment(out io.Writer, schema string, index *storepb.IndexMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON INDEX "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(index.Comment)); err != nil {
		return err
	}

	_, err := io.WriteString(out, `';\n\n`)
	return err
}

func writePartitionUniqueKey(out io.Writer, schema string, partition *storepb.TablePartitionMetadata) error {
	for _, index := range partition.Indexes {
		if index.Unique && !index.Primary && index.IsConstraint {
			if err := writeUniqueKey(out, schema, partition.Name, index); err != nil {
				return err
			}
		}
	}

	for _, subpartition := range partition.Subpartitions {
		if err := writePartitionUniqueKey(out, schema, subpartition); err != nil {
			return err
		}
	}
	return nil
}

func writeUniqueKey(out io.Writer, schema string, table string, index *storepb.IndexMetadata) error {
	if _, err := io.WriteString(out, `ALTER TABLE ONLY "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" ADD CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" UNIQUE (`); err != nil {
		return err
	}
	for i, expression := range index.Expressions {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, expression); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, ");\n\n"); err != nil {
		return err
	}

	if len(index.Comment) > 0 {
		if err := writeConstraintComment(out, schema, table, index); err != nil {
			return err
		}
	}

	return nil
}

func writePartitionPrimaryKey(out io.Writer, schema string, partition *storepb.TablePartitionMetadata) error {
	for _, index := range partition.Indexes {
		if index.Primary {
			if err := writePrimaryKey(out, schema, partition.Name, index); err != nil {
				return err
			}
		}
	}

	for _, subpartition := range partition.Subpartitions {
		if err := writePartitionPrimaryKey(out, schema, subpartition); err != nil {
			return err
		}
	}
	return nil
}

func writePrimaryKey(out io.Writer, schema string, table string, index *storepb.IndexMetadata) error {
	if _, err := io.WriteString(out, `ALTER TABLE ONLY "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" ADD CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" PRIMARY KEY (`); err != nil {
		return err
	}
	for i, expression := range index.Expressions {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, expression); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, ");\n\n"); err != nil {
		return err
	}

	if len(index.Comment) > 0 {
		if err := writeConstraintComment(out, schema, table, index); err != nil {
			return err
		}
	}

	return nil
}

func writeConstraintComment(out io.Writer, schema string, table string, index *storepb.IndexMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON CONSTRAINT "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" ON "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, table); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(index.Comment)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeColumnComment(out io.Writer, schema string, table string, column *storepb.ColumnMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON COLUMN "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, column.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, escapeSingleQuote(column.Comment)); err != nil {
		return err
	}
	_, err := io.WriteString(out, "';\n\n")
	return err
}

func writeTableComment(out io.Writer, schema string, table *storepb.TableMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON TABLE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, escapeSingleQuote(table.Comment)); err != nil {
		return err
	}
	_, err := io.WriteString(out, "';\n\n")
	return err
}

func writePartitionClause(out io.Writer, partition *storepb.TablePartitionMetadata) error {
	if _, err := io.WriteString(out, " PARTITION BY "); err != nil {
		return err
	}
	_, err := io.WriteString(out, partition.Expression)
	return err
}

func writeAttachPartition(out io.Writer, schema string, tableName string, partition *storepb.TablePartitionMetadata) error {
	if _, err := io.WriteString(out, `ALTER TABLE ONLY "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, tableName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" ATTACH PARTITION "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, partition.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, partition.Value); err != nil {
		return err
	}
	_, err := io.WriteString(out, ";\n\n")
	return err
}

func writePartitionTable(out io.Writer, schema string, columns []*storepb.ColumnMetadata, partition *storepb.TablePartitionMetadata) error {
	if err := writeCreateTable(out, schema, partition.Name, columns, partition.CheckConstraints, partition.ExcludeConstraints); err != nil {
		return err
	}

	if len(partition.Subpartitions) > 0 {
		if err := writePartitionClause(out, partition.Subpartitions[0]); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(out, ";\n\n"); err != nil {
		return err
	}

	// Construct subpartition tables.
	for _, subpartition := range partition.Subpartitions {
		if err := writePartitionTable(out, schema, columns, subpartition); err != nil {
			return err
		}
	}

	for _, subpartition := range partition.Subpartitions {
		if err := writeAttachPartition(out, schema, partition.Name, subpartition); err != nil {
			return err
		}
	}

	return nil
}

func writeFunction(out io.Writer, schema string, function *storepb.FunctionMetadata) error {
	if _, err := io.WriteString(out, function.Definition); err != nil {
		return err
	}

	if _, err := io.WriteString(out, ";\n\n"); err != nil {
		return err
	}

	if len(function.Comment) > 0 {
		if err := writeFunctionComment(out, schema, function); err != nil {
			return err
		}
	}

	return nil
}

// isDefinitionProcedure checks if the definition string represents a PROCEDURE (not a FUNCTION)
// Returns true if it's a PROCEDURE, false if it's a FUNCTION
// This function uses AST-based parsing for robust detection
func isDefinitionProcedure(definition string) bool {
	if definition == "" {
		return false
	}

	// Parse the definition to get AST
	parseResults, err := pgparser.ParsePostgreSQL(definition)
	if err != nil {
		// If parsing fails, fall back to string-based detection
		// This should rarely happen for valid definitions
		upperDef := strings.ToUpper(definition)
		return strings.Contains(upperDef, " PROCEDURE ") ||
			strings.HasPrefix(upperDef, "CREATE PROCEDURE") ||
			strings.HasPrefix(upperDef, "CREATE OR REPLACE PROCEDURE")
	}

	// For function/procedure definition, we expect exactly one statement
	if len(parseResults) != 1 {
		return false
	}

	tree := parseResults[0].Tree
	if tree == nil {
		return false
	}

	// Walk the AST to find CREATE FUNCTION/PROCEDURE statement
	var result *parser.CreatefunctionstmtContext
	extractor := &functionExtractor{result: &result}
	antlr.NewParseTreeWalker().Walk(extractor, tree)

	if result == nil {
		return false
	}

	// Check if it's a PROCEDURE by examining the AST node
	// CreatefunctionstmtContext has both FUNCTION() and PROCEDURE() methods
	return result.PROCEDURE() != nil
}

func writeFunctionComment(out io.Writer, schema string, function *storepb.FunctionMetadata) error {
	// Determine if this is a PROCEDURE or FUNCTION by checking the definition
	objectType := "FUNCTION"
	if isDefinitionProcedure(function.Definition) {
		objectType = "PROCEDURE"
	}

	if _, err := io.WriteString(out, "COMMENT ON "+objectType+" \""); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `".`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, function.Signature); err != nil {
		return err
	}

	if _, err := io.WriteString(out, ` IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(function.Comment)); err != nil {
		return err
	}

	_, err := io.WriteString(out, "';\n\n")
	return err
}

func writeExtension(out io.Writer, extension *storepb.ExtensionMetadata) error {
	if _, err := io.WriteString(out, `CREATE EXTENSION IF NOT EXISTS "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, extension.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"`); err != nil {
		return err
	}

	// Add WITH SCHEMA clause if schema is specified and not empty
	if extension.Schema != "" {
		if _, err := io.WriteString(out, ` WITH SCHEMA "`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, extension.Schema); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
	}

	// Add VERSION clause if version is specified
	if extension.Version != "" {
		if _, err := io.WriteString(out, ` VERSION '`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, extension.Version); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `'`); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(out, `;`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, "\n"); err != nil {
		return err
	}

	// Write extension description (comment) if present
	if extension.Description != "" {
		if _, err := io.WriteString(out, "\n"); err != nil {
			return err
		}
		if err := writeExtensionComment(out, extension); err != nil {
			return err
		}
	} else {
		// Add blank line even if no description
		if _, err := io.WriteString(out, "\n"); err != nil {
			return err
		}
	}

	return nil
}

func writeExtensionComment(out io.Writer, extension *storepb.ExtensionMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON EXTENSION "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, extension.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, escapeSingleQuote(extension.Description)); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func getSDLFormat(metadata *storepb.DatabaseSchemaMetadata) (string, error) {
	var buf strings.Builder

	// Write CREATE SCHEMA statements first for non-public schemas
	for _, schema := range metadata.Schemas {
		if schema.SkipDump {
			continue
		}
		if err := writeSchema(&buf, schema); err != nil {
			return "", err
		}

		// Write schema comment if present
		if len(schema.Comment) > 0 {
			if err := writeSchemaCommentSDL(&buf, schema); err != nil {
				return "", err
			}
		}
	}

	// Write extensions (before enum types and tables as they might provide types used in definitions)
	for _, extension := range metadata.Extensions {
		if err := writeExtension(&buf, extension); err != nil {
			return "", err
		}
	}

	// Build a map of serial and identity sequences that should be skipped
	// Serial sequences are identified by checking if their owner columns match the serial pattern
	// Identity sequences are identified by checking if their owner columns have IdentityGeneration set
	skipSequences := make(map[string]bool)
	for _, schema := range metadata.Schemas {
		if schema.SkipDump {
			continue
		}
		for _, table := range schema.Tables {
			if table.SkipDump {
				continue
			}
			for _, column := range table.Columns {
				// Check for serial columns
				isSerial, _ := isSerialColumn(column, table.Name, schema.Sequences)
				if isSerial {
					// Extract the sequence name from the DEFAULT clause to match the exact sequence
					// This ensures we skip the correct sequence, especially when multiple sequences
					// claim ownership of the same column.
					sequenceName := extractSequenceNameFromNextval(column.Default)

					// Find the sequence that belongs to this serial column
					for _, sequence := range schema.Sequences {
						// Match by sequence name AND ownership to ensure we skip the exact sequence
						// referenced in the DEFAULT clause
						if sequence.Name == sequenceName && sequence.OwnerTable == table.Name && sequence.OwnerColumn == column.Name {
							sequenceKey := schema.Name + "." + sequence.Name
							skipSequences[sequenceKey] = true
							break
						}
					}
				}
				// Check for identity columns
				if isIdentityColumn(column) {
					// Find the sequence that belongs to this identity column
					for _, sequence := range schema.Sequences {
						if sequence.OwnerTable == table.Name && sequence.OwnerColumn == column.Name {
							sequenceKey := schema.Name + "." + sequence.Name
							skipSequences[sequenceKey] = true
							break
						}
					}
				}
			}
		}
	}

	// Write all enum types before sequences and tables
	for _, schema := range metadata.Schemas {
		if schema.SkipDump {
			continue
		}
		for _, enumType := range schema.EnumTypes {
			if enumType.SkipDump {
				continue
			}
			if err := writeEnum(&buf, schema.Name, enumType); err != nil {
				return "", err
			}
			if _, err := buf.WriteString(";\n\n"); err != nil {
				return "", err
			}

			// Write enum type comment if present
			if len(enumType.Comment) > 0 {
				if err := writeEnumComment(&buf, schema.Name, enumType); err != nil {
					return "", err
				}
			}
		}
	}

	// Write all sequences before tables to ensure they exist before any table references them.
	// Skip sequences that belong to serial or identity columns as they will be implicitly created.
	sequenceOwnershipMap := make(map[string][]*storepb.SequenceMetadata)

	for _, schema := range metadata.Schemas {
		if schema.SkipDump {
			continue
		}

		for _, sequence := range schema.Sequences {
			if sequence.SkipDump {
				continue
			}
			// Skip sequences that belong to serial or identity columns
			sequenceKey := schema.Name + "." + sequence.Name
			if skipSequences[sequenceKey] {
				continue
			}
			if err := writeSequenceSDL(&buf, schema.Name, sequence); err != nil {
				return "", err
			}
			if _, err := buf.WriteString(";\n\n"); err != nil {
				return "", err
			}

			// Write sequence comment if present
			if len(sequence.Comment) > 0 {
				if err := writeSequenceCommentSDL(&buf, schema.Name, sequence); err != nil {
					return "", err
				}
			}

			// Track sequences with owners for later ownership establishment
			if sequence.OwnerTable != "" && sequence.OwnerColumn != "" {
				tableID := getObjectID(schema.Name, sequence.OwnerTable)
				sequenceOwnershipMap[tableID] = append(sequenceOwnershipMap[tableID], sequence)
			}
		}
	}

	// Build a map of sequences by table for easy lookup during table creation
	tableSequencesMap := make(map[string][]*storepb.SequenceMetadata)
	for _, schema := range metadata.Schemas {
		if schema.SkipDump {
			continue
		}
		for _, sequence := range schema.Sequences {
			if sequence.SkipDump {
				continue
			}
			if sequence.OwnerTable != "" {
				tableKey := schema.Name + "." + sequence.OwnerTable
				tableSequencesMap[tableKey] = append(tableSequencesMap[tableKey], sequence)
			}
		}
	}

	for _, schema := range metadata.Schemas {
		if schema.SkipDump {
			continue
		}

		// Write tables (sequences are already written above)
		for _, table := range schema.Tables {
			if table.SkipDump {
				continue
			}

			// Get sequences for this table
			tableKey := schema.Name + "." + table.Name
			tableSequences := tableSequencesMap[tableKey]

			// Write CREATE TABLE statement
			if err := writeCreateTableSDL(&buf, schema.Name, table, tableSequences); err != nil {
				return "", err
			}

			if _, err := buf.WriteString(";\n\n"); err != nil {
				return "", err
			}

			// Write table comment if present
			if len(table.Comment) > 0 {
				if err := writeTableCommentSDL(&buf, schema.Name, table); err != nil {
					return "", err
				}
			}

			// Write column comments if present
			for _, column := range table.Columns {
				if len(column.Comment) > 0 {
					if err := writeColumnCommentSDL(&buf, schema.Name, table.Name, column); err != nil {
						return "", err
					}
				}
			}

			// Write CREATE INDEX statements for non-constraint indexes
			if err := writeIndexesSDL(&buf, schema.Name, table); err != nil {
				return "", err
			}

			// Write index comments if present
			for _, index := range table.Indexes {
				// Only write comment for standalone indexes (not primary key or unique constraint indexes)
				if !index.Primary && !index.Unique {
					if len(index.Comment) > 0 {
						if err := writeIndexCommentSDL(&buf, schema.Name, index); err != nil {
							return "", err
						}
					}
				}
			}

			// Write CREATE TRIGGER statements
			if err := writeTriggersSDL(&buf, schema.Name, table); err != nil {
				return "", err
			}

			// Write trigger comments if present
			for _, trigger := range table.Triggers {
				if len(trigger.Comment) > 0 {
					if err := writeTriggerCommentSDL(&buf, schema.Name, table.Name, trigger); err != nil {
						return "", err
					}
				}
			}
		}

		// Write ALTER SEQUENCE OWNED BY statements for sequences that have owners
		// but are not serial or identity sequences
		for _, sequence := range schema.Sequences {
			if sequence.SkipDump {
				continue
			}
			// Skip sequences that belong to serial or identity columns
			sequenceKey := schema.Name + "." + sequence.Name
			if skipSequences[sequenceKey] {
				continue
			}
			// Write ALTER SEQUENCE OWNED BY for sequences with owners
			if sequence.OwnerTable != "" && sequence.OwnerColumn != "" {
				if err := writeAlterSequenceOwnedBy(&buf, schema.Name, sequence); err != nil {
					return "", err
				}
			}
		}

		// Write views after tables
		for _, view := range schema.Views {
			if view.SkipDump {
				continue
			}

			if err := writeViewSDL(&buf, schema.Name, view); err != nil {
				return "", err
			}

			if _, err := buf.WriteString(";\n\n"); err != nil {
				return "", err
			}

			// Write view comment if present
			if len(view.Comment) > 0 {
				if err := writeViewCommentSDL(&buf, schema.Name, view); err != nil {
					return "", err
				}
			}
		}

		// Write materialized views after views
		for _, materializedView := range schema.MaterializedViews {
			if materializedView.SkipDump {
				continue
			}

			if err := writeMaterializedViewSDL(&buf, schema.Name, materializedView); err != nil {
				return "", err
			}

			if _, err := buf.WriteString(";\n\n"); err != nil {
				return "", err
			}

			// Write materialized view comment if present
			if len(materializedView.Comment) > 0 {
				if err := writeMaterializedViewCommentSDL(&buf, schema.Name, materializedView); err != nil {
					return "", err
				}
			}

			// Write indexes on materialized view
			for _, index := range materializedView.Indexes {
				// Skip constraint-based indexes as they are part of table definition
				if index.Primary || index.IsConstraint {
					continue
				}

				if err := writeIndexSDL(&buf, schema.Name, materializedView.Name, index); err != nil {
					return "", err
				}
				if _, err := buf.WriteString(";\n\n"); err != nil {
					return "", err
				}

				// Write index comment if present
				if len(index.Comment) > 0 {
					if err := writeIndexCommentSDL(&buf, schema.Name, index); err != nil {
						return "", err
					}
				}
			}
		}

		// Write functions and procedures after materialized views
		for _, function := range schema.Functions {
			if function.SkipDump {
				continue
			}

			if err := writeFunctionSDL(&buf, schema.Name, function); err != nil {
				return "", err
			}

			if _, err := buf.WriteString(";\n\n"); err != nil {
				return "", err
			}

			// Write function comment if present
			if len(function.Comment) > 0 {
				if err := writeFunctionCommentSDL(&buf, schema.Name, function); err != nil {
					return "", err
				}
			}
		}
	}

	// Write event triggers (last, as they depend on functions)
	for _, eventTrigger := range metadata.EventTriggers {
		if eventTrigger.SkipDump {
			continue
		}
		if err := writeEventTrigger(&buf, eventTrigger); err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}

// isSerialColumn checks if a column is a serial/bigserial/smallserial type based on its properties
// Serial pattern: integer/bigint/smallint + NOT NULL + nextval(sequence) default
// extractSequenceNameFromNextval extracts the sequence name from a nextval() default value.
// It handles various PostgreSQL formats including:
// - nextval('sequence_name')
// - nextval('schema.sequence_name')
// - nextval('"Schema"."Sequence"')
// - nextval('public."MixedCase"')
// - nextval('sequence_name'::regclass)
// Returns the unquoted sequence name without schema qualification.
func extractSequenceNameFromNextval(defaultValue string) string {
	if defaultValue == "" {
		return ""
	}

	// Find nextval( - case insensitive
	defaultLower := strings.ToLower(defaultValue)
	nextvalIdx := strings.Index(defaultLower, "nextval(")
	if nextvalIdx == -1 {
		return ""
	}

	// Find the opening quote after nextval(
	startIdx := nextvalIdx + len("nextval(")
	if startIdx >= len(defaultValue) {
		return ""
	}

	// Determine quote type (' or ")
	quoteChar := byte(0)
	for startIdx < len(defaultValue) && (defaultValue[startIdx] == ' ' || defaultValue[startIdx] == '\t') {
		startIdx++
	}
	if startIdx >= len(defaultValue) {
		return ""
	}

	switch defaultValue[startIdx] {
	case '\'':
		quoteChar = '\''
	case '"':
		quoteChar = '"'
	default:
		return ""
	}

	startIdx++ // Skip opening quote

	// Find the closing quote
	endIdx := startIdx
	for endIdx < len(defaultValue) && defaultValue[endIdx] != quoteChar {
		endIdx++
	}
	if endIdx >= len(defaultValue) {
		return ""
	}

	// Extract the full sequence reference (may include schema and ::regclass)
	sequenceRef := defaultValue[startIdx:endIdx]

	// Remove ::regclass suffix if present
	if idx := strings.Index(sequenceRef, "::"); idx != -1 {
		sequenceRef = sequenceRef[:idx]
	}

	// Parse schema-qualified name: schema.sequence or "schema"."sequence"
	// We need to extract just the sequence name, handling quoted identifiers
	sequenceName := extractIdentifierFromQualifiedName(sequenceRef)

	// Remove surrounding quotes from the sequence name if present
	sequenceName = strings.TrimSpace(sequenceName)
	if len(sequenceName) >= 2 && sequenceName[0] == '"' && sequenceName[len(sequenceName)-1] == '"' {
		sequenceName = sequenceName[1 : len(sequenceName)-1]
	}

	return sequenceName
}

// extractIdentifierFromQualifiedName extracts the identifier (sequence name) from a qualified name.
// Handles cases like:
// - sequence_name -> sequence_name
// - schema.sequence_name -> sequence_name
// - "Schema"."Sequence" -> "Sequence"
// - public."MixedCase" -> "MixedCase"
func extractIdentifierFromQualifiedName(qualifiedName string) string {
	if qualifiedName == "" {
		return ""
	}

	// Look for the last dot that's not inside quotes
	inQuotes := false
	lastDotIdx := -1

	for i := 0; i < len(qualifiedName); i++ {
		if qualifiedName[i] == '"' {
			inQuotes = !inQuotes
		} else if qualifiedName[i] == '.' && !inQuotes {
			lastDotIdx = i
		}
	}

	// If we found a dot outside quotes, take everything after it
	if lastDotIdx != -1 {
		return qualifiedName[lastDotIdx+1:]
	}

	// No schema qualification found
	return qualifiedName
}

func isSerialColumn(column *storepb.ColumnMetadata, tableName string, sequences []*storepb.SequenceMetadata) (isSerial bool, serialType string) {
	// Serial columns must be NOT NULL
	if column.Nullable {
		return false, ""
	}

	// Serial columns must have a nextval() default
	if !strings.Contains(strings.ToLower(column.Default), "nextval(") {
		return false, ""
	}

	// Check type and map to serial type
	var expectedSerialType string
	switch strings.ToLower(column.Type) {
	case "integer":
		expectedSerialType = "serial"
	case "bigint":
		expectedSerialType = "bigserial"
	case "smallint":
		expectedSerialType = "smallserial"
	default:
		return false, ""
	}

	// IMPORTANT: Only treat as serial if the sequence is owned by this column
	// This prevents converting columns that use independent sequences to serial type
	sequenceName := extractSequenceNameFromNextval(column.Default)
	if sequenceName == "" {
		return false, ""
	}

	// Check if any sequence with this name is owned by this column
	for _, seq := range sequences {
		if seq.Name == sequenceName && seq.OwnerTable == tableName && seq.OwnerColumn == column.Name {
			return true, expectedSerialType
		}
	}

	// Sequence is not owned by this column, so don't convert to serial
	return false, ""
}

// writeColumnSDL writes a single column definition to the output writer
// This function is extracted from writeCreateTableSDL to enable code reuse
// sequences parameter is optional and used to find identity sequences for the column
// tableName is used to verify sequence ownership for serial column detection
func writeColumnSDL(out io.Writer, column *storepb.ColumnMetadata, tableName string, sequences []*storepb.SequenceMetadata) error {
	if _, err := io.WriteString(out, `"`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, column.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" `); err != nil {
		return err
	}

	// Check if this is an identity column
	if isIdentityColumn(column) {
		// Find the sequence for this identity column
		var identitySequence *storepb.SequenceMetadata
		for _, seq := range sequences {
			if seq.OwnerColumn == column.Name {
				identitySequence = seq
				break
			}
		}

		// Write type
		if _, err := io.WriteString(out, column.Type); err != nil {
			return err
		}

		// Write GENERATED ... AS IDENTITY
		if _, err := io.WriteString(out, " GENERATED "); err != nil {
			return err
		}
		if column.IdentityGeneration == storepb.ColumnMetadata_ALWAYS {
			if _, err := io.WriteString(out, "ALWAYS"); err != nil {
				return err
			}
		} else {
			if _, err := io.WriteString(out, "BY DEFAULT"); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, " AS IDENTITY"); err != nil {
			return err
		}

		// Write identity options if we have the sequence
		if identitySequence != nil {
			hasOptions := false
			// Check if we need to write options (non-default values)
			if identitySequence.Start != "" && identitySequence.Start != "1" {
				hasOptions = true
			}
			if identitySequence.Increment != "" && identitySequence.Increment != "1" {
				hasOptions = true
			}

			if hasOptions {
				if _, err := io.WriteString(out, " ("); err != nil {
					return err
				}

				first := true
				if identitySequence.Start != "" && identitySequence.Start != "1" {
					if _, err := fmt.Fprintf(out, "START WITH %s", identitySequence.Start); err != nil {
						return err
					}
					first = false
				}
				if identitySequence.Increment != "" && identitySequence.Increment != "1" {
					if !first {
						if _, err := io.WriteString(out, " "); err != nil {
							return err
						}
					}
					if _, err := fmt.Fprintf(out, "INCREMENT BY %s", identitySequence.Increment); err != nil {
						return err
					}
				}

				if _, err := io.WriteString(out, ")"); err != nil {
					return err
				}
			}
		}

		// Identity columns are implicitly NOT NULL, skip that
		return nil
	}

	// Check if this is a serial column and convert it back to serial type
	isSerial, serialType := isSerialColumn(column, tableName, sequences)
	if isSerial {
		// Use serial type instead of integer + nextval + NOT NULL
		if _, err := io.WriteString(out, serialType); err != nil {
			return err
		}
		// Skip DEFAULT and NOT NULL as they are implicit in serial
	} else {
		// Normal column handling
		if _, err := io.WriteString(out, column.Type); err != nil {
			return err
		}

		if column.Default != "" {
			if _, err := io.WriteString(out, ` DEFAULT `); err != nil {
				return err
			}
			if _, err := io.WriteString(out, column.Default); err != nil {
				return err
			}
		}

		if !column.Nullable {
			if _, err := io.WriteString(out, ` NOT NULL`); err != nil {
				return err
			}
		}
	}

	if column.Collation != "" {
		if _, err := io.WriteString(out, ` COLLATE "`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, column.Collation); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
	}

	return nil
}

func writeCreateTableSDL(out io.Writer, schemaName string, table *storepb.TableMetadata, sequences []*storepb.SequenceMetadata) error {
	if _, err := io.WriteString(out, `CREATE TABLE "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schemaName); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, table.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" (`); err != nil {
		return err
	}

	// Write columns
	for i, column := range table.Columns {
		if i > 0 {
			if _, err := io.WriteString(out, ","); err != nil {
				return err
			}
		}

		if _, err := io.WriteString(out, "\n    "); err != nil {
			return err
		}

		// Use the extracted writeColumnSDL function
		if err := writeColumnSDL(out, column, table.Name, sequences); err != nil {
			return err
		}
	}

	// Write table constraints
	if err := writeTableConstraintsSDL(out, table); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n)")
	return err
}

func writeTableConstraintsSDL(out io.Writer, table *storepb.TableMetadata) error {
	// Write primary key constraint
	for _, index := range table.Indexes {
		if index.Primary {
			if _, err := io.WriteString(out, ",\n    "); err != nil {
				return err
			}
			if err := writePrimaryKeyConstraintSDL(out, index); err != nil {
				return err
			}
		}
	}

	// Write unique constraints
	for _, index := range table.Indexes {
		if index.Unique && !index.Primary && index.IsConstraint {
			if _, err := io.WriteString(out, ",\n    "); err != nil {
				return err
			}
			if err := writeUniqueKeyConstraintSDL(out, index); err != nil {
				return err
			}
		}
	}

	// Write check constraints
	for _, check := range table.CheckConstraints {
		if _, err := io.WriteString(out, ",\n    "); err != nil {
			return err
		}
		if err := writeCheckConstraintSDL(out, check); err != nil {
			return err
		}
	}

	// Write EXCLUDE constraints
	for _, exclude := range table.ExcludeConstraints {
		if _, err := io.WriteString(out, ",\n    "); err != nil {
			return err
		}
		if err := writeExcludeConstraintSDL(out, exclude); err != nil {
			return err
		}
	}

	// Write foreign key constraints
	for _, fk := range table.ForeignKeys {
		if _, err := io.WriteString(out, ",\n    "); err != nil {
			return err
		}
		if err := writeForeignKeyConstraintSDL(out, fk); err != nil {
			return err
		}
	}

	return nil
}

func writeIndexesSDL(out io.Writer, schemaName string, table *storepb.TableMetadata) error {
	for _, index := range table.Indexes {
		// Skip indexes that are constraints (primary key, unique constraints)
		// These are already handled in the CREATE TABLE statement
		if index.Primary || index.IsConstraint {
			continue
		}

		if err := writeIndexSDL(out, schemaName, table.Name, index); err != nil {
			return err
		}

		if _, err := io.WriteString(out, ";\n\n"); err != nil {
			return err
		}
	}
	return nil
}

func writeIndexSDL(out io.Writer, schemaName string, tableName string, index *storepb.IndexMetadata) error {
	return writeIndexInternal(out, schemaName, tableName, index, true, false)
}

// writeIndexInternal is the core index writing function with options for different modes
func writeIndexInternal(out io.Writer, schema string, table string, index *storepb.IndexMetadata, useOnlyClause bool, includeTerminatorAndComment bool) error {
	if index.Unique {
		if _, err := io.WriteString(out, `CREATE UNIQUE INDEX "`); err != nil {
			return err
		}
	} else {
		if _, err := io.WriteString(out, `CREATE INDEX "`); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}

	if useOnlyClause {
		if _, err := io.WriteString(out, `" ON ONLY "`); err != nil {
			return err
		}
	} else {
		if _, err := io.WriteString(out, `" ON "`); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"`); err != nil {
		return err
	}

	// Add USING clause for non-btree indexes
	if index.Type != "" && index.Type != "btree" {
		if _, err := fmt.Fprintf(out, " USING %s", strings.ToUpper(index.Type)); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(out, ` `); err != nil {
		return err
	}
	if err := writeIndexKeyList(out, index); err != nil {
		return err
	}

	if includeTerminatorAndComment {
		if _, err := io.WriteString(out, ";\n\n"); err != nil {
			return err
		}

		if len(index.Comment) > 0 {
			if err := writeIndexComment(out, schema, index); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeViewSDL(out io.Writer, schemaName string, view *storepb.ViewMetadata) error {
	if _, err := io.WriteString(out, `CREATE VIEW "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schemaName); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" AS `); err != nil {
		return err
	}

	// The view definition should already include the SELECT statement
	definition := strings.TrimSpace(view.Definition)
	// Remove trailing semicolon if present
	definition = strings.TrimSuffix(definition, ";")

	_, err := io.WriteString(out, definition)
	return err
}

func writeMaterializedViewSDL(out io.Writer, schemaName string, view *storepb.MaterializedViewMetadata) error {
	if _, err := io.WriteString(out, `CREATE MATERIALIZED VIEW "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schemaName); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" AS `); err != nil {
		return err
	}

	// The materialized view definition should already include the SELECT statement
	definition := strings.TrimSpace(view.Definition)
	// Remove trailing semicolon if present
	definition = strings.TrimSuffix(definition, ";")

	_, err := io.WriteString(out, definition)
	return err
}

func writeFunctionSDL(out io.Writer, _ string, function *storepb.FunctionMetadata) error {
	// The function definition should already include the complete CREATE FUNCTION statement
	definition := strings.TrimSpace(function.Definition)
	// Remove trailing semicolon if present
	definition = strings.TrimSuffix(definition, ";")

	_, err := io.WriteString(out, definition)
	return err
}

func writeSequenceSDL(out io.Writer, schemaName string, sequence *storepb.SequenceMetadata) error {
	// Write CREATE SEQUENCE statement with schema and name
	if _, err := fmt.Fprintf(out, "CREATE SEQUENCE \"%s\".\"%s\"", schemaName, sequence.Name); err != nil {
		return err
	}

	// Add data type (always output for consistency with non-SDL format)
	if sequence.DataType != "" {
		if _, err := fmt.Fprintf(out, " AS %s", sequence.DataType); err != nil {
			return err
		}
	}

	// Add START WITH (always output for consistency with non-SDL format)
	if sequence.Start != "" {
		if _, err := fmt.Fprintf(out, " START WITH %s", sequence.Start); err != nil {
			return err
		}
	}

	// Add INCREMENT BY (always output for consistency with non-SDL format)
	if sequence.Increment != "" {
		if _, err := fmt.Fprintf(out, " INCREMENT BY %s", sequence.Increment); err != nil {
			return err
		}
	}

	// Add MINVALUE (always output for consistency with non-SDL format)
	if sequence.MinValue != "" {
		if _, err := fmt.Fprintf(out, " MINVALUE %s", sequence.MinValue); err != nil {
			return err
		}
	}

	// Add MAXVALUE (always output for consistency with non-SDL format)
	if sequence.MaxValue != "" {
		if _, err := fmt.Fprintf(out, " MAXVALUE %s", sequence.MaxValue); err != nil {
			return err
		}
	}

	// Add CYCLE/NO CYCLE (always output for consistency with non-SDL format)
	if sequence.Cycle {
		if _, err := io.WriteString(out, " CYCLE"); err != nil {
			return err
		}
	} else {
		if _, err := io.WriteString(out, " NO CYCLE"); err != nil {
			return err
		}
	}

	// Add CACHE (new field to match non-SDL format)
	if sequence.CacheSize != "" {
		if _, err := fmt.Fprintf(out, " CACHE %s", sequence.CacheSize); err != nil {
			return err
		}
	}

	return nil
}

func writeSchema(out io.Writer, schema *storepb.SchemaMetadata) error {
	if schema.Name == "public" {
		return nil
	}

	if _, err := io.WriteString(out, `CREATE SCHEMA IF NOT EXISTS "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `";`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

// writeCheckConstraintSDL writes a single check constraint SDL
func writeCheckConstraintSDL(out io.Writer, check *storepb.CheckConstraintMetadata) error {
	if _, err := io.WriteString(out, `CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, check.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" CHECK `); err != nil {
		return err
	}
	if _, err := io.WriteString(out, check.Expression); err != nil {
		return err
	}
	return nil
}

func writeExcludeConstraintSDL(out io.Writer, exclude *storepb.ExcludeConstraintMetadata) error {
	if _, err := io.WriteString(out, `CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, exclude.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" `); err != nil {
		return err
	}
	// Expression already includes "EXCLUDE USING ..." prefix
	if _, err := io.WriteString(out, exclude.Expression); err != nil {
		return err
	}
	return nil
}

// writeForeignKeyConstraintSDL writes a single foreign key constraint SDL
func writeForeignKeyConstraintSDL(out io.Writer, fk *storepb.ForeignKeyMetadata) error {
	if _, err := io.WriteString(out, `CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, fk.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" FOREIGN KEY (`); err != nil {
		return err
	}
	for i, column := range fk.Columns {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, column); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, ") REFERENCES "); err != nil {
		return err
	}

	// Always add schema qualifier according to project principle
	if fk.ReferencedSchema != "" {
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, fk.ReferencedSchema); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `".`); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(out, `"`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, fk.ReferencedTable); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" ("); err != nil {
		return err
	}
	for i, column := range fk.ReferencedColumns {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
		if _, err := io.WriteString(out, column); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, ")"); err != nil {
		return err
	}

	// Add ON DELETE clause if specified
	if fk.OnDelete != "" && fk.OnDelete != "NO ACTION" {
		if _, err := io.WriteString(out, " ON DELETE "); err != nil {
			return err
		}
		if _, err := io.WriteString(out, fk.OnDelete); err != nil {
			return err
		}
	}

	// Add ON UPDATE clause if specified
	if fk.OnUpdate != "" && fk.OnUpdate != "NO ACTION" {
		if _, err := io.WriteString(out, " ON UPDATE "); err != nil {
			return err
		}
		if _, err := io.WriteString(out, fk.OnUpdate); err != nil {
			return err
		}
	}
	return nil
}

// GetMultiFileDatabaseDefinition generates multi-file SDL schema for PostgreSQL.
func GetMultiFileDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *storepb.DatabaseSchemaMetadata) (*schema.MultiFileSchemaResult, error) {
	metadata = filterBackupSchemaIfNecessary(ctx, metadata)

	if len(metadata.Schemas) == 0 {
		return &schema.MultiFileSchemaResult{Files: []schema.File{}}, nil
	}

	var files []schema.File

	// Build skip sequences map (for serial and identity columns)
	skipSequences := buildSkipSequencesMap(metadata)

	// Build table sequences map for each table
	tableSequencesMap := buildTableSequencesMap(metadata)

	// Generate files for each schema
	for _, schemaMetadata := range metadata.Schemas {
		if schemaMetadata.SkipDump {
			continue
		}

		schemaName := schemaMetadata.Name
		if schemaName == "" {
			schemaName = "public"
		}

		// Collect independent sequences (no owner) for this schema
		var independentSequences []*storepb.SequenceMetadata
		for _, sequence := range schemaMetadata.Sequences {
			if sequence.SkipDump {
				continue
			}
			// Skip sequences that belong to serial or identity columns
			sequenceKey := schemaName + "." + sequence.Name
			if skipSequences[sequenceKey] {
				continue
			}

			// Collect independent sequences (no owner)
			if sequence.OwnerTable == "" || sequence.OwnerColumn == "" {
				independentSequences = append(independentSequences, sequence)
			}
		}

		// Generate table files
		for _, table := range schemaMetadata.Tables {
			if table.SkipDump {
				continue
			}

			// Get sequences for this table
			tableKey := schemaName + "." + table.Name
			tableSequences := tableSequencesMap[tableKey]

			var buf strings.Builder
			// Write CREATE TABLE statement in SDL format
			if err := writeCreateTableSDL(&buf, schemaName, table, tableSequences); err != nil {
				return nil, errors.Wrapf(err, "failed to generate table SDL for %s.%s", schemaName, table.Name)
			}
			buf.WriteString(";\n\n")

			// Write table comment if present
			if len(table.Comment) > 0 {
				if err := writeTableCommentSDL(&buf, schemaName, table); err != nil {
					return nil, errors.Wrapf(err, "failed to generate table comment for %s.%s", schemaName, table.Name)
				}
			}

			// Write column comments if present
			for _, column := range table.Columns {
				if len(column.Comment) > 0 {
					if err := writeColumnCommentSDL(&buf, schemaName, table.Name, column); err != nil {
						return nil, errors.Wrapf(err, "failed to generate column comment for %s.%s.%s", schemaName, table.Name, column.Name)
					}
				}
			}

			// Write CREATE INDEX statements for non-constraint indexes
			if err := writeIndexesSDL(&buf, schemaName, table); err != nil {
				return nil, errors.Wrapf(err, "failed to generate indexes SDL for %s.%s", schemaName, table.Name)
			}

			// Write index comments if present
			for _, index := range table.Indexes {
				// Only write comment for standalone indexes (not primary key or unique constraint indexes)
				if !index.Primary && !index.Unique {
					if len(index.Comment) > 0 {
						if err := writeIndexCommentSDL(&buf, schemaName, index); err != nil {
							return nil, errors.Wrapf(err, "failed to generate index comment for %s.%s", schemaName, index.Name)
						}
					}
				}
			}

			// Write CREATE TRIGGER statements
			if err := writeTriggersSDL(&buf, schemaName, table); err != nil {
				return nil, errors.Wrapf(err, "failed to generate triggers SDL for %s.%s", schemaName, table.Name)
			}

			// Write trigger comments if present
			for _, trigger := range table.Triggers {
				if len(trigger.Comment) > 0 {
					if err := writeTriggerCommentSDL(&buf, schemaName, table.Name, trigger); err != nil {
						return nil, errors.Wrapf(err, "failed to generate trigger comment for %s.%s.%s", schemaName, table.Name, trigger.Name)
					}
				}
			}

			// Write owned sequences (non-serial/non-identity) for this table
			for _, sequence := range tableSequences {
				// Skip sequences that belong to serial or identity columns
				sequenceKey := schemaName + "." + sequence.Name
				if skipSequences[sequenceKey] {
					continue
				}

				// Only include sequences with owners (non-independent sequences)
				if sequence.OwnerTable != "" && sequence.OwnerColumn != "" {
					buf.WriteString("\n")
					if err := writeSequenceSDL(&buf, schemaName, sequence); err != nil {
						return nil, errors.Wrapf(err, "failed to generate sequence SDL for %s.%s", schemaName, sequence.Name)
					}
					buf.WriteString(";\n")

					// Add sequence comment if present
					if len(sequence.Comment) > 0 {
						buf.WriteString("\n")
						if err := writeSequenceCommentSDL(&buf, schemaName, sequence); err != nil {
							return nil, errors.Wrapf(err, "failed to generate sequence comment for %s.%s", schemaName, sequence.Name)
						}
					}

					// Add ALTER SEQUENCE OWNED BY
					buf.WriteString("\n")
					if err := writeAlterSequenceOwnedBy(&buf, schemaName, sequence); err != nil {
						return nil, errors.Wrapf(err, "failed to generate ALTER SEQUENCE OWNED BY for %s.%s", schemaName, sequence.Name)
					}
				}
			}

			files = append(files, schema.File{
				Name:    fmt.Sprintf("schemas/%s/tables/%s.sql", schemaName, table.Name),
				Content: buf.String(),
			})
		}

		// Generate view files
		for _, view := range schemaMetadata.Views {
			if view.SkipDump {
				continue
			}

			var buf strings.Builder
			if err := writeViewSDL(&buf, schemaName, view); err != nil {
				return nil, errors.Wrapf(err, "failed to generate view SDL for %s.%s", schemaName, view.Name)
			}
			buf.WriteString(";\n")

			// Write view comment if present
			if len(view.Comment) > 0 {
				buf.WriteString("\n")
				if err := writeViewCommentSDL(&buf, schemaName, view); err != nil {
					return nil, errors.Wrapf(err, "failed to generate view comment for %s.%s", schemaName, view.Name)
				}
			}

			files = append(files, schema.File{
				Name:    fmt.Sprintf("schemas/%s/views/%s.sql", schemaName, view.Name),
				Content: buf.String(),
			})
		}

		// Generate materialized view files
		for _, materializedView := range schemaMetadata.MaterializedViews {
			if materializedView.SkipDump {
				continue
			}

			var buf strings.Builder
			if err := writeMaterializedViewSDL(&buf, schemaName, materializedView); err != nil {
				return nil, errors.Wrapf(err, "failed to generate materialized view SDL for %s.%s", schemaName, materializedView.Name)
			}
			buf.WriteString(";\n")

			// Write materialized view comment if present
			if len(materializedView.Comment) > 0 {
				buf.WriteString("\n")
				if err := writeMaterializedViewCommentSDL(&buf, schemaName, materializedView); err != nil {
					return nil, errors.Wrapf(err, "failed to generate materialized view comment for %s.%s", schemaName, materializedView.Name)
				}
			}

			// Write indexes on materialized view
			for _, index := range materializedView.Indexes {
				// Skip constraint-based indexes
				if index.Primary || index.IsConstraint {
					continue
				}

				buf.WriteString("\n")
				if err := writeIndexSDL(&buf, schemaName, materializedView.Name, index); err != nil {
					return nil, errors.Wrapf(err, "failed to generate index SDL for %s.%s", schemaName, index.Name)
				}
				buf.WriteString(";\n")

				// Write index comment if present
				if len(index.Comment) > 0 {
					buf.WriteString("\n")
					if err := writeIndexCommentSDL(&buf, schemaName, index); err != nil {
						return nil, errors.Wrapf(err, "failed to generate index comment for %s.%s", schemaName, index.Name)
					}
				}
			}

			files = append(files, schema.File{
				Name:    fmt.Sprintf("schemas/%s/materialized_views/%s.sql", schemaName, materializedView.Name),
				Content: buf.String(),
			})
		}

		// Generate function files
		for _, function := range schemaMetadata.Functions {
			if function.SkipDump {
				continue
			}

			var buf strings.Builder
			if err := writeFunctionSDL(&buf, schemaName, function); err != nil {
				return nil, errors.Wrapf(err, "failed to generate function SDL for %s.%s", schemaName, function.Name)
			}
			buf.WriteString(";\n")

			// Write function comment if present
			if len(function.Comment) > 0 {
				buf.WriteString("\n")
				if err := writeFunctionCommentSDL(&buf, schemaName, function); err != nil {
					return nil, errors.Wrapf(err, "failed to generate function comment for %s.%s", schemaName, function.Name)
				}
			}

			// Determine if this is a PROCEDURE or FUNCTION to use different folder
			folderName := "functions"
			if isDefinitionProcedure(function.Definition) {
				folderName = "procedures"
			}

			files = append(files, schema.File{
				Name:    fmt.Sprintf("schemas/%s/%s/%s.sql", schemaName, folderName, function.Name),
				Content: buf.String(),
			})
		}

		// Generate a single file for all enum types in this schema
		if len(schemaMetadata.EnumTypes) > 0 {
			var buf strings.Builder
			hasEnumTypes := false
			for i, enumType := range schemaMetadata.EnumTypes {
				if enumType.SkipDump {
					continue
				}

				if hasEnumTypes {
					buf.WriteString("\n")
				}
				hasEnumTypes = true

				if i > 0 {
					buf.WriteString("\n")
				}

				if err := writeEnum(&buf, schemaName, enumType); err != nil {
					return nil, errors.Wrapf(err, "failed to generate enum type SDL for %s.%s", schemaName, enumType.Name)
				}
				buf.WriteString(";\n")

				// Add enum type comment if present
				if len(enumType.Comment) > 0 {
					buf.WriteString("\n")
					if err := writeEnumComment(&buf, schemaName, enumType); err != nil {
						return nil, errors.Wrapf(err, "failed to generate enum type comment for %s.%s", schemaName, enumType.Name)
					}
				}
			}

			if hasEnumTypes {
				files = append(files, schema.File{
					Name:    fmt.Sprintf("schemas/%s/types.sql", schemaName),
					Content: buf.String(),
				})
			}
		}

		// Generate a single file for all independent sequences (no owner) in this schema
		if len(independentSequences) > 0 {
			var buf strings.Builder
			for i, sequence := range independentSequences {
				if i > 0 {
					buf.WriteString("\n")
				}

				if err := writeSequenceSDL(&buf, schemaName, sequence); err != nil {
					return nil, errors.Wrapf(err, "failed to generate sequence SDL for %s.%s", schemaName, sequence.Name)
				}
				buf.WriteString(";\n")

				// Add sequence comment if present
				if len(sequence.Comment) > 0 {
					buf.WriteString("\n")
					if err := writeSequenceCommentSDL(&buf, schemaName, sequence); err != nil {
						return nil, errors.Wrapf(err, "failed to generate sequence comment for %s.%s", schemaName, sequence.Name)
					}
				}
			}

			files = append(files, schema.File{
				Name:    fmt.Sprintf("schemas/%s/sequences.sql", schemaName),
				Content: buf.String(),
			})
		}
	}

	// Generate extensions.sql file if there are any extensions
	if len(metadata.Extensions) > 0 {
		var buf strings.Builder
		for i, extension := range metadata.Extensions {
			if i > 0 {
				buf.WriteString("\n")
			}

			if err := writeExtension(&buf, extension); err != nil {
				return nil, errors.Wrapf(err, "failed to generate extension SDL for %s", extension.Name)
			}
		}

		files = append(files, schema.File{
			Name:    "extensions.sql",
			Content: buf.String(),
		})
	}

	// Generate event_triggers.sql file if there are any event triggers
	if len(metadata.EventTriggers) > 0 {
		var buf strings.Builder
		for i, eventTrigger := range metadata.EventTriggers {
			if eventTrigger.SkipDump {
				continue
			}
			if i > 0 {
				buf.WriteString("\n")
			}

			if err := writeEventTrigger(&buf, eventTrigger); err != nil {
				return nil, errors.Wrapf(err, "failed to generate event trigger SDL for %s", eventTrigger.Name)
			}
		}

		if buf.Len() > 0 {
			files = append(files, schema.File{
				Name:    "event_triggers.sql",
				Content: buf.String(),
			})
		}
	}

	return &schema.MultiFileSchemaResult{Files: files}, nil
}

// buildSkipSequencesMap builds a map of sequences that should be skipped (serial and identity sequences).
func buildSkipSequencesMap(metadata *storepb.DatabaseSchemaMetadata) map[string]bool {
	skipSequences := make(map[string]bool)
	for _, schema := range metadata.Schemas {
		if schema.SkipDump {
			continue
		}
		for _, table := range schema.Tables {
			if table.SkipDump {
				continue
			}
			for _, column := range table.Columns {
				// Check for serial columns
				isSerial, _ := isSerialColumn(column, table.Name, schema.Sequences)
				if isSerial {
					// Extract the sequence name from the DEFAULT clause to match the exact sequence
					// This ensures we skip the correct sequence, especially when multiple sequences
					// claim ownership of the same column.
					sequenceName := extractSequenceNameFromNextval(column.Default)

					for _, sequence := range schema.Sequences {
						// Match by sequence name AND ownership to ensure we skip the exact sequence
						// referenced in the DEFAULT clause
						if sequence.Name == sequenceName && sequence.OwnerTable == table.Name && sequence.OwnerColumn == column.Name {
							sequenceKey := schema.Name + "." + sequence.Name
							skipSequences[sequenceKey] = true
							break
						}
					}
				}
				// Check for identity columns
				if isIdentityColumn(column) {
					for _, sequence := range schema.Sequences {
						if sequence.OwnerTable == table.Name && sequence.OwnerColumn == column.Name {
							sequenceKey := schema.Name + "." + sequence.Name
							skipSequences[sequenceKey] = true
							break
						}
					}
				}
			}
		}
	}
	return skipSequences
}

// buildTableSequencesMap builds a map of sequences by table for easy lookup during table creation.
func buildTableSequencesMap(metadata *storepb.DatabaseSchemaMetadata) map[string][]*storepb.SequenceMetadata {
	tableSequencesMap := make(map[string][]*storepb.SequenceMetadata)
	for _, schema := range metadata.Schemas {
		if schema.SkipDump {
			continue
		}
		for _, sequence := range schema.Sequences {
			if sequence.SkipDump {
				continue
			}
			if sequence.OwnerTable != "" {
				tableKey := schema.Name + "." + sequence.OwnerTable
				tableSequencesMap[tableKey] = append(tableSequencesMap[tableKey], sequence)
			}
		}
	}
	return tableSequencesMap
}

func writeMaterializedViewCommentSDL(out io.Writer, schemaName string, view *storepb.MaterializedViewMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON MATERIALIZED VIEW "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schemaName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, escapeSingleQuote(view.Comment)); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\n\n")
	return err
}

// SDL Comment Functions

func writeSchemaCommentSDL(out io.Writer, schema *storepb.SchemaMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON SCHEMA "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, escapeSingleQuote(schema.Comment)); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeTableCommentSDL(out io.Writer, schemaName string, table *storepb.TableMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON TABLE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schemaName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, escapeSingleQuote(table.Comment)); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeColumnCommentSDL(out io.Writer, schemaName, tableName string, column *storepb.ColumnMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON COLUMN "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schemaName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, tableName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, column.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, escapeSingleQuote(column.Comment)); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeViewCommentSDL(out io.Writer, schemaName string, view *storepb.ViewMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON VIEW "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schemaName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, escapeSingleQuote(view.Comment)); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeFunctionCommentSDL(out io.Writer, schemaName string, function *storepb.FunctionMetadata) error {
	// Determine if this is a PROCEDURE or FUNCTION by checking the definition
	objectType := "FUNCTION"
	if isDefinitionProcedure(function.Definition) {
		objectType = "PROCEDURE"
	}

	if _, err := io.WriteString(out, "COMMENT ON "+objectType+" \""); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schemaName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `".`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, function.Signature); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ` IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, escapeSingleQuote(function.Comment)); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeSequenceCommentSDL(out io.Writer, schemaName string, sequence *storepb.SequenceMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON SEQUENCE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schemaName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, escapeSingleQuote(sequence.Comment)); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeIndexCommentSDL(out io.Writer, schemaName string, index *storepb.IndexMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON INDEX "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schemaName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, escapeSingleQuote(index.Comment)); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeTriggersSDL(out io.Writer, schemaName string, table *storepb.TableMetadata) error {
	for _, trigger := range table.Triggers {
		if trigger.SkipDump {
			continue
		}

		if err := writeTriggerSDL(out, schemaName, table.Name, trigger); err != nil {
			return err
		}

		if _, err := io.WriteString(out, ";\n\n"); err != nil {
			return err
		}
	}
	return nil
}

func writeTriggerSDL(out io.Writer, _ /* schemaName */, _ /* tableName */ string, trigger *storepb.TriggerMetadata) error {
	// For PostgreSQL, trigger.Body contains the complete CREATE TRIGGER statement
	// built by buildTriggerDefinition in get_database_metadata.go
	_, err := io.WriteString(out, trigger.Body)
	return err
}

func writeTriggerCommentSDL(out io.Writer, schemaName, tableName string, trigger *storepb.TriggerMetadata) error {
	if len(trigger.Comment) == 0 {
		return nil
	}

	if _, err := io.WriteString(out, `COMMENT ON TRIGGER "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, trigger.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" ON "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schemaName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, tableName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, escapeSingleQuote(trigger.Comment)); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `';`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}
