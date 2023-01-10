package catalog

import (
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
)

const (
	publicSchemaName = "public"
)

func (d *DatabaseState) pgWalkThrough(stmt string) error {
	nodeList, err := pgParse(stmt)
	if err != nil {
		return err
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
	default:
		return nil
	}
}

func (d *DatabaseState) pgCreateIndex(node *ast.CreateIndexStmt) *WalkThroughError {
	schema, err := d.getSchema(node.Index.Table.Schema)
	if err != nil {
		return err
	}
	return schema.pgCreateIndex(node, false /* isPrimary */)
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
		indexSet:  make(indexStateMap),
	}
	schema.tableSet[table.name] = table

	for _, column := range node.ColumnList {
		if err := schema.pgCreateColumn(table, column); err != nil {
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
		}, false /* isPrimary */)
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

func (s *SchemaState) pgCreateColumn(t *TableState, column *ast.ColumnDef) *WalkThroughError {
	if _, exists := t.columnSet[column.ColumnName]; exists {
		return &WalkThroughError{
			Type:    ErrorTypeColumnExists,
			Content: fmt.Sprintf("The column %q already exists in table %q", column.ColumnName, t.name),
		}
	}

	pos := len(t.columnSet) + 1
	typeString, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, column.Type)
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
		// TODO(rebelice): support collation and comment here.
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
			}, false /* isPrimary */); err != nil {
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
	}, true /* isPrimary */)
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

func (s *SchemaState) pgCreateIndex(createIndex *ast.CreateIndexStmt, isPrimary bool) *WalkThroughError {
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
		return nil, &WalkThroughError{
			Type:    ErrorTypeSchemaNotExists,
			Content: fmt.Sprintf("The schema %q doesn't exist", schemaName),
		}
	}
	return schema, nil
}

func pgParse(stmt string) ([]ast.Node, error) {
	return parser.Parse(parser.Postgres, parser.ParseContext{}, stmt)
}
