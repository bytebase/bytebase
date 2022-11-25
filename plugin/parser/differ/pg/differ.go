// Package pg provides the PostgreSQL differ plugin.
package pg

import (
	"bytes"
	"io"
	"sort"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
	"github.com/bytebase/bytebase/plugin/parser/differ"
)

var (
	_ differ.SchemaDiffer = (*SchemaDiffer)(nil)
)

func init() {
	differ.Register(parser.Postgres, &SchemaDiffer{})
}

// SchemaDiffer it the parser for PostgreSQL dialect.
type SchemaDiffer struct {
}

// diffNode defines different modification types as the safe change order.
// The safe change order means we can change them with no dependency conflicts as this order.
type diffNode struct {
	// Drop nodes
	dropForeignKeyList         []ast.Node
	dropConstraintExceptFkList []ast.Node
	dropIndexList              []ast.Node
	dropColumnList             []ast.Node
	dropTableList              []ast.Node
	dropSchemaList             []ast.Node

	// Create nodes
	createSchemaList             []ast.Node
	createTableList              []ast.Node
	createColumnList             []ast.Node
	alterColumnList              []ast.Node
	createIndexList              []ast.Node
	createConstraintExceptFkList []ast.Node
	createForeignKeyList         []ast.Node
}

type schemaMap map[string]*schemaInfo
type tableMap map[string]*tableInfo
type constraintMap map[string]*constraintInfo
type indexMap map[string]*indexInfo

type schemaInfo struct {
	id           int
	existsInNew  bool
	createSchema *ast.CreateSchemaStmt
	tableMap     tableMap
	indexMap     indexMap
}

func newSchemaInfo(id int, createSchema *ast.CreateSchemaStmt) *schemaInfo {
	return &schemaInfo{
		id:           id,
		existsInNew:  false,
		createSchema: createSchema,
		tableMap:     make(tableMap),
		indexMap:     make(indexMap),
	}
}

type tableInfo struct {
	id            int
	existsInNew   bool
	createTable   *ast.CreateTableStmt
	constraintMap constraintMap
}

func newTableInfo(id int, createTable *ast.CreateTableStmt) *tableInfo {
	return &tableInfo{
		id:            id,
		existsInNew:   false,
		createTable:   createTable,
		constraintMap: make(constraintMap),
	}
}

type constraintInfo struct {
	id            int
	existsInNew   bool
	addConstraint *ast.AddConstraintStmt
}

func newConstraintInfo(id int, addConstraint *ast.AddConstraintStmt) *constraintInfo {
	return &constraintInfo{
		id:            id,
		existsInNew:   false,
		addConstraint: addConstraint,
	}
}

type indexInfo struct {
	id          int
	existsInNew bool
	createIndex *ast.CreateIndexStmt
}

func newIndexInfo(id int, createIndex *ast.CreateIndexStmt) *indexInfo {
	return &indexInfo{
		id:          id,
		existsInNew: false,
		createIndex: createIndex,
	}
}

func (m schemaMap) addTable(id int, table *ast.CreateTableStmt) error {
	schema, exists := m[table.Name.Schema]
	if !exists {
		return errors.Errorf("failed to add table: schema %s not found", table.Name.Schema)
	}
	schema.tableMap[table.Name.Name] = newTableInfo(id, table)
	return nil
}

func (m schemaMap) getTable(schemaName string, tableName string) *tableInfo {
	schema, exists := m[schemaName]
	if !exists {
		return nil
	}
	return schema.tableMap[tableName]
}

func (m schemaMap) addConstraint(id int, addConstraint *ast.AddConstraintStmt) error {
	schema, exists := m[addConstraint.Table.Schema]
	if !exists {
		return errors.Errorf("failed to add constraint: schema %s not found", addConstraint.Table.Schema)
	}
	table, exists := schema.tableMap[addConstraint.Table.Name]
	if !exists {
		return errors.Errorf("failed to add constraint: table %s not found", addConstraint.Table.Name)
	}
	constraintName := addConstraint.Constraint.Name
	if constraintName == "" {
		return errors.Errorf("failed to add constraint: constraint name is empty")
	}
	table.constraintMap[constraintName] = newConstraintInfo(id, addConstraint)
	return nil
}

