// Package mysql provides the MySQL transformer plugin.
package mysql

import (
	"bytes"
	"sort"
	"strings"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pkg/errors"

	bbparser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/transform"
	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
)

var (
	_ transform.SchemaTransformer = (*SchemaTransformer)(nil)
)

func init() {
	transform.Register(bbparser.MySQL, &SchemaTransformer{})
	transform.Register(bbparser.TiDB, &SchemaTransformer{})
	transform.Register(bbparser.OceanBase, &SchemaTransformer{})
}

// SchemaTransformer it the transformer for MySQL dialect.
type SchemaTransformer struct {
}

// Accepted MySQL SDL Format:
// 1. CREATE TABLE statements.
//    i.  Column define without constraints.
//    ii. Primary key, check and foreign key constraints define in table-level.
// 2. CREATE INDEX statements.

type indexInfo struct {
	originPos   int
	missing     bool
	createIndex *ast.CreateIndexStmt
}

type indexSet map[string]*indexInfo

func newIndexInfo(pos int, createIndex *ast.CreateIndexStmt) *indexInfo {
	return &indexInfo{
		originPos:   pos,
		missing:     true,
		createIndex: createIndex,
	}
}

type tableInfo struct {
	originPos   int
	missing     bool
	createTable *ast.CreateTableStmt
	indexSet    indexSet
}
type tableSet map[string]*tableInfo

func newTableInfo(pos int, createTable *ast.CreateTableStmt) *tableInfo {
	return &tableInfo{
		originPos:   pos,
		missing:     true,
		createTable: createTable,
		indexSet:    make(indexSet),
	}
}

