package tidb

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	tidbformat "github.com/pingcap/tidb/pkg/parser/format"
	tidbmysql "github.com/pingcap/tidb/pkg/parser/mysql"
	tidbtypes "github.com/pingcap/tidb/pkg/parser/types"

	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	autoIncrementSymbol = "AUTO_INCREMENT"
	autoRandSymbol      = "AUTO_RANDOM"
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

type tidbTransformer struct {
	tidbast.StmtNode

	state *databaseState
	err   error
}

func ParseToMetadata(schema string) (*storepb.DatabaseSchemaMetadata, error) {
	stmts, err := tidbparser.ParseTiDB(schema, "", "")
	if err != nil {
		return nil, err
	}

	transformer := &tidbTransformer{
		state: newDatabaseState(),
	}
	transformer.state.schemas[""] = newSchemaState()

	for _, stmt := range stmts {
		(stmt).Accept(transformer)
	}
	return transformer.state.convertToDatabaseMetadata(), transformer.err
}

func (t *tidbTransformer) Enter(in tidbast.Node) (tidbast.Node, bool) {
	if node, ok := in.(*tidbast.CreateTableStmt); ok {
		dbInfo := node.Table.DBInfo
		databaseName := ""
		if dbInfo != nil {
			databaseName = dbInfo.Name.String()
		}
		if databaseName != "" {
			if t.state.name == "" {
				t.state.name = databaseName
			} else if t.state.name != databaseName {
				t.err = errors.New("multiple database names found: " + t.state.name + ", " + databaseName)
				return in, true
			}
		}

		tableName := node.Table.Name.String()
		schema := t.state.schemas[""]
		if _, ok := schema.tables[tableName]; ok {
			t.err = errors.New("multiple table names found: " + tableName)
			return in, true
		}
		schema.tables[tableName] = newTableState(len(schema.tables), tableName)

		table := t.state.schemas[""].tables[tableName]

		// column definition
		for _, column := range node.Cols {
			dataType := columnTypeStr(column.Tp)
			columnName := column.Name.Name.String()
			if _, ok := table.columns[columnName]; ok {
				t.err = errors.New("multiple column names found: " + columnName + " in table " + tableName)
				return in, true
			}

			columnState := &columnState{
				id:       len(table.columns),
				name:     columnName,
				tp:       dataType,
				comment:  "",
				nullable: tidbColumnCanNull(column),
			}

			for _, option := range column.Options {
				switch option.Tp {
				case tidbast.ColumnOptionDefaultValue:
					defaultValue, err := columnDefaultValue(column)
					if err != nil {
						t.err = err
						return in, true
					}
					if defaultValue == nil {
						columnState.hasDefault = false
					} else {
						columnState.hasDefault = true
						switch {
						case strings.EqualFold(*defaultValue, "NULL"):
							columnState.defaultValue = &defaultValueNull{}
						case strings.HasPrefix(*defaultValue, "'") && strings.HasSuffix(*defaultValue, "'"):
							columnState.defaultValue = &defaultValueString{value: strings.ReplaceAll((*defaultValue)[1:len(*defaultValue)-1], "''", "'")}
						default:
							columnState.defaultValue = &defaultValueExpression{value: *defaultValue}
						}
					}
				case tidbast.ColumnOptionComment:
					comment, err := columnComment(column)
					if err != nil {
						t.err = err
						return in, true
					}
					columnState.comment = comment
				case tidbast.ColumnOptionAutoIncrement:
					defaultValue := autoIncrementSymbol
					columnState.hasDefault = true
					columnState.defaultValue = &defaultValueExpression{value: defaultValue}
				case tidbast.ColumnOptionAutoRandom:
					defaultValue := autoRandSymbol
					unspecifiedLength := -1
					if option.AutoRandOpt.ShardBits != unspecifiedLength {
						if option.AutoRandOpt.RangeBits != unspecifiedLength {
							defaultValue += fmt.Sprintf("(%d, %d)", option.AutoRandOpt.ShardBits, option.AutoRandOpt.RangeBits)
						} else {
							defaultValue += fmt.Sprintf("(%d)", option.AutoRandOpt.ShardBits)
						}
					}
					columnState.hasDefault = true
					columnState.defaultValue = &defaultValueExpression{value: defaultValue}
				}
			}
			table.columns[columnName] = columnState
		}
		for _, tableOption := range node.Options {
			if tableOption.Tp == tidbast.TableOptionComment {
				table.comment = tableComment(tableOption)
			}
		}

		// primary and foreign key definition
		for _, constraint := range node.Constraints {
			constraintType := constraint.Tp
			switch constraintType {
			case tidbast.ConstraintPrimaryKey:
				var pkList []string
				for _, constraint := range node.Constraints {
					if constraint.Tp == tidbast.ConstraintPrimaryKey {
						var pks []string
						for _, key := range constraint.Keys {
							columnName := key.Column.Name.String()
							pks = append(pks, columnName)
						}
						pkList = append(pkList, pks...)
					}
				}

				table.indexes["PRIMARY"] = &indexState{
					id:      len(table.indexes),
					name:    "PRIMARY",
					keys:    pkList,
					primary: true,
					unique:  true,
				}
			case tidbast.ConstraintForeignKey:
				var referencingColumnList []string
				for _, key := range constraint.Keys {
					referencingColumnList = append(referencingColumnList, key.Column.Name.String())
				}
				var referencedColumnList []string
				for _, spec := range constraint.Refer.IndexPartSpecifications {
					referencedColumnList = append(referencedColumnList, spec.Column.Name.String())
				}

				fkName := constraint.Name
				if fkName == "" {
					t.err = errors.New("empty foreign key name")
					return in, true
				}
				if table.foreignKeys[fkName] != nil {
					t.err = errors.New("multiple foreign keys found: " + fkName)
					return in, true
				}

				fk := &foreignKeyState{
					id:                len(table.foreignKeys),
					name:              fkName,
					columns:           referencingColumnList,
					referencedTable:   constraint.Refer.Table.Name.String(),
					referencedColumns: referencedColumnList,
				}
				table.foreignKeys[fkName] = fk
			case tidbast.ConstraintIndex, tidbast.ConstraintUniq, tidbast.ConstraintUniqKey, tidbast.ConstraintUniqIndex, tidbast.ConstraintKey:
				var referencingColumnList []string
				for _, spec := range constraint.Keys {
					var specString string
					var err error
					if spec.Column != nil {
						specString = spec.Column.Name.String()
						if spec.Length > 0 {
							specString = fmt.Sprintf("`%s`(%d)", specString, spec.Length)
						}
					} else {
						specString, err = tidbRestoreNode(spec, tidbformat.RestoreKeyWordLowercase|tidbformat.RestoreStringSingleQuotes|tidbformat.RestoreNameBackQuotes)
						if err != nil {
							t.err = err
							return in, true
						}
					}
					referencingColumnList = append(referencingColumnList, specString)
				}

				var indexName string
				if constraint.Name != "" {
					indexName = constraint.Name
				} else {
					t.err = errors.New("empty index name")
					return in, true
				}

				if table.indexes[indexName] != nil {
					t.err = errors.New("multiple foreign keys found: " + indexName)
					return in, true
				}

				table.indexes[indexName] = &indexState{
					id:      len(table.indexes),
					name:    indexName,
					keys:    referencingColumnList,
					primary: false,
					unique:  constraintType == tidbast.ConstraintUniq || constraintType == tidbast.ConstraintUniqKey || constraintType == tidbast.ConstraintUniqIndex,
				}
			}
		}
	}
	return in, false
}

