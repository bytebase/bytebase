// Package plsql provides the plsql parser plugin.
package plsql

import (
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	plsql "github.com/bytebase/plsql-parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSchemaDiffFunc(storepb.Engine_ORACLE, SchemaDiff)
}

type diffNode struct {
	// The different between the strict mode and non-strict mode is that the non-strict mode:
	// 1. does not compare the index or constraint name, use the definition instead.
	// 2. does not compare the storage option.
	strictMode bool

	schemaName             string
	dropConstraint         []string
	dropIndex              []string
	dropColumn             []string
	dropTable              []string
	createTable            []string
	addColumn              []string
	modifyColumn           []string
	modifyColumnVisibility []string
	addIndex               []string
	addConstraint          []string
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
	for _, modifyColumnVisibility := range diff.modifyColumnVisibility {
		if _, err := buf.WriteString(modifyColumnVisibility); err != nil {
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
func SchemaDiff(ctx base.DiffContext, oldStmt, newStmt string) (string, error) {
	oldSchemaInfo, err := buildSchemaInfo(oldStmt, ctx.StrictMode)
	if err != nil {
		return "", errors.Wrapf(err, "failed to build schema info for old statement")
	}
	newSchemaInfo, err := buildSchemaInfo(newStmt, ctx.StrictMode)
	if err != nil {
		return "", errors.Wrapf(err, "failed to build schema info for new statement")
	}

	diff := &diffNode{
		strictMode: ctx.StrictMode,
		schemaName: oldSchemaInfo.name,
	}
	var newTables []*tableInfo
	for _, table := range newSchemaInfo.tableMap {
		newTables = append(newTables, table)
	}
	slices.SortFunc(newTables, func(i, j *tableInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})
	for _, newTable := range newTables {
		tableName := newTable.name
		oldTable, exists := oldSchemaInfo.tableMap[tableName]
		if !exists {
			diff.createTable = append(diff.createTable, getCreateTableWithoutStoreOption(newTable.createTable))
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
	slices.SortFunc(remainingTables, func(i, j *tableInfo) int {
		if i.id < j.id {
			return -1
		}
		if i.id > j.id {
			return 1
		}
		return 0
	})
	for _, table := range remainingTables {
		if ctx.StrictMode {
			diff.dropTable = append(diff.dropTable, fmt.Sprintf(`DROP TABLE "%s"."%s";`, oldSchemaInfo.name, table.name))
		} else {
			diff.dropTable = append(diff.dropTable, fmt.Sprintf(`DROP TABLE "%s";`, table.name))
		}
	}

	var newIndexes []*indexInfo
	for _, index := range newSchemaInfo.indexMap {
		newIndexes = append(newIndexes, index)
	}
	slices.SortFunc(newIndexes, func(i, j *indexInfo) int {
		if i.pos < j.pos {
			return -1
		}
		if i.pos > j.pos {
			return 1
		}
		return 0
	})
	for _, newIndex := range newIndexes {
		id := newIndex.id
		oldIndex, exists := oldSchemaInfo.indexMap[id]
		if !exists {
			diff.addIndex = append(diff.addIndex, newIndex.createIndex.GetParser().GetTokenStream().GetTextFromRuleContext(newIndex.createIndex))
			continue
		}
		if ctx.StrictMode {
			diff.diffIndex(oldIndex, newIndex)
		}
		delete(oldSchemaInfo.indexMap, id)
	}

	var remainingIndexes []*indexInfo
	for _, index := range oldSchemaInfo.indexMap {
		remainingIndexes = append(remainingIndexes, index)
	}
	slices.SortFunc(remainingIndexes, func(i, j *indexInfo) int {
		if i.pos < j.pos {
			return -1
		}
		if i.pos > j.pos {
			return 1
		}
		return 0
	})
	for _, index := range remainingIndexes {
		if ctx.StrictMode {
			diff.dropIndex = append(diff.dropIndex, fmt.Sprintf(`DROP INDEX "%s"."%s";`, oldSchemaInfo.name, index.name))
		} else {
			diff.dropIndex = append(diff.dropIndex, fmt.Sprintf(`DROP INDEX "%s";`, index.name))
		}
	}

	return diff.String()
}

func getCreateTableWithoutStoreOption(createTable plsql.ICreate_tableContext) string {
	if createTable.Relational_table() == nil {
		return createTable.GetParser().GetTokenStream().GetTextFromRuleContext(createTable)
	}
	return createTable.GetParser().GetTokenStream().GetTextFromTokens(
		createTable.GetStart(),
		createTable.Relational_table().RIGHT_PAREN().GetSymbol(),
	) + ";"
}

func (diff *diffNode) diffIndex(oldIndex, newIndex *indexInfo) {
	// TODO: compare index definition instead of text.
	oldString := oldIndex.createIndex.GetParser().GetTokenStream().GetTextFromRuleContext(oldIndex.createIndex)
	newString := newIndex.createIndex.GetParser().GetTokenStream().GetTextFromRuleContext(newIndex.createIndex)
	if oldString != newString {
		if diff.strictMode {
			diff.dropIndex = append(diff.dropIndex, fmt.Sprintf(`DROP INDEX "%s"."%s";`, diff.schemaName, oldIndex.name))
		} else {
			diff.dropIndex = append(diff.dropIndex, fmt.Sprintf(`DROP INDEX "%s";`, oldIndex.name))
		}
		diff.addIndex = append(diff.addIndex, newIndex.createIndex.GetParser().GetTokenStream().GetTextFromRuleContext(newIndex.createIndex))
	}
}

func (diff *diffNode) diffTable(oldTable, newTable *tableInfo) error {
	if err := diff.diffColumn(oldTable, newTable); err != nil {
		return err
	}
	return diff.diffConstraint(oldTable, newTable)
}

func (diff *diffNode) buildConstraintMap(table *tableInfo) map[string]plsql.IRelational_propertyContext {
	constraintMap := make(map[string]plsql.IRelational_propertyContext)
	if table.createTable.Relational_table() == nil {
		return constraintMap
	}
	for _, item := range table.createTable.Relational_table().AllRelational_property() {
		id := diff.getConstraintID(item)
		if id != "" {
			constraintMap[id] = item
		}
	}
	return constraintMap
}

func (diff *diffNode) getConstraintID(ctx plsql.IRelational_propertyContext) string {
	if diff.strictMode {
		switch {
		case ctx.Out_of_line_constraint() != nil:
			constraint := ctx.Out_of_line_constraint()
			if constraint.Constraint_name() == nil {
				return ""
			}
			_, constraintName := NormalizeConstraintName(constraint.Constraint_name())
			return strings.TrimSpace(constraintName)
		case ctx.Out_of_line_ref_constraint() != nil:
			constraint := ctx.Out_of_line_ref_constraint()
			if constraint.Constraint_name() == nil {
				return ""
			}
			_, constraintName := NormalizeConstraintName(constraint.Constraint_name())
			return strings.TrimSpace(constraintName)
		default:
			return ""
		}
	} else if ctx.Out_of_line_constraint() != nil || ctx.Out_of_line_ref_constraint() != nil {
		return strings.TrimSpace(EraseString(EraseContext{
			eraseConstraintName: true,
			eraseSchemaName:     true,
			eraseIndexName:      true,
		}, ctx, ctx.GetParser().GetTokenStream()))
	}
	return ""
}

func (diff *diffNode) diffConstraint(oldTable, newTable *tableInfo) error {
	if newTable.createTable.Relational_table() == nil {
		// TODO: support object_table and xmltype_table
		return nil
	}

	var addConstraints []plsql.IRelational_propertyContext
	var dropConstraints []string
	oldConstraintMap := diff.buildConstraintMap(oldTable)
	for _, item := range newTable.createTable.Relational_table().AllRelational_property() {
		switch {
		case item.Out_of_line_constraint() != nil:
			id := diff.getConstraintID(item)
			if id == "" {
				continue
			}
			oldConstraint, ok := oldConstraintMap[id]
			if !ok {
				addConstraints = append(addConstraints, item)
				continue
			}
			// Compare the constraint definition.
			if diff.strictMode && !isConstraintEqual(oldConstraint, item) {
				addConstraints = append(addConstraints, item)
				dropConstraints = append(dropConstraints, id)
			}
			delete(oldConstraintMap, id)
		case item.Out_of_line_ref_constraint() != nil:
			id := diff.getConstraintID(item)
			oldConstraint, ok := oldConstraintMap[id]
			if !ok {
				addConstraints = append(addConstraints, item)
				continue
			}
			// Compare the constraint definition.
			if diff.strictMode && !isConstraintEqual(oldConstraint, item) {
				addConstraints = append(addConstraints, item)
				dropConstraints = append(dropConstraints, id)
			}
			delete(oldConstraintMap, id)
		default:
			// Ignore other relational properties
		}
	}
	for _, item := range oldConstraintMap {
		constraintName := getConstraintName(item)
		if constraintName == "" {
			continue
		}
		dropConstraints = append(dropConstraints, constraintName)
	}

	slices.Sort(dropConstraints)

	return diff.appendConstraintDiff(newTable.name, addConstraints, dropConstraints)
}

func getConstraintName(ctx plsql.IRelational_propertyContext) string {
	switch {
	case ctx.Out_of_line_constraint() != nil:
		constraint := ctx.Out_of_line_constraint()
		if constraint.Constraint_name() == nil {
			return ""
		}
		_, constraintName := NormalizeConstraintName(constraint.Constraint_name())
		return constraintName
	case ctx.Out_of_line_ref_constraint() != nil:
		constraint := ctx.Out_of_line_ref_constraint()
		if constraint.Constraint_name() == nil {
			return ""
		}
		_, constraintName := NormalizeConstraintName(constraint.Constraint_name())
		return constraintName
	default:
		return ""
	}
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
	if diff.strictMode {
		if _, err := buf.WriteString(diff.schemaName); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"."`); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(tableName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" DROP CONSTRAINT "`); err != nil {
		return err
	}
	if diff.strictMode {
		if _, err := buf.WriteString(diff.schemaName); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"."`); err != nil {
			return err
		}
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
	if diff.strictMode {
		if _, err := buf.WriteString(diff.schemaName); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"."`); err != nil {
			return err
		}
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
	var modifyColumnVisibility []plsql.IColumn_definitionContext
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
		if !isColumnVisibilityEqual(oldColumn, item.Column_definition()) {
			modifyColumnVisibility = append(modifyColumnVisibility, item.Column_definition())
		}
		delete(oldColumnMap, newColumnName)
	}
	for _, column := range oldColumnMap {
		dropColumns = append(dropColumns, NormalizeIdentifierContext(column.Column_name().Identifier()))
	}

	slices.Sort(dropColumns)

	return diff.appendColumnDiff(oldTable.name, addColumns, modifyColumns, modifyColumnVisibility, dropColumns)
}

func (diff *diffNode) appendColumnDiff(tableName string, addColumns, modifyColumns, modifyColumnVisibility []plsql.IColumn_definitionContext, dropColumns []string) error {
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
	if len(modifyColumnVisibility) != 0 {
		if err := diff.appendModifyColumnVisibility(tableName, modifyColumnVisibility); err != nil {
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
	if diff.strictMode {
		if _, err := buf.WriteString(diff.schemaName); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"."`); err != nil {
			return err
		}
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

func (diff *diffNode) appendModifyColumnVisibility(tableName string, modifyColumnVisibility []plsql.IColumn_definitionContext) error {
	var buf strings.Builder

	if _, err := buf.WriteString(`ALTER TABLE "`); err != nil {
		return err
	}
	if diff.strictMode {
		if _, err := buf.WriteString(diff.schemaName); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"."`); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(tableName); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" MODIFY (`); err != nil {
		return err
	}
	for i, column := range modifyColumnVisibility {
		if i != 0 {
			if _, err := buf.WriteString(`,`); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("\n	"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column.GetParser().GetTokenStream().GetTextFromRuleContext(column.Column_name())); err != nil {
			return err
		}
		columnVisibility := " VISIBLE"
		if column.INVISIBLE() != nil {
			columnVisibility = " INVISIBLE"
		}
		if _, err := buf.WriteString(columnVisibility); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n);"); err != nil {
		return err
	}
	diff.modifyColumnVisibility = append(diff.modifyColumnVisibility, buf.String())
	return nil
}

func (diff *diffNode) appendModifyColumn(tableName string, modifyColumns []plsql.IColumn_definitionContext) error {
	var buf strings.Builder

	if _, err := buf.WriteString(`ALTER TABLE "`); err != nil {
		return err
	}
	if diff.strictMode {
		if _, err := buf.WriteString(diff.schemaName); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"."`); err != nil {
			return err
		}
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
		if _, err := buf.WriteString(convertToModifyColumn(column)); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n);"); err != nil {
		return err
	}
	diff.modifyColumn = append(diff.modifyColumn, buf.String())
	return nil
}

func convertToModifyColumn(column plsql.IColumn_definitionContext) string {
	var results []string
	results = append(results, column.GetParser().GetTokenStream().GetTextFromRuleContext(column.Column_name()))
	if column.Datatype() != nil {
		results = append(results, column.GetParser().GetTokenStream().GetTextFromRuleContext(column.Datatype()))
	}

	if column.COLLATE() != nil {
		results = append(results, column.GetParser().GetTokenStream().GetTextFromTokens(
			column.COLLATE().GetSymbol(),
			column.Column_collation_name().GetStop(),
		))
	}

	if column.DEFAULT() != nil && column.Expression() != nil {
		results = append(results, "DEFAULT")
		if column.NULL_() != nil {
			results = append(results, "NO NULL")
		}
		results = append(results, column.GetParser().GetTokenStream().GetTextFromRuleContext(column.Expression()))
	}

	for _, item := range column.AllInline_constraint() {
		if item.NULL_() != nil {
			if item.NOT() != nil {
				results = append(results, "NOT NULL")
			} else {
				results = append(results, "NULL")
			}
		}
	}

	return strings.Join(results, " ")
}

func (diff *diffNode) appendAddColumn(tableName string, addColumns []plsql.IColumn_definitionContext) error {
	var buf strings.Builder

	if _, err := buf.WriteString(`ALTER TABLE "`); err != nil {
		return err
	}
	if diff.strictMode {
		if _, err := buf.WriteString(diff.schemaName); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"."`); err != nil {
			return err
		}
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

func isColumnVisibilityEqual(oldColumn, newColumn plsql.IColumn_definitionContext) bool {
	oldVisible := true
	newVisible := true
	if oldColumn.INVISIBLE() != nil {
		oldVisible = false
	}
	if newColumn.INVISIBLE() != nil {
		newVisible = false
	}
	return oldVisible == newVisible
}

func isColumnEqual(oldColumn, newColumn plsql.IColumn_definitionContext) bool {
	// TODO: compare column definition instead of text.
	// TODO: ignore visible now.
	oldString := getColumnCompareString(oldColumn)
	newString := getColumnCompareString(newColumn)
	return oldString == newString
}

func getColumnCompareString(column plsql.IColumn_definitionContext) string {
	var results []string
	if column.Datatype() != nil {
		results = append(results, column.GetParser().GetTokenStream().GetTextFromRuleContext(column.Datatype()))
	}

	if column.COLLATE() != nil {
		results = append(results, column.GetParser().GetTokenStream().GetTextFromTokens(
			column.COLLATE().GetSymbol(),
			column.Column_collation_name().GetStop(),
		))
	}

	if column.DEFAULT() != nil {
		if column.Expression() != nil {
			results = append(results, column.GetParser().GetTokenStream().GetTextFromTokens(
				column.DEFAULT().GetSymbol(),
				column.Expression().GetStop(),
			))
		} else if column.Identity_clause() != nil {
			results = append(results, column.GetParser().GetTokenStream().GetTextFromTokens(
				column.DEFAULT().GetSymbol(),
				column.Identity_clause().GetStop(),
			))
		}
	}

	for _, item := range column.AllInline_constraint() {
		if item.NULL_() != nil {
			if item.NOT() != nil {
				results = append(results, "NOT NULL")
			} else {
				results = append(results, "NULL")
			}
		}
	}

	return strings.Join(results, " ")
}

func buildSchemaInfo(statement string, strictMode bool) (*schemaInfo, error) {
	node, _, err := ParsePLSQL(statement)
	if err != nil {
		return nil, err
	}

	listener := &buildSchemaInfoListener{
		strictMode: strictMode,
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

	strictMode bool
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

	id := getIndexID(ctx, l.strictMode)
	schema, index := NormalizeIndexName(ctx.Index_name())
	if schema != "" && l.schemaInfo.name == "" {
		l.schemaInfo.name = schema
	}
	if schema != "" && schema != l.schemaInfo.name {
		l.err = errors.Errorf("schema name mismatch: %s != %s", schema, l.schemaInfo.name)
		return
	}
	l.schemaInfo.indexMap[id] = &indexInfo{
		pos:         len(l.schemaInfo.indexMap),
		id:          id,
		name:        index,
		existsInNew: false,
		createIndex: ctx,
	}
}

func getIndexID(ctx plsql.ICreate_indexContext, strictMode bool) string {
	if strictMode {
		_, indexName := NormalizeIndexName(ctx.Index_name())
		return strings.TrimSpace(indexName)
	}
	return strings.TrimSpace(EraseString(EraseContext{
		eraseIndexName:      true,
		eraseSchemaName:     true,
		eraseConstraintName: true,
		eraseStoreOption:    true,
	}, ctx, ctx.GetParser().GetTokenStream()))
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
	pos         int
	id          string
	name        string
	existsInNew bool
	createIndex plsql.ICreate_indexContext
}