func (m schemaMap) getConstraint(schemaName string, tableName string, constraintName string) *constraintInfo {
	schema, exists := m[schemaName]
	if !exists {
		return nil
	}
	table, exists := schema.tableMap[tableName]
	if !exists {
		return nil
	}
	return table.constraintMap[constraintName]
}

func (m schemaMap) addIndex(id int, index *ast.CreateIndexStmt) error {
	schema, exists := m[index.Index.Table.Schema]
	if !exists {
		return errors.Errorf("failed to add table: schema %s not found", index.Index.Table.Schema)
	}
	schema.indexMap[index.Index.Name] = newIndexInfo(id, index)
	return nil
}

func (m schemaMap) getIndex(schemaName string, indexName string) *indexInfo {
	schema, exists := m[schemaName]
	if !exists {
		return nil
	}
	return schema.indexMap[indexName]
}

// mergeStatements merges some statements into one statement, for example, merge
// `CREATE TABLE tbl(id INT DEFAULT '3');`,
// `ALTER TABLE ONLY tbl ALTER COLUMN id nextval('tbl_id_seq'::regclass);`
// to `CREATE TABLE tbl(id INT DEFAULT nextval('tbl_id_seq'::regclass));`.
func mergeStatements(statements string) ([]ast.Node, error) {
	nodes, err := parser.Parse(parser.Postgres, parser.ParseContext{}, statements)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse statements %q", statements)
	}
	var retNodes []ast.Node

	schemaTableNameToRetNodesIdx := make(map[string]int)
	for _, node := range nodes {
		switch node := node.(type) {
		case *ast.CreateTableStmt:
			schemaTableName := node.Name.Schema + "." + node.Name.Name
			retNodes = append(retNodes, node)
			schemaTableNameToRetNodesIdx[schemaTableName] = len(retNodes) - 1
		case *ast.AlterTableStmt:
			schemaTableName := node.Table.Schema + "." + node.Table.Name
			retNodesIdx, ok := schemaTableNameToRetNodesIdx[schemaTableName]
			if !ok {
				return nil, errors.Errorf("cannot find table %s when merging statements %q", schemaTableName, statements)
			}
			for _, alterItem := range node.AlterItemList {
				switch item := alterItem.(type) {
				case *ast.SetDefaultStmt:
					columnFind := false
					for _, columnDef := range retNodes[retNodesIdx].(*ast.CreateTableStmt).ColumnList {
						if columnDef.ColumnName == item.ColumnName {
							columnFind = true
							constraintDef := &ast.ConstraintDef{
								Type:       ast.ConstraintTypeDefault,
								Expression: item.Expression,
							}
							defaultConstraintFind := false
							for idx, constraint := range columnDef.ConstraintList {
								if constraint.Type == ast.ConstraintTypeDefault {
									columnDef.ConstraintList[idx] = constraintDef
									defaultConstraintFind = true
									break
								}
							}
							if !defaultConstraintFind {
								columnDef.ConstraintList = append(columnDef.ConstraintList, constraintDef)
							}
							break
						}
					}
					if !columnFind {
						return nil, errors.Errorf("cannot find column %s when merging statements %q", item.ColumnName, statements)
					}
				default:
					retNodes = append(retNodes, node)
				}
			}
		default:
			retNodes = append(retNodes, node)
		}
	}
	return retNodes, nil
}

