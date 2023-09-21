package pg

import (
	"fmt"
	"sort"
	"strings"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/pkg/errors"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	ast "github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ParseToMetadata converts a schema string to database metadata.
func ParseToMetadata(schema string) (*v1pb.DatabaseMetadata, error) {
	list, err := parser.Parse(parser.Postgres, parser.ParseContext{}, schema)
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
				typeText, err := parser.Deparse(parser.Postgres, parser.DeparseContext{}, column.Type)
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
						columnState.defaultValue = &defaultText
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

				schema.tables[stmt.Name.Name] = table
			}
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
					column.defaultValue = &defaultText
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

func convertToDatabaseState(database *v1pb.DatabaseMetadata) *databaseState {
	state := newDatabaseState()
	state.name = database.Name
	for i, schema := range database.Schemas {
		state.schemas[schema.Name] = convertToSchemaState(i, schema)
	}
	return state
}

func (s *databaseState) convertToDatabaseMetadata() *v1pb.DatabaseMetadata {
	schemaStates := []*schemaState{}
	for _, schema := range s.schemas {
		schemaStates = append(schemaStates, schema)
	}
	sort.Slice(schemaStates, func(i, j int) bool {
		return schemaStates[i].id < schemaStates[j].id
	})
	schemas := []*v1pb.SchemaMetadata{}
	for _, schema := range schemaStates {
		schemas = append(schemas, schema.convertToSchemaMetadata())
	}
	return &v1pb.DatabaseMetadata{
		Name:    s.name,
		Schemas: schemas,
		// Unsupported, for tests only.
		Extensions: []*v1pb.ExtensionMetadata{},
	}
}

type schemaState struct {
	id     int
	name   string
	tables map[string]*tableState
}

func newSchemaState(id int, name string) *schemaState {
	return &schemaState{
		id:     id,
		name:   name,
		tables: make(map[string]*tableState),
	}
}

func convertToSchemaState(id int, schema *v1pb.SchemaMetadata) *schemaState {
	state := newSchemaState(id, schema.Name)
	for i, table := range schema.Tables {
		state.tables[table.Name] = convertToTableState(i, table)
	}
	return state
}

func (s *schemaState) convertToSchemaMetadata() *v1pb.SchemaMetadata {
	tableStates := []*tableState{}
	for _, table := range s.tables {
		tableStates = append(tableStates, table)
	}
	sort.Slice(tableStates, func(i, j int) bool {
		return tableStates[i].id < tableStates[j].id
	})
	tables := []*v1pb.TableMetadata{}
	for _, table := range tableStates {
		tables = append(tables, table.convertToTableMetadata())
	}
	return &v1pb.SchemaMetadata{
		Name:   s.name,
		Tables: tables,
		// Unsupported, for tests only.
		Views:     []*v1pb.ViewMetadata{},
		Functions: []*v1pb.FunctionMetadata{},
		Streams:   []*v1pb.StreamMetadata{},
		Tasks:     []*v1pb.TaskMetadata{},
	}
}

type tableState struct {
	id          int
	name        string
	columns     map[string]*columnState
	indexes     map[string]*indexState
	foreignKeys map[string]*foreignKeyState
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

func (t *tableState) toString(buf *strings.Builder) error {
	if _, err := buf.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n  ", t.name)); err != nil {
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

	indexes := []*indexState{}
	t.removeUnsupportedIndex()
	for _, index := range t.indexes {
		indexes = append(indexes, index)
	}
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i].id < indexes[j].id
	})

	for i, index := range indexes {
		if i+len(columns) > 0 {
			if _, err := buf.WriteString(",\n  "); err != nil {
				return err
			}
		}
		if err := index.toString(buf); err != nil {
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

	for i, fk := range foreignKeys {
		if i+len(columns)+len(indexes) > 0 {
			if _, err := buf.WriteString(",\n  "); err != nil {
				return err
			}
		}
		if err := fk.toString(buf); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString("\n);\n"); err != nil {
		return err
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

func convertToTableState(id int, table *v1pb.TableMetadata) *tableState {
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
	return state
}

func (t *tableState) convertToTableMetadata() *v1pb.TableMetadata {
	columnStates := []*columnState{}
	for _, column := range t.columns {
		columnStates = append(columnStates, column)
	}
	sort.Slice(columnStates, func(i, j int) bool {
		return columnStates[i].id < columnStates[j].id
	})
	columns := []*v1pb.ColumnMetadata{}
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
	indexes := []*v1pb.IndexMetadata{}
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
	fks := []*v1pb.ForeignKeyMetadata{}
	for _, fk := range fkStates {
		fks = append(fks, fk.convertToForeignKeyMetadata())
	}

	return &v1pb.TableMetadata{
		Name:        t.name,
		Columns:     columns,
		Indexes:     indexes,
		ForeignKeys: fks,
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

func (f *foreignKeyState) convertToForeignKeyMetadata() *v1pb.ForeignKeyMetadata {
	return &v1pb.ForeignKeyMetadata{
		Name:              f.name,
		Columns:           f.columns,
		ReferencedTable:   f.referencedTable,
		ReferencedColumns: f.referencedColumns,
	}
}

func convertToForeignKeyState(id int, foreignKey *v1pb.ForeignKeyMetadata) *foreignKeyState {
	return &foreignKeyState{
		id:                id,
		name:              foreignKey.Name,
		columns:           foreignKey.Columns,
		referencedTable:   foreignKey.ReferencedTable,
		referencedColumns: foreignKey.ReferencedColumns,
	}
}

func (f *foreignKeyState) toString(buf *strings.Builder) error {
	if _, err := buf.WriteString("CONSTRAINT `"); err != nil {
		return err
	}
	if _, err := buf.WriteString(f.name); err != nil {
		return err
	}
	if _, err := buf.WriteString("` FOREIGN KEY ("); err != nil {
		return err
	}
	for i, column := range f.columns {
		if i > 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(") REFERENCES `"); err != nil {
		return err
	}
	if _, err := buf.WriteString(f.referencedTable); err != nil {
		return err
	}
	if _, err := buf.WriteString("` ("); err != nil {
		return err
	}
	for i, column := range f.referencedColumns {
		if i > 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(")"); err != nil {
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

func (i *indexState) convertToIndexMetadata() *v1pb.IndexMetadata {
	return &v1pb.IndexMetadata{
		Name:        i.name,
		Expressions: i.keys,
		Primary:     i.primary,
		Unique:      i.unique,
		// Unsupported, for tests only.
		Visible: true,
	}
}

func convertToIndexState(id int, index *v1pb.IndexMetadata) *indexState {
	return &indexState{
		id:      id,
		name:    index.Name,
		keys:    index.Expressions,
		primary: index.Primary,
		unique:  index.Unique,
	}
}

func (i *indexState) toString(buf *strings.Builder) error {
	if i.primary {
		if _, err := buf.WriteString("PRIMARY KEY ("); err != nil {
			return err
		}
		for i, key := range i.keys {
			if i > 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString(fmt.Sprintf("`%s`", key)); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	}
	// TODO: support other type indexes.
	return nil
}

type columnState struct {
	id           int
	name         string
	tp           string
	defaultValue *string
	comment      string
	nullable     bool
}

func (c *columnState) toString(buf *strings.Builder) error {
	if _, err := buf.WriteString(fmt.Sprintf("`%s` %s", c.name, c.tp)); err != nil {
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
	if c.defaultValue != nil {
		if _, err := buf.WriteString(fmt.Sprintf(" DEFAULT %s", *c.defaultValue)); err != nil {
			return err
		}
	}
	if c.comment != "" {
		if _, err := buf.WriteString(fmt.Sprintf(" COMMENT '%s'", c.comment)); err != nil {
			return err
		}
	}
	return nil
}

func (c *columnState) convertToColumnMetadata() *v1pb.ColumnMetadata {
	result := &v1pb.ColumnMetadata{
		Name:     c.name,
		Type:     c.tp,
		Nullable: c.nullable,
		Comment:  c.comment,
	}
	if c.defaultValue != nil {
		result.Default = &wrapperspb.StringValue{Value: *c.defaultValue}
	}
	return result
}

func convertToColumnState(id int, column *v1pb.ColumnMetadata) *columnState {
	result := &columnState{
		id:       id,
		name:     column.Name,
		tp:       column.Type,
		nullable: column.Nullable,
		comment:  column.Comment,
	}
	if column.Default != nil {
		result.defaultValue = &column.Default.Value
	}
	return result
}
