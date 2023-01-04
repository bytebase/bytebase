// Package mysql provides the MySQL differ plugin.
package mysql

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"go.uber.org/zap"

	"github.com/bytebase/bytebase/common/log"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/types"
	"github.com/pkg/errors"

	bbparser "github.com/bytebase/bytebase/plugin/parser"

	"github.com/bytebase/bytebase/plugin/parser/differ"

	// Register pingcap parser driver.
	driver "github.com/pingcap/tidb/types/parser_driver"
)

var (
	_ differ.SchemaDiffer = (*SchemaDiffer)(nil)
)

var (
	tableNameRegexpNeedFilter = map[*regexp.Regexp]bool{}
)

func init() {
	differ.Register(bbparser.MySQL, &SchemaDiffer{})
	differ.Register(bbparser.TiDB, &SchemaDiffer{})

	// Compile the regexp.
	// Filter out ghost del table.
	ghostDelTableRegexp := regexp.MustCompile(`^~.+_\d+_del$`)
	tableNameRegexpNeedFilter[ghostDelTableRegexp] = true
}

const (
	disableFKCheckStmt string = "SET FOREIGN_KEY_CHECKS=0;\n"
	enableFKCheckStmt  string = "SET FOREIGN_KEY_CHECKS=1;\n"
)

// SchemaDiffer it the parser for MySQL dialect.
type SchemaDiffer struct {
}

// constraintMap returns a map of constraint name to constraint.
type constraintMap map[string]*ast.Constraint

