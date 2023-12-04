package pg

import (
	"fmt"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"

	postgres "github.com/bytebase/postgresql-parser"

	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// ParseToMetadata converts a schema string to database metadata.
func ParseToMetadata(schema string) (*storepb.DatabaseSchemaMetadata, error) {
	list, err := pgrawparser.Parse(pgrawparser.ParseContext{}, schema)
	if err != nil {
		return nil, err
	}

	state := newDatabaseState()
	state.schemas["public"] = newSchemaState(0, "public")

	for _, stmt := range list {
		switch stmt := stmt.(type) {
		case *ast.CreateSchemaStmt:
			state.schemas[stmt.Name] = newSchemaState(len(state.schemas), stmt.Name)
		case *ast.CreateTableStmt:
			if stmt.Name.Type == ast.TableTypeView {
				// Skip view for now.
				continue
			}

			schema, ok := state.schemas[stmt.Name.Schema]
			if !ok {
				return nil, errors.Errorf("schema %q not found", stmt.Name.Schema)
			}
			if _, ok := schema.tables[stmt.Name.Name]; ok {
				return nil, errors.Errorf("table %q already exists in schema %q", stmt.Name.Name, stmt.Name.Schema)
			}
			table := newTableState(len(schema.tables), stmt.Name.Name)

			for _, column := range stmt.ColumnList {
				if _, ok := table.columns[column.ColumnName]; ok {
					return nil, errors.Errorf("column %q already exists in table %q.%q", column.ColumnName, stmt.Name.Schema, stmt.Name.Name)
				}
				typeText, err := pgrawparser.Deparse(pgrawparser.DeparseContext{}, column.Type)
				if err != nil {
					return nil, err
				}
				columnState := &columnState{
					id:           len(table.columns),
					name:         column.ColumnName,
					tp:           typeText,
					defaultValue: nil,
					comment:      "",
					nullable:     true,
				}

				for _, constraint := range column.ConstraintList {
					switch constraint.Type {
					case ast.ConstraintTypeNotNull:
						columnState.nullable = false
					case ast.ConstraintTypeDefault:
						defaultText := constraint.Expression.Text()
						columnState.hasDefault = true
						columnState.defaultValue = &defaultValueExpression{value: defaultText}
					}
				}

				table.columns[column.ColumnName] = columnState
			}

			for _, constraint := range stmt.ConstraintList {
				switch constraint.Type {
				case ast.ConstraintTypePrimary:
					if constraint.Name == "" {
						return nil, errors.Errorf("primary key constraint must have a name")
					}
					if _, ok := table.indexes[constraint.Name]; ok {
						return nil, errors.Errorf("index %q already exists in table %q.%q", constraint.Name, stmt.Name.Schema, stmt.Name.Name)
					}
					table.indexes[constraint.Name] = &indexState{
						id:      len(table.indexes),
						name:    constraint.Name,
						keys:    constraint.KeyList,
						primary: true,
						unique:  true,
					}
				case ast.ConstraintTypeForeign:
					if constraint.Name == "" {
						return nil, errors.Errorf("foreign key constraint must have a name")
					}
					if _, ok := table.foreignKeys[constraint.Name]; ok {
						return nil, errors.Errorf("foreign key %q already exists in table %q.%q", constraint.Name, stmt.Name.Schema, stmt.Name.Name)
					}
					table.foreignKeys[constraint.Name] = &foreignKeyState{
						id:                len(table.foreignKeys),
						name:              constraint.Name,
						columns:           constraint.KeyList,
						referencedSchema:  constraint.Foreign.Table.Schema,
						referencedTable:   constraint.Foreign.Table.Name,
						referencedColumns: constraint.Foreign.ColumnList,
					}
				}
			}

			schema.tables[stmt.Name.Name] = table
		case *ast.AlterTableStmt:
			if stmt.Table.Type == ast.TableTypeView {
				// Skip view for now.
				continue
			}
			schema, ok := state.schemas[stmt.Table.Schema]
			if !ok {
				return nil, errors.Errorf("schema %q not found", stmt.Table.Schema)
			}
			table, ok := schema.tables[stmt.Table.Name]
			if !ok {
				return nil, errors.Errorf("table %q not found in schema %q", stmt.Table.Name, stmt.Table.Schema)
			}

			for _, alterItem := range stmt.AlterItemList {
				switch item := alterItem.(type) {
				case *ast.SetDefaultStmt:
					column, ok := table.columns[item.ColumnName]
					if !ok {
						return nil, errors.Errorf("column %q not found in table %q.%q", item.ColumnName, stmt.Table.Schema, stmt.Table.Name)
					}
					defaultText := item.Expression.Text()
					column.defaultValue = &defaultValueExpression{value: defaultText}
					column.hasDefault = true
				case *ast.AddConstraintStmt:
					switch item.Constraint.Type {
					case ast.ConstraintTypePrimary:
						if item.Constraint.Name == "" {
							return nil, errors.Errorf("primary key constraint must have a name")
						}
						if _, ok := table.indexes[item.Constraint.Name]; ok {
							return nil, errors.Errorf("index %q already exists in table %q.%q", item.Constraint.Name, stmt.Table.Schema, stmt.Table.Name)
						}
						table.indexes[item.Constraint.Name] = &indexState{
							id:      len(table.indexes),
							name:    item.Constraint.Name,
							keys:    item.Constraint.KeyList,
							primary: true,
							unique:  true,
						}
					case ast.ConstraintTypeForeign:
						if item.Constraint.Name == "" {
							return nil, errors.Errorf("foreign key constraint must have a name")
						}
						if _, ok := table.foreignKeys[item.Constraint.Name]; ok {
							return nil, errors.Errorf("foreign key %q already exists in table %q.%q", item.Constraint.Name, stmt.Table.Schema, stmt.Table.Name)
						}
						table.foreignKeys[item.Constraint.Name] = &foreignKeyState{
							id:                len(table.foreignKeys),
							name:              item.Constraint.Name,
							columns:           item.Constraint.KeyList,
							referencedSchema:  item.Constraint.Foreign.Table.Schema,
							referencedTable:   item.Constraint.Foreign.Table.Name,
							referencedColumns: item.Constraint.Foreign.ColumnList,
						}
					}
				}
			}
		case *ast.CreateIndexStmt:
			// Not fully supported yet.
			// Only for foreign key check now.
			schema, ok := state.schemas[stmt.Index.Table.Schema]
			if !ok {
				continue
			}
			table, ok := schema.tables[stmt.Index.Table.Name]
			if !ok {
				continue
			}
			table.indexes[stmt.Index.Name] = &indexState{
				id:      len(table.indexes),
				name:    stmt.Index.Name,
				primary: false,
				unique:  stmt.Index.Unique,
			}
		case *ast.CommentStmt:
			switch stmt.Type {
			case ast.ObjectTypeColumn:
				columnDef, ok := stmt.Object.(*ast.ColumnNameDef)
				if !ok {
					return nil, errors.Errorf("failed to convert to ColumnNameDef")
				}
				schema, ok := state.schemas[columnDef.Table.Schema]
				if !ok {
					// Skip unknown schema for comments.
					continue
				}
				table, ok := schema.tables[columnDef.Table.Name]
				if !ok {
					// Skip unknown table for comments.
					continue
				}
				column, ok := table.columns[columnDef.ColumnName]
				if !ok {
					// Skip unknown column for comments.
					continue
				}
				column.comment = stmt.Comment
			case ast.ObjectTypeTable:
				tableDef, ok := stmt.Object.(*ast.TableDef)
				if !ok {
					return nil, errors.Errorf("failed to convert to TableDef")
				}
				schema, ok := state.schemas[tableDef.Schema]
				if !ok {
					// Skip unknown schema for comments.
					continue
				}
				table, ok := schema.tables[tableDef.Name]
				if !ok {
					// Skip unknown table for comments.
					continue
				}
				table.comment = stmt.Comment
			default:
				// Skip other comment types for now.
			}
		}
	}
	return state.convertToDatabaseMetadata(), nil
}

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
	for i, schema := range database.Schemas {
		state.schemas[schema.Name] = convertToSchemaState(i, schema)
	}
	return state
}

