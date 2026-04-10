package mysql

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

const (
	disableUniqueAndForeignKeyCheckStmt = "SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;\nSET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;\n"
	restoreUniqueAndForeignKeyCheckStmt = "SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;\nSET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;\n"

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

	mysqlTypeBlob       = "blob"
	mysqlTypeTinyBob    = "tinyblob"
	mysqlTypeMediumBlob = "mediumblob"
	mysqlTypeLongBlob   = "longblob"
	mysqlTypeJSON       = "json"
	mysqlTypeGeometry   = "geometry"

	mysqlIndexFullText = "FULLTEXT"
	mysqlIndexSpatial  = "SPATIAL"

	mysqlNoAction = "NO ACTION"

	autoIncrementSymbol = "AUTO_INCREMENT"
)

func init() {
	schema.RegisterGetDatabaseDefinition(storepb.Engine_MYSQL, GetDatabaseDefinition)
	schema.RegisterGetDatabaseDefinition(storepb.Engine_OCEANBASE, GetDatabaseDefinition)

	schema.RegisterGetTableDefinition(storepb.Engine_MYSQL, GetTableDefinition)
	schema.RegisterGetTableDefinition(storepb.Engine_OCEANBASE, GetTableDefinition)

	schema.RegisterGetViewDefinition(storepb.Engine_MYSQL, GetViewDefinition)
	schema.RegisterGetViewDefinition(storepb.Engine_OCEANBASE, GetViewDefinition)

	schema.RegisterGetFunctionDefinition(storepb.Engine_MYSQL, GetFunctionDefinition)
	schema.RegisterGetFunctionDefinition(storepb.Engine_OCEANBASE, GetFunctionDefinition)

	schema.RegisterGetProcedureDefinition(storepb.Engine_MYSQL, GetProcedureDefinition)
	schema.RegisterGetProcedureDefinition(storepb.Engine_OCEANBASE, GetProcedureDefinition)
}

