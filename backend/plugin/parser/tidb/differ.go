package tidb

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	tidbparser "github.com/bytebase/tidb-parser"

	"github.com/bytebase/bytebase/backend/common/log"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	"github.com/pingcap/tidb/pkg/parser/model"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/types"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	// Register pingcap parser driver.
	driver "github.com/pingcap/tidb/pkg/types/parser_driver"
)

func init() {
	base.RegisterSchemaDiffFunc(storepb.Engine_TIDB, SchemaDiff)
}

// diffNode defines different modification types as the safe change order.
// The safe change order means we can change them with no dependency conflicts as this order.
type diffNode struct {
	// Ignore the case sensitive when comparing the table and view names.
	ignoreCaseSensitive bool

	dropForeignKeyList         []ast.Node
	dropConstraintExceptFkList []ast.Node
	dropIndexList              []ast.Node
	dropViewList               []ast.Node
	dropTableList              []ast.Node

	createTableList             []ast.Node
	alterTableOptionList        []ast.Node
	addAndModifyColumnList      []ast.Node
	dropColumnList              []ast.Node
	createTempViewList          []ast.Node
	createIndexList             []ast.Node
	addConstraintExceptFkList   []ast.Node
	addForeignKeyList           []ast.Node
	createViewList              []ast.Node
	alterTablePartitionedByList map[string]*partitionState
}

func (diff *diffNode) diffSupportedStatement(oldStatement, newStatement string) error {
	oldNodeList, _, err := parser.New().Parse(oldStatement, "", "")
	if err != nil {
		return errors.Wrapf(err, "failed to parse old statement %q", oldStatement)
	}
	newNodeList, _, err := parser.New().Parse(newStatement, "", "")
	if err != nil {
		return errors.Wrapf(err, "failed to parse new statement %q", newStatement)
	}

	oldSchemaInfo, err := diff.buildSchemaInfo(oldNodeList)
	if err != nil {
		return err
	}
	newSchemaInfo, err := diff.buildSchemaInfo(newNodeList)
	if err != nil {
		return err
	}

	for tableName, newTable := range newSchemaInfo.tableMap {
		oldTable, exists := oldSchemaInfo.tableMap[tableName]
		if !exists {
			newTable.createTable.IfNotExists = true
			diff.createTableList = append(diff.createTableList, newTable.createTable)
			// Create indexes.
			for _, index := range newTable.indexMap {
				diff.createIndexList = append(diff.createIndexList, index.createIndex)
			}
			continue
		}
		diff.diffTable(oldTable, newTable)
		delete(oldSchemaInfo.tableMap, tableName)
	}

	for _, oldTable := range oldSchemaInfo.tableMap {
		diff.dropTableList = append(diff.dropTableList, &ast.DropTableStmt{
			IfExists: true,
			Tables:   []*ast.TableName{oldTable.createTable.Table},
		})
	}

	var newViewList []*ast.CreateViewStmt
	for _, newNode := range newNodeList {
		if newView, ok := newNode.(*ast.CreateViewStmt); ok {
			newViewList = append(newViewList, newView)
		}
	}

	if err := diff.diffView(oldSchemaInfo.viewMap, newSchemaInfo.viewMap, newViewList); err != nil {
		return errors.Wrapf(err, "failed to diff view")
	}

	return nil
}

func (diff *diffNode) diffView(oldViewMap viewMap, newViewMap viewMap, newViewList []*ast.CreateViewStmt) error {
	var tempViewList []ast.Node
	var viewList []ast.Node
	for _, view := range newViewList {
		viewName := view.ViewName.Name.O
		if diff.ignoreCaseSensitive {
			viewName = view.ViewName.Name.L
		}
		if newNode, ok := newViewMap[viewName]; ok {
			if !diff.isViewEqual(view, newNode) {
				// Skip predefined view such as the temporary view from mysqldump.
				continue
			}
		}
		oldNode, ok := oldViewMap[viewName]
		if ok {
			if !diff.isViewEqual(view, oldNode) {
				createViewStmt := *view
				createViewStmt.OrReplace = true
				viewList = append(viewList, &createViewStmt)
			}
			// We should delete the view in the oldViewMap, because we will drop the all views in the oldViewMap explicitly at last.
			delete(oldViewMap, viewName)
		} else {
			// We should create the view.
			// We create the temporary view first and replace it to avoid break the dependency like mysqldump does.
			tempViewStmt, err := getTempView(view)
			if err != nil {
				return errors.Wrapf(err, "failed to get temporary view for view %s", view.ViewName.Name.O)
			}
			tempViewList = append(tempViewList, tempViewStmt)
			createViewStmt := *view
			createViewStmt.OrReplace = true
			viewList = append(viewList, &createViewStmt)
		}
	}
	diff.createTempViewList = append(diff.createTempViewList, tempViewList...)
	diff.createViewList = append(diff.createViewList, viewList...)

	// Remove the remaining views in the oldViewMap.
	dropViewStmt := &ast.DropTableStmt{
		IsView: true,
	}
	for _, oldView := range oldViewMap {
		dropViewStmt.Tables = append(dropViewStmt.Tables, oldView.ViewName)
	}
	if len(dropViewStmt.Tables) > 0 {
		diff.dropViewList = append(diff.dropViewList, dropViewStmt)
	}
	return nil
}

func (diff *diffNode) diffTable(oldTable, newTable *tableInfo) {
	diff.diffTableOption(oldTable, newTable)
	diff.diffColumn(oldTable, newTable)
	diff.diffIndex(oldTable, newTable)
	diff.diffConstraint(oldTable, newTable)
	diff.diffPartition(oldTable, newTable)
}

func (diff *diffNode) diffPartition(oldTable, newTable *tableInfo) {
	// The situations we supported:
	// 1. Turn a table partitioned.
	// 2. Add a partition.
	// 3. Delete a partition.
	// 4. Create a new partition table.

	// Skip for some unsupported cases.
	oldHasPartition := oldTable.partition != nil && oldTable.partition.info.tp != storepb.TablePartitionMetadata_TYPE_UNSPECIFIED
	newHasPartition := newTable.partition != nil && newTable.partition.info.tp != storepb.TablePartitionMetadata_TYPE_UNSPECIFIED
	// Turn a partitioned table un-partitioned.
	if oldHasPartition && !newHasPartition {
		return
	}

	// TODO(zp): We use the ALTER TABLE PARTITIONED BY statement to delegate migration calculations to the MySQL server,
	// avoiding the need for manual interventions like ALTER TABLE REORGANIZE PARTITION or ALTER TABLE ADD PARTITION.
	// Are there any drawbacks to this approach?
	oldPartitions := oldTable.partition
	newPartitions := newTable.partition
	equal := reflect.DeepEqual(oldPartitions, newPartitions)
	if !equal {
		diff.alterTablePartitionedByList[newTable.createTable.Table.Name.O] = newPartitions
		return
	}
}

