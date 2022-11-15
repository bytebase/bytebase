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

type diffNode struct {
	newSchemaList   []*ast.CreateSchemaStmt
	newTableList    []*ast.CreateTableStmt
	modifyTableList []ast.Node
	newIndexList    []*ast.CreateIndexStmt
	modifyIndexList []ast.Node
	dropNodeList    []ast.Node
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

// SchemaDiff computes the schema differences between old and new schema.
func (*SchemaDiffer) SchemaDiff(oldStmt, newStmt string) (string, error) {
	oldNodes, err := parser.Parse(parser.Postgres, parser.ParseContext{}, oldStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse old statement %q", oldStmt)
	}
	newNodes, err := parser.Parse(parser.Postgres, parser.ParseContext{}, newStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse new statement %q", newStmt)
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
					case ast.ConstraintTypeUnique, ast.ConstraintTypePrimary, ast.ConstraintTypeExclusion:
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
				diff.newTableList = append(diff.newTableList, stmt)
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
				diff.newSchemaList = append(diff.newSchemaList, stmt)
				continue
			}
			schema.existsInNew = true
		case *ast.AlterTableStmt:
			alterTableStmt := &ast.AlterTableStmt{
				Table: stmt.Table,
			}
			for _, alterItem := range stmt.AlterItemList {
				switch item := alterItem.(type) {
				case *ast.AddConstraintStmt:
					switch item.Constraint.Type {
					case ast.ConstraintTypeUnique, ast.ConstraintTypePrimary, ast.ConstraintTypeExclusion:
						oldConstraint := oldSchemaMap.getConstraint(item.Table.Schema, item.Table.Name, item.Constraint.Name)
						if oldConstraint == nil {
							alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, item)
							continue
						}

						oldConstraint.existsInNew = true
						if oldConstraint.addConstraint.Table.Name != item.Table.Name {
							alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.DropConstraintStmt{
								Table:    alterTableStmt.Table,
								IfExists: true,
							})
							alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.AddConstraintStmt{
								Table:      alterTableStmt.Table,
								Constraint: item.Constraint,
							})
							continue
						}
						if err := diff.modifyConstraint(alterTableStmt, oldConstraint.addConstraint.Constraint, item.Constraint); err != nil {
							return "", err
						}
					default:
						return "", errors.Errorf("unsupported constraint type %d", item.Constraint.Type)
					}
				default:
					return "", errors.Errorf("unsupported alter table item type %T", item)
				}
			}
			if len(alterTableStmt.AlterItemList) > 0 {
				diff.modifyTableList = append(diff.modifyTableList, alterTableStmt)
			}
		case *ast.CreateIndexStmt:
			oldIndex := oldSchemaMap.getIndex(stmt.Index.Table.Schema, stmt.Index.Name)
			// Add the new index.
			if oldIndex == nil {
				diff.newIndexList = append(diff.newIndexList, stmt)
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

func (diff *diffNode) dropObject(oldSchemaMap schemaMap) error {
	// Drop the remaining old schema.
	if dropSchemaStmt := dropSchema(oldSchemaMap); dropSchemaStmt != nil {
		diff.dropNodeList = append(diff.dropNodeList, dropSchemaStmt)
	}

	// Drop the remaining old table.
	if dropTableStmt := dropTable(oldSchemaMap); dropTableStmt != nil {
		diff.dropNodeList = append(diff.dropNodeList, dropTableStmt)
	}

	// Drop the remaining old constraints.
	dropConstraintStmtList := dropConstraint(oldSchemaMap)
	for _, dropConstraintStmt := range dropConstraintStmtList {
		diff.dropNodeList = append(diff.dropNodeList, dropConstraintStmt)
	}

	// Drop the remaining old index.
	if dropIndexStmt := dropIndex(oldSchemaMap); dropIndexStmt != nil {
		diff.dropNodeList = append(diff.dropNodeList, dropIndexStmt)
	}

	return nil
}

func (diff *diffNode) modifyTable(oldTable *ast.CreateTableStmt, newTable *ast.CreateTableStmt) error {
	tableName := oldTable.Name
	alterTableStmt := &ast.AlterTableStmt{
		Table: tableName,
	}

	// Modify table for columns.
	oldColumnMap := make(map[string]*ast.ColumnDef)
	for _, column := range oldTable.ColumnList {
		oldColumnMap[column.ColumnName] = column
	}

	for _, newColumn := range newTable.ColumnList {
		oldColumn, exists := oldColumnMap[newColumn.ColumnName]
		// Add the new column.
		if !exists {
			alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.AddColumnListStmt{
				Table:      tableName,
				ColumnList: []*ast.ColumnDef{newColumn},
			})
			continue
		}
		// Modify the column.
		if err := diff.modifyColumn(alterTableStmt, oldColumn, newColumn); err != nil {
			return err
		}
		delete(oldColumnMap, oldColumn.ColumnName)
	}

	for _, oldColumn := range oldTable.ColumnList {
		if _, exists := oldColumnMap[oldColumn.ColumnName]; exists {
			alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.DropColumnStmt{
				Table:      alterTableStmt.Table,
				ColumnName: oldColumn.ColumnName,
			})
		}
	}

	// Modify table for constraints.
	oldConstraintMap := make(map[string]*ast.ConstraintDef)
	for _, constraint := range oldTable.ConstraintList {
		oldConstraintMap[constraint.Name] = constraint
	}
	for _, newConstraint := range newTable.ConstraintList {
		oldConstraint, exists := oldConstraintMap[newConstraint.Name]
		// Add the new constraint.
		if !exists {
			alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.AddConstraintStmt{
				Table:      tableName,
				Constraint: newConstraint,
			})
			continue
		}
		if err := diff.modifyConstraint(alterTableStmt, oldConstraint, newConstraint); err != nil {
			return err
		}
		delete(oldConstraintMap, oldConstraint.Name)
	}

	for _, oldConstraint := range oldTable.ConstraintList {
		if _, exists := oldConstraintMap[oldConstraint.Name]; exists {
			alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.DropConstraintStmt{
				Table:          alterTableStmt.Table,
				ConstraintName: oldConstraint.Name,
			})
		}
	}

	if len(alterTableStmt.AlterItemList) > 0 {
		diff.modifyTableList = append(diff.modifyTableList, alterTableStmt)
	}

	return nil
}