func (s *databaseState) convertToDatabaseMetadata() *storepb.DatabaseSchemaMetadata {
	schemaStates := []*schemaState{}
	for _, schema := range s.schemas {
		schemaStates = append(schemaStates, schema)
	}
	sort.Slice(schemaStates, func(i, j int) bool {
		return schemaStates[i].id < schemaStates[j].id
	})
	schemas := []*storepb.SchemaMetadata{}
	for _, schema := range schemaStates {
		schemas = append(schemas, schema.convertToSchemaMetadata())
	}
	return &storepb.DatabaseSchemaMetadata{
		Name:    s.name,
		Schemas: schemas,
		// Unsupported, for tests only.
		Extensions: []*storepb.ExtensionMetadata{},
	}
}

type schemaState struct {
	id int
	// ignore means CREATE SCHEMA statement for this schema is already in the target schema info.
	// But we need the schemaState to deal with the other objects.
	// So we cannot delete the schemaState, instead we set ignore to true.
	ignore bool
	name   string
	tables map[string]*tableState
}

func (s *schemaState) printCreateSchema(buf *strings.Builder) error {
	if s.ignore || s.name == "public" {
		return nil
	}

	_, err := buf.WriteString(fmt.Sprintf("\nCREATE SCHEMA \"%s\";\n\n", s.name))

	return err
}

func newSchemaState(id int, name string) *schemaState {
	return &schemaState{
		id:     id,
		name:   name,
		tables: make(map[string]*tableState),
	}
}

func convertToSchemaState(id int, schema *storepb.SchemaMetadata) *schemaState {
	state := newSchemaState(id, schema.Name)
	for i, table := range schema.Tables {
		state.tables[table.Name] = convertToTableState(i, table)
	}
	return state
}

func (s *schemaState) convertToSchemaMetadata() *storepb.SchemaMetadata {
	tableStates := []*tableState{}
	for _, table := range s.tables {
		tableStates = append(tableStates, table)
	}
	sort.Slice(tableStates, func(i, j int) bool {
		return tableStates[i].id < tableStates[j].id
	})
	tables := []*storepb.TableMetadata{}
	for _, table := range tableStates {
		tables = append(tables, table.convertToTableMetadata())
	}
	return &storepb.SchemaMetadata{
		Name:   s.name,
		Tables: tables,
		// Unsupported, for tests only.
		Views:     []*storepb.ViewMetadata{},
		Functions: []*storepb.FunctionMetadata{},
		Streams:   []*storepb.StreamMetadata{},
		Tasks:     []*storepb.TaskMetadata{},
	}
}

type tableState struct {
	ignoreTable bool
	id          int
	name        string
	columns     map[string]*columnState
	indexes     map[string]*indexState
	foreignKeys map[string]*foreignKeyState
	// ignoreComment means this column is already in the target schema.
	ignoreComment bool
	comment       string
}

func (t *tableState) commentToString(buf *strings.Builder, schemaName string) error {
	if _, err := buf.WriteString(fmt.Sprintf("COMMENT ON TABLE \"%s\".\"%s\" IS '%s';\n", schemaName, t.name, escapePostgreSQLString(t.comment))); err != nil {
		return err
	}
	return nil
}

func (t *tableState) removeUnsupportedIndex() {
	unsupported := []string{}
	for name, index := range t.indexes {
		if index.primary {
			continue
		}
		unsupported = append(unsupported, name)
	}
	for _, name := range unsupported {
		delete(t.indexes, name)
	}
}