func (diff *diffNode) diffConstraint(oldTable, newTable *tableInfo) {
	oldPrimaryKey, oldConstraintMap := buildConstraintMap(oldTable.createTable)
	// Compare the create definitions.
	for _, constraint := range newTable.createTable.Constraints {
		switch constraint.Tp {
		case ast.ConstraintPrimaryKey:
			if oldPrimaryKey != nil {
				if !isPrimaryKeyEqual(constraint, oldPrimaryKey) {
					diff.dropConstraintExceptFkList = append(diff.dropConstraintExceptFkList, &ast.AlterTableStmt{
						Table: newTable.createTable.Table,
						Specs: []*ast.AlterTableSpec{
							{
								Tp: ast.AlterTableDropPrimaryKey,
							},
						},
					})
					diff.addConstraintExceptFkList = append(diff.addConstraintExceptFkList, &ast.AlterTableStmt{
						Table: newTable.createTable.Table,
						Specs: []*ast.AlterTableSpec{
							{
								Tp:         ast.AlterTableAddConstraint,
								Constraint: constraint,
							},
						},
					})
				}
				oldPrimaryKey = nil
				continue
			}
			diff.addConstraintExceptFkList = append(diff.addConstraintExceptFkList, &ast.AlterTableStmt{
				Table: newTable.createTable.Table,
				Specs: []*ast.AlterTableSpec{
					{
						Tp:         ast.AlterTableAddConstraint,
						Constraint: constraint,
					},
				},
			})
		case ast.ConstraintForeignKey:
			if oldConstraint, ok := oldConstraintMap[strings.ToLower(constraint.Name)]; ok {
				if !isForeignKeyConstraintEqual(constraint, oldConstraint) {
					diff.dropForeignKeyList = append(diff.dropForeignKeyList, &ast.AlterTableStmt{
						Table: newTable.createTable.Table,
						Specs: []*ast.AlterTableSpec{
							{
								Tp:   ast.AlterTableDropForeignKey,
								Name: constraint.Name,
							},
						},
					})
					diff.addForeignKeyList = append(diff.addForeignKeyList, &ast.AlterTableStmt{
						Table: newTable.createTable.Table,
						Specs: []*ast.AlterTableSpec{
							{
								Tp:         ast.AlterTableAddConstraint,
								Constraint: constraint,
							},
						},
					})
				}
				delete(oldConstraintMap, strings.ToLower(constraint.Name))
				continue
			}
			diff.addForeignKeyList = append(diff.addForeignKeyList, &ast.AlterTableStmt{
				Table: newTable.createTable.Table,
				Specs: []*ast.AlterTableSpec{
					{
						Tp:         ast.AlterTableAddConstraint,
						Constraint: constraint,
					},
				},
			})
		case ast.ConstraintCheck:
			if oldConstraint, ok := oldConstraintMap[strings.ToLower(constraint.Name)]; ok {
				if !isCheckConstraintEqual(constraint, oldConstraint) {
					diff.dropConstraintExceptFkList = append(diff.dropConstraintExceptFkList, &ast.AlterTableStmt{
						Table: newTable.createTable.Table,
						Specs: []*ast.AlterTableSpec{
							{
								Tp:         ast.AlterTableDropCheck,
								Constraint: constraint,
							},
						},
					})
					diff.addConstraintExceptFkList = append(diff.addConstraintExceptFkList, &ast.AlterTableStmt{
						Table: newTable.createTable.Table,
						Specs: []*ast.AlterTableSpec{
							{
								Tp:         ast.AlterTableAddConstraint,
								Constraint: constraint,
							},
						},
					})
				}
				delete(oldConstraintMap, strings.ToLower(constraint.Name))
				continue
			}
			diff.addConstraintExceptFkList = append(diff.addConstraintExceptFkList, &ast.AlterTableStmt{
				Table: newTable.createTable.Table,
				Specs: []*ast.AlterTableSpec{
					{
						Tp:         ast.AlterTableAddConstraint,
						Constraint: constraint,
					},
				},
			})
		}
	}

	if oldPrimaryKey != nil {
		diff.dropConstraintExceptFkList = append(diff.dropConstraintExceptFkList, &ast.AlterTableStmt{
			Table: newTable.createTable.Table,
			Specs: []*ast.AlterTableSpec{
				{
					Tp: ast.AlterTableDropPrimaryKey,
				},
			},
		})
	}

	for _, oldConstraint := range oldConstraintMap {
		switch oldConstraint.Tp {
		case ast.ConstraintCheck:
			diff.dropConstraintExceptFkList = append(diff.dropConstraintExceptFkList, &ast.AlterTableStmt{
				Table: newTable.createTable.Table,
				Specs: []*ast.AlterTableSpec{
					{
						Tp:         ast.AlterTableDropCheck,
						Constraint: oldConstraint,
					},
				},
			})
		case ast.ConstraintForeignKey:
			diff.dropForeignKeyList = append(diff.dropForeignKeyList, &ast.AlterTableStmt{
				Table: newTable.createTable.Table,
				Specs: []*ast.AlterTableSpec{
					{
						Tp:   ast.AlterTableDropForeignKey,
						Name: oldConstraint.Name,
					},
				},
			})
		}
	}
}

func (diff *diffNode) diffIndex(oldTable, newTable *tableInfo) {
	for indexName, newIndex := range newTable.indexMap {
		if oldIndex, ok := oldTable.indexMap[indexName]; ok {
			if !isIndexEqual(newIndex.createIndex, oldIndex.createIndex) {
				diff.dropIndexList = append(diff.dropIndexList, &ast.DropIndexStmt{
					IndexName: oldIndex.createIndex.IndexName,
					Table:     oldIndex.createIndex.Table,
				})
				diff.createIndexList = append(diff.createIndexList, newIndex.createIndex)
			}
			delete(oldTable.indexMap, indexName)
			continue
		}
		diff.createIndexList = append(diff.createIndexList, newIndex.createIndex)
	}

	for _, oldIndex := range oldTable.indexMap {
		diff.dropIndexList = append(diff.dropIndexList, &ast.DropIndexStmt{
			IndexName: oldIndex.createIndex.IndexName,
			Table:     oldIndex.createIndex.Table,
		})
	}
}

func (diff *diffNode) diffColumn(oldTable, newTable *tableInfo) {
	// We use a single ALTER TABLE statement to add and modify columns,
	// because we need to maintain a fixed order of these two operations.
	// This approach ensures that we can manipulate the column position as needed.
	oldColumnMap := buildColumnMap(oldTable.createTable.Cols)
	oldColumnPositionMap := buildColumnPositionMap(oldTable.createTable)
	for idx, columnDef := range newTable.createTable.Cols {
		// Column names are always case insensitive.
		newColumnName := columnDef.Name.Name.L
		oldColumnDef, ok := oldColumnMap[newColumnName]
		if !ok {
			columnPosition := &ast.ColumnPosition{Tp: ast.ColumnPositionFirst}
			if idx >= 1 {
				columnPosition.Tp = ast.ColumnPositionAfter
				columnPosition.RelativeColumn = &ast.ColumnName{Name: model.NewCIStr(newTable.createTable.Cols[idx-1].Name.Name.O)}
			}
			addAndModifyColumnStatement := &ast.AlterTableStmt{Table: newTable.createTable.Table}
			addAndModifyColumnStatement.Specs = append(addAndModifyColumnStatement.Specs, &ast.AlterTableSpec{
				Tp:         ast.AlterTableAddColumns,
				NewColumns: []*ast.ColumnDef{columnDef},
				Position:   columnPosition,
			})
			diff.addAndModifyColumnList = append(diff.addAndModifyColumnList, addAndModifyColumnStatement)
			continue
		}

		// Compare the column positions.
		columnPosition := &ast.ColumnPosition{Tp: ast.ColumnPositionNone}
		columnPosInOld := oldColumnPositionMap[newColumnName]
		if hasColumnsIntersection(oldTable.createTable.Cols[:columnPosInOld], newTable.createTable.Cols[idx+1:]) {
			if idx == 0 {
				columnPosition.Tp = ast.ColumnPositionFirst
			} else {
				columnPosition.Tp = ast.ColumnPositionAfter
				columnPosition.RelativeColumn = &ast.ColumnName{Name: model.NewCIStr(newTable.createTable.Cols[idx-1].Name.Name.O)}
			}
		}
		// Compare the column definitions.
		if !isColumnEqual(oldColumnDef, columnDef) || columnPosition.Tp != ast.ColumnPositionNone {
			addAndModifyColumnStatement := &ast.AlterTableStmt{Table: newTable.createTable.Table}
			addAndModifyColumnStatement.Specs = append(addAndModifyColumnStatement.Specs, &ast.AlterTableSpec{
				Tp:         ast.AlterTableModifyColumn,
				NewColumns: []*ast.ColumnDef{columnDef},
				Position:   columnPosition,
			})
			diff.addAndModifyColumnList = append(diff.addAndModifyColumnList, addAndModifyColumnStatement)
		}
		delete(oldColumnMap, newColumnName)
	}
	// TODO(zp): add an option to control whether to drop the excess columns.
	for _, columnDef := range oldColumnMap {
		diff.dropColumnList = append(diff.dropColumnList, &ast.AlterTableStmt{
			Table: newTable.createTable.Table,
			Specs: []*ast.AlterTableSpec{
				{
					Tp: ast.AlterTableDropColumn,
					OldColumnName: &ast.ColumnName{
						Name: model.NewCIStr(columnDef.Name.Name.O),
					},
				},
			},
		})
	}
}

func (diff *diffNode) diffTableOption(oldTable, newTable *tableInfo) {
	if alterTableOptionStmt := diffTableOptions(newTable.createTable.Table, oldTable.createTable.Options, newTable.createTable.Options); alterTableOptionStmt != nil {
		diff.alterTableOptionList = append(diff.alterTableOptionList, alterTableOptionStmt)
	}
}

