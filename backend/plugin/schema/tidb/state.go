package tidb

import (
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	tidbparser "github.com/bytebase/tidb-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db/tidb"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type databaseState struct {
	name    string
	schemas map[string]*schemaState
}

func newDatabaseState() *databaseState {
	return &databaseState{
		schemas: make(map[string]*schemaState),
	}
}

func convertToDatabaseState(database *storepb.DatabaseSchemaMetadata) *databaseState {
	state := newDatabaseState()
	state.name = database.Name
	for _, schema := range database.Schemas {
		state.schemas[schema.Name] = convertToSchemaState(schema)
	}
	return state
}

type schemaState struct {
	name   string
	tables map[string]*tableState
	views  map[string]*viewState
}

func newSchemaState() *schemaState {
	return &schemaState{
		tables: make(map[string]*tableState),
		views:  make(map[string]*viewState),
	}
}

func convertToSchemaState(schema *storepb.SchemaMetadata) *schemaState {
	state := newSchemaState()
	state.name = schema.Name
	for i, table := range schema.Tables {
		state.tables[table.Name] = convertToTableState(i, table)
	}
	for i, view := range schema.Views {
		state.views[view.Name] = convertToViewState(i, view)
	}
	return state
}

type tableState struct {
	id          int
	name        string
	columns     map[string]*columnState
	indexes     map[string]*indexState
	foreignKeys map[string]*foreignKeyState
	comment     string
	// Engine and collation are only supported in ParseToMetadata.
	engine    string
	collation string
	partition *partitionState
}

func (t *tableState) toString(buf *strings.Builder) error {
	if _, err := fmt.Fprintf(buf, "CREATE TABLE `%s` (\n  ", t.name); err != nil {
		return err
	}
	columns := []*columnState{}
	for _, column := range t.columns {
		columns = append(columns, column)
	}
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].id < columns[j].id
	})
	for i, column := range columns {
		if i > 0 {
			if _, err := buf.WriteString(",\n  "); err != nil {
				return err
			}
		}
		if err := column.toString(buf); err != nil {
			return err
		}
	}

	indexes := []*indexState{}
	for _, index := range t.indexes {
		indexes = append(indexes, index)
	}
	sort.Slice(indexes, func(i, j int) bool {
		if indexes[i].primary {
			return true
		}
		if indexes[j].primary {
			return false
		}
		return indexes[i].name < indexes[j].name
	})

	for i, index := range indexes {
		if i+len(columns) > 0 {
			if _, err := buf.WriteString(",\n  "); err != nil {
				return err
			}
		}
		if err := index.toString(buf); err != nil {
			return err
		}
	}

	foreignKeys := []*foreignKeyState{}
	for _, fk := range t.foreignKeys {
		foreignKeys = append(foreignKeys, fk)
	}
	sort.Slice(foreignKeys, func(i, j int) bool {
		return foreignKeys[i].name < foreignKeys[j].name
	})

	for i, fk := range foreignKeys {
		if i+len(columns)+len(indexes) > 0 {
			if _, err := buf.WriteString(",\n  "); err != nil {
				return err
			}
		}
		if err := fk.toString(buf); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString("\n)"); err != nil {
		return err
	}

	if t.engine != "" {
		if _, err := fmt.Fprintf(buf, " ENGINE=%s", t.engine); err != nil {
			return err
		}
	}

	if t.collation != "" {
		if _, err := fmt.Fprintf(buf, " COLLATE=%s", t.collation); err != nil {
			return err
		}
	}

	if t.comment != "" {
		if _, err := fmt.Fprintf(buf, " COMMENT '%s'", t.comment); err != nil {
			return err
		}
	}

	if t.partition != nil {
		if err := t.partition.toString(buf, nil); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString(";\n"); err != nil {
		return err
	}
	return nil
}

func newTableState(id int, name string) *tableState {
	return &tableState{
		id:          id,
		name:        name,
		columns:     make(map[string]*columnState),
		indexes:     make(map[string]*indexState),
		foreignKeys: make(map[string]*foreignKeyState),
	}
}

func convertToTableState(id int, table *storepb.TableMetadata) *tableState {
	state := newTableState(id, table.Name)
	state.engine = table.Engine
	state.collation = table.Collation
	state.comment = table.Comment
	for i, column := range table.Columns {
		state.columns[column.Name] = convertToColumnState(i, column)
	}
	for i, index := range table.Indexes {
		state.indexes[index.Name] = convertToIndexState(i, index)
	}
	for i, fk := range table.ForeignKeys {
		state.foreignKeys[fk.Name] = convertToForeignKeyState(i, fk)
	}
	state.partition = convertToPartitionState(table.Partitions)
	return state
}

type foreignKeyState struct {
	id                int
	name              string
	columns           []string
	referencedTable   string
	referencedColumns []string
}

func convertToForeignKeyState(id int, foreignKey *storepb.ForeignKeyMetadata) *foreignKeyState {
	return &foreignKeyState{
		id:                id,
		name:              foreignKey.Name,
		columns:           foreignKey.Columns,
		referencedTable:   foreignKey.ReferencedTable,
		referencedColumns: foreignKey.ReferencedColumns,
	}
}

func (f *foreignKeyState) toString(buf *strings.Builder) error {
	if _, err := buf.WriteString("CONSTRAINT `"); err != nil {
		return err
	}
	if _, err := buf.WriteString(f.name); err != nil {
		return err
	}
	if _, err := buf.WriteString("` FOREIGN KEY ("); err != nil {
		return err
	}
	for i, column := range f.columns {
		if i > 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(") REFERENCES `"); err != nil {
		return err
	}
	if _, err := buf.WriteString(f.referencedTable); err != nil {
		return err
	}
	if _, err := buf.WriteString("` ("); err != nil {
		return err
	}
	for i, column := range f.referencedColumns {
		if i > 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(")"); err != nil {
		return err
	}
	return nil
}

type indexState struct {
	id      int
	name    string
	keys    []string
	lengths []int64
	primary bool
	unique  bool
	tp      string
}

func convertToIndexState(id int, index *storepb.IndexMetadata) *indexState {
	return &indexState{
		id:      id,
		name:    index.Name,
		keys:    index.Expressions,
		lengths: index.KeyLength,
		primary: index.Primary,
		unique:  index.Unique,
		tp:      index.Type,
	}
}

func (i *indexState) toString(buf *strings.Builder) error {
	if i.primary {
		if _, err := buf.WriteString("PRIMARY KEY ("); err != nil {
			return err
		}
		for i, key := range i.keys {
			if i > 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(buf, "`%s`", key); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	} else {
		if strings.ToUpper(i.tp) == "FULLTEXT" {
			if _, err := buf.WriteString("FULLTEXT KEY "); err != nil {
				return err
			}
		} else if strings.ToUpper(i.tp) == "SPATIAL" {
			if _, err := buf.WriteString("SPATIAL KEY "); err != nil {
				return err
			}
		} else if i.unique {
			if _, err := buf.WriteString("UNIQUE KEY "); err != nil {
				return err
			}
		} else {
			if _, err := buf.WriteString("KEY "); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprintf(buf, "`%s` (", i.name); err != nil {
			return err
		}
		for j, key := range i.keys {
			if j > 0 {
				if _, err := buf.WriteString(","); err != nil {
					return err
				}
			}
			if len(key) > 2 && key[0] == '(' && key[len(key)-1] == ')' {
				// Expressions are surrounded by parentheses.
				if _, err := buf.WriteString(key); err != nil {
					return err
				}
			} else {
				columnText := fmt.Sprintf("`%s`", key)
				if len(i.lengths) > j && i.lengths[j] > 0 {
					columnText = fmt.Sprintf("`%s`(%d)", key, i.lengths[j])
				}
				if _, err := buf.WriteString(columnText); err != nil {
					return err
				}
			}
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	}
	return nil
}

type partitionState struct {
	info       partitionInfo
	subInfo    *partitionInfo
	partitions map[string]*partitionDefinition
}

type partitionInfo struct {
	tp         storepb.TablePartitionMetadata_Type
	useDefault int
	expr       string
}

type partitionDefinition struct {
	id            int
	name          string
	value         string
	subpartitions map[string]*partitionDefinition
}

func convertToPartitionState(partitions []*storepb.TablePartitionMetadata) *partitionState {
	if len(partitions) == 0 {
		return nil
	}
	state := &partitionState{
		partitions: make(map[string]*partitionDefinition),
	}
	for i, partition := range partitions {
		if i == 0 {
			state.info.tp = partition.Type
			state.info.expr = partition.Expression
			if partition.UseDefault != "" {
				var err error
				state.info.useDefault, err = strconv.Atoi(partition.UseDefault)
				if err != nil {
					slog.Warn(err.Error())
				}
			}
		}
		partitionDef := &partitionDefinition{
			id:            i,
			name:          partition.Name,
			value:         partition.Value,
			subpartitions: make(map[string]*partitionDefinition),
		}

		for j, subPartition := range partition.Subpartitions {
			if j == 0 {
				state.subInfo = &partitionInfo{
					tp:   subPartition.Type,
					expr: subPartition.Expression,
				}
				if subPartition.UseDefault != "" {
					var err error
					state.subInfo.useDefault, err = strconv.Atoi(subPartition.UseDefault)
					if err != nil {
						slog.Warn(err.Error())
					}
				}
			}
			partitionDef.subpartitions[subPartition.Name] = &partitionDefinition{
				id:    j,
				name:  subPartition.Name,
				value: subPartition.Value,
			}
		}
		state.partitions[partition.Name] = partitionDef
	}
	return state
}

// toString() writes the partition state as SHOW CREATE TABLE syntax to buf, referencing MySQL source code:
// https://sourcegraph.com/github.com/mysql/mysql-server@824e2b4064053f7daf17d7f3f84b7a3ed92e5fb4/-/blob/sql/sql_show.cc?L2528-2550
// partitionClauseCtx is use to minimize the difference between the original one and the output, it is safe to pass nil.
func (p *partitionState) toString(buf io.StringWriter, partitionClauseCtx tidbparser.IPartitionClauseContext) error {
	// Write version specific comment.
	vsc := p.getVersionSpecificComment(partitionClauseCtx)
	curComment := vsc
	if _, err := buf.WriteString(vsc); err != nil {
		return err
	}
	if _, err := buf.WriteString(" PARTITION BY "); err != nil {
		return err
	}

	switch p.info.tp {
	case storepb.TablePartitionMetadata_RANGE, storepb.TablePartitionMetadata_RANGE_COLUMNS:
		if _, err := buf.WriteString("RANGE "); err != nil {
			return err
		}
		if p.info.tp == storepb.TablePartitionMetadata_RANGE {
			fields := splitPartitionExprIntoFields(p.info.expr)
			for i, field := range fields {
				if !strings.Contains(field, "(") {
					fields[i] = fmt.Sprintf("`%s`", field)
				}
			}
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		} else {
			// I think MySQL need to write "COLUMNS " instead of " COLUMNS" here...
			if _, err := buf.WriteString(" COLUMNS"); err != nil {
				return err
			}
			fields := splitPartitionExprIntoFields(p.info.expr)
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		}
	case storepb.TablePartitionMetadata_LIST, storepb.TablePartitionMetadata_LIST_COLUMNS:
		if _, err := buf.WriteString("LIST "); err != nil {
			return err
		}
		if p.info.tp == storepb.TablePartitionMetadata_LIST {
			fields := splitPartitionExprIntoFields(p.info.expr)
			for i, field := range fields {
				if !strings.Contains(field, "(") {
					fields[i] = fmt.Sprintf("`%s`", field)
				}
			}
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		} else {
			// I think MySQL need to write "COLUMNS " instead of " COLUMNS" here...
			if _, err := buf.WriteString(" COLUMNS"); err != nil {
				return err
			}
			fields := splitPartitionExprIntoFields(p.info.expr)
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		}
	case storepb.TablePartitionMetadata_HASH, storepb.TablePartitionMetadata_KEY, storepb.TablePartitionMetadata_LINEAR_HASH, storepb.TablePartitionMetadata_LINEAR_KEY:
		if p.info.tp == storepb.TablePartitionMetadata_LINEAR_HASH || p.info.tp == storepb.TablePartitionMetadata_LINEAR_KEY {
			if _, err := buf.WriteString("LINEAR "); err != nil {
				return err
			}
		}
		if p.info.tp == storepb.TablePartitionMetadata_KEY || p.info.tp == storepb.TablePartitionMetadata_LINEAR_KEY {
			if _, err := buf.WriteString("KEY "); err != nil {
				return err
			}
			// NOTE: MySQL supports an ALGORITHM option with [SUB]PARTITION BY [LINEAR KEY]. ALGORITHM=1 causes the server to use the same key-hashing function as MYSQL 5.1, and ALGORITHM=1 is the only possible output in
			// the following code. Sadly, I do not know how to get the key_algorithm from the INFORMATION_SCHEMA, AND 5.1 IS TOO LEGACY TO SUPPORT! So use the original one.
			/*
			   current_comment_start is given when called from SHOW CREATE TABLE,
			   Then only add ALGORITHM = 1, not the default 2 or non-set 0!
			   For .frm current_comment_start is NULL, then add ALGORITHM if != 0.
			*/
			// if (part_info->key_algorithm ==
			// 	enum_key_algorithm::KEY_ALGORITHM_51 ||  // SHOW
			// (!current_comment_start &&                   // .frm
			//  (part_info->key_algorithm != enum_key_algorithm::KEY_ALGORITHM_NONE))) {
			// 	/* If we already are within a comment, end that comment first. */
			// 	if (current_comment_start) err += add_string(fptr, "*/ ");
			// 	err += add_string(fptr, "/*!50611 ");
			// 	err += add_part_key_word(fptr, partition_keywords[PKW_ALGORITHM].str);
			// 	err += add_equal(fptr);
			// 	err += add_space(fptr);
			// 	err += add_int(fptr, static_cast<longlong>(part_info->key_algorithm));
			// 	err += add_space(fptr);
			// 	err += add_string(fptr, "*/ ");
			// 	if (current_comment_start) {
			// 		/* Skip new line. */
			// 		if (current_comment_start[0] == '\n') current_comment_start++;
			// 		err += add_string(fptr, current_comment_start);
			// 		err += add_space(fptr);
			// 	}
			// }
			if partitionClauseCtx != nil && partitionClauseCtx.PartitionTypeDef() != nil {
				v := partitionClauseCtx.PartitionTypeDef()
				partitionDefKeyCtx, ok := v.(*tidbparser.PartitionDefKeyContext)
				if ok && partitionDefKeyCtx.PartitionKeyAlgorithm() != nil {
					numText := partitionDefKeyCtx.PartitionKeyAlgorithm().Real_ulong_number().GetText()
					num, err := strconv.Atoi(numText)
					if err != nil {
						slog.Warn(err.Error())
					} else if num == 1 || (num == 0 && len(curComment) == 0) {
						if _, err := buf.WriteString(fmt.Sprintf("*/ /*!50611 ALGORITHM = %d */ ", num)); err != nil {
							return err
						}
						if len(curComment) > 0 {
							s := curComment
							if curComment[0] == '\n' {
								s = curComment[1:]
							}
							if _, err := buf.WriteString(fmt.Sprintf("%s ", s)); err != nil {
								return err
							}
						}
					}
				}
			}
			// HACK(zp): Write the part field list. In the MySQL source code, it calls append_identifier(), which considers the quote character. We should figure out the logic of it later.
			// Currently, I just found that if the expr contains more than one field, it would not be quoted by '`'.
			// KEY and LINEAR KEY can take the field list.
			// While MySQL calls append_field_list() to write the field list, it unmasks the OPTION_QUOTE_SHOW_CREATE flag,
			// for us, we do the best effort to split the expr by ',' and trim the leading and trailing '`', and write it to the buffer after joining them with ','.
			fields := splitPartitionExprIntoFields(p.info.expr)
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		} else {
			if _, err := buf.WriteString("HASH "); err != nil {
				return err
			}
			fields := splitPartitionExprIntoFields(p.info.expr)
			for i, field := range fields {
				if !strings.Contains(field, "(") {
					fields[i] = fmt.Sprintf("`%s`", field)
				}
			}
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		}
	default:
		return errors.Errorf("unsupported partition type: %v", p.info.tp)
	}

	// NOTE: MySQL writes the default partitions in the following code, which means that the server
	// takes the responsibility to generate the partitions. Sadly, we cannot get whether the user
	// use this or not in the metadata. So we skip it.
	/*
		if ((!part_info->use_default_num_partitions) &&
		    part_info->use_default_partitions) {
		    	err += add_string(fptr, "\n");
		    	err += add_string(fptr, "PARTITIONS ");
		    	err += add_int(fptr, part_info->num_parts);
		}
	*/
	if p.info.useDefault != 0 {
		if _, err := buf.WriteString(fmt.Sprintf("\nPARTITIONS %d", p.info.useDefault)); err != nil {
			return err
		}
	}

	isSubpartitioned := p.subInfo != nil && p.subInfo.tp != storepb.TablePartitionMetadata_TYPE_UNSPECIFIED
	if isSubpartitioned {
		if _, err := buf.WriteString("\nSUBPARTITION BY "); err != nil {
			return err
		}
	}
	// Subpartition must be hash or key.
	if isSubpartitioned {
		switch p.subInfo.tp {
		case storepb.TablePartitionMetadata_HASH, storepb.TablePartitionMetadata_LINEAR_HASH:
			if p.subInfo.tp == storepb.TablePartitionMetadata_LINEAR_HASH {
				if _, err := buf.WriteString("LINEAR "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString("HASH "); err != nil {
				return err
			}
			fields := splitPartitionExprIntoFields(p.subInfo.expr)
			for i, field := range fields {
				if !strings.Contains(field, "(") {
					fields[i] = fmt.Sprintf("`%s`", field)
				}
			}
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		case storepb.TablePartitionMetadata_KEY, storepb.TablePartitionMetadata_LINEAR_KEY:
			if p.subInfo.tp == storepb.TablePartitionMetadata_LINEAR_KEY {
				if _, err := buf.WriteString("LINEAR "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString("KEY "); err != nil {
				return err
			}
			if partitionClauseCtx != nil && partitionClauseCtx.SubPartitions() != nil {
				if v := partitionClauseCtx.SubPartitions().PartitionKeyAlgorithm(); v != nil {
					numText := v.Real_ulong_number().GetText()
					num, err := strconv.Atoi(numText)
					if err != nil {
						slog.Warn(err.Error())
					} else if num == 1 || (num == 0 && len(curComment) == 0) {
						if _, err := buf.WriteString(fmt.Sprintf("*/ /*!50611 ALGORITHM = %d */ ", num)); err != nil {
							return err
						}
						if len(curComment) > 0 {
							s := curComment
							if curComment[0] == '\n' {
								s = curComment[1:]
							}
							if _, err := buf.WriteString(fmt.Sprintf("%s ", s)); err != nil {
								return err
							}
						}
					}
				}
			}
			fields := splitPartitionExprIntoFields(p.subInfo.expr)
			if _, err := buf.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ","))); err != nil {
				return err
			}
		default:
			return errors.Errorf("invalid subpartition type: %v", p.subInfo.tp)
		}
	}

	// TODO(zp): MySQL writes the default subpartitions in the following code, which means that the server
	// takes the responsibility to generate the subpartitions. Sadly, we cannot get whether the user
	// use this or not in the metadata. So we skip it.
	/*
		if ((!part_info->use_default_num_subpartitions) &&
			part_info->use_default_subpartitions) {
				err += add_string(fptr, "\n");
				err += add_string(fptr, "SUBPARTITIONS ");
				err += add_int(fptr, part_info->num_subparts);
		}
	*/
	if isSubpartitioned && p.subInfo.useDefault != 0 {
		if _, err := buf.WriteString(fmt.Sprintf("\nSUBPARTITIONS %d", p.subInfo.useDefault)); err != nil {
			return err
		}
	}

	if p.info.useDefault == 0 {
		// Write the partition list.
		if len(p.partitions) == 0 {
			return errors.New("empty partition list")
		}
		sortedPartitions := make([]*partitionDefinition, 0, len(p.partitions))
		for _, partition := range p.partitions {
			sortedPartitions = append(sortedPartitions, partition)
		}
		sort.Slice(sortedPartitions, func(i, j int) bool {
			return sortedPartitions[i].id < sortedPartitions[j].id
		})
		if _, err := buf.WriteString("\n("); err != nil {
			return err
		}
		preposition, err := getPrepositionByType(p.info.tp)
		if err != nil {
			return err
		}
		for i, partition := range sortedPartitions {
			if i != 0 {
				if _, err := buf.WriteString(",\n "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString(fmt.Sprintf("PARTITION %s", partition.name)); err != nil {
				return err
			}
			if preposition != "" {
				if partition.value != "MAXVALUE" {
					if _, err := buf.WriteString(fmt.Sprintf(" VALUES %s (%s)", preposition, partition.value)); err != nil {
						return err
					}
				} else {
					if _, err := buf.WriteString(fmt.Sprintf(" VALUES %s %s", preposition, partition.value)); err != nil {
						return err
					}
				}
			}

			if isSubpartitioned && p.subInfo.useDefault == 0 {
				if len(partition.subpartitions) == 0 {
					return errors.New("empty subpartition list")
				}
				if _, err := buf.WriteString("\n ("); err != nil {
					return err
				}
				sortedSubpartitions := make([]*partitionDefinition, 0, len(partition.subpartitions))
				for _, subPartition := range partition.subpartitions {
					sortedSubpartitions = append(sortedSubpartitions, subPartition)
				}
				sort.Slice(sortedSubpartitions, func(i, j int) bool {
					return sortedSubpartitions[i].id < sortedSubpartitions[j].id
				})
				for j, subPartition := range sortedSubpartitions {
					if _, err := buf.WriteString(fmt.Sprintf("SUBPARTITION %s", subPartition.name)); err != nil {
						return err
					}
					if err := writePartitionOptions(buf); err != nil {
						return err
					}
					if j == len(sortedSubpartitions)-1 {
						if _, err := buf.WriteString(")"); err != nil {
							return err
						}
					} else {
						if _, err := buf.WriteString(",\n  "); err != nil {
							return err
						}
					}
				}
			} else {
				if err := writePartitionOptions(buf); err != nil {
					return err
				}
			}

			if i == len(sortedPartitions)-1 {
				if _, err := buf.WriteString(")"); err != nil {
					return err
				}
			}
		}
	}

	if _, err := buf.WriteString(" */"); err != nil {
		return err
	}

	return nil
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

// getVersionSpecificComment is the go code equivalent of MySQL void partition_info::set_show_version_string(String *packet).
// partitionClauseCtx is use to minimize the difference between the original one and the output, it is safe to pass nil.
func (p *partitionState) getVersionSpecificComment(partitionClauseCtx tidbparser.IPartitionClauseContext) string {
	if p.info.tp == storepb.TablePartitionMetadata_RANGE_COLUMNS || p.info.tp == storepb.TablePartitionMetadata_LIST_COLUMNS {
		// MySQL introduce columns partitioning in 5.5+.
		return "\n/*!50500"
	} else if partitionClauseCtx != nil {
		/*
				if (part_expr)
			      part_expr->walk(&Item::intro_version, enum_walk::POSTFIX,
			                      (uchar *)&version);
			    if (subpart_expr)
			      subpart_expr->walk(&Item::intro_version, enum_walk::POSTFIX,
			                         (uchar *)&version);
		*/
		tokenStream := partitionClauseCtx.GetParser().GetTokenStream()
		startPos := partitionClauseCtx.GetStart().GetTokenIndex()
		if tokenStream != nil {
			if startPos-2 > 0 && tokenStream.Size() > startPos-2 {
				regexp := regexp.MustCompile(`\/\*![0-9]+`)
				for i := 0; i < 2; i++ {
					if tokenStream.Get(startPos-i-1).GetChannel() == antlr.TokenHiddenChannel {
						if regexp.MatchString(tokenStream.Get(startPos - i - 1).GetText()) {
							return fmt.Sprintf("\n%s", tokenStream.Get(startPos-i-1).GetText())
						}
					}
				}
			}
		}
	}
	// NOTE: Users can use function in partition expr or subpartition expr, and the intro version of function should be the infimum of the version.
	// But sadly, it's a huge work for us to copy the intro version for each function in MySQL. So we use the original one.
	return "\n/*!50100"
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

// splitPartitioNExprIntoFields splits the partition expression by ',', and trims the leading and trailing '`' for each element.
func splitPartitionExprIntoFields(expr string) []string {
	// We do not support the expression contains parentheses, so we can split the expression by ','.
	ss := strings.Split(expr, ",")
	for i, s := range ss {
		if strings.HasPrefix(s, "`") && strings.HasSuffix(s, "`") {
			ss[i] = s[1 : len(s)-1]
		}
	}
	return ss
}

type defaultValue interface {
	toString() string
}

type defaultValueNull struct {
}

func (*defaultValueNull) toString() string {
	return "NULL"
}

type defaultValueString struct {
	value string
}

func (d *defaultValueString) toString() string {
	return fmt.Sprintf("'%s'", d.value)
}

type defaultValueExpression struct {
	value string
}

func (d *defaultValueExpression) toString() string {
	return d.value
}

type columnState struct {
	id           int
	name         string
	tp           string
	defaultValue defaultValue
	onUpdate     string
	comment      string
	nullable     bool
}

func (c *columnState) toString(buf *strings.Builder) error {
	columnCanonicalType := tidb.GetColumnTypeCanonicalSynonym(strings.ToLower(c.tp))
	if _, err := fmt.Fprintf(buf, "`%s` %s", c.name, columnCanonicalType); err != nil {
		return err
	}
	if !c.nullable {
		if _, err := buf.WriteString(" NOT NULL"); err != nil {
			return err
		}
	}
	if c.defaultValue != nil {
		_, isDefaultNull := c.defaultValue.(*defaultValueNull)
		dontWriteDefaultNull := isDefaultNull && c.nullable && expressionDefaultOnlyTypes[strings.ToUpper(c.tp)]
		// Some types do not default to NULL, but support default expressions.
		if !dontWriteDefaultNull {
			// todo(zp): refactor column attribute.
			if strings.EqualFold(c.defaultValue.toString(), autoIncrementSymbol) {
				if _, err := buf.WriteString(" " + autoIncrementSymbol); err != nil {
					return err
				}
			} else if strings.Contains(strings.ToUpper(c.defaultValue.toString()), autoRandSymbol) {
				if _, err := fmt.Fprintf(buf, " /*T![auto_rand] %s */", c.defaultValue.toString()); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(buf, " DEFAULT %s", c.defaultValue.toString()); err != nil {
					return err
				}
			}
		}
	}
	if c.onUpdate != "" {
		if _, err := fmt.Fprintf(buf, " ON UPDATE %s", c.onUpdate); err != nil {
			return err
		}
	}
	if c.comment != "" {
		if _, err := fmt.Fprintf(buf, " COMMENT '%s'", c.comment); err != nil {
			return err
		}
	}
	return nil
}

func convertToColumnState(id int, column *storepb.ColumnMetadata) *columnState {
	result := &columnState{
		id:       id,
		name:     column.Name,
		tp:       column.Type,
		nullable: column.Nullable,
		onUpdate: normalizeOnUpdate(column.OnUpdate),
		comment:  column.Comment,
	}
	if column.GetDefaultValue() != nil {
		switch value := column.GetDefaultValue().(type) {
		case *storepb.ColumnMetadata_DefaultNull:
			result.defaultValue = &defaultValueNull{}
		case *storepb.ColumnMetadata_Default:
			if value.Default == nil {
				result.defaultValue = &defaultValueNull{}
			} else {
				result.defaultValue = &defaultValueString{value: value.Default.GetValue()}
			}
		case *storepb.ColumnMetadata_DefaultExpression:
			result.defaultValue = &defaultValueExpression{value: value.DefaultExpression}
		}
	}
	return result
}

func normalizeOnUpdate(s string) string {
	if s == "" {
		return ""
	}

	lowerS := strings.ToLower(s)
	re := regexp.MustCompile(`(current_timestamp|now|localtime|localtimestamp)(?:\((\d+)\))?`)
	match := re.FindStringSubmatch(lowerS)
	if len(match) > 0 {
		if len(match) >= 3 && match[2] != "" {
			// has precision
			return fmt.Sprintf("CURRENT_TIMESTAMP(%s)", match[2])
		}
		// no precision
		return "CURRENT_TIMESTAMP"
	}
	// not a current_timestamp family function
	return s
}

type viewState struct {
	id         int
	name       string
	definition string
}

func convertToViewState(id int, view *storepb.ViewMetadata) *viewState {
	return &viewState{
		id:         id,
		name:       view.Name,
		definition: view.Definition,
	}
}

func (v *viewState) toString(buf io.StringWriter) error {
	stmt := fmt.Sprintf("CREATE OR REPLACE VIEW `%s` AS %s", v.name, v.definition)
	if !strings.HasSuffix(stmt, ";") {
		stmt += ";"
	}
	stmt += "\n"
	if _, err := buf.WriteString(stmt); err != nil {
		return err
	}
	return nil
}
