package oracle

import (
	"fmt"
	"sort"
	"strings"

	"google.golang.org/protobuf/types/known/wrapperspb"

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

// nolint
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

// nolint
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
	deleted bool
	id      int
	name    string
	columns map[string]*columnState
	indexes map[string]*indexState
	comment string
}

func (t *tableState) toString(schemaName string, buf *strings.Builder) error {
	if _, err := buf.WriteString(`CREATE TABLE "`); err != nil {
		return err
	}
	if schemaName != "" {
		if _, err := buf.WriteString(schemaName); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"."`); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(t.name); err != nil {
		return err
	}
	if _, err := buf.WriteString("\" (\n"); err != nil {
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
			if _, err := buf.WriteString(",\n"); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`  `); err != nil {
			return err
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
		return indexes[i].id < indexes[j].id
	})
	constraintCount := 0
	for _, index := range indexes {
		if !index.primary && !index.unique {
			continue
		}
		constraintCount++
		if constraintCount+len(columns) > 0 {
			if _, err := buf.WriteString(",\n"); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`  `); err != nil {
			return err
		}
		if err := index.toInlineString(buf); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n)\n;\n"); err != nil {
		return err
	}
	return nil
}

func newTableState(id int, name string) *tableState {
	return &tableState{
		id:      id,
		name:    name,
		columns: make(map[string]*columnState),
		indexes: make(map[string]*indexState),
	}
}

// nolint
func convertToTableState(id int, table *storepb.TableMetadata) *tableState {
	state := newTableState(id, table.Name)
	for i, column := range table.Columns {
		state.columns[column.Name] = convertToColumnState(i, column)
	}
	for i, index := range table.Indexes {
		state.indexes[index.Name] = convertToIndexState(i, index)
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

	return &storepb.TableMetadata{
		Name:    t.name,
		Columns: columns,
		Indexes: indexes,
		Comment: t.comment,
	}
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
	id           int
	name         string
	tp           string
	defaultValue defaultValue
	comment      string
	nullable     bool
}

func (c *columnState) toString(buf *strings.Builder) error {
	if _, err := buf.WriteString(`"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(c.name); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" `); err != nil {
		return err
	}
	if _, err := buf.WriteString(c.tp); err != nil {
		return err
	}
	if _, err := buf.WriteString(" VISIBLE"); err != nil {
		return err
	}
	if c.defaultValue != nil {
		if _, err := buf.WriteString(" DEFAULT "); err != nil {
			return err
		}
		if _, err := buf.WriteString(c.defaultValue.toString()); err != nil {
			return err
		}
	}
	if !c.nullable {
		if _, err := buf.WriteString(" NOT NULL"); err != nil {
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
	if c.defaultValue != nil {
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

// nolint
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

type indexState struct {
	id      int
	name    string
	keys    []string
	primary bool
	unique  bool
}

func (i *indexState) toOutlineString(schemaName, tableName string, buf *strings.Builder) error {
	if _, err := buf.WriteString(`CREATE`); err != nil {
		return err
	}

	if i.unique {
		if _, err := buf.WriteString(" UNIQUE"); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString(` INDEX "`); err != nil {
		return err
	}

	if schemaName != "" {
		if _, err := buf.WriteString(schemaName); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"."`); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString(i.name); err != nil {
		return err
	}

	if _, err := buf.WriteString(`" ON "`); err != nil {
		return err
	}

	if _, err := buf.WriteString(tableName); err != nil {
		return err
	}

	if _, err := buf.WriteString(`" (`); err != nil {
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

	if _, err := buf.WriteString(")\n;\n"); err != nil {
		return err
	}

	return nil
}

func (i *indexState) toInlineString(buf *strings.Builder) error {
	if _, err := buf.WriteString(`CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(i.name); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"`); err != nil {
		return err
	}

	if i.primary {
		if _, err := buf.WriteString(" PRIMARY KEY ("); err != nil {
			return err
		}
	} else if i.unique {
		if _, err := buf.WriteString(" UNIQUE ("); err != nil {
			return err
		}
	}

	for i, key := range i.keys {
		if i > 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`"`); err != nil {
			return err
		}
		if _, err := buf.WriteString(key); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"`); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString(")"); err != nil {
		return err
	}

	return nil
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

// nolint
func convertToIndexState(id int, index *storepb.IndexMetadata) *indexState {
	return &indexState{
		id:      id,
		name:    index.Name,
		keys:    index.Expressions,
		primary: index.Primary,
		unique:  index.Unique,
	}
}