func (diff *diffNode) deparse() (string, error) {
	var buf bytes.Buffer
	flag := format.DefaultRestoreFlags | format.RestoreStringWithoutCharset | format.RestorePrettyFormat

	if err := sortAndWriteNodeList(&buf, diff.dropForeignKeyList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.dropConstraintExceptFkList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.dropIndexList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.dropViewList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.dropTableList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.createTableList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.alterTableOptionList, flag); err != nil {
		return "", err
	}
	if err := writeNodeList(&buf, diff.addAndModifyColumnList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.dropColumnList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteAlterTablePartitionedByList(&buf, diff.alterTablePartitionedByList); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.createTempViewList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.createIndexList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.addConstraintExceptFkList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.addForeignKeyList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.createViewList, flag); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func sortAndWriteAlterTablePartitionedByList(buf io.Writer, partitions map[string]*partitionState) error {
	// Sort by table name.
	tableNames := make([]string, 0, len(partitions))
	for tableName := range partitions {
		tableNames = append(tableNames, tableName)
	}
	sort.Strings(tableNames)
	for _, tableName := range tableNames {
		var body strings.Builder
		if _, err := body.WriteString(fmt.Sprintf("ALTER TABLE `%s` PARTITION BY ", tableName)); err != nil {
			return err
		}
		if err := partitions[tableName].toString(&body); err != nil {
			return err
		}
		if _, err := buf.Write([]byte(body.String())); err != nil {
			return err
		}
		if _, err := buf.Write([]byte(";\n")); err != nil {
			return err
		}
	}

	return nil
}

// Copy from backend/plugin/schema/mysql/state.go.
func (p *partitionState) toString(buf io.StringWriter) error {
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

// splitPartitionExprIntoFields splits the partition expression by ',', and trims the leading and trailing '`' for each element.
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

// constraintMap returns a map of constraint name to constraint.
type constraintMap map[string]*ast.Constraint

// SchemaDiff returns the schema diff.
// It only supports schema information from mysqldump.
func SchemaDiff(ctx base.DiffContext, oldStmt, newStmt string) (string, error) {
	diff := &diffNode{
		ignoreCaseSensitive:         ctx.IgnoreCaseSensitive,
		alterTablePartitionedByList: make(map[string]*partitionState),
	}
	if err := diff.diffSupportedStatement(oldStmt, newStmt); err != nil {
		return "", err
	}

	return diff.deparse()
}

func writeNodeStatement(w format.RestoreWriter, n ast.Node, flags format.RestoreFlags, ending string) error {
	restoreCtx := format.NewRestoreCtx(flags, w)
	if err := n.Restore(restoreCtx); err != nil {
		return err
	}
	if _, err := w.Write([]byte(";" + ending)); err != nil {
		return err
	}
	return nil
}

func getID(node ast.Node) string {
	switch in := node.(type) {
	case *ast.CreateTableStmt:
		return in.Table.Name.String()
	case *ast.DropTableStmt:
		return in.Tables[0].Name.String()
	case *ast.AlterTableStmt:
		for _, spec := range in.Specs {
			switch spec.Tp {
			case ast.AlterTableOption:
				return in.Table.Name.String()
			case ast.AlterTableAddColumns:
				return fmt.Sprintf("%s.%s", in.Table.Name.String(), spec.NewColumns[0].Name)
			case ast.AlterTableDropColumn:
				return fmt.Sprintf("%s.%s", in.Table.Name.String(), spec.OldColumnName.Name.String())
			case ast.AlterTableModifyColumn:
				return fmt.Sprintf("%s.%s", in.Table.Name.String(), spec.NewColumns[0].Name)
			case ast.AlterTableAddConstraint:
				return fmt.Sprintf("%s.%s", in.Table.Name.String(), spec.Constraint.Name)
			case ast.AlterTableDropForeignKey:
				return fmt.Sprintf("%s.%s", in.Table.Name.String(), spec.Name)
			case ast.AlterTableDropPrimaryKey:
				return in.Table.Name.String()
			case ast.AlterTableDropCheck:
				return fmt.Sprintf("%s.%s", in.Table.Name.String(), spec.Constraint.Name)
			}
		}
	case *ast.CreateIndexStmt:
		return fmt.Sprintf("%s.%s", in.Table.Name.String(), in.IndexName)
	case *ast.DropIndexStmt:
		return fmt.Sprintf("%s.%s", in.Table.Name.String(), in.IndexName)
	case *ast.CreateViewStmt:
		return in.ViewName.Name.String()
	}
	return ""
}

func writeNodeList(w format.RestoreWriter, ns []ast.Node, flags format.RestoreFlags) error {
	for _, n := range ns {
		if err := writeNodeStatement(w, n, flags, "\n"); err != nil {
			return err
		}
	}
	return nil
}

func sortAndWriteNodeList(w format.RestoreWriter, ns []ast.Node, flags format.RestoreFlags) error {
	sort.Slice(ns, func(i, j int) bool {
		return getID(ns[i]) < getID(ns[j])
	})

	for _, n := range ns {
		if err := writeNodeStatement(w, n, flags, "\n\n"); err != nil {
			return err
		}
	}
	return nil
}

type schemaInfo struct {
	tableMap tableMap
	viewMap  viewMap
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

type viewMap map[string]*ast.CreateViewStmt

type indexInfo struct {
	createIndex *ast.CreateIndexStmt
}
type indexMap map[string]*indexInfo

func newIndexInfo(createIndex *ast.CreateIndexStmt) *indexInfo {
	return &indexInfo{
		createIndex: createIndex,
	}
}

type tableInfo struct {
	createTable *ast.CreateTableStmt
	indexMap    indexMap
	partition   *partitionState
}
type tableMap map[string]*tableInfo

func newTableInfo(createTable *ast.CreateTableStmt) (*tableInfo, error) {
	result := &tableInfo{
		createTable: createTable,
		indexMap:    make(indexMap),
	}

	var newConstraintList []*ast.Constraint
	for _, constraint := range createTable.Constraints {
		if createIndex := transformConstraintToIndex(createTable.Table, constraint); createIndex != nil {
			if _, exists := result.indexMap[strings.ToLower(createIndex.IndexName)]; exists {
				return nil, errors.Errorf("Try to create index `%s` on table `%s`, but index already exists", createIndex.IndexName, createIndex.Table.Name.String())
			}
			result.indexMap[strings.ToLower(constraint.Name)] = newIndexInfo(createIndex)
		} else {
			newConstraintList = append(newConstraintList, constraint)
		}
	}
	createTable.Constraints = newConstraintList

	return result, nil
}

func transformConstraintToIndex(tableName *ast.TableName, constraint *ast.Constraint) *ast.CreateIndexStmt {
	indexType := ast.IndexKeyTypeNone
	switch constraint.Tp {
	case ast.ConstraintKey, ast.ConstraintIndex:
	case ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex:
		indexType = ast.IndexKeyTypeUnique
	case ast.ConstraintFulltext:
		indexType = ast.IndexKeyTypeFullText
	default:
		return nil
	}
	result := &ast.CreateIndexStmt{
		IndexName:               constraint.Name,
		Table:                   tableName,
		IndexPartSpecifications: constraint.Keys,
		IndexOption:             constraint.Option,
		KeyType:                 indexType,
	}
	if result.IndexOption == nil {
		result.IndexOption = &ast.IndexOption{Tp: model.IndexTypeInvalid}
	}
	return result
}

// buildSchemaInfo returns schema information built by statements.
func (diff *diffNode) buildSchemaInfo(nodes []ast.StmtNode) (*schemaInfo, error) {
	result := &schemaInfo{
		tableMap: make(tableMap),
		viewMap:  make(viewMap),
	}
	for _, node := range nodes {
		switch stmt := node.(type) {
		case *ast.CreateTableStmt:
			var err error
			tableName := stmt.Table.Name.O
			if diff.ignoreCaseSensitive {
				tableName = stmt.Table.Name.L
			}
			if result.tableMap[tableName], err = newTableInfo(stmt); err != nil {
				return nil, err
			}
			createTableText := node.OriginalText()
			list, err := ANTLRParseTiDB(createTableText)
			if err != nil {
				return nil, err
			}

			listener := &tidbTransformer{
				currentTable: stmt.Table.Name.O,
				tableState:   result.tableMap[tableName],
			}

			for _, stmt := range list {
				antlr.ParseTreeWalkerDefault.Walk(listener, stmt.Tree)
			}
		case *ast.CreateIndexStmt:
			tableName := stmt.Table.Name.O
			if diff.ignoreCaseSensitive {
				tableName = stmt.Table.Name.L
			}
			table, exists := result.tableMap[tableName]
			if !exists {
				return nil, errors.Errorf("Try to create index `%s` on table `%s`, but table not found", stmt.IndexName, stmt.Table.Name.String())
			}
			// Index names are always case insensitive
			if _, exists := table.indexMap[strings.ToLower(stmt.IndexName)]; exists {
				return nil, errors.Errorf("Try to create index `%s` on table `%s`, but index already exists", stmt.IndexName, stmt.Table.Name.String())
			}
			table.indexMap[strings.ToLower(stmt.IndexName)] = newIndexInfo(stmt)
		case *ast.CreateViewStmt:
			viewName := stmt.ViewName.Name.O
			if diff.ignoreCaseSensitive {
				viewName = stmt.ViewName.Name.L
			}
			result.viewMap[viewName] = stmt
		default:
		}
	}
	return result, nil
}

type tidbTransformer struct {
	*tidbparser.BaseTiDBParserListener

	tableState   *tableInfo
	currentTable string
	err          error
}

func (t *tidbTransformer) EnterPartitionClause(ctx *tidbparser.PartitionClauseContext) {
	if t.err != nil || t.currentTable == "" {
		return
	}

	table := t.tableState
	if table == nil {
		t.err = errors.New("table not found: " + t.currentTable)
		return
	}

	parititonInfo := partitionInfo{}

	iTypeDefCtx := ctx.PartitionTypeDef()
	if iTypeDefCtx != nil {
		switch typeDefCtx := iTypeDefCtx.(type) {
		case *tidbparser.PartitionDefKeyContext:
			parititonInfo.tp = storepb.TablePartitionMetadata_KEY
			if typeDefCtx.LINEAR_SYMBOL() != nil {
				parititonInfo.tp = storepb.TablePartitionMetadata_LINEAR_KEY
			}
			// TODO(zp): handle the key algorithm
			if typeDefCtx.IdentifierList() != nil {
				identifiers := extractIdentifierList(typeDefCtx.IdentifierList())
				for i, identifier := range identifiers {
					identifier := strings.TrimSpace(identifier)
					if !strings.HasPrefix(identifier, "`") || !strings.HasSuffix(identifier, "`") {
						identifiers[i] = fmt.Sprintf("`%s`", identifier)
					}
				}
				parititonInfo.expr = strings.Join(identifiers, ",")
			}
		case *tidbparser.PartitionDefHashContext:
			parititonInfo.tp = storepb.TablePartitionMetadata_HASH
			if typeDefCtx.LINEAR_SYMBOL() != nil {
				parititonInfo.tp = storepb.TablePartitionMetadata_LINEAR_HASH
			}
			bitExprText := typeDefCtx.GetParser().GetTokenStream().GetTextFromRuleContext(typeDefCtx.BitExpr())
			bitExprFields := strings.Split(bitExprText, ",")
			for i, bitExprField := range bitExprFields {
				bitExprField := strings.TrimSpace(bitExprField)
				if !strings.HasPrefix(bitExprField, "`") || !strings.HasSuffix(bitExprField, "`") {
					bitExprFields[i] = fmt.Sprintf("`%s`", bitExprField)
				}
			}
			parititonInfo.expr = strings.Join(bitExprFields, ",")
		case *tidbparser.PartitionDefRangeListContext:
			if typeDefCtx.RANGE_SYMBOL() != nil {
				parititonInfo.tp = storepb.TablePartitionMetadata_RANGE
			} else {
				parititonInfo.tp = storepb.TablePartitionMetadata_LIST
			}
			if typeDefCtx.COLUMNS_SYMBOL() != nil {
				if parititonInfo.tp == storepb.TablePartitionMetadata_RANGE {
					parititonInfo.tp = storepb.TablePartitionMetadata_RANGE_COLUMNS
				} else {
					parititonInfo.tp = storepb.TablePartitionMetadata_LIST_COLUMNS
				}

				identifierList := extractIdentifierList(typeDefCtx.IdentifierList())
				for i, identifier := range identifierList {
					identifier := strings.TrimSpace(identifier)
					if !strings.HasPrefix(identifier, "`") || !strings.HasSuffix(identifier, "`") {
						identifierList[i] = fmt.Sprintf("`%s`", identifier)
					}
				}
				parititonInfo.expr = strings.Join(identifierList, ",")
			} else {
				bitExprText := typeDefCtx.GetParser().GetTokenStream().GetTextFromRuleContext(typeDefCtx.BitExpr())
				bitExprFields := strings.Split(bitExprText, ",")
				for i, bitExprField := range bitExprFields {
					bitExprField := strings.TrimSpace(bitExprField)
					if !strings.HasPrefix(bitExprField, "`") || !strings.HasSuffix(bitExprField, "`") {
						bitExprFields[i] = fmt.Sprintf("`%s`", bitExprField)
					}
				}
				parititonInfo.expr = strings.Join(bitExprFields, ",")
			}
		default:
			t.err = errors.New("unknown partition type")
			return
		}
	}

	if n := ctx.Real_ulong_number(); n != nil {
		number, err := strconv.ParseInt(n.GetText(), 10, 64)
		if err != nil {
			t.err = errors.Wrap(err, "failed to parse partition number")
			return
		}
		parititonInfo.useDefault = int(number)
	}

	var subInfo *partitionInfo
	if subPartitionCtx := ctx.SubPartitions(); subPartitionCtx != nil {
		subInfo = new(partitionInfo)
		if subPartitionCtx.HASH_SYMBOL() != nil {
			subInfo.tp = storepb.TablePartitionMetadata_HASH
			if subPartitionCtx.LINEAR_SYMBOL() != nil {
				subInfo.tp = storepb.TablePartitionMetadata_LINEAR_HASH
			}
			if bitExprCtx := subPartitionCtx.BitExpr(); bitExprCtx != nil {
				bitExprText := bitExprCtx.GetParser().GetTokenStream().GetTextFromRuleContext(bitExprCtx)
				bitExprFields := strings.Split(bitExprText, ",")
				for i, bitExprField := range bitExprFields {
					bitExprField := strings.TrimSpace(bitExprField)
					if !strings.HasPrefix(bitExprField, "`") || !strings.HasSuffix(bitExprField, "`") {
						bitExprFields[i] = fmt.Sprintf("`%s`", bitExprField)
					}
				}
				subInfo.expr = strings.Join(bitExprFields, ",")
			}
		} else if subPartitionCtx.KEY_SYMBOL() != nil {
			subInfo.tp = storepb.TablePartitionMetadata_KEY
			if subPartitionCtx.LINEAR_SYMBOL() != nil {
				subInfo.tp = storepb.TablePartitionMetadata_LINEAR_KEY
			}
			if identifierListParensCtx := subPartitionCtx.IdentifierListWithParentheses(); identifierListParensCtx != nil {
				identifiers := extractIdentifierList(identifierListParensCtx.IdentifierList())
				for i, identifier := range identifiers {
					identifier := strings.TrimSpace(identifier)
					if !strings.HasPrefix(identifier, "`") || !strings.HasSuffix(identifier, "`") {
						identifiers[i] = fmt.Sprintf("`%s`", identifier)
					}
				}
				subInfo.expr = strings.Join(identifiers, ",")
			}
		}

		if n := subPartitionCtx.Real_ulong_number(); n != nil {
			number, err := strconv.ParseInt(n.GetText(), 10, 64)
			if err != nil {
				t.err = errors.Wrap(err, "failed to parse sub partition number")
				return
			}
			subInfo.useDefault = int(number)
		}
	}

	partitionDefinitions := make(map[string]*partitionDefinition)
	var allPartDefs []tidbparser.IPartitionDefinitionContext
	if v := ctx.PartitionDefinitions(); v != nil {
		allPartDefs = ctx.PartitionDefinitions().AllPartitionDefinition()
	}
	for i, partDef := range allPartDefs {
		pd := &partitionDefinition{
			id:   i + 1,
			name: NormalizeTiDBIdentifier(partDef.Identifier()),
		}
		switch parititonInfo.tp {
		case storepb.TablePartitionMetadata_RANGE_COLUMNS, storepb.TablePartitionMetadata_RANGE:
			if partDef.LESS_SYMBOL() == nil {
				t.err = errors.New("RANGE partition but no LESS THAN clause")
				return
			}
			if partDef.PartitionValueItemListParen() != nil {
				itemsText := partDef.PartitionValueItemListParen().GetParser().GetTokenStream().GetTextFromInterval(
					antlr.NewInterval(
						partDef.PartitionValueItemListParen().OPEN_PAR_SYMBOL().GetSymbol().GetTokenIndex()+1,
						partDef.PartitionValueItemListParen().CLOSE_PAR_SYMBOL().GetSymbol().GetTokenIndex()-1,
					),
				)
				itemsTextFields := strings.Split(itemsText, ",")
				for i, itemsTextField := range itemsTextFields {
					itemsTextField := strings.TrimSpace(itemsTextField)
					if strings.HasPrefix(itemsTextField, "`") && strings.HasSuffix(itemsTextField, "`") {
						itemsTextField = itemsTextField[1 : len(itemsTextField)-1]
					}
					itemsTextFields[i] = itemsTextField
				}
				pd.value = strings.Join(itemsTextFields, ",")
			} else {
				pd.value = "MAXVALUE"
			}
		case storepb.TablePartitionMetadata_LIST_COLUMNS, storepb.TablePartitionMetadata_LIST:
			if partDef.PartitionValuesIn() == nil {
				t.err = errors.New("COLUMNS partition but no partition value item in IN clause")
				return
			}
			var itemsText string
			if partDef.PartitionValuesIn().OPEN_PAR_SYMBOL() != nil {
				itemsText = partDef.PartitionValuesIn().GetParser().GetTokenStream().GetTextFromInterval(
					antlr.NewInterval(
						partDef.PartitionValuesIn().OPEN_PAR_SYMBOL().GetSymbol().GetTokenIndex()+1,
						partDef.PartitionValuesIn().CLOSE_PAR_SYMBOL().GetSymbol().GetTokenIndex()-1,
					),
				)
			} else {
				itemsText = partDef.PartitionValuesIn().GetParser().GetTokenStream().GetTextFromRuleContext(partDef.PartitionValuesIn().PartitionValueItemListParen(0))
			}

			itemsTextFields := strings.Split(itemsText, ",")
			for i, itemsTextField := range itemsTextFields {
				itemsTextField := strings.TrimSpace(itemsTextField)
				if strings.HasPrefix(itemsTextField, "`") && strings.HasSuffix(itemsTextField, "`") {
					itemsTextField = itemsTextField[1 : len(itemsTextField)-1]
				}
				itemsTextFields[i] = itemsTextField
			}
			pd.value = strings.Join(itemsTextFields, ",")
		case storepb.TablePartitionMetadata_HASH, storepb.TablePartitionMetadata_LINEAR_HASH, storepb.TablePartitionMetadata_KEY, storepb.TablePartitionMetadata_LINEAR_KEY:
		default:
			t.err = errors.New("unknown partition type")
			return
		}

		if subInfo != nil {
			allSubpartitions := partDef.AllSubpartitionDefinition()
			if subInfo.tp == storepb.TablePartitionMetadata_TYPE_UNSPECIFIED && len(allSubpartitions) > 0 {
				t.err = errors.New("specify subpartition definition but no subpartition type specified")
				return
			}
			subPartitionDefinitions := make(map[string]*partitionDefinition)
			for i, sub := range allSubpartitions {
				subpd := &partitionDefinition{
					id:   i + 1,
					name: NormalizeTiDBTextOrIdentifier(sub.TextOrIdentifier()),
				}
				subPartitionDefinitions[subpd.name] = subpd
			}
			pd.subpartitions = subPartitionDefinitions
		}

		partitionDefinitions[pd.name] = pd
	}

	table.partition = &partitionState{
		info:       parititonInfo,
		subInfo:    subInfo,
		partitions: partitionDefinitions,
	}
}

// extract column names.
func extractIdentifierList(ctx tidbparser.IIdentifierListContext) []string {
	var result []string
	for _, identifier := range ctx.AllIdentifier() {
		result = append(result, NormalizeTiDBIdentifier(identifier))
	}
	return result
}

// getTempView returns the temporary view name and the create statement.
func getTempView(stmt *ast.CreateViewStmt) (*ast.CreateViewStmt, error) {
	// We create the temp view similar to what mysqldump does.
	// Create a temporary view with the same name as the view. Its columns should
	// have the same name in order to satisfy views that depend on this view.
	// This temporary view will be removed when the actual view is created.
	// The column properties are unnecessary and not preserved in this temporary view.
	// because other views only need to reference the column name.
	//  Example: SELECT 1 AS colName1, 1 AS colName2.
	// TODO(zp): support SDL for GitOps.
	var selectFields []*ast.SelectField
	// mysqldump always show field list
	if len(stmt.Cols) > 0 {
		for _, col := range stmt.Cols {
			selectFields = append(selectFields, &ast.SelectField{
				Expr: &driver.ValueExpr{
					Datum: types.NewDatum(1),
				},
				AsName: col,
			})
		}
	} else {
		colName, err := extractColNameFromSelectListClauseOfCreateViewStmt(stmt.Select)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract column name from select list clause of create view statement")
		}
		for _, name := range colName {
			selectFields = append(selectFields, &ast.SelectField{
				Expr: &driver.ValueExpr{
					Datum: types.NewDatum(1),
				},
				AsName: model.NewCIStr(name),
			})
		}
	}

	return &ast.CreateViewStmt{
		ViewName: stmt.ViewName,
		Select: &ast.SelectStmt{
			SelectStmtOpts: &ast.SelectStmtOpts{
				// Avoid generating SQL_NO_CACHE
				// https://sourcegraph.com/github.com/pingcap/tidb/-/blob/parser/ast/dml.go?L1234
				SQLCache: true,
			},
			Fields: &ast.FieldList{
				Fields: selectFields,
			},
		},
		OrReplace: true,
		// Avoid nil pointer dereference panic.
		// https://sourcegraph.com/github.com/pingcap/tidb/-/blob/parser/ast/ddl.go?L1398
		Definer:     stmt.Definer,
		Security:    stmt.Security,
		CheckOption: model.CheckOptionCascaded,
	}, nil
}

// extractColNameFromCreateViewStmt extracts column names from create view statement.
// For example, CREATE OR REPLACE VIEW `v` AS select `d` as `c` WILL return `c`.
func extractColNameFromSelectListClauseOfCreateViewStmt(stmt ast.Node) ([]string, error) {
	var result []string
	switch stmt := (stmt).(type) {
	case *ast.SelectStmt:
		for _, field := range stmt.Fields.Fields {
			if field.WildCard != nil {
				return nil, errors.New("wildcard(*) is not supported now in select statement")
			}
			var fieldName string
			if field.AsName.O != "" {
				fieldName = field.AsName.O
			} else {
				fieldName = field.Expr.(*ast.ColumnNameExpr).Name.Name.O
			}
			result = append(result, fieldName)
		}
		return result, nil
	case *ast.SetOprStmt:
		// For SetOprStmt, we focus on the first select statement, for example:
		// with `tt` as (select `t1`.`id` AS `id` from `t1` union select `t2`.`id2` AS `id2` from `t2`) select `tt`.`id` AS `id` from `tt` union select `t1`.`id` AS `id` from `t1`
		// we just focus on select `tt`.`id` as `id` from `tt`.
		if len(stmt.SelectList.Selects) == 0 {
			return nil, errors.New("select list in SetOprStmt is empty")
		}
		return extractColNameFromSelectListClauseOfCreateViewStmt(stmt.SelectList.Selects[0])
	case *ast.SetOprSelectList:
		if len(stmt.Selects) == 0 {
			return nil, errors.New("select list in SetOprSelectList is empty")
		}
		return extractColNameFromSelectListClauseOfCreateViewStmt(stmt.Selects[0])
	default:
		stmtStr, err := toString(stmt)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert create view statement to string")
		}
		return nil, errors.Errorf("unsupported create view statement %q which select is not supported node", stmtStr)
	}
}

// buildColumnMap returns a map of column name to column definition on a given table.
func buildColumnMap(columnList []*ast.ColumnDef) map[string]*ast.ColumnDef {
	oldColumnMap := make(map[string]*ast.ColumnDef)
	for _, columnDef := range columnList {
		// Column names are always case insensitive.
		oldColumnMap[columnDef.Name.Name.L] = columnDef
	}
	return oldColumnMap
}

// buildColumnPositionMap returns a map of column name to column position.
func buildColumnPositionMap(stmt *ast.CreateTableStmt) map[string]int {
	m := make(map[string]int)
	for i, col := range stmt.Cols {
		// Column names are always case insensitive.
		m[col.Name.Name.L] = i
	}
	return m
}

func buildConstraintMap(stmt *ast.CreateTableStmt) (*ast.Constraint, constraintMap) {
	var primaryKey *ast.Constraint
	constraintMap := make(constraintMap)
	for _, constraint := range stmt.Constraints {
		switch constraint.Tp {
		case ast.ConstraintForeignKey, ast.ConstraintCheck:
			constraintMap[strings.ToLower(constraint.Name)] = constraint
		case ast.ConstraintPrimaryKey:
			primaryKey = constraint
		default:
		}
	}
	return primaryKey, constraintMap
}

// isColumnEqual returns true if definitions of two columns with the same name are the same.
func isColumnEqual(o, n *ast.ColumnDef) bool {
	if !isColumnTypesEqual(o, n) {
		return false
	}
	if !isColumnOptionsEqual(o.Options, n.Options) {
		return false
	}
	return true
}

func isColumnTypesEqual(o, n *ast.ColumnDef) bool {
	return o.Tp.String() == n.Tp.String()
}

func isColumnOptionsEqual(o, n []*ast.ColumnOption) bool {
	oldCollate, oldNormalizeOptions := normalizeColumnOptions(o)
	newCollate, newNormalizeOptions := normalizeColumnOptions(n)
	if len(oldNormalizeOptions) != len(newNormalizeOptions) {
		return false
	}
	if newCollate != nil {
		if oldCollate == nil {
			return false
		}
		if oldCollate.StrValue != newCollate.StrValue {
			return false
		}
	}
	for idx, oldOption := range oldNormalizeOptions {
		oldOptionStr, err := toString(oldOption)
		if err != nil {
			slog.Error("failed to convert old column option to string", log.BBError(err))
			return false
		}
		newOption := newNormalizeOptions[idx]
		newOptionStr, err := toString(newOption)
		if err != nil {
			slog.Error("failed to convert new column option to string", log.BBError(err))
			return false
		}
		if oldOptionStr != newOptionStr {
			return false
		}
	}
	return true
}

// normalizeColumnOptions normalizes the column options.
// It skips the NULL option, NO option and then order the options by OptionType.
func normalizeColumnOptions(options []*ast.ColumnOption) (*ast.ColumnOption, []*ast.ColumnOption) {
	var retOptions []*ast.ColumnOption
	var collateOption *ast.ColumnOption
	for _, option := range options {
		switch option.Tp {
		case ast.ColumnOptionCollate:
			collateOption = option
			continue
		case ast.ColumnOptionNull:
			continue
		case ast.ColumnOptionNoOption:
			continue
		case ast.ColumnOptionDefaultValue:
			if option.Expr.GetType().GetType() == mysql.TypeNull {
				continue
			}
		}
		retOptions = append(retOptions, option)
	}
	sort.Slice(retOptions, func(i, j int) bool {
		return retOptions[i].Tp < retOptions[j].Tp
	})
	return collateOption, retOptions
}

// isIndexEqual returns true if definitions of two indexes are the same.
func isIndexEqual(o, n *ast.CreateIndexStmt) bool {
	// CREATE [UNIQUE | FULLTEXT | SPATIAL] INDEX index_name
	// [index_type]
	// ON tbl_name (key_part,...)
	// [index_option]
	// [algorithm_option | lock_option] ...

	// MySQL index names are case insensitive.
	if !strings.EqualFold(o.IndexName, n.IndexName) {
		return false
	}
	if (o.IndexOption == nil) != (n.IndexOption == nil) {
		return false
	}
	if o.IndexOption != nil && n.IndexOption != nil {
		if o.IndexOption.Tp != n.IndexOption.Tp {
			return false
		}
	}

	if !isKeyPartEqual(o.IndexPartSpecifications, n.IndexPartSpecifications) {
		return false
	}
	if !isIndexOptionEqual(o.IndexOption, n.IndexOption) {
		return false
	}
	return true
}

// isPrimaryKeyEqual returns true if definitions of two indexes are the same.
func isPrimaryKeyEqual(o, n *ast.Constraint) bool {
	// {INDEX | KEY} [index_name] [index_type] (key_part,...) [index_option] ...
	if o.Name != n.Name {
		return false
	}
	if (o.Option == nil) != (n.Option == nil) {
		return false
	}
	if o.Option != nil && n.Option != nil {
		if o.Option.Tp != n.Option.Tp {
			return false
		}
	}

	if !isKeyPartEqual(o.Keys, n.Keys) {
		return false
	}
	if !isIndexOptionEqual(o.Option, n.Option) {
		return false
	}
	return true
}

// isKeyPartEqual returns true if two key parts are the same.
func isKeyPartEqual(o, n []*ast.IndexPartSpecification) bool {
	if len(o) != len(n) {
		return false
	}
	// key_part: {col_name [(length)] | (expr)} [ASC | DESC]
	for idx, oldKeyPart := range o {
		newKeyPart := n[idx]
		if (oldKeyPart.Column == nil) != (newKeyPart.Column == nil) {
			return false
		}
		if oldKeyPart.Column != nil && newKeyPart.Column != nil {
			if oldKeyPart.Column.Name.String() != newKeyPart.Column.Name.String() {
				return false
			}
			if oldKeyPart.Length != newKeyPart.Length {
				return false
			}
		}
		if (oldKeyPart.Expr == nil) != (newKeyPart.Expr == nil) {
			// if the key part uses expression instead of column name, the expression node is nil.
			return false
		}
		if oldKeyPart.Expr != nil && newKeyPart.Expr != nil {
			oldKeyPartStr, err := toString(oldKeyPart.Expr)
			if err != nil {
				slog.Error("failed to convert old key part to string", log.BBError(err))
				return false
			}
			newKeyPartStr, err := toString(newKeyPart.Expr)
			if err != nil {
				slog.Error("failed to convert new key part to string", log.BBError(err))
				return false
			}
			return oldKeyPartStr == newKeyPartStr
		}
		if oldKeyPart.Desc != newKeyPart.Desc {
			return false
		}
		// TODO(zp): TiDB MySQL parser doesn't record the index order field in go struct, but it can parse correctly.
		// https://sourcegraph.com/github.com/pingcap/tidb/-/blob/parser/parser.y?L3688
		// We can support the index order until we implement the parser by ourself or https://github.com/pingcap/tidb/pull/38137 is merged.
	}
	return true
}

// isIndexOptionEqual returns true if two index options are the same.
func isIndexOptionEqual(o, n *ast.IndexOption) bool {
	// index_option: {
	// 	KEY_BLOCK_SIZE [=] value
	//   | index_type
	//   | WITH PARSER parser_name
	//   | COMMENT 'string'
	//   | {VISIBLE | INVISIBLE}
	//   |ENGINE_ATTRIBUTE [=] 'string'
	//   |SECONDARY_ENGINE_ATTRIBUTE [=] 'string'
	// }
	if (o == nil) != (n == nil) {
		return false
	}
	if o == nil && n == nil {
		return true
	}
	if o.KeyBlockSize != n.KeyBlockSize {
		return false
	}
	if o.Tp != n.Tp {
		return false
	}
	if o.ParserName.O != n.ParserName.O {
		return false
	}
	if o.Comment != n.Comment {
		return false
	}
	if o.Visibility != n.Visibility {
		return false
	}
	// TODO(zp): support ENGINE_ATTRIBUTE and SECONDARY_ENGINE_ATTRIBUTE.
	return true
}

// isForeignKeyEqual returns true if two foreign keys are the same.
func isForeignKeyConstraintEqual(o, n *ast.Constraint) bool {
	// FOREIGN KEY [index_name] (index_col_name,...) reference_definition
	if !strings.EqualFold(o.Name, n.Name) {
		return false
	}
	if !isKeyPartEqual(o.Keys, n.Keys) {
		return false
	}
	if !isReferenceDefinitionEqual(o.Refer, n.Refer) {
		return false
	}
	return true
}

// isReferenceDefinitionEqual returns true if two reference definitions are the same.
func isReferenceDefinitionEqual(o, n *ast.ReferenceDef) bool {
	// reference_definition:
	// 	REFERENCES tbl_name (index_col_name,...)
	//   [MATCH FULL | MATCH PARTIAL | MATCH SIMPLE]
	//   [ON DELETE reference_option]
	//   [ON UPDATE reference_option]
	if o.Table.Name.String() != n.Table.Name.String() {
		return false
	}

	if len(o.IndexPartSpecifications) != len(n.IndexPartSpecifications) {
		return false
	}
	for idx, oldIndexColName := range o.IndexPartSpecifications {
		newIndexColName := n.IndexPartSpecifications[idx]
		if oldIndexColName.Column.Name.String() != newIndexColName.Column.Name.String() {
			return false
		}
	}

	if o.Match != n.Match {
		return false
	}

	if (o.OnDelete == nil) != (n.OnDelete == nil) {
		return false
	}
	if o.OnDelete != nil && n.OnDelete != nil && o.OnDelete.ReferOpt != n.OnDelete.ReferOpt {
		return false
	}

	if (o.OnUpdate == nil) != (n.OnUpdate == nil) {
		return false
	}
	if o.OnUpdate != nil && n.OnUpdate != nil && o.OnUpdate.ReferOpt != n.OnUpdate.ReferOpt {
		return false
	}
	return true
}

// trimParentheses trims outer parentheses.
func trimParentheses(expr ast.ExprNode) ast.ExprNode {
	result := expr
	for {
		if node, yes := result.(*ast.ParenthesesExpr); yes {
			result = node.Expr
		} else {
			break
		}
	}
	return result
}

// isCheckConstraintEqual returns true if two check constraints are the same.
func isCheckConstraintEqual(o, n *ast.Constraint) bool {
	// check_constraint_definition:
	// 		[CONSTRAINT [symbol]] CHECK (expr) [[NOT] ENFORCED]
	if !strings.EqualFold(o.Name, n.Name) {
		return false
	}

	if o.Enforced != n.Enforced {
		return false
	}

	oldExpr, err := toString(trimParentheses(o.Expr))
	if err != nil {
		slog.Error("failed to convert old check constraint expression to string", log.BBError(err))
		return false
	}
	newExpr, err := toString(trimParentheses(n.Expr))
	if err != nil {
		slog.Error("failed to convert new check constraint expression to string", log.BBError(err))
		return false
	}
	return oldExpr == newExpr
}

// diffTableOptions returns the diff of two table options, returns nil if they are the same.
func diffTableOptions(tableName *ast.TableName, o, n []*ast.TableOption) *ast.AlterTableStmt {
	// https://dev.mysql.com/doc/refman/8.0/en/create-table.html
	// table_option: {
	// 	AUTOEXTEND_SIZE [=] value
	//   | AUTO_INCREMENT [=] value
	//   | AVG_ROW_LENGTH [=] value
	//   | [DEFAULT] CHARACTER SET [=] charset_name
	//   | CHECKSUM [=] {0 | 1}
	//   | [DEFAULT] COLLATE [=] collation_name
	//   | COMMENT [=] 'string'
	//   | COMPRESSION [=] {'ZLIB' | 'LZ4' | 'NONE'}
	//   | CONNECTION [=] 'connect_string'
	//   | {DATA | INDEX} DIRECTORY [=] 'absolute path to directory'
	//   | DELAY_KEY_WRITE [=] {0 | 1}
	//   | ENCRYPTION [=] {'Y' | 'N'}
	//   | ENGINE [=] engine_name
	//   | ENGINE_ATTRIBUTE [=] 'string'
	//   | INSERT_METHOD [=] { NO | FIRST | LAST }
	//   | KEY_BLOCK_SIZE [=] value
	//   | MAX_ROWS [=] value
	//   | MIN_ROWS [=] value
	//   | PACK_KEYS [=] {0 | 1 | DEFAULT}
	//   | PASSWORD [=] 'string'
	//   | ROW_FORMAT [=] {DEFAULT | DYNAMIC | FIXED | COMPRESSED | REDUNDANT | COMPACT}
	//   | START TRANSACTION	// This is an internal-use table option.
	//							// It was introduced in MySQL 8.0.21 to permit CREATE TABLE ... SELECT to be logged as a single,
	//							// atomic transaction in the binary log when using row-based replication with a storage engine that supports atomic DDL.
	//   | SECONDARY_ENGINE_ATTRIBUTE [=] 'string'
	//   | STATS_AUTO_RECALC [=] {DEFAULT | 0 | 1}
	//   | STATS_PERSISTENT [=] {DEFAULT | 0 | 1}
	//   | STATS_SAMPLE_PAGES [=] value
	//   | TABLESPACE tablespace_name [STORAGE {DISK | MEMORY}]
	//   | UNION [=] (tbl_name[,tbl_name]...)
	// }

	// We use map to record the table options, so we can easily find the difference.
	oldOptionsMap := buildTableOptionMap(o)
	newOptionsMap := buildTableOptionMap(n)

	var options []*ast.TableOption
	for oldTp, oldOption := range oldOptionsMap {
		newOption, ok := newOptionsMap[oldTp]
		if !ok {
			switch oldOption.Tp {
			// For table engine, table charset and table collation, if oldTable has but newTable doesn't,
			// we skip drop them.
			case ast.TableOptionEngine, ast.TableOptionCharset, ast.TableOptionCollate:
				continue
			}
			// We should drop the table option if it doesn't exist in the new table options.
			if astOption := dropTableOption(oldOption); astOption != nil {
				options = append(options, astOption)
			}
			continue
		}
		if !isTableOptionValEqual(oldOption, newOption) {
			options = append(options, newOption)
		}
	}
	// We should add the table option if it doesn't exist in the old table options.
	for newTp, newOption := range newOptionsMap {
		if _, ok := oldOptionsMap[newTp]; !ok {
			// We should add the table option if it doesn't exist in the old table options.
			options = append(options, newOption)
		}
	}
	if len(options) == 0 {
		return nil
	}
	return &ast.AlterTableStmt{
		Table: tableName,
		Specs: []*ast.AlterTableSpec{
			{
				Tp:      ast.AlterTableOption,
				Options: options,
			},
		},
	}
}

// dropTableOption generate the table options node need to oppended to the ALTER TABLE OPTION spec.
func dropTableOption(option *ast.TableOption) *ast.TableOption {
	switch option.Tp {
	case ast.TableOptionAutoIncrement:
		// You cannot reset the counter to a value less than or equal to the value that is currently in use.
		// For both InnoDB and MyISAM, if the value is less than or equal to the maximum value currently in the AUTO_INCREMENT column,
		// the value is reset to the current maximum AUTO_INCREMENT column value plus one.
		// https://dev.mysql.com/doc/refman/8.0/en/alter-table.html
		// So we always set the auto_increment value to 0, it will be reset to the current maximum AUTO_INCREMENT column value plus one.
		return &ast.TableOption{
			Tp:        ast.TableOptionAutoIncrement,
			UintValue: 0,
		}
	case ast.TableOptionAvgRowLength:
		// AVG_ROW_LENGTH only works in MyISAM tables.
		return &ast.TableOption{
			Tp:        ast.TableOptionAvgRowLength,
			UintValue: 0,
		}
	case ast.TableOptionCharset:
		// TODO(zp): we use utf8mb4 as the default charset, but it's not always true. We should consider the database default charset.
		defaultCharset := "utf8mb4"
		if option.StrValue == defaultCharset {
			return nil
		}
		return &ast.TableOption{
			Tp:       ast.TableOptionCharset,
			StrValue: defaultCharset,
		}
	case ast.TableOptionCollate:
		// TODO(zp): default collate is related with the charset.
		defaultCollate := "utf8mb4_general_ci"
		if option.StrValue == defaultCollate {
			return nil
		}
		return &ast.TableOption{
			Tp:       ast.TableOptionCollate,
			StrValue: defaultCollate,
		}
	case ast.TableOptionCheckSum:
		return &ast.TableOption{
			Tp:        ast.TableOptionCheckSum,
			UintValue: 0,
		}
	case ast.TableOptionComment:
		// Set to "" to remove the comment.
		return &ast.TableOption{
			Tp: ast.TableOptionComment,
		}
	case ast.TableOptionCompression:
		// TODO(zp): handle the compression
	case ast.TableOptionConnection:
		// Set to "" to remove the connection.
		return &ast.TableOption{
			Tp: ast.TableOptionConnection,
		}
	case ast.TableOptionDataDirectory:
	case ast.TableOptionIndexDirectory:
		// TODO(zp): handle the default data directory and index directory, there are a lot of situations we need to consider.
		// 1. Data Directory and Index Directory will be ignored on Windows.
		// 2. For a normal table, data directory and index directory cannot changed after the table is created, but for a partition table, it can be changed by USING `alter add PARTITIONS data DIRECTORY ...`
		// 3. We should know the default data directory like /var/mysql/data, and the default index directory like /var/mysql/index.
	case ast.TableOptionDelayKeyWrite:
		return &ast.TableOption{
			Tp:        ast.TableOptionDelayKeyWrite,
			UintValue: 0,
		}
	case ast.TableOptionEncryption:
		return &ast.TableOption{
			Tp:       ast.TableOptionEncryption,
			StrValue: "N",
		}
	case ast.TableOptionEngine:
		// TODO(zp): handle the default engine
	case ast.TableOptionInsertMethod:
		// INSERT METHOD only support in MERGE storage engine.
		// https://dev.mysql.com/doc/refman/8.0/en/create-table.html
		return &ast.TableOption{
			Tp:       ast.TableOptionInsertMethod,
			StrValue: "NO",
		}
	case ast.TableOptionKeyBlockSize:
		// TODO(zp): InnoDB doesn't support this. And the default value will be ensured when compiling.
	case ast.TableOptionMaxRows:
		return &ast.TableOption{
			Tp:        ast.TableOptionMaxRows,
			UintValue: 0,
		}
	case ast.TableOptionMinRows:
		return &ast.TableOption{
			Tp:        ast.TableOptionMinRows,
			UintValue: 0,
		}
	case ast.TableOptionPackKeys:
		// TiDB doesn't support this, and the restore will always write "DEFAULT".
		return &ast.TableOption{
			Tp: ast.TableOptionPackKeys,
		}
	case ast.TableOptionPassword:
		// mysqldump will not dump the password, but we handle it to pass the test.
		return &ast.TableOption{
			Tp: ast.TableOptionPassword,
		}
	case ast.TableOptionRowFormat:
		return &ast.TableOption{
			Tp:        ast.TableOptionRowFormat,
			UintValue: ast.RowFormatDefault,
		}
	case ast.TableOptionStatsAutoRecalc:
		return &ast.TableOption{
			Tp:      ast.TableOptionStatsAutoRecalc,
			Default: true,
		}
	case ast.TableOptionStatsPersistent:
		// TiDB doesn't support this, and the restore will always write "DEFAULT".
		return &ast.TableOption{
			Tp: ast.TableOptionStatsPersistent,
		}
	case ast.TableOptionStatsSamplePages:
		return &ast.TableOption{
			Tp:      ast.TableOptionStatsSamplePages,
			Default: true,
		}
	case ast.TableOptionTablespace:
		// TODO(zp): handle the table space
	case ast.TableOptionUnion:
		// TODO(zp): handle the union
	}
	return nil
}

// isTableOptionValEqual compare the two table options value, if they are equal, returns true.
// Caller need to ensure the two table options are not nil and the type is the same.
func isTableOptionValEqual(o, n *ast.TableOption) bool {
	switch o.Tp {
	case ast.TableOptionAutoIncrement:
		// You cannot reset the counter to a value less than or equal to the value that is currently in use.
		// For both InnoDB and MyISAM, if the value is less than or equal to the maximum value currently in the AUTO_INCREMENT column,
		// the value is reset to the current maximum AUTO_INCREMENT column value plus one.
		// https://dev.mysql.com/doc/refman/8.0/en/alter-table.html
		return o.UintValue == n.UintValue
	case ast.TableOptionAvgRowLength:
		return o.UintValue == n.UintValue
	case ast.TableOptionCharset:
		if o.Default != n.Default {
			return false
		}
		if o.Default && n.Default {
			return true
		}
		return o.StrValue == n.StrValue
	case ast.TableOptionCollate:
		return o.StrValue == n.StrValue
	case ast.TableOptionCheckSum:
		return o.UintValue == n.UintValue
	case ast.TableOptionComment:
		return o.StrValue == n.StrValue
	case ast.TableOptionCompression:
		return o.StrValue == n.StrValue
	case ast.TableOptionConnection:
		return o.StrValue == n.StrValue
	case ast.TableOptionDataDirectory:
		return o.StrValue == n.StrValue
	case ast.TableOptionIndexDirectory:
		return o.StrValue == n.StrValue
	case ast.TableOptionDelayKeyWrite:
		return o.UintValue == n.UintValue
	case ast.TableOptionEncryption:
		return o.StrValue == n.StrValue
	case ast.TableOptionEngine:
		return o.StrValue == n.StrValue
	case ast.TableOptionInsertMethod:
		return o.StrValue == n.StrValue
	case ast.TableOptionKeyBlockSize:
		return o.UintValue == n.UintValue
	case ast.TableOptionMaxRows:
		return o.UintValue == n.UintValue
	case ast.TableOptionMinRows:
		return o.UintValue == n.UintValue
	case ast.TableOptionPackKeys:
		// TiDB doesn't support this, and the restore will always write "DEFAULT".
		// The parser will ignore it. So we can only ignore it here.
		// https://github.com/pingcap/tidb/blob/master/parser/parser.y#L11661
		return o.Default == n.Default
	case ast.TableOptionPassword:
		// mysqldump will not dump the password, so we just return false here.
		return false
	case ast.TableOptionRowFormat:
		return o.UintValue == n.UintValue
	case ast.TableOptionStatsAutoRecalc:
		// TiDB parser will ignore the DEFAULT value. So we just can compare the UINT value here.
		// https://github.com/pingcap/tidb/blob/master/parser/parser.y#L11599
		return o.UintValue == n.UintValue
	case ast.TableOptionStatsPersistent:
		// TiDB parser doesn't support this, it only assign the type without any value.
		// https://github.com/pingcap/tidb/blob/master/parser/parser.y#L11595
		return true
	case ast.TableOptionStatsSamplePages:
		return o.UintValue == n.UintValue
	case ast.TableOptionTablespace:
		return o.StrValue == n.StrValue
	case ast.TableOptionUnion:
		oldTableNames := o.TableNames
		newTableNames := n.TableNames
		if len(oldTableNames) != len(newTableNames) {
			return false
		}
		for i, oldTableName := range oldTableNames {
			newTableName := newTableNames[i]
			if oldTableName.Name.O != newTableName.Name.O {
				return false
			}
		}
		return true
	}
	return true
}

// isViewEqual checks whether two views with same name are equal.
func (diff *diffNode) isViewEqual(o, n *ast.CreateViewStmt) bool {
	// CREATE
	// 		[OR REPLACE]
	// 		[ALGORITHM = {UNDEFINED | MERGE | TEMPTABLE}]
	// 		[DEFINER = user]
	// 		[SQL SECURITY { DEFINER | INVOKER }]
	// 		VIEW view_name [(column_list)]
	// 		AS select_statement
	// 		[WITH [CASCADED | LOCAL] CHECK OPTION]
	// We can easily replace view statement by using `CREATE OR REPLACE VIEW` statement to replace the old one.
	// So we don't need to compare each part, just compare the restore string.
	if diff.ignoreCaseSensitive {
		oldViewStr, err := toLowerNameString(o)
		if err != nil {
			slog.Error("fail to convert old view to string", log.BBError(err))
			return false
		}
		newViewStr, err := toLowerNameString(n)
		if err != nil {
			slog.Error("fail to convert new view to string", log.BBError(err))
			return false
		}
		return oldViewStr == newViewStr
	}
	oldViewStr, err := toString(o)
	if err != nil {
		slog.Error("fail to convert old view to string", log.BBError(err))
		return false
	}
	newViewStr, err := toString(n)
	if err != nil {
		slog.Error("fail to convert new view to string", log.BBError(err))
		return false
	}
	return oldViewStr == newViewStr
}

// buildTableOptionMap builds a map of table options.
func buildTableOptionMap(options []*ast.TableOption) map[ast.TableOptionType]*ast.TableOption {
	m := make(map[ast.TableOptionType]*ast.TableOption)
	for _, option := range options {
		m[option.Tp] = option
	}
	return m
}

// hasColumnsIntersection returns true if two column slices have column name intersection.
func hasColumnsIntersection(a, b []*ast.ColumnDef) bool {
	bMap := make(map[string]bool)
	for _, col := range b {
		// MySQL column name is case insensitive.
		bMap[col.Name.Name.L] = true
	}
	for _, col := range a {
		// MySQL column name is case insensitive.
		if _, ok := bMap[col.Name.Name.L]; ok {
			return true
		}
	}
	return false
}