// SchemaDiff computes the schema differences between old and new schema.
func (*SchemaDiffer) SchemaDiff(oldStmt, newStmt string) (string, error) {
	oldNodes, err := mergeStatements(oldStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to merge old statements %q", oldStmt)
	}
	newNodes, err := mergeStatements(newStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to merge new statements %q", newStmt)
	}
	oldSchemaMap := make(schemaMap)
	oldSchemaMap["public"] = newSchemaInfo(-1, &ast.CreateSchemaStmt{Name: "public"})
	oldSchemaMap["public"].existsInNew = true
	for i, node := range oldNodes {
		switch stmt := node.(type) {
		case *ast.CreateSchemaStmt:
			oldSchemaMap[stmt.Name] = newSchemaInfo(i, stmt)
		case *ast.CreateTableStmt:
			if err := oldSchemaMap.addTable(i, stmt); err != nil {
				return "", err
			}
		case *ast.AlterTableStmt:
			for _, alterItem := range stmt.AlterItemList {
				switch item := alterItem.(type) {
				case *ast.AddConstraintStmt:
					switch item.Constraint.Type {
					case ast.ConstraintTypeUnique, ast.ConstraintTypePrimary, ast.ConstraintTypeExclusion, ast.ConstraintTypeForeign:
						if err := oldSchemaMap.addConstraint(i, item); err != nil {
							return "", err
						}
					default:
						return "", errors.Errorf("unsupported constraint type %d", item.Constraint.Type)
					}
				default:
					return "", errors.Errorf("unsupported alter table item type %T", item)
				}
			}
		case *ast.CreateIndexStmt:
			if err := oldSchemaMap.addIndex(i, stmt); err != nil {
				return "", err
			}
		}
	}

	diff := &diffNode{}
	for _, node := range newNodes {
		switch stmt := node.(type) {
		case *ast.CreateTableStmt:
			oldTable := oldSchemaMap.getTable(stmt.Name.Schema, stmt.Name.Name)
			// Add the new table.
			if oldTable == nil {
				diff.createTableList = append(diff.createTableList, stmt)
				continue
			}
			oldTable.existsInNew = true
			// Modify the table.
			if err := diff.modifyTable(oldTable.createTable, stmt); err != nil {
				return "", err
			}
		case *ast.CreateSchemaStmt:
			schema, hasSchema := oldSchemaMap[stmt.Name]
			if !hasSchema {
				diff.createSchemaList = append(diff.createSchemaList, stmt)
				continue
			}
			schema.existsInNew = true
		case *ast.AlterTableStmt:
			for _, alterItem := range stmt.AlterItemList {
				switch item := alterItem.(type) {
				case *ast.AddConstraintStmt:
					switch item.Constraint.Type {
					case ast.ConstraintTypeUnique, ast.ConstraintTypePrimary, ast.ConstraintTypeExclusion, ast.ConstraintTypeForeign:
						oldConstraint := oldSchemaMap.getConstraint(item.Table.Schema, item.Table.Name, item.Constraint.Name)
						if oldConstraint == nil {
							diff.appendAddConstraint(stmt.Table, []*ast.ConstraintDef{item.Constraint})
							continue
						}

						oldConstraint.existsInNew = true
						nameEqual := oldConstraint.addConstraint.Table.Name == item.Table.Name
						equal, err := isEqualConstraint(oldConstraint.addConstraint.Constraint, item.Constraint)
						if err != nil {
							return "", err
						}
						if !nameEqual || !equal {
							diff.appendDropConstraint(stmt.Table, []*ast.ConstraintDef{oldConstraint.addConstraint.Constraint})
							diff.appendAddConstraint(stmt.Table, []*ast.ConstraintDef{item.Constraint})
							continue
						}
					default:
						return "", errors.Errorf("unsupported constraint type %d", item.Constraint.Type)
					}
				default:
					return "", errors.Errorf("unsupported alter table item type %T", item)
				}
			}
		case *ast.CreateIndexStmt:
			oldIndex := oldSchemaMap.getIndex(stmt.Index.Table.Schema, stmt.Index.Name)
			// Add the new index.
			if oldIndex == nil {
				diff.createIndexList = append(diff.createIndexList, stmt)
				continue
			}
			oldIndex.existsInNew = true
			// Modify the index.
			if err := diff.modifyIndex(oldIndex.createIndex, stmt); err != nil {
				return "", err
			}
		default:
			return "", errors.Errorf("unsupported statement %+v", stmt)
		}
	}

	// Drop remaining old objects.
	if err := diff.dropObject(oldSchemaMap); err != nil {
		return "", err
	}

	return diff.deparse()
}

