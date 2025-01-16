package mysql

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterGetDesignSchema(storepb.Engine_MYSQL, GetDesignSchema)
	schema.RegisterGetDesignSchema(storepb.Engine_OCEANBASE, GetDesignSchema)
}

func GetDesignSchema(to *storepb.DatabaseSchemaMetadata) (string, error) {
	toState := convertToDatabaseState(to)

	var sb strings.Builder

	if err := writeTables(&sb, to, toState); err != nil {
		return "", err
	}

	if err := writeViews(&sb, to, toState); err != nil {
		return "", err
	}

	if err := writeFunctions(&sb, to, toState); err != nil {
		return "", err
	}

	if err := writeProcedures(&sb, to, toState); err != nil {
		return "", err
	}

	s := sb.String()
	// Make goyamlv3 happy.
	s = strings.TrimLeft(s, "\n")
	result, err := mysqlparser.RestoreDelimiter(s)
	if err != nil {
		slog.Warn("Failed to restore delimiter", slog.String("result", s), slog.String("error", err.Error()))
		return s, nil
	}
	return result, nil
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

			// Avoid new line.
			buf := &strings.Builder{}
			if err := table.toString(buf); err != nil {
				return err
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
			_, handled := schemaState.handledViews[view.Name]
			if handled {
				continue
			}
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
			if _, err := w.WriteString(buf.String()); err != nil {
				return err
			}
			delete(schemaState.views, view.name)
		}
	}
	return nil
}

func writeProcedures(w io.StringWriter, to *storepb.DatabaseSchemaMetadata, state *databaseState) error {
	// Follow the order of the input schemas.
	for _, schema := range to.Schemas {
		schemaState, ok := state.schemas[schema.Name]
		if !ok {
			continue
		}
		// Follow the order of the input procedures.
		for _, procedure := range schema.Procedures {
			procedure, ok := schemaState.procedures[procedure.Name]
			if !ok {
				continue
			}
			if _, err := w.WriteString(getProcedureAnnouncement(procedure.name)); err != nil {
				return err
			}

			buf := &strings.Builder{}
			if err := procedure.toString(buf); err != nil {
				return err
			}
			if _, err := w.WriteString(buf.String()); err != nil {
				return err
			}
			delete(schemaState.procedures, procedure.name)
		}
	}
	return nil
}

func writeFunctions(w io.StringWriter, to *storepb.DatabaseSchemaMetadata, state *databaseState) error {
	// Follow the order of the input schemas.
	for _, schema := range to.Schemas {
		schemaState, ok := state.schemas[schema.Name]
		if !ok {
			continue
		}
		// Follow the order of the input functions.
		for _, function := range schema.Functions {
			function, ok := schemaState.functions[function.Name]
			if !ok {
				continue
			}
			if _, err := w.WriteString(getFunctionAnnouncement(function.name)); err != nil {
				return err
			}

			buf := &strings.Builder{}
			if err := function.toString(buf); err != nil {
				return err
			}
			if _, err := w.WriteString(buf.String()); err != nil {
				return err
			}
			delete(schemaState.functions, function.name)
		}
	}
	return nil
}

func getViewAnnouncement(name string) string {
	return fmt.Sprintf("\n--\n-- View structure for `%s`\n--\n", name)
}

func getTableAnnouncement(name string) string {
	return fmt.Sprintf("\n--\n-- Table structure for `%s`\n--\n", name)
}

func getFunctionAnnouncement(name string) string {
	return fmt.Sprintf("\n--\n-- Function structure for `%s`\n--\n", name)
}

func getProcedureAnnouncement(name string) string {
	return fmt.Sprintf("\n--\n-- Procedure structure for `%s`\n--\n", name)
}
