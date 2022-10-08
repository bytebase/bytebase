// Package mysql provides the MySQL differ plugin.
package mysql

import (
	"bytes"
	"sort"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pkg/errors"

	bbparser "github.com/bytebase/bytebase/plugin/parser"

	"github.com/bytebase/bytebase/plugin/parser/differ"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
)

var (
	_ differ.SchemaDiffer = (*SchemaDiffer)(nil)
)

func init() {
	differ.Register(bbparser.MySQL, &SchemaDiffer{})
	differ.Register(bbparser.TiDB, &SchemaDiffer{})
}

// SchemaDiffer it the parser for MySQL dialect.
type SchemaDiffer struct {
}

// constraintMap returns a map of constraint name to constraint.
type constraintMap map[string]*ast.Constraint

// SchemaDiff returns the schema diff.
func (*SchemaDiffer) SchemaDiff(oldStmt, newStmt string) (string, error) {
	oldNodes, _, err := parser.New().Parse(oldStmt, "", "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse old statement %q", oldStmt)
	}
	newNodes, _, err := parser.New().Parse(newStmt, "", "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse new statement %q", newStmt)
	}

	var diff []ast.Node

	oldTableMap := buildTableMap(oldNodes)

	for _, node := range newNodes {
		switch newStmt := node.(type) {
		case *ast.CreateTableStmt:
			tableName := newStmt.Table.Name.O
			oldStmt, ok := oldTableMap[tableName]
			if !ok {
				stmt := *newStmt
				stmt.IfNotExists = true
				diff = append(diff, &stmt)
				continue
			}
			indexMap := buildIndexMap(oldStmt)
			var alterTableAddColumnSpecs []*ast.AlterTableSpec
			var alterTableModifyColumnSpecs []*ast.AlterTableSpec
			var alterTableAddConstraintSpecs []*ast.AlterTableSpec
			var alterTableDropConstraintSpecs []*ast.AlterTableSpec

			oldColumnMap := buildColumnMap(oldNodes, newStmt.Table.Name)
			for _, columnDef := range newStmt.Cols {
				newColumnName := columnDef.Name.Name.O
				oldColumnDef, ok := oldColumnMap[newColumnName]
				if !ok {
					alterTableAddColumnSpecs = append(alterTableAddColumnSpecs, &ast.AlterTableSpec{
						Tp:         ast.AlterTableAddColumns,
						NewColumns: []*ast.ColumnDef{columnDef},
					})
					continue
				}
				// We need to compare the two column definitions.
				if !isColumnEqual(oldColumnDef, columnDef) {
					alterTableModifyColumnSpecs = append(alterTableModifyColumnSpecs, &ast.AlterTableSpec{
						Tp:         ast.AlterTableModifyColumn,
						NewColumns: []*ast.ColumnDef{columnDef},
						Position:   &ast.ColumnPosition{Tp: ast.ColumnPositionNone},
					})
				}
			}
			for _, constraint := range newStmt.Constraints {
				if constraint.Tp == ast.ConstraintIndex || constraint.Tp == ast.ConstraintKey ||
					constraint.Tp == ast.ConstraintUniq || constraint.Tp == ast.ConstraintUniqKey ||
					constraint.Tp == ast.ConstraintUniqIndex || constraint.Tp == ast.ConstraintFulltext {
					indexName := getIndexName(constraint)
					if oldConstraint, ok := indexMap[indexName]; ok {
						if isIndexEqual(constraint, oldConstraint) {
							delete(indexMap, indexName)
							continue
						}
					}
					alterTableAddConstraintSpecs = append(alterTableAddConstraintSpecs, &ast.AlterTableSpec{
						Tp:         ast.AlterTableAddConstraint,
						Constraint: constraint,
					})
				}
			}
			if len(alterTableAddColumnSpecs) > 0 {
				diff = append(diff, &ast.AlterTableStmt{
					Table: &ast.TableName{
						Name: model.NewCIStr(tableName),
					},
					Specs: alterTableAddColumnSpecs,
				})
			}
			if len(alterTableModifyColumnSpecs) > 0 {
				diff = append(diff, &ast.AlterTableStmt{
					Table: &ast.TableName{
						Name: model.NewCIStr(tableName),
					},
					Specs: alterTableModifyColumnSpecs,
				})
			}
			// We should drop the remaining indices in the indexMap.
			for indexName := range indexMap {
				alterTableDropConstraintSpecs = append(alterTableDropConstraintSpecs, &ast.AlterTableSpec{
					Tp:   ast.AlterTableDropIndex,
					Name: indexName,
				})
			}

			if len(alterTableAddConstraintSpecs) > 0 {
				diff = append(diff, &ast.AlterTableStmt{
					Table: &ast.TableName{
						Name: model.NewCIStr(tableName),
					},
					Specs: alterTableDropConstraintSpecs,
				})
			}

			if len(alterTableDropConstraintSpecs) > 0 {
				diff = append(diff, &ast.AlterTableStmt{
					Table: &ast.TableName{
						Name: model.NewCIStr(tableName),
					},
					Specs: alterTableAddConstraintSpecs,
				})
			}
		default:
		}
	}

	deparse := func(nodes []ast.Node) (string, error) {
		restoreFlags := format.DefaultRestoreFlags | format.RestoreStringWithoutCharset
		var buf bytes.Buffer
		for _, node := range nodes {
			if err := node.Restore(format.NewRestoreCtx(restoreFlags, &buf)); err != nil {
				return "", err
			}
			if _, err := buf.Write([]byte(";\n")); err != nil {
				return "", err
			}
		}
		return buf.String(), nil
	}
	return deparse(diff)
}

