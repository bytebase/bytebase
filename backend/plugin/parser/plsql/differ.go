// Package plsql provides the plsql parser plugin.
package plsql

import (
	"fmt"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	plsql "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterSchemaDiffFunc(storepb.Engine_ORACLE, SchemaDiff)
	base.RegisterSchemaDiffFunc(storepb.Engine_OCEANBASE_ORACLE, SchemaDiff)
}

type diffNode struct {
	schemaName     string
	dropConstraint []string
	dropIndex      []string
	dropColumn     []string
	dropTable      []string
	createTable    []string
	addColumn      []string
	modifyColumn   []string
	addIndex       []string
	addConstraint  []string
}

func (diff *diffNode) String() (string, error) {
	var buf strings.Builder
	for _, dropConstraint := range diff.dropConstraint {
		if _, err := buf.WriteString(dropConstraint); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("\n"); err != nil {
			return "", err
		}
	}
	for _, dropIndex := range diff.dropIndex {
		if _, err := buf.WriteString(dropIndex); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("\n"); err != nil {
			return "", err
		}
	}
	for _, dropColumn := range diff.dropColumn {
		if _, err := buf.WriteString(dropColumn); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("\n"); err != nil {
			return "", err
		}
	}
	for _, dropTable := range diff.dropTable {
		if _, err := buf.WriteString(dropTable); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("\n"); err != nil {
			return "", err
		}
	}
	for _, createTable := range diff.createTable {
		if _, err := buf.WriteString(createTable); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("\n"); err != nil {
			return "", err
		}
	}
	for _, addColumn := range diff.addColumn {
		if _, err := buf.WriteString(addColumn); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("\n"); err != nil {
			return "", err
		}
	}
	for _, modifyColumn := range diff.modifyColumn {
		if _, err := buf.WriteString(modifyColumn); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("\n"); err != nil {
			return "", err
		}
	}
	for _, addIndex := range diff.addIndex {
		if _, err := buf.WriteString(addIndex); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("\n"); err != nil {
			return "", err
		}
	}
	for _, addConstraint := range diff.addConstraint {
		if _, err := buf.WriteString(addConstraint); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("\n"); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

// SchemaDiff implements the differ.SchemaDiffer interface.
func SchemaDiff(oldStmt, newStmt string, _ bool) (string, error) {
	oldSchemaInfo, err := buildSchemaInfo(oldStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to build schema info for old statement")
	}
	newSchemaInfo, err := buildSchemaInfo(newStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to build schema info for new statement")
	}

	diff := &diffNode{
		schemaName: oldSchemaInfo.name,
	}
	var newTables []*tableInfo
	for _, table := range newSchemaInfo.tableMap {
		newTables = append(newTables, table)
	}
	sort.Slice(newTables, func(i, j int) bool {
		return newTables[i].id < newTables[j].id
	})
	for _, newTable := range newTables {
		tableName := newTable.name
		oldTable, exists := oldSchemaInfo.tableMap[tableName]
		if !exists {
			diff.createTable = append(diff.createTable, newTable.createTable.GetParser().GetTokenStream().GetTextFromRuleContext(newTable.createTable))
			continue
		}
		if err := diff.diffTable(oldTable, newTable); err != nil {
			return "", err
		}
		delete(oldSchemaInfo.tableMap, tableName)
	}

	var remainingTables []*tableInfo
	for _, table := range oldSchemaInfo.tableMap {
		remainingTables = append(remainingTables, table)
	}
	sort.Slice(remainingTables, func(i, j int) bool {
		return remainingTables[i].id < remainingTables[j].id
	})
	for _, table := range remainingTables {
		diff.dropTable = append(diff.dropTable, fmt.Sprintf(`DROP TABLE "%s"."%s";`, oldSchemaInfo.name, table.name))
	}

	var newIndexes []*indexInfo
	for _, index := range newSchemaInfo.indexMap {
		newIndexes = append(newIndexes, index)
	}
	sort.Slice(newIndexes, func(i, j int) bool {
		return newIndexes[i].id < newIndexes[j].id
	})
	for _, newIndex := range newIndexes {
		indexName := newIndex.name
		oldIndex, exists := oldSchemaInfo.indexMap[indexName]
		if !exists {
			diff.addIndex = append(diff.addIndex, newIndex.createIndex.GetParser().GetTokenStream().GetTextFromRuleContext(newIndex.createIndex))
			continue
		}
		if err := diff.diffIndex(oldIndex, newIndex); err != nil {
			return "", err
		}
		delete(oldSchemaInfo.indexMap, indexName)
	}

	var remainingIndexes []*indexInfo
	for _, index := range oldSchemaInfo.indexMap {
		remainingIndexes = append(remainingIndexes, index)
	}
	sort.Slice(remainingIndexes, func(i, j int) bool {
		return remainingIndexes[i].id < remainingIndexes[j].id
	})
	for _, index := range remainingIndexes {
		diff.dropIndex = append(diff.dropIndex, fmt.Sprintf(`DROP INDEX "%s"."%s";`, oldSchemaInfo.name, index.name))
	}

	return diff.String()
}

func (diff *diffNode) diffIndex(oldIndex, newIndex *indexInfo) error {
	// TODO: compare index definition instead of text.
	oldString := oldIndex.createIndex.GetParser().GetTokenStream().GetTextFromRuleContext(oldIndex.createIndex)
	newString := newIndex.createIndex.GetParser().GetTokenStream().GetTextFromRuleContext(newIndex.createIndex)
	if oldString != newString {
		diff.dropIndex = append(diff.dropIndex, fmt.Sprintf(`DROP INDEX "%s"."%s";`, diff.schemaName, oldIndex.name))
		diff.addIndex = append(diff.addIndex, newIndex.createIndex.GetParser().GetTokenStream().GetTextFromRuleContext(newIndex.createIndex))
	}
	return nil
}

func (diff *diffNode) diffTable(oldTable, newTable *tableInfo) error {
	if err := diff.diffColumn(oldTable, newTable); err != nil {
		return err
	}
	return diff.diffConstraint(oldTable, newTable)
}

func buildConstraintMap(table *tableInfo) map[string]plsql.IRelational_propertyContext {
	constraintMap := make(map[string]plsql.IRelational_propertyContext)
	if table.createTable.Relational_table() == nil {
		return constraintMap
	}
	for _, item := range table.createTable.Relational_table().AllRelational_property() {
		switch {
		case item.Out_of_line_constraint() != nil:
			constraint := item.Out_of_line_constraint()
			if constraint.Constraint_name() == nil {
				continue
			}
			_, constraintName := NormalizeConstraintName(constraint.Constraint_name())
			if constraintName == "" {
				continue
			}
			constraintMap[constraintName] = item
		case item.Out_of_line_ref_constraint() != nil:
			constraint := item.Out_of_line_ref_constraint()
			if constraint.Constraint_name() == nil {
				continue
			}
			_, constraintName := NormalizeConstraintName(constraint.Constraint_name())
			if constraintName == "" {
				continue
			}
			constraintMap[constraintName] = item
		}
	}
	return constraintMap
}

func (diff *diffNode) diffConstraint(oldTable, newTable *tableInfo) error {
	if newTable.createTable.Relational_table() == nil {
		// TODO: support object_table and xmltype_table
		return nil
	}

	var addConstraints []plsql.IRelational_propertyContext
	var dropConstraints []string
	oldConstraintMap := buildConstraintMap(oldTable)
	for _, item := range newTable.createTable.Relational_table().AllRelational_property() {
		switch {
		case item.Out_of_line_constraint() != nil:
			constraint := item.Out_of_line_constraint()
			if constraint.Constraint_name() == nil {
				continue
			}
			_, constraintName := NormalizeConstraintName(constraint.Constraint_name())
			if constraintName == "" {
				continue
			}
			oldConstraint, ok := oldConstraintMap[constraintName]
			if !ok {
				addConstraints = append(addConstraints, item)
				continue
			}
			// Compare the constraint definition.
			if !isConstraintEqual(oldConstraint, item) {
				addConstraints = append(addConstraints, item)
				dropConstraints = append(dropConstraints, constraintName)
			}
			delete(oldConstraintMap, constraintName)
		case item.Out_of_line_ref_constraint() != nil:
			constraint := item.Out_of_line_ref_constraint()
			if constraint.Constraint_name() == nil {
				continue
			}
			_, constraintName := NormalizeConstraintName(constraint.Constraint_name())
			if constraintName == "" {
				continue
			}
			oldConstraint, ok := oldConstraintMap[constraintName]
			if !ok {
				addConstraints = append(addConstraints, item)
				continue
			}
			// Compare the constraint definition.
			if !isConstraintEqual(oldConstraint, item) {
				addConstraints = append(addConstraints, item)
				dropConstraints = append(dropConstraints, constraintName)
			}
			delete(oldConstraintMap, constraintName)
		}
	}
	for constraintName := range oldConstraintMap {
		dropConstraints = append(dropConstraints, constraintName)
	}

	sort.Slice(dropConstraints, func(i, j int) bool {
		return dropConstraints[i] < dropConstraints[j]
	})

	return diff.appendConstraintDiff(newTable.name, addConstraints, dropConstraints)
}

func (diff *diffNode) appendConstraintDiff(tableName string, addConstraints []plsql.IRelational_propertyContext, dropConstraints []string) error {
	for _, constraint := range addConstraints {
		if err := diff.appendAddConstraint(tableName, constraint); err != nil {
			return err
		}
	}
	for _, constraint := range dropConstraints {
		if err := diff.appendDropConstraint(tableName, constraint); err != nil {
			return err
		}
	}
	return nil
}

func (diff *diffNode) appendDropConstraint(tableName string, constraint string) error {
	var buf strings.Builder

	if _, err := buf.WriteString(`ALTER TABLE "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(diff.schemaName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"."`); err != nil {
		return err
	}
	if _, err := buf.WriteString(tableName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" DROP CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(diff.schemaName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"."`); err != nil {
		return err
	}
	if _, err := buf.WriteString(constraint); err != nil {
		return err
	}
	if _, err := buf.WriteString("\";"); err != nil {
		return err
	}
	diff.dropConstraint = append(diff.dropConstraint, buf.String())
	return nil
}

func (diff *diffNode) appendAddConstraint(tableName string, constraint plsql.IRelational_propertyContext) error {
	var buf strings.Builder

	if _, err := buf.WriteString(`ALTER TABLE "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(diff.schemaName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"."`); err != nil {
		return err
	}
	if _, err := buf.WriteString(tableName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" ADD `); err != nil {
		return err
	}
	if _, err := buf.WriteString(constraint.GetParser().GetTokenStream().GetTextFromRuleContext(constraint)); err != nil {
		return err
	}
	if _, err := buf.WriteString(";"); err != nil {
		return err
	}
	diff.addConstraint = append(diff.addConstraint, buf.String())
	return nil
}

func isConstraintEqual(oldConstraint, newConstraint plsql.IRelational_propertyContext) bool {
	// TODO: compare constraint definition instead of text.
	oldString := oldConstraint.GetParser().GetTokenStream().GetTextFromRuleContext(oldConstraint)
	newString := newConstraint.GetParser().GetTokenStream().GetTextFromRuleContext(newConstraint)
	return oldString == newString
}

func buildColumnMap(table *tableInfo) map[string]plsql.IColumn_definitionContext {
	columnMap := make(map[string]plsql.IColumn_definitionContext)
	if table.createTable.Relational_table() == nil {
		return columnMap
	}
	for _, item := range table.createTable.Relational_table().AllRelational_property() {
		if item.Column_definition() == nil {
			continue
		}
		columnName := NormalizeIdentifierContext(item.Column_definition().Column_name().Identifier())
		columnMap[columnName] = item.Column_definition()
	}
	return columnMap
}

func (diff *diffNode) diffColumn(oldTable, newTable *tableInfo) error {
	if newTable.createTable.Relational_table() == nil {
		// TODO: support object_table and xmltype_table
		return nil
	}

	var addColumns []plsql.IColumn_definitionContext
	var modifyColumns []plsql.IColumn_definitionContext
	var dropColumns []string
	oldColumnMap := buildColumnMap(oldTable)
	for _, item := range newTable.createTable.Relational_table().AllRelational_property() {
		if item.Column_definition() == nil {
			continue
		}

		newColumnName := NormalizeIdentifierContext(item.Column_definition().Column_name().Identifier())
		oldColumn, ok := oldColumnMap[newColumnName]
		if !ok {
			addColumns = append(addColumns, item.Column_definition())
			continue
		}

		// Compare the column definition.
		if !isColumnEqual(oldColumn, item.Column_definition()) {
			modifyColumns = append(modifyColumns, item.Column_definition())
		}
		delete(oldColumnMap, newColumnName)
	}
	for _, column := range oldColumnMap {
		dropColumns = append(dropColumns, NormalizeIdentifierContext(column.Column_name().Identifier()))
	}

	sort.Slice(dropColumns, func(i, j int) bool {
		return dropColumns[i] < dropColumns[j]
	})

	return diff.appendColumnDiff(oldTable.name, addColumns, modifyColumns, dropColumns)
}

func (diff *diffNode) appendColumnDiff(tableName string, addColumns []plsql.IColumn_definitionContext, modifyColumns []plsql.IColumn_definitionContext, dropColumns []string) error {
	if len(addColumns) != 0 {
		if err := diff.appendAddColumn(tableName, addColumns); err != nil {
			return err
		}
	}
	if len(modifyColumns) != 0 {
		if err := diff.appendModifyColumn(tableName, modifyColumns); err != nil {
			return err
		}
	}
	if len(dropColumns) != 0 {
		if err := diff.appendDropColumn(tableName, dropColumns); err != nil {
			return err
		}
	}
	return nil
}

func (diff *diffNode) appendDropColumn(tableName string, dropColumns []string) error {
	var buf strings.Builder

	if _, err := buf.WriteString(`ALTER TABLE "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(diff.schemaName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"."`); err != nil {
		return err
	}
	if _, err := buf.WriteString(tableName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" DROP (`); err != nil {
		return err
	}
	for i, column := range dropColumns {
		if i != 0 {
			if _, err := buf.WriteString(`,`); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("\n	"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n);"); err != nil {
		return err
	}
	diff.dropColumn = append(diff.dropColumn, buf.String())
	return nil
}

func (diff *diffNode) appendModifyColumn(tableName string, modifyColumns []plsql.IColumn_definitionContext) error {
	var buf strings.Builder

	if _, err := buf.WriteString(`ALTER TABLE "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(diff.schemaName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"."`); err != nil {
		return err
	}
	if _, err := buf.WriteString(tableName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" MODIFY (`); err != nil {
		return err
	}
	for i, column := range modifyColumns {
		if i != 0 {
			if _, err := buf.WriteString(`,`); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("\n	"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column.GetParser().GetTokenStream().GetTextFromRuleContext(column)); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n);"); err != nil {
		return err
	}
	diff.modifyColumn = append(diff.modifyColumn, buf.String())
	return nil
}

func (diff *diffNode) appendAddColumn(tableName string, addColumns []plsql.IColumn_definitionContext) error {
	var buf strings.Builder

	if _, err := buf.WriteString(`ALTER TABLE "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(diff.schemaName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"."`); err != nil {
		return err
	}
	if _, err := buf.WriteString(tableName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" ADD (`); err != nil {
		return err
	}
	for i, column := range addColumns {
		if i != 0 {
			if _, err := buf.WriteString(`,`); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("\n	"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column.GetParser().GetTokenStream().GetTextFromRuleContext(column)); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n);"); err != nil {
		return err
	}
	diff.addColumn = append(diff.addColumn, buf.String())
	return nil
}

func isColumnEqual(oldColumn, newColumn plsql.IColumn_definitionContext) bool {
	// TODO: compare column definition instead of text.
	oldString := oldColumn.GetParser().GetTokenStream().GetTextFromRuleContext(oldColumn)
	newString := newColumn.GetParser().GetTokenStream().GetTextFromRuleContext(newColumn)
	return oldString == newString
}

func buildSchemaInfo(statement string) (*schemaInfo, error) {
	node, _, err := ParsePLSQL(statement)
	if err != nil {
		return nil, err
	}

	listener := &buildSchemaInfoListener{
		schemaInfo: &schemaInfo{
			name:     "",
			tableMap: make(tableMap),
			indexMap: make(indexMap),
		},
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, node)
	if listener.err != nil {
		if strings.Contains(listener.err.Error(), "schema name mismatch") {
			return nil, errors.Wrapf(listener.err, "Oracle sync schema only supports single schema, please use \"Manage based on schema\" sync mode for this operation")
		}
		return nil, listener.err
	}
	return listener.schemaInfo, nil
}

type buildSchemaInfoListener struct {
	*plsql.BasePlSqlParserListener

	schemaInfo *schemaInfo
	err        error
}

// EnterCreate_table is called when production create_table is entered.
func (l *buildSchemaInfoListener) EnterCreate_table(ctx *plsql.Create_tableContext) {
	if l.err != nil {
		return
	}

	if ctx.Schema_name() != nil {
		schemaName := NormalizeIdentifierContext(ctx.Schema_name().Identifier())
		if l.schemaInfo.name == "" {
			l.schemaInfo.name = schemaName
		}
		if schemaName != l.schemaInfo.name {
			l.err = errors.Errorf("schema name mismatch: %s != %s", schemaName, l.schemaInfo.name)
			return
		}
	}
	tableName := NormalizeIdentifierContext(ctx.Table_name().Identifier())
	l.schemaInfo.tableMap[tableName] = &tableInfo{
		id:          len(l.schemaInfo.tableMap),
		name:        tableName,
		existsInNew: false,
		createTable: ctx,
	}
}

// EnterCreate_index is called when production create_index is entered.
func (l *buildSchemaInfoListener) EnterCreate_index(ctx *plsql.Create_indexContext) {
	if l.err != nil {
		return
	}

	if _, ok := ctx.GetParent().(*plsql.Unit_statementContext); !ok {
		// Skip index creation inside other statements.
		return
	}

	schema, index := NormalizeIndexName(ctx.Index_name())
	if schema != "" && l.schemaInfo.name == "" {
		l.schemaInfo.name = schema
	}
	if schema != "" && schema != l.schemaInfo.name {
		l.err = errors.Errorf("schema name mismatch: %s != %s", schema, l.schemaInfo.name)
		return
	}
	l.schemaInfo.indexMap[index] = &indexInfo{
		id:          len(l.schemaInfo.indexMap),
		name:        index,
		existsInNew: false,
		createIndex: ctx,
	}
}

type tableMap map[string]*tableInfo
type indexMap map[string]*indexInfo

type schemaInfo struct {
	name     string
	tableMap tableMap
	indexMap indexMap
}

type tableInfo struct {
	id          int
	name        string
	existsInNew bool
	createTable plsql.ICreate_tableContext
}

type indexInfo struct {
	id          int
	name        string
	existsInNew bool
	createIndex plsql.ICreate_indexContext
}