func (*diffNode) modifyConstraint(alterTableStmt *ast.AlterTableStmt, oldConstraint *ast.ConstraintDef, newConstraint *ast.ConstraintDef) error {
	constraintName := oldConstraint.Name

	if oldConstraint.Type != newConstraint.Type {
		alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.DropConstraintStmt{
			Table:          alterTableStmt.Table,
			ConstraintName: constraintName,
		})
		alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.AddConstraintStmt{
			Table:      alterTableStmt.Table,
			Constraint: newConstraint,
		})
		return nil
	}

	switch newConstraint.Type {
	case ast.ConstraintTypeCheck:
		if newConstraint.Expression.Text() != oldConstraint.Expression.Text() {
			alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.DropConstraintStmt{
				Table:          alterTableStmt.Table,
				ConstraintName: constraintName,
			})
			alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.AddConstraintStmt{
				Table:      alterTableStmt.Table,
				Constraint: newConstraint,
			})
			return nil
		}
	case ast.ConstraintTypeUnique, ast.ConstraintTypePrimary, ast.ConstraintTypeExclusion:
		// TODO(zp): To make the logic simple now, we just restore the statement, and drop and create the new one if
		// there is any difference.
		oldAlterTableAddConstraint, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, oldConstraint)
		if err != nil {
			return errors.Wrapf(err, "failed to deparse old alter table constraintDef: %v", oldConstraint)
		}
		newAlterTableAddConstraint, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, newConstraint)
		if err != nil {
			return errors.Wrapf(err, "failed to deparse new alter table constraintDef: %v", newConstraint)
		}
		if oldAlterTableAddConstraint != newAlterTableAddConstraint {
			alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.DropConstraintStmt{
				Table:          alterTableStmt.Table,
				ConstraintName: constraintName,
				IfExists:       true,
			})
			alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.AddConstraintStmt{
				Table:      alterTableStmt.Table,
				Constraint: newConstraint,
			})
		}
	default:
		return errors.Errorf("Unsupported table constraint type: %d for modifyConstraint", newConstraint.Type)
	}
	return nil
}