// Normalize normalizes the schema format. The schema and standard should be SDL format.
func (t *SchemaTransformer) Normalize(schema string, standard string) (string, error) {
	var orderedList []ast.Node

	if _, err := t.Check(schema); err != nil {
		return "", errors.Wrapf(err, "Schema is not the SDL format")
	}
	if _, err := t.Check(standard); err != nil {
		return "", errors.Wrapf(err, "Standard is not the SDL format")
	}
	var remainingStatement []string

	// Phase One: build the schema table set.
	tableSet := make(tableSet)
	list, err := bbparser.SplitMultiSQL(bbparser.MySQL, schema)
	if err != nil {
		return "", errors.Wrapf(err, "failed to split SQL")
	}

	changeDelimiter := false
	for i, stmt := range list {
		if bbparser.IsDelimiter(stmt.Text) {
			delimiter, err := bbparser.ExtractDelimiter(stmt.Text)
			if err != nil {
				return "", errors.Wrapf(err, "failed to extract delimiter from %q", stmt.Text)
			}
			if delimiter == ";" {
				changeDelimiter = false
			} else {
				changeDelimiter = true
			}
			remainingStatement = append(remainingStatement, stmt.Text)
			continue
		}
		if changeDelimiter {
			// TiDB parser cannot deal with delimiter change.
			// So we need to skip the statement if the delimiter is not `;`.
			remainingStatement = append(remainingStatement, stmt.Text)
			continue
		}
		if bbparser.IsTiDBUnsupportDDLStmt(stmt.Text) {
			remainingStatement = append(remainingStatement, stmt.Text)
			continue
		}
		nodeList, _, err := parser.New().Parse(stmt.Text, "", "")
		if err != nil {
			return "", errors.Wrapf(err, "failed to parse schema %q", schema)
		}
		if len(nodeList) != 1 {
			return "", errors.Errorf("Expect one statement after splitting but found %d", len(nodeList))
		}

		switch node := nodeList[0].(type) {
		case *ast.CreateTableStmt:
			tableSet[node.Table.Name.String()] = newTableInfo(i, node)
		case *ast.CreateIndexStmt:
			table, exists := tableSet[node.Table.Name.String()]
			if !exists {
				return "", errors.Errorf("Table `%s` not found", node.Table.Name.String())
			}
			table.indexSet[node.IndexName] = newIndexInfo(i, node)
		default:
			remainingStatement = append(remainingStatement, stmt.Text)
		}
	}

	// Phase Two: find the missing table and index for schema and remove the collation and charset if needed.
	standardList, err := bbparser.SplitMultiSQL(bbparser.MySQL, standard)
	if err != nil {
		return "", errors.Wrapf(err, "failed to split SQL")
	}

	for _, stmt := range standardList {
		if bbparser.IsTiDBUnsupportDDLStmt(stmt.Text) {
			// TODO(rebelice): consider the unsupported DDL.
			continue
		}
		nodeList, _, err := parser.New().Parse(stmt.Text, "", "")
		if err != nil {
			return "", errors.Wrapf(err, "failed to parse schema %q", schema)
		}
		if len(nodeList) != 1 {
			return "", errors.Errorf("Expect one statement after splitting but found %d", len(nodeList))
		}

		switch node := nodeList[0].(type) {
		case *ast.CreateTableStmt:
			if table, exists := tableSet[node.Table.Name.String()]; exists {
				table.missing = false
				removeRedundantTableOption(table.createTable, node)
				removeRedundantColumnOption(table.createTable, node)
			}
		case *ast.CreateIndexStmt:
			if table, exists := tableSet[node.Table.Name.String()]; exists {
				if index, exists := table.indexSet[node.IndexName]; exists {
					index.missing = false
				}
			}
		default:
			remainingStatement = append(remainingStatement, stmt.Text)
		}
	}

	// Phase Three: generate ordered statements.
	// The order rule is:
	//   1. existed tables are on top of missing tables.
	//   2. existed tables and indexes are ordered as the table order in the standard schema.
	//   3. missing indexes for existed table are below of this table and as the origin order.
	//   4. missing tables are below of existed tables and as the origin order.
	for _, stmt := range standardList {
		if bbparser.IsTiDBUnsupportDDLStmt(stmt.Text) {
			// TODO(rebelice): consider the unsupported DDL.
			continue
		}
		nodeList, _, err := parser.New().Parse(stmt.Text, "", "")
		if err != nil {
			return "", errors.Wrapf(err, "failed to parse schema %q", schema)
		}
		if len(nodeList) != 1 {
			return "", errors.Errorf("Expect one statement after splitting but found %d", len(nodeList))
		}

		switch node := nodeList[0].(type) {
		case *ast.CreateTableStmt:
			if table, exists := tableSet[node.Table.Name.String()]; exists {
				orderedList = append(orderedList, table.createTable)
				var missingIndexList []*indexInfo
				for _, index := range table.indexSet {
					if index.missing {
						missingIndexList = append(missingIndexList, index)
					}
				}
				sort.Slice(missingIndexList, func(i, j int) bool {
					return missingIndexList[i].originPos < missingIndexList[j].originPos
				})
				for _, index := range missingIndexList {
					orderedList = append(orderedList, index.createIndex)
				}
			}
		case *ast.CreateIndexStmt:
			if table, exists := tableSet[node.Table.Name.String()]; exists {
				if index, exists := table.indexSet[node.IndexName]; exists {
					orderedList = append(orderedList, index.createIndex)
				}
			}
		default:
			remainingStatement = append(remainingStatement, stmt.Text)
		}
	}

	var missingTableList []*tableInfo
	for _, table := range tableSet {
		if table.missing {
			missingTableList = append(missingTableList, table)
		}
	}
	sort.Slice(missingTableList, func(i, j int) bool {
		return missingTableList[i].originPos < missingTableList[j].originPos
	})
	for _, table := range missingTableList {
		orderedList = append(orderedList, table.createTable)
		var missingIndexList []*indexInfo
		for _, index := range table.indexSet {
			if index.missing {
				missingIndexList = append(missingIndexList, index)
			}
		}
		sort.Slice(missingIndexList, func(i, j int) bool {
			return missingIndexList[i].originPos < missingIndexList[j].originPos
		})
		for _, index := range missingIndexList {
			orderedList = append(orderedList, index.createIndex)
		}
	}

	orderedSDL, err := deparse(orderedList)
	if err != nil {
		return "", err
	}
	remainingStatement = append([]string{orderedSDL}, remainingStatement...)
	return strings.Join(remainingStatement, ""), nil
}

func removeRedundantColumnOption(table *ast.CreateTableStmt, standard *ast.CreateTableStmt) {
	columnSet := make(map[string]*ast.ColumnDef)
	for _, column := range standard.Cols {
		columnSet[column.Name.Name.O] = column
	}

	for _, column := range table.Cols {
		var newOptionList []*ast.ColumnOption
		if standardColumn, exists := columnSet[column.Name.Name.O]; exists {
			for _, option := range column.Options {
				if option.Tp == ast.ColumnOptionCollate {
					standardCollate := extractColumnCollate(standardColumn.Options)
					if standardCollate == nil {
						continue
					}
				} else if option.Tp == ast.ColumnOptionDefaultValue {
					standardDefault := extractColumnDefault(standardColumn.Options)
					if standardDefault == nil && option.Expr.GetType().GetType() == mysql.TypeNull {
						continue
					}
				}
				newOptionList = append(newOptionList, option)
			}
		}
		column.Options = newOptionList
	}
}

func extractColumnCollate(list []*ast.ColumnOption) *ast.ColumnOption {
	for _, option := range list {
		if option.Tp == ast.ColumnOptionCollate {
			return option
		}
	}
	return nil
}

func extractColumnDefault(list []*ast.ColumnOption) *ast.ColumnOption {
	for _, option := range list {
		if option.Tp == ast.ColumnOptionDefaultValue {
			return option
		}
	}
	return nil
}