func (t *tableState) toString(buf *strings.Builder, schemaName string) error {
	if !t.ignoreTable {
		if _, err := buf.WriteString(fmt.Sprintf("CREATE TABLE \"%s\".\"%s\" (\n  ", schemaName, t.name)); err != nil {
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
		if _, err := buf.WriteString("\n);\n"); err != nil {
			return err
		}
	}

	indexes := []*indexState{}
	t.removeUnsupportedIndex()
	for _, index := range t.indexes {
		indexes = append(indexes, index)
	}
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i].id < indexes[j].id
	})

	for _, index := range indexes {
		if err := index.toString(buf, schemaName, t.name); err != nil {
			return err
		}
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
	}

	foreignKeys := []*foreignKeyState{}
	for _, fk := range t.foreignKeys {
		foreignKeys = append(foreignKeys, fk)
	}
	sort.Slice(foreignKeys, func(i, j int) bool {
		return foreignKeys[i].id < foreignKeys[j].id
	})

	for _, fk := range foreignKeys {
		if err := fk.toString(buf, schemaName, t.name); err != nil {
			return err
		}
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
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
	for i, column := range table.Columns {
		state.columns[column.Name] = convertToColumnState(i, column)
	}
	for i, index := range table.Indexes {
		state.indexes[index.Name] = convertToIndexState(i, index)
	}
	for i, fk := range table.ForeignKeys {
		state.foreignKeys[fk.Name] = convertToForeignKeyState(i, fk)
	}
	state.comment = table.Comment
	return state
}

func (t *tableState) convertToTableMetadata() *storepb.TableMetadata {
	columnStates := []*columnState{}
	for _, column := range t.columns {
		columnStates = append(columnStates, column)
	}
	sort.Slice(columnStates, func(i, j int) bool {
		return columnStates[i].id < columnStates[j].id
	})
	columns := []*storepb.ColumnMetadata{}
	for _, column := range columnStates {
		columns = append(columns, column.convertToColumnMetadata())
	}

	indexStates := []*indexState{}
	for _, index := range t.indexes {
		indexStates = append(indexStates, index)
	}
	sort.Slice(indexStates, func(i, j int) bool {
		return indexStates[i].id < indexStates[j].id
	})
	indexes := []*storepb.IndexMetadata{}
	for _, index := range indexStates {
		indexes = append(indexes, index.convertToIndexMetadata())
	}

	fkStates := []*foreignKeyState{}
	for _, fk := range t.foreignKeys {
		fkStates = append(fkStates, fk)
	}
	sort.Slice(fkStates, func(i, j int) bool {
		return fkStates[i].id < fkStates[j].id
	})
	fks := []*storepb.ForeignKeyMetadata{}
	for _, fk := range fkStates {
		fks = append(fks, fk.convertToForeignKeyMetadata())
	}

	return &storepb.TableMetadata{
		Name:        t.name,
		Columns:     columns,
		Indexes:     indexes,
		ForeignKeys: fks,
		Comment:     t.comment,
	}
}

type foreignKeyState struct {
	id                int
	name              string
	columns           []string
	referencedSchema  string
	referencedTable   string
	referencedColumns []string
}

func (f *foreignKeyState) convertToForeignKeyMetadata() *storepb.ForeignKeyMetadata {
	return &storepb.ForeignKeyMetadata{
		Name:              f.name,
		Columns:           f.columns,
		ReferencedSchema:  f.referencedSchema,
		ReferencedTable:   f.referencedTable,
		ReferencedColumns: f.referencedColumns,
	}
}

func convertToForeignKeyState(id int, foreignKey *storepb.ForeignKeyMetadata) *foreignKeyState {
	return &foreignKeyState{
		id:                id,
		name:              foreignKey.Name,
		columns:           foreignKey.Columns,
		referencedSchema:  foreignKey.ReferencedSchema,
		referencedTable:   foreignKey.ReferencedTable,
		referencedColumns: foreignKey.ReferencedColumns,
	}
}

func (f *foreignKeyState) toString(buf *strings.Builder, schemaName, tableName string) error {
	if _, err := buf.WriteString("ALTER TABLE ONLY \""); err != nil {
		return err
	}
	if _, err := buf.WriteString(schemaName); err != nil {
		return err
	}
	if _, err := buf.WriteString("\".\""); err != nil {
		return err
	}
	if _, err := buf.WriteString(tableName); err != nil {
		return err
	}
	if _, err := buf.WriteString("\"\n    ADD CONSTRAINT \""); err != nil {
		return err
	}
	if _, err := buf.WriteString(f.name); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" FOREIGN KEY (`); err != nil {
		return err
	}
	for i, column := range f.columns {
		if i > 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("\""); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString("\""); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(`) REFERENCES "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(f.referencedSchema); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"."`); err != nil {
		return err
	}
	if _, err := buf.WriteString(f.referencedTable); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"(`); err != nil {
		return err
	}
	for i, column := range f.referencedColumns {
		if i > 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("\""); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString("\""); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(");\n"); err != nil {
		return err
	}
	return nil
}

type indexState struct {
	id      int
	name    string
	keys    []string
	primary bool
	unique  bool
}

func (i *indexState) convertToIndexMetadata() *storepb.IndexMetadata {
	return &storepb.IndexMetadata{
		Name:        i.name,
		Expressions: i.keys,
		Primary:     i.primary,
		Unique:      i.unique,
		// Unsupported, for tests only.
		Visible: true,
	}
}

func convertToIndexState(id int, index *storepb.IndexMetadata) *indexState {
	return &indexState{
		id:      id,
		name:    index.Name,
		keys:    index.Expressions,
		primary: index.Primary,
		unique:  index.Unique,
	}
}