// SchemaDiff returns the schema diff.
// It only supports schema information from mysqldump.
func (*SchemaDiffer) SchemaDiff(oldStmt, newStmt string) (string, error) {
	// TiDB parser doesn't support some statements like `CREATE EVENT`, so we need to extract them out and diff them based on string compare.
	oldUnsupportStmts, oldSupportStmts, err := bbparser.ExtractTiDBUnsupportStmts(oldStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to extract TiDB unsupport statements from old statements %q", oldStmt)
	}
	var oldUnsupportFilterStmts []string
	for _, stmt := range oldUnsupportStmts {
		if !bbparser.IsDelimiter(stmt) {
			oldUnsupportFilterStmts = append(oldUnsupportFilterStmts, stmt)
		}
	}

	newUnsupportStmts, newSupportStmts, err := bbparser.ExtractTiDBUnsupportStmts(newStmt)
	if err != nil {
		return "", errors.Wrapf(err, "failed to extract TiDB unsupport statements from old statements %q", oldStmt)
	}
	var newUnsupportFilterStmts []string
	for _, stmt := range newUnsupportStmts {
		if !bbparser.IsDelimiter(stmt) {
			newUnsupportFilterStmts = append(newUnsupportFilterStmts, stmt)
		}
	}

	oldNodes, _, err := parser.New().Parse(oldSupportStmts, "", "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse old statement %q", oldStmt)
	}
	newNodes, _, err := parser.New().Parse(newSupportStmts, "", "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse new statement %q", newStmt)
	}

	var newNodeList []ast.Node
	var inplaceUpdate []ast.Node
	// inplaceDropNodeList and inplaceAddNodeList are used to handle destructive node updates.
	// For example, we should drop the old index named 'id_idx' and then add a new index named 'id_idx' in the same table.
	var inplaceDropNodeList []ast.Node
	var inplaceAddNodeList []ast.Node
	var dropNodeList []ast.Node
	var viewStmts []*ast.CreateViewStmt

	oldTableMap := buildTableMap(oldNodes, tableNameRegexpNeedFilter)
	oldViewMap := buildViewMap(oldNodes)
	newViewMap := buildViewMap(newNodes)
	var newViewList []*ast.CreateViewStmt

	for _, node := range newNodes {
		switch newStmt := node.(type) {
		case *ast.CreateTableStmt:
			tableName := newStmt.Table.Name.O
			if matchFilterRegexp(tableName, tableNameRegexpNeedFilter) {
				continue
			}
			oldStmt, ok := oldTableMap[tableName]
			if !ok {
				stmt := *newStmt
				stmt.IfNotExists = true
				newNodeList = append(newNodeList, &stmt)
				continue
			}
			if alterTableOptionStmt := diffTableOptions(newStmt.Table, oldStmt.Options, newStmt.Options); alterTableOptionStmt != nil {
				inplaceUpdate = append(inplaceUpdate, alterTableOptionStmt)
			}
			indexMap := buildIndexMap(oldStmt)
			constraintMap := buildConstraintMap(oldStmt)
			var alterTableAddColumnSpecs []*ast.AlterTableSpec
			var alterTableDropColumnSpecs []*ast.AlterTableSpec
			var alterTableModifyColumnSpecs []*ast.AlterTableSpec
			var alterTableAddNewConstraintSpecs []*ast.AlterTableSpec
			var alterTableDropExcessConstraintSpecs []*ast.AlterTableSpec
			var alterTableInplaceAddConstraintSpecs []*ast.AlterTableSpec
			var alterTableInplaceDropConstraintSpecs []*ast.AlterTableSpec

			oldColumnMap := buildColumnMap(oldNodes, newStmt.Table.Name)
			oldColumnPositionMap := buildColumnPositionMap(oldStmt)
			for idx, columnDef := range newStmt.Cols {
				newColumnName := columnDef.Name.Name.O
				oldColumnDef, ok := oldColumnMap[newColumnName]
				if !ok {
					columnPosition := &ast.ColumnPosition{Tp: ast.ColumnPositionFirst}
					if idx >= 1 {
						columnPosition.Tp = ast.ColumnPositionAfter
						columnPosition.RelativeColumn = &ast.ColumnName{Name: model.NewCIStr(newStmt.Cols[idx-1].Name.Name.O)}
					}
					alterTableAddColumnSpecs = append(alterTableAddColumnSpecs, &ast.AlterTableSpec{
						Tp:         ast.AlterTableAddColumns,
						NewColumns: []*ast.ColumnDef{columnDef},
						Position:   columnPosition,
					})
					continue
				}

				// Compare the column positions.
				columnPosition := &ast.ColumnPosition{Tp: ast.ColumnPositionNone}
				columnPosInOld := oldColumnPositionMap[newColumnName]
				if hasColumnsIntersection(oldStmt.Cols[:columnPosInOld], newStmt.Cols[idx+1:]) {
					if idx == 0 {
						columnPosition.Tp = ast.ColumnPositionFirst
					} else {
						columnPosition.Tp = ast.ColumnPositionAfter
						columnPosition.RelativeColumn = &ast.ColumnName{Name: model.NewCIStr(newStmt.Cols[idx-1].Name.Name.O)}
					}
				}
				// Compare the column definitions.
				if !isColumnEqual(oldColumnDef, columnDef) || columnPosition.Tp != ast.ColumnPositionNone {
					alterTableModifyColumnSpecs = append(alterTableModifyColumnSpecs, &ast.AlterTableSpec{
						Tp:         ast.AlterTableModifyColumn,
						NewColumns: []*ast.ColumnDef{columnDef},
						Position:   columnPosition,
					})
				}
				delete(oldColumnMap, newColumnName)
			}
			// TODO(zp): add an option to control whether to drop the excess columns.
			for _, columnDef := range oldColumnMap {
				alterTableDropColumnSpecs = append(alterTableDropColumnSpecs, &ast.AlterTableSpec{
					Tp: ast.AlterTableDropColumn,
					OldColumnName: &ast.ColumnName{
						Name: model.NewCIStr(columnDef.Name.Name.O),
					},
				})
			}
			// Compare the create definitions
			for _, constraint := range newStmt.Constraints {
				switch constraint.Tp {
				case ast.ConstraintIndex, ast.ConstraintKey, ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex, ast.ConstraintFulltext:
					indexName := constraint.Name
					if oldConstraint, ok := indexMap[indexName]; ok {
						if !isIndexEqual(constraint, oldConstraint) {
							alterTableInplaceDropConstraintSpecs = append(alterTableInplaceDropConstraintSpecs, &ast.AlterTableSpec{
								Tp:   ast.AlterTableDropIndex,
								Name: indexName,
							})
							alterTableInplaceAddConstraintSpecs = append(alterTableInplaceAddConstraintSpecs, &ast.AlterTableSpec{
								Tp:         ast.AlterTableAddConstraint,
								Constraint: constraint,
							})
						}
						delete(indexMap, indexName)
						continue
					}
					alterTableAddNewConstraintSpecs = append(alterTableAddNewConstraintSpecs, &ast.AlterTableSpec{
						Tp:         ast.AlterTableAddConstraint,
						Constraint: constraint,
					})
				case ast.ConstraintPrimaryKey:
					primaryKeyName := "PRIMARY"
					if oldConstraint, ok := indexMap[primaryKeyName]; ok {
						if !isIndexEqual(constraint, oldConstraint) {
							alterTableInplaceDropConstraintSpecs = append(alterTableInplaceDropConstraintSpecs, &ast.AlterTableSpec{
								Tp: ast.AlterTableDropPrimaryKey,
							})
							alterTableInplaceAddConstraintSpecs = append(alterTableInplaceAddConstraintSpecs, &ast.AlterTableSpec{
								Tp:         ast.AlterTableAddConstraint,
								Constraint: constraint,
							})
						}
						delete(indexMap, primaryKeyName)
						continue
					}
					alterTableAddNewConstraintSpecs = append(alterTableAddNewConstraintSpecs, &ast.AlterTableSpec{
						Tp:         ast.AlterTableAddConstraint,
						Constraint: constraint,
					})
				// The parent column in the foreign key always needs an index, so in the case of referencing itself,
				// we need to drop the foreign key before dropping the primary key.
				// Since the mysqldump statement always puts the primary key in front of the foreign key, and we will reverse the drop statements order.
				// TODO(zp): So we don't have to worry about this now until one of the statements doesn't come from mysqldump.
				case ast.ConstraintForeignKey:
					if oldConstraint, ok := constraintMap[constraint.Name]; ok {
						if !isForeignKeyConstraintEqual(constraint, oldConstraint) {
							alterTableInplaceDropConstraintSpecs = append(alterTableInplaceDropConstraintSpecs, &ast.AlterTableSpec{
								Tp:   ast.AlterTableDropForeignKey,
								Name: constraint.Name,
							})
							alterTableInplaceAddConstraintSpecs = append(alterTableInplaceAddConstraintSpecs, &ast.AlterTableSpec{
								Tp:         ast.AlterTableAddConstraint,
								Constraint: constraint,
							})
						}
						delete(constraintMap, constraint.Name)
						continue
					}
					alterTableAddNewConstraintSpecs = append(alterTableAddNewConstraintSpecs, &ast.AlterTableSpec{
						Tp:         ast.AlterTableAddConstraint,
						Constraint: constraint,
					})
				case ast.ConstraintCheck:
					if oldConstraint, ok := constraintMap[constraint.Name]; ok {
						if !isCheckConstraintEqual(constraint, oldConstraint) {
							alterTableInplaceDropConstraintSpecs = append(alterTableInplaceDropConstraintSpecs, &ast.AlterTableSpec{
								Tp:         ast.AlterTableDropCheck,
								Constraint: constraint,
							})
							alterTableInplaceAddConstraintSpecs = append(alterTableInplaceAddConstraintSpecs, &ast.AlterTableSpec{
								Tp:         ast.AlterTableAddConstraint,
								Constraint: constraint,
							})
						}
						delete(constraintMap, constraint.Name)
						continue
					}
					alterTableAddNewConstraintSpecs = append(alterTableAddNewConstraintSpecs, &ast.AlterTableSpec{
						Tp:         ast.AlterTableAddConstraint,
						Constraint: constraint,
					})
				}
			}
			if len(alterTableAddColumnSpecs) > 0 {
				newNodeList = append(newNodeList, &ast.AlterTableStmt{
					Table: &ast.TableName{
						Name: model.NewCIStr(tableName),
					},
					Specs: alterTableAddColumnSpecs,
				})
			}
			if len(alterTableDropColumnSpecs) > 0 {
				dropNodeList = append(dropNodeList, &ast.AlterTableStmt{
					Table: &ast.TableName{
						Name: model.NewCIStr(tableName),
					},
					Specs: alterTableDropColumnSpecs,
				})
			}
			if len(alterTableModifyColumnSpecs) > 0 {
				inplaceUpdate = append(inplaceUpdate, &ast.AlterTableStmt{
					Table: &ast.TableName{
						Name: model.NewCIStr(tableName),
					},
					Specs: alterTableModifyColumnSpecs,
				})
			}
			// Drop the remaining indices.
			for indexName, constraint := range indexMap {
				switch constraint.Tp {
				case ast.ConstraintIndex, ast.ConstraintKey, ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex, ast.ConstraintFulltext:
					alterTableDropExcessConstraintSpecs = append(alterTableDropExcessConstraintSpecs, &ast.AlterTableSpec{
						Tp:   ast.AlterTableDropIndex,
						Name: indexName,
					})
				case ast.ConstraintPrimaryKey:
					alterTableDropExcessConstraintSpecs = append(alterTableDropExcessConstraintSpecs, &ast.AlterTableSpec{
						Tp: ast.AlterTableDropPrimaryKey,
					})
				}
			}
			// Drop the remaining constraints.
			for constraintName, constraint := range constraintMap {
				switch constraint.Tp {
				case ast.ConstraintForeignKey:
					alterTableDropExcessConstraintSpecs = append(alterTableDropExcessConstraintSpecs, &ast.AlterTableSpec{
						Tp:   ast.AlterTableDropForeignKey,
						Name: constraintName,
					})
				case ast.ConstraintCheck:
					alterTableDropExcessConstraintSpecs = append(alterTableDropExcessConstraintSpecs, &ast.AlterTableSpec{
						Tp:         ast.AlterTableDropCheck,
						Constraint: constraint,
					})
				}
			}

			if len(alterTableAddNewConstraintSpecs) > 0 {
				newNodeList = append(newNodeList, &ast.AlterTableStmt{
					Table: &ast.TableName{
						Name: model.NewCIStr(tableName),
					},
					Specs: alterTableAddNewConstraintSpecs,
				})
			}

			if len(alterTableDropExcessConstraintSpecs) > 0 {
				dropNodeList = append(dropNodeList, &ast.AlterTableStmt{
					Table: &ast.TableName{
						Name: model.NewCIStr(tableName),
					},
					Specs: alterTableDropExcessConstraintSpecs,
				})
			}

			if len(alterTableInplaceDropConstraintSpecs) > 0 {
				inplaceDropNodeList = append(inplaceDropNodeList, &ast.AlterTableStmt{
					Table: &ast.TableName{
						Name: model.NewCIStr(tableName),
					},
					Specs: alterTableInplaceDropConstraintSpecs,
				})
			}

			if len(alterTableInplaceAddConstraintSpecs) > 0 {
				inplaceAddNodeList = append(inplaceAddNodeList, &ast.AlterTableStmt{
					Table: &ast.TableName{
						Name: model.NewCIStr(tableName),
					},
					Specs: alterTableInplaceAddConstraintSpecs,
				})
			}
			delete(oldTableMap, tableName)
		case *ast.CreateViewStmt:
			newViewList = append(newViewList, newStmt)
		}
	}

	// We compare the CREATE TRIGGER/EVENT/FUNCTION/PROCEDURE statements based on strcmp.
	var newNodeStmt []string
	var inplaceDropStmt []string
	var inplaceAddStmt []string
	var dropStmt []string

	oldUnsupportMap, err := buildUnsupportObjectMap(oldUnsupportFilterStmts)
	if err != nil {
		return "", err
	}
	newUnsupportMap, err := buildUnsupportObjectMap(newUnsupportFilterStmts)
	if err != nil {
		return "", err
	}
	for tp, objs := range newUnsupportMap {
		for newName, newStmt := range objs {
			if oldStmt, ok := oldUnsupportMap[tp][newName]; ok {
				if strings.Compare(oldStmt, newStmt) != 0 {
					// We should drop the old function and create the new function.
					// https://dev.mysql.com/doc/refman/8.0/en/drop-procedure.html
					// https://dev.mysql.com/doc/refman/5.7/en/drop-procedure.html
					inplaceDropStmt = append(inplaceDropStmt, fmt.Sprintf("DROP %s IF EXISTS `%s`;", tp, newName))
					inplaceAddStmt = append(inplaceAddStmt, newStmt)
				}
				delete(oldUnsupportMap[tp], newName)
				continue
			}
			// Now, the input of differ comes from the our mysqldump, mysqldump use ;; to separate the CREATE TRIGGER/FUNCTION/PROCEDURE/EVENT statements;
			// So we should append DELIMITER statement to the newStmt.
			delimiterNewStmt := fmt.Sprintf("DELIMITER ;;\n%s\nDELIMITER ;\n", newStmt)
			newNodeStmt = append(newNodeStmt, delimiterNewStmt)
		}
	}
	// drop remaining TiDB unsupported objects
	for tp, objs := range oldUnsupportMap {
		for name := range objs {
			dropStmt = append(dropStmt, fmt.Sprintf("DROP %s IF EXISTS `%s`;", tp, name))
		}
	}

	var tempViewList []*ast.CreateViewStmt
	var viewList []*ast.CreateViewStmt
	for _, view := range newViewList {
		viewName := view.ViewName.Name.O
		if newNode, ok := newViewMap[viewName]; ok {
			if !isViewEqual(view, newNode) {
				// Skip predefined view such as the temporary view from mysqldump.
				continue
			}
		}
		oldNode, ok := oldViewMap[viewName]
		if ok {
			if !isViewEqual(view, oldNode) {
				createViewStmt := *view
				createViewStmt.OrReplace = true
				viewList = append(viewList, &createViewStmt)
			}
			// We should delete the view in the oldViewMap, because we will drop the all views in the oldViewMap explicitly at last.
			delete(oldViewMap, viewName)
		} else {
			// We should create the view.
			// We create the temporary view first and replace it to avoid break the rependency like mysqldump does.
			tempViewStmt := getTempView(view)
			tempViewList = append(tempViewList, tempViewStmt)
			createViewStmt := *view
			createViewStmt.OrReplace = true
			viewList = append(viewList, &createViewStmt)
		}
	}
	viewStmts = append(viewStmts, tempViewList...)
	viewStmts = append(viewStmts, viewList...)

	// Remove the remaining views in the oldViewMap.
	dropViewStmt := &ast.DropTableStmt{
		IsView: true,
	}
	for _, oldView := range oldViewMap {
		dropViewStmt.Tables = append(dropViewStmt.Tables, oldView.ViewName)
	}
	if len(dropViewStmt.Tables) > 0 {
		dropNodeList = append(dropNodeList, dropViewStmt)
	}

	// TODO(zp): Add an option to control whether to drop the excess table.
	for _, oldTable := range oldTableMap {
		dropTableStmt := &ast.DropTableStmt{
			Tables: []*ast.TableName{oldTable.Table},
		}
		dropNodeList = append(dropNodeList, dropTableStmt)
	}

	var buf bytes.Buffer
	if err := deparse(&buf, newNodeList, newNodeStmt, inplaceUpdate,
		inplaceAddNodeList, inplaceAddStmt, inplaceDropNodeList,
		inplaceDropStmt, dropNodeList, dropStmt, viewStmts,
		format.DefaultRestoreFlags|format.RestoreStringWithoutCharset|format.RestorePrettyFormat); err != nil {
		return "", errors.Wrapf(err, "deparse failed")
	}
	if buf.Len() > 0 {
		return fmt.Sprintf("%s%s%s", disableFKCheckStmt, buf.String(), enableFKCheckStmt), nil
	}
	return "", nil
}

