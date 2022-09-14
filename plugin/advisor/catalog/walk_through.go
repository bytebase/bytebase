package catalog

import (
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/bytebase/bytebase/plugin/parser"

	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
)

// WalkThroughErrorType is the type of WalkThroughError.
type WalkThroughErrorType int

const (
	// PrimaryKeyName is the string for PK.
	PrimaryKeyName string = "PRIMARY"
	// FullTextName is the string for FULLTEXT.
	FullTextName string = "FULLTEXT"

	// ErrorTypeUnsupported is the error for unsupported cases.
	ErrorTypeUnsupported WalkThroughErrorType = 1

	// 101 parse error type.

	// ErrorTypeParseError is the error in parsing.
	ErrorTypeParseError WalkThroughErrorType = 101
	// ErrorTypeRestoreError is the error in restoring.
	ErrorTypeRestoreError WalkThroughErrorType = 102

	// 201 ~ 299 database error type.

	// ErrorTypeAccessOtherDatabase is the error that try to access other database.
	ErrorTypeAccessOtherDatabase = 201

	// 301 ~ 399 table error type.

	// ErrorTypeTableExists is the error that table exists.
	ErrorTypeTableExists = 301
	// ErrorTypeTableNotExists is the error that table not exists.
	ErrorTypeTableNotExists = 302

	// 401 ~ 499 column error type.

	// ErrorTypeColumnExists is the error that column exists.
	ErrorTypeColumnExists = 401
	// ErrorTypeColumnNotExists is the error that column not exists.
	ErrorTypeColumnNotExists = 402

	// 501 ~ 599 index error type.

	// ErrorTypePrimaryKeyExists is the error that PK exists.
	ErrorTypePrimaryKeyExists = 501
	// ErrorTypeIndexExists is the error that index exists.
	ErrorTypeIndexExists = 502
	// ErrorTypeIndexEmptyKeys is the error that index has empty keys.
	ErrorTypeIndexEmptyKeys = 503
)

// WalkThroughError is the error for walking-through.
type WalkThroughError struct {
	Type    WalkThroughErrorType
	Content string
}

// NewParseError returns a new ErrorTypeParseError.
func NewParseError(content string) *WalkThroughError {
	return &WalkThroughError{
		Type:    ErrorTypeParseError,
		Content: content,
	}
}

// Error implements the error interface.
func (e *WalkThroughError) Error() string {
	return e.Content
}

// WalkThrough will collect the catalog schema in the databaseState as it walks through the stmts.
func (d *databaseState) WalkThrough(stmts string) error {
	if d.dbType != db.MySQL {
		return &WalkThroughError{
			Type:    ErrorTypeUnsupported,
			Content: fmt.Sprintf("Engine type %s is not supported", d.dbType),
		}
	}

	// We define the Catalog as Database -> Schema -> Table. The Schema is only for PostgreSQL.
	// So we use a Schema whose name is empty for other engines, such as MySQL.
	// If there is no empty-string-name schema, create it to avoid corner cases.
	if _, exists := d.schemaSet[""]; !exists {
		d.createSchema("")
	}

	nodeList, err := d.parse(stmts)
	if err != nil {
		return err
	}

	for _, node := range nodeList {
		if err := d.changeState(node); err != nil {
			return err
		}
	}

	return nil
}

func (d *databaseState) changeState(in tidbast.StmtNode) error {
	switch node := in.(type) {
	case *tidbast.CreateTableStmt:
		return d.createTable(node)
	case *tidbast.DropTableStmt:
		return d.dropTable(node)
	default:
		return nil
	}
}

func (d *databaseState) dropTable(node *tidbast.DropTableStmt) error {
	// TODO(rebelice): deal with DROP VIEW statement.
	if !node.IsView {
		for _, name := range node.Tables {
			if name.Schema.O != "" && d.name != name.Schema.O {
				return &WalkThroughError{
					Type:    ErrorTypeAccessOtherDatabase,
					Content: fmt.Sprintf("Database `%s` is not the current database `%s`", name.Schema.O, d.name),
				}
			}

			schema, exists := d.schemaSet[""]
			if !exists {
				schema = d.createSchema("")
			}

			if _, exists = schema.tableSet[name.Name.O]; !exists {
				if node.IfExists || !schema.context.CheckIntegrity {
					return nil
				}
				return &WalkThroughError{
					Type:    ErrorTypeTableNotExists,
					Content: fmt.Sprintf("Table `%s` does not exist", name.Name.O),
				}
			}

			delete(schema.tableSet, name.Name.O)
		}
	}
	return nil
}