func (i *indexState) toString(buf *strings.Builder, schemaName, tableName string) error {
	if i.primary {
		if _, err := buf.WriteString(fmt.Sprintf("ALTER TABLE ONLY \"%s\".\"%s\"\n    ADD CONSTRAINT \"%s\" PRIMARY KEY (", schemaName, tableName, i.name)); err != nil {
			return err
		}
		for i, key := range i.keys {
			if i > 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString("\""); err != nil {
				return err
			}
			if _, err := buf.WriteString(key); err != nil {
				return err
			}
			if _, err := buf.WriteString("\""); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(");\n"); err != nil {
			return err
		}
	}
	// TODO: support other type indexes.
	return nil
}

type defaultValue interface {
	isDefaultValue()
	toString() string
}

type defaultValueNull struct {
}

func (*defaultValueNull) isDefaultValue() {}
func (*defaultValueNull) toString() string {
	return "NULL"
}

type defaultValueString struct {
	value string
}

func (*defaultValueString) isDefaultValue() {}
func (d *defaultValueString) toString() string {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(d.value, "'", "''"))
}

type defaultValueExpression struct {
	value string
}

func (*defaultValueExpression) isDefaultValue() {}
func (d *defaultValueExpression) toString() string {
	return d.value
}

type columnState struct {
	// ignore means this column is already in the target schema.
	// But we need the columnState to deal with the comment statement.
	// So we cannot delete the columnState, instead we set ignore to true.
	ignore bool
	// ignoreComment means this column is already in the target schema.
	ignoreComment bool
	id            int
	name          string
	tp            string
	hasDefault    bool
	defaultValue  defaultValue
	comment       string
	nullable      bool
}

func (c *columnState) commentToString(buf *strings.Builder, schemaName, tableName string) error {
	if _, err := buf.WriteString(fmt.Sprintf("COMMENT ON COLUMN \"%s\".\"%s\".\"%s\" IS '%s';\n", schemaName, tableName, c.name, escapePostgreSQLString(c.comment))); err != nil {
		return err
	}
	return nil
}

func (c *columnState) toString(buf *strings.Builder) error {
	if _, err := buf.WriteString(fmt.Sprintf(`"%s" %s`, c.name, c.tp)); err != nil {
		return err
	}
	if c.nullable {
		if _, err := buf.WriteString(" NULL"); err != nil {
			return err
		}
	} else {
		if _, err := buf.WriteString(" NOT NULL"); err != nil {
			return err
		}
	}
	if c.hasDefault {
		if _, err := buf.WriteString(fmt.Sprintf(" DEFAULT %s", c.defaultValue.toString())); err != nil {
			return err
		}
	}
	return nil
}

func (c *columnState) convertToColumnMetadata() *storepb.ColumnMetadata {
	result := &storepb.ColumnMetadata{
		Name:     c.name,
		Type:     c.tp,
		Nullable: c.nullable,
		Comment:  c.comment,
	}
	if c.hasDefault {
		switch value := c.defaultValue.(type) {
		case *defaultValueNull:
			result.DefaultValue = &storepb.ColumnMetadata_DefaultNull{DefaultNull: true}
		case *defaultValueString:
			result.DefaultValue = &storepb.ColumnMetadata_Default{Default: wrapperspb.String(value.value)}
		case *defaultValueExpression:
			result.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: value.value}
		}
	}
	return result
}

func convertToColumnState(id int, column *storepb.ColumnMetadata) *columnState {
	result := &columnState{
		id:         id,
		name:       column.Name,
		tp:         column.Type,
		hasDefault: column.GetDefaultValue() != nil,
		nullable:   column.Nullable,
		comment:    column.Comment,
	}
	if result.hasDefault {
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

type designSchemaGenerator struct {
	*postgres.BasePostgreSQLParserListener

	to                  *databaseState
	result              strings.Builder
	currentTable        *tableState
	firstElementInTable bool
	columnDefine        strings.Builder
	tableConstraints    strings.Builder
	err                 error

	lastTokenIndex int
}

// GetDesignSchema returns the schema string for the design schema.
func GetDesignSchema(baselineSchema string, to *storepb.DatabaseSchemaMetadata) (string, error) {
	toState := convertToDatabaseState(to)
	parseResult, err := pgparser.ParsePostgreSQL(baselineSchema)
	if err != nil {
		return "", err
	}
	if parseResult == nil {
		return "", nil
	}
	if parseResult.Tree == nil {
		return "", nil
	}

	listener := &designSchemaGenerator{
		lastTokenIndex: 0,
		to:             toState,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, parseResult.Tree)
	if listener.err != nil {
		return "", listener.err
	}
	root, ok := parseResult.Tree.(*postgres.RootContext)
	if !ok {
		return "", errors.Errorf("failed to convert to RootContext")
	}
	if root.GetStop() != nil {
		if _, err := listener.result.WriteString(root.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: listener.lastTokenIndex,
			Stop:  root.GetStop().GetTokenIndex(),
		})); err != nil {
			return "", err
		}
	}

	// Follow the order of the input schema.
	for _, schema := range to.Schemas {
		schemaState, exists := listener.to.schemas[schema.Name]
		if !exists {
			continue
		}
		if err := schemaState.printCreateSchema(&listener.result); err != nil {
			return "", err
		}
		// Follow the order of the input table.
		for _, table := range schema.Tables {
			tableState, exists := schemaState.tables[table.Name]
			if !exists {
				continue
			}
			if err := tableState.toString(&listener.result, schema.Name); err != nil {
				return "", err
			}

			if _, err := listener.result.WriteString("\n"); err != nil {
				return "", err
			}

			for _, column := range table.Columns {
				columnState, exists := tableState.columns[column.Name]
				if !exists {
					continue
				}

				if column.Comment != "" && !columnState.ignoreComment {
					if err := columnState.commentToString(&listener.result, schema.Name, table.Name); err != nil {
						return "", err
					}
					if _, err := listener.result.WriteString("\n"); err != nil {
						return "", err
					}
				}
			}

			if table.Comment != "" && !tableState.ignoreComment {
				if err := tableState.commentToString(&listener.result, schema.Name); err != nil {
					return "", err
				}
				if _, err := listener.result.WriteString("\n"); err != nil {
					return "", err
				}
			}
		}
	}

	return listener.result.String(), nil
}