func (diff *diffNode) appendDropConstraint(table *ast.TableDef, constraintList []*ast.ConstraintDef) {
	dropConstraintExceptFkStmt := &ast.AlterTableStmt{Table: table}
	dropForeignStmt := &ast.AlterTableStmt{Table: table}

	for _, constraint := range constraintList {
		if constraint == nil {
			continue
		}

		dropStmt := &ast.DropConstraintStmt{
			Table:          table,
			ConstraintName: constraint.Name,
		}

		if constraint.Type == ast.ConstraintTypeForeign {
			dropForeignStmt.AlterItemList = append(dropForeignStmt.AlterItemList, dropStmt)
		} else {
			dropConstraintExceptFkStmt.AlterItemList = append(dropConstraintExceptFkStmt.AlterItemList, dropStmt)
		}
	}

	if len(dropConstraintExceptFkStmt.AlterItemList) > 0 {
		diff.dropForeignKeyList = append(diff.dropForeignKeyList, dropConstraintExceptFkStmt)
	}
	if len(dropForeignStmt.AlterItemList) > 0 {
		diff.dropConstraintExceptFkList = append(diff.dropConstraintExceptFkList, dropForeignStmt)
	}
}

func (diff *diffNode) appendAddConstraint(table *ast.TableDef, constraintList []*ast.ConstraintDef) {
	addConstraintExceptFkStmt := &ast.AlterTableStmt{Table: table}
	addForeignStmt := &ast.AlterTableStmt{Table: table}

	for _, constraint := range constraintList {
		if constraint == nil {
			continue
		}

		addStmt := &ast.AddConstraintStmt{
			Table:      table,
			Constraint: constraint,
		}

		if constraint.Type == ast.ConstraintTypeForeign {
			addForeignStmt.AlterItemList = append(addForeignStmt.AlterItemList, addStmt)
		} else {
			addConstraintExceptFkStmt.AlterItemList = append(addConstraintExceptFkStmt.AlterItemList, addStmt)
		}
	}

	if len(addConstraintExceptFkStmt.AlterItemList) > 0 {
		diff.createConstraintExceptFkList = append(diff.createConstraintExceptFkList, addConstraintExceptFkStmt)
	}
	if len(addForeignStmt.AlterItemList) > 0 {
		diff.createForeignKeyList = append(diff.createForeignKeyList, addForeignStmt)
	}
}

func (diff *diffNode) dropObject(oldSchemaMap schemaMap) error {
	// Drop the remaining old schema.
	if dropSchemaStmt := dropSchema(oldSchemaMap); dropSchemaStmt != nil {
		diff.dropSchemaList = append(diff.dropSchemaList, dropSchemaStmt)
	}

	// Drop the remaining old table.
	if dropTableStmt := dropTable(oldSchemaMap); dropTableStmt != nil {
		diff.dropTableList = append(diff.dropTableList, dropTableStmt)
	}

	// Drop the remaining old constraints.
	diff.dropConstraint(oldSchemaMap)

	// Drop the remaining old index.
	if dropIndexStmt := dropIndex(oldSchemaMap); dropIndexStmt != nil {
		diff.dropIndexList = append(diff.dropIndexList, dropIndexStmt)
	}

	return nil
}

