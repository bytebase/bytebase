// Package pg provides the PostgreSQL differ plugin.
package pg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"

	"github.com/pkg/errors"

	pgquery "github.com/pganalyze/pg_query_go/v6"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterSchemaDiffFunc(storepb.Engine_POSTGRES, SchemaDiff)
}

// diffNode defines different modification types as the safe change order.
// The safe change order means we can change them with no dependency conflicts as this order.
type diffNode struct {
	oldSchemaMap schemaMap
	newSchemaMap schemaMap

	// Drop nodes
	dropForeignKeyList         []ast.Node
	dropConstraintExceptFkList []ast.Node
	dropTriggerList            []ast.Node
	dropIndexList              []ast.Node
	dropDefaultList            []ast.Node
	dropSequenceOwnedByList    []ast.Node

	dropViewList             []*ast.CreateViewStmt
	dropMaterializedViewList []*ast.CreateMaterializedViewStmt

	dropColumnList []ast.Node

	dropTableList    []*ast.CreateTableStmt
	dropFunctionList []*ast.CreateFunctionStmt

	dropSequenceList  []ast.Node
	dropExtensionList []ast.Node
	dropTypeList      []ast.Node
	dropSchemaList    []ast.Node

	// Create nodes
	createSchemaList               []ast.Node
	createTypeList                 []ast.Node
	alterTypeList                  []ast.Node
	createExtensionList            []ast.Node
	createSequenceList             []ast.Node
	alterSequenceExceptOwnedByList []ast.Node

	createTableList    []*ast.CreateTableStmt
	createFunctionList []*ast.CreateFunctionStmt

	createColumnList             []ast.Node
	alterColumnList              []ast.Node
	setSequenceOwnedByList       []ast.Node
	setDefaultList               []ast.Node
	createIndexList              []ast.Node
	createTriggerList            []ast.Node
	createConstraintExceptFkList []ast.Node
	createForeignKeyList         []ast.Node

	createViewList             []*ast.CreateViewStmt
	createMaterializedViewList []*ast.CreateMaterializedViewStmt

	setCommentList []ast.Node
}

type schemaMap map[string]*schemaInfo
type tableMap map[string]*tableInfo
type viewMap map[string]*viewInfo
type materializedViewMap map[string]*materializedViewInfo
type constraintMap map[string]*constraintInfo
type indexMap map[string]*indexInfo
type sequenceMap map[string]*sequenceInfo
type extensionMap map[string]*extensionInfo
type functionMap map[string]*functionInfo
type triggerMap map[string]*triggerInfo
type typeMap map[string]*typeInfo

type schemaInfo struct {
	id                  int
	existsInNew         bool
	createSchema        *ast.CreateSchemaStmt
	tableMap            tableMap
	viewMap             viewMap
	materializedViewMap materializedViewMap
	indexMap            indexMap
	sequenceMap         sequenceMap
	extensionMap        extensionMap
	functionMap         functionMap
	typeMap             typeMap
}

func newSchemaInfo(id int, createSchema *ast.CreateSchemaStmt) *schemaInfo {
	return &schemaInfo{
		id:                  id,
		existsInNew:         false,
		createSchema:        createSchema,
		tableMap:            make(tableMap),
		viewMap:             make(viewMap),
		materializedViewMap: make(materializedViewMap),
		indexMap:            make(indexMap),
		sequenceMap:         make(sequenceMap),
		extensionMap:        make(extensionMap),
		functionMap:         make(functionMap),
		typeMap:             make(typeMap),
	}
}

type tableInfo struct {
	id               int
	existsInNew      bool
	createTable      *ast.CreateTableStmt
	constraintMap    constraintMap
	triggerMap       triggerMap
	comment          string
	columnCommentMap map[string]string
}

func newTableInfo(id int, createTable *ast.CreateTableStmt) *tableInfo {
	return &tableInfo{
		id:               id,
		existsInNew:      false,
		createTable:      createTable,
		constraintMap:    make(constraintMap),
		triggerMap:       make(triggerMap),
		comment:          "",
		columnCommentMap: make(map[string]string),
	}
}

type materializedViewInfo struct {
	id                     int
	existsInNew            bool
	createMaterializedView *ast.CreateMaterializedViewStmt
	comment                string
	columnCommentMap       map[string]string
}

func newMaterializedViewInfo(id int, createMaterializedView *ast.CreateMaterializedViewStmt) *materializedViewInfo {
	return &materializedViewInfo{
		id:                     id,
		existsInNew:            false,
		createMaterializedView: createMaterializedView,
		comment:                "",
		columnCommentMap:       make(map[string]string),
	}
}

type viewInfo struct {
	id               int
	existsInNew      bool
	createView       *ast.CreateViewStmt
	comment          string
	columnCommentMap map[string]string
}