func (d *databaseState) createTable(node *tidbast.CreateTableStmt) error {
	if node.Table.Schema.O != "" && d.name != node.Table.Schema.O {
		return &WalkThroughError{
			Type:    ErrorTypeAccessOtherDatabase,
			Content: fmt.Sprintf("Database `%s` is not the current database `%s`", node.Table.Schema.O, d.name),
		}
	}

	schema, exists := d.schemaSet[""]
	if !exists {
		schema = d.createSchema("")
	}

	if _, exists = schema.tableSet[node.Table.Name.O]; exists {
		if node.IfNotExists {
			return nil
		}
		return &WalkThroughError{
			Type:    ErrorTypeTableExists,
			Content: fmt.Sprintf("Table `%s` already exists", node.Table.Name.O),
		}
	}

	table := &tableState{
		name:      node.Table.Name.O,
		columnSet: make(columnStateMap),
		indexSet:  make(indexStateMap),
		context:   &FinderContext{CheckIntegrity: true},
	}
	schema.tableSet[table.name] = table

	for i, column := range node.Cols {
		if err := table.createColumn(column, i+1); err != nil {
			return err
		}
	}

	for _, constraint := range node.Constraints {
		if err := table.createConstraint(constraint); err != nil {
			return err
		}
	}

	return nil
}

func (t *tableState) createConstraint(constraint *tidbast.Constraint) error {
	switch constraint.Tp {
	case tidbast.ConstraintPrimaryKey:
		keyList, err := t.validateAndGetKeyStringList(constraint.Keys, true /* primary */)
		if err != nil {
			return err
		}
		if err := t.createPrimaryKey(keyList, getIndexType(constraint.Option)); err != nil {
			return err
		}
	case tidbast.ConstraintKey, tidbast.ConstraintIndex:
		keyList, err := t.validateAndGetKeyStringList(constraint.Keys, false /* primary */)
		if err != nil {
			return err
		}
		if err := t.createIndex(constraint.Name, keyList, false /* unique */, getIndexType(constraint.Option)); err != nil {
			return err
		}
	case tidbast.ConstraintUniq, tidbast.ConstraintUniqKey, tidbast.ConstraintUniqIndex:
		keyList, err := t.validateAndGetKeyStringList(constraint.Keys, false /* primary */)
		if err != nil {
			return err
		}
		if err := t.createIndex(constraint.Name, keyList, true /* unique */, getIndexType(constraint.Option)); err != nil {
			return err
		}
	case tidbast.ConstraintForeignKey:
		// we do not deal with FOREIGN KEY constraints
	case tidbast.ConstraintFulltext:
		keyList, err := t.validateAndGetKeyStringList(constraint.Keys, false /* primary */)
		if err != nil {
			return err
		}
		if err := t.createIndex(constraint.Name, keyList, false /* unique */, FullTextName); err != nil {
			return err
		}
	case tidbast.ConstraintCheck:
		// we do not deal with CHECK constraints
	}

	return nil
}

func (t *tableState) validateAndGetKeyStringList(keyList []*tidbast.IndexPartSpecification, primary bool) ([]string, error) {
	var res []string
	for _, key := range keyList {
		if key.Expr != nil {
			str, err := restoreNode(key, format.DefaultRestoreFlags)
			if err != nil {
				return nil, err
			}
			res = append(res, str)
		} else {
			columnName := key.Column.Name.O
			column, exists := t.columnSet[columnName]
			if !exists {
				return nil, &WalkThroughError{
					Type:    ErrorTypeColumnNotExists,
					Content: fmt.Sprintf("Column `%s` in table `%s` not exists", columnName, t.name),
				}
			}
			if primary {
				column.nullable = false
			}
			res = append(res, columnName)
		}
	}
	return res, nil
}

func (t *tableState) createColumn(column *tidbast.ColumnDef, position int) error {
	if _, exists := t.columnSet[column.Name.Name.O]; exists {
		return &WalkThroughError{
			Type:    ErrorTypeColumnExists,
			Content: fmt.Sprintf("Column `%s` exists in table `%s`", column.Name.Name.O, t.name),
		}
	}

	col := &columnState{
		name:         column.Name.Name.O,
		position:     position,
		columnType:   column.Tp.CompactStr(),
		characterSet: column.Tp.GetCharset(),
		collation:    column.Tp.GetCollate(),
		nullable:     true,
	}

	for _, option := range column.Options {
		switch option.Tp {
		case tidbast.ColumnOptionPrimaryKey:
			col.nullable = false
			if err := t.createPrimaryKey([]string{col.name}, model.IndexTypeBtree.String()); err != nil {
				return err
			}
		case tidbast.ColumnOptionNotNull:
			col.nullable = false
		case tidbast.ColumnOptionAutoIncrement:
			// we do not deal with AUTO-INCREMENT
		case tidbast.ColumnOptionDefaultValue:
			defaultValue, err := restoreNode(option.Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				return err
			}
			col.defaultValue = &defaultValue
		case tidbast.ColumnOptionUniqKey:
			if err := t.createIndex("", []string{col.name}, true /* unique */, model.IndexTypeBtree.String()); err != nil {
				return err
			}
		case tidbast.ColumnOptionNull:
			col.nullable = true
		case tidbast.ColumnOptionOnUpdate:
			// we do not deal with ON UPDATE
		case tidbast.ColumnOptionComment:
			comment, err := restoreNode(option.Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				return err
			}
			col.comment = comment
		case tidbast.ColumnOptionGenerated:
			// we do not deal with GENERATED ALWAYS AS
		case tidbast.ColumnOptionReference:
			// MySQL will ignore the inline REFERENCE
			// https://dev.mysql.com/doc/refman/8.0/en/create-table.html
		case tidbast.ColumnOptionCollate:
			col.collation = option.StrValue
		case tidbast.ColumnOptionCheck:
			// we do not deal with CHECK constraint
		case tidbast.ColumnOptionColumnFormat:
			// we do not deal with COLUMN_FORMAT
		case tidbast.ColumnOptionStorage:
			// we do not deal with STORAGE
		case tidbast.ColumnOptionAutoRandom:
			// we do not deal with AUTO_RANDOM
		}
	}

	t.columnSet[col.name] = col
	return nil
}

