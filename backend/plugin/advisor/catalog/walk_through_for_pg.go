package catalog

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
)

const (
	publicSchemaName = "public"
)

func (d *DatabaseState) pgWalkThrough(stmt string) error {
	nodeList, err := pgParse(stmt)
	if err != nil {
		return NewParseError(err.Error())
	}

	for _, node := range nodeList {
		if err := d.pgChangeState(node); err != nil {
			return err
		}
	}

	return nil
}

func (d *DatabaseState) pgChangeState(in ast.Node) (err *WalkThroughError) {
	defer func() {
		if err == nil {
			return
		}
		if err.Line == 0 {
			err.Line = in.LastLine()
		}
	}()

	if d.deleted {
		return &WalkThroughError{
			Type:    ErrorTypeDatabaseIsDeleted,
			Content: fmt.Sprintf(`Database %q is deleted`, d.name),
		}
	}
	switch node := in.(type) {
	case *ast.CreateTableStmt:
		return d.pgCreateTable(node)
	case *ast.CreateIndexStmt:
		return d.pgCreateIndex(node)
	case *ast.AlterTableStmt:
		return d.pgAlterTable(node)
	case *ast.RenameIndexStmt:
		return d.pgRenameIndex(node)
	case *ast.CreateSchemaStmt:
		return d.pgCreateSchema(node)
	case *ast.DropSchemaStmt:
		return d.pgDropSchema(node)
	case *ast.DropTableStmt:
		return d.pgDropTableList(node)
	case *ast.DropIndexStmt:
		return d.pgDropIndexList(node)
	default:
		return nil
	}
}

func (d *DatabaseState) pgDropIndexList(node *ast.DropIndexStmt) *WalkThroughError {
	for _, indexDef := range node.IndexList {
		if err := d.pgDropIndex(indexDef, node.IfExists, node.Behavior); err != nil {
			return err
		}
	}

	return nil
}

func (d *DatabaseState) pgDropIndex(indexDef *ast.IndexDef, ifExists bool, _ ast.DropBehavior) *WalkThroughError {
	schemaName := ""
	if indexDef.Table != nil {
		schemaName = indexDef.Table.Schema
	}
	schema, err := d.getSchema(schemaName)
	if err != nil {
		if ifExists {
			return nil
		}
		return err
	}

	table, index, err := schema.getIndex(indexDef.Name)
	if err != nil {
		return err
	}

	delete(schema.identifierMap, index.name)
	delete(table.indexSet, index.name)
	return nil
}

