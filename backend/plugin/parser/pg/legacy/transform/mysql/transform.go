// Package mysql provides the MySQL transformer plugin.
package mysql

import (
	"bytes"
	"strings"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	"github.com/pingcap/tidb/pkg/parser/model"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"

	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/transform"
)

var (
	_ transform.SchemaTransformer = (*SchemaTransformer)(nil)
)

func init() {
	transform.Register(storepb.Engine_MYSQL, &SchemaTransformer{})
	transform.Register(storepb.Engine_TIDB, &SchemaTransformer{})
	transform.Register(storepb.Engine_OCEANBASE, &SchemaTransformer{})
}

// SchemaTransformer it the transformer for MySQL dialect.
type SchemaTransformer struct {
}

// Transform returns the transformed schema.
func (*SchemaTransformer) Transform(schema string) (string, error) {
	var result []string
	list, err := mysqlparser.SplitSQL(schema)
	if err != nil {
		return "", errors.Wrapf(err, "failed to split SQL")
	}

	changeDelimiter := false
	for _, stmt := range list {
		if mysqlparser.IsDelimiter(stmt.Text) {
			delimiter, err := mysqlparser.ExtractDelimiter(stmt.Text)
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
		if tidbparser.IsTiDBUnsupportDDLStmt(stmt.Text) {
			result = append(result, stmt.Text+"\n\n")
			continue
		}
		nodeList, _, err := parser.New().Parse(stmt.Text, "", "")
		if err != nil {
			// If the TiDB parser cannot parse the statement, we just skip it.
			result = append(result, stmt.Text+"\n\n")
			continue
		}
		if len(nodeList) == 0 {
			continue
		}
		if len(nodeList) > 1 {
			return "", errors.Errorf("Expect one statement after splitting but found %d, text %q", len(nodeList), stmt.Text)
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
				default:
					// Other constraint types
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
		if _, err := buf.WriteString(";\n\n"); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}