func newViewInfo(id int, createView *ast.CreateViewStmt) *viewInfo {
	return &viewInfo{
		id:               id,
		existsInNew:      false,
		createView:       createView,
		comment:          "",
		columnCommentMap: make(map[string]string),
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

func (m schemaMap) addMaterializedView(id int, materializedView *ast.CreateMaterializedViewStmt) error {
	if IsSystemSchema(materializedView.Name.Schema) {
		return nil
	}
	schema, exists := m[materializedView.Name.Schema]
	if !exists {
		return errors.Errorf("failed to add materialized view: schema %s not found", materializedView.Name.Schema)
	}
	schema.materializedViewMap[materializedView.Name.Name] = newMaterializedViewInfo(id, materializedView)
	return nil
}

func (m schemaMap) getMaterializedView(schemaName string, materializedViewName string) *materializedViewInfo {
	schema, exists := m[schemaName]
	if !exists {
		return nil
	}
	return schema.materializedViewMap[materializedViewName]
}

func (m schemaMap) addView(id int, view *ast.CreateViewStmt) error {
	if IsSystemSchema(view.Name.Schema) {
		return nil
	}
	schema, exists := m[view.Name.Schema]
	if !exists {
		return errors.Errorf("failed to add view: schema %s not found", view.Name.Schema)
	}
	schema.viewMap[view.Name.Name] = newViewInfo(id, view)
	return nil
}

func (m schemaMap) getView(schemaName string, viewName string) *viewInfo {
	schema, exists := m[schemaName]
	if !exists {
		return nil
	}
	return schema.viewMap[viewName]
}

func (m schemaMap) addTable(id int, table *ast.CreateTableStmt) error {
	if IsSystemSchema(table.Name.Schema) {
		return nil
	}
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
	if IsSystemSchema(addConstraint.Table.Schema) {
		return nil
	}
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
	if IsSystemSchema(extension.Schema) {
		return nil
	}
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
	if IsSystemSchema(function.Function.Schema) {
		return nil
	}
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
	if IsSystemSchema(index.Index.Table.Schema) {
		return nil
	}
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
	if IsSystemSchema(sequence.SequenceDef.SequenceName.Schema) {
		return nil
	}
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
	if IsSystemSchema(trigger.Trigger.Table.Schema) {
		return nil
	}
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
	if IsSystemSchema(createType.Type.TypeName().Schema) {
		return nil
	}
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

func (m schemaMap) addComment(comment *ast.CommentStmt) error {
	switch comment.Type {
	case ast.ObjectTypeTable:
		tableDef, ok := comment.Object.(*ast.TableDef)
		if !ok {
			return errors.Errorf("failed to add comment: expect table def, but found %v", comment.Object)
		}
		schema, exists := m[tableDef.Schema]
		if !exists {
			return errors.Errorf("failed to add comment: schema %s not found", tableDef.Schema)
		}
		table, exists := schema.tableMap[tableDef.Name]
		if !exists {
			return errors.Errorf("failed to add comment: table %s not found", tableDef.Name)
		}
		table.comment = comment.Comment
		return nil
	case ast.ObjectTypeView:
		tableDef, ok := comment.Object.(*ast.TableDef)
		if !ok {
			return errors.Errorf("failed to add comment: expect view def, but found %v", comment.Object)
		}
		schema, exists := m[tableDef.Schema]
		if !exists {
			return errors.Errorf("failed to add comment: schema %s not found", tableDef.Schema)
		}
		view, exists := schema.viewMap[tableDef.Name]
		if !exists {
			return errors.Errorf("failed to add comment: view %s not found", tableDef.Name)
		}
		view.comment = comment.Comment
		return nil
	case ast.ObjectTypeColumn:
		columnNameDef, ok := comment.Object.(*ast.ColumnNameDef)
		if !ok {
			return errors.Errorf("failed to add comment: expect column name def, but found %v", comment.Object)
		}
		schema, exists := m[columnNameDef.Table.Schema]
		if !exists {
			slog.Debug("failed to add comment", slog.String("schema", columnNameDef.Table.Schema))
			return nil
		}
		table, exists := schema.tableMap[columnNameDef.Table.Name]
		if exists {
			table.columnCommentMap[columnNameDef.ColumnName] = comment.Comment
			return nil
		}
		view, exists := schema.viewMap[columnNameDef.Table.Name]
		if exists {
			view.columnCommentMap[columnNameDef.ColumnName] = comment.Comment
			return nil
		}
		slog.Debug("failed to add comment", slog.String("table or view", columnNameDef.Table.Name))
		return nil
	default:
		// We only support table and column comment.
		return nil
	}
}

func (m schemaMap) getTableComment(schemaName string, tableName string) string {
	schema, exists := m[schemaName]
	if !exists {
		return ""
	}
	table, exists := schema.tableMap[tableName]
	if !exists {
		return ""
	}
	return table.comment
}

func (m schemaMap) getViewComment(schemaName string, viewName string) string {
	schema, exists := m[schemaName]
	if !exists {
		return ""
	}
	view, exists := schema.viewMap[viewName]
	if !exists {
		return ""
	}
	return view.comment
}

func (m schemaMap) removeTableComment(schemaName string, tableName string) {
	schema, exists := m[schemaName]
	if !exists {
		return
	}
	table, exists := schema.tableMap[tableName]
	if !exists {
		return
	}
	table.comment = ""
}

func (m schemaMap) removeViewComment(schemaName string, viewName string) {
	schema, exists := m[schemaName]
	if !exists {
		return
	}
	view, exists := schema.viewMap[viewName]
	if !exists {
		return
	}
	view.comment = ""
}

func (m schemaMap) getColumnComment(schemaName string, tableName string, columnName string) string {
	schema, exists := m[schemaName]
	if !exists {
		return ""
	}
	table, exists := schema.tableMap[tableName]
	if exists {
		columnComment, exists := table.columnCommentMap[columnName]
		if exists {
			return columnComment
		}
	}
	view, exists := schema.viewMap[tableName]
	if exists {
		columnComment, exists := view.columnCommentMap[columnName]
		if exists {
			return columnComment
		}
	}
	return ""
}

func (m schemaMap) removeColumnComment(schemaName string, tableName string, columnName string) {
	schema, exists := m[schemaName]
	if !exists {
		return
	}
	table, exists := schema.tableMap[tableName]
	if exists {
		delete(table.columnCommentMap, columnName)
		return
	}
	view, exists := schema.viewMap[tableName]
	if exists {
		delete(view.columnCommentMap, columnName)
	}
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

	if IsSystemSchema(alterStmt.Name.Schema) {
		return nil
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

func parseAndPreprocessStatement(statement string) ([]ast.Node, []string, error) {
	nodeList, err := pgrawparser.Parse(pgrawparser.ParseContext{}, statement)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse statement %q", statement)
	}
	nodes, err := mergeDefaultIntoColumn(nodeList)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to merge default into column for statement %q", statement)
	}

	partitions := extractPartitions(nodes)
	return nodes, partitions, nil
}

func extractPartitions(nodes []ast.Node) []string {
	var partitions []string
	for _, node := range nodes {
		switch node := node.(type) {
		case *ast.AlterTableStmt:
			for _, alterItem := range node.AlterItemList {
				switch item := alterItem.(type) {
				case *ast.AttachPartitionStmt:
					if item.Partition != nil {
						partitions = append(partitions, fmt.Sprintf("%s.%s", item.Partition.Schema, item.Partition.Name))
					}
				default:
					continue
				}
			}
		default:
			continue
		}
	}
	return partitions
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
func SchemaDiff(_ base.DiffContext, oldStmt, newStmt string) (string, error) {
	oldNodes, oldPartitions, err := parseAndPreprocessStatement(oldStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse and preprocess old statements %q", oldStmt)
	}
	newNodes, newPartitions, err := parseAndPreprocessStatement(newStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to and preprocess new statements %q", newStmt)
	}
	diff := &diffNode{
		oldSchemaMap: make(schemaMap),
		newSchemaMap: make(schemaMap),
	}
	diff.oldSchemaMap["public"] = newSchemaInfo(-1, &ast.CreateSchemaStmt{Name: "public"})
	diff.newSchemaMap["public"] = newSchemaInfo(-1, &ast.CreateSchemaStmt{Name: "public"})
	diff.oldSchemaMap["public"].existsInNew = true
	oldPartitionMap := make(map[string]bool)
	for _, partition := range oldPartitions {
		oldPartitionMap[partition] = true
	}
	for i, node := range oldNodes {
		switch stmt := node.(type) {
		case *ast.CreateSchemaStmt:
			diff.oldSchemaMap[stmt.Name] = newSchemaInfo(i, stmt)
		case *ast.CreateTableStmt:
			if _, exists := oldPartitionMap[fmt.Sprintf("%s.%s", stmt.Name.Schema, stmt.Name.Name)]; exists {
				// ignore partition table
				continue
			}
			if err := diff.oldSchemaMap.addTable(i, stmt); err != nil {
				return "", err
			}
		case *ast.CreateViewStmt:
			if err := diff.oldSchemaMap.addView(i, stmt); err != nil {
				return "", err
			}
		case *ast.CreateMaterializedViewStmt:
			if err := diff.oldSchemaMap.addMaterializedView(i, stmt); err != nil {
				return "", err
			}
		case *ast.AlterTableStmt:
			// ignore partition table
			if _, exists := oldPartitionMap[fmt.Sprintf("%s.%s", stmt.Table.Schema, stmt.Table.Name)]; exists {
				continue
			}
			for _, alterItem := range stmt.AlterItemList {
				switch item := alterItem.(type) {
				case *ast.AddConstraintStmt:
					switch item.Constraint.Type {
					case ast.ConstraintTypeUnique, ast.ConstraintTypePrimary, ast.ConstraintTypeExclusion, ast.ConstraintTypeForeign:
						if err := diff.oldSchemaMap.addConstraint(i, item); err != nil {
							return "", err
						}
					default:
						return "", errors.Errorf("unsupported constraint type %d", item.Constraint.Type)
					}
				case *ast.AttachPartitionStmt:
					// ignore attach partition
					continue
				default:
					return "", errors.Errorf("unsupported alter table item type %T", item)
				}
			}
		case *ast.CreateIndexStmt:
			// ignore partition index
			if _, exists := oldPartitionMap[fmt.Sprintf("%s.%s", stmt.Index.Table.Schema, stmt.Index.Table.Name)]; exists {
				continue
			}
			if err := diff.oldSchemaMap.addIndex(i, stmt); err != nil {
				return "", err
			}
		case *ast.CreateSequenceStmt:
			if err := diff.oldSchemaMap.addSequence(i, stmt); err != nil {
				return "", err
			}
		case *ast.AlterSequenceStmt:
			// pg_dump will separate the SET OWNED BY clause into a ALTER SEQUENCE statement.
			// There would be no other ALTER SEQUENCE statements.
			if err := diff.oldSchemaMap.addSequenceOwnedBy(i, stmt); err != nil {
				return "", err
			}
		case *ast.CreateExtensionStmt:
			if err := diff.oldSchemaMap.addExtension(i, stmt); err != nil {
				return "", err
			}
		case *ast.CreateFunctionStmt:
			if err := diff.oldSchemaMap.addFunction(i, stmt); err != nil {
				return "", err
			}
		case *ast.CreateTriggerStmt:
			if err := diff.oldSchemaMap.addTrigger(i, stmt); err != nil {
				return "", err
			}
		case *ast.CreateTypeStmt:
			if err := diff.oldSchemaMap.addType(i, stmt); err != nil {
				return "", err
			}
		case *ast.CommentStmt:
			if err := diff.oldSchemaMap.addComment(stmt); err != nil {
				return "", err
			}
			// TODO(rebelice): add default back here
		}
	}

	newPartitionMap := make(map[string]bool)
	for _, partition := range newPartitions {
		newPartitionMap[partition] = true
	}
	for i, node := range newNodes {
		switch stmt := node.(type) {
		case *ast.CreateTableStmt:
			if err := diff.newSchemaMap.addTable(i, stmt); err != nil {
				return "", errors.Wrapf(err, "failed to add table %s", stmt.Name.Name)
			}
			// ignore partition table
			if _, exists := newPartitionMap[fmt.Sprintf("%s.%s", stmt.Name.Schema, stmt.Name.Name)]; exists {
				continue
			}
			if IsSystemSchema(stmt.Name.Schema) {
				continue
			}
			oldTable := diff.oldSchemaMap.getTable(stmt.Name.Schema, stmt.Name.Name)
			// Add the new table.
			if oldTable == nil {
				diff.createTableList = append(diff.createTableList, stmt)
				continue
			}
			oldTable.existsInNew = true
			// Modify the table.
			if err := diff.modifyTable(oldTable, stmt); err != nil {
				return "", err
			}
		case *ast.CreateSchemaStmt:
			diff.newSchemaMap[stmt.Name] = newSchemaInfo(i, stmt)
			if IsSystemSchema(stmt.Name) {
				continue
			}
			schema, hasSchema := diff.oldSchemaMap[stmt.Name]
			if !hasSchema {
				diff.createSchemaList = append(diff.createSchemaList, stmt)
				continue
			}
			schema.existsInNew = true
		case *ast.CreateViewStmt:
			if err := diff.newSchemaMap.addView(i, stmt); err != nil {
				return "", errors.Wrapf(err, "failed to add view %s", stmt.Name.Name)
			}
			if IsSystemSchema(stmt.Name.Schema) {
				continue
			}
			oldView := diff.oldSchemaMap.getView(stmt.Name.Schema, stmt.Name.Name)
			// Add the new view.
			if oldView == nil {
				diff.createViewList = append(diff.createViewList, stmt)
				continue
			}
			oldView.existsInNew = true
			if err := diff.ModifyView(oldView, i, stmt); err != nil {
				return "", err
			}
		case *ast.CreateMaterializedViewStmt:
			if err := diff.newSchemaMap.addMaterializedView(i, stmt); err != nil {
				return "", errors.Wrapf(err, "failed to add materialized view %s", stmt.Name.Name)
			}
			if IsSystemSchema(stmt.Name.Schema) {
				continue
			}
			oldMaterializedView := diff.oldSchemaMap.getMaterializedView(stmt.Name.Schema, stmt.Name.Name)
			// Add the new materialized view.
			if oldMaterializedView == nil {
				diff.createMaterializedViewList = append(diff.createMaterializedViewList, stmt)
				continue
			}
			oldMaterializedView.existsInNew = true
			if err := diff.ModifyMaterializedView(oldMaterializedView, i, stmt); err != nil {
				return "", err
			}
		case *ast.AlterTableStmt:
			// ignore partition table
			if _, exists := newPartitionMap[fmt.Sprintf("%s.%s", stmt.Table.Schema, stmt.Table.Name)]; exists {
				continue
			}
			if IsSystemSchema(stmt.Table.Schema) {
				continue
			}
			for _, alterItem := range stmt.AlterItemList {
				switch item := alterItem.(type) {
				case *ast.AddConstraintStmt:
					switch item.Constraint.Type {
					case ast.ConstraintTypeUnique, ast.ConstraintTypePrimary, ast.ConstraintTypeExclusion, ast.ConstraintTypeForeign:
						oldConstraint := diff.oldSchemaMap.getConstraint(item.Table.Schema, item.Table.Name, item.Constraint.Name)
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
				case *ast.AttachPartitionStmt:
					// ignore attach partition
					continue
				default:
					return "", errors.Errorf("unsupported alter table item type %T", item)
				}
			}
		case *ast.CreateIndexStmt:
			// ignore partition index
			if _, exists := newPartitionMap[fmt.Sprintf("%s.%s", stmt.Index.Table.Schema, stmt.Index.Table.Name)]; exists {
				continue
			}
			if IsSystemSchema(stmt.Index.Table.Schema) {
				continue
			}
			oldIndex := diff.oldSchemaMap.getIndex(stmt.Index.Table.Schema, stmt.Index.Name)
			// Add the new index.
			if oldIndex == nil {
				diff.createIndexList = append(diff.createIndexList, stmt)
				continue
			}
			oldIndex.existsInNew = true
			// Modify the index.
			diff.modifyIndex(oldIndex.createIndex, stmt)
		case *ast.CreateSequenceStmt:
			if IsSystemSchema(stmt.SequenceDef.SequenceName.Schema) {
				continue
			}
			oldSequence := diff.oldSchemaMap.getSequence(stmt.SequenceDef.SequenceName.Schema, stmt.SequenceDef.SequenceName.Name)
			// Add the new sequence.
			if oldSequence == nil {
				diff.createSequenceList = append(diff.createSequenceList, stmt)
				continue
			}
			oldSequence.existsInNew = true
			// Modify the sequence.
			diff.modifySequenceExceptOwnedBy(oldSequence.createSequence, stmt)
		case *ast.AlterSequenceStmt:
			if !onlySetOwnedBy(stmt) {
				return "", errors.Errorf("expect OwnedBy only, but found %v", stmt)
			}
			if IsSystemSchema(stmt.Name.Schema) {
				continue
			}
			oldSequence := diff.oldSchemaMap.getSequence(stmt.Name.Schema, stmt.Name.Name)
			// Add the new sequence owned by.
			if oldSequence == nil || oldSequence.ownedByInfo == nil {
				diff.setSequenceOwnedByList = append(diff.setSequenceOwnedByList, stmt)
				continue
			}
			oldSequence.ownedByInfo.existsInNew = true
			diff.modifySequenceOwnedBy(oldSequence.ownedByInfo.ownedBy, stmt)
		case *ast.CreateExtensionStmt:
			if IsSystemSchema(stmt.Schema) {
				continue
			}
			oldExtension := diff.oldSchemaMap.getExtension(stmt.Schema, stmt.Name)
			// Add the extension.
			if oldExtension == nil {
				diff.createExtensionList = append(diff.createExtensionList, stmt)
				continue
			}
			oldExtension.existsInNew = true
			// Modify the extension.
			diff.modifyExtension(oldExtension.createExtension, stmt)
		case *ast.CreateFunctionStmt:
			if IsSystemSchema(stmt.Function.Schema) {
				continue
			}
			signature, err := functionSignature(stmt.Function)
			if err != nil {
				return "", err
			}
			oldFunction := diff.oldSchemaMap.getFunction(stmt.Function.Schema, signature)
			// Add the function.
			if oldFunction == nil {
				diff.createFunctionList = append(diff.createFunctionList, stmt)
				continue
			}
			oldFunction.existsInNew = true
			// Modify the function.
			diff.modifyFunction(oldFunction.createFunction, stmt)
		case *ast.CreateTriggerStmt:
			if IsSystemSchema(stmt.Trigger.Table.Schema) {
				continue
			}
			oldTrigger := diff.oldSchemaMap.getTrigger(stmt.Trigger.Table.Schema, stmt.Trigger.Table.Name, stmt.Trigger.Name)
			// Add the trigger.
			if oldTrigger == nil {
				diff.createTriggerList = append(diff.createTriggerList, stmt)
				continue
			}
			oldTrigger.existsInNew = true
			// Modify the trigger.
			diff.modifyTrigger(oldTrigger.createTrigger, stmt)
		case *ast.CreateTypeStmt:
			if IsSystemSchema(stmt.Type.TypeName().Schema) {
				continue
			}
			oldType := diff.oldSchemaMap.getType(stmt.Type.TypeName().Schema, stmt.Type.TypeName().Name)
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
		case *ast.CommentStmt:
			switch stmt.Type {
			case ast.ObjectTypeTable:
				tableDef, ok := stmt.Object.(*ast.TableDef)
				if !ok {
					return "", errors.Errorf("failed to add comment: expect table def, but found %v", stmt.Object)
				}
				oldComment := diff.oldSchemaMap.getTableComment(tableDef.Schema, tableDef.Name)
				// Set the table comment.
				if oldComment == "" || oldComment != stmt.Comment {
					diff.setCommentList = append(diff.setCommentList, stmt)
				}
				diff.oldSchemaMap.removeTableComment(tableDef.Schema, tableDef.Name)
			case ast.ObjectTypeView:
				viewDef, ok := stmt.Object.(*ast.TableDef)
				if !ok {
					return "", errors.Errorf("failed to add comment: expect view def, but found %v", stmt.Object)
				}
				oldComment := diff.oldSchemaMap.getViewComment(viewDef.Schema, viewDef.Name)
				if oldComment == "" || oldComment != stmt.Comment {
					diff.setCommentList = append(diff.setCommentList, stmt)
				}
				diff.oldSchemaMap.removeViewComment(viewDef.Schema, viewDef.Name)
			case ast.ObjectTypeColumn:
				columnNameDef, ok := stmt.Object.(*ast.ColumnNameDef)
				if !ok {
					return "", errors.Errorf("failed to add comment: expect column name def, but found %v", stmt.Object)
				}
				oldComment := diff.oldSchemaMap.getColumnComment(columnNameDef.Table.Schema, columnNameDef.Table.Name, columnNameDef.ColumnName)
				// Set the column comment.
				if oldComment == "" || oldComment != stmt.Comment {
					diff.setCommentList = append(diff.setCommentList, stmt)
				}
				diff.oldSchemaMap.removeColumnComment(columnNameDef.Table.Schema, columnNameDef.Table.Name, columnNameDef.ColumnName)
			default:
				// We only support table and column comment.
			}
		}
	}

	// Drop remaining old objects.
	diff.dropObject()

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

func (diff *diffNode) dropObject() {
	// Drop the remaining old schema.
	if dropSchemaStmt := dropSchema(diff.oldSchemaMap); dropSchemaStmt != nil {
		diff.dropSchemaList = append(diff.dropSchemaList, dropSchemaStmt)
	}

	// Drop the remaining old table.
	if dropTableStmt := dropTable(diff.oldSchemaMap); dropTableStmt != nil {
		diff.dropTableList = append(diff.dropTableList, dropTableStmt...)
	}

	// Drop the remaining old view.
	if dropViewStmt := dropView(diff.oldSchemaMap); dropViewStmt != nil {
		diff.dropViewList = append(diff.dropViewList, dropViewStmt...)
	}

	// Drop the remaining old materialized view.
	if dropMaterializedViewStmt := dropMaterializedView(diff.oldSchemaMap); dropMaterializedViewStmt != nil {
		diff.dropMaterializedViewList = append(diff.dropMaterializedViewList, dropMaterializedViewStmt...)
	}

	// Drop the remaining old constraints.
	diff.dropConstraint(diff.oldSchemaMap)

	// Drop the remaining old index.
	if dropIndexStmt := dropIndex(diff.oldSchemaMap); dropIndexStmt != nil {
		diff.dropIndexList = append(diff.dropIndexList, dropIndexStmt)
	}

	// Drop the remaining old sequence owned by.
	diff.dropSequenceOwnedBy(diff.oldSchemaMap)

	// Drop the remaining old sequence.
	if dropSequenceStmt := dropSequence(diff.oldSchemaMap); dropSequenceStmt != nil {
		diff.dropSequenceList = append(diff.dropSequenceList, dropSequenceStmt)
	}

	// Drop the remaining old extension.
	if dropExtensionStmt := dropExtension(diff.oldSchemaMap); dropExtensionStmt != nil {
		diff.dropExtensionList = append(diff.dropExtensionList, dropExtensionStmt)
	}

	// Drop the remaining old function.
	if dropFunctionStmt := dropFunction(diff.oldSchemaMap); dropFunctionStmt != nil {
		diff.dropFunctionList = append(diff.dropFunctionList, dropFunctionStmt...)
	}

	// Drop the remaining old trigger.
	diff.dropTriggerStmt(diff.oldSchemaMap)

	// Drop the remaining old type.
	diff.dropTypeStmt(diff.oldSchemaMap)

	// Drop the remaining old comment.
	diff.dropComment(diff.oldSchemaMap)
}

func (diff *diffNode) modifyTableByColumn(oldTableInfo *tableInfo, newTable *ast.CreateTableStmt) error {
	oldTable := oldTableInfo.createTable
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
			// No need to drop comment for the column, because we will drop the column.
			delete(oldTableInfo.columnCommentMap, oldColumn.ColumnName)
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

func (diff *diffNode) ModifyMaterializedView(oldMaterializedView *materializedViewInfo, _ int, newMaterializedView *ast.CreateMaterializedViewStmt) error {
	if !equalMaterializedView(oldMaterializedView.createMaterializedView, newMaterializedView) {
		diff.dropMaterializedViewList = append(diff.dropMaterializedViewList, oldMaterializedView.createMaterializedView)
		// drop the comments of the view
		oldMaterializedView.comment = ""
		oldMaterializedView.columnCommentMap = make(map[string]string)
		diff.createMaterializedViewList = append(diff.createMaterializedViewList, newMaterializedView)
	}
	return nil
}

func equalMaterializedView(oldMaterializedView *ast.CreateMaterializedViewStmt, newMaterializedView *ast.CreateMaterializedViewStmt) bool {
	oldText, err := pgquery.Deparse(&pgquery.ParseResult{
		Stmts: []*pgquery.RawStmt{
			{
				Stmt: &pgquery.Node{
					Node: oldMaterializedView.GetOriginalNode(),
				},
			},
		},
	})
	if err != nil {
		slog.Debug("failed to deparse old materialized view", log.BBError(err))
		return false
	}
	newText, err := pgquery.Deparse(&pgquery.ParseResult{
		Stmts: []*pgquery.RawStmt{
			{
				Stmt: &pgquery.Node{
					Node: newMaterializedView.GetOriginalNode(),
				},
			},
		},
	})
	if err != nil {
		slog.Debug("failed to deparse new materialized view", log.BBError(err))
		return false
	}
	return oldText == newText
}

func (diff *diffNode) ModifyView(oldView *viewInfo, _ int, newView *ast.CreateViewStmt) error {
	if !equalView(oldView.createView, newView) {
		diff.dropViewList = append(diff.dropViewList, oldView.createView)
		// drop the comments of the view
		oldView.comment = ""
		oldView.columnCommentMap = make(map[string]string)
		diff.createViewList = append(diff.createViewList, newView)
	}
	return nil
}

func equalView(oldView *ast.CreateViewStmt, newView *ast.CreateViewStmt) bool {
	// oldText, err := pgquery.Deparse(oldView.ParseResult)
	// if err != nil {
	// 	slog.Debug("failed to deparse old view", log.BBError(err))
	// 	return false
	// }
	// newText, err := pgquery.Deparse(newView.ParseResult)
	// if err != nil {
	// 	slog.Debug("failed to deparse new view", log.BBError(err))
	// 	return false
	// }
	oldText, err := pgquery.Deparse(&pgquery.ParseResult{
		Stmts: []*pgquery.RawStmt{
			{
				Stmt: &pgquery.Node{
					Node: oldView.GetOriginalNode(),
				},
			},
		},
	})
	if err != nil {
		slog.Debug("failed to deparse old view", log.BBError(err))
		return false
	}
	newText, err := pgquery.Deparse(&pgquery.ParseResult{
		Stmts: []*pgquery.RawStmt{
			{
				Stmt: &pgquery.Node{
					Node: newView.GetOriginalNode(),
				},
			},
		},
	})
	if err != nil {
		slog.Debug("failed to deparse new view", log.BBError(err))
		return false
	}
	return oldText == newText
}

func (diff *diffNode) modifyTable(oldTable *tableInfo, newTable *ast.CreateTableStmt) error {
	if err := diff.modifyTableByColumn(oldTable, newTable); err != nil {
		return err
	}

	return diff.modifyTableByConstraint(oldTable.createTable, newTable)
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
		oldAlterTableAddConstraint, err := pgrawparser.Deparse(pgrawparser.DeparseContext{}, oldConstraint)
		if err != nil {
			return false, errors.Wrapf(err, "failed to deparse old alter table constraintDef: %v", oldConstraint)
		}
		newAlterTableAddConstraint, err := pgrawparser.Deparse(pgrawparser.DeparseContext{}, newConstraint)
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

func (diff *diffNode) modifyExtension(oldExtension *ast.CreateExtensionStmt, newExtension *ast.CreateExtensionStmt) {
	// TODO(rebelice): not use Text(), it only works for pg_dump.
	if oldExtension.Text() != newExtension.Text() {
		diff.dropExtensionList = append(diff.dropExtensionList, &ast.DropExtensionStmt{
			NameList: []string{oldExtension.Name},
		})
		diff.createExtensionList = append(diff.createExtensionList, newExtension)
	}
}

func (diff *diffNode) modifyFunction(oldFunction *ast.CreateFunctionStmt, newFunction *ast.CreateFunctionStmt) {
	// TODO(rebelice): not use Text(), it only works for pg_dump.
	if oldFunction.Text() != newFunction.Text() {
		diff.dropFunctionList = append(diff.dropFunctionList, oldFunction)
		diff.createFunctionList = append(diff.createFunctionList, newFunction)
	}
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
		return pgrawparser.NewConvertErrorf("expected EnumTypeDef but found %t", oldType.Type)
	}
	newEnum, ok := newType.Type.(*ast.EnumTypeDef)
	if !ok {
		// never catch
		return pgrawparser.NewConvertErrorf("expected EnumTypeDef but found %t", newType.Type)
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
	for newEnum.LabelList[firstOldLabelPos] != oldEnum.LabelList[0] {
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

func (diff *diffNode) modifyTrigger(oldTrigger *ast.CreateTriggerStmt, newTrigger *ast.CreateTriggerStmt) {
	// TODO(rebelice): not use Text(), it only works for pg_dump.
	if oldTrigger.Text() != newTrigger.Text() {
		diff.dropTriggerList = append(diff.dropTriggerList, &ast.DropTriggerStmt{
			Trigger: oldTrigger.Trigger,
		})
		diff.createTriggerList = append(diff.createTriggerList, newTrigger)
	}
}

func (diff *diffNode) modifyIndex(oldIndex *ast.CreateIndexStmt, newIndex *ast.CreateIndexStmt) {
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
}

func (diff *diffNode) modifySequenceOwnedBy(oldSequenceOwnedBy *ast.AlterSequenceStmt, newSequenceOwnedBy *ast.AlterSequenceStmt) {
	if !isEqualColumnNameDef(oldSequenceOwnedBy.OwnedBy, newSequenceOwnedBy.OwnedBy) {
		diff.setDefaultList = append(diff.setDefaultList, newSequenceOwnedBy)
	}
}

func (diff *diffNode) modifySequenceExceptOwnedBy(oldSequence *ast.CreateSequenceStmt, newSequence *ast.CreateSequenceStmt) {
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
	if !isEqualInt64Pointer(oldSequence.SequenceDef.IncrementBy, newSequence.SequenceDef.IncrementBy) {
		alterSequence.IncrementBy = newSequence.SequenceDef.IncrementBy
		isEqual = false
	}

	// compare min value
	if !isEqualInt64Pointer(oldSequence.SequenceDef.MinValue, newSequence.SequenceDef.MinValue) {
		if newSequence.SequenceDef.MinValue == nil {
			alterSequence.NoMinValue = true
		} else {
			alterSequence.MinValue = newSequence.SequenceDef.MinValue
		}
		isEqual = false
	}

	// compare max value
	if !isEqualInt64Pointer(oldSequence.SequenceDef.MaxValue, newSequence.SequenceDef.MaxValue) {
		if newSequence.SequenceDef.MaxValue == nil {
			alterSequence.NoMaxValue = true
		} else {
			alterSequence.MaxValue = newSequence.SequenceDef.MaxValue
		}
		isEqual = false
	}

	// compare start with
	if !isEqualInt64Pointer(oldSequence.SequenceDef.StartWith, newSequence.SequenceDef.StartWith) {
		if newSequence.SequenceDef.StartWith != nil {
			alterSequence.StartWith = newSequence.SequenceDef.StartWith
			isEqual = false
		}
	}

	// compare cache
	if !isEqualInt64Pointer(oldSequence.SequenceDef.Cache, newSequence.SequenceDef.Cache) {
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

func isEqualInt64Pointer(a *int64, b *int64) bool {
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
	typeStringA, err := pgrawparser.Deparse(pgrawparser.DeparseContext{}, typeA)
	if err != nil {
		return false, err
	}
	typeStringB, err := pgrawparser.Deparse(pgrawparser.DeparseContext{}, typeB)
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
		sql, err := pgrawparser.Deparse(pgrawparser.DeparseContext{}, node)
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

func printDropMaterializedView(buf io.Writer, viewName *ast.TableDef) error {
	if viewName == nil {
		return nil
	}

	if _, err := fmt.Fprint(buf, "DROP MATERIALIZED VIEW "); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buf, `"%s"."%s"`, viewName.Schema, viewName.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprint(buf, ";\n\n"); err != nil {
		return err
	}
	return nil
}

func printDropView(buf io.Writer, viewName *ast.TableDef) error {
	if viewName == nil {
		return nil
	}
	if _, err := fmt.Fprint(buf, "DROP VIEW "); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buf, `"%s"."%s"`, viewName.Schema, viewName.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprint(buf, ";\n\n"); err != nil {
		return err
	}
	return nil
}

func printDropFunction(buf io.Writer, function *ast.FunctionDef) error {
	signature, err := functionSignature(function)
	if err != nil {
		return errors.Wrapf(err, "failed to get function signature: %v", function.Name)
	}
	_, err = fmt.Fprintf(buf, "%s\n\n", signature)
	return err
}

func printDropTable(buf io.Writer, table *ast.TableDef) error {
	if _, err := fmt.Fprint(buf, "DROP TABLE "); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(buf, `"%s"."%s"`, table.Schema, table.Name); err != nil {
		return err
	}

	_, err := fmt.Fprint(buf, ";\n\n")
	return err
}

func printCreateMaterializedViewStmt(buf io.Writer, view *ast.CreateMaterializedViewStmt) error {
	node := view.GetOriginalNode()
	text, err := pgquery.Deparse(&pgquery.ParseResult{
		Stmts: []*pgquery.RawStmt{
			{
				Stmt: &pgquery.Node{Node: node},
			},
		},
	})
	if err != nil {
		return err
	}
	err = writeStringWithNewLine(buf, text+";")
	return err
}

func printCreateViewStmt(buf io.Writer, view *ast.CreateViewStmt) error {
	node := view.GetOriginalNode()
	node.ViewStmt.Replace = true
	text, err := pgquery.Deparse(&pgquery.ParseResult{
		Stmts: []*pgquery.RawStmt{
			{
				Stmt: &pgquery.Node{Node: node},
			},
		},
	})
	if err != nil {
		return err
	}
	err = writeStringWithNewLine(buf, text+";")
	return err
}

func printCreateFunctionStmt(buf io.Writer, function *ast.CreateFunctionStmt) error {
	if err := writeStringWithNewLine(buf, function.Text()); err != nil {
		return err
	}
	_, err := buf.Write([]byte("\n"))
	return err
}

func printCreateTableStmt(buf io.Writer, table *ast.CreateTableStmt) error {
	sql, err := pgrawparser.Deparse(pgrawparser.DeparseContext{}, table)
	if err != nil {
		return err
	}
	if err := writeStringWithNewLine(buf, sql); err != nil {
		return err
	}
	_, err = buf.Write([]byte("\n"))
	return err
}

// deparse statements as the safe change order.
func (diff *diffNode) deparse() (string, error) {
	var buf bytes.Buffer

	// Print header.
	if len(diff.createFunctionList) > 0 {
		if _, err := buf.WriteString("SET check_function_bodies = false;\n\n"); err != nil {
			return "", err
		}
	}

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
	if err := diff.printDropFuncTableViewAndMateView(&buf); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropColumnList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.dropSequenceList); err != nil {
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
	if err := printStmtSlice(&buf, diff.createSequenceList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.alterSequenceExceptOwnedByList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.createColumnList); err != nil {
		return "", err
	}
	if err := printStmtSlice(&buf, diff.alterColumnList); err != nil {
		return "", err
	}

	if err := diff.printCreateFuncTableViewAndMateView(&buf); err != nil {
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
	if err := printStmtSlice(&buf, diff.setCommentList); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func getObjectID(schema, object string) string {
	return fmt.Sprintf("%s.%s", schema, object)
}

func (diff *diffNode) getNewDatabaseMetadataFunc() base.GetDatabaseMetadataFunc {
	return func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
		return "", model.NewDatabaseMetadata(schemaMapToMetadata(diff.newSchemaMap), true /* isObjectCaseSensitive */, true /* isDetailCaseSensitive */), nil
	}
}

func (diff *diffNode) getOldDatabaseMetadataFunc() base.GetDatabaseMetadataFunc {
	return func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
		return "", model.NewDatabaseMetadata(schemaMapToMetadata(diff.oldSchemaMap), true /* isObjectCaseSensitive */, true /* isDetailCaseSensitive */), nil
	}
}

func schemaMapToMetadata(m schemaMap) *storepb.DatabaseSchemaMetadata {
	meta := storepb.DatabaseSchemaMetadata{}
	for _, schema := range m {
		// We only need to add the views and materialized views to the schema metadata.
		// This is only use to get the resources column from query span.
		meta.Schemas = append(meta.Schemas, &storepb.SchemaMetadata{
			Name:              schema.createSchema.Name,
			Views:             viewToMetadata(schema.viewMap),
			MaterializedViews: materializedViewToMetadata(schema.materializedViewMap),
		})
	}

	return &meta
}

func materializedViewToMetadata(m materializedViewMap) []*storepb.MaterializedViewMetadata {
	var materializedViews []*storepb.MaterializedViewMetadata
	for _, materializedView := range m {
		materializedViews = append(materializedViews, &storepb.MaterializedViewMetadata{
			Name: materializedView.createMaterializedView.Name.Name,
		})
	}
	return materializedViews
}

func viewToMetadata(m viewMap) []*storepb.ViewMetadata {
	var views []*storepb.ViewMetadata
	for _, view := range m {
		views = append(views, &storepb.ViewMetadata{
			Name: view.createView.Name.Name,
		})
	}
	return views
}

func (diff *diffNode) printCreateFuncTableViewAndMateView(buf io.Writer) error {
	graph := base.NewGraph()
	functionMap := make(map[string]*ast.CreateFunctionStmt)
	tableMap := make(map[string]*ast.CreateTableStmt)
	viewMap := make(map[string]*ast.CreateViewStmt)
	materializedViewMap := make(map[string]*ast.CreateMaterializedViewStmt)

	for _, function := range diff.createFunctionList {
		signature, err := functionSignature(function.Function)
		if err != nil {
			return errors.Wrapf(err, "failed to get function signature for %s", function.Function.Name)
		}
		funcID := getObjectID(function.Function.Schema, signature)
		functionMap[funcID] = function
		graph.AddNode(funcID)
		tSchema, tName := getFunctionReturnType(function)
		graph.AddEdge(getObjectID(tSchema, tName), funcID)
	}

	for _, table := range diff.createTableList {
		tableID := getObjectID(table.Name.Schema, table.Name.Name)
		tableMap[tableID] = table
		graph.AddNode(tableID)
	}

	for _, view := range diff.createViewList {
		viewID := getObjectID(view.Name.Schema, view.Name.Name)
		viewMap[viewID] = view
		graph.AddNode(viewID)
		dependency, err := diff.getViewDependency(view, diff.getNewDatabaseMetadataFunc())
		if err != nil {
			return errors.Wrapf(err, "failed to get view dependency for %s", viewID)
		}
		for _, dep := range dependency {
			graph.AddEdge(dep, viewID)
		}
	}

	for _, materializedView := range diff.createMaterializedViewList {
		viewID := getObjectID(materializedView.Name.Schema, materializedView.Name.Name)
		materializedViewMap[viewID] = materializedView
		graph.AddNode(viewID)
		dependency, err := diff.getMaterializedViewDependency(materializedView, diff.getNewDatabaseMetadataFunc())
		if err != nil {
			return errors.Wrapf(err, "failed to get view dependency for %s", viewID)
		}
		for _, dep := range dependency {
			graph.AddEdge(dep, viewID)
		}
	}

	orderedList, err := graph.TopologicalSort()
	if err != nil {
		return errors.Wrap(err, "failed to topological sort")
	}

	for _, objectID := range orderedList {
		if function, ok := functionMap[objectID]; ok {
			if err := printCreateFunctionStmt(buf, function); err != nil {
				return err
			}
			continue
		}

		if table, ok := tableMap[objectID]; ok {
			if err := printCreateTableStmt(buf, table); err != nil {
				return err
			}
			continue
		}

		if view, ok := viewMap[objectID]; ok {
			if err := printCreateViewStmt(buf, view); err != nil {
				return err
			}
			continue
		}

		if materializedView, ok := materializedViewMap[objectID]; ok {
			if err := printCreateMaterializedViewStmt(buf, materializedView); err != nil {
				return err
			}
		}
	}

	return nil
}

func getFunctionReturnType(function *ast.CreateFunctionStmt) (string, string) {
	node := function.GetOriginalNode()
	returnType := node.CreateFunctionStmt.ReturnType
	if returnType == nil {
		return "", ""
	}

	nameList := convertNodeSliceToStringList(returnType.Names)
	switch len(nameList) {
	case 1:
		return "public", nameList[0]
	case 2:
		return nameList[0], nameList[1]
	default:
		return "", ""
	}
}

func convertNodeSliceToStringList(nodeList []*pgquery.Node) []string {
	var result []string
	for _, node := range nodeList {
		if stringNode, ok := node.Node.(*pgquery.Node_String_); ok {
			result = append(result, stringNode.String_.GetSval())
		}
	}
	return result
}

func (diff *diffNode) printDropFuncTableViewAndMateView(buf io.Writer) error {
	graph := base.NewGraph()
	functionMap := make(map[string]*ast.CreateFunctionStmt)
	tableMap := make(map[string]*ast.CreateTableStmt)
	viewMap := make(map[string]*ast.CreateViewStmt)
	materializedViewMap := make(map[string]*ast.CreateMaterializedViewStmt)

	for _, function := range diff.dropFunctionList {
		signature, err := functionSignature(function.Function)
		if err != nil {
			return errors.Wrapf(err, "failed to get function signature for %s", function.Function.Name)
		}
		funcID := getObjectID(function.Function.Schema, signature)
		functionMap[funcID] = function
		graph.AddNode(funcID)
		tSchema, tName := getFunctionReturnType(function)
		graph.AddEdge(funcID, getObjectID(tSchema, tName))
	}

	for _, table := range diff.dropTableList {
		tableID := getObjectID(table.Name.Schema, table.Name.Name)
		tableMap[tableID] = table
		graph.AddNode(tableID)
	}

	for _, view := range diff.dropViewList {
		viewID := getObjectID(view.Name.Schema, view.Name.Name)
		viewMap[viewID] = view
		graph.AddNode(viewID)
		dependency, err := diff.getViewDependency(view, diff.getOldDatabaseMetadataFunc())
		if err != nil {
			return errors.Wrapf(err, "failed to get view dependency for %s", viewID)
		}
		for _, dep := range dependency {
			graph.AddEdge(viewID, dep)
		}
	}

	for _, materializedView := range diff.dropMaterializedViewList {
		viewID := getObjectID(materializedView.Name.Schema, materializedView.Name.Name)
		materializedViewMap[viewID] = materializedView
		graph.AddNode(viewID)
		dependency, err := diff.getMaterializedViewDependency(materializedView, diff.getOldDatabaseMetadataFunc())
		if err != nil {
			return errors.Wrapf(err, "failed to get view dependency for %s", viewID)
		}
		for _, dep := range dependency {
			graph.AddEdge(viewID, dep)
		}
	}

	orderedList, err := graph.TopologicalSort()
	if err != nil {
		return errors.Wrap(err, "failed to topological sort")
	}

	for _, objectID := range orderedList {
		if function, ok := functionMap[objectID]; ok {
			if err := printDropFunction(buf, function.Function); err != nil {
				return err
			}
			continue
		}

		if table, ok := tableMap[objectID]; ok {
			if err := printDropTable(buf, table.Name); err != nil {
				return err
			}
			continue
		}

		if view, ok := viewMap[objectID]; ok {
			if err := printDropView(buf, view.Name); err != nil {
				return err
			}
			continue
		}

		if materializedView, ok := materializedViewMap[objectID]; ok {
			if err := printDropMaterializedView(buf, materializedView.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

func (*diffNode) getMaterializedViewDependency(materializedView *ast.CreateMaterializedViewStmt, getDatabaseMetadata base.GetDatabaseMetadataFunc) ([]string, error) {
	queryText, err := pgquery.Deparse(&pgquery.ParseResult{
		Stmts: []*pgquery.RawStmt{
			{
				Stmt: &pgquery.Node{
					Node: materializedView.GetOriginalNode().CreateTableAsStmt.Query.Node,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	span, err := GetQuerySpan(
		context.Background(),
		base.GetQuerySpanContext{
			GetDatabaseMetadataFunc: getDatabaseMetadata,
		}, // We only need the source column, so resource is not needed.
		queryText,
		"", // No need database name.
		"public",
		false,
	)
	if err != nil {
		return nil, err
	}

	var result []string
	for sourceColumn := range span.SourceColumns {
		result = append(result, getObjectID(sourceColumn.Schema, sourceColumn.Table))
	}
	return result, nil
}

func (*diffNode) getViewDependency(view *ast.CreateViewStmt, getDatabaseMetadata base.GetDatabaseMetadataFunc) ([]string, error) {
	queryText, err := pgquery.Deparse(&pgquery.ParseResult{
		Stmts: []*pgquery.RawStmt{
			{
				Stmt: &pgquery.Node{
					Node: view.GetOriginalNode().ViewStmt.Query.Node,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	span, err := GetQuerySpan(
		context.Background(),
		base.GetQuerySpanContext{
			GetDatabaseMetadataFunc: base.GetDatabaseMetadataFunc(getDatabaseMetadata),
		}, // We only need the source column, so resource is not needed.
		queryText,
		"", // No need database name.
		"public",
		false,
	)
	if err != nil {
		return nil, err
	}

	var result []string
	for sourceColumn := range span.SourceColumns {
		result = append(result, getObjectID(sourceColumn.Schema, sourceColumn.Table))
	}
	return result, nil
}

func (diff *diffNode) dropConstraint(m schemaMap) {
	for _, schemaInfo := range m {
		if !schemaInfo.existsInNew {
			continue
		}
		for _, tableInfo := range schemaInfo.tableMap {
			if !tableInfo.existsInNew {
				continue
			}
			var constraintInfoList []*constraintInfo
			for _, constraintInfo := range tableInfo.constraintMap {
				if !constraintInfo.existsInNew {
					constraintInfoList = append(constraintInfoList, constraintInfo)
				}
			}
			slices.SortFunc(constraintInfoList, func(i, j *constraintInfo) int {
				if i.id < j.id {
					return -1
				}
				if i.id > j.id {
					return 1
				}
				return 0
			})
			var dropConstraintList []*ast.ConstraintDef
			for _, constraintInfo := range constraintInfoList {
				dropConstraintList = append(dropConstraintList, constraintInfo.addConstraint.Constraint)
			}

			diff.appendDropConstraint(tableInfo.createTable.Name, dropConstraintList)
		}
	}
}

func dropMaterializedView(m schemaMap) []*ast.CreateMaterializedViewStmt {
	var materializedViewList []*materializedViewInfo
	for _, schema := range m {
		for _, materializedView := range schema.materializedViewMap {
			if materializedView.existsInNew {
				// no need to drop
				continue
			}
			materializedViewList = append(materializedViewList, materializedView)
		}
	}
	if len(materializedViewList) == 0 {
		return nil
	}
	slices.SortFunc(materializedViewList, func(i, j *materializedViewInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})

	var materializedViewDefList []*ast.CreateMaterializedViewStmt
	for _, materializedView := range materializedViewList {
		materializedViewDefList = append(materializedViewDefList, materializedView.createMaterializedView)
	}
	return materializedViewDefList
}

func dropView(m schemaMap) []*ast.CreateViewStmt {
	var viewList []*viewInfo
	for _, schema := range m {
		for _, view := range schema.viewMap {
			if view.existsInNew {
				// no need to drop
				continue
			}
			viewList = append(viewList, view)
		}
	}
	if len(viewList) == 0 {
		return nil
	}
	slices.SortFunc(viewList, func(i, j *viewInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})

	var viewDefList []*ast.CreateViewStmt
	for _, view := range viewList {
		viewDefList = append(viewDefList, view.createView)
	}
	return viewDefList
}

func dropTable(m schemaMap) []*ast.CreateTableStmt {
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
	slices.SortFunc(tableList, func(i, j *tableInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})

	var result []*ast.CreateTableStmt
	for _, table := range tableList {
		result = append(result, table.createTable)
	}
	return result
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
	slices.SortFunc(schemaList, func(i, j *schemaInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
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
	slices.SortFunc(extensionList, func(i, j *extensionInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})

	var extensionNameList []string
	for _, extension := range extensionList {
		extensionNameList = append(extensionNameList, extension.createExtension.Name)
	}
	return &ast.DropExtensionStmt{
		NameList: extensionNameList,
	}
}

func dropFunction(m schemaMap) []*ast.CreateFunctionStmt {
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
	slices.SortFunc(functionList, func(i, j *functionInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})

	var result []*ast.CreateFunctionStmt
	for _, function := range functionList {
		result = append(result, function.createFunction)
	}
	return result
}

func (diff *diffNode) dropComment(m schemaMap) {
	var tables []*tableInfo
	var views []*viewInfo
	for _, schema := range m {
		for _, table := range schema.tableMap {
			if !table.existsInNew {
				// no need to drop, because the comments are dropped when drop table
				continue
			}
			tables = append(tables, table)
		}
		for _, view := range schema.viewMap {
			if !view.existsInNew {
				// no need to drop, because the comments are dropped when drop view
				continue
			}
			views = append(views, view)
		}
	}
	slices.SortFunc(tables, func(i, j *tableInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})

	for _, table := range tables {
		if table.comment != "" {
			diff.setCommentList = append(diff.setCommentList, &ast.CommentStmt{
				Comment: "",
				Object:  &ast.TableDef{Schema: table.createTable.Name.Schema, Name: table.createTable.Name.Name},
				Type:    ast.ObjectTypeTable,
			})
		}

		type commentInfo struct {
			column  string
			comment string
		}
		var columnCommentList []commentInfo
		for k, v := range table.columnCommentMap {
			columnCommentList = append(columnCommentList, commentInfo{
				column:  k,
				comment: v,
			})
		}
		slices.SortFunc(columnCommentList, func(i, j commentInfo) int {
			if i.column < j.column {
				return -1
			}
			if i.column > j.column {
				return 1
			}
			return 0
		})
		for _, columnComment := range columnCommentList {
			diff.setCommentList = append(diff.setCommentList, &ast.CommentStmt{
				Comment: "",
				Object: &ast.ColumnNameDef{
					Table:      &ast.TableDef{Schema: table.createTable.Name.Schema, Name: table.createTable.Name.Name},
					ColumnName: columnComment.column,
				},
				Type: ast.ObjectTypeColumn,
			})
		}
	}

	slices.SortFunc(views, func(i, j *viewInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})

	for _, view := range views {
		if view.comment != "" {
			diff.setCommentList = append(diff.setCommentList, &ast.CommentStmt{
				Comment: "",
				Object:  &ast.TableDef{Schema: view.createView.Name.Schema, Name: view.createView.Name.Name},
				Type:    ast.ObjectTypeView,
			})
		}

		type commentInfo struct {
			column  string
			comment string
		}
		var columnCommentList []commentInfo
		for k, v := range view.columnCommentMap {
			columnCommentList = append(columnCommentList, commentInfo{
				column:  k,
				comment: v,
			})
		}
		slices.SortFunc(columnCommentList, func(i, j commentInfo) int {
			if i.column < j.column {
				return -1
			}
			if i.column > j.column {
				return 1
			}
			return 0
		})
		for _, columnComment := range columnCommentList {
			diff.setCommentList = append(diff.setCommentList, &ast.CommentStmt{
				Comment: "",
				Object: &ast.ColumnNameDef{
					Table:      &ast.TableDef{Schema: view.createView.Name.Schema, Name: view.createView.Name.Name},
					ColumnName: columnComment.column,
				},
				Type: ast.ObjectTypeColumn,
			})
		}
	}
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
	slices.SortFunc(typeList, func(i, j *typeInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
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
	slices.SortFunc(triggerList, func(i, j *triggerInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})

	for _, trigger := range triggerList {
		diff.dropTriggerList = append(diff.dropTriggerList,
			&ast.DropTriggerStmt{Trigger: trigger.createTrigger.Trigger})
	}
}

func dropIndex(m schemaMap) *ast.DropIndexStmt {
	var indexList []*indexInfo
	for _, schema := range m {
		if !schema.existsInNew {
			continue
		}
		for _, index := range schema.indexMap {
			if index.existsInNew {
				// no need to drop
				continue
			}
			tbl, ok := schema.tableMap[index.createIndex.Index.Table.Name]
			if !ok {
				continue
			}
			if !tbl.existsInNew {
				continue
			}
			indexList = append(indexList, index)
		}
	}
	if len(indexList) == 0 {
		return nil
	}
	slices.SortFunc(indexList, func(i, j *indexInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
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
	slices.SortFunc(sequenceOwnedByList, func(i, j *sequenceOwnedByInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
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
	slices.SortFunc(sequenceList, func(i, j *sequenceInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
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
	return pgrawparser.Deparse(pgrawparser.DeparseContext{}, &ast.DropFunctionStmt{
		FunctionList: []*ast.FunctionDef{function},
	})
}