// deparse deparses the ast node list and stmt list to sql string and write it to the out.
func deparse(out io.Writer, newNodeList []ast.Node, newNodeStmt []string, inplaceUpdate []ast.Node,
	inplaceAdd []ast.Node, inplaceAddStmt []string, inplaceDrop []ast.Node,
	inplaceDropStmt []string, dropNodeList []ast.Node, dropStmt []string,
	viewStmts []*ast.CreateViewStmt, flag format.RestoreFlags) error {
	// We should following the right order to avoid break the dependency:
	// Additions for new nodes.
	// Updates for in-place node updates.
	// Deletions for destructive (none in-place) node updates (in reverse order).
	// Additions for destructive node updates.
	// Deletions for deleted nodes (in reverse order).
	if err := writeNodeStatementList(out, newNodeList, flag, false /*reverse*/); err != nil {
		return errors.Wrap(err, "failed to write new node list" /*reverse*/)
	}
	if err := writeStringStatementList(out, newNodeStmt, false /*reverse*/); err != nil {
		return errors.Wrap(err, "failed to write new node statement list")
	}
	if err := writeNodeStatementList(out, inplaceUpdate, flag, false /*reverse*/); err != nil {
		return errors.Wrap(err, "failed to write inplace update node list")
	}
	if err := writeNodeStatementList(out, inplaceDrop, flag, true /*reverse*/); err != nil {
		return errors.Wrap(err, "failed to write inplace drop node list")
	}
	if err := writeStringStatementList(out, inplaceDropStmt, true /*reverse*/); err != nil {
		return errors.Wrap(err, "failed to write inplace drop statement list")
	}
	if err := writeNodeStatementList(out, inplaceAdd, flag, false /*reverse*/); err != nil {
		return errors.Wrap(err, "failed to write inplace add node list")
	}
	if err := writeStringStatementList(out, inplaceAddStmt, false /*reverse*/); err != nil {
		return errors.Wrap(err, "failed to write inplace add statement list")
	}
	if err := writeNodeStatementList(out, dropNodeList, flag, true /*reverse*/); err != nil {
		return errors.Wrap(err, "failed to write drop node list")
	}
	if err := writeStringStatementList(out, dropStmt, true /*reverse*/); err != nil {
		return errors.Wrap(err, "failed to write drop statement list")
	}
	for _, node := range viewStmts {
		if err := node.Restore(format.NewRestoreCtx(flag, out)); err != nil {
			return err
		}
		if _, err := out.Write([]byte(";\n\n")); err != nil {
			return err
		}
	}
	return nil
}

