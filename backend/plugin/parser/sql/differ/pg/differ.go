// Package pg provides the PostgreSQL differ plugin.
package pg

import (
	"bytes"
	"io"
	"sort"

	"github.com/pkg/errors"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/differ"
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
	dropTriggerList            []ast.Node
	dropIndexList              []ast.Node
	dropDefaultList            []ast.Node
	dropSequenceOwnedByList    []ast.Node
	dropColumnList             []ast.Node
	dropTableList              []ast.Node
	dropSequenceList           []ast.Node
	dropFunctionList           []ast.Node
	dropExtensionList          []ast.Node
	dropTypeList               []ast.Node
	dropSchemaList             []ast.Node

	// Create nodes
	createSchemaList               []ast.Node
	createTypeList                 []ast.Node
	alterTypeList                  []ast.Node
	createExtensionList            []ast.Node
	createFunctionList             []ast.Node
	createSequenceList             []ast.Node
	alterSequenceExceptOwnedByList []ast.Node
	createTableList                []ast.Node
	createColumnList               []ast.Node
	alterColumnList                []ast.Node
	setSequenceOwnedByList         []ast.Node
	setDefaultList                 []ast.Node
	createIndexList                []ast.Node
	createTriggerList              []ast.Node
	createConstraintExceptFkList   []ast.Node
	createForeignKeyList           []ast.Node
}

type schemaMap map[string]*schemaInfo
type tableMap map[string]*tableInfo
type constraintMap map[string]*constraintInfo
type indexMap map[string]*indexInfo
type sequenceMap map[string]*sequenceInfo
type extensionMap map[string]*extensionInfo
type functionMap map[string]*functionInfo
type triggerMap map[string]*triggerInfo
type typeMap map[string]*typeInfo

type schemaInfo struct {
	id           int
	existsInNew  bool
	createSchema *ast.CreateSchemaStmt
	tableMap     tableMap
	indexMap     indexMap
	sequenceMap  sequenceMap
	extensionMap extensionMap
	functionMap  functionMap
	typeMap      typeMap
}

func newSchemaInfo(id int, createSchema *ast.CreateSchemaStmt) *schemaInfo {
	return &schemaInfo{
		id:           id,
		existsInNew:  false,
		createSchema: createSchema,
		tableMap:     make(tableMap),
		indexMap:     make(indexMap),
		sequenceMap:  make(sequenceMap),
		extensionMap: make(extensionMap),
		functionMap:  make(functionMap),
		typeMap:      make(typeMap),
	}
}

type tableInfo struct {
	id            int
	existsInNew   bool
	createTable   *ast.CreateTableStmt
	constraintMap constraintMap
	triggerMap    triggerMap
}