// buildTableMap returns a map of table name to create table statements.
func buildTableMap(nodes []ast.StmtNode) map[string]*ast.CreateTableStmt {
	tableMap := make(map[string]*ast.CreateTableStmt)
	for _, node := range nodes {
		switch stmt := node.(type) {
		case *ast.CreateTableStmt:
			tableName := stmt.Table.Name.String()
			tableMap[tableName] = stmt
		default:
		}
	}
	return tableMap
}

// buildColumnMap returns a map of column name to column definition on a given table.
func buildColumnMap(nodes []ast.StmtNode, tableName model.CIStr) map[string]*ast.ColumnDef {
	oldColumnMap := make(map[string]*ast.ColumnDef)
	for _, node := range nodes {
		switch stmt := node.(type) {
		case *ast.CreateTableStmt:
			if stmt.Table.Name.O != tableName.O {
				continue
			}
			for _, columnDef := range stmt.Cols {
				oldColumnMap[columnDef.Name.Name.O] = columnDef
			}
		default:
		}
	}
	return oldColumnMap
}

// buildIndexMap build a map of index name to constraint on given table name.
func buildIndexMap(stmt *ast.CreateTableStmt) constraintMap {
	indexMap := make(constraintMap)
	for _, constraint := range stmt.Constraints {
		if constraint.Tp == ast.ConstraintIndex || constraint.Tp == ast.ConstraintKey ||
			constraint.Tp == ast.ConstraintUniq || constraint.Tp == ast.ConstraintUniqKey ||
			constraint.Tp == ast.ConstraintUniqIndex || constraint.Tp == ast.ConstraintFulltext {
			indexName := getIndexName(constraint)
			indexMap[indexName] = constraint
		}
	}
	return indexMap
}

// isColumnEqual returns true if definitions of two columns with the same name are the same.
func isColumnEqual(old, new *ast.ColumnDef) bool {
	if !isColumnTypesEqual(old, new) {
		return false
	}
	if !isColumnOptionsEqual(old.Options, new.Options) {
		return false
	}
	return true
}

func isColumnTypesEqual(old, new *ast.ColumnDef) bool {
	return old.Tp.String() == new.Tp.String()
}

func isColumnOptionsEqual(old, new []*ast.ColumnOption) bool {
	oldNormalizeOptions := normalizeColumnOptions(old)
	newNormalizeOptions := normalizeColumnOptions(new)
	if len(oldNormalizeOptions) != len(newNormalizeOptions) {
		return false
	}
	for idx, oldOption := range oldNormalizeOptions {
		newOption := newNormalizeOptions[idx]
		if oldOption.Tp != newOption.Tp {
			return false
		}
		// TODO(zp): it's not enough to compare the type for some options.
		switch oldOption.Tp {
		case ast.ColumnOptionComment, ast.ColumnOptionDefaultValue:
			if oldOption.Expr.(ast.ValueExpr).GetValue() != newOption.Expr.(ast.ValueExpr).GetValue() {
				return false
			}
		case ast.ColumnOptionCollate:
			if oldOption.StrValue != newOption.StrValue {
				return false
			}
		default:
		}
	}
	return true
}