// EnterCreatestmt is called when production createstmt is entered.
func (g *designSchemaGenerator) EnterCreatestmt(ctx *postgres.CreatestmtContext) {
	if g.err != nil {
		return
	}
	if ctx.Opttableelementlist() == nil {
		// Skip other create statement for now.
		return
	}
	schemaName, tableName, err := pgparser.NormalizePostgreSQLQualifiedNameAsTableName(ctx.Qualified_name(0))
	if err != nil {
		g.err = err
		return
	}

	if _, err := g.result.WriteString(
		ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: g.lastTokenIndex,
			Stop:  ctx.GetStart().GetTokenIndex() - 1,
		}),
	); err != nil {
		g.err = err
		return
	}
	g.lastTokenIndex = ctx.GetStart().GetTokenIndex()

	schema, exists := g.to.schemas[schemaName]
	if !exists {
		// Skip not found schema.
		g.lastTokenIndex = skipFollowingSemiIndex(ctx.GetParser().GetTokenStream(), ctx.GetStop().GetTokenIndex()+1)
		return
	}

	table, exists := schema.tables[tableName]
	if !exists {
		// Skip not found table.
		g.lastTokenIndex = skipFollowingSemiIndex(ctx.GetParser().GetTokenStream(), ctx.GetStop().GetTokenIndex()+1)
		return
	}

	g.currentTable = table
	g.firstElementInTable = true
	g.columnDefine.Reset()
	g.tableConstraints.Reset()

	table.ignoreTable = true
	// Write the text before the table element list.
	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.GetStart().GetTokenIndex(),
		Stop:  ctx.Opttableelementlist().GetStart().GetTokenIndex() - 1,
	})); err != nil {
		g.err = err
		return
	}
}

func (g *designSchemaGenerator) ExitCreatestmt(ctx *postgres.CreatestmtContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	var columnList []*columnState
	for _, column := range g.currentTable.columns {
		if column.ignore {
			continue
		}
		columnList = append(columnList, column)
	}
	sort.Slice(columnList, func(i, j int) bool {
		return columnList[i].id < columnList[j].id
	})
	for _, column := range columnList {
		if g.firstElementInTable {
			g.firstElementInTable = false
		} else {
			if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
				g.err = err
				return
			}
		}
		if err := column.toString(&g.columnDefine); err != nil {
			g.err = err
			return
		}
	}

	if _, err := g.result.WriteString(g.columnDefine.String()); err != nil {
		g.err = err
		return
	}
	if _, err := g.result.WriteString(g.tableConstraints.String()); err != nil {
		g.err = err
		return
	}

	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.Opttableelementlist().GetStop().GetTokenIndex() + 1,
		Stop:  ctx.GetStop().GetTokenIndex(),
	})); err != nil {
		g.err = err
		return
	}
	g.lastTokenIndex = ctx.GetStop().GetTokenIndex() + 1
	g.currentTable = nil
	g.firstElementInTable = false
}

func skipFollowingSemiIndex(stream antlr.TokenStream, index int) int {
	for i := index; i < stream.Size(); i++ {
		token := stream.Get(i)
		if token.GetTokenType() == postgres.PostgreSQLParserSEMI {
			return i + 1
		}
		if token.GetTokenType() == postgres.PostgreSQLParserEOF {
			return i
		}
	}
	return index
}