func (diff *diffNode) modifyTableByColumn(oldTable *ast.CreateTableStmt, newTable *ast.CreateTableStmt) error {
	tableName := oldTable.Name
	addColumn := &ast.AlterTableStmt{
		Table: tableName,
	}
	dropColumn := &ast.AlterTableStmt{
		Table: tableName,
	}

	oldColumnMap := make(map[string]*ast.ColumnDef)
	for _, column := range oldTable.ColumnList {
		oldColumnMap[column.ColumnName] = column
	}

	for _, newColumn := range newTable.ColumnList {
		oldColumn, exists := oldColumnMap[newColumn.ColumnName]
		// Add the new column.
		if !exists {
			addColumn.AlterItemList = append(addColumn.AlterItemList, &ast.AddColumnListStmt{
				Table:      tableName,
				ColumnList: []*ast.ColumnDef{newColumn},
			})
			continue
		}
		// Modify the column.
		if err := diff.modifyColumn(tableName, oldColumn, newColumn); err != nil {
			return err
		}
		delete(oldColumnMap, oldColumn.ColumnName)
	}

	for _, oldColumn := range oldTable.ColumnList {
		if _, exists := oldColumnMap[oldColumn.ColumnName]; exists {
			dropColumn.AlterItemList = append(dropColumn.AlterItemList, &ast.DropColumnStmt{
				Table:      tableName,
				ColumnName: oldColumn.ColumnName,
			})
		}
	}

	if len(addColumn.AlterItemList) > 0 {
		diff.createColumnList = append(diff.createColumnList, addColumn)
	}
	if len(dropColumn.AlterItemList) > 0 {
		diff.dropColumnList = append(diff.dropColumnList, dropColumn)
	}
	return nil
}

func (diff *diffNode) modifyTableByConstraint(oldTable *ast.CreateTableStmt, newTable *ast.CreateTableStmt) error {
	tableName := oldTable.Name
	var addConstraintList []*ast.ConstraintDef
	var dropConstraintList []*ast.ConstraintDef

	// Modify table for constraints.
	oldConstraintMap := make(map[string]*ast.ConstraintDef)
	for _, constraint := range oldTable.ConstraintList {
		oldConstraintMap[constraint.Name] = constraint
	}
	for _, newConstraint := range newTable.ConstraintList {
		oldConstraint, exists := oldConstraintMap[newConstraint.Name]
		// Add the new constraint.
		if !exists {
			addConstraintList = append(addConstraintList, newConstraint)
			continue
		}

		isEqual, err := isEqualConstraint(oldConstraint, newConstraint)
		if err != nil {
			return err
		}
		if !isEqual {
			dropConstraintList = append(dropConstraintList, oldConstraint)
			addConstraintList = append(addConstraintList, newConstraint)
		}
		delete(oldConstraintMap, oldConstraint.Name)
	}

	for _, oldConstraint := range oldTable.ConstraintList {
		if _, exists := oldConstraintMap[oldConstraint.Name]; exists {
			dropConstraintList = append(dropConstraintList, oldConstraint)
		}
	}

	diff.appendAddConstraint(tableName, addConstraintList)
	diff.appendDropConstraint(tableName, dropConstraintList)
	return nil
}

func (diff *diffNode) modifyTable(oldTable *ast.CreateTableStmt, newTable *ast.CreateTableStmt) error {
	if err := diff.modifyTableByColumn(oldTable, newTable); err != nil {
		return err
	}

	return diff.modifyTableByConstraint(oldTable, newTable)
}

func isEqualConstraint(oldConstraint *ast.ConstraintDef, newConstraint *ast.ConstraintDef) (bool, error) {
	if oldConstraint.Type != newConstraint.Type {
		return false, nil
	}

	switch newConstraint.Type {
	case ast.ConstraintTypeCheck:
		if newConstraint.Expression.Text() != oldConstraint.Expression.Text() {
			return false, nil
		}
	case ast.ConstraintTypeUnique, ast.ConstraintTypePrimary, ast.ConstraintTypeExclusion, ast.ConstraintTypeForeign:
		// TODO(zp): To make the logic simple now, we just restore the statement, and drop and create the new one if
		// there is any difference.
		oldAlterTableAddConstraint, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, oldConstraint)
		if err != nil {
			return false, errors.Wrapf(err, "failed to deparse old alter table constraintDef: %v", oldConstraint)
		}
		newAlterTableAddConstraint, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, newConstraint)
		if err != nil {
			return false, errors.Wrapf(err, "failed to deparse new alter table constraintDef: %v", newConstraint)
		}
		if oldAlterTableAddConstraint != newAlterTableAddConstraint {
			return false, nil
		}
	default:
		return false, errors.Errorf("Unsupported table constraint type: %d for modifyConstraint", newConstraint.Type)
	}
	return true, nil
}

