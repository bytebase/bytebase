package mysql

import (
	"cmp"
	"fmt"
	"io"
	"slices"
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

	if ctx.SDLFormat {
		return getSDLFormat(metadata)
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

	isFullText := strings.EqualFold(index.Type, mysqlIndexFullText)
	isSpatial := strings.EqualFold(index.Type, mysqlIndexSpatial)
	if index.Unique {
		if _, err := fmt.Fprint(buf, "UNIQUE "); err != nil {
			return err
		}
	} else if isFullText {
		if _, err := fmt.Fprint(buf, "FULLTEXT "); err != nil {
			return err
		}
	} else if isSpatial {
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
		// SPATIAL and FULLTEXT index parts do not take a key-part prefix length. The sync
		// records SUB_PART for spatial parts (e.g. 32 for a POINT) even though SHOW CREATE
		// renders none; emitting `(32)` would diverge from the user form and DROP+ADD the
		// index on every no-op. Suppress the prefix for those index types.
		if err := printIndexKeyPart(buf, expr, keyLength, descending, isSpatial || isFullText); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(buf, ")"); err != nil {
		return err
	}

	// INVISIBLE secondary index (MySQL 8.0). The version-gated executable comment is parsed
	// back by omni's loader, so the dumped `from` carries invisibility and the diff against
	// an INVISIBLE user `to` is a no-op instead of a DROP+ADD.
	if !index.Visible {
		if _, err := fmt.Fprint(buf, " /*!80000 INVISIBLE */"); err != nil {
			return err
		}
	}

	return nil
}

func printIndexKeyPart(buf *strings.Builder, expr string, length int64, descending bool, suppressPrefix bool) error {
	if len(expr) == 0 {
		return errors.New("index key part expression is empty")
	}
	if expr[0] == '(' && expr[len(expr)-1] == ')' {
		if _, err := buf.WriteString(normalizeFunctionalIndexExpr(expr)); err != nil {
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
	if length > 0 && !suppressPrefix {
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

// normalizeFunctionalIndexExpr rewrites a synced functional (expression) index key part into
// the form the omni SDL loader can parse. The information_schema.STATISTICS.EXPRESSION text
// MySQL records for an `((expr))` key part:
//   - backslash-escapes the single quotes around string literals (e.g. json_extract(`tags`,
//     _utf8mb4\'$.ids\')), which the omni lexer rejects (`syntax error at or near "\"`); and
//   - prefixes string literals with a charset introducer (`_utf8mb4'…'`, `_latin1'…'`, …)
//     reflecting the connection charset, which the omni expression parser does not accept in
//     this position (`unexpected token`).
//
// Both make LoadSDL fail outright on the dumped source even though omni accepts the canonical
// user spelling (e.g. `((cast(json_extract(`tags`,'$.ids') as unsigned array)))`). This
// unescapes the quotes and strips the introducer so the emitted expression is omni-parseable;
// omni's functional-index normalizer then canonicalizes the dumped and user forms identically,
// yielding an empty no-op. The transform is string-literal aware: the introducer is removed
// only when it immediately precedes a string literal, and literal contents are left untouched.
func normalizeFunctionalIndexExpr(expr string) string {
	return stripCharsetIntroducers(unescapeFunctionalIndexQuotes(expr))
}

// unescapeFunctionalIndexQuotes undoes the backslash escaping that
// information_schema.STATISTICS.EXPRESSION applies to single quotes (\' -> ') and backslashes
// (\\ -> \). MySQL stores the expression text escaped; the omni lexer wants the raw SQL.
func unescapeFunctionalIndexQuotes(expr string) string {
	if !strings.Contains(expr, `\`) {
		return expr
	}
	var b strings.Builder
	b.Grow(len(expr))
	for i := 0; i < len(expr); i++ {
		if expr[i] == '\\' && i+1 < len(expr) && (expr[i+1] == '\'' || expr[i+1] == '\\') {
			b.WriteByte(expr[i+1])
			i++
			continue
		}
		b.WriteByte(expr[i])
	}
	return b.String()
}

// stripCharsetIntroducers removes a MySQL charset introducer token (an underscore-prefixed
// identifier such as `_utf8mb4` or `_latin1`) when it immediately precedes a single-quoted
// string literal, e.g. `_utf8mb4'$.ids'` -> `'$.ids'`. The scan tracks whether it is inside a
// '…'/"…" string literal (honoring backslash escapes and the doubled-quote escape) so that an
// underscore sequence occurring inside literal text is never mistaken for an introducer and
// literal contents are preserved byte for byte. Backtick-quoted identifiers are passed through
// untouched. Only an introducer attached to a string literal is stripped; underscores that are
// part of an ordinary identifier (`my_col`, a leading-underscore name not followed by a quote)
// are left intact.
func stripCharsetIntroducers(expr string) string {
	if !strings.Contains(expr, "_") {
		return expr
	}
	var b strings.Builder
	b.Grow(len(expr))
	var quote byte // open delimiter of the literal currently being copied, or 0 when outside one
	for i := 0; i < len(expr); {
		c := expr[i]
		if quote != 0 {
			b.WriteByte(c)
			switch {
			case c == '\\' && i+1 < len(expr):
				b.WriteByte(expr[i+1])
				i += 2
				continue
			case c == quote:
				if i+1 < len(expr) && expr[i+1] == quote {
					b.WriteByte(expr[i+1])
					i += 2
					continue
				}
				quote = 0
			default:
			}
			i++
			continue
		}
		switch c {
		case '\'', '"':
			quote = c
			b.WriteByte(c)
			i++
		case '`':
			// Copy a backtick-quoted identifier verbatim so an underscore inside it is never
			// treated as an introducer.
			b.WriteByte(c)
			i++
			for i < len(expr) {
				b.WriteByte(expr[i])
				if expr[i] == '`' {
					i++
					break
				}
				i++
			}
		case '_':
			if n := charsetIntroducerLen(expr[i:]); n > 0 {
				// Drop the `_<charset>` introducer; the following string literal is copied on the
				// next iterations.
				i += n
				continue
			}
			b.WriteByte(c)
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// charsetIntroducerLen returns the byte length of a leading charset-introducer token in s
// when s begins with `_<charset>` immediately followed by a single-quoted string literal,
// or 0 otherwise. The charset name accepts the ASCII letters/digits MySQL charset names use
// (e.g. utf8mb4, latin1, big5); the terminating quote is required so a bare leading-underscore
// identifier is not stripped.
func charsetIntroducerLen(s string) int {
	if len(s) < 2 || s[0] != '_' {
		return 0
	}
	i := 1
	for i < len(s) && isCharsetNameByte(s[i]) {
		i++
	}
	if i == 1 || i >= len(s) || s[i] != '\'' {
		return 0
	}
	return i
}

func isCharsetNameByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
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
				// Primary keys are never SPATIAL/FULLTEXT, so prefix lengths are valid here.
				if err := printIndexKeyPart(buf, column, keyLength, descending, false); err != nil {
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
	if _, err := fmt.Fprintf(buf, "  `%s` %s", column.Name, normalizeColumnType(column.Type)); err != nil {
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

	// SRID for spatial columns (MySQL 8.0). Emitted after NOT NULL and before DEFAULT to
	// match the SHOW CREATE / omni-canonical position; the version-gated executable comment
	// is spliced back as live SQL by omni's loader so the dumped `from` carries the SRID and
	// the diff against a user `to` with the same SRID is a no-op.
	writeColumnSRIDAttribute(buf, column)

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

	// INVISIBLE column (MySQL 8.0.23+). Emitted last to match the SHOW CREATE / omni-canonical
	// position; like SRID the version-gated comment is parsed back by omni's loader so the
	// dumped `from` carries invisibility and the diff against an INVISIBLE user `to` is a no-op.
	writeColumnInvisibleAttribute(buf, column)
	return nil
}

// writeColumnSRIDAttribute emits the ` /*!80003 SRID n */` column attribute when the
// column carries an explicit SRID — including the valid SRID 0, which presence (not a
// zero sentinel) distinguishes from "no SRID". Shared by the SDL dumper
// (printColumnClause) and the legacy migration generator (writeAddColumn /
// writeModifyColumn) so both render the same canonical form.
func writeColumnSRIDAttribute(buf *strings.Builder, column *storepb.ColumnMetadata) {
	if column.Srid != nil {
		_, _ = fmt.Fprintf(buf, " /*!80003 SRID %d */", column.GetSrid())
	}
}

// writeColumnInvisibleAttribute emits the ` /*!80023 INVISIBLE */` column attribute for
// invisible columns (MySQL 8.0.23+), as the final attribute to match the SHOW CREATE
// position. Shared by the SDL dumper and the legacy migration generator.
func writeColumnInvisibleAttribute(buf *strings.Builder, column *storepb.ColumnMetadata) {
	if column.IsInvisible {
		_, _ = buf.WriteString(" /*!80023 INVISIBLE */")
	}
}

// normalizeColumnType rewrites engine-specific column-type spellings the omni SDL parser
// does not accept into the canonical spelling it does. MySQL 8.0's SHOW CREATE /
// information_schema render GEOMETRYCOLLECTION as the synonym `geomcollection`, which the
// omni loader rejects with "expected data type"; the canonical `geometrycollection`
// (what 5.7 already dumps) round-trips. The two spellings denote the same type, so this is
// a pure rename with no semantic change.
func normalizeColumnType(columnType string) string {
	if strings.EqualFold(columnType, "geomcollection") {
		return "geometrycollection"
	}
	return columnType
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
		if _, err := fmt.Fprintf(buf, " DEFAULT %s", renderColumnDefault(column)); err != nil {
			return err
		}
	}

	return nil
}

// renderColumnDefault returns the DEFAULT expression text to emit for a column.
//
// For a BIT column, the synced COLUMN_DEFAULT is the bit-literal text (e.g. b'0'),
// which the sync stores QUOTE()-escaped as a string literal in ColumnMetadata.Default
// (the bit literal wrapped in single quotes with the inner quotes backslash-escaped).
// Emitting that verbatim produces a quoted string default that the omni loader takes as a
// string — never matching the user's BIT literal b'0', so the no-op phantom-re-emits
// MODIFY ... DEFAULT b'0' forever. Real mysqldump renders the bit literal unquoted
// (DEFAULT b'0'); match it by recovering the inner literal and emitting it without the
// surrounding quotes when the column is BIT and the recovered text is a bit/hex literal
// (b'…'/0x…/x'…'). Any other default (including a genuine string) is emitted verbatim.
func renderColumnDefault(column *storepb.ColumnMetadata) string {
	if isBitColumnType(column.Type) {
		if lit, ok := bitLiteralFromQuotedDefault(column.Default); ok {
			return lit
		}
	}
	return column.Default
}

// isBitColumnType reports whether a column type spelling is BIT (with or without a
// display width, e.g. "bit" or "bit(8)").
func isBitColumnType(columnType string) bool {
	lower := strings.ToLower(strings.TrimSpace(columnType))
	return lower == "bit" || strings.HasPrefix(lower, "bit(")
}

// bitLiteralFromQuotedDefault recovers a bit/hex literal from a stored column default and
// reports whether the default denotes one. The default may already be a bare literal
// (b'0', 0x0A, x'1f') or the QUOTE()-escaped string form the MySQL sync stores for static
// defaults (the literal wrapped in single quotes with inner quotes backslash-escaped). It
// strips at most one layer of surrounding single quotes and the QUOTE() backslash escaping,
// then recognizes the MySQL bit/hex literal spellings
// (b'…' / B'…' bit, 0x… hex, x'…' / X'…' hex). On success it returns the canonical literal
// text to emit unquoted; otherwise it returns ("", false) and the caller emits the default
// verbatim.
func bitLiteralFromQuotedDefault(def string) (string, bool) {
	if isBitOrHexLiteral(def) {
		return def, true
	}
	if len(def) >= 2 && def[0] == '\'' && def[len(def)-1] == '\'' {
		inner := unescapeQuotedDefault(def[1 : len(def)-1])
		if isBitOrHexLiteral(inner) {
			return inner, true
		}
	}
	return "", false
}

// isBitOrHexLiteral reports whether s is a MySQL bit literal (b'…' / B'…') or hex literal
// (0x… / 0X… / x'…' / X'…') with only valid digits between the delimiters.
func isBitOrHexLiteral(s string) bool {
	switch {
	case len(s) >= 3 && (s[0] == 'b' || s[0] == 'B') && s[1] == '\'' && s[len(s)-1] == '\'':
		return allDigitsIn(s[2:len(s)-1], "01")
	case len(s) >= 3 && (s[0] == 'x' || s[0] == 'X') && s[1] == '\'' && s[len(s)-1] == '\'':
		return allDigitsIn(s[2:len(s)-1], "0123456789abcdefABCDEF")
	case len(s) >= 3 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X'):
		return allDigitsIn(s[2:], "0123456789abcdefABCDEF")
	default:
		return false
	}
}

func allDigitsIn(s, allowed string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune(allowed, rune(s[i])) {
			return false
		}
	}
	return true
}

// unescapeQuotedDefault undoes the QUOTE() backslash escaping (\' -> ', \\ -> \) the MySQL
// sync applies to static defaults. Only these two sequences appear inside a bit/hex literal,
// so the full QUOTE() escape set is intentionally not reproduced here.
func unescapeQuotedDefault(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && (s[i+1] == '\'' || s[i+1] == '\\') {
			b.WriteByte(s[i+1])
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
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

// getSDLFormat emits a canonical, deterministic SDL dump of the database metadata
// such that the omni catalog LoadSDL round-trips it. Unlike the mysqldump-style
// output produced by GetDatabaseDefinition, the SDL dump:
//   - contains only the constructive CREATE statements LoadSDL accepts (no SET
//     headers, no DELIMITER blocks, no temporary-view scaffolding, no version-gated
//     /*!50100 PARTITION */ executable comments);
//   - emits objects and intra-table clauses in a stable order (tables and views
//     sorted by name; secondary indexes, foreign keys, and check constraints sorted
//     by name) so the same metadata always produces byte-identical SDL;
//   - strips the volatile AUTO_INCREMENT=N table option (a live counter, never
//     schema);
//   - emits stored routines, triggers, and events as their canonical CREATE forms
//     (DEFINER omitted, no DELIMITER blocks — omni's LoadSDL splitter is BEGIN…END
//     aware and the routine/trigger/event differs ignore DEFINER).
//
// Emission order is dependency-safe so every CREATE resolves against objects loaded
// before it: tables → functions → procedures → views → triggers → events. Functions
// precede views/triggers because a view or trigger body may call a stored function;
// triggers and events follow their tables because the omni loader requires a
// trigger's owning table (and any table an event/trigger body touches) to exist.
//
// Type, charset/collation, and default canonicalization are intentionally NOT done
// here: the omni Diff path resolves both the dumped source and the user target
// through the same Normalizer (CanonicalColumn), so faithful emission is sufficient
// for the no-op idempotence property — the canonical comparison happens in omni.
func getSDLFormat(metadata *storepb.DatabaseSchemaMetadata) (string, error) {
	var buf strings.Builder

	schema := metadata.Schemas[0]

	tables := make([]*storepb.TableMetadata, 0, len(schema.Tables))
	for _, table := range schema.Tables {
		if table.SkipDump {
			continue
		}
		tables = append(tables, table)
	}
	slices.SortFunc(tables, func(a, b *storepb.TableMetadata) int { return cmp.Compare(a.Name, b.Name) })

	for _, table := range tables {
		if err := writeTableSDL(&buf, table); err != nil {
			return "", err
		}
	}

	functions := make([]*storepb.FunctionMetadata, 0, len(schema.Functions))
	for _, function := range schema.Functions {
		if function.SkipDump {
			continue
		}
		functions = append(functions, function)
	}
	slices.SortFunc(functions, func(a, b *storepb.FunctionMetadata) int { return cmp.Compare(a.Name, b.Name) })
	for _, function := range functions {
		if err := writeRoutineSDL(&buf, function.Definition); err != nil {
			return "", err
		}
	}

	procedures := make([]*storepb.ProcedureMetadata, 0, len(schema.Procedures))
	for _, procedure := range schema.Procedures {
		if procedure.SkipDump {
			continue
		}
		procedures = append(procedures, procedure)
	}
	slices.SortFunc(procedures, func(a, b *storepb.ProcedureMetadata) int { return cmp.Compare(a.Name, b.Name) })
	for _, procedure := range procedures {
		if err := writeRoutineSDL(&buf, procedure.Definition); err != nil {
			return "", err
		}
	}

	views := make([]*storepb.ViewMetadata, 0, len(schema.Views))
	for _, view := range schema.Views {
		if view.SkipDump {
			continue
		}
		views = append(views, view)
	}
	slices.SortFunc(views, func(a, b *storepb.ViewMetadata) int { return cmp.Compare(a.Name, b.Name) })

	for _, view := range views {
		if err := writeViewSDL(&buf, metadata.Name, view); err != nil {
			return "", err
		}
	}

	// Triggers hang off tables (TableMetadata.Triggers); emit them after every table
	// in deterministic (table, trigger) order.
	for _, table := range tables {
		triggers := make([]*storepb.TriggerMetadata, 0, len(table.Triggers))
		for _, trigger := range table.Triggers {
			if trigger.SkipDump {
				continue
			}
			triggers = append(triggers, trigger)
		}
		slices.SortFunc(triggers, func(a, b *storepb.TriggerMetadata) int { return cmp.Compare(a.Name, b.Name) })
		for _, trigger := range triggers {
			if err := writeTriggerSDL(&buf, table.Name, trigger); err != nil {
				return "", err
			}
		}
	}

	events := make([]*storepb.EventMetadata, 0, len(schema.Events))
	events = append(events, schema.Events...)
	slices.SortFunc(events, func(a, b *storepb.EventMetadata) int { return cmp.Compare(a.Name, b.Name) })
	for _, event := range events {
		if err := writeEventSDL(&buf, event); err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}

// writeTableSDL emits a single CREATE TABLE statement in canonical SDL form with
// deterministic clause ordering. It reuses the same column/index/foreign-key/check
// clause writers as the mysqldump path; only the statement framing and the sorting
// of sub-objects differ.
func writeTableSDL(buf *strings.Builder, table *storepb.TableMetadata) error {
	if _, err := fmt.Fprintf(buf, "CREATE TABLE `%s` (\n", table.Name); err != nil {
		return err
	}

	for i, column := range table.Columns {
		if i != 0 {
			if _, err := fmt.Fprint(buf, ",\n"); err != nil {
				return err
			}
		}
		if err := printColumnClause(buf, column, table); err != nil {
			return err
		}
	}

	if err := printPrimaryKeyClause(buf, table.Indexes); err != nil {
		return err
	}

	secondaryIndexes := make([]*storepb.IndexMetadata, 0, len(table.Indexes))
	for _, index := range table.Indexes {
		if index.Primary {
			continue
		}
		secondaryIndexes = append(secondaryIndexes, index)
	}
	slices.SortFunc(secondaryIndexes, func(a, b *storepb.IndexMetadata) int { return cmp.Compare(a.Name, b.Name) })
	for _, index := range secondaryIndexes {
		if err := printIndexClause(buf, index); err != nil {
			return err
		}
	}

	foreignKeys := make([]*storepb.ForeignKeyMetadata, len(table.ForeignKeys))
	copy(foreignKeys, table.ForeignKeys)
	slices.SortFunc(foreignKeys, func(a, b *storepb.ForeignKeyMetadata) int { return cmp.Compare(a.Name, b.Name) })
	for _, fk := range foreignKeys {
		if err := printForeignKeyClause(buf, fk); err != nil {
			return err
		}
	}

	checks := make([]*storepb.CheckConstraintMetadata, len(table.CheckConstraints))
	copy(checks, table.CheckConstraints)
	slices.SortFunc(checks, func(a, b *storepb.CheckConstraintMetadata) int { return cmp.Compare(a.Name, b.Name) })
	for _, check := range checks {
		if err := printCheckClause(buf, check); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString("\n)"); err != nil {
		return err
	}
	if table.Engine != "" {
		if _, err := fmt.Fprintf(buf, " ENGINE=%s", table.Engine); err != nil {
			return err
		}
	}
	// CREATE_OPTIONS (ROW_FORMAT, KEY_BLOCK_SIZE, COMPRESSION, STATS_PERSISTENT, ...) are
	// recorded schema the sync captures into TableMetadata.CreateOptions; emit them in the
	// canonical SHOW CREATE position (after ENGINE/AUTO_INCREMENT, before DEFAULT CHARSET) so
	// the dumped `from` keeps an explicit ROW_FORMAT and the omni diff against a user `to`
	// carrying the same option is a no-op (entry: TestSDLStressTableCreateOptions). The
	// synthetic `partitioned` token is dropped — partitioning is surfaced via Partitions, not
	// as a literal table option, and emitting it would be invalid DDL.
	if opts := filterCreateOptions(table.CreateOptions); opts != "" {
		if _, err := fmt.Fprintf(buf, " %s", opts); err != nil {
			return err
		}
	}
	if table.Charset != "" {
		if _, err := fmt.Fprintf(buf, " DEFAULT CHARSET=%s", table.Charset); err != nil {
			return err
		}
	}
	if table.Collation != "" {
		if _, err := fmt.Fprintf(buf, " COLLATE=%s", table.Collation); err != nil {
			return err
		}
	}
	if table.Comment != "" {
		if _, err := fmt.Fprintf(buf, " COMMENT='%s'", table.Comment); err != nil {
			return err
		}
	}

	// Partitioning is a table-level clause that follows the table options. The omni
	// loader accepts the version-gated /*!50100 PARTITION BY … */ executable comment
	// (its lexer splices the inner text back as live SQL), and the omni partition
	// differ canonicalizes the expression and per-partition ENGINE, so reusing the
	// mysqldump partition writer round-trips as a no-op without an SDL-specific form.
	if len(table.Partitions) > 0 {
		if err := printPartitionClause(buf, table.Partitions); err != nil {
			return err
		}
	}

	_, err := io.WriteString(buf, ";\n\n")
	return err
}

// filterCreateOptions returns the table CREATE_OPTIONS string with the synthetic
// `partitioned` token removed. MySQL's information_schema.TABLES.CREATE_OPTIONS folds the
// fact that a table is partitioned into a literal `partitioned` token alongside the real
// declared options (e.g. `partitioned`, or `row_format=DYNAMIC partitioned`). Partitioning
// is emitted separately via the PARTITION BY clause from TableMetadata.Partitions, so the
// token must not be rendered as a table option (it is not valid table-option DDL). The
// remaining tokens (row_format=…, key_block_size=…, …) are already valid `name=value`
// table-option syntax that the omni loader parses case-insensitively, so they are returned
// verbatim and joined by single spaces.
func filterCreateOptions(createOptions string) string {
	if createOptions == "" {
		return ""
	}
	kept := make([]string, 0, 4)
	for _, tok := range strings.Fields(createOptions) {
		if strings.EqualFold(tok, "partitioned") {
			continue
		}
		kept = append(kept, tok)
	}
	return strings.Join(kept, " ")
}

// writeViewSDL emits a single CREATE OR REPLACE VIEW statement. LoadSDL accepts
// CREATE OR REPLACE VIEW and is order-tolerant, so no temporary-view scaffolding is
// needed. dbName is the database being dumped; its own-database qualifier is stripped
// from the body so the emitted form matches the user's unqualified spelling (see
// stripViewBodyDatabaseQualifier — required for 5.7 derived-table view idempotence,
// entry TestSDLStressViewDerivedTable57). The stored one-line body is then
// pretty-printed by formatViewBodySDL — a whitespace-only, deterministic rewrite, so
// the omni no-op invariant and the dump-cycle byte stability both hold.
func writeViewSDL(buf *strings.Builder, dbName string, view *storepb.ViewMetadata) error {
	body := formatViewBodySDL(stripViewBodyDatabaseQualifier(view.Definition, dbName))
	if body == "" {
		// Defensive: an empty definition keeps the historical inline framing.
		if _, err := fmt.Fprintf(buf, "CREATE OR REPLACE VIEW `%s` AS ;\n\n", view.Name); err != nil {
			return err
		}
		return nil
	}
	if _, err := fmt.Fprintf(buf, "CREATE OR REPLACE VIEW `%s` AS\n%s;\n\n", view.Name, body); err != nil {
		return err
	}
	return nil
}

// stripViewBodyDatabaseQualifier removes the dumped database's own-database qualifier
// (the leading “ `<dbName>`. “ prefix on a table/column reference) from a view body.
//
// MySQL 5.7's SHOW CREATE VIEW / information_schema.VIEWS.VIEW_DEFINITION fully qualifies
// references — including the inner refs of a FROM-clause derived table — with the schema
// name (e.g. “ `mydb`.`assignment` “), whereas 8.0 stores them unqualified and the user
// authors them unqualified. The omni SDL diff loads both inputs under the synthetic
// database bbcatalog and its canonicalViewBody only folds the loaded own-database prefix
// (bbcatalog), so a body still carrying the real synced DB name phantom-diffs against the
// unqualified user body. Stripping the qualifier at dump time yields the db-neutral form
// both 5.7 and 8.0 share.
//
// Only the current database's own qualifier is removed (genuine cross-database references,
// which MySQL views in this single-database scope effectively never carry, are left
// intact). The body is walked with scanViewBodyToken — the same quote/identifier-aware
// scanner formatViewBodySDL uses — so ONE scanner owns the literal rules: a
// “ `<dbName>`. “ sequence inside a '…' or "…" string literal (or inside a backtick
// identifier such as an alias) is NOT a qualifier and is left untouched.
//
// A qualifier is recognized only where one can grammatically start: a backtick
// identifier spelling dbName, immediately followed by "." and another backtick
// identifier, and NOT itself preceded by "." (then it is the table/column part of an
// enclosing dotted reference, e.g. the table part of `otherdb`.`<dbName>`.`c`). After a
// strip, the following identifier is copied verbatim BEFORE re-arming, so a table named
// like the database (`<dbName>`.`<dbName>`.`col`, the 5.7 three-part form) loses only
// the database part.
func stripViewBodyDatabaseQualifier(body, dbName string) string {
	if dbName == "" || body == "" {
		return body
	}
	quoted := "`" + strings.ReplaceAll(dbName, "`", "``") + "`"
	if !strings.Contains(body, quoted+".") {
		return body
	}

	var out strings.Builder
	out.Grow(len(body))
	// prevTok is the previous non-whitespace token; "." marks the current identifier as
	// a continuation part of a dotted reference, never a qualifier start.
	prevTok := ""
	for i := 0; i < len(body); {
		if isViewBodySpaceByte(body[i]) {
			out.WriteByte(body[i])
			i++
			continue
		}
		end := scanViewBodyToken(body, i)
		tok := body[i:end]
		if tok == quoted && prevTok != "." && end < len(body) && body[end] == '.' && end+1 < len(body) && body[end+1] == '`' {
			// Own-database qualifier: drop it and the dot, then copy the following
			// identifier verbatim so it is not re-tested as a qualifier.
			identEnd := scanViewBodyToken(body, end+1)
			ident := body[end+1 : identEnd]
			out.WriteString(ident)
			prevTok = ident
			i = identEnd
			continue
		}
		out.WriteString(tok)
		prevTok = tok
		i = end
	}
	return out.String()
}

// writeRoutineSDL emits a stored function or procedure. The synced Definition is
// already a complete, canonical CREATE FUNCTION/PROCEDURE statement (the MySQL sync
// rewrites SHOW CREATE to drop the DEFINER and the RETURNS charset attribute), so the
// only framing the SDL form adds is a terminating ";". The body is emitted verbatim:
// omni's loader splitter is BEGIN…END aware, so internal ";" inside a compound body
// do not need DELIMITER wrapping, and omni's routine differ compares the body byte for
// byte after trimming — reformatting would phantom-diff. A leading DEFINER, if one
// somehow survives, is stripped to match omni's DEFINER-agnostic routine identity.
func writeRoutineSDL(buf *strings.Builder, definition string) error {
	def := strings.TrimSpace(stripLeadingDefiner(definition))
	if def == "" {
		return nil
	}
	if _, err := fmt.Fprintf(buf, "%s;\n\n", def); err != nil {
		return err
	}
	return nil
}

// writeTriggerSDL constructs a canonical CREATE TRIGGER from the trigger's metadata.
// MySQL stores only the action statement (TriggerMetadata.Body) plus the timing,
// event, and owning table, so the CREATE wrapper is rebuilt here exactly as omni's
// loader expects: `CREATE TRIGGER <name> <timing> <event> ON <table> FOR EACH ROW
// <body>`. DEFINER is omitted (omni's trigger differ ignores it) and the body is
// emitted verbatim (compared byte for byte after trimming).
func writeTriggerSDL(buf *strings.Builder, tableName string, trigger *storepb.TriggerMetadata) error {
	if _, err := fmt.Fprintf(buf, "CREATE TRIGGER `%s` %s %s ON `%s` FOR EACH ROW\n%s;\n\n",
		trigger.Name, trigger.Timing, trigger.Event, tableName, strings.TrimSpace(trigger.Body)); err != nil {
		return err
	}
	return nil
}

// writeEventSDL emits a scheduled event. The synced Definition is the full
// SHOW CREATE EVENT text, which (unlike functions/procedures) still carries the
// DEFINER clause, so it is stripped here to match omni's DEFINER-agnostic event
// identity. The schedule's auto-injected STARTS '<create-time>' is left intact: omni's
// event differ normalizes STARTS out of the canonical schedule key, so it does not
// perturb the no-op.
func writeEventSDL(buf *strings.Builder, event *storepb.EventMetadata) error {
	def := strings.TrimSpace(stripLeadingDefiner(event.Definition))
	if def == "" {
		return nil
	}
	if _, err := fmt.Fprintf(buf, "%s;\n\n", def); err != nil {
		return err
	}
	return nil
}

// stripLeadingDefiner removes a leading `CREATE DEFINER=<account> ` clause from a
// CREATE statement, collapsing it back to a bare `CREATE <OBJECT> …`. The account is
// parsed properly — `user`@`host` with each part backtick, single- or double-quoted
// (quoted parts may contain spaces, '@', and doubled/escaped quotes), an unquoted
// user@host, or CURRENT_USER[()] — so a legal quoted account with a space
// (DEFINER=`my user`@`%`) does not get cut at its first space. Statements without a
// DEFINER, or with an account this scan cannot parse, are returned unchanged (omni
// tolerates a DEFINER and never diffs on it, so keeping one is safe while corrupting
// the statement is not). Stripping keeps the dump canonical and matches the
// routine/trigger emission, which carry no DEFINER at all.
func stripLeadingDefiner(definition string) string {
	const createKW = "CREATE "
	trimmed := strings.TrimLeft(definition, " \t\r\n")
	if !strings.HasPrefix(strings.ToUpper(trimmed), createKW) {
		return definition
	}
	rest := strings.TrimLeft(trimmed[len(createKW):], " \t")
	const definerKW = "DEFINER"
	if !strings.HasPrefix(strings.ToUpper(rest), definerKW) {
		return definition
	}
	pos := len(definerKW)
	pos = skipSpacesAndTabs(rest, pos)
	if pos >= len(rest) || rest[pos] != '=' {
		return definition
	}
	pos = skipSpacesAndTabs(rest, pos+1)
	end, ok := scanDefinerAccount(rest, pos)
	if !ok {
		return definition
	}
	// The clause must be followed by whitespace and the object keyword.
	afterAccount := skipSpacesAndTabs(rest, end)
	if afterAccount == end || afterAccount >= len(rest) {
		return definition
	}
	return createKW + rest[afterAccount:]
}

// skipSpacesAndTabs returns the first offset >= pos in s that is not a space or tab.
func skipSpacesAndTabs(s string, pos int) int {
	for pos < len(s) && (s[pos] == ' ' || s[pos] == '\t') {
		pos++
	}
	return pos
}

// scanDefinerAccount scans a MySQL account name starting at pos: CURRENT_USER[()] or
// <part>[@<part>] where each part is backtick/single/double quoted (doubled-delimiter
// and, for '/'"', backslash escapes) or an unquoted run. Returns the offset just past
// the account and whether the scan succeeded.
func scanDefinerAccount(s string, pos int) (int, bool) {
	const currentUser = "CURRENT_USER"
	if len(s)-pos >= len(currentUser) && strings.EqualFold(s[pos:pos+len(currentUser)], currentUser) {
		end := pos + len(currentUser)
		if strings.HasPrefix(s[end:], "()") {
			end += 2
		}
		return end, true
	}
	end, ok := scanAccountPart(s, pos)
	if !ok {
		return 0, false
	}
	if end < len(s) && s[end] == '@' {
		hostEnd, ok := scanAccountPart(s, end+1)
		if !ok {
			return 0, false
		}
		end = hostEnd
	}
	return end, true
}

// scanAccountPart scans one user or host part of an account name starting at pos.
func scanAccountPart(s string, pos int) (int, bool) {
	if pos >= len(s) {
		return 0, false
	}
	switch q := s[pos]; q {
	case '`', '\'', '"':
		for i := pos + 1; i < len(s); {
			switch {
			case q != '`' && s[i] == '\\' && i+1 < len(s):
				// Backslash escapes apply inside '…' and "…" (not backticks).
				i += 2
			case s[i] == q:
				if i+1 < len(s) && s[i+1] == q {
					// Doubled delimiter stays inside the part.
					i += 2
					continue
				}
				return i + 1, true
			default:
				i++
			}
		}
		return 0, false // Unterminated quote.
	default:
		// Unquoted part: a run of bytes up to the '@' separator or whitespace.
		i := pos
		for i < len(s) && s[i] != '@' && s[i] != ' ' && s[i] != '\t' && s[i] != '\r' && s[i] != '\n' {
			i++
		}
		if i == pos {
			return 0, false
		}
		return i, true
	}
}