func (g *designSchemaGenerator) EnterAltertablestmt(ctx *postgres.AltertablestmtContext) {
	if g.err != nil {
		return
	}

	if ctx.TABLE() == nil || ctx.Alter_table_cmds() == nil || len(ctx.Alter_table_cmds().AllAlter_table_cmd()) != 1 {
		// Skip other alter table statement for now.
		return
	}

	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: g.lastTokenIndex,
		Stop:  ctx.GetStart().GetTokenIndex() - 1,
	})); err != nil {
		g.err = err
		return
	}
	g.lastTokenIndex = skipFollowingSemiIndex(ctx.GetParser().GetTokenStream(), ctx.GetStop().GetTokenIndex()+1)

	schemaName, tableName, err := pgparser.NormalizePostgreSQLQualifiedNameAsTableName(ctx.Relation_expr().Qualified_name())
	if err != nil {
		g.err = err
		return
	}

	schema, exists := g.to.schemas[schemaName]
	if !exists {
		// Skip not found schema.
		return
	}

	table, exists := schema.tables[tableName]
	if !exists {
		// Skip not found table.
		return
	}

	cmd := ctx.Alter_table_cmds().Alter_table_cmd(0)
	switch {
	case cmd.ADD_P() != nil && cmd.Tableconstraint() != nil:
		constraint := cmd.Tableconstraint().Constraintelem()
		switch {
		case constraint.PRIMARY() != nil && constraint.KEY() != nil:
			name := cmd.Tableconstraint().Name()
			if name == nil {
				g.err = errors.Errorf("primary key constraint must have a name")
				return
			}
			nameText := pgparser.NormalizePostgreSQLColid(name.Colid())
			index, exists := table.indexes[nameText]
			if !exists {
				// Skip not found primary key.
				return
			}
			delete(table.indexes, nameText)
			keys := extractColumnList(constraint.Columnlist())
			if equalKeys(keys, index.keys) {
				if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: ctx.GetStart().GetTokenIndex(),
					Stop:  g.lastTokenIndex - 1,
				})); err != nil {
					g.err = err
					return
				}
			} else {
				if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: ctx.GetStart().GetTokenIndex(),
					Stop:  constraint.Columnlist().GetStart().GetTokenIndex() - 1,
				})); err != nil {
					g.err = err
					return
				}
				newKeys := []string{}
				for _, key := range index.keys {
					newKeys = append(newKeys, fmt.Sprintf(`"%s"`, key))
				}
				if _, err := g.result.WriteString(strings.Join(newKeys, ", ")); err != nil {
					g.err = err
					return
				}
				if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: constraint.Columnlist().GetStop().GetTokenIndex() + 1,
					Stop:  g.lastTokenIndex - 1,
				})); err != nil {
					g.err = err
					return
				}
			}
		case constraint.FOREIGN() != nil && constraint.KEY() != nil:
			name := cmd.Tableconstraint().Name()
			if name == nil {
				g.err = errors.Errorf("foreign key constraint must have a name")
				return
			}
			nameText := pgparser.NormalizePostgreSQLColid(name.Colid())
			fk, exists := table.foreignKeys[nameText]
			if !exists {
				// Skip not found foreign key.
				return
			}
			delete(table.foreignKeys, nameText)
			columns := extractColumnList(constraint.Columnlist())
			referencedSchemaName, referencedTableName, err := pgparser.NormalizePostgreSQLQualifiedNameAsTableName(constraint.Qualified_name())
			if err != nil {
				g.err = err
				return
			}
			referencedColumns := extractColumnList(constraint.Opt_column_list().Columnlist())
			equal := equalKeys(columns, fk.columns) && equalKeys(referencedColumns, fk.referencedColumns) && referencedSchemaName == fk.referencedSchema && referencedTableName == fk.referencedTable
			if equal {
				if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: ctx.GetStart().GetTokenIndex(),
					Stop:  g.lastTokenIndex - 1,
				})); err != nil {
					g.err = err
					return
				}
			} else {
				if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: ctx.GetStart().GetTokenIndex(),
					Stop:  constraint.Columnlist().GetStart().GetTokenIndex() - 1,
				})); err != nil {
					g.err = err
					return
				}
				newColumns := []string{}
				for _, column := range fk.columns {
					newColumns = append(newColumns, fmt.Sprintf(`"%s"`, column))
				}
				if _, err := g.result.WriteString(strings.Join(newColumns, ", ")); err != nil {
					g.err = err
					return
				}
				if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: constraint.Columnlist().GetStop().GetTokenIndex() + 1,
					Stop:  constraint.Qualified_name().GetStart().GetTokenIndex() - 1,
				})); err != nil {
					g.err = err
					return
				}
				if _, err := g.result.WriteString(fmt.Sprintf(`"%s"."%s"(`, fk.referencedSchema, fk.referencedTable)); err != nil {
					g.err = err
					return
				}
				newReferencedColumns := []string{}
				for _, column := range fk.referencedColumns {
					newReferencedColumns = append(newReferencedColumns, fmt.Sprintf(`"%s"`, column))
				}
				if _, err := g.result.WriteString(strings.Join(newReferencedColumns, ", ")); err != nil {
					g.err = err
					return
				}
				if _, err := g.result.WriteString(")"); err != nil {
					g.err = err
					return
				}
				if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: constraint.Opt_column_list().GetStop().GetTokenIndex() + 1,
					Stop:  g.lastTokenIndex - 1,
				})); err != nil {
					g.err = err
					return
				}
			}
		default:
			if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.GetStart().GetTokenIndex(),
				Stop:  g.lastTokenIndex - 1,
			})); err != nil {
				g.err = err
				return
			}
		}
	default:
		if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.GetStart().GetTokenIndex(),
			Stop:  g.lastTokenIndex - 1,
		})); err != nil {
			g.err = err
			return
		}
	}
}

func equalKeys(keys1, keys2 []string) bool {
	if len(keys1) != len(keys2) {
		return false
	}
	for i, key := range keys1 {
		if key != keys2[i] {
			return false
		}
	}
	return true
}

func extractColumnList(columnList postgres.IColumnlistContext) []string {
	result := []string{}
	for _, item := range columnList.AllColumnElem() {
		result = append(result, pgparser.NormalizePostgreSQLColid(item.Colid()))
	}
	return result
}

func (g *designSchemaGenerator) EnterTableconstraint(ctx *postgres.TableconstraintContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	if g.firstElementInTable {
		g.firstElementInTable = false
	} else {
		if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
			g.err = err
			return
		}
	}
	if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.GetStart().GetTokenIndex(),
		Stop:  ctx.GetStop().GetTokenIndex(),
	})); err != nil {
		g.err = err
		return
	}
}

func (g *designSchemaGenerator) EnterTablelikeclause(ctx *postgres.TablelikeclauseContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	if g.firstElementInTable {
		g.firstElementInTable = false
	} else {
		if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
			g.err = err
			return
		}
	}
	if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.GetStart().GetTokenIndex(),
		Stop:  ctx.GetStop().GetTokenIndex(),
	})); err != nil {
		g.err = err
		return
	}
}

func (g *designSchemaGenerator) EnterCreateschemastmt(ctx *postgres.CreateschemastmtContext) {
	if g.err != nil {
		return
	}
	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: g.lastTokenIndex,
		Stop:  ctx.GetStart().GetTokenIndex() - 1,
	})); err != nil {
		g.err = err
		return
	}

	g.lastTokenIndex = skipFollowingSemiIndex(ctx.GetParser().GetTokenStream(), ctx.GetStop().GetTokenIndex()+1)
	endTokenIndex := g.lastTokenIndex - 1

	var schemaName string
	if ctx.Colid() != nil {
		schemaName = pgparser.NormalizePostgreSQLColid(ctx.Colid())
	} else if ctx.Optschemaname() != nil && ctx.Optschemaname().Colid() != nil {
		schemaName = pgparser.NormalizePostgreSQLColid(ctx.Optschemaname().Colid())
	}

	schema, exists := g.to.schemas[schemaName]
	if !exists {
		// Skip not found schema.
		return
	}

	schema.ignore = true
	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.GetStart().GetTokenIndex(),
		Stop:  endTokenIndex,
	})); err != nil {
		g.err = err
		return
	}
}