func (t *tableState) createIndex(name string, keyList []string, unique bool, tp string) error {
	if len(keyList) == 0 {
		return &WalkThroughError{
			Type:    ErrorTypeIndexEmptyKeys,
			Content: fmt.Sprintf("Index `%s` in table `%s` has empty key", name, t.name),
		}
	}
	if name != "" {
		if _, exists := t.indexSet[name]; exists {
			return &WalkThroughError{
				Type:    ErrorTypeIndexExists,
				Content: fmt.Sprintf("Index `%s` exists in table `%s`", name, t.name),
			}
		}
	} else {
		suffix := 1
		for {
			name = keyList[0]
			if suffix > 1 {
				name = fmt.Sprintf("%s_%d", keyList[0], suffix)
			}
			if _, exists := t.indexSet[name]; !exists {
				break
			}
			suffix++
		}
	}

	index := &indexState{
		name:           name,
		expressionList: keyList,
		unique:         unique,
		primary:        false,
		indextype:      tp,
	}
	t.indexSet[name] = index
	return nil
}

func (t *tableState) createPrimaryKey(keys []string, tp string) error {
	if _, exists := t.indexSet[PrimaryKeyName]; exists {
		return &WalkThroughError{
			Type:    ErrorTypePrimaryKeyExists,
			Content: fmt.Sprintf("Primary key exists in table `%s`", t.name),
		}
	}

	pk := &indexState{
		name:           PrimaryKeyName,
		expressionList: keys,
		unique:         true,
		primary:        true,
		indextype:      tp,
	}
	t.indexSet[pk.name] = pk
	return nil
}

func (d *databaseState) createSchema(name string) *schemaState {
	schema := &schemaState{
		name:         name,
		tableSet:     make(tableStateMap),
		viewSet:      make(viewStateMap),
		extensionSet: make(extensionStateMap),
		context:      d.context.Copy(),
	}

	d.schemaSet[name] = schema
	return schema
}

func (d *databaseState) parse(stmts string) ([]tidbast.StmtNode, error) {
	p := tidbparser.New()
	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	nodeList, _, err := p.Parse(stmts, d.characterSet, d.collation)
	if err != nil {
		return nil, NewParseError(err.Error())
	}
	sqlList, err := parser.SplitMultiSQL(parser.MySQL, stmts)
	if err != nil {
		return nil, NewParseError(err.Error())
	}
	if len(sqlList) != len(nodeList) {
		return nil, NewParseError(fmt.Sprintf("split multi-SQL failed: the length should be %d, but get %d. stmt: \"%s\"", len(nodeList), len(sqlList), stmts))
	}

	for i, node := range nodeList {
		node.SetOriginTextPosition(sqlList[i].Line)
		if n, ok := node.(*tidbast.CreateTableStmt); ok {
			if err := parser.SetLineForMySQLCreateTableStmt(n); err != nil {
				return nil, NewParseError(err.Error())
			}
		}
	}

	return nodeList, nil
}

func restoreNode(node tidbast.Node, flag format.RestoreFlags) (string, error) {
	var buffer strings.Builder
	ctx := format.NewRestoreCtx(flag, &buffer)
	if err := node.Restore(ctx); err != nil {
		return "", &WalkThroughError{
			Type:    ErrorTypeRestoreError,
			Content: err.Error(),
		}
	}
	return buffer.String(), nil
}

func getIndexType(option *tidbast.IndexOption) string {
	if option != nil {
		switch option.Tp {
		case model.IndexTypeBtree,
			model.IndexTypeHash,
			model.IndexTypeRtree:
			return option.Tp.String()
		}
	}
	return model.IndexTypeBtree.String()
}