func removeRedundantTableOption(table *ast.CreateTableStmt, standard *ast.CreateTableStmt) {
	engine, charset, collation := extractEngineCharsetAndCollation(standard)
	var newOptionList []*ast.TableOption
	for _, option := range table.Options {
		switch option.Tp {
		case ast.TableOptionEngine:
			if engine == nil {
				continue
			}
		case ast.TableOptionCharset:
			if charset == nil {
				continue
			}
		case ast.TableOptionCollate:
			if collation == nil {
				continue
			}
		}
		newOptionList = append(newOptionList, option)
	}
	table.Options = newOptionList
}

func extractEngineCharsetAndCollation(table *ast.CreateTableStmt) (engine, charset, collation *ast.TableOption) {
	for _, option := range table.Options {
		switch option.Tp {
		case ast.TableOptionEngine:
			engine = option
		case ast.TableOptionCharset:
			charset = option
		case ast.TableOptionCollate:
			collation = option
		}
	}
	return engine, charset, collation
}

// Check checks the schema format.
func (*SchemaTransformer) Check(schema string) (int, error) {
	list, err := bbparser.SplitMultiSQL(bbparser.MySQL, schema)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to split SQL")
	}

	changeDelimiter := false
	for _, stmt := range list {
		if bbparser.IsDelimiter(stmt.Text) {
			delimiter, err := bbparser.ExtractDelimiter(stmt.Text)
			if err != nil {
				return 0, errors.Wrapf(err, "failed to extract delimiter from %q", stmt.Text)
			}
			if delimiter == ";" {
				changeDelimiter = false
			} else {
				changeDelimiter = true
			}
			continue
		}
		if changeDelimiter {
			// TiDB parser cannot deal with delimiter change.
			// So we need to skip the statement if the delimiter is not `;`.
			continue
		}
		if bbparser.IsTiDBUnsupportDDLStmt(stmt.Text) {
			continue
		}
		nodeList, _, err := parser.New().Parse(stmt.Text, "", "")
		if err != nil {
			return stmt.LastLine, errors.Wrapf(err, "failed to parse schema %q", schema)
		}
		if len(nodeList) != 1 {
			return stmt.LastLine, errors.Errorf("Expect one statement after splitting but found %d", len(nodeList))
		}

		switch node := nodeList[0].(type) {
		case *ast.CreateTableStmt:
			for _, column := range node.Cols {
				for _, option := range column.Options {
					switch option.Tp {
					case ast.ColumnOptionNoOption,
						ast.ColumnOptionNotNull,
						ast.ColumnOptionAutoIncrement,
						ast.ColumnOptionDefaultValue,
						ast.ColumnOptionNull,
						ast.ColumnOptionOnUpdate,
						ast.ColumnOptionFulltext,
						ast.ColumnOptionComment,
						ast.ColumnOptionGenerated,
						ast.ColumnOptionCollate,
						ast.ColumnOptionColumnFormat,
						ast.ColumnOptionStorage,
						ast.ColumnOptionAutoRandom:
					case ast.ColumnOptionPrimaryKey:
						return stmt.LastLine, errors.Errorf("The column-level primary key constraint is invalid SDL format. Please use table-level primary key, such as \"CREATE TABLE t(id INT, PRIMARY KEY (id));\"")
					case ast.ColumnOptionUniqKey:
						return stmt.LastLine, errors.Errorf("The column-level unique key constraint is invalid SDL format. Please use table-level unique key, such as \"CREATE TABLE t(id INT, UNIQUE KEY uk_t_id (id));\"")
					case ast.ColumnOptionCheck:
						return stmt.LastLine, errors.Errorf("The column-level check constraint is invalid SDL format. Please use table-level check constraints, such as \"CREATE TABLE t(id INT, CONSTRAINT ck_t CHECK (id > 0));\"")
					case ast.ColumnOptionReference:
						return stmt.LastLine, errors.Errorf("The column-level foreign key constraint is invalid SDL format. Please use table-level foreign key constraints, such as \"CREATE TABLE t(id INT, CONSTRAINT fk_t_id FOREIGN KEY (id) REFERENCES t1(c1));\"")
					}
				}
			}
			for _, constraint := range node.Constraints {
				switch constraint.Tp {
				case ast.ConstraintKey, ast.ConstraintIndex:
					return stmt.LastLine, errors.Errorf("The index/key define in CREATE TABLE statements is invalid SDL format. Please use CREATE INDEX statements, such as \"CREATE INDEX idx_t_id ON t(id);\"")
				case ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex:
					return stmt.LastLine, errors.Errorf("The unique constraint in CREATE TABLE statements is invalid SDL format. Please use CREATE UNIQUE INDEX statements, such as \"CREATE UNIQUE INDEX uk_t_id ON t(id);\"")
				case ast.ConstraintFulltext:
					return stmt.LastLine, errors.Errorf("The fulltext constraint in CREATE TABLE statements is invalid SDL format. Please use CREATE FULLTEXT INDEX statements, such as \"CREATE UNIQUE INDEX fdx_t_id ON t(id);\"")
				case ast.ConstraintCheck, ast.ConstraintForeignKey:
					if constraint.Name == "" {
						return stmt.LastLine, errors.Errorf("The constraint name is required for SDL format")
					}
				}
			}
			if node.Partition != nil {
				return stmt.LastLine, errors.Errorf("The SDL does not support partition table currently")
			}
		case *ast.CreateIndexStmt:
		default:
			return stmt.LastLine, errors.Errorf("%T is invalid SDL statement", node)
		}
	}
	return 0, nil
}