func (diff *diffNode) modifyColumn(tableName *ast.TableDef, oldColumn *ast.ColumnDef, newColumn *ast.ColumnDef) error {
	alterTableStmt := &ast.AlterTableStmt{
		Table: tableName,
	}
	columnName := oldColumn.ColumnName
	// compare the data type
	equivalent, err := equivalentType(oldColumn.Type, newColumn.Type)
	if err != nil {
		return err
	}
	if !equivalent {
		alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.AlterColumnTypeStmt{
			Table:      tableName,
			ColumnName: columnName,
			Type:       newColumn.Type,
		})
	}
	// compare the NOT NULL
	oldNotNull := hasNotNull(oldColumn)
	newNotNull := hasNotNull(newColumn)
	needSetNotNull := !oldNotNull && newNotNull
	needDropNotNull := oldNotNull && !newNotNull
	if needSetNotNull {
		alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.SetNotNullStmt{
			Table:      alterTableStmt.Table,
			ColumnName: columnName,
		})
	} else if needDropNotNull {
		alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.DropNotNullStmt{
			Table:      alterTableStmt.Table,
			ColumnName: columnName,
		})
	}
	// compare the DEFAULT
	oldDefault, oldHasDefault := getDefault(oldColumn)
	newDefault, newHasDefault := getDefault(newColumn)
	needSetDefault := (!oldHasDefault && newHasDefault) || (oldHasDefault && newHasDefault && oldDefault != newDefault)
	needDropDefault := oldHasDefault && !newHasDefault
	if needSetDefault {
		expression := &ast.UnconvertedExpressionDef{}
		expression.SetText(newDefault)
		alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.SetDefaultStmt{
			Table:      alterTableStmt.Table,
			ColumnName: columnName,
			Expression: expression,
		})
	} else if needDropDefault {
		alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.DropDefaultStmt{
			Table:      alterTableStmt.Table,
			ColumnName: columnName,
		})
	}

	// TODO(rebelice): compare other column properties
	if len(alterTableStmt.AlterItemList) > 0 {
		diff.alterColumnList = append(diff.alterColumnList, alterTableStmt)
	}
	return nil
}

func (diff *diffNode) modifyIndex(oldIndex *ast.CreateIndexStmt, newIndex *ast.CreateIndexStmt) error {
	// TODO(rebelice): not use Text(), it only works for pg_dump.
	if oldIndex.Text() != newIndex.Text() {
		diff.dropIndexList = append(diff.dropIndexList, &ast.DropIndexStmt{
			IndexList: []*ast.IndexDef{
				{
					Table: &ast.TableDef{Schema: oldIndex.Index.Table.Schema},
					Name:  oldIndex.Index.Name,
				},
			},
		})
		diff.createIndexList = append(diff.createIndexList, newIndex)
	}
	return nil
}

func getDefault(column *ast.ColumnDef) (string, bool) {
	for _, constraint := range column.ConstraintList {
		if constraint.Type == ast.ConstraintTypeDefault {
			return constraint.Expression.Text(), true
		}
	}
	return "", false
}

func hasNotNull(column *ast.ColumnDef) bool {
	for _, constraint := range column.ConstraintList {
		if constraint.Type == ast.ConstraintTypeNotNull {
			return true
		}
	}
	return false
}

func equivalentType(typeA ast.DataType, typeB ast.DataType) (bool, error) {
	typeStringA, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, typeA)
	if err != nil {
		return false, err
	}
	typeStringB, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, typeB)
	if err != nil {
		return false, err
	}
	return typeStringA == typeStringB, nil
}

func printStmtSlice(buf io.Writer, nodeList []ast.Node) error {
	for _, node := range nodeList {
		sql, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, node)
		if err != nil {
			return err
		}
		if err := writeStringWithNewLine(buf, sql); err != nil {
			return err
		}
	}
	return nil
}