// columnTypeStr returns the type string of tp.
func columnTypeStr(tp *tidbtypes.FieldType) string {
	switch tp.GetType() {
	// https://pkg.go.dev/github.com/pingcap/tidb/pkg/parser/mysql#TypeLong
	case tidbmysql.TypeLong:
		// tp.String() return int(11)
		return "int"
		// https://pkg.go.dev/github.com/pingcap/tidb/pkg/parser/mysql#TypeLonglong
	case tidbmysql.TypeLonglong:
		// tp.String() return bigint(20)
		return "bigint"
	default:
		text, err := tidbRestoreFieldType(tp)
		if err != nil {
			slog.Debug("tidbRestoreFieldType failed", "err", err, "type", tp.String())
			return tp.String()
		}
		return text
	}
}

func tidbColumnCanNull(column *tidbast.ColumnDef) bool {
	for _, option := range column.Options {
		if option.Tp == tidbast.ColumnOptionNotNull || option.Tp == tidbast.ColumnOptionPrimaryKey {
			return false
		}
	}
	return true
}

func columnDefaultValue(column *tidbast.ColumnDef) (*string, error) {
	for _, option := range column.Options {
		if option.Tp == tidbast.ColumnOptionDefaultValue {
			defaultValue, err := tidbRestoreNode(option.Expr, tidbformat.RestoreStringSingleQuotes|tidbformat.RestoreStringWithoutCharset)
			if err != nil {
				return nil, err
			}
			return &defaultValue, nil
		}
	}
	// no default value.
	return nil, nil
}