func (g *designSchemaGenerator) EnterCommentstmt(ctx *postgres.CommentstmtContext) {
	if g.err != nil {
		return
	}
	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: g.lastTokenIndex,
		Stop:  ctx.GetStart().GetTokenIndex() - 1,
	})); err != nil {
		g.err = err
		return
	}

	g.lastTokenIndex = skipFollowingSemiIndex(ctx.GetParser().GetTokenStream(), ctx.GetStop().GetTokenIndex()+1)
	endTokenIndex := g.lastTokenIndex - 1

	if ctx.Object_type_any_name() != nil {
		if ctx.Object_type_any_name().TABLE() != nil && ctx.Object_type_any_name().FOREIGN() == nil {
			schemaName, tableName, err := pgparser.NormalizePostgreSQLAnyNameAsTableName(ctx.Any_name())
			if err != nil {
				g.err = err
				return
			}
			schema, exists := g.to.schemas[schemaName]
			if !exists {
				// Skip not found schema.
				return
			}
			_, exists = schema.tables[tableName]
			if !exists {
				// Skip not found table.
				return
			}
		}
	}

	switch {
	case ctx.COLUMN() != nil:
		schemaName, tableName, columnName, err := pgparser.NormalizePostgreSQLAnyNameAsColumnName(ctx.Any_name())
		if err != nil {
			g.err = err
			return
		}
		schema, exists := g.to.schemas[schemaName]
		if !exists {
			// Skip not found schema.
			return
		}
		table, exists := schema.tables[tableName]
		if !exists {
			// Skip not found table.
			return
		}
		column, exists := table.columns[columnName]
		if !exists {
			// Skip not found column.
			return
		}
		equal := false
		column.ignoreComment = true
		if ctx.Comment_text().NULL_P() != nil {
			equal = len(column.comment) == 0
		} else {
			if len(column.comment) == 0 {
				// Skip for empty comment string.
				return
			}
			commentText := ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.Comment_text().GetStart().GetTokenIndex(),
				Stop:  ctx.Comment_text().GetStop().GetTokenIndex(),
			})
			if len(commentText) > 2 && commentText[0] == '\'' && commentText[len(commentText)-1] == '\'' {
				commentText = unescapePostgreSQLString(commentText[1 : len(commentText)-1])
			}

			equal = commentText == column.comment
		}

		if equal {
			if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.GetStart().GetTokenIndex(),
				Stop:  endTokenIndex,
			})); err != nil {
				g.err = err
				return
			}
		} else {
			if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.GetStart().GetTokenIndex(),
				Stop:  ctx.Comment_text().GetStart().GetTokenIndex() - 1,
			})); err != nil {
				g.err = err
				return
			}
			if _, err := g.result.WriteString(fmt.Sprintf("'%s'", escapePostgreSQLString(column.comment))); err != nil {
				g.err = err
				return
			}
			if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.Comment_text().GetStop().GetTokenIndex() + 1,
				Stop:  endTokenIndex,
			})); err != nil {
				g.err = err
				return
			}
		}
	case ctx.Object_type_any_name() != nil && ctx.Object_type_any_name().TABLE() != nil:
		schemaName, tableName, err := pgparser.NormalizePostgreSQLAnyNameAsTableName(ctx.Any_name())
		if err != nil {
			g.err = err
			return
		}
		schema, exists := g.to.schemas[schemaName]
		if !exists {
			// Skip not found schema.
			return
		}
		table, exists := schema.tables[tableName]
		if !exists {
			// Skip not found table.
			return
		}
		equal := false
		table.ignoreComment = true
		if ctx.Comment_text().NULL_P() != nil {
			equal = len(table.comment) == 0
		} else {
			if len(table.comment) == 0 {
				// Skip for empty comment string.
				return
			}
			commentText := ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.Comment_text().GetStart().GetTokenIndex(),
				Stop:  ctx.Comment_text().GetStop().GetTokenIndex(),
			})
			if len(commentText) > 2 && commentText[0] == '\'' && commentText[len(commentText)-1] == '\'' {
				commentText = unescapePostgreSQLString(commentText[1 : len(commentText)-1])
			}

			equal = commentText == table.comment
		}

		if equal {
			if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.GetStart().GetTokenIndex(),
				Stop:  endTokenIndex,
			})); err != nil {
				g.err = err
				return
			}
		} else {
			if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.GetStart().GetTokenIndex(),
				Stop:  ctx.Comment_text().GetStart().GetTokenIndex() - 1,
			})); err != nil {
				g.err = err
				return
			}
			if _, err := g.result.WriteString(fmt.Sprintf("'%s'", escapePostgreSQLString(table.comment))); err != nil {
				g.err = err
				return
			}
			if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.Comment_text().GetStop().GetTokenIndex() + 1,
				Stop:  endTokenIndex,
			})); err != nil {
				g.err = err
				return
			}
		}
	default:
		// Keep other comment statements.
		if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.GetStart().GetTokenIndex(),
			Stop:  endTokenIndex,
		})); err != nil {
			g.err = err
			return
		}
		return
	}
}

func escapePostgreSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func unescapePostgreSQLString(s string) string {
	return strings.ReplaceAll(s, "''", "'")
}

