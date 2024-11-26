package mysql

import (
	"context"
	"io"

	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump and restore.
const (
	emptyCommentLine      = "--\n"
	tempViewHeader        = "-- Temporary view structure for "
	tableHeader           = "-- Table structure for "
	viewHeader            = "-- View structure for "
	functionHeader        = "-- Function structure for "
	procedureHeader       = "-- Procedure structure for "
	triggerHeader         = "-- Trigger structure for "
	eventHeader           = "-- Event structure for "
	setCharacterSetClient = "SET character_set_client = "
	setCharacterSetResult = "SET character_set_results = "
	setCollation          = "SET collation_connection = "
	setSQLMode            = "SET sql_mode = "
	setTimezone           = "SET time_zone = "
	delimiterDoubleSemi   = "DELIMITER ;;\n"
	delimiterSemi         = "DELIMITER ;\n"

	disableUniqueAndForeignKeyCheckStmt = "SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;\nSET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;\n"
	restoreUniqueAndForeignKeyCheckStmt = "SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;\nSET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;\n"
)

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, out io.Writer, dbSchema *storepb.DatabaseSchemaMetadata) error {
	if len(dbSchema.Schemas) == 0 {
		return nil
	}

	// Disable foreign key check.
	// mysqldump uses the same mechanism. When there is any schema or data dependency, we have to disable
	// the unique and foreign key check so that the restoring will not fail.
	if _, err := io.WriteString(out, disableUniqueAndForeignKeyCheckStmt); err != nil {
		return err
	}

	schema := dbSchema.Schemas[0]

	// Construct temporal views.
	// Create a temporary view with the same name as the view and with columns of
	// the same name in order to satisfy views that depend on this view.
	// This temporary view will be removed when the actual view is created.
	// The properties of each column, are not preserved in this temporary
	// view. They are not necessary because other views only need to reference
	// the column name, thus we generate SELECT 1 AS colName1, 1 AS colName2.
	// This will not be necessary once we can determine dependencies
	// between views and can simply dump them in the appropriate order.
	// https://sourcegraph.com/github.com/mysql/mysql-server/-/blob/client/mysqldump.cc?L2781
	for _, view := range schema.Views {
		if len(view.Columns) == 0 {
			if err := writeInvalidTemporaryView(out, view); err != nil {
				return err
			}
			continue
		}
		if err := writeTemporaryView(out, view); err != nil {
			return err
		}
	}

	// Construct tables.
	for _, table := range schema.Tables {
		if err := writeTable(out, table); err != nil {
			return err
		}
	}

	// Construct views.
	for _, view := range schema.Views {
		if err := writeView(out, view); err != nil {
			return err
		}
	}

	// Construct functions.
	for _, function := range schema.Functions {
		if err := writeFunction(out, function); err != nil {
			return err
		}
	}

	// Construct procedures.
	for _, procedure := range schema.Procedures {
		if err := writeProcedure(out, procedure); err != nil {
			return err
		}
	}

	// Construct events.
	for _, event := range schema.Events {
		if err := writeEvent(out, event); err != nil {
			return err
		}
	}

	// Construct triggers.
	for _, trigger := range schema.Triggers {
		if err := writeTrigger(out, trigger); err != nil {
			return err
		}
	}

	// Restore foreign key check.
	if _, err := io.WriteString(out, restoreUniqueAndForeignKeyCheckStmt); err != nil {
		return err
	}

	return nil
}

func writeEvent(out io.Writer, event *storepb.EventMetadata) error {
	// Header.
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}
	if _, err := io.WriteString(out, eventHeader); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, event.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}

	// Set charset, collation, sql mode and timezone.
	if _, err := io.WriteString(out, setCharacterSetClient); err != nil {
		return err
	}
	if _, err := io.WriteString(out, event.CharacterSetClient); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, setCharacterSetResult); err != nil {
		return err
	}
	if _, err := io.WriteString(out, event.CharacterSetClient); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, setCollation); err != nil {
		return err
	}
	if _, err := io.WriteString(out, event.CollationConnection); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, setSQLMode); err != nil {
		return err
	}
	if _, err := io.WriteString(out, event.SqlMode); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, setTimezone); err != nil {
		return err
	}
	if _, err := io.WriteString(out, event.TimeZone); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, delimiterDoubleSemi); err != nil {
		return err
	}

	// Definition.
	if _, err := io.WriteString(out, event.Definition); err != nil {
		return err
	}

	if _, err := io.WriteString(out, ";;\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, delimiterSemi); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n")
	return err
}