// normalizeColumnOptions normalizes the column options.
// It skips the NULL option, NO option and then order the options by OptionType.
func normalizeColumnOptions(options []*ast.ColumnOption) []*ast.ColumnOption {
	var retOptions []*ast.ColumnOption
	for _, option := range options {
		if option.Tp == ast.ColumnOptionNull || option.Tp == ast.ColumnOptionNoOption {
			continue
		}
		retOptions = append(retOptions, option)
	}
	sort.Slice(retOptions, func(i, j int) bool {
		return retOptions[i].Tp < retOptions[j].Tp
	})
	return retOptions
}

// isIndexEqual returns true if definitions of two indexes are the same.
func isIndexEqual(old, new *ast.Constraint) bool {
	// {INDEX | KEY} [index_name] [index_type] (key_part,...) [index_option] ...
	if old.Name != new.Name {
		return false
	}
	if (old.Option == nil) != (new.Option == nil) {
		return false
	}
	if old.Option != nil && new.Option != nil {
		if old.Option.Tp != new.Option.Tp {
			return false
		}
	}

	if !isKeyPartEqual(old.Keys, new.Keys) {
		return false
	}
	if !isIndexOptionEqual(old.Option, new.Option) {
		return false
	}
	return true
}

// isKeyPartEqual returns true if two key parts are the same.
func isKeyPartEqual(old, new []*ast.IndexPartSpecification) bool {
	if len(old) != len(new) {
		return false
	}
	// key_part: {col_name [(length)] | (expr)} [ASC | DESC]
	for idx, oldKeyPart := range old {
		newKeyPart := new[idx]
		if (oldKeyPart.Column == nil) != (newKeyPart.Column == nil) {
			return false
		}
		if oldKeyPart.Column != nil && newKeyPart.Column != nil {
			if oldKeyPart.Column.Name.String() != newKeyPart.Column.Name.String() {
				return false
			}
			if oldKeyPart.Length != newKeyPart.Length {
				return false
			}
		}
		if (oldKeyPart.Expr == nil) != (newKeyPart.Expr == nil) {
			// if the key part uses expression instead of column name, the expression node is nil.
			return false
		}
		if oldKeyPart.Expr != nil && newKeyPart.Expr != nil {
			var oldExpr bytes.Buffer
			var newExpr bytes.Buffer
			if err := oldKeyPart.Expr.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &oldExpr)); err != nil {
				// Return error will cause the logic to be more complicated, so we just return false here.
				return false
			}
			if err := newKeyPart.Expr.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &newExpr)); err != nil {
				return false
			}
			if oldExpr.String() != newExpr.String() {
				return false
			}
		}
		// TODO(zp): TiDB MySQL parser doesn't record the index order field in go struct, but it can parse correctly.
		// https://sourcegraph.com/github.com/pingcap/tidb/-/blob/parser/parser.y?L3688
		// We can support the index order until we implement the parser by ourself or https://github.com/pingcap/tidb/pull/38137 is merged.
	}
	return true
}

// isIndexOptionEqual returns true if two index options are the same.
func isIndexOptionEqual(old, new *ast.IndexOption) bool {
	// index_option: {
	// 	KEY_BLOCK_SIZE [=] value
	//   | index_type
	//   | WITH PARSER parser_name
	//   | COMMENT 'string'
	//   | {VISIBLE | INVISIBLE}
	//   |ENGINE_ATTRIBUTE [=] 'string'
	//   |SECONDARY_ENGINE_ATTRIBUTE [=] 'string'
	// }
	if (old == nil) != (new == nil) {
		return false
	}
	if old == nil && new == nil {
		return true
	}
	if old.KeyBlockSize != new.KeyBlockSize {
		return false
	}
	if old.Tp != new.Tp {
		return false
	}
	if old.ParserName.O != new.ParserName.O {
		return false
	}
	if old.Comment != new.Comment {
		return false
	}
	if old.Visibility != new.Visibility {
		return false
	}
	// TODO(zp): support ENGINE_ATTRIBUTE and SECONDARY_ENGINE_ATTRIBUTE.
	return true
}

// getIndexName returns the name of the index.
func getIndexName(index *ast.Constraint) string {
	if index.Name != "" {
		return index.Name
	}
	// If the index name is empty, it will be generated by MySQL.
	// https://dba.stackexchange.com/questions/160708/is-naming-an-index-required
	// TODO(zp): handle the duplicated index name.
	return index.Keys[0].Column.Name.O
}
