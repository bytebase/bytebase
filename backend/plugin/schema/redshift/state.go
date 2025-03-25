package redshift

import (
	"fmt"
	"io"
	"sort"
	"strings"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

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
	for _, schema := range database.Schemas {
		state.schemas[schema.Name] = convertToSchemaState(schema)
	}
	return state
}

type schemaState struct {
	name   string
	tables map[string]*tableState
	views  map[string]*viewState
}

func newSchemaState() *schemaState {
	return &schemaState{
		tables: make(map[string]*tableState),
		views:  make(map[string]*viewState),
	}
}

func convertToSchemaState(schema *storepb.SchemaMetadata) *schemaState {
	state := newSchemaState()
	state.name = schema.Name
	for i, table := range schema.Tables {
		state.tables[table.Name] = convertToTableState(i, table)
	}
	for i, view := range schema.Views {
		state.views[view.Name] = convertToViewState(i, view)
	}
	return state
}

type tableState struct {
	id          int
	name        string
	columns     map[string]*columnState
	indexes     map[string]*indexState
	foreignKeys map[string]*foreignKeyState
	comment     string
}

func (t *tableState) toString(buf *strings.Builder) error {
	if _, err := fmt.Fprintf(buf, "CREATE TABLE %s (\n  ", t.name); err != nil {
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
	for _, index := range t.indexes {
		indexes = append(indexes, index)
	}
	sort.Slice(indexes, func(i, j int) bool {
		if indexes[i].primary {
			return true
		}
		if indexes[j].primary {
			return false
		}
		return indexes[i].name < indexes[j].name
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
		return foreignKeys[i].name < foreignKeys[j].name
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

func convertToTableState(id int, table *storepb.TableMetadata) *tableState {
	state := newTableState(id, table.Name)
	state.comment = table.Comment
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

type foreignKeyState struct {
	id                int
	name              string
	columns           []string
	referencedTable   string
	referencedColumns []string
}

func convertToForeignKeyState(id int, foreignKey *storepb.ForeignKeyMetadata) *foreignKeyState {
	return &foreignKeyState{
		id:                id,
		name:              foreignKey.Name,
		columns:           foreignKey.Columns,
		referencedTable:   foreignKey.ReferencedTable,
		referencedColumns: foreignKey.ReferencedColumns,
	}
}

func (f *foreignKeyState) toString(buf *strings.Builder) error {
	for i, column := range f.columns {
		if _, err := buf.WriteString("FOREIGN KEY ("); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString(") REFERENCES "); err != nil {
			return err
		}
		referencedColumn := f.referencedColumns[i]
		if _, err := fmt.Fprintf(buf, "%s(%s)", f.referencedTable, referencedColumn); err != nil {
			return err
		}
	}
	return nil
}

type indexState struct {
	id      int
	name    string
	keys    []string
	lengths []int64
	primary bool
	unique  bool
	tp      string
}

func convertToIndexState(id int, index *storepb.IndexMetadata) *indexState {
	return &indexState{
		id:      id,
		name:    index.Name,
		keys:    index.Expressions,
		lengths: index.KeyLength,
		primary: index.Primary,
		unique:  index.Unique,
		tp:      index.Type,
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
			if _, err := buf.WriteString(key); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	} else if i.unique {
		if _, err := buf.WriteString("UNIQUE ("); err != nil {
			return err
		}
		for i, key := range i.keys {
			if i > 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString(key); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	}
	return nil
}

type defaultValue interface {
	toString() string
}

type defaultValueNull struct {
}

func (*defaultValueNull) toString() string {
	return "NULL"
}

type defaultValueString struct {
	value string
}

func (d *defaultValueString) toString() string {
	return fmt.Sprintf("'%s'", d.value)
}

type defaultValueExpression struct {
	value string
}

func (d *defaultValueExpression) toString() string {
	return d.value
}

type columnState struct {
	id           int
	name         string
	tp           string
	defaultValue defaultValue
	comment      string
	nullable     bool
}

func (c *columnState) toString(buf *strings.Builder) error {
	if _, err := fmt.Fprintf(buf, "%s ", c.name); err != nil {
		return err
	}
	if _, err := buf.WriteString(c.tp); err != nil {
		return err
	}
	if !c.nullable {
		if _, err := buf.WriteString(" NOT NULL"); err != nil {
			return err
		}
	}
	if c.defaultValue != nil {
		if _, err := fmt.Fprintf(buf, " DEFAULT %s", c.defaultValue.toString()); err != nil {
			return err
		}
	}
	return nil
}

func convertToColumnState(id int, column *storepb.ColumnMetadata) *columnState {
	result := &columnState{
		id:       id,
		name:     column.Name,
		tp:       column.Type,
		nullable: column.Nullable,
		comment:  column.Comment,
	}
	if column.GetDefaultValue() != nil {
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

type viewState struct {
	id         int
	name       string
	definition string
	comment    string
}

func convertToViewState(id int, view *storepb.ViewMetadata) *viewState {
	return &viewState{
		id:         id,
		name:       view.Name,
		definition: view.Definition,
		comment:    view.Comment,
	}
}

func (v *viewState) toString(buf io.StringWriter) error {
	stmt := fmt.Sprintf("CREATE OR REPLACE VIEW %s AS %s", v.name, v.definition)
	if !strings.HasSuffix(stmt, ";") {
		stmt += ";"
	}
	stmt += "\n"
	if _, err := buf.WriteString(stmt); err != nil {
		return err
	}
	return nil
}
