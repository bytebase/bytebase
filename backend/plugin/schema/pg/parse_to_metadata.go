package pg

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bytebase/bytebase/backend/plugin/schema"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterParseToMetadatas(storepb.Engine_POSTGRES, ParseToMetadata)
}

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
				case ast.ConstraintTypeUnique:
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
				id:         len(table.indexes),
				name:       stmt.Index.Name,
				primary:    false,
				unique:     stmt.Index.Unique,
				keys:       stmt.Index.GetKeyNameList(),
				definition: stmt.Text(),
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
		Views:             []*storepb.ViewMetadata{},
		Functions:         []*storepb.FunctionMetadata{},
		Streams:           []*storepb.StreamMetadata{},
		Tasks:             []*storepb.TaskMetadata{},
		MaterializedViews: []*storepb.MaterializedViewMetadata{},
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
	id         int
	name       string
	keys       []string
	primary    bool
	unique     bool
	definition string
}

func (i *indexState) convertToIndexMetadata() *storepb.IndexMetadata {
	return &storepb.IndexMetadata{
		Name:        i.name,
		Expressions: i.keys,
		Primary:     i.primary,
		Unique:      i.unique,
		Definition:  i.definition,
		// Unsupported, for tests only.
		Visible: true,
	}
}

func convertToIndexState(id int, index *storepb.IndexMetadata) *indexState {
	return &indexState{
		id:         id,
		name:       index.Name,
		keys:       index.Expressions,
		primary:    index.Primary,
		unique:     index.Unique,
		definition: index.Definition,
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
	} else {
		if _, err := buf.WriteString(i.definition); err != nil {
			return err
		}
	}
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
