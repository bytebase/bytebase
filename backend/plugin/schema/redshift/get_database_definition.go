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

			buf := &strings.Builder{}
			if err := table.toString(buf); err != nil {
				return err
			}
			// Generate comment for table and columns.
			if table.comment != "" {
				if _, err := fmt.Fprintf(buf, "COMMENT ON TABLE %s IS '%s';\n", table.name, table.comment); err != nil {
					return err
				}
			}
			for _, column := range table.columns {
				if column.comment != "" {
					if _, err := fmt.Fprintf(buf, "COMMENT ON COLUMN %s.%s IS '%s';\n", table.name, column.name, column.comment); err != nil {
						return err
					}
				}
			}
			if _, err := w.WriteString(buf.String()); err != nil {
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

			buf := &strings.Builder{}
			if err := view.toString(buf); err != nil {
				return err
			}
			// Generate comment for view.
			if view.comment != "" {
				if _, err := fmt.Fprintf(buf, "COMMENT ON VIEW %s IS '%s';\n", view.name, view.comment); err != nil {
					return err
				}
			}
			if _, err := w.WriteString(buf.String()); err != nil {
				return err
			}
			delete(schemaState.views, view.name)
		}
	}
	return nil
}

func getTableAnnouncement(name string) string {
	return fmt.Sprintf("\n--\n-- Table structure for `%s`\n--\n", name)
}

func getViewAnnouncement(name string) string {
	return fmt.Sprintf("\n--\n-- View structure for `%s`\n--\n", name)
}