func (d *DatabaseState) pgDropTableList(node *ast.DropTableStmt) *WalkThroughError {
	for _, tableName := range node.TableList {
		if tableName.Type == ast.TableTypeView {
			if err := d.pgDropView(tableName, node.IfExists, node.Behavior); err != nil {
				return err
			}
		} else {
			if err := d.pgDropTable(tableName, node.IfExists, node.Behavior); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *DatabaseState) pgDropView(tableDef *ast.TableDef, ifExists bool, _ ast.DropBehavior) *WalkThroughError {
	schema, err := d.getSchema(tableDef.Schema)
	if err != nil {
		if ifExists {
			return nil
		}
		return err
	}

	delete(schema.identifierMap, tableDef.Name)
	delete(schema.viewSet, tableDef.Name)
	return nil
}

func parseViewName(viewName string) (string, string, error) {
	pattern := `^"(.+?)"\."(.+?)"$`

	re := regexp.MustCompile(pattern)

	match := re.FindStringSubmatch(viewName)

	if len(match) != 3 {
		return "", "", errors.Errorf("invalid view name: %s", viewName)
	}

	return match[1], match[2], nil
}

func (d *DatabaseState) existedViewList(viewMap map[string]bool) ([]string, *WalkThroughError) {
	var result []string
	for viewName := range viewMap {
		schemaName, viewName, err := parseViewName(viewName)
		if err != nil {
			return nil, &WalkThroughError{
				Type:    ErrorTypeInternal,
				Content: fmt.Sprintf("failed to check view dependency: %s", err.Error()),
			}
		}
		schemaMeta, exists := d.schemaSet[schemaName]
		if !exists {
			continue
		}
		if _, exists := schemaMeta.viewSet[viewName]; !exists {
			continue
		}

		result = append(result, fmt.Sprintf("%q.%q", schemaName, viewName))
	}
	return result, nil
}

func (d *DatabaseState) pgDropTable(tableDef *ast.TableDef, ifExists bool, _ ast.DropBehavior) *WalkThroughError {
	schema, err := d.getSchema(tableDef.Schema)
	if err != nil {
		if ifExists {
			return nil
		}
		return err
	}

	table, err := schema.pgGetTable(tableDef.Name)
	if err != nil {
		if ifExists {
			return nil
		}
		return err
	}

	viewList, err := d.existedViewList(table.dependentView)
	if err != nil {
		return err
	}
	if len(viewList) > 0 {
		return &WalkThroughError{
			Type:    ErrorTypeTableIsReferencedByView,
			Content: fmt.Sprintf("Cannot drop table %q.%q, it's referenced by view: %s", schema.name, table.name, strings.Join(viewList, ", ")),
			Payload: viewList,
		}
	}

	for indexName := range table.indexSet {
		delete(schema.identifierMap, indexName)
	}

	delete(schema.identifierMap, table.name)
	delete(schema.tableSet, table.name)
	return nil
}

func (d *DatabaseState) pgDropSchema(node *ast.DropSchemaStmt) *WalkThroughError {
	for _, schemaName := range node.SchemaList {
		schema, err := d.getSchema(schemaName)
		if err != nil {
			if node.IfExists {
				continue
			}
			return err
		}

		delete(d.schemaSet, schema.name)
	}

	return nil
}

func (d *DatabaseState) pgCreateSchema(node *ast.CreateSchemaStmt) *WalkThroughError {
	if _, exists := d.schemaSet[node.Name]; exists {
		return &WalkThroughError{
			Type:    ErrorTypeSchemaExists,
			Content: fmt.Sprintf("Schema %q already exists", node.Name),
		}
	}

	schema := &SchemaState{
		name:          node.Name,
		identifierMap: make(identifierMap),
		tableSet:      make(tableStateMap),
		viewSet:       make(viewStateMap),
	}
	d.schemaSet[schema.name] = schema

	for _, item := range node.SchemaElementList {
		switch itemNode := item.(type) {
		// TODO(rebelice): deal with other item type here.
		case *ast.CreateTableStmt:
			if err := d.pgCreateTable(itemNode); err != nil {
				return err
			}
		default:
			// TODO: hack the linter.
		}
	}

	return nil
}

func (d *DatabaseState) pgRenameIndex(node *ast.RenameIndexStmt) *WalkThroughError {
	schema, err := d.getSchema(node.Table.Schema)
	if err != nil {
		return err
	}
	table, index, err := schema.getIndex(node.IndexName)
	if err != nil {
		return err
	}
	if _, exists := schema.identifierMap[node.NewName]; exists {
		return NewRelationExistsError(node.NewName, schema.name)
	}

	delete(schema.identifierMap, index.name)
	delete(table.indexSet, index.name)
	index.name = node.NewName
	schema.identifierMap[index.name] = true
	table.indexSet[index.name] = index
	return nil
}

func (d *DatabaseState) pgAlterTable(node *ast.AlterTableStmt) *WalkThroughError {
	// Do nothing for view.
	if node.Table.Type == ast.TableTypeView {
		return nil
	}
	schema, err := d.getSchema(node.Table.Schema)
	if err != nil {
		return err
	}
	table, err := schema.pgGetTable(node.Table.Name)
	if err != nil {
		return err
	}

	for _, item := range node.AlterItemList {
		switch itemNode := item.(type) {
		case *ast.RenameColumnStmt:
			if err := table.pgRenameColumn(itemNode); err != nil {
				return err
			}
		case *ast.RenameConstraintStmt:
			if err := schema.pgRenameConstraint(table, itemNode); err != nil {
				return err
			}
		case *ast.RenameTableStmt:
			if err := schema.pgRenameTable(table, itemNode); err != nil {
				return err
			}
		case *ast.SetSchemaStmt:
			if err := d.pgSetSchema(schema, table, itemNode); err != nil {
				return err
			}
		case *ast.AddColumnListStmt:
			if err := schema.pgAddColumn(table, itemNode); err != nil {
				return err
			}
		case *ast.DropColumnStmt:
			if err := d.pgDropColumn(schema, table, itemNode); err != nil {
				return err
			}
		case *ast.AlterColumnTypeStmt:
			if err := d.pgAlterColumnType(schema, table, itemNode); err != nil {
				return err
			}
		case *ast.SetDefaultStmt:
			if err := table.pgSetDefault(itemNode); err != nil {
				return err
			}
		case *ast.DropDefaultStmt:
			if err := table.pgDropDefault(itemNode); err != nil {
				return err
			}
		case *ast.SetNotNullStmt:
			if err := table.pgSetNotNull(itemNode); err != nil {
				return err
			}
		case *ast.DropNotNullStmt:
			if err := table.pgDropNotNull(itemNode); err != nil {
				return err
			}
		case *ast.AddConstraintStmt:
			if err := schema.pgCreateTableConstraint(table, itemNode.Constraint); err != nil {
				return err
			}
		case *ast.DropConstraintStmt:
			if err := schema.pgDropConstraint(table, itemNode); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SchemaState) pgDropConstraint(t *TableState, node *ast.DropConstraintStmt) *WalkThroughError {
	if index, exists := t.indexSet[node.ConstraintName]; exists {
		delete(s.identifierMap, index.name)
		delete(t.indexSet, index.name)
	}

	// TODO(rebelice): deal with other constraints

	// TODO(rebelice): deal with CASCADE

	return nil
}

func (t *TableState) pgDropNotNull(node *ast.DropNotNullStmt) *WalkThroughError {
	column, err := t.getColumn(node.ColumnName)
	if err != nil {
		return err
	}

	column.nullable = newTruePointer()
	return nil
}

func (t *TableState) pgSetNotNull(node *ast.SetNotNullStmt) *WalkThroughError {
	column, err := t.getColumn(node.ColumnName)
	if err != nil {
		return err
	}

	column.nullable = newFalsePointer()
	return nil
}

func (t *TableState) pgDropDefault(node *ast.DropDefaultStmt) *WalkThroughError {
	column, err := t.getColumn(node.ColumnName)
	if err != nil {
		return err
	}

	column.defaultValue = nil
	return nil
}

func (t *TableState) pgSetDefault(node *ast.SetDefaultStmt) *WalkThroughError {
	column, err := t.getColumn(node.ColumnName)
	if err != nil {
		return err
	}

	column.defaultValue = newStringPointer(node.Expression.Text())
	return nil
}

func (d *DatabaseState) pgAlterColumnType(schema *SchemaState, t *TableState, node *ast.AlterColumnTypeStmt) *WalkThroughError {
	column, err := t.getColumn(node.ColumnName)
	if err != nil {
		return err
	}

	viewList, err := d.existedViewList(column.dependentView)
	if err != nil {
		return err
	}
	if len(viewList) > 0 {
		return &WalkThroughError{
			Type:    ErrorTypeColumnIsReferencedByView,
			Content: fmt.Sprintf("Cannot alter type of column %q in table %q.%q, it's referenced by view: %s", column.name, schema.name, t.name, strings.Join(viewList, ", ")),
			Payload: viewList,
		}
	}

	typeString, deparseErr := pgrawparser.Deparse(pgrawparser.DeparseContext{}, node.Type)
	if deparseErr != nil {
		return &WalkThroughError{
			Type:    ErrorTypeDeparseError,
			Content: err.Error(),
		}
	}
	column.columnType = &typeString
	column.collation = newStringPointer(normalizeCollation(node.Collation))
	// TODO(rebelice): support USING expression
	return nil
}

func (d *DatabaseState) pgDropColumn(schema *SchemaState, t *TableState, node *ast.DropColumnStmt) *WalkThroughError {
	column, exists := t.columnSet[node.ColumnName]
	if !exists {
		if node.IfExists {
			return nil
		}
		return NewColumnNotExistsError(t.name, node.ColumnName)
	}

	viewList, err := d.existedViewList(column.dependentView)
	if err != nil {
		return err
	}
	if len(viewList) > 0 {
		return &WalkThroughError{
			Type:    ErrorTypeColumnIsReferencedByView,
			Content: fmt.Sprintf("Cannot drop column %q in table %q.%q, it's referenced by view: %s", column.name, schema.name, t.name, strings.Join(viewList, ", ")),
			Payload: viewList,
		}
	}

	// Drop the constraints and indexes involving the column.
	var dropIndexList []string

	for _, index := range t.indexSet {
		for _, key := range index.expressionList {
			// TODO(rebelice): deal with expression key.
			if key == index.name {
				dropIndexList = append(dropIndexList, index.name)
			}
		}
	}
	for _, indexName := range dropIndexList {
		delete(t.indexSet, indexName)
	}

	// TODO(rebelice): deal with other constraints.

	// TODO(rebelice): deal with CASCADE.

	delete(t.columnSet, node.ColumnName)
	return nil
}

func (s *SchemaState) pgAddColumn(t *TableState, node *ast.AddColumnListStmt) *WalkThroughError {
	if len(node.ColumnList) != 1 {
		return &WalkThroughError{
			Type:    ErrorTypeInvalidStatement,
			Content: "PostgreSQL doesn't support to add multi-columns in one ADD COLUMN statement",
		}
	}
	return s.pgCreateColumn(t, node.ColumnList[0], node.IfNotExists)
}

func (d *DatabaseState) pgSetSchema(oldSchema *SchemaState, t *TableState, node *ast.SetSchemaStmt) *WalkThroughError {
	newSchema, exists := d.schemaSet[node.NewSchema]
	if !exists {
		return &WalkThroughError{
			Type:    ErrorTypeSchemaNotExists,
			Content: fmt.Sprintf("Schema %q does not exist", node.NewSchema),
		}
	}

	if _, exists := newSchema.identifierMap[t.name]; exists {
		return NewRelationExistsError(t.name, newSchema.name)
	}

	for indexName := range t.indexSet {
		if _, exists := newSchema.identifierMap[indexName]; exists {
			return NewRelationExistsError(indexName, newSchema.name)
		}
	}

	// TODO(rebelice): check other constraints and sequences here.

	for indexName := range t.indexSet {
		delete(oldSchema.identifierMap, indexName)
		newSchema.identifierMap[indexName] = true
	}

	delete(oldSchema.identifierMap, t.name)
	delete(oldSchema.tableSet, t.name)
	newSchema.identifierMap[t.name] = true
	newSchema.tableSet[t.name] = t
	return nil
}

func (s *SchemaState) pgRenameTable(t *TableState, node *ast.RenameTableStmt) *WalkThroughError {
	if _, exists := s.identifierMap[node.NewName]; exists {
		return NewRelationExistsError(node.NewName, s.name)
	}

	delete(s.identifierMap, t.name)
	delete(s.tableSet, t.name)
	t.name = node.NewName
	s.identifierMap[t.name] = true
	s.tableSet[t.name] = t
	return nil
}

func (s *SchemaState) pgRenameConstraint(t *TableState, node *ast.RenameConstraintStmt) *WalkThroughError {
	index, exists := t.indexSet[node.ConstraintName]
	if !exists {
		// We haven't deal with foreign and check constraints, so skip if not exists.
		return nil
	}
	// TODO(rebelice): check other constraints here.

	if !index.isConstraint {
		return &WalkThroughError{
			Type:    ErrorTypeConstraintNotExists,
			Content: fmt.Sprintf("Constraint %q for table %q does not exist", node.ConstraintName, t.name),
		}
	}

	if _, exists := s.identifierMap[node.NewName]; exists {
		return NewRelationExistsError(node.NewName, s.name)
	}

	delete(s.identifierMap, node.ConstraintName)
	delete(t.indexSet, node.ConstraintName)
	index.name = node.NewName
	s.identifierMap[index.name] = true
	t.indexSet[index.name] = index
	return nil
}

func (t *TableState) pgRenameColumn(node *ast.RenameColumnStmt) *WalkThroughError {
	column, err := t.getColumn(node.ColumnName)
	if err != nil {
		return err
	}
	if node.ColumnName == node.NewName {
		return nil
	}
	if _, exists := t.columnSet[node.NewName]; exists {
		return &WalkThroughError{
			Type:    ErrorTypeColumnExists,
			Content: fmt.Sprintf("The column %q already exists in table %q", node.NewName, t.name),
		}
	}

	delete(t.columnSet, column.name)

	for _, index := range t.indexSet {
		for i, key := range index.expressionList {
			// TODO(rebelice): only deal with the column type index key here.
			if key == column.name {
				index.expressionList[i] = node.NewName
			}
		}
	}

	column.name = node.NewName
	t.columnSet[node.NewName] = column
	return nil
}

func (d *DatabaseState) pgCreateIndex(node *ast.CreateIndexStmt) *WalkThroughError {
	schema, err := d.getSchema(node.Index.Table.Schema)
	if err != nil {
		return err
	}
	return schema.pgCreateIndex(node, false /* isPrimary */, false /* isConstraint */)
}

func (d *DatabaseState) pgCreateTable(node *ast.CreateTableStmt) *WalkThroughError {
	if node.Name.Database != "" && d.name != node.Name.Database {
		return &WalkThroughError{
			Type:    ErrorTypeAccessOtherDatabase,
			Content: fmt.Sprintf("Database %q is not the current database %q", node.Name.Database, d.name),
		}
	}

	schema, err := d.getSchema(node.Name.Schema)
	if err != nil {
		return err
	}

	if _, exists := schema.tableSet[node.Name.Name]; exists {
		if node.IfNotExists {
			return nil
		}
		return &WalkThroughError{
			Type:    ErrorTypeTableExists,
			Content: fmt.Sprintf(`The table %q already exists in the schema %q`, node.Name.Name, schema.name),
		}
	}

	table := &TableState{
		name:      node.Name.Name,
		columnSet: make(columnStateMap),
		indexSet:  make(IndexStateMap),
	}
	schema.tableSet[table.name] = table

	for _, column := range node.ColumnList {
		if err := schema.pgCreateColumn(table, column, false /* ifNotExists */); err != nil {
			err.Line = column.LastLine()
			return err
		}
	}

	for _, constraint := range node.ConstraintList {
		if err := schema.pgCreateTableConstraint(table, constraint); err != nil {
			return err
		}
	}

	return nil
}

func (s *SchemaState) pgCreateTableConstraint(t *TableState, constraint *ast.ConstraintDef) *WalkThroughError {
	switch constraint.Type {
	case ast.ConstraintTypePrimary:
		return s.pgCreatePrimaryKey(t, constraint)
	case ast.ConstraintTypePrimaryUsingIndex:
		return s.pgCreatePrimaryKeyUsingIndex(t, constraint)
	case ast.ConstraintTypeUnique:
		var indexKeyList []*ast.IndexKeyDef
		for _, key := range constraint.KeyList {
			indexKeyList = append(indexKeyList, &ast.IndexKeyDef{
				Type: ast.IndexKeyTypeColumn,
				Key:  key,
			})
		}
		return s.pgCreateIndex(&ast.CreateIndexStmt{
			IfNotExists: false,
			Index: &ast.IndexDef{
				Name: constraint.Name,
				Table: &ast.TableDef{
					Name:   t.name,
					Schema: s.name,
				},
				Unique:  true,
				Method:  ast.IndexMethodTypeBTree,
				KeyList: indexKeyList,
			},
		}, false /* isPrimary */, true /* isConstraint */)
	case ast.ConstraintTypeUniqueUsingIndex:
		return s.pgCreateUniqueUsingIndex(t, constraint)
	case ast.ConstraintTypeCheck:
		// We do not deal with CHECK constraint.
	case ast.ConstraintTypeExclusion:
		// We do not deal with EXCLUSION constraint.
	case ast.ConstraintTypeForeign:
		// We do not deal with FOREIGN constraint.
	}
	return nil
}

func (s *SchemaState) pgCreateColumn(t *TableState, column *ast.ColumnDef, ifNotExists bool) *WalkThroughError {
	if _, exists := t.columnSet[column.ColumnName]; exists {
		if ifNotExists {
			return nil
		}
		return &WalkThroughError{
			Type:    ErrorTypeColumnExists,
			Content: fmt.Sprintf("The column %q already exists in table %q", column.ColumnName, t.name),
		}
	}

	pos := len(t.columnSet) + 1
	typeString, err := pgrawparser.Deparse(pgrawparser.DeparseContext{}, column.Type)
	if err != nil {
		return &WalkThroughError{
			Type:    ErrorTypeDeparseError,
			Content: err.Error(),
		}
	}

	columnState := &ColumnState{
		name:         column.ColumnName,
		position:     &pos,
		defaultValue: nil,
		nullable:     newTruePointer(),
		columnType:   &typeString,
		collation:    newStringPointer(normalizeCollation(column.Collation)),
		// TODO(rebelice): support comment here.
	}
	t.columnSet[columnState.name] = columnState

	for _, constraint := range column.ConstraintList {
		switch constraint.Type {
		case ast.ConstraintTypePrimary:
			if err := s.pgCreatePrimaryKey(t, constraint); err != nil {
				return err
			}
		case ast.ConstraintTypePrimaryUsingIndex:
			if err := s.pgCreatePrimaryKeyUsingIndex(t, constraint); err != nil {
				return err
			}
		case ast.ConstraintTypeNotNull:
			columnState.nullable = newFalsePointer()
		case ast.ConstraintTypeDefault:
			columnState.defaultValue = newStringPointer(constraint.Expression.Text())
		case ast.ConstraintTypeUnique:
			if err := s.pgCreateIndex(&ast.CreateIndexStmt{
				IfNotExists: false,
				Index: &ast.IndexDef{
					Table: &ast.TableDef{
						Schema: s.name,
						Name:   t.name,
					},
					Unique: true,
					KeyList: []*ast.IndexKeyDef{
						{
							Type: ast.IndexKeyTypeColumn,
							Key:  column.ColumnName,
						},
					},
					Method: ast.IndexMethodTypeBTree,
				},
			}, false /* isPrimary */, true /* isConstraint */); err != nil {
				return err
			}
		case ast.ConstraintTypeUniqueUsingIndex:
			if err := s.pgCreateUniqueUsingIndex(t, constraint); err != nil {
				return err
			}
		case ast.ConstraintTypeCheck:
			// We do not deal with CHECK constraint.
		case ast.ConstraintTypeExclusion:
			// We do not deal with EXCLUSION constraint.
		case ast.ConstraintTypeForeign:
			// We do not deal with FOREIGN constraint.
		}
	}

	return nil
}

func (s *SchemaState) pgCreateUniqueUsingIndex(t *TableState, constraint *ast.ConstraintDef) *WalkThroughError {
	index, exists := t.indexSet[constraint.IndexName]
	if !exists {
		return NewIndexNotExistsError(t.name, constraint.IndexName)
	}

	var indexKeyList []*ast.IndexKeyDef
	for _, key := range index.expressionList {
		indexKeyList = append(indexKeyList, &ast.IndexKeyDef{
			Type: ast.IndexKeyTypeColumn,
			Key:  key,
		})
	}
	if err := t.checkIndexKey(indexKeyList); err != nil {
		return err
	}

	index.unique = newTruePointer()

	if constraint.Name != "" && constraint.Name != index.name {
		if _, exists := s.identifierMap[constraint.Name]; exists {
			return NewRelationExistsError(constraint.Name, s.name)
		}
		delete(s.identifierMap, index.name)
		delete(t.indexSet, index.name)
		s.identifierMap[constraint.Name] = true
		index.name = constraint.Name
		t.indexSet[index.name] = index
	}
	return nil
}

func (s *SchemaState) pgCreatePrimaryKeyUsingIndex(t *TableState, constraint *ast.ConstraintDef) *WalkThroughError {
	index, exists := t.indexSet[constraint.IndexName]
	if !exists {
		return NewIndexNotExistsError(t.name, constraint.IndexName)
	}

	pkName := constraint.Name
	if pkName == "" {
		pkName = constraint.IndexName
	}

	createPK := &ast.ConstraintDef{
		Type:    ast.ConstraintTypePrimary,
		Name:    pkName,
		KeyList: index.expressionList,
	}
	delete(s.identifierMap, index.name)
	delete(t.indexSet, index.name)
	return s.pgCreatePrimaryKey(t, createPK)
}

func (s *SchemaState) pgCreatePrimaryKey(t *TableState, createPrimaryKey *ast.ConstraintDef) *WalkThroughError {
	for _, columnName := range createPrimaryKey.KeyList {
		column, exists := t.columnSet[columnName]
		if !exists {
			return NewColumnNotExistsError(t.name, columnName)
		}
		column.nullable = newFalsePointer()
	}

	pkName := createPrimaryKey.Name
	if pkName == "" {
		pkName = s.pgGeneratePrimaryKeyName(t.name)
	}

	return s.pgCreateIndex(&ast.CreateIndexStmt{
		IfNotExists: false,
		Index: &ast.IndexDef{
			Name: pkName,
			Table: &ast.TableDef{
				Name:   t.name,
				Schema: s.name,
			},
			Unique:  true,
			Method:  ast.IndexMethodTypeBTree,
			KeyList: pgColumnNameListToKeyList(createPrimaryKey.KeyList),
		},
	}, true /* isPrimary */, true /* isConstraint */)
}

func pgColumnNameListToKeyList(columnList []string) []*ast.IndexKeyDef {
	var result []*ast.IndexKeyDef
	for _, columnName := range columnList {
		result = append(result, &ast.IndexKeyDef{
			Type: ast.IndexKeyTypeColumn,
			Key:  columnName,
		})
	}
	return result
}

func (s *SchemaState) pgGeneratePrimaryKeyName(tableName string) string {
	pkName := fmt.Sprintf("%s_pkey", tableName)
	if _, exists := s.identifierMap[pkName]; !exists {
		return pkName
	}
	suffix := 1
	for {
		if _, exists := s.identifierMap[fmt.Sprintf("%s%d", pkName, suffix)]; !exists {
			return fmt.Sprintf("%s%d", pkName, suffix)
		}
		suffix++
	}
}

func (s *SchemaState) pgCreateIndex(createIndex *ast.CreateIndexStmt, isPrimary bool, isConstraint bool) *WalkThroughError {
	indexName := createIndex.Index.Name
	if len(createIndex.Index.KeyList) == 0 {
		return &WalkThroughError{
			Type:    ErrorTypeIndexEmptyKeys,
			Content: fmt.Sprintf("Index %q in table %q has empty key", indexName, createIndex.Index.Table.Name),
		}
	}

	table, exists := s.tableSet[createIndex.Index.Table.Name]
	if !exists {
		return NewTableNotExistsError(createIndex.Index.Table.Name)
	}

	if indexName != "" {
		if _, exists := s.identifierMap[indexName]; exists {
			if createIndex.IfNotExists {
				return nil
			}
			return NewRelationExistsError(indexName, s.name)
		}
	} else {
		var err error
		indexName, err = s.pgGenerateIndexName(createIndex)
		if err != nil {
			return &WalkThroughError{
				Type:    ErrorTypeInternal,
				Content: fmt.Sprintf("Failed to generate PostgreSQL index name: %s", err.Error()),
			}
		}
	}

	if err := table.checkIndexKey(createIndex.Index.KeyList); err != nil {
		return err
	}

	index := &IndexState{
		name:           indexName,
		expressionList: pgKeyListToStringList(createIndex.Index.KeyList),
		indexType:      newStringPointer(getIndexMethod(createIndex.Index.Method)),
		unique:         newBoolPointer(createIndex.Index.Unique),
		primary:        newBoolPointer(isPrimary),
		isConstraint:   isConstraint,
	}

	table.indexSet[index.name] = index
	s.identifierMap[index.name] = true
	return nil
}

func (t *TableState) checkIndexKey(keyList []*ast.IndexKeyDef) *WalkThroughError {
	for _, key := range keyList {
		if key.Type == ast.IndexKeyTypeColumn {
			if _, exists := t.columnSet[key.Key]; !exists {
				return NewColumnNotExistsError(t.name, key.Key)
			}
		}
	}
	return nil
}

func getIndexMethod(method ast.IndexMethodType) string {
	switch method {
	case ast.IndexMethodTypeBTree:
		return "btree"
	case ast.IndexMethodTypeHash:
		return "hash"
	case ast.IndexMethodTypeGiST:
		return "gist"
	case ast.IndexMethodTypeSpGiST:
		return "spgist"
	case ast.IndexMethodTypeGin:
		return "gin"
	case ast.IndexMethodTypeBrin:
		return "brin"
	case ast.IndexMethodTypeIvfflat:
		return "ivfflat"
	}
	return ""
}

func pgKeyListToStringList(keyList []*ast.IndexKeyDef) []string {
	var result []string
	for _, key := range keyList {
		result = append(result, key.Key)
	}
	return result
}

func (s *SchemaState) pgGenerateIndexName(createIndex *ast.CreateIndexStmt) (string, error) {
	buf := &strings.Builder{}
	if _, err := buf.WriteString(createIndex.Index.Table.Name); err != nil {
		return "", err
	}
	exprText := "expr"
	expressionID := 0
	for _, key := range createIndex.Index.KeyList {
		if err := buf.WriteByte('_'); err != nil {
			return "", err
		}
		switch key.Type {
		case ast.IndexKeyTypeColumn:
			if _, err := buf.WriteString(key.Key); err != nil {
				return "", err
			}
		case ast.IndexKeyTypeExpression:
			if _, err := buf.WriteString(exprText); err != nil {
				return "", err
			}
			if expressionID != 0 {
				if _, err := buf.WriteString(fmt.Sprintf("%d", expressionID)); err != nil {
					return "", err
				}
			}
			expressionID++
		}
	}
	if _, err := buf.WriteString("_idx"); err != nil {
		return "", err
	}
	indexName := buf.String()
	if _, exists := s.identifierMap[indexName]; !exists {
		return indexName, nil
	}
	suffix := 1
	for {
		if _, exists := s.identifierMap[fmt.Sprintf("%s%d", indexName, suffix)]; !exists {
			return fmt.Sprintf("%s%d", indexName, suffix), nil
		}
		suffix++
	}
}

func (d *DatabaseState) getSchema(schemaName string) (*SchemaState, *WalkThroughError) {
	if schemaName == "" {
		schemaName = publicSchemaName
	}
	schema, exists := d.schemaSet[schemaName]
	if !exists {
		if schemaName != publicSchemaName {
			return nil, &WalkThroughError{
				Type:    ErrorTypeSchemaNotExists,
				Content: fmt.Sprintf("The schema %q doesn't exist", schemaName),
			}
		}
		schema = &SchemaState{
			name:          publicSchemaName,
			tableSet:      make(tableStateMap),
			viewSet:       make(viewStateMap),
			identifierMap: make(identifierMap),
		}
		d.schemaSet[publicSchemaName] = schema
	}
	return schema, nil
}

func (s *SchemaState) pgGetTable(tableName string) (*TableState, *WalkThroughError) {
	table, exists := s.tableSet[tableName]
	if !exists {
		return nil, &WalkThroughError{
			Type:    ErrorTypeTableNotExists,
			Content: fmt.Sprintf("The table %q doesn't exists in schema %q", tableName, s.name),
		}
	}
	return table, nil
}

func (s *SchemaState) getIndex(indexName string) (*TableState, *IndexState, *WalkThroughError) {
	for _, table := range s.tableSet {
		if index, exists := table.indexSet[indexName]; exists {
			return table, index, nil
		}
	}

	return nil, nil, &WalkThroughError{
		Type:    ErrorTypeIndexNotExists,
		Content: fmt.Sprintf("Index %q does not exists in schema %q", indexName, s.name),
	}
}

func (t *TableState) getColumn(columnName string) (*ColumnState, *WalkThroughError) {
	column, exists := t.columnSet[columnName]
	if !exists {
		return nil, &WalkThroughError{
			Type:    ErrorTypeColumnNotExists,
			Content: fmt.Sprintf("The column %q doesn't exists in table %q", columnName, t.name),
		}
	}
	return column, nil
}

func pgParse(stmt string) ([]ast.Node, error) {
	return pgrawparser.Parse(pgrawparser.ParseContext{}, stmt)
}

func normalizeCollation(collation *ast.CollationNameDef) string {
	if collation == nil {
		return ""
	}

	if collation.Schema == "" || collation.Schema == "public" {
		return collation.Name
	}

	return fmt.Sprintf("%q.%q", collation.Schema, collation.Name)
}