func GetDatabaseDefinition(ctx schema.GetDefinitionContext, metadata *storepb.DatabaseSchemaMetadata) (string, error) {
	if len(metadata.Schemas) == 0 {
		return "", nil
	}

	var buf strings.Builder

	if ctx.PrintHeader {
		if _, err := buf.WriteString(disableUniqueAndForeignKeyCheckStmt); err != nil {
			return "", err
		}
	}

	schema := metadata.Schemas[0]

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
			if err := writeInvalidTemporaryView(&buf, view); err != nil {
				return "", err
			}
			continue
		}
		if err := writeTemporaryView(&buf, view); err != nil {
			return "", err
		}
	}

	// Construct tables.
	for _, table := range schema.Tables {
		if err := writeTable(&buf, table); err != nil {
			return "", err
		}
	}

	// Construct views.
	for _, view := range schema.Views {
		if err := writeView(&buf, view); err != nil {
			return "", err
		}
	}

	// Construct functions.
	for _, function := range schema.Functions {
		if err := writeFunction(&buf, function); err != nil {
			return "", err
		}
	}

	// Construct procedures.
	for _, procedure := range schema.Procedures {
		if err := writeProcedure(&buf, procedure); err != nil {
			return "", err
		}
	}

	// Construct events.
	for _, event := range schema.Events {
		if err := writeEvent(&buf, event); err != nil {
			return "", err
		}
	}

	// Construct triggers.
	for _, table := range schema.Tables {
		for _, trigger := range table.Triggers {
			if err := writeTrigger(&buf, table.Name, trigger); err != nil {
				return "", err
			}
		}
	}

	if ctx.PrintHeader {
		if _, err := buf.WriteString(restoreUniqueAndForeignKeyCheckStmt); err != nil {
			return "", err
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

func GetViewDefinition(_ string, view *storepb.ViewMetadata) (string, error) {
	var buf strings.Builder
	if err := writeView(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func GetFunctionDefinition(_ string, function *storepb.FunctionMetadata) (string, error) {
	var buf strings.Builder
	if err := writeFunction(&buf, function); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func GetProcedureDefinition(_ string, procedure *storepb.ProcedureMetadata) (string, error) {
	var buf strings.Builder
	if err := writeProcedure(&buf, procedure); err != nil {
		return "", err
	}
	return buf.String(), nil
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
	if _, err := io.WriteString(out, "'"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, event.SqlMode); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "';\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, setTimezone); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "'"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, event.TimeZone); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "';\n"); err != nil {
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

func writeTrigger(out io.Writer, tableName string, trigger *storepb.TriggerMetadata) error {
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
	if _, err := io.WriteString(out, "'"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, trigger.SqlMode); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "';\n"); err != nil {
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
	if _, err := io.WriteString(out, tableName); err != nil {
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
	if err := writeAdditionalEventsIfSet(out, procedure.CharacterSetClient, procedure.CharacterSetClient, procedure.CollationConnection, procedure.SqlMode); err != nil {
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
	if err := writeAdditionalEventsIfSet(out, function.CharacterSetClient, function.CharacterSetClient, function.CollationConnection, function.SqlMode); err != nil {
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

	// Definition.
	if _, err := io.WriteString(out, "CREATE OR REPLACE VIEW `"); err != nil {
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

func writeTable(out *strings.Builder, table *storepb.TableMetadata) error {
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
	if _, err := fmt.Fprintf(out, "CREATE TABLE `%s` (\n", table.Name); err != nil {
		return err
	}

	for i, column := range table.Columns {
		if i != 0 {
			if _, err := fmt.Fprint(out, ",\n"); err != nil {
				return err
			}
		}
		if err := printColumnClause(out, column, table); err != nil {
			return err
		}
	}

	if err := printPrimaryKeyClause(out, table.Indexes); err != nil {
		return err
	}

	for _, index := range table.Indexes {
		if index.Primary {
			continue
		}
		if err := printIndexClause(out, index); err != nil {
			return err
		}
	}

	for _, fk := range table.ForeignKeys {
		if err := printForeignKeyClause(out, fk); err != nil {
			return err
		}
	}

	for _, check := range table.CheckConstraints {
		if err := printCheckClause(out, check); err != nil {
			return err
		}
	}

	if _, err := out.WriteString("\n)"); err != nil {
		return err
	}
	if table.Engine != "" {
		if _, err := fmt.Fprintf(out, " ENGINE=%s", table.Engine); err != nil {
			return err
		}
	}

	if table.Charset != "" {
		if _, err := fmt.Fprintf(out, " DEFAULT CHARSET=%s", table.Charset); err != nil {
			return err
		}
	}

	if table.Collation != "" {
		if _, err := fmt.Fprintf(out, " COLLATE=%s", table.Collation); err != nil {
			return err
		}
	}

	if table.Comment != "" {
		if _, err := fmt.Fprintf(out, " COMMENT='%s'", table.Comment); err != nil {
			return err
		}
	}

	if len(table.Partitions) > 0 {
		if err := printPartitionClause(out, table.Partitions); err != nil {
			return err
		}
	}

	_, err := io.WriteString(out, ";\n\n")
	return err
}

// Copy the logic from backend/plugin/schema/mysql/state.go.
func printPartitionClause(buf *strings.Builder, partitions []*storepb.TablePartitionMetadata) error {
	if len(partitions) == 0 {
		return nil
	}
	vsc := getVersionSpecificComment(partitions)
	curComment := vsc
	if _, err := fmt.Fprintf(buf, "%s PARTITION BY ", curComment); err != nil {
		return err
	}
	switch partitions[0].Type {
	case storepb.TablePartitionMetadata_RANGE:
		if _, err := fmt.Fprintf(buf, "RANGE (%s)", partitions[0].Expression); err != nil {
			return err
		}
	case storepb.TablePartitionMetadata_RANGE_COLUMNS:
		if _, err := fmt.Fprintf(buf, "RANGE COLUMNS (%s)", partitions[0].Expression); err != nil {
			return err
		}
	case storepb.TablePartitionMetadata_LIST:
		if _, err := fmt.Fprintf(buf, "LIST (%s)", partitions[0].Expression); err != nil {
			return err
		}
	case storepb.TablePartitionMetadata_LIST_COLUMNS:
		if _, err := fmt.Fprintf(buf, "LIST COLUMNS (%s)", partitions[0].Expression); err != nil {
			return err
		}
	case storepb.TablePartitionMetadata_HASH:
		if _, err := fmt.Fprintf(buf, "HASH (%s)", partitions[0].Expression); err != nil {
			return err
		}
	case storepb.TablePartitionMetadata_KEY:
		if _, err := fmt.Fprintf(buf, "KEY (%s)", partitions[0].Expression); err != nil {
			return err
		}
	case storepb.TablePartitionMetadata_LINEAR_HASH:
		if _, err := fmt.Fprintf(buf, "LINEAR HASH (%s)", partitions[0].Expression); err != nil {
			return err
		}
	case storepb.TablePartitionMetadata_LINEAR_KEY:
		if _, err := fmt.Fprintf(buf, "LINEAR KEY (%s)", partitions[0].Expression); err != nil {
			return err
		}
	default:
		return errors.Errorf("unknown partition type: %v", partitions[0].Type)
	}

	useDefault := int64(0)
	if partitions[0].UseDefault != "" {
		var err error
		useDefault, err = strconv.ParseInt(partitions[0].UseDefault, 10, 64)
		if err != nil {
			return err
		}
	}
	if useDefault != 0 {
		if _, err := fmt.Fprintf(buf, "\nPARTITIONS %d", useDefault); err != nil {
			return err
		}
	}

	if len(partitions[0].Subpartitions) > 0 {
		if _, err := fmt.Fprint(buf, "\nSUBPARTITION BY "); err != nil {
			return err
		}
		switch partitions[0].Subpartitions[0].Type {
		case storepb.TablePartitionMetadata_HASH:
			if _, err := fmt.Fprintf(buf, "HASH (%s)", partitions[0].Subpartitions[0].Expression); err != nil {
				return err
			}
		case storepb.TablePartitionMetadata_LINEAR_HASH:
			if _, err := fmt.Fprintf(buf, "LINEAR HASH (%s)", partitions[0].Subpartitions[0].Expression); err != nil {
				return err
			}
		case storepb.TablePartitionMetadata_KEY:
			if _, err := fmt.Fprintf(buf, "KEY (%s)", partitions[0].Subpartitions[0].Expression); err != nil {
				return err
			}
		case storepb.TablePartitionMetadata_LINEAR_KEY:
			if _, err := fmt.Fprintf(buf, "LINEAR KEY (%s)", partitions[0].Subpartitions[0].Expression); err != nil {
				return err
			}
		default:
			return errors.Errorf("invalid subpartition type: %v", partitions[0].Subpartitions[0].Type)
		}
	}

	subUseDefault := 0
	if len(partitions[0].Subpartitions) > 0 && partitions[0].Subpartitions[0].UseDefault != "" {
		var err error
		subUseDefault, err = strconv.Atoi(partitions[0].Subpartitions[0].UseDefault)
		if err != nil {
			return err
		}
	}

	if subUseDefault != 0 {
		if _, err := fmt.Fprintf(buf, "\nSUBPARTITIONS %d", subUseDefault); err != nil {
			return err
		}
	}

	if useDefault == 0 {
		if _, err := fmt.Fprint(buf, "\n("); err != nil {
			return err
		}
		preposition, err := getPrepositionByType(partitions[0].Type)
		if err != nil {
			return err
		}
		for i, partition := range partitions {
			if i != 0 {
				if _, err := fmt.Fprint(buf, ",\n "); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(buf, "PARTITION %s", partition.Name); err != nil {
				return err
			}
			if preposition != "" {
				if partition.Value != "MAXVALUE" {
					if _, err := fmt.Fprintf(buf, " VALUES %s (%s)", preposition, partition.Value); err != nil {
						return err
					}
				} else {
					if _, err := fmt.Fprintf(buf, " VALUES %s %s", preposition, partition.Value); err != nil {
						return err
					}
				}
			}

			if subUseDefault == 0 && len(partition.Subpartitions) > 0 {
				if _, err := fmt.Fprint(buf, "\n ("); err != nil {
					return err
				}
				for j, subPartition := range partition.Subpartitions {
					if _, err := fmt.Fprintf(buf, "SUBPARTITION %s", subPartition.Name); err != nil {
						return err
					}
					if err := writePartitionOptions(buf); err != nil {
						return err
					}
					if j == len(partition.Subpartitions)-1 {
						if _, err := fmt.Fprint(buf, ")"); err != nil {
							return err
						}
					} else {
						if _, err := fmt.Fprint(buf, ",\n  "); err != nil {
							return err
						}
					}
				}
			} else {
				if err := writePartitionOptions(buf); err != nil {
					return err
				}
			}

			if i == len(partitions)-1 {
				if _, err := fmt.Fprint(buf, ")"); err != nil {
					return err
				}
			}
		}
	}

	if _, err := fmt.Fprint(buf, " */"); err != nil {
		return err
	}

	return nil
}

func getPrepositionByType(tp storepb.TablePartitionMetadata_Type) (string, error) {
	switch tp {
	case storepb.TablePartitionMetadata_RANGE:
		return "LESS THAN", nil
	case storepb.TablePartitionMetadata_RANGE_COLUMNS:
		return "LESS THAN", nil
	case storepb.TablePartitionMetadata_LIST:
		return "IN", nil
	case storepb.TablePartitionMetadata_LIST_COLUMNS:
		return "IN", nil
	case storepb.TablePartitionMetadata_HASH, storepb.TablePartitionMetadata_KEY, storepb.TablePartitionMetadata_LINEAR_HASH, storepb.TablePartitionMetadata_LINEAR_KEY:
		return "", nil
	default:
		return "", errors.Errorf("unsupported partition type: %v", tp)
	}
}

func writePartitionOptions(buf io.StringWriter) error {
	/*
		int err = 0;
		err += add_space(fptr);
		if (p_elem->tablespace_name) {
			err += add_string(fptr, "TABLESPACE = ");
			err += add_ident_string(fptr, p_elem->tablespace_name);
			err += add_space(fptr);
		}
		if (p_elem->nodegroup_id != UNDEF_NODEGROUP)
			err += add_keyword_int(fptr, "NODEGROUP", (longlong)p_elem->nodegroup_id);
		if (p_elem->part_max_rows)
			err += add_keyword_int(fptr, "MAX_ROWS", (longlong)p_elem->part_max_rows);
		if (p_elem->part_min_rows)
			err += add_keyword_int(fptr, "MIN_ROWS", (longlong)p_elem->part_min_rows);
		if (!(current_thd->variables.sql_mode & MODE_NO_DIR_IN_CREATE)) {
			if (p_elem->data_file_name)
			err += add_keyword_path(fptr, "DATA DIRECTORY", p_elem->data_file_name);
			if (p_elem->index_file_name)
			err += add_keyword_path(fptr, "INDEX DIRECTORY", p_elem->index_file_name);
		}
		if (p_elem->part_comment)
			err += add_keyword_string(fptr, "COMMENT", true, p_elem->part_comment);
		return err + add_engine(fptr, p_elem->engine_type);
	*/
	// TODO(zp): Get all the partition options from the metadata is too complex, just write ENGINE=InnoDB for now.
	if _, err := buf.WriteString(" ENGINE=InnoDB"); err != nil {
		return err
	}

	return nil
}

func getVersionSpecificComment(partitions []*storepb.TablePartitionMetadata) string {
	if len(partitions) == 0 {
		return ""
	}
	partition := partitions[0]
	if partition.Type == storepb.TablePartitionMetadata_RANGE_COLUMNS || partition.Type == storepb.TablePartitionMetadata_LIST_COLUMNS {
		// MySQL introduce columns partitioning in 5.5+
		return "\n/*!50500"
	}
	return "\n/*!50100"
}

func printCheckClause(buf *strings.Builder, check *storepb.CheckConstraintMetadata) error {
	if _, err := fmt.Fprintf(buf, ",\n  CONSTRAINT `%s` CHECK %s", check.Name, check.Expression); err != nil {
		return err
	}
	return nil
}

func printForeignKeyClause(buf *strings.Builder, fk *storepb.ForeignKeyMetadata) error {
	if _, err := fmt.Fprintf(buf, ",\n  CONSTRAINT `%s` FOREIGN KEY (", fk.Name); err != nil {
		return err
	}

	for i, column := range fk.Columns {
		if i != 0 {
			if _, err := fmt.Fprint(buf, ", "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(buf, "`%s`", column); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(buf, ") REFERENCES `%s` (", fk.ReferencedTable); err != nil {
		return err
	}

	for i, column := range fk.ReferencedColumns {
		if i != 0 {
			if _, err := fmt.Fprint(buf, ", "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(buf, "`%s`", column); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(buf, ")"); err != nil {
		return err
	}

	if fk.OnDelete != "" && !strings.EqualFold(fk.OnDelete, mysqlNoAction) {
		if _, err := fmt.Fprintf(buf, " ON DELETE %s", fk.OnDelete); err != nil {
			return err
		}
	}

	if fk.OnUpdate != "" && !strings.EqualFold(fk.OnUpdate, mysqlNoAction) {
		if _, err := fmt.Fprintf(buf, " ON UPDATE %s", fk.OnUpdate); err != nil {
			return err
		}
	}

	return nil
}

func printIndexClause(buf *strings.Builder, index *storepb.IndexMetadata) error {
	if index.Primary {
		return nil
	}

	if _, err := fmt.Fprint(buf, ",\n  "); err != nil {
		return err
	}

	if index.Unique {
		if _, err := fmt.Fprint(buf, "UNIQUE "); err != nil {
			return err
		}
	} else if strings.EqualFold(index.Type, mysqlIndexFullText) {
		if _, err := fmt.Fprint(buf, "FULLTEXT "); err != nil {
			return err
		}
	} else if strings.EqualFold(index.Type, mysqlIndexSpatial) {
		if _, err := fmt.Fprint(buf, "SPATIAL "); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(buf, "KEY `%s` (", index.Name); err != nil {
		return err
	}

	for i, expr := range index.Expressions {
		if i != 0 {
			if _, err := fmt.Fprint(buf, ", "); err != nil {
				return err
			}
		}
		keyLength := int64(-1)
		descending := false
		if len(index.KeyLength) > i {
			keyLength = index.KeyLength[i]
		}
		if len(index.Descending) > i {
			descending = index.Descending[i]
		}
		if err := printIndexKeyPart(buf, expr, keyLength, descending); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(buf, ")"); err != nil {
		return err
	}

	return nil
}

func printIndexKeyPart(buf *strings.Builder, expr string, length int64, descending bool) error {
	if len(expr) == 0 {
		return errors.New("index key part expression is empty")
	}
	if expr[0] == '(' && expr[len(expr)-1] == ')' {
		if _, err := buf.WriteString(expr); err != nil {
			return err
		}
	} else {
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
		if _, err := buf.WriteString(expr); err != nil {
			return err
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
	}
	if length > 0 {
		if _, err := buf.WriteString("("); err != nil {
			return err
		}
		if _, err := buf.WriteString(strconv.FormatInt(length, 10)); err != nil {
			return err
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	}
	if descending {
		if _, err := buf.WriteString(" DESC"); err != nil {
			return err
		}
	}
	return nil
}

func printPrimaryKeyClause(buf *strings.Builder, indexes []*storepb.IndexMetadata) error {
	for _, index := range indexes {
		if index.Primary {
			if _, err := fmt.Fprint(buf, ",\n  PRIMARY KEY ("); err != nil {
				return err
			}
			for i, column := range index.Expressions {
				if i != 0 {
					if _, err := fmt.Fprint(buf, ", "); err != nil {
						return err
					}
				}
				keyLength := int64(-1)
				descending := false
				if len(index.KeyLength) > i {
					keyLength = index.KeyLength[i]
				}
				if len(index.Descending) > i {
					descending = index.Descending[i]
				}
				if err := printIndexKeyPart(buf, column, keyLength, descending); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprint(buf, ")"); err != nil {
				return err
			}
			return nil
		}
	}

	return nil
}

func isAutoIncrement(column *storepb.ColumnMetadata) bool {
	return strings.EqualFold(column.GetDefault(), autoIncrementSymbol)
}

func printColumnClause(buf *strings.Builder, column *storepb.ColumnMetadata, table *storepb.TableMetadata) error {
	if _, err := fmt.Fprintf(buf, "  `%s` %s", column.Name, column.Type); err != nil {
		return err
	}

	if column.CharacterSet != "" && column.CharacterSet != table.Charset {
		if _, err := fmt.Fprintf(buf, " CHARACTER SET %s", column.CharacterSet); err != nil {
			return err
		}
	}

	if column.Collation != "" && column.Collation != table.Collation {
		if _, err := fmt.Fprintf(buf, " COLLATE %s", column.Collation); err != nil {
			return err
		}
	}

	if column.Generation != nil && column.Generation.Expression != "" {
		if _, err := fmt.Fprintf(buf, " GENERATED ALWAYS AS (%s) ", column.Generation.Expression); err != nil {
			return err
		}
		switch column.Generation.Type {
		case storepb.GenerationMetadata_TYPE_STORED:
			if _, err := fmt.Fprint(buf, "STORED"); err != nil {
				return err
			}
		case storepb.GenerationMetadata_TYPE_VIRTUAL:
			if _, err := fmt.Fprint(buf, "VIRTUAL"); err != nil {
				return err
			}
		default:
			// Unknown generation type
		}
	}

	if !column.Nullable {
		if _, err := fmt.Fprint(buf, " NOT NULL"); err != nil {
			return err
		}
	}

	if err := printDefaultClause(buf, column); err != nil {
		return err
	}

	// Handle auto_increment.
	if isAutoIncrement(column) {
		if _, err := fmt.Fprintf(buf, " %s", autoIncrementSymbol); err != nil {
			return err
		}
	}

	if column.OnUpdate != "" {
		if _, err := fmt.Fprintf(buf, " ON UPDATE %s", column.OnUpdate); err != nil {
			return err
		}
	}
	if column.Comment != "" {
		if _, err := fmt.Fprintf(buf, " COMMENT '%s'", column.Comment); err != nil {
			return err
		}
	}
	return nil
}

func printDefaultClause(buf *strings.Builder, column *storepb.ColumnMetadata) error {
	// Check if column has any default value
	hasDefault := column.Default != ""
	if !hasDefault {
		return nil
	}

	if column.Default == "NULL" {
		if !column.Nullable || !typeSupportsDefaultValue(column.Type) {
			// If the column is not nullable, then the default value should not be null.
			// For this case, we should not print the default clause.
			return nil
		}
		if column.Generation != nil && column.Generation.Expression != "" {
			return nil
		}
		if _, err := fmt.Fprint(buf, " DEFAULT NULL"); err != nil {
			return err
		}
		return nil
	}

	if column.Default != "" {
		if isAutoIncrement(column) {
			// If the default value is auto_increment, then we should not print the default clause.
			// We'll handle this in the following AUTO_INCREMENT clause.
			return nil
		}
		if _, err := fmt.Fprintf(buf, " DEFAULT %s", column.Default); err != nil {
			return err
		}
	}

	return nil
}

func typeSupportsDefaultValue(tp string) bool {
	switch strings.ToLower(tp) {
	case mysqlTypeBlob, mysqlTypeTinyBob, mysqlTypeMediumBlob, mysqlTypeLongBlob, mysqlTypeJSON, mysqlTypeGeometry:
		return false
	default:
		return true
	}
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

func writeAdditionalEventsIfSet(out io.Writer, characterSetClient, characterSetResult, collationConnection, sqlMode string) error {
	events := []struct {
		condition bool
		prefix    string
		value     string
	}{
		{characterSetClient != "", setCharacterSetClient, characterSetClient},
		{characterSetResult != "", setCharacterSetResult, characterSetResult},
		{collationConnection != "", setCollation, collationConnection},
	}

	for _, event := range events {
		if event.condition {
			if _, err := io.WriteString(out, event.prefix); err != nil {
				return err
			}
			if _, err := io.WriteString(out, event.value); err != nil {
				return err
			}
			if _, err := io.WriteString(out, ";\n"); err != nil {
				return err
			}
		}
	}
	if sqlMode != "" {
		if _, err := io.WriteString(out, setSQLMode); err != nil {
			return err
		}
		if _, err := io.WriteString(out, "'"); err != nil {
			return err
		}
		if _, err := io.WriteString(out, sqlMode); err != nil {
			return err
		}
		if _, err := io.WriteString(out, "';\n"); err != nil {
			return err
		}
	}

	return nil
}
