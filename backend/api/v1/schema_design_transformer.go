package v1

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	mysqlse "github.com/bytebase/bytebase/backend/plugin/schema-engine/mysql"
	pgse "github.com/bytebase/bytebase/backend/plugin/schema-engine/pg"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	autoIncrementSymbol = "AUTO_INCREMENT"
)

func transformDatabaseMetadataToSchemaString(engine storepb.Engine, database *storepb.DatabaseSchemaMetadata) (string, error) {
	switch engine {
	case storepb.Engine_MYSQL:
		return mysqlse.GetDesignSchema("", database)
	case storepb.Engine_TIDB:
		return getTiDBDesignSchema("", database)
	case storepb.Engine_POSTGRES:
		return pgse.GetDesignSchema("", database)
	default:
		return "", status.Errorf(codes.InvalidArgument, fmt.Sprintf("unsupported engine: %v", engine))
	}
}

func TransformSchemaStringToDatabaseMetadata(engine storepb.Engine, schema string) (*storepb.DatabaseSchemaMetadata, error) {
	dbSchema, err := func() (*storepb.DatabaseSchemaMetadata, error) {
		switch engine {
		case storepb.Engine_MYSQL:
			return mysqlse.ParseToMetadata(schema)
		case storepb.Engine_POSTGRES:
			return pgse.ParseToMetadata(schema)
		case storepb.Engine_TIDB:
			return parseTiDBSchemaStringToDatabaseMetadata(schema)
		default:
			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("unsupported engine: %v", engine))
		}
	}()
	if err != nil {
		return nil, err
	}
	setClassificationAndUserCommentFromComment(dbSchema)
	return dbSchema, nil
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
	for _, schema := range database.Schemas {
		state.schemas[schema.Name] = convertToSchemaState(schema)
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

func newSchemaState() *schemaState {
	return &schemaState{
		tables: make(map[string]*tableState),
	}
}

func convertToSchemaState(schema *storepb.SchemaMetadata) *schemaState {
	state := newSchemaState()
	state.name = schema.Name
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
	id          int
	name        string
	columns     map[string]*columnState
	indexes     map[string]*indexState
	foreignKeys map[string]*foreignKeyState
	comment     string
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

	if _, err := buf.WriteString("\n)"); err != nil {
		return err
	}

	if t.comment != "" {
		if _, err := buf.WriteString(fmt.Sprintf(" COMMENT '%s'", strings.ReplaceAll(t.comment, "'", "''"))); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString(";\n"); err != nil {
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
	referencedTable   string
	referencedColumns []string
}

func (f *foreignKeyState) convertToForeignKeyMetadata() *storepb.ForeignKeyMetadata {
	return &storepb.ForeignKeyMetadata{
		Name:              f.name,
		Columns:           f.columns,
		ReferencedTable:   f.referencedTable,
		ReferencedColumns: f.referencedColumns,
	}
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
	} else {
		if i.unique {
			if _, err := buf.WriteString("UNIQUE INDEX "); err != nil {
				return err
			}
		} else {
			if _, err := buf.WriteString("INDEX "); err != nil {
				return err
			}
		}

		if _, err := buf.WriteString(fmt.Sprintf("`%s` (", i.name)); err != nil {
			return err
		}
		for j, key := range i.keys {
			if j > 0 {
				if _, err := buf.WriteString(","); err != nil {
					return err
				}
			}
			if len(key) > 2 && key[0] == '(' && key[len(key)-1] == ')' {
				// Expressions are surrounded by parentheses.
				if _, err := buf.WriteString(key); err != nil {
					return err
				}
			} else {
				if _, err := buf.WriteString(fmt.Sprintf("`%s`", key)); err != nil {
					return err
				}
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
	return fmt.Sprintf("'%s'", strings.ReplaceAll(d.value, "'", "''"))
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
	hasDefault   bool
	defaultValue defaultValue
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
	if c.hasDefault {
		// todo(zp): refactor column attribute.
		if strings.EqualFold(c.defaultValue.toString(), autoIncrementSymbol) {
			if _, err := buf.WriteString(fmt.Sprintf(" %s", c.defaultValue.toString())); err != nil {
				return err
			}
		} else if strings.Contains(strings.ToUpper(c.defaultValue.toString()), autoRandSymbol) {
			if _, err := buf.WriteString(fmt.Sprintf(" /*T![auto_rand] %s */", c.defaultValue.toString())); err != nil {
				return err
			}
		} else {
			if _, err := buf.WriteString(fmt.Sprintf(" DEFAULT %s", c.defaultValue.toString())); err != nil {
				return err
			}
		}
	}
	if c.comment != "" {
		if _, err := buf.WriteString(fmt.Sprintf(" COMMENT '%s'", c.comment)); err != nil {
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

func getDesignSchema(engine storepb.Engine, baselineSchema string, to *storepb.DatabaseSchemaMetadata) (string, error) {
	switch engine {
	case storepb.Engine_MYSQL:
		result, err := mysqlse.GetDesignSchema(baselineSchema, to)
		if err != nil {
			return "", status.Errorf(codes.Internal, "failed to generate mysql design schema: %v", err)
		}
		return result, nil
	case storepb.Engine_TIDB:
		result, err := getTiDBDesignSchema(baselineSchema, to)
		if err != nil {
			return "", status.Errorf(codes.Internal, "failed to generate tidb design schema: %v", err)
		}
		return result, nil
	case storepb.Engine_POSTGRES:
		result, err := pgse.GetDesignSchema(baselineSchema, to)
		if err != nil {
			return "", status.Errorf(codes.Internal, "failed to generate postgres design schema: %v", err)
		}
		return result, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, fmt.Sprintf("unsupported engine: %v", engine))
	}
}

type columnAttr struct {
	text  string
	order int
}

var columnAttrOrder = map[string]int{
	"NULL":           1,
	"DEFAULT":        2,
	"VISIBLE":        3,
	"AUTO_INCREMENT": 4,
	"AUTO_RAND":      4,
	"UNIQUE":         5,
	"KEY":            6,
	"COMMENT":        7,
	"COLLATE":        8,
	"COLUMN_FORMAT":  9,
	"SECONDARY":      10,
	"STORAGE":        11,
	"SERIAL":         12,
	"SRID":           13,
	"ON":             14,
	"CHECK":          15,
	"ENFORCED":       16,
}

func equalKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, key := range a {
		if key != b[i] {
			return false
		}
	}
	return true
}

func checkDatabaseMetadata(engine storepb.Engine, metadata *storepb.DatabaseSchemaMetadata) error {
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_POSTGRES:
	default:
		return errors.Errorf("unsupported engine for check database metadata: %v", engine)
	}

	schemaMap := make(map[string]bool)
	for _, schema := range metadata.GetSchemas() {
		if (engine == storepb.Engine_MYSQL || engine == storepb.Engine_TIDB) && schema.GetName() != "" {
			return errors.Errorf("schema name should be empty for MySQL and TiDB")
		}
		if _, ok := schemaMap[schema.GetName()]; ok {
			return errors.Errorf("duplicate schema name %s", schema.GetName())
		}
		schemaMap[schema.GetName()] = true

		tableNameMap := make(map[string]bool)
		for _, table := range schema.GetTables() {
			if table.GetName() == "" {
				return errors.Errorf("table name should not be empty")
			}
			if _, ok := tableNameMap[table.GetName()]; ok {
				return errors.Errorf("duplicate table name %s", table.GetName())
			}
			tableNameMap[table.GetName()] = true

			columnNameMap := make(map[string]bool)
			for _, column := range table.GetColumns() {
				if column.GetName() == "" {
					return errors.Errorf("column name should not be empty in table %s", table.GetName())
				}
				if _, ok := columnNameMap[column.GetName()]; ok {
					return errors.Errorf("duplicate column name %s in table %s", column.GetName(), table.GetName())
				}
				columnNameMap[column.GetName()] = true

				if column.GetType() == "" {
					return errors.Errorf("column %s type should not be empty in table %s", column.GetName(), table.GetName())
				}
			}

			indexNameMap := make(map[string]bool)
			for _, index := range table.GetIndexes() {
				if index.GetName() == "" {
					return errors.Errorf("index name should not be empty in table %s", table.GetName())
				}
				if _, ok := indexNameMap[index.GetName()]; ok {
					return errors.Errorf("duplicate index name %s in table %s", index.GetName(), table.GetName())
				}
				indexNameMap[index.GetName()] = true
				if index.Primary {
					for _, key := range index.GetExpressions() {
						if _, ok := columnNameMap[key]; !ok {
							return errors.Errorf("primary key column %s not found in table %s", key, table.GetName())
						}
					}
				}
			}
		}
	}
	return nil
}

func checkDatabaseMetadataColumnType(engine storepb.Engine, metadata *storepb.DatabaseSchemaMetadata) error {
	for _, schema := range metadata.GetSchemas() {
		for _, table := range schema.GetTables() {
			for _, column := range table.GetColumns() {
				if !checkColumnType(engine, column.Type) {
					return errors.Errorf("column %s type %s is invalid in table %s", column.Name, column.Type, table.Name)
				}
			}
		}
	}
	return nil
}

func checkColumnType(engine storepb.Engine, tp string) bool {
	switch engine {
	case storepb.Engine_MYSQL:
		return checkMySQLColumnType(tp)
	case storepb.Engine_TIDB:
		return checkTiDBColumnType(tp)
	case storepb.Engine_POSTGRES:
		return checkPostgreSQLColumnType(tp)
	default:
		return false
	}
}

func checkMySQLColumnType(tp string) bool {
	_, err := mysqlparser.ParseMySQL(fmt.Sprintf("CREATE TABLE t (a %s NOT NULL)", tp))
	return err == nil
}

func checkPostgreSQLColumnType(tp string) bool {
	_, err := pgrawparser.Parse(pgrawparser.ParseContext{}, fmt.Sprintf("CREATE TABLE t (a %s NOT NULL)", tp))
	return err == nil
}