func writeStringStatement(w io.Writer, s string) error {
	if _, err := w.Write([]byte(s)); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n\n")); err != nil {
		return err
	}
	return nil
}

func writeStringStatementList(w io.Writer, ss []string, reverse bool) error {
	if reverse {
		for i := len(ss) - 1; i >= 0; i-- {
			if err := writeStringStatement(w, ss[i]); err != nil {
				return err
			}
		}
	} else {
		for _, s := range ss {
			if err := writeStringStatement(w, s); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeNodeStatement(w io.Writer, n ast.Node, flags format.RestoreFlags) error {
	restoreCtx := format.NewRestoreCtx(flags, w)
	if err := n.Restore(restoreCtx); err != nil {
		return err
	}
	if _, err := w.Write([]byte(";\n\n")); err != nil {
		return err
	}
	return nil
}

func writeNodeStatementList(w io.Writer, ns []ast.Node, flags format.RestoreFlags, reverse bool) error {
	if reverse {
		for i := len(ns) - 1; i >= 0; i-- {
			if err := writeNodeStatement(w, ns[i], flags); err != nil {
				return err
			}
		}
	} else {
		for _, n := range ns {
			if err := writeNodeStatement(w, n, flags); err != nil {
				return err
			}
		}
	}
	return nil
}

// buildTableMap returns a map of table name to create table statements.
func buildTableMap(nodes []ast.StmtNode, filterRegexp map[*regexp.Regexp]bool) map[string]*ast.CreateTableStmt {
	tableMap := make(map[string]*ast.CreateTableStmt)
	for _, node := range nodes {
		switch stmt := node.(type) {
		case *ast.CreateTableStmt:
			tableName := stmt.Table.Name.String()
			if matchFilterRegexp(tableName, filterRegexp) {
				continue
			}
			tableMap[tableName] = stmt
		default:
		}
	}
	return tableMap
}

func matchFilterRegexp(name string, filterRegexp map[*regexp.Regexp]bool) bool {
	for re := range filterRegexp {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

// buildViewMap returns a map of view name to create view statements.
func buildViewMap(nodes []ast.StmtNode) map[string]*ast.CreateViewStmt {
	viewMap := make(map[string]*ast.CreateViewStmt)
	for _, node := range nodes {
		if stmt, ok := node.(*ast.CreateViewStmt); ok {
			viewName := stmt.ViewName.Name.O
			viewMap[viewName] = stmt
		}
	}
	return viewMap
}

// buildUnsupportObjectMap builds map for trigger, function, procedure, event to correspond create object string statements.
func buildUnsupportObjectMap(stmts []string) (map[objectType]map[string]string, error) {
	m := make(map[objectType]map[string]string)
	m[trigger] = make(map[string]string)
	m[function] = make(map[string]string)
	m[procedure] = make(map[string]string)
	m[event] = make(map[string]string)
	for _, stmt := range stmts {
		objName, objType, err := extractUnsupportObjNameAndType(stmt)
		if err != nil {
			return nil, err
		}
		m[objType][objName] = stmt
	}
	return m, nil
}

// getTempView returns the temporary view name and the create statement.
func getTempView(stmt *ast.CreateViewStmt) *ast.CreateViewStmt {
	// We create the temp view similar to what mysqldump does.
	// Create a temporary view with the same name as the view. Its columns should
	// have the same name in order to satisfy views that depend on this view.
	// This temporary view will be removed when the actual view is created.
	// The column properties are unnecessary and not preserved in this temporary view.
	// because other views only need to reference the column name.
	//  Example: SELECT 1 AS colName1, 1 AS colName2.
	// TODO(zp): support SDL for GitOps.
	var selectFileds []*ast.SelectField
	// mysqldump always show field list
	if len(stmt.Cols) > 0 {
		for _, col := range stmt.Cols {
			selectFileds = append(selectFileds, &ast.SelectField{
				Expr: &driver.ValueExpr{
					Datum: types.NewDatum(1),
				},
				AsName: col,
			})
		}
	} else {
		for _, field := range stmt.Select.(*ast.SelectStmt).Fields.Fields {
			var fieldName string
			if field.AsName.O != "" {
				fieldName = field.AsName.O
			} else {
				fieldName = field.Expr.(*ast.ColumnNameExpr).Name.Name.O
			}
			selectFileds = append(selectFileds, &ast.SelectField{
				Expr: &driver.ValueExpr{
					Datum: types.NewDatum(1),
				},
				AsName: model.NewCIStr(fieldName),
			})
		}
	}

	return &ast.CreateViewStmt{
		ViewName: stmt.ViewName,
		Select: &ast.SelectStmt{
			SelectStmtOpts: &ast.SelectStmtOpts{
				// Avoid generating SQL_NO_CACHE
				// https://sourcegraph.com/github.com/pingcap/tidb/-/blob/parser/ast/dml.go?L1234
				SQLCache: true,
			},
			Fields: &ast.FieldList{
				Fields: selectFileds,
			},
		},
		OrReplace: true,
		// Avoid nil pointer dereference panic.
		// https://sourcegraph.com/github.com/pingcap/tidb/-/blob/parser/ast/ddl.go?L1398
		Definer:     stmt.Definer,
		Security:    stmt.Security,
		CheckOption: model.CheckOptionCascaded,
	}
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

// buildColumnPositionMap returns a map of column name to column position.
func buildColumnPositionMap(stmt *ast.CreateTableStmt) map[string]int {
	m := make(map[string]int)
	for i, col := range stmt.Cols {
		m[col.Name.Name.O] = i
	}
	return m
}

// buildIndexMap build a map of index name to constraint on given table name.
func buildIndexMap(stmt *ast.CreateTableStmt) constraintMap {
	indexMap := make(constraintMap)
	for _, constraint := range stmt.Constraints {
		switch constraint.Tp {
		case ast.ConstraintIndex, ast.ConstraintKey, ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex, ast.ConstraintFulltext:
			indexMap[constraint.Name] = constraint
		case ast.ConstraintPrimaryKey:
			// A table can have only one PRIMARY KEY.
			// The name of a PRIMARY KEY is always PRIMARY, which thus cannot be used as the name for any other kind of index.
			// https://dev.mysql.com/doc/refman/8.0/en/create-table.html
			// https://dev.mysql.com/doc/refman/5.7/en/create-table.html
			indexMap["PRIMARY"] = constraint
		}
	}
	return indexMap
}

func buildConstraintMap(stmt *ast.CreateTableStmt) constraintMap {
	constraintMap := make(constraintMap)
	for _, constraint := range stmt.Constraints {
		switch constraint.Tp {
		case ast.ConstraintForeignKey, ast.ConstraintCheck:
			constraintMap[constraint.Name] = constraint
		default:
		}
	}
	return constraintMap
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
		oldOptionStr, err := toString(oldOption)
		if err != nil {
			log.Error("failed to convert old column option to string", zap.Error(err))
			return false
		}
		newOption := newNormalizeOptions[idx]
		newOptionStr, err := toString(newOption)
		if err != nil {
			log.Error("failed to convert new column option to string", zap.Error(err))
			return false
		}
		if oldOptionStr != newOptionStr {
			return false
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
			oldKeyPartStr, err := toString(oldKeyPart.Expr)
			if err != nil {
				log.Error("failed to convert old key part to string", zap.Error(err))
				return false
			}
			newKeyPartStr, err := toString(newKeyPart.Expr)
			if err != nil {
				log.Error("failed to convert new key part to string", zap.Error(err))
				return false
			}
			return oldKeyPartStr == newKeyPartStr
		}
		if oldKeyPart.Desc != newKeyPart.Desc {
			return false
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

// isForeignKeyEqual returns true if two foreign keys are the same.
func isForeignKeyConstraintEqual(old, new *ast.Constraint) bool {
	// FOREIGN KEY [index_name] (index_col_name,...) reference_definition
	if old.Name != new.Name {
		return false
	}
	if !isKeyPartEqual(old.Keys, new.Keys) {
		return false
	}
	if !isReferenceDefinitionEqual(old.Refer, new.Refer) {
		return false
	}
	return true
}

// isReferenceDefinitionEqual returns true if two reference definitions are the same.
func isReferenceDefinitionEqual(old, new *ast.ReferenceDef) bool {
	// reference_definition:
	// 	REFERENCES tbl_name (index_col_name,...)
	//   [MATCH FULL | MATCH PARTIAL | MATCH SIMPLE]
	//   [ON DELETE reference_option]
	//   [ON UPDATE reference_option]
	if old.Table.Name.String() != new.Table.Name.String() {
		return false
	}

	if len(old.IndexPartSpecifications) != len(new.IndexPartSpecifications) {
		return false
	}
	for idx, oldIndexColName := range old.IndexPartSpecifications {
		newIndexColName := new.IndexPartSpecifications[idx]
		if oldIndexColName.Column.Name.String() != newIndexColName.Column.Name.String() {
			return false
		}
	}

	if old.Match != new.Match {
		return false
	}

	if (old.OnDelete == nil) != (new.OnDelete == nil) {
		return false
	}
	if old.OnDelete != nil && new.OnDelete != nil && old.OnDelete.ReferOpt != new.OnDelete.ReferOpt {
		return false
	}

	if (old.OnUpdate == nil) != (new.OnUpdate == nil) {
		return false
	}
	if old.OnUpdate != nil && new.OnUpdate != nil && old.OnUpdate.ReferOpt != new.OnUpdate.ReferOpt {
		return false
	}
	return true
}

// isCheckConstraintEqual returns true if two check constraints are the same.
func isCheckConstraintEqual(old, new *ast.Constraint) bool {
	// check_constraint_definition:
	// 		[CONSTRAINT [symbol]] CHECK (expr) [[NOT] ENFORCED]
	if old.Name != new.Name {
		return false
	}

	oldStr, err := toString(old)
	if err != nil {
		log.Error("failed to convert old check constraint to string", zap.Error(err))
		return false
	}
	newStr, err := toString(new)
	if err != nil {
		log.Error("failed to convert new check constraint to string", zap.Error(err))
		return false
	}
	return oldStr == newStr
}

// diffTableOptions returns the diff of two table options, returns nil if they are the same.
func diffTableOptions(tableName *ast.TableName, old, new []*ast.TableOption) *ast.AlterTableStmt {
	// https://dev.mysql.com/doc/refman/8.0/en/create-table.html
	// table_option: {
	// 	AUTOEXTEND_SIZE [=] value
	//   | AUTO_INCREMENT [=] value
	//   | AVG_ROW_LENGTH [=] value
	//   | [DEFAULT] CHARACTER SET [=] charset_name
	//   | CHECKSUM [=] {0 | 1}
	//   | [DEFAULT] COLLATE [=] collation_name
	//   | COMMENT [=] 'string'
	//   | COMPRESSION [=] {'ZLIB' | 'LZ4' | 'NONE'}
	//   | CONNECTION [=] 'connect_string'
	//   | {DATA | INDEX} DIRECTORY [=] 'absolute path to directory'
	//   | DELAY_KEY_WRITE [=] {0 | 1}
	//   | ENCRYPTION [=] {'Y' | 'N'}
	//   | ENGINE [=] engine_name
	//   | ENGINE_ATTRIBUTE [=] 'string'
	//   | INSERT_METHOD [=] { NO | FIRST | LAST }
	//   | KEY_BLOCK_SIZE [=] value
	//   | MAX_ROWS [=] value
	//   | MIN_ROWS [=] value
	//   | PACK_KEYS [=] {0 | 1 | DEFAULT}
	//   | PASSWORD [=] 'string'
	//   | ROW_FORMAT [=] {DEFAULT | DYNAMIC | FIXED | COMPRESSED | REDUNDANT | COMPACT}
	//   | START TRANSACTION	// This is an internal-use table option.
	//							// It was introduced in MySQL 8.0.21 to permit CREATE TABLE ... SELECT to be logged as a single,
	//							// atomic transaction in the binary log when using row-based replication with a storage engine that supports atomic DDL.
	//   | SECONDARY_ENGINE_ATTRIBUTE [=] 'string'
	//   | STATS_AUTO_RECALC [=] {DEFAULT | 0 | 1}
	//   | STATS_PERSISTENT [=] {DEFAULT | 0 | 1}
	//   | STATS_SAMPLE_PAGES [=] value
	//   | TABLESPACE tablespace_name [STORAGE {DISK | MEMORY}]
	//   | UNION [=] (tbl_name[,tbl_name]...)
	// }

	// We use map to record the table options, so we can easily find the difference.
	oldOptionsMap := buildTableOptionMap(old)
	newOptionsMap := buildTableOptionMap(new)

	var options []*ast.TableOption
	for oldTp, oldOption := range oldOptionsMap {
		newOption, ok := newOptionsMap[oldTp]
		if !ok {
			// We should drop the table option if it doesn't exist in the new table options.
			if astOption := dropTableOption(oldOption); astOption != nil {
				options = append(options, astOption)
			}
			continue
		}
		if !isTableOptionValEqual(oldOption, newOption) {
			options = append(options, newOption)
		}
	}
	// We should add the table option if it doesn't exist in the old table options.
	for newTp, newOption := range newOptionsMap {
		if _, ok := oldOptionsMap[newTp]; !ok {
			// We should add the table option if it doesn't exist in the old table options.
			options = append(options, newOption)
		}
	}
	if len(options) == 0 {
		return nil
	}
	return &ast.AlterTableStmt{
		Table: tableName,
		Specs: []*ast.AlterTableSpec{
			{
				Tp:      ast.AlterTableOption,
				Options: options,
			},
		},
	}
}

// dropTableOption generate the table options node need to oppended to the ALTER TABLE OPTION spec.
func dropTableOption(option *ast.TableOption) *ast.TableOption {
	switch option.Tp {
	case ast.TableOptionAutoIncrement:
		// You cannot reset the counter to a value less than or equal to the value that is currently in use.
		// For both InnoDB and MyISAM, if the value is less than or equal to the maximum value currently in the AUTO_INCREMENT column,
		// the value is reset to the current maximum AUTO_INCREMENT column value plus one.
		// https://dev.mysql.com/doc/refman/8.0/en/alter-table.html
		// So we always set the auto_increment value to 0, it will be reset to the current maximum AUTO_INCREMENT column value plus one.
		return &ast.TableOption{
			Tp:        ast.TableOptionAutoIncrement,
			UintValue: 0,
		}
	case ast.TableOptionAvgRowLength:
		// AVG_ROW_LENGTH only works in MyISAM tables.
		return &ast.TableOption{
			Tp:        ast.TableOptionAvgRowLength,
			UintValue: 0,
		}
	case ast.TableOptionCharset:
		// TODO(zp): we use utf8mb4 as the default charset, but it's not always true. We should consider the database default charset.
		return &ast.TableOption{
			Tp:       ast.TableOptionCharset,
			StrValue: "utf8mb4",
		}
	case ast.TableOptionCollate:
		// TODO(zp): default collate is related with the charset.
		return &ast.TableOption{
			Tp:       ast.TableOptionCollate,
			StrValue: "utf8mb4_general_ci",
		}
	case ast.TableOptionCheckSum:
		return &ast.TableOption{
			Tp:        ast.TableOptionCheckSum,
			UintValue: 0,
		}
	case ast.TableOptionComment:
		// Set to "" to remove the comment.
		return &ast.TableOption{
			Tp: ast.TableOptionComment,
		}
	case ast.TableOptionCompression:
		// TODO(zp): handle the compression
	case ast.TableOptionConnection:
		// Set to "" to remove the connection.
		return &ast.TableOption{
			Tp: ast.TableOptionConnection,
		}
	case ast.TableOptionDataDirectory:
	case ast.TableOptionIndexDirectory:
		// TODO(zp): handle the default data directory and index directory, there are a lot of situations we need to consider.
		// 1. Data Directory and Index Directory will be ignored on Windows.
		// 2. For a normal table, data directory and index directory cannot changed after the table is created, but for a partition table, it can be changed by USING `alter add PARTITIONS data DIRECTORY ...`
		// 3. We should know the default data directory like /var/mysql/data, and the default index directory like /var/mysql/index.
	case ast.TableOptionDelayKeyWrite:
		return &ast.TableOption{
			Tp:        ast.TableOptionDelayKeyWrite,
			UintValue: 0,
		}
	case ast.TableOptionEncryption:
		return &ast.TableOption{
			Tp:       ast.TableOptionEncryption,
			StrValue: "N",
		}
	case ast.TableOptionEngine:
		// TODO(zp): handle the default engine
	case ast.TableOptionInsertMethod:
		// INSERT METHOD only support in MERGE storage engine.
		// https://dev.mysql.com/doc/refman/8.0/en/create-table.html
		return &ast.TableOption{
			Tp:       ast.TableOptionInsertMethod,
			StrValue: "NO",
		}
	case ast.TableOptionKeyBlockSize:
		// TODO(zp): InnoDB doesn't support this. And the default value will be ensured when compiling.
	case ast.TableOptionMaxRows:
		return &ast.TableOption{
			Tp:        ast.TableOptionMaxRows,
			UintValue: 0,
		}
	case ast.TableOptionMinRows:
		return &ast.TableOption{
			Tp:        ast.TableOptionMinRows,
			UintValue: 0,
		}
	case ast.TableOptionPackKeys:
		// TiDB doesn't support this, and the restore will always write "DEFAULT".
		return &ast.TableOption{
			Tp: ast.TableOptionPackKeys,
		}
	case ast.TableOptionPassword:
		// mysqldump will not dump the password, but we handle it to pass the test.
		return &ast.TableOption{
			Tp: ast.TableOptionPassword,
		}
	case ast.TableOptionRowFormat:
		return &ast.TableOption{
			Tp:        ast.TableOptionRowFormat,
			UintValue: ast.RowFormatDefault,
		}
	case ast.TableOptionStatsAutoRecalc:
		return &ast.TableOption{
			Tp:      ast.TableOptionStatsAutoRecalc,
			Default: true,
		}
	case ast.TableOptionStatsPersistent:
		// TiDB doesn't support this, and the restore will always write "DEFAULT".
		return &ast.TableOption{
			Tp: ast.TableOptionStatsPersistent,
		}
	case ast.TableOptionStatsSamplePages:
		return &ast.TableOption{
			Tp:      ast.TableOptionStatsSamplePages,
			Default: true,
		}
	case ast.TableOptionTablespace:
		// TODO(zp): handle the table space
	case ast.TableOptionUnion:
		// TODO(zp): handle the union
	}
	return nil
}

// isTableOptionValEqual compare the two table options value, if they are equal, returns true.
// Caller need to ensure the two table options are not nil and the type is the same.
func isTableOptionValEqual(old, new *ast.TableOption) bool {
	switch old.Tp {
	case ast.TableOptionAutoIncrement:
		// You cannot reset the counter to a value less than or equal to the value that is currently in use.
		// For both InnoDB and MyISAM, if the value is less than or equal to the maximum value currently in the AUTO_INCREMENT column,
		// the value is reset to the current maximum AUTO_INCREMENT column value plus one.
		// https://dev.mysql.com/doc/refman/8.0/en/alter-table.html
		return old.UintValue == new.UintValue
	case ast.TableOptionAvgRowLength:
		return old.UintValue == new.UintValue
	case ast.TableOptionCharset:
		if old.Default != new.Default {
			return false
		}
		if old.Default && new.Default {
			return true
		}
		return old.StrValue == new.StrValue
	case ast.TableOptionCollate:
		return old.StrValue == new.StrValue
	case ast.TableOptionCheckSum:
		return old.UintValue == new.UintValue
	case ast.TableOptionComment:
		return old.StrValue == new.StrValue
	case ast.TableOptionCompression:
		return old.StrValue == new.StrValue
	case ast.TableOptionConnection:
		return old.StrValue == new.StrValue
	case ast.TableOptionDataDirectory:
		return old.StrValue == new.StrValue
	case ast.TableOptionIndexDirectory:
		return old.StrValue == new.StrValue
	case ast.TableOptionDelayKeyWrite:
		return old.UintValue == new.UintValue
	case ast.TableOptionEncryption:
		return old.StrValue == new.StrValue
	case ast.TableOptionEngine:
		return old.StrValue == new.StrValue
	case ast.TableOptionInsertMethod:
		return old.StrValue == new.StrValue
	case ast.TableOptionKeyBlockSize:
		return old.UintValue == new.UintValue
	case ast.TableOptionMaxRows:
		return old.UintValue == new.UintValue
	case ast.TableOptionMinRows:
		return old.UintValue == new.UintValue
	case ast.TableOptionPackKeys:
		// TiDB doesn't support this, and the restore will always write "DEFAULT".
		// The parser will ignore it. So we can only ignore it here.
		// https://github.com/pingcap/tidb/blob/master/parser/parser.y#L11661
		return old.Default == new.Default
	case ast.TableOptionPassword:
		// mysqldump will not dump the password, so we just return false here.
		return false
	case ast.TableOptionRowFormat:
		return old.UintValue == new.UintValue
	case ast.TableOptionStatsAutoRecalc:
		// TiDB parser will ignore the DEFAULT value. So we just can compare the UINT value here.
		// https://github.com/pingcap/tidb/blob/master/parser/parser.y#L11599
		return old.UintValue == new.UintValue
	case ast.TableOptionStatsPersistent:
		// TiDB parser doesn't support this, it only assign the type without any value.
		// https://github.com/pingcap/tidb/blob/master/parser/parser.y#L11595
		return true
	case ast.TableOptionStatsSamplePages:
		return old.UintValue == new.UintValue
	case ast.TableOptionTablespace:
		return old.StrValue == new.StrValue
	case ast.TableOptionUnion:
		oldTableNames := old.TableNames
		newTableNames := new.TableNames
		if len(oldTableNames) != len(newTableNames) {
			return false
		}
		for i, oldTableName := range oldTableNames {
			newTableName := newTableNames[i]
			if oldTableName.Name.O != newTableName.Name.O {
				return false
			}
		}
		return true
	}
	return true
}

// isViewEqual checks whether two views with same name are equal.
func isViewEqual(old, new *ast.CreateViewStmt) bool {
	// CREATE
	// 		[OR REPLACE]
	// 		[ALGORITHM = {UNDEFINED | MERGE | TEMPTABLE}]
	// 		[DEFINER = user]
	// 		[SQL SECURITY { DEFINER | INVOKER }]
	// 		VIEW view_name [(column_list)]
	// 		AS select_statement
	// 		[WITH [CASCADED | LOCAL] CHECK OPTION]
	// We can easily replace view statement by using `CREATE OR REPLACE VIEW` statement to replace the old one.
	// So we don't need to compare each part, just compare the restore string.
	oldViewStr, err := toString(old)
	if err != nil {
		log.Error("fail to convert old view to string", zap.Error(err))
		return false
	}
	newViewStr, err := toString(new)
	if err != nil {
		log.Error("fail to convert new view to string", zap.Error(err))
		return false
	}
	return oldViewStr == newViewStr
}

// buildTableOptionMap builds a map of table options.
func buildTableOptionMap(options []*ast.TableOption) map[ast.TableOptionType]*ast.TableOption {
	m := make(map[ast.TableOptionType]*ast.TableOption)
	for _, option := range options {
		m[option.Tp] = option
	}
	return m
}

// hasColumnsIntersection returns true if two column slices have column name intersaction.
func hasColumnsIntersection(a, b []*ast.ColumnDef) bool {
	bMap := make(map[string]bool)
	for _, col := range b {
		bMap[col.Name.Name.O] = true
	}
	for _, col := range a {
		if _, ok := bMap[col.Name.Name.O]; ok {
			return true
		}
	}
	return false
}