func (*diffNode) modifyColumn(alterTableStmt *ast.AlterTableStmt, oldColumn *ast.ColumnDef, newColumn *ast.ColumnDef) error {
	columnName := oldColumn.ColumnName
	// compare the data type
	equivalent, err := equivalentType(oldColumn.Type, newColumn.Type)
	if err != nil {
		return err
	}
	if !equivalent {
		alterTableStmt.AlterItemList = append(alterTableStmt.AlterItemList, &ast.AlterColumnTypeStmt{
			Table:      alterTableStmt.Table,
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
	return nil
}

func (diff *diffNode) modifyIndex(oldIndex *ast.CreateIndexStmt, newIndex *ast.CreateIndexStmt) error {
	// TODO(rebelice): not use Text(), it only works for pg_dump.
	if oldIndex.Text() != newIndex.Text() {
		diff.modifyIndexList = append(diff.modifyIndexList, &ast.DropIndexStmt{
			IfExists: true,
			Behavior: ast.DropBehaviorCascade,
			IndexList: []*ast.IndexDef{
				{
					Table: &ast.TableDef{Schema: oldIndex.Index.Table.Schema},
					Name:  oldIndex.Index.Name,
				},
			},
		})
		diff.modifyIndexList = append(diff.modifyIndexList, newIndex)
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

func (diff *diffNode) deparse() (string, error) {
	var buf bytes.Buffer
	for _, newSchema := range diff.newSchemaList {
		newSchema.IfNotExists = true
		sql, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, newSchema)
		if err != nil {
			return "", err
		}
		if err := writeStringWithNewLine(&buf, sql); err != nil {
			return "", err
		}
	}

	for _, newTable := range diff.newTableList {
		if err := writeStringWithNewLine(&buf, newTable.Text()); err != nil {
			return "", err
		}
	}

	for _, modifyTable := range diff.modifyTableList {
		sql, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, modifyTable)
		if err != nil {
			return "", err
		}
		if err := writeStringWithNewLine(&buf, sql); err != nil {
			return "", err
		}
	}

	for _, newIndex := range diff.newIndexList {
		sql, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, newIndex)
		if err != nil {
			return "", err
		}
		if err := writeStringWithNewLine(&buf, sql); err != nil {
			return "", err
		}
	}

	for _, modifyIndex := range diff.modifyIndexList {
		sql, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, modifyIndex)
		if err != nil {
			return "", err
		}
		if err := writeStringWithNewLine(&buf, sql); err != nil {
			return "", err
		}
	}

	// Deparse the drop node in reverse order.
	for i := len(diff.dropNodeList) - 1; i >= 0; i-- {
		sql, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, diff.dropNodeList[i])
		if err != nil {
			return "", err
		}
		if err := writeStringWithNewLine(&buf, sql); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func dropConstraint(m schemaMap) []*ast.AlterTableStmt {
	var dropConstraintList []*ast.AlterTableStmt
	for schemaName, schemaInfo := range m {
		for tableName, tableInfo := range schemaInfo.tableMap {
			var constraintInfoList []*constraintInfo
			for _, constraintInfo := range tableInfo.constraintMap {
				if !constraintInfo.existsInNew {
					constraintInfoList = append(constraintInfoList, constraintInfo)
				}
			}
			sort.Slice(constraintInfoList, func(i, j int) bool {
				return constraintInfoList[i].id < constraintInfoList[j].id
			})
			var alterItemList []ast.Node
			for _, constraintInfo := range constraintInfoList {
				alterItemList = append(alterItemList, &ast.DropConstraintStmt{
					Table:          &ast.TableDef{Schema: schemaName, Name: tableName},
					ConstraintName: constraintInfo.addConstraint.Constraint.Name,
					IfExists:       true,
				})
			}
			if len(alterItemList) > 0 {
				dropConstraintList = append(dropConstraintList, &ast.AlterTableStmt{
					Table:         &ast.TableDef{Schema: schemaName, Name: tableName},
					AlterItemList: alterItemList,
				})
			}
		}
	}
	return dropConstraintList
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