func tableComment(option *tidbast.TableOption) string {
	return option.StrValue
}

func columnComment(column *tidbast.ColumnDef) (string, error) {
	for _, option := range column.Options {
		if option.Tp == tidbast.ColumnOptionComment {
			comment, err := tidbRestoreNode(option.Expr, tidbformat.RestoreStringWithoutCharset)
			if err != nil {
				return "", err
			}
			return comment, nil
		}
	}

	return "", nil
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

func tidbRestoreNode(node tidbast.Node, flag tidbformat.RestoreFlags) (string, error) {
	var buffer strings.Builder
	ctx := tidbformat.NewRestoreCtx(flag, &buffer)
	if err := node.Restore(ctx); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func tidbRestoreNodeDefault(node tidbast.Node) (string, error) {
	return tidbRestoreNode(node, tidbformat.DefaultRestoreFlags)
}

func tidbRestoreFieldType(fieldType *tidbtypes.FieldType) (string, error) {
	var buffer strings.Builder
	// we want to use Default format flags but with lowercase keyword.
	flag := tidbformat.RestoreKeyWordLowercase | tidbformat.RestoreStringSingleQuotes | tidbformat.RestoreNameBackQuotes
	ctx := tidbformat.NewRestoreCtx(flag, &buffer)
	if err := fieldType.Restore(ctx); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func tidbRestoreTableOption(tableOption *tidbast.TableOption) (string, error) {
	var buffer strings.Builder
	flag := tidbformat.DefaultRestoreFlags
	ctx := tidbformat.NewRestoreCtx(flag, &buffer)
	if err := tableOption.Restore(ctx); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func (*tidbTransformer) Leave(in tidbast.Node) (tidbast.Node, bool) {
	return in, true
}

func GetDesignSchema(baselineSchema string, to *storepb.DatabaseSchemaMetadata) (string, error) {
	toState := convertToDatabaseState(to)
	stmts, err := tidbparser.ParseTiDB(baselineSchema, "", "")
	if err != nil {
		return "", err
	}

	generator := &tidbDesignSchemaGenerator{
		to: toState,
	}

	for _, stmt := range stmts {
		switch stmt.(type) {
		case *tidbast.CreateTableStmt:
			stmt.Accept(generator)
		default:
			if _, err := generator.result.WriteString(stmt.OriginalText() + "\n"); err != nil {
				return "", err
			}
		}
	}
	if generator.err != nil {
		return "", generator.err
	}

	firstTable := true
	for _, schema := range to.Schemas {
		schemaState, ok := toState.schemas[schema.Name]
		if !ok {
			continue
		}
		for _, table := range schema.Tables {
			table, ok := schemaState.tables[table.Name]
			if !ok {
				continue
			}
			if firstTable {
				firstTable = false
				if _, err := generator.result.WriteString("\n"); err != nil {
					return "", err
				}
			}
			if err := table.toString(&generator.result); err != nil {
				return "", err
			}
		}
	}

	return generator.result.String(), nil
}

type tidbDesignSchemaGenerator struct {
	tidbast.Node

	to                  *databaseState
	result              strings.Builder
	currentTable        *tableState
	firstElementInTable bool
	columnDefine        strings.Builder
	tableConstraints    strings.Builder
	err                 error
}

func (g *tidbDesignSchemaGenerator) Enter(in tidbast.Node) (tidbast.Node, bool) {
	if g.err != nil {
		return in, true
	}

	if node, ok := in.(*tidbast.CreateTableStmt); ok {
		dbInfo := node.Table.DBInfo
		databaseName := ""
		if dbInfo != nil {
			databaseName = dbInfo.Name.String()
		}
		if databaseName != "" && g.to.name != "" && databaseName != g.to.name {
			g.err = errors.New("multiple database names found: " + g.to.name + ", " + databaseName)
			return in, true
		}

		schema, ok := g.to.schemas[""]
		if !ok || schema == nil {
			return in, true
		}

		tableName := node.Table.Name.String()
		table, ok := schema.tables[tableName]
		if !ok {
			return in, true
		}
		g.currentTable = table
		g.firstElementInTable = true
		g.columnDefine.Reset()
		g.tableConstraints.Reset()

		delete(schema.tables, tableName)

		// Start constructing sql.
		// Temporary keyword.
		var temporaryKeyword string
		switch node.TemporaryKeyword {
		case tidbast.TemporaryNone:
			temporaryKeyword = "CREATE TABLE "
		case tidbast.TemporaryGlobal:
			temporaryKeyword = "CREATE GLOBAL TEMPORARY TABLE "
		case tidbast.TemporaryLocal:
			temporaryKeyword = "CREATE TEMPORARY TABLE "
		}
		if _, err := g.result.WriteString(temporaryKeyword); err != nil {
			g.err = err
			return in, true
		}

		// if not exists
		if node.IfNotExists {
			if _, err := g.result.WriteString("IF NOT EXISTS "); err != nil {
				g.err = err
				return in, true
			}
		}

		if tableNameStr, err := tidbRestoreNodeDefault(tidbast.Node(node.Table)); err == nil {
			if _, err := g.result.WriteString(tableNameStr + " "); err != nil {
				g.err = err
				return in, true
			}
		}

		if node.ReferTable != nil {
			if _, err := g.result.WriteString(" LIKE "); err != nil {
				g.err = err
				return in, true
			}
			if referTableStr, err := tidbRestoreNodeDefault(tidbast.Node(node.ReferTable)); err == nil {
				if _, err := g.result.WriteString(referTableStr + " "); err != nil {
					g.err = err
					return in, true
				}
			}
		}

		if _, err := g.result.WriteString("(\n  "); err != nil {
			g.err = err
			return in, true
		}

		// Column definition.
		for _, column := range node.Cols {
			columnName := column.Name.Name.String()
			stateColumn, ok := g.currentTable.columns[columnName]
			if !ok {
				continue
			}

			delete(g.currentTable.columns, columnName)

			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
					g.err = err
					return in, true
				}
			}

			// Column name.
			if columnNameStr, err := tidbRestoreNodeDefault(tidbast.Node(column.Name)); err == nil {
				if _, err := g.columnDefine.WriteString(columnNameStr + " "); err != nil {
					g.err = err
					return in, true
				}
			}

			// Compare column types.
			dataType := columnTypeStr(column.Tp)
			if !strings.EqualFold(dataType, stateColumn.tp) {
				if _, err := g.columnDefine.WriteString(stateColumn.tp); err != nil {
					g.err = err
					return in, true
				}
			} else {
				if typeStr, err := tidbRestoreFieldType(column.Tp); err == nil {
					if _, err := g.columnDefine.WriteString(typeStr); err != nil {
						g.err = err
						return in, true
					}
				}
			}

			// Column attributes.
			// todo(zp): refactor column auto_increment.
			skipSchemaAutoIncrement := false
			skipAutoRand := false
			// Default value, auto increment and auto random are mutually exclusive.
			for _, option := range column.Options {
				switch option.Tp {
				case tidbast.ColumnOptionDefaultValue:
					skipAutoRand = stateColumn.hasDefault
					skipSchemaAutoIncrement = stateColumn.hasDefault
				case tidbast.ColumnOptionAutoIncrement:
					skipSchemaAutoIncrement = stateColumn.hasDefault
				case tidbast.ColumnOptionAutoRandom:
					skipAutoRand = stateColumn.hasDefault
				}
			}
			newAttr := tidbExtractNewAttrs(stateColumn, column.Options)
			for _, option := range column.Options {
				attrOrder := tidbGetAttrOrder(option)
				for ; len(newAttr) > 0 && newAttr[0].order < attrOrder; newAttr = newAttr[1:] {
					if _, err := g.columnDefine.WriteString(" " + newAttr[0].text); err != nil {
						g.err = err
						return in, true
					}
				}

				switch option.Tp {
				case tidbast.ColumnOptionNull, tidbast.ColumnOptionNotNull:
					sameNullable := option.Tp == tidbast.ColumnOptionNull && stateColumn.nullable
					sameNullable = sameNullable || (option.Tp == tidbast.ColumnOptionNotNull && !stateColumn.nullable)

					if sameNullable {
						if optionStr, err := tidbRestoreNodeDefault(tidbast.Node(option)); err == nil {
							if _, err := g.columnDefine.WriteString(" " + optionStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else {
						if stateColumn.nullable {
							if _, err := g.columnDefine.WriteString(" NULL"); err != nil {
								g.err = err
								return in, true
							}
						} else {
							if _, err := g.columnDefine.WriteString(" NOT NULL"); err != nil {
								g.err = err
								return in, true
							}
						}
					}
				case tidbast.ColumnOptionDefaultValue:
					defaultValueText, err := columnDefaultValue(column)
					if err != nil {
						g.err = err
						return in, true
					}
					var defaultValue defaultValue
					switch {
					case strings.EqualFold(*defaultValueText, "NULL"):
						defaultValue = &defaultValueNull{}
					case strings.HasPrefix(*defaultValueText, "'") && strings.HasSuffix(*defaultValueText, "'"):
						defaultValue = &defaultValueString{value: strings.ReplaceAll((*defaultValueText)[1:len(*defaultValueText)-1], "''", "'")}
					default:
						defaultValue = &defaultValueExpression{value: *defaultValueText}
					}
					if stateColumn.hasDefault && stateColumn.defaultValue.toString() == defaultValue.toString() {
						if defaultStr, err := tidbRestoreNodeDefault(option); err == nil {
							if _, err := g.columnDefine.WriteString(" " + defaultStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else if stateColumn.hasDefault {
						if strings.EqualFold(stateColumn.defaultValue.toString(), autoIncrementSymbol) {
							if _, err := g.columnDefine.WriteString(" " + stateColumn.defaultValue.toString()); err != nil {
								g.err = err
								return in, true
							}
						} else if strings.Contains(strings.ToUpper(stateColumn.defaultValue.toString()), autoRandSymbol) {
							if _, err := g.columnDefine.WriteString(fmt.Sprintf(" /*T![auto_rand] %s */" + stateColumn.defaultValue.toString())); err != nil {
								g.err = err
								return in, true
							}
						} else {
							if _, err := g.columnDefine.WriteString(" DEFAULT"); err != nil {
								g.err = err
								return in, true
							}
							if _, err := g.columnDefine.WriteString(" " + stateColumn.defaultValue.toString()); err != nil {
								g.err = err
								return in, true
							}
						}
					}
				case tidbast.ColumnOptionComment:
					commentValue, err := columnComment(column)
					if err != nil {
						g.err = err
						return in, true
					}
					if stateColumn.comment == commentValue {
						if commentStr, err := tidbRestoreNodeDefault(option); err == nil {
							if _, err := g.columnDefine.WriteString(" " + commentStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else if stateColumn.comment != "" {
						if _, err := g.columnDefine.WriteString(" COMMENT"); err != nil {
							g.err = err
							return in, true
						}
						if _, err := g.columnDefine.WriteString(fmt.Sprintf(" '%s'", stateColumn.comment)); err != nil {
							g.err = err
							return in, true
						}
					}
				default:
					if skipSchemaAutoIncrement && option.Tp == tidbast.ColumnOptionAutoIncrement {
						continue
					}
					if skipAutoRand && option.Tp == tidbast.ColumnOptionAutoRandom {
						continue
					}
					if optionStr, err := tidbRestoreNodeDefault(option); err == nil {
						if _, err := g.columnDefine.WriteString(" " + optionStr); err != nil {
							g.err = err
							return in, true
						}
					}
				}
			}

			for _, attr := range newAttr {
				if _, err := g.columnDefine.WriteString(" " + attr.text); err != nil {
					g.err = err
					return in, true
				}
			}
		}

		// Table Constraint.
		for _, constraint := range node.Constraints {
			switch constraint.Tp {
			case tidbast.ConstraintPrimaryKey:
				if g.currentTable.indexes["PRIMARY"] != nil {
					if g.firstElementInTable {
						g.firstElementInTable = false
					} else {
						if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
							g.err = err
							return in, true
						}
					}
					var keys []string
					for _, key := range constraint.Keys {
						keys = append(keys, key.Column.Name.String())
					}
					if equalKeys(keys, g.currentTable.indexes["PRIMARY"].keys) {
						if constraintStr, err := tidbRestoreNodeDefault(constraint); err == nil {
							if _, err := g.tableConstraints.WriteString(constraintStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else {
						if err := g.currentTable.indexes["PRIMARY"].toString(&g.tableConstraints); err != nil {
							g.err = err
							return in, true
						}
					}
					delete(g.currentTable.indexes, "PRIMARY")
				}
			case tidbast.ConstraintForeignKey:
				fkName := constraint.Name
				if fkName == "" {
					g.err = errors.New("empty foreign key name")
					return in, true
				}
				if g.currentTable.foreignKeys[fkName] != nil {
					if g.firstElementInTable {
						g.firstElementInTable = false
					} else {
						if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
							g.err = err
							return in, true
						}
					}

					fk := g.currentTable.foreignKeys[fkName]

					var columns []string
					for _, key := range constraint.Keys {
						columns = append(columns, key.Column.Name.String())
					}

					var referencedColumnList []string
					for _, spec := range constraint.Refer.IndexPartSpecifications {
						referencedColumnList = append(referencedColumnList, spec.Column.Name.String())
					}
					referencedTable := constraint.Refer.Table.Name.String()
					if equalKeys(columns, fk.columns) && referencedTable == fk.referencedTable && equalKeys(referencedColumnList, fk.referencedColumns) {
						if constraintStr, err := tidbRestoreNodeDefault(constraint); err == nil {
							if _, err := g.tableConstraints.WriteString(constraintStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else {
						if err := fk.toString(&g.tableConstraints); err != nil {
							g.err = err
							return in, true
						}
					}
					delete(g.currentTable.foreignKeys, fkName)
				}
			default:
				if g.firstElementInTable {
					g.firstElementInTable = false
				} else {
					if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
						g.err = err
						return in, true
					}
				}
				if constraintStr, err := tidbRestoreNodeDefault(constraint); err == nil {
					if _, err := g.tableConstraints.WriteString(constraintStr); err != nil {
						g.err = err
						return in, true
					}
				}
			}
		}
	}

	return in, false
}

func (g *tidbDesignSchemaGenerator) Leave(in tidbast.Node) (tidbast.Node, bool) {
	if g.err != nil || g.currentTable == nil {
		return in, true
	}
	if node, ok := in.(*tidbast.CreateTableStmt); ok {
		// Column definition.
		var columnList []*columnState
		for _, column := range g.currentTable.columns {
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
					return in, true
				}
			}
			if err := column.toString(&g.columnDefine); err != nil {
				g.err = err
				return in, true
			}
		}

		// Primary key definition.
		if g.currentTable.indexes["PRIMARY"] != nil {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
					g.err = err
					return in, true
				}
			}
			if err := g.currentTable.indexes["PRIMARY"].toString(&g.tableConstraints); err != nil {
				return in, true
			}
		}

		// Foreign key definition.
		var fks []*foreignKeyState
		for _, fk := range g.currentTable.foreignKeys {
			fks = append(fks, fk)
		}
		sort.Slice(fks, func(i, j int) bool {
			return fks[i].id < fks[j].id
		})
		for _, fk := range fks {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.columnDefine.WriteString(",\n "); err != nil {
					g.err = err
					return in, true
				}
			}
			if err := fk.toString(&g.tableConstraints); err != nil {
				return in, true
			}
		}

		if _, err := g.result.WriteString(g.columnDefine.String()); err != nil {
			g.err = err
			return in, true
		}
		if _, err := g.result.WriteString(g.tableConstraints.String()); err != nil {
			g.err = err
			return in, true
		}
		if _, err := g.result.WriteString("\n)"); err != nil {
			g.err = err
			return in, true
		}

		// Table option.
		hasTableComment := false
		for _, option := range node.Options {
			if option.Tp == tidbast.TableOptionComment {
				commentValue := tableComment(option)
				if g.currentTable.comment == commentValue && g.currentTable.comment != "" {
					if _, err := g.result.WriteString(" COMMENT"); err != nil {
						g.err = err
						return in, true
					}
					if _, err := g.result.WriteString(fmt.Sprintf(" '%s'", g.currentTable.comment)); err != nil {
						g.err = err
						return in, true
					}
				}
				hasTableComment = true
			} else {
				if optionStr, err := tidbRestoreTableOption(option); err == nil {
					if _, err := g.result.WriteString(" " + optionStr); err != nil {
						g.err = err
						return in, true
					}
				}
			}
		}
		if !hasTableComment && g.currentTable.comment != "" {
			if _, err := g.result.WriteString(" COMMENT"); err != nil {
				g.err = err
				return in, true
			}
			if _, err := g.result.WriteString(fmt.Sprintf(" '%s'", g.currentTable.comment)); err != nil {
				g.err = err
				return in, true
			}
		}

		// Table partition.
		if node.Partition != nil {
			if partitionStr, err := tidbRestoreNodeDefault(node.Partition); err == nil {
				if _, err := g.result.WriteString(" " + partitionStr); err != nil {
					g.err = err
					return in, true
				}
			}
		}

		// Table select.
		if node.Select != nil {
			duplicateStr := ""
			switch node.OnDuplicate {
			case tidbast.OnDuplicateKeyHandlingError:
				duplicateStr = " AS "
			case tidbast.OnDuplicateKeyHandlingIgnore:
				duplicateStr = " IGNORE AS "
			case tidbast.OnDuplicateKeyHandlingReplace:
				duplicateStr = " REPLACE AS "
			}

			if selectStr, err := tidbRestoreNodeDefault(node.Select); err == nil {
				if _, err := g.result.WriteString(duplicateStr + selectStr); err != nil {
					g.err = err
					return in, true
				}
			}
		}

		if node.TemporaryKeyword == tidbast.TemporaryGlobal {
			if node.OnCommitDelete {
				if _, err := g.result.WriteString(" ON COMMIT DELETE ROWS"); err != nil {
					g.err = err
					return in, true
				}
			} else {
				if _, err := g.result.WriteString(" ON COMMIT PRESERVE ROWS"); err != nil {
					g.err = err
					return in, true
				}
			}
		}
		if _, err := g.result.WriteString(";\n"); err != nil {
			g.err = err
			return in, true
		}

		g.currentTable = nil
		g.firstElementInTable = false
	}
	return in, true
}

func tidbExtractNewAttrs(column *columnState, options []*tidbast.ColumnOption) []columnAttr {
	var result []columnAttr
	nullExists := false
	defaultExists := false
	commentExists := false

	for _, option := range options {
		switch option.Tp {
		case tidbast.ColumnOptionNull, tidbast.ColumnOptionNotNull:
			nullExists = true
		case tidbast.ColumnOptionDefaultValue:
			defaultExists = true
		case tidbast.ColumnOptionComment:
			commentExists = true
		}
	}

	if !nullExists && !column.nullable {
		result = append(result, columnAttr{
			text:  "NOT NULL",
			order: columnAttrOrder["NULL"],
		})
	}
	if !defaultExists && column.hasDefault {
		// todo(zp): refactor column attribute.
		if strings.EqualFold(column.defaultValue.toString(), autoIncrementSymbol) {
			result = append(result, columnAttr{
				text:  column.defaultValue.toString(),
				order: columnAttrOrder["DEFAULT"],
			})
		} else if strings.Contains(strings.ToUpper(column.defaultValue.toString()), autoRandSymbol) {
			result = append(result, columnAttr{
				text:  fmt.Sprintf("/*T![auto_rand] %s */", column.defaultValue.toString()),
				order: columnAttrOrder["DEFAULT"],
			})
		} else {
			result = append(result, columnAttr{
				text:  "DEFAULT " + column.defaultValue.toString(),
				order: columnAttrOrder["DEFAULT"],
			})
		}
	}
	if !commentExists && column.comment != "" {
		result = append(result, columnAttr{
			text:  "COMMENT '" + column.comment + "'",
			order: columnAttrOrder["COMMENT"],
		})
	}
	return result
}

func tidbGetAttrOrder(option *tidbast.ColumnOption) int {
	switch option.Tp {
	case tidbast.ColumnOptionDefaultValue:
		return columnAttrOrder["DEFAULT"]
	case tidbast.ColumnOptionNull, tidbast.ColumnOptionNotNull:
		return columnAttrOrder["NULL"]
	case tidbast.ColumnOptionUniqKey:
		return columnAttrOrder["UNIQUE"]
	case tidbast.ColumnOptionColumnFormat:
		return columnAttrOrder["COLUMN_FORMAT"]
	case tidbast.ColumnOptionAutoIncrement:
		return columnAttrOrder["AUTO_INCREMENT"]
	case tidbast.ColumnOptionAutoRandom:
		return columnAttrOrder["AUTO_RANDOM"]
	case tidbast.ColumnOptionComment:
		return columnAttrOrder["COMMENT"]
	case tidbast.ColumnOptionCollate:
		return columnAttrOrder["COLLATE"]
	case tidbast.ColumnOptionStorage:
		return columnAttrOrder["STORAGE"]
	case tidbast.ColumnOptionCheck:
		return columnAttrOrder["CHECK"]
	}
	if option.Enforced {
		return columnAttrOrder["ENFORCED"]
	}
	return len(columnAttrOrder) + 1
}