func (g *designSchemaGenerator) EnterColumnDef(ctx *postgres.ColumnDefContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	columnName := pgparser.NormalizePostgreSQLColid(ctx.Colid())
	column, exists := g.currentTable.columns[columnName]
	if !exists {
		return
	}
	column.ignore = true

	if g.firstElementInTable {
		g.firstElementInTable = false
	} else {
		if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
			g.err = err
			return
		}
	}

	// compare column type
	columnType := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(
		ctx.Typename(),
	)
	equal, err := equalType(column.tp, columnType)
	if err != nil {
		g.err = err
		return
	}
	if !equal {
		if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.GetStart().GetTokenIndex(),
			Stop:  ctx.Typename().GetStart().GetTokenIndex() - 1,
		})); err != nil {
			g.err = err
			return
		}
		if _, err := g.columnDefine.WriteString(column.tp); err != nil {
			g.err = err
			return
		}
	} else {
		if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.GetStart().GetTokenIndex(),
			Stop:  ctx.Typename().GetStop().GetTokenIndex(),
		})); err != nil {
			g.err = err
			return
		}
	}
	needOneSpace := false

	// if there are other tokens between column type and column constraint, write them.
	if ctx.Colquallist().GetStop().GetTokenIndex() > ctx.Colquallist().GetStart().GetTokenIndex() {
		if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.Typename().GetStop().GetTokenIndex() + 1,
			Stop:  ctx.Colquallist().GetStart().GetTokenIndex() - 1,
		})); err != nil {
			g.err = err
			return
		}
	} else {
		needOneSpace = true
	}
	startPos := ctx.Colquallist().GetStart().GetTokenIndex()

	if !column.nullable && !nullableExists(ctx.Colquallist()) {
		if needOneSpace {
			if _, err := g.columnDefine.WriteString(" "); err != nil {
				g.err = err
				return
			}
		}
		if _, err := g.columnDefine.WriteString("NOT NULL"); err != nil {
			g.err = err
			return
		}
		needOneSpace = true
	}

	if column.hasDefault && !defaultExists(ctx.Colquallist()) {
		if needOneSpace {
			if _, err := g.columnDefine.WriteString(" "); err != nil {
				g.err = err
				return
			}
		}
		if _, err := g.columnDefine.WriteString(fmt.Sprintf("DEFAULT %s", column.defaultValue.toString())); err != nil {
			g.err = err
			return
		}
		needOneSpace = true
	}

	for i, item := range ctx.Colquallist().AllColconstraint() {
		if i == 0 && needOneSpace {
			if _, err := g.columnDefine.WriteString(" "); err != nil {
				g.err = err
				return
			}
		}
		if item.Colconstraintelem() == nil {
			if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(
				antlr.Interval{
					Start: startPos,
					Stop:  item.GetStop().GetTokenIndex(),
				},
			)); err != nil {
				g.err = err
				return
			}
			startPos = item.GetStop().GetTokenIndex() + 1
			continue
		}

		constraint := item.Colconstraintelem()

		switch {
		case constraint.NULL_P() != nil:
			sameNullable := (constraint.NOT() == nil && column.nullable) || (constraint.NOT() != nil && !column.nullable)
			if sameNullable {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(
					antlr.Interval{
						Start: startPos,
						Stop:  item.GetStop().GetTokenIndex(),
					},
				)); err != nil {
					g.err = err
					return
				}
			}
		case constraint.DEFAULT() != nil:
			defaultValue := ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: constraint.B_expr().GetStart().GetTokenIndex(),
				Stop:  constraint.B_expr().GetStop().GetTokenIndex(),
			})
			if column.hasDefault && column.defaultValue.toString() == defaultValue {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(
					antlr.Interval{
						Start: startPos,
						Stop:  item.GetStop().GetTokenIndex(),
					},
				)); err != nil {
					g.err = err
					return
				}
			} else if column.hasDefault {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(
					antlr.Interval{
						Start: startPos,
						Stop:  constraint.B_expr().GetStart().GetTokenIndex() - 1,
					},
				)); err != nil {
					g.err = err
					return
				}
				if _, err := g.columnDefine.WriteString(column.defaultValue.toString()); err != nil {
					g.err = err
					return
				}
			}
		default:
			if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(
				antlr.Interval{
					Start: startPos,
					Stop:  item.GetStop().GetTokenIndex(),
				},
			)); err != nil {
				g.err = err
				return
			}
		}
		startPos = item.GetStop().GetTokenIndex() + 1
	}
}

func defaultExists(colquallist postgres.IColquallistContext) bool {
	if colquallist == nil {
		return false
	}

	for _, item := range colquallist.AllColconstraint() {
		if item.Colconstraintelem() == nil {
			continue
		}

		if item.Colconstraintelem().DEFAULT() != nil {
			return true
		}
	}

	return false
}

func nullableExists(colquallist postgres.IColquallistContext) bool {
	if colquallist == nil {
		return false
	}

	for _, item := range colquallist.AllColconstraint() {
		if item.Colconstraintelem() == nil {
			continue
		}

		if item.Colconstraintelem().NULL_P() != nil {
			return true
		}
	}

	return false
}

func equalType(typeA, typeB string) (bool, error) {
	list, err := pgrawparser.Parse(pgrawparser.ParseContext{}, fmt.Sprintf("CREATE TABLE t (a %s)", typeA))
	if err != nil {
		return false, err
	}
	if len(list) != 1 {
		return false, errors.Errorf("failed to compare type %q and %q: more than one statement", typeA, typeB)
	}
	node, ok := list[0].(*ast.CreateTableStmt)
	if !ok {
		return false, errors.Errorf("failed to compare type %q and %q: not CreateTableStmt", typeA, typeB)
	}
	if len(node.ColumnList) != 1 {
		return false, errors.Errorf("failed to compare type %q and %q: more than one column", typeA, typeB)
	}
	column := node.ColumnList[0]
	return column.Type.EquivalentType(typeB), nil
}