func writeTrigger(out io.Writer, trigger *storepb.TriggerMetadata) error {
	// Header.
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}
	if _, err := io.WriteString(out, triggerHeader); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, trigger.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}

	// Set charset, collation, and sql mode.
	if _, err := io.WriteString(out, setCharacterSetClient); err != nil {
		return err
	}
	if _, err := io.WriteString(out, trigger.CharacterSetClient); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, setCharacterSetResult); err != nil {
		return err
	}
	if _, err := io.WriteString(out, trigger.CharacterSetClient); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, setCollation); err != nil {
		return err
	}
	if _, err := io.WriteString(out, trigger.CollationConnection); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, setSQLMode); err != nil {
		return err
	}
	if _, err := io.WriteString(out, trigger.SqlMode); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, delimiterDoubleSemi); err != nil {
		return err
	}

	// Definition.
	if _, err := io.WriteString(out, "CREATE TRIGGER `"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, trigger.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, trigger.Timing); err != nil {
		return err
	}
	if _, err := io.WriteString(out, " "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, trigger.Event); err != nil {
		return err
	}
	if _, err := io.WriteString(out, " ON `"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, trigger.TableName); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "` FOR EACH ROW\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, trigger.Body); err != nil {
		return err
	}

	if _, err := io.WriteString(out, ";;\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, delimiterSemi); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n")
	return err
}

func writeProcedure(out io.Writer, procedure *storepb.ProcedureMetadata) error {
	// Header.
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}
	if _, err := io.WriteString(out, procedureHeader); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, procedure.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}

	// Set charset, collation, and sql mode.
	if _, err := io.WriteString(out, setCharacterSetClient); err != nil {
		return err
	}
	if _, err := io.WriteString(out, procedure.CharacterSetClient); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, setCharacterSetResult); err != nil {
		return err
	}
	if _, err := io.WriteString(out, procedure.CharacterSetClient); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, setCollation); err != nil {
		return err
	}
	if _, err := io.WriteString(out, procedure.CollationConnection); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, setSQLMode); err != nil {
		return err
	}
	if _, err := io.WriteString(out, procedure.SqlMode); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, delimiterDoubleSemi); err != nil {
		return err
	}

	// Definition.
	if _, err := io.WriteString(out, procedure.Definition); err != nil {
		return err
	}

	if _, err := io.WriteString(out, ";;\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, delimiterSemi); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n")
	return err
}

func writeFunction(out io.Writer, function *storepb.FunctionMetadata) error {
	// Header.
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}
	if _, err := io.WriteString(out, functionHeader); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, function.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}

	// Set charset, collation, and sql mode.
	if _, err := io.WriteString(out, setCharacterSetClient); err != nil {
		return err
	}
	if _, err := io.WriteString(out, function.CharacterSetClient); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, setCharacterSetResult); err != nil {
		return err
	}
	if _, err := io.WriteString(out, function.CharacterSetClient); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, setCollation); err != nil {
		return err
	}
	if _, err := io.WriteString(out, function.CollationConnection); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, setSQLMode); err != nil {
		return err
	}
	if _, err := io.WriteString(out, function.SqlMode); err != nil {
		return err
	}
	if _, err := io.WriteString(out, ";\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, delimiterDoubleSemi); err != nil {
		return err
	}

	// Definition.
	if _, err := io.WriteString(out, function.Definition); err != nil {
		return err
	}

	if _, err := io.WriteString(out, ";;\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, delimiterSemi); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n")
	return err
}

func writeView(out io.Writer, view *storepb.ViewMetadata) error {
	// Header.
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}
	if _, err := io.WriteString(out, viewHeader); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}

	// Drop temporary view.
	if _, err := io.WriteString(out, "DROP VIEW IF EXISTS `"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`;\n"); err != nil {
		return err
	}

	// Definition.
	if _, err := io.WriteString(out, "CREATE VIEW `"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "` AS "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Definition); err != nil {
		return err
	}
	_, err := io.WriteString(out, ";\n\n")
	return err
}

func writeTable(out io.Writer, table *storepb.TableMetadata) error {
	// Header.
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}
	if _, err := io.WriteString(out, tableHeader); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}

	// Definition.
	definition, err := schema.StringifyTable(storepb.Engine_MYSQL, table)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(out, definition); err != nil {
		return err
	}
	_, err = io.WriteString(out, "\n")
	return err
}

func writeInvalidTemporaryView(out io.Writer, view *storepb.ViewMetadata) error {
	// Header.
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}
	if _, err := io.WriteString(out, tempViewHeader); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "-- `"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}
	_, err := io.WriteString(out, "` references invalid table(s) or column(s) or function(s) or definer/invoker of view lack rights to use them\n\n")
	return err
}

func writeTemporaryView(out io.Writer, view *storepb.ViewMetadata) error {
	// Header.
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}
	if _, err := io.WriteString(out, tempViewHeader); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "`\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, emptyCommentLine); err != nil {
		return err
	}

	// Definition.
	if _, err := io.WriteString(out, "CREATE VIEW `"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "` AS SELECT\n  "); err != nil {
		return err
	}
	for i, column := range view.Columns {
		if i != 0 {
			if _, err := io.WriteString(out, ",\n  "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, "1 AS `"); err != nil {
			return err
		}
		if _, err := io.WriteString(out, column.Name); err != nil {
			return err
		}
		if _, err := io.WriteString(out, "`"); err != nil {
			return err
		}
	}
	_, err := io.WriteString(out, ";\n\n")
	return err
}