// deparse statements as the safe change order.
func (diff *diffNode) deparse() (string, error) {
	var buf bytes.Buffer

	// drop
	if err := printStmtSlice(&buf, diff.dropForeignKeyList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropConstraintExceptFkList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropIndexList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropColumnList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropTableList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropSchemaList); err != nil {
		return "", err
	}

	// create
	if err := printStmtSlice(&buf, diff.createSchemaList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.createTableList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.createColumnList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.alterColumnList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.createIndexList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.createConstraintExceptFkList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.createForeignKeyList); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (diff *diffNode) dropConstraint(m schemaMap) {
	for _, schemaInfo := range m {
		for _, tableInfo := range schemaInfo.tableMap {
			var constraintInfoList []*constraintInfo
			for _, constraintInfo := range tableInfo.constraintMap {
				if !constraintInfo.existsInNew {
					constraintInfoList = append(constraintInfoList, constraintInfo)
				}
			}
			sort.Slice(constraintInfoList, func(i, j int) bool {
				return constraintInfoList[i].id < constraintInfoList[j].id
			})
			var dropConstraintList []*ast.ConstraintDef
			for _, constraintInfo := range constraintInfoList {
				dropConstraintList = append(dropConstraintList, constraintInfo.addConstraint.Constraint)
			}

			diff.appendDropConstraint(tableInfo.createTable.Name, dropConstraintList)
		}
	}
}

func dropTable(m schemaMap) *ast.DropTableStmt {
	var tableList []*tableInfo
	for _, schema := range m {
		if !schema.existsInNew {
			// dropped by DROP SCHEMA ... CASCADE statements
			continue
		}
		for _, table := range schema.tableMap {
			if table.existsInNew {
				// no need to drop
				continue
			}
			tableList = append(tableList, table)
		}
	}
	if len(tableList) == 0 {
		return nil
	}
	sort.Slice(tableList, func(i, j int) bool {
		return tableList[i].id < tableList[j].id
	})

	var tableDefList []*ast.TableDef
	for _, table := range tableList {
		tableDefList = append(tableDefList, table.createTable.Name)
	}
	return &ast.DropTableStmt{
		IfExists:  true,
		TableList: tableDefList,
		Behavior:  ast.DropBehaviorCascade,
	}
}

func dropSchema(m schemaMap) *ast.DropSchemaStmt {
	var schemaList []*schemaInfo
	for _, schema := range m {
		if schema.createSchema.Name == "public" || schema.existsInNew {
			continue
		}
		schemaList = append(schemaList, schema)
	}
	if len(schemaList) == 0 {
		return nil
	}
	sort.Slice(schemaList, func(i, j int) bool {
		return schemaList[i].id < schemaList[j].id
	})

	var schemaNameList []string
	for _, schema := range schemaList {
		schemaNameList = append(schemaNameList, schema.createSchema.Name)
	}
	return &ast.DropSchemaStmt{
		IfExists:   true,
		SchemaList: schemaNameList,
		Behavior:   ast.DropBehaviorCascade,
	}
}

func dropIndex(m schemaMap) *ast.DropIndexStmt {
	var indexList []*indexInfo
	for _, schema := range m {
		if !schema.existsInNew {
			// dropped by DROP SCHEMA ... CASCADE statements
			continue
		}
		for _, index := range schema.indexMap {
			if index.existsInNew {
				// no need to drop
				continue
			}
			indexList = append(indexList, index)
		}
	}
	if len(indexList) == 0 {
		return nil
	}
	sort.Slice(indexList, func(i, j int) bool {
		return indexList[i].id < indexList[j].id
	})

	var indexDefList []*ast.IndexDef
	for _, index := range indexList {
		indexDefList = append(indexDefList, &ast.IndexDef{
			Table: &ast.TableDef{Schema: index.createIndex.Index.Table.Schema},
			Name:  index.createIndex.Index.Name,
		})
	}
	return &ast.DropIndexStmt{
		IfExists:  true,
		IndexList: indexDefList,
		Behavior:  ast.DropBehaviorCascade,
	}
}

func writeStringWithNewLine(out io.Writer, str string) error {
	if _, err := out.Write([]byte(str)); err != nil {
		return err
	}
	if _, err := out.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}
