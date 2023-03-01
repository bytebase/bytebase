// Package mysql provides the MySQL transformer plugin.
package mysql

import (
	"bytes"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pkg/errors"

	bbparser "github.com/bytebase/bytebase/backend/plugin/parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/transform"
	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
)

var (
	_ transform.SchemaTransformer = (*SchemaTransformer)(nil)
)

func init() {
	transform.Register(bbparser.MySQL, &SchemaTransformer{})
	transform.Register(bbparser.TiDB, &SchemaTransformer{})
}

// SchemaTransformer it the transformer for MySQL dialect.
type SchemaTransformer struct {
}

// Accepted MySQL SDL Format:
// 1. CREATE TABLE statements.
//    i.  Column define without constraints.
//    ii. Primary key, check and foreign key constraints define in table-level.
// 2. CREATE INDEX statements.

// Check checks the schema format.
func (*SchemaTransformer) Check(schema string) (int, error) {
	list, err := bbparser.SplitMultiSQL(bbparser.MySQL, schema)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to split SQL")
	}

	for _, stmt := range list {
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
	nodes, _, err := parser.New().Parse(schema, "", "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse schema %q", schema)
	}
	var newNodeList []ast.Node
	for _, node := range nodes {
		switch newStmt := node.(type) {
		case *ast.CreateTableStmt:
			var constraintList []*ast.Constraint
			var indexList []*ast.CreateIndexStmt
			for _, constraint := range newStmt.Constraints {
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
							Name: model.NewCIStr(newStmt.Table.Name.O),
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
							Name: model.NewCIStr(newStmt.Table.Name.O),
						},
						IndexPartSpecifications: constraint.Keys,
						IndexOption:             indexOption,
						KeyType:                 ast.IndexKeyTypeNone,
					})
				case ast.ConstraintPrimaryKey, ast.ConstraintKey, ast.ConstraintForeignKey, ast.ConstraintFulltext, ast.ConstraintCheck:
					constraintList = append(constraintList, constraint)
				}
			}
			newStmt.Constraints = constraintList
			newNodeList = append(newNodeList, newStmt)
			for _, node := range indexList {
				newNodeList = append(newNodeList, node)
			}
		case *ast.SetStmt:
			// Skip these spammy set session variable statements.
			continue
		default:
			newNodeList = append(newNodeList, node)
		}
	}

	return deparse(newNodeList)
}

func deparse(newNodeList []ast.Node) (string, error) {
	var buf bytes.Buffer
	for _, node := range newNodeList {
		if err := node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags|format.RestoreStringWithoutCharset|format.RestorePrettyFormat, &buf)); err != nil {
			return "", err
		}
		if _, err := buf.Write([]byte(";\n")); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}
