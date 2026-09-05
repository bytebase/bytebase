package redshift

import (
	"fmt"
	"io"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func init() {
	schema.RegisterGetDatabaseDefinition(storepb.Engine_REDSHIFT, GetDatabaseDefinition)
	schema.RegisterGetTableDefinition(storepb.Engine_REDSHIFT, GetTableDefinition)
	schema.RegisterGetViewDefinition(storepb.Engine_REDSHIFT, GetViewDefinition)
}

// GetTableDefinition renders one table exactly as the whole-database output
// renders it, minus the section banner.
func GetTableDefinition(schemaName string, table *storepb.TableMetadata, _ []*storepb.SequenceMetadata) (string, error) {
	var sb strings.Builder
	if err := writeTable(&sb, convertToTableState(0, schemaName, table)); err != nil {
		return "", err
	}
	return sb.String(), nil
}

func GetViewDefinition(schemaName string, view *storepb.ViewMetadata) (string, error) {
	var sb strings.Builder
	if err := writeView(&sb, convertToViewState(0, schemaName, view)); err != nil {
		return "", err
	}
	return sb.String(), nil
}

func GetDatabaseDefinition(_ schema.GetDefinitionContext, to *storepb.DatabaseSchemaMetadata) (string, error) {
	toState := convertToDatabaseState(to)

	var sb strings.Builder

	if err := writeTables(&sb, to, toState); err != nil {
		return "", err
	}
	if err := writeViews(&sb, to, toState); err != nil {
		return "", err
	}

	s := sb.String()
	// Make goyamlv3 happy.
	s = strings.TrimLeft(s, "\n")
	return s, nil
}

func writeTables(w io.StringWriter, to *storepb.DatabaseSchemaMetadata, state *databaseState) error {
	// Follow the order of the input schemas.
	for _, schema := range to.Schemas {
		schemaState, ok := state.schemas[schema.Name]
		if !ok {
			continue
		}
		// Follow the order of the input tables.
		for _, table := range schema.Tables {
			table, ok := schemaState.tables[table.Name]
			if !ok {
				continue
			}
			if _, err := w.WriteString(getTableAnnouncement(table.name)); err != nil {
				return err
			}

			if err := writeTable(w, table); err != nil {
				return err
			}
			delete(schemaState.tables, table.name)
		}
	}
	return nil
}

func writeViews(w io.StringWriter, to *storepb.DatabaseSchemaMetadata, state *databaseState) error {
	// Follow the order of the input schemas.
	for _, schema := range to.Schemas {
		schemaState, ok := state.schemas[schema.Name]
		if !ok {
			continue
		}
		// Follow the order of the input views.
		for _, view := range schema.Views {
			view, ok := schemaState.views[view.Name]
			if !ok {
				continue
			}
			if _, err := w.WriteString(getViewAnnouncement(view.name)); err != nil {
				return err
			}

			if err := writeView(w, view); err != nil {
				return err
			}
			delete(schemaState.views, view.name)
		}
	}
	return nil
}

// writeTable writes one table and the comments that belong to it. Column
// comments follow the column order the CREATE TABLE body uses; ranging over the
// map directly, as this did before, ordered them differently run to run.
func writeTable(w io.StringWriter, table *tableState) error {
	buf := &strings.Builder{}
	if err := table.toString(buf); err != nil {
		return err
	}
	if table.comment != "" {
		if _, err := fmt.Fprintf(buf, "COMMENT ON TABLE %s IS '%s';\n", table.qualifiedName(), escapeSingleQuote(table.comment)); err != nil {
			return err
		}
	}
	for _, column := range sortedColumns(table.columns) {
		if column.comment == "" {
			continue
		}
		if _, err := fmt.Fprintf(buf, "COMMENT ON COLUMN %s.%s IS '%s';\n", table.qualifiedName(), column.name, escapeSingleQuote(column.comment)); err != nil {
			return err
		}
	}
	_, err := w.WriteString(buf.String())
	return err
}

// writeView writes one view and its comment.
func writeView(w io.StringWriter, view *viewState) error {
	buf := &strings.Builder{}
	if err := view.toString(buf); err != nil {
		return err
	}
	if view.comment != "" {
		if _, err := fmt.Fprintf(buf, "COMMENT ON VIEW %s IS '%s';\n", view.qualifiedName(), escapeSingleQuote(view.comment)); err != nil {
			return err
		}
	}
	_, err := w.WriteString(buf.String())
	return err
}

func getTableAnnouncement(name string) string {
	return fmt.Sprintf("\n--\n-- Table structure for `%s`\n--\n", name)
}

func getViewAnnouncement(name string) string {
	return fmt.Sprintf("\n--\n-- View structure for `%s`\n--\n", name)
}