func newTableInfo(id int, createTable *ast.CreateTableStmt) *tableInfo {
	return &tableInfo{
		id:            id,
		existsInNew:   false,
		createTable:   createTable,
		constraintMap: make(constraintMap),
		triggerMap:    make(triggerMap),
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

type sequenceInfo struct {
	id             int
	existsInNew    bool
	createSequence *ast.CreateSequenceStmt
	ownedByInfo    *sequenceOwnedByInfo
}

func newSequenceInfo(id int, createSequence *ast.CreateSequenceStmt) *sequenceInfo {
	return &sequenceInfo{
		id:             id,
		existsInNew:    false,
		createSequence: createSequence,
	}
}

type sequenceOwnedByInfo struct {
	id          int
	existsInNew bool
	ownedBy     *ast.AlterSequenceStmt
}

func newSequenceOwnedByInfo(id int, ownedBy *ast.AlterSequenceStmt) *sequenceOwnedByInfo {
	return &sequenceOwnedByInfo{
		id:          id,
		existsInNew: false,
		ownedBy:     ownedBy,
	}
}

type extensionInfo struct {
	id              int
	existsInNew     bool
	createExtension *ast.CreateExtensionStmt
}

func newExtensionInfo(id int, createExtension *ast.CreateExtensionStmt) *extensionInfo {
	return &extensionInfo{
		id:              id,
		existsInNew:     false,
		createExtension: createExtension,
	}
}

type functionInfo struct {
	id             int
	existsInNew    bool
	createFunction *ast.CreateFunctionStmt
}

func newFunctionInfo(id int, createFunction *ast.CreateFunctionStmt) *functionInfo {
	return &functionInfo{
		id:             id,
		existsInNew:    false,
		createFunction: createFunction,
	}
}

type triggerInfo struct {
	id            int
	existsInNew   bool
	createTrigger *ast.CreateTriggerStmt
}

func newTriggerInfo(id int, createTrigger *ast.CreateTriggerStmt) *triggerInfo {
	return &triggerInfo{
		id:            id,
		existsInNew:   false,
		createTrigger: createTrigger,
	}
}

type typeInfo struct {
	id          int
	existsInNew bool
	createType  *ast.CreateTypeStmt
}

func newTypeInfo(id int, createType *ast.CreateTypeStmt) *typeInfo {
	return &typeInfo{
		id:          id,
		existsInNew: false,
		createType:  createType,
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

func (m schemaMap) addExtension(id int, extension *ast.CreateExtensionStmt) error {
	schema, exists := m[extension.Schema]
	if !exists {
		return errors.Errorf("failed to add extension: schema %s not found", extension.Schema)
	}
	schema.extensionMap[extension.Name] = newExtensionInfo(id, extension)
	return nil
}

func (m schemaMap) getExtension(schemaName string, extensionName string) *extensionInfo {
	schema, exists := m[schemaName]
	if !exists {
		return nil
	}
	return schema.extensionMap[extensionName]
}

func (m schemaMap) addFunction(id int, function *ast.CreateFunctionStmt) error {
	schema, exists := m[function.Function.Schema]
	if !exists {
		return errors.Errorf("failed to add function: schema %s not found", function.Function.Schema)
	}
	signature, err := functionSignature(function.Function)
	if err != nil {
		return err
	}
	schema.functionMap[signature] = newFunctionInfo(id, function)
	return nil
}

func (m schemaMap) getFunction(schemaName string, signature string) *functionInfo {
	schema, exists := m[schemaName]
	if !exists {
		return nil
	}
	return schema.functionMap[signature]
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

func (m schemaMap) addSequence(id int, sequence *ast.CreateSequenceStmt) error {
	schema, exists := m[sequence.SequenceDef.SequenceName.Schema]
	if !exists {
		return errors.Errorf("failed to add sequence: schema %s not found", sequence.SequenceDef.SequenceName.Schema)
	}
	schema.sequenceMap[sequence.SequenceDef.SequenceName.Name] = newSequenceInfo(id, sequence)
	return nil
}

func (m schemaMap) getSequence(schemaName string, sequenceName string) *sequenceInfo {
	schema, exists := m[schemaName]
	if !exists {
		return nil
	}
	return schema.sequenceMap[sequenceName]
}

func (m schemaMap) addTrigger(id int, trigger *ast.CreateTriggerStmt) error {
	schema, exists := m[trigger.Trigger.Table.Schema]
	if !exists {
		return errors.Errorf("failed to add trigger: schema %s not found", trigger.Trigger.Table.Schema)
	}
	table, exists := schema.tableMap[trigger.Trigger.Table.Name]
	if !exists {
		return errors.Errorf("failed to add trigger: table %s no found", trigger.Trigger.Table.Name)
	}
	table.triggerMap[trigger.Trigger.Name] = newTriggerInfo(id, trigger)
	return nil
}

func (m schemaMap) getTrigger(schemaName string, tableName string, triggerName string) *triggerInfo {
	schema, exists := m[schemaName]
	if !exists {
		return nil
	}
	table, exists := schema.tableMap[tableName]
	if !exists {
		return nil
	}
	return table.triggerMap[triggerName]
}

func (m schemaMap) addType(id int, createType *ast.CreateTypeStmt) error {
	schema, exists := m[createType.Type.TypeName().Schema]
	if !exists {
		return errors.Errorf("failed to add type: schema %s not found", createType.Type.TypeName().Schema)
	}
	schema.typeMap[createType.Type.TypeName().Name] = newTypeInfo(id, createType)
	return nil
}

func (m schemaMap) getType(schemaName string, typeName string) *typeInfo {
	schema, exists := m[schemaName]
	if !exists {
		return nil
	}
	return schema.typeMap[typeName]
}

func onlySetOwnedBy(sequence *ast.AlterSequenceStmt) bool {
	return sequence.Type == nil &&
		sequence.IncrementBy == nil &&
		!sequence.NoMinValue &&
		sequence.MinValue == nil &&
		!sequence.NoMaxValue &&
		sequence.MaxValue == nil &&
		sequence.StartWith == nil &&
		sequence.RestartWith == nil &&
		sequence.Cache == nil &&
		sequence.Cycle == nil &&
		!sequence.OwnedByNone &&
		sequence.OwnedBy != nil
}

func (m schemaMap) addSequenceOwnedBy(id int, alterStmt *ast.AlterSequenceStmt) error {
	// pg_dump will separate the SET OWNED BY clause into a ALTER SEQUENCE statement.
	// There would be no other ALTER SEQUENCE statements.
	if !onlySetOwnedBy(alterStmt) {
		return errors.Errorf("expect OwnedBy only, but found %v", alterStmt)
	}

	schema, exists := m[alterStmt.Name.Schema]
	if !exists {
		return errors.Errorf("failed to add sequence owned by: schema %s not found", alterStmt.Name.Schema)
	}
	sequence, exists := schema.sequenceMap[alterStmt.Name.Name]
	if !exists {
		return errors.Errorf("failed to add sequence owned by: sequence %s not found", alterStmt.Name.Name)
	}
	sequence.ownedByInfo = newSequenceOwnedByInfo(id, alterStmt)
	return nil
}

func parseAndPreprocessStatment(statement string) ([]ast.Node, error) {
	nodeList, err := parser.Parse(parser.Postgres, parser.ParseContext{}, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse statement %q", statement)
	}
	return mergeDefaultIntoColumn(nodeList)
}

// mergeDefaultIntoColumn merges some statements into one statement, for example, merge
// `CREATE TABLE tbl(id INT DEFAULT '3');`,
// `ALTER TABLE ONLY tbl ALTER COLUMN id nextval('tbl_id_seq'::regclass);`
// to `CREATE TABLE tbl(id INT DEFAULT nextval('tbl_id_seq'::regclass));`.
func mergeDefaultIntoColumn(nodeList []ast.Node) ([]ast.Node, error) {
	var retNodes []ast.Node

	schemaTableNameToRetNodesIdx := make(map[string]int)
	for _, node := range nodeList {
		switch node := node.(type) {
		case *ast.CreateTableStmt:
			schemaTableName := node.Name.Schema + "." + node.Name.Name
			retNodes = append(retNodes, node)
			schemaTableNameToRetNodesIdx[schemaTableName] = len(retNodes) - 1
		case *ast.AlterTableStmt:
			schemaTableName := node.Table.Schema + "." + node.Table.Name
			retNodesIdx, ok := schemaTableNameToRetNodesIdx[schemaTableName]
			if !ok {
				// For pg_dump, this will never happen.
				return nil, errors.Errorf("cannot find table %s", schemaTableName)
			}
			for _, alterItem := range node.AlterItemList {
				switch item := alterItem.(type) {
				case *ast.SetDefaultStmt:
					for _, columnDef := range retNodes[retNodesIdx].(*ast.CreateTableStmt).ColumnList {
						if columnDef.ColumnName == item.ColumnName {
							constraintDef := &ast.ConstraintDef{
								Type:       ast.ConstraintTypeDefault,
								Expression: item.Expression,
							}
							// pg_dump will ensure that there is only one default constraint, in ColumnDef or AlterTableStmt.
							columnDef.ConstraintList = append(columnDef.ConstraintList, constraintDef)
							break
						}
					}
				default:
					retNodes = append(retNodes, node)
				}
				// For pg_dump, Set Default statements will be alone in one alter tabel statement.
				// So here sikp append is safe.
			}
		default:
			retNodes = append(retNodes, node)
		}
	}
	return retNodes, nil
}

// SchemaDiff computes the schema differences between old and new schema.
func (*SchemaDiffer) SchemaDiff(oldStmt, newStmt string, _ bool) (string, error) {
	oldNodes, err := parseAndPreprocessStatment(oldStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse and preprocess old statements %q", oldStmt)
	}
	newNodes, err := parseAndPreprocessStatment(newStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to and preprocess new statements %q", newStmt)
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
		case *ast.CreateSequenceStmt:
			if err := oldSchemaMap.addSequence(i, stmt); err != nil {
				return "", err
			}
		case *ast.AlterSequenceStmt:
			// pg_dump will separate the SET OWNED BY clause into a ALTER SEQUENCE statement.
			// There would be no other ALTER SEQUENCE statements.
			if err := oldSchemaMap.addSequenceOwnedBy(i, stmt); err != nil {
				return "", err
			}
		case *ast.CreateExtensionStmt:
			if err := oldSchemaMap.addExtension(i, stmt); err != nil {
				return "", err
			}
		case *ast.CreateFunctionStmt:
			if err := oldSchemaMap.addFunction(i, stmt); err != nil {
				return "", err
			}
		case *ast.CreateTriggerStmt:
			if err := oldSchemaMap.addTrigger(i, stmt); err != nil {
				return "", err
			}
		case *ast.CreateTypeStmt:
			if err := oldSchemaMap.addType(i, stmt); err != nil {
				return "", err
			}
			// TODO(rebelice): add default back here
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
		case *ast.CreateSequenceStmt:
			oldSequence := oldSchemaMap.getSequence(stmt.SequenceDef.SequenceName.Schema, stmt.SequenceDef.SequenceName.Name)
			// Add the new sequence.
			if oldSequence == nil {
				diff.createSequenceList = append(diff.createSequenceList, stmt)
				continue
			}
			oldSequence.existsInNew = true
			// Modify the sequence.
			if err := diff.modifySequenceExceptOwnedBy(oldSequence.createSequence, stmt); err != nil {
				return "", err
			}
		case *ast.AlterSequenceStmt:
			if !onlySetOwnedBy(stmt) {
				return "", errors.Errorf("expect OwnedBy only, but found %v", stmt)
			}
			oldSequence := oldSchemaMap.getSequence(stmt.Name.Schema, stmt.Name.Name)
			// Add the new sequence owned by.
			if oldSequence == nil || oldSequence.ownedByInfo == nil {
				diff.setSequenceOwnedByList = append(diff.setSequenceOwnedByList, stmt)
				continue
			}
			oldSequence.ownedByInfo.existsInNew = true
			if err := diff.modifySequenceOwnedBy(oldSequence.ownedByInfo.ownedBy, stmt); err != nil {
				return "", err
			}
		case *ast.CreateExtensionStmt:
			oldExtension := oldSchemaMap.getExtension(stmt.Schema, stmt.Name)
			// Add the extension.
			if oldExtension == nil {
				diff.createExtensionList = append(diff.createExtensionList, stmt)
				continue
			}
			oldExtension.existsInNew = true
			// Modify the extension.
			if err := diff.modifyExtension(oldExtension.createExtension, stmt); err != nil {
				return "", err
			}
		case *ast.CreateFunctionStmt:
			signature, err := functionSignature(stmt.Function)
			if err != nil {
				return "", err
			}
			oldFunction := oldSchemaMap.getFunction(stmt.Function.Schema, signature)
			// Add the function.
			if oldFunction == nil {
				diff.createFunctionList = append(diff.createFunctionList, stmt)
				continue
			}
			oldFunction.existsInNew = true
			// Modify the function.
			if err := diff.modifyFunction(oldFunction.createFunction, stmt); err != nil {
				return "", err
			}
		case *ast.CreateTriggerStmt:
			oldTrigger := oldSchemaMap.getTrigger(stmt.Trigger.Table.Schema, stmt.Trigger.Table.Name, stmt.Trigger.Name)
			// Add the trigger.
			if oldTrigger == nil {
				diff.createTriggerList = append(diff.createTriggerList, stmt)
				continue
			}
			oldTrigger.existsInNew = true
			// Modify the trigger.
			if err := diff.modifyTrigger(oldTrigger.createTrigger, stmt); err != nil {
				return "", err
			}
		case *ast.CreateTypeStmt:
			oldType := oldSchemaMap.getType(stmt.Type.TypeName().Schema, stmt.Type.TypeName().Name)
			// Add the type.
			if oldType == nil {
				diff.createTypeList = append(diff.createTypeList, stmt)
				continue
			}
			oldType.existsInNew = true
			// Modify the type.
			if err := diff.modifyType(oldType.createType, stmt); err != nil {
				return "", err
			}
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

	// Drop the remaining old sequence owned by.
	diff.dropSequenceOwnedBy(oldSchemaMap)

	// Drop the remaining old sequence.
	if dropSequenceStmt := dropSequence(oldSchemaMap); dropSequenceStmt != nil {
		diff.dropSequenceList = append(diff.dropSequenceList, dropSequenceStmt)
	}

	// Drop the remaining old extension.
	if dropExtensionStmt := dropExtension(oldSchemaMap); dropExtensionStmt != nil {
		diff.dropExtensionList = append(diff.dropExtensionList, dropExtensionStmt)
	}

	// Drop the remaining old function.
	if dropFunctionStmt := dropFunction(oldSchemaMap); dropFunctionStmt != nil {
		diff.dropFunctionList = append(diff.dropFunctionList, dropFunctionStmt)
	}

	// Drop the remaining old trigger.
	diff.dropTriggerStmt(oldSchemaMap)

	// Drop the remaining old type.
	diff.dropTypeStmt(oldSchemaMap)

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
		diff.setDefaultList = append(diff.setDefaultList, &ast.AlterTableStmt{
			Table: tableName,
			AlterItemList: []ast.Node{
				&ast.SetDefaultStmt{
					Table:      alterTableStmt.Table,
					ColumnName: columnName,
					Expression: expression,
				},
			},
		})
	} else if needDropDefault {
		diff.dropDefaultList = append(diff.dropDefaultList, &ast.AlterTableStmt{
			Table: tableName,
			AlterItemList: []ast.Node{
				&ast.DropDefaultStmt{
					Table:      alterTableStmt.Table,
					ColumnName: columnName,
				},
			},
		})
	}

	// TODO(rebelice): compare other column properties
	if len(alterTableStmt.AlterItemList) > 0 {
		diff.alterColumnList = append(diff.alterColumnList, alterTableStmt)
	}
	return nil
}

func (diff *diffNode) modifyExtension(oldExtension *ast.CreateExtensionStmt, newExtension *ast.CreateExtensionStmt) error {
	// TODO(rebelice): not use Text(), it only works for pg_dump.
	if oldExtension.Text() != newExtension.Text() {
		diff.dropExtensionList = append(diff.dropExtensionList, &ast.DropExtensionStmt{
			NameList: []string{oldExtension.Name},
		})
		diff.createExtensionList = append(diff.createExtensionList, newExtension)
	}
	return nil
}

func (diff *diffNode) modifyFunction(oldFunction *ast.CreateFunctionStmt, newFunction *ast.CreateFunctionStmt) error {
	// TODO(rebelice): not use Text(), it only works for pg_dump.
	if oldFunction.Text() != newFunction.Text() {
		diff.dropFunctionList = append(diff.dropFunctionList, &ast.DropFunctionStmt{
			FunctionList: []*ast.FunctionDef{oldFunction.Function},
		})
		diff.createFunctionList = append(diff.createFunctionList, newFunction)
	}
	return nil
}

func isSubsequenceEnum(oldType ast.UserDefinedType, newType ast.UserDefinedType) bool {
	oldEnum, ok := oldType.(*ast.EnumTypeDef)
	if !ok {
		return false
	}
	newEnum, ok := newType.(*ast.EnumTypeDef)
	if !ok {
		return false
	}

	pos := 0
	for _, oldLabel := range oldEnum.LabelList {
		for {
			if pos >= len(newEnum.LabelList) {
				return false
			}
			if newEnum.LabelList[pos] == oldLabel {
				break
			}
			pos++
		}
	}
	return true
}

func (diff *diffNode) addEnumValue(oldType *ast.CreateTypeStmt, newType *ast.CreateTypeStmt) error {
	oldEnum, ok := oldType.Type.(*ast.EnumTypeDef)
	if !ok {
		// never catch
		return parser.NewConvertErrorf("expected EnumTypeDef but found %t", oldType.Type)
	}
	newEnum, ok := newType.Type.(*ast.EnumTypeDef)
	if !ok {
		// never catch
		return parser.NewConvertErrorf("expected EnumTypeDef but found %t", newType.Type)
	}

	// oldEnum has empty label list, so append newEnum labels.
	if len(oldEnum.LabelList) == 0 {
		for _, label := range newEnum.LabelList {
			diff.alterTypeList = append(diff.alterTypeList, &ast.AlterTypeStmt{
				Type: newType.Type.TypeName(),
				AlterItemList: []ast.Node{&ast.AddEnumLabelStmt{
					EnumType: newType.Type.TypeName(),
					NewLabel: label,
					Position: ast.PositionTypeEnd,
				}},
			})
		}
		return nil
	}

	firstOldLabelPos := 0
	for {
		if newEnum.LabelList[firstOldLabelPos] == oldEnum.LabelList[0] {
			break
		}
		firstOldLabelPos++
	}

	// Add Labels before first equal label by BEFORE.
	for i := firstOldLabelPos - 1; i >= 0; i-- {
		diff.alterTypeList = append(diff.alterTypeList, &ast.AlterTypeStmt{
			Type: newType.Type.TypeName(),
			AlterItemList: []ast.Node{&ast.AddEnumLabelStmt{
				EnumType:      newType.Type.TypeName(),
				NewLabel:      newEnum.LabelList[i],
				Position:      ast.PositionTypeBefore,
				NeighborLabel: newEnum.LabelList[i+1],
			}},
		})
	}

	// Add remaining labels by AFTER.
	oldLabelPos := 1
	for i := firstOldLabelPos + 1; i < len(newEnum.LabelList); i++ {
		newLabel := newEnum.LabelList[i]
		if len(oldEnum.LabelList) > oldLabelPos && newLabel == oldEnum.LabelList[oldLabelPos] {
			oldLabelPos++
			continue
		}
		diff.alterTypeList = append(diff.alterTypeList, &ast.AlterTypeStmt{
			Type: newType.Type.TypeName(),
			AlterItemList: []ast.Node{&ast.AddEnumLabelStmt{
				EnumType:      newType.Type.TypeName(),
				NewLabel:      newLabel,
				Position:      ast.PositionTypeAfter,
				NeighborLabel: newEnum.LabelList[i-1],
			}},
		})
	}

	return nil
}

func (diff *diffNode) modifyType(oldType *ast.CreateTypeStmt, newType *ast.CreateTypeStmt) error {
	// TODO(rebelice): not use Text(), it only works for pg_dump.
	if oldType.Text() != newType.Text() {
		// Add enum value.
		if isSubsequenceEnum(oldType.Type, newType.Type) {
			return diff.addEnumValue(oldType, newType)
		}

		// DROP and RE-CREATE.
		diff.dropTypeList = append(diff.dropTypeList, &ast.DropTypeStmt{
			TypeNameList: []*ast.TypeNameDef{oldType.Type.TypeName()},
		})
		diff.createTypeList = append(diff.createTypeList, newType)
	}
	return nil
}

func (diff *diffNode) modifyTrigger(oldTrigger *ast.CreateTriggerStmt, newTrigger *ast.CreateTriggerStmt) error {
	// TODO(rebelice): not use Text(), it only works for pg_dump.
	if oldTrigger.Text() != newTrigger.Text() {
		diff.dropTriggerList = append(diff.dropTriggerList, &ast.DropTriggerStmt{
			Trigger: oldTrigger.Trigger,
		})
		diff.createTriggerList = append(diff.createTriggerList, newTrigger)
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

func (diff *diffNode) modifySequenceOwnedBy(oldSequenceOwnedBy *ast.AlterSequenceStmt, newSequenceOwnedBy *ast.AlterSequenceStmt) error {
	if !isEqualColumnNameDef(oldSequenceOwnedBy.OwnedBy, newSequenceOwnedBy.OwnedBy) {
		diff.setDefaultList = append(diff.setDefaultList, newSequenceOwnedBy)
	}
	return nil
}

func (diff *diffNode) modifySequenceExceptOwnedBy(oldSequence *ast.CreateSequenceStmt, newSequence *ast.CreateSequenceStmt) error {
	isEqual := true
	alterSequence := &ast.AlterSequenceStmt{
		Name: oldSequence.SequenceDef.SequenceName,
	}

	// compare data type
	if !isEqualInteger(oldSequence.SequenceDef.SequenceDataType, newSequence.SequenceDef.SequenceDataType) {
		alterSequence.Type = newSequence.SequenceDef.SequenceDataType
		isEqual = false
	}

	// compare increment
	if !isEqualInt32Pointer(oldSequence.SequenceDef.IncrementBy, newSequence.SequenceDef.IncrementBy) {
		alterSequence.IncrementBy = newSequence.SequenceDef.IncrementBy
		isEqual = false
	}

	// compare min value
	if !isEqualInt32Pointer(oldSequence.SequenceDef.MinValue, newSequence.SequenceDef.MinValue) {
		if newSequence.SequenceDef.MinValue == nil {
			alterSequence.NoMinValue = true
		} else {
			alterSequence.MinValue = newSequence.SequenceDef.MinValue
		}
		isEqual = false
	}

	// compare max value
	if !isEqualInt32Pointer(oldSequence.SequenceDef.MaxValue, newSequence.SequenceDef.MaxValue) {
		if newSequence.SequenceDef.MaxValue == nil {
			alterSequence.NoMaxValue = true
		} else {
			alterSequence.MaxValue = newSequence.SequenceDef.MaxValue
		}
		isEqual = false
	}

	// compare start with
	if !isEqualInt32Pointer(oldSequence.SequenceDef.StartWith, newSequence.SequenceDef.StartWith) {
		if newSequence.SequenceDef.StartWith != nil {
			alterSequence.StartWith = newSequence.SequenceDef.StartWith
			isEqual = false
		}
	}

	// compare cache
	if !isEqualInt32Pointer(oldSequence.SequenceDef.Cache, newSequence.SequenceDef.Cache) {
		if newSequence.SequenceDef.Cache != nil {
			alterSequence.Cache = newSequence.SequenceDef.Cache
			isEqual = false
		}
	}

	// compare cycle
	if oldSequence.SequenceDef.Cycle != newSequence.SequenceDef.Cycle {
		alterSequence.Cycle = &newSequence.SequenceDef.Cycle
		isEqual = false
	}

	if !isEqual {
		diff.alterSequenceExceptOwnedByList = append(diff.alterSequenceExceptOwnedByList, alterSequence)
	}
	return nil
}

func isEqualTableDef(tableA *ast.TableDef, tableB *ast.TableDef) bool {
	if tableA == nil && tableB == nil {
		return true
	}
	if tableA == nil || tableB == nil {
		return false
	}

	if tableA.Database != tableB.Database {
		return false
	}

	if tableA.Schema != tableB.Schema {
		return false
	}

	if tableA.Name != tableB.Name {
		return false
	}

	return true
}

func isEqualColumnNameDef(columnA *ast.ColumnNameDef, columnB *ast.ColumnNameDef) bool {
	if columnA == nil && columnB == nil {
		return true
	}
	if columnA == nil || columnB == nil {
		return false
	}

	if !isEqualTableDef(columnA.Table, columnB.Table) {
		return false
	}

	return columnA.ColumnName == columnB.ColumnName
}

func isEqualInt32Pointer(a *int32, b *int32) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func isEqualInteger(typeA *ast.Integer, typeB *ast.Integer) bool {
	if typeA == nil && typeB == nil {
		return true
	}
	if typeA == nil || typeB == nil {
		return false
	}
	return typeA.Size == typeB.Size
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

func printStmtSliceByText(buf io.Writer, nodeList []ast.Node) error {
	for _, node := range nodeList {
		if err := writeStringWithNewLine(buf, node.Text()); err != nil {
			return err
		}
		if _, err := buf.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
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
		if _, err := buf.Write([]byte("\n")); err != nil {
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
	if err := printStmtSlice(&buf, diff.dropTriggerList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropIndexList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropDefaultList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropSequenceOwnedByList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropColumnList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropTableList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropSequenceList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropFunctionList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropExtensionList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropTypeList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropSchemaList); err != nil {
		return "", err
	}

	// create
	if err := printStmtSlice(&buf, diff.createSchemaList); err != nil {
		return "", err
	}
	if err := printStmtSliceByText(&buf, diff.createTypeList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.alterTypeList); err != nil {
		return "", err
	}
	if err := printStmtSliceByText(&buf, diff.createExtensionList); err != nil {
		return "", err
	}
	if err := printStmtSliceByText(&buf, diff.createFunctionList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.createSequenceList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.alterSequenceExceptOwnedByList); err != nil {
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
	if err := printStmtSlice(&buf, diff.setSequenceOwnedByList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.setDefaultList); err != nil {
		return "", err
	}
	if err := printStmtSliceByText(&buf, diff.createIndexList); err != nil {
		return "", err
	}
	if err := printStmtSliceByText(&buf, diff.createTriggerList); err != nil {
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
		TableList: tableDefList,
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
		SchemaList: schemaNameList,
	}
}

func dropExtension(m schemaMap) *ast.DropExtensionStmt {
	var extensionList []*extensionInfo
	for _, schema := range m {
		for _, extension := range schema.extensionMap {
			if extension.existsInNew {
				// no need to drop
				continue
			}
			extensionList = append(extensionList, extension)
		}
	}
	if len(extensionList) == 0 {
		return nil
	}
	sort.Slice(extensionList, func(i, j int) bool {
		return extensionList[i].id < extensionList[j].id
	})

	var extensionNameList []string
	for _, extension := range extensionList {
		extensionNameList = append(extensionNameList, extension.createExtension.Name)
	}
	return &ast.DropExtensionStmt{
		NameList: extensionNameList,
	}
}

func dropFunction(m schemaMap) *ast.DropFunctionStmt {
	var functionList []*functionInfo
	for _, schema := range m {
		for _, function := range schema.functionMap {
			if function.existsInNew {
				// no need to drop
				continue
			}
			functionList = append(functionList, function)
		}
	}
	if len(functionList) == 0 {
		return nil
	}
	sort.Slice(functionList, func(i, j int) bool {
		return functionList[i].id < functionList[j].id
	})

	var functionDefList []*ast.FunctionDef
	for _, function := range functionList {
		functionDefList = append(functionDefList, function.createFunction.Function)
	}
	return &ast.DropFunctionStmt{FunctionList: functionDefList}
}

func (diff *diffNode) dropTypeStmt(m schemaMap) {
	var typeList []*typeInfo
	for _, schema := range m {
		for _, tp := range schema.typeMap {
			if tp.existsInNew {
				// no need to drop
				continue
			}
			typeList = append(typeList, tp)
		}
	}
	if len(typeList) == 0 {
		return
	}
	sort.Slice(typeList, func(i, j int) bool {
		return typeList[i].id < typeList[j].id
	})

	for _, tp := range typeList {
		diff.dropTypeList = append(diff.dropTypeList, &ast.DropTypeStmt{
			TypeNameList: []*ast.TypeNameDef{tp.createType.Type.TypeName()},
		})
	}
}

func (diff *diffNode) dropTriggerStmt(m schemaMap) {
	var triggerList []*triggerInfo
	for _, schema := range m {
		for _, table := range schema.tableMap {
			for _, trigger := range table.triggerMap {
				if trigger.existsInNew {
					// no need to drop
					continue
				}
				triggerList = append(triggerList, trigger)
			}
		}
	}
	if len(triggerList) == 0 {
		return
	}
	sort.Slice(triggerList, func(i, j int) bool {
		return triggerList[i].id < triggerList[j].id
	})

	for _, trigger := range triggerList {
		diff.dropTriggerList = append(diff.dropTriggerList,
			&ast.DropTriggerStmt{Trigger: trigger.createTrigger.Trigger})
	}
}

func dropIndex(m schemaMap) *ast.DropIndexStmt {
	var indexList []*indexInfo
	for _, schema := range m {
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
		IndexList: indexDefList,
	}
}

func (diff *diffNode) dropSequenceOwnedBy(m schemaMap) {
	var sequenceOwnedByList []*sequenceOwnedByInfo
	for _, schema := range m {
		for _, sequence := range schema.sequenceMap {
			if sequence.ownedByInfo == nil || sequence.ownedByInfo.existsInNew {
				// no need to drop
				continue
			}
			sequenceOwnedByList = append(sequenceOwnedByList, sequence.ownedByInfo)
		}
	}

	if len(sequenceOwnedByList) == 0 {
		return
	}
	sort.Slice(sequenceOwnedByList, func(i, j int) bool {
		return sequenceOwnedByList[i].id < sequenceOwnedByList[j].id
	})

	for _, sequenceOwnedBy := range sequenceOwnedByList {
		diff.dropSequenceOwnedByList = append(diff.dropSequenceOwnedByList, &ast.AlterSequenceStmt{
			Name:        sequenceOwnedBy.ownedBy.Name,
			OwnedByNone: true,
		})
	}
}

func dropSequence(m schemaMap) *ast.DropSequenceStmt {
	var sequenceList []*sequenceInfo
	for _, schema := range m {
		for _, sequence := range schema.sequenceMap {
			if sequence.existsInNew {
				// no need to drop
				continue
			}
			sequenceList = append(sequenceList, sequence)
		}
	}

	if len(sequenceList) == 0 {
		return nil
	}
	sort.Slice(sequenceList, func(i, j int) bool {
		return sequenceList[i].id < sequenceList[j].id
	})

	var sequenceNameList []*ast.SequenceNameDef
	for _, sequence := range sequenceList {
		sequenceNameList = append(sequenceNameList, &ast.SequenceNameDef{
			Schema: sequence.createSequence.SequenceDef.SequenceName.Schema,
			Name:   sequence.createSequence.SequenceDef.SequenceName.Name,
		})
	}
	return &ast.DropSequenceStmt{
		SequenceNameList: sequenceNameList,
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

// use DROP FUNCTION statement as the function signature.
func functionSignature(function *ast.FunctionDef) (string, error) {
	return parser.Deparse(parser.Postgres, parser.DeparseContext{}, &ast.DropFunctionStmt{
		FunctionList: []*ast.FunctionDef{function},
	})
}
