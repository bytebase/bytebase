package pg

import (
	"fmt"
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
	for i, schema := range database.Schemas {
		state.schemas[schema.Name] = convertToSchemaState(i, schema)
	}
	return state
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

type foreignKeyState struct {
	id                int
	name              string
	columns           []string
	referencedSchema  string
	referencedTable   string
	referencedColumns []string
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