// Transform returns the transformed schema.
func (*SchemaTransformer) Transform(schema string) (string, error) {
	var result []string
	list, err := bbparser.SplitMultiSQL(bbparser.MySQL, schema)
	if err != nil {
		return "", errors.Wrapf(err, "failed to split SQL")
	}

	changeDelimiter := false
	for _, stmt := range list {
		if bbparser.IsDelimiter(stmt.Text) {
			delimiter, err := bbparser.ExtractDelimiter(stmt.Text)
			if err != nil {
				return "", errors.Wrapf(err, "failed to extract delimiter from %q", stmt.Text)
			}
			if delimiter == ";" {
				changeDelimiter = false
			} else {
				changeDelimiter = true
			}
			result = append(result, stmt.Text+"\n\n")
			continue
		}
		if changeDelimiter {
			// TiDB parser cannot deal with delimiter change.
			// So we need to skip the statement if the delimiter is not `;`.
			result = append(result, stmt.Text+"\n\n")
			continue
		}
		if bbparser.IsTiDBUnsupportDDLStmt(stmt.Text) {
			result = append(result, stmt.Text+"\n\n")
			continue
		}
		nodeList, _, err := parser.New().Parse(stmt.Text, "", "")
		if err != nil {
			// If the TiDB parser cannot parse the statement, we just skip it.
			result = append(result, stmt.Text+"\n\n")
			continue
		}
		if len(nodeList) != 1 {
			return "", errors.Errorf("Expect one statement after splitting but found %d", len(nodeList))
		}

		switch node := nodeList[0].(type) {
		case *ast.CreateTableStmt:
			var constraintList []*ast.Constraint
			var indexList []*ast.CreateIndexStmt
			for _, constraint := range node.Constraints {
				switch constraint.Tp {
				case ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex:
					// This becomes the unique index.
					indexOption := constraint.Option
					if indexOption == nil {
						indexOption = &ast.IndexOption{}
					}
					indexList = append(indexList, &ast.CreateIndexStmt{
						IndexName: constraint.Name,
						Table: &ast.TableName{
							Name: model.NewCIStr(node.Table.Name.O),
						},
						IndexPartSpecifications: constraint.Keys,
						IndexOption:             indexOption,
						KeyType:                 ast.IndexKeyTypeUnique,
					})
				case ast.ConstraintIndex:
					// This becomes the index.
					indexOption := constraint.Option
					if indexOption == nil {
						indexOption = &ast.IndexOption{}
					}
					indexList = append(indexList, &ast.CreateIndexStmt{
						IndexName: constraint.Name,
						Table: &ast.TableName{
							Name: model.NewCIStr(node.Table.Name.O),
						},
						IndexPartSpecifications: constraint.Keys,
						IndexOption:             indexOption,
						KeyType:                 ast.IndexKeyTypeNone,
					})
				case ast.ConstraintPrimaryKey, ast.ConstraintKey, ast.ConstraintForeignKey, ast.ConstraintFulltext, ast.ConstraintCheck:
					constraintList = append(constraintList, constraint)
				}
			}
			node.Constraints = constraintList
			nodeList := []ast.Node{node}
			for _, node := range indexList {
				nodeList = append(nodeList, node)
			}
			text, err := deparse(nodeList)
			if err != nil {
				return "", errors.Wrapf(err, "failed to deparse %q", stmt.Text)
			}
			result = append(result, text)
		case *ast.SetStmt:
			// Skip these spammy set session variable statements.
			continue
		default:
			result = append(result, stmt.Text+"\n\n")
		}
	}

	return strings.Join(result, ""), nil
}

func deparse(newNodeList []ast.Node) (string, error) {
	var buf bytes.Buffer
	for _, node := range newNodeList {
		if err := node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags|format.RestoreStringWithoutCharset|format.RestorePrettyFormat, &buf)); err != nil {
			return "", err
		}
		if _, err := buf.Write([]byte(";\n\n")); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}
