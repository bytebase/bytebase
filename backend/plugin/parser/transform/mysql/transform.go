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
