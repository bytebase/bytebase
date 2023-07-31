// Package mysql provides the MySQL differ plugin.
package mysql

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pingcap/tidb/types"
	"github.com/pkg/errors"

	bbparser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/differ"

	// Register pingcap parser driver.
	driver "github.com/pingcap/tidb/types/parser_driver"
)

var (
	_ differ.SchemaDiffer = (*SchemaDiffer)(nil)
)

func init() {
	differ.Register(bbparser.MySQL, &SchemaDiffer{})
	differ.Register(bbparser.TiDB, &SchemaDiffer{})
	differ.Register(bbparser.OceanBase, &SchemaDiffer{})
}

const (
	disableFKCheckStmt string = "SET FOREIGN_KEY_CHECKS=0;\n\n"
	enableFKCheckStmt  string = "SET FOREIGN_KEY_CHECKS=1;\n"
)

// SchemaDiffer it the parser for MySQL dialect.
type SchemaDiffer struct {
}

// diffNode defines different modification types as the safe change order.
// The safe change order means we can change them with no dependency conflicts as this order.
type diffNode struct {
	// Ignore the case sensitive when comparing the table and view names.
	ignoreCaseSensitive bool

	dropUnsupportedStatement   []string
	dropForeignKeyList         []ast.Node
	dropConstraintExceptFkList []ast.Node
	dropIndexList              []ast.Node
	dropViewList               []ast.Node
	dropTableList              []ast.Node

	createTableList                 []ast.Node
	alterTableOptionList            []ast.Node
	addAndModifyColumnList          []ast.Node
	dropColumnList                  []ast.Node
	createTempViewList              []ast.Node
	createIndexList                 []ast.Node
	addConstraintExceptFkList       []ast.Node
	addForeignKeyList               []ast.Node
	createViewList                  []ast.Node
	createUnsupportedStatement      []string
	inPlaceDropUnsupportedStatement []string
	inPlaceAddUnsupportedStatement  []string
}

func (diff *diffNode) diffUnsupportedStatement(oldUnsupportedStmtList, newUnsupportedStmtList []string) error {
	// We compare the CREATE TRIGGER/EVENT/FUNCTION/PROCEDURE statements based on strcmp.
	oldUnsupportMap, err := buildUnsupportObjectMap(oldUnsupportedStmtList)
	if err != nil {
		return err
	}
	newUnsupportMap, err := buildUnsupportObjectMap(newUnsupportedStmtList)
	if err != nil {
		return err
	}
	for tp, objs := range newUnsupportMap {
		for newName, newStmt := range objs {
			if oldStmt, ok := oldUnsupportMap[tp][newName]; ok {
				if strings.Compare(oldStmt, newStmt) != 0 {
					// We should drop the old function and create the new function.
					// https://dev.mysql.com/doc/refman/8.0/en/drop-procedure.html
					// https://dev.mysql.com/doc/refman/5.7/en/drop-procedure.html
					diff.inPlaceDropUnsupportedStatement = append(diff.inPlaceDropUnsupportedStatement, fmt.Sprintf("DROP %s IF EXISTS `%s`;", tp, newName))
					diff.inPlaceAddUnsupportedStatement = append(diff.inPlaceAddUnsupportedStatement, newStmt)
				}
				delete(oldUnsupportMap[tp], newName)
				continue
			}
			// Now, the input of differ comes from the our mysqldump, mysqldump use ;; to separate the CREATE TRIGGER/FUNCTION/PROCEDURE/EVENT statements;
			// So we should append DELIMITER statement to the newStmt.
			delimiterNewStmt := fmt.Sprintf("DELIMITER ;;\n%s\nDELIMITER ;\n", newStmt)
			diff.createUnsupportedStatement = append(diff.createUnsupportedStatement, delimiterNewStmt)
		}
	}
	// drop remaining TiDB unsupported objects
	for tp, objs := range oldUnsupportMap {
		for name := range objs {
			diff.dropUnsupportedStatement = append(diff.dropUnsupportedStatement, fmt.Sprintf("DROP %s IF EXISTS `%s`;", tp, name))
		}
	}

	return nil
}

func (diff *diffNode) diffSupportedStatement(oldStatement, newStatement string) error {
	oldNodeList, _, err := parser.New().Parse(oldStatement, "", "")
	if err != nil {
		return errors.Wrapf(err, "failed to parse old statement %q", oldStatement)
	}
	newNodeList, _, err := parser.New().Parse(newStatement, "", "")
	if err != nil {
		return errors.Wrapf(err, "failed to parse new statement %q", newStatement)
	}

	oldSchemaInfo, err := diff.buildSchemaInfo(oldNodeList)
	if err != nil {
		return err
	}
	newSchemaInfo, err := diff.buildSchemaInfo(newNodeList)
	if err != nil {
		return err
	}

	for tableName, newTable := range newSchemaInfo.tableMap {
		oldTable, exists := oldSchemaInfo.tableMap[tableName]
		if !exists {
			newTable.createTable.IfNotExists = true
			diff.createTableList = append(diff.createTableList, newTable.createTable)
			// Create indexes.
			for _, index := range newTable.indexMap {
				diff.createIndexList = append(diff.createIndexList, index.createIndex)
			}
			continue
		}
		diff.diffTable(oldTable, newTable)
		delete(oldSchemaInfo.tableMap, tableName)
	}

	for _, oldTable := range oldSchemaInfo.tableMap {
		diff.dropTableList = append(diff.dropTableList, &ast.DropTableStmt{
			IfExists: true,
			Tables:   []*ast.TableName{oldTable.createTable.Table},
		})
	}

	var newViewList []*ast.CreateViewStmt
	for _, newNode := range newNodeList {
		if newView, ok := newNode.(*ast.CreateViewStmt); ok {
			newViewList = append(newViewList, newView)
		}
	}

	diff.diffView(oldSchemaInfo.viewMap, newSchemaInfo.viewMap, newViewList)

	return nil
}

func (diff *diffNode) diffView(oldViewMap viewMap, newViewMap viewMap, newViewList []*ast.CreateViewStmt) {
	var tempViewList []ast.Node
	var viewList []ast.Node
	for _, view := range newViewList {
		viewName := view.ViewName.Name.O
		if diff.ignoreCaseSensitive {
			viewName = view.ViewName.Name.L
		}
		if newNode, ok := newViewMap[viewName]; ok {
			if !diff.isViewEqual(view, newNode) {
				// Skip predefined view such as the temporary view from mysqldump.
				continue
			}
		}
		oldNode, ok := oldViewMap[viewName]
		if ok {
			if !diff.isViewEqual(view, oldNode) {
				createViewStmt := *view
				createViewStmt.OrReplace = true
				viewList = append(viewList, &createViewStmt)
			}
			// We should delete the view in the oldViewMap, because we will drop the all views in the oldViewMap explicitly at last.
			delete(oldViewMap, viewName)
		} else {
			// We should create the view.
			// We create the temporary view first and replace it to avoid break the dependency like mysqldump does.
			tempViewStmt := getTempView(view)
			tempViewList = append(tempViewList, tempViewStmt)
			createViewStmt := *view
			createViewStmt.OrReplace = true
			viewList = append(viewList, &createViewStmt)
		}
	}
	diff.createTempViewList = append(diff.createTempViewList, tempViewList...)
	diff.createViewList = append(diff.createViewList, viewList...)

	// Remove the remaining views in the oldViewMap.
	dropViewStmt := &ast.DropTableStmt{
		IsView: true,
	}
	for _, oldView := range oldViewMap {
		dropViewStmt.Tables = append(dropViewStmt.Tables, oldView.ViewName)
	}
	if len(dropViewStmt.Tables) > 0 {
		diff.dropViewList = append(diff.dropViewList, dropViewStmt)
	}
}

func (diff *diffNode) diffTable(oldTable, newTable *tableInfo) {
	diff.diffTableOption(oldTable, newTable)
	diff.diffColumn(oldTable, newTable)
	diff.diffIndex(oldTable, newTable)
	diff.diffConstraint(oldTable, newTable)
}

func (diff *diffNode) diffConstraint(oldTable, newTable *tableInfo) {
	oldPrimaryKey, oldConstraintMap := buildConstraintMap(oldTable.createTable)
	// Compare the create definitions.
	for _, constraint := range newTable.createTable.Constraints {
		switch constraint.Tp {
		case ast.ConstraintPrimaryKey:
			if oldPrimaryKey != nil {
				if !isPrimaryKeyEqual(constraint, oldPrimaryKey) {
					diff.dropConstraintExceptFkList = append(diff.dropConstraintExceptFkList, &ast.AlterTableStmt{
						Table: newTable.createTable.Table,
						Specs: []*ast.AlterTableSpec{
							{
								Tp: ast.AlterTableDropPrimaryKey,
							},
						},
					})
					diff.addConstraintExceptFkList = append(diff.addConstraintExceptFkList, &ast.AlterTableStmt{
						Table: newTable.createTable.Table,
						Specs: []*ast.AlterTableSpec{
							{
								Tp:         ast.AlterTableAddConstraint,
								Constraint: constraint,
							},
						},
					})
				}
				oldPrimaryKey = nil
				continue
			}
			diff.addConstraintExceptFkList = append(diff.addConstraintExceptFkList, &ast.AlterTableStmt{
				Table: newTable.createTable.Table,
				Specs: []*ast.AlterTableSpec{
					{
						Tp:         ast.AlterTableAddConstraint,
						Constraint: constraint,
					},
				},
			})
		case ast.ConstraintForeignKey:
			if oldConstraint, ok := oldConstraintMap[strings.ToLower(constraint.Name)]; ok {
				if !isForeignKeyConstraintEqual(constraint, oldConstraint) {
					diff.dropForeignKeyList = append(diff.dropForeignKeyList, &ast.AlterTableStmt{
						Table: newTable.createTable.Table,
						Specs: []*ast.AlterTableSpec{
							{
								Tp:   ast.AlterTableDropForeignKey,
								Name: constraint.Name,
							},
						},
					})
					diff.addForeignKeyList = append(diff.addForeignKeyList, &ast.AlterTableStmt{
						Table: newTable.createTable.Table,
						Specs: []*ast.AlterTableSpec{
							{
								Tp:         ast.AlterTableAddConstraint,
								Constraint: constraint,
							},
						},
					})
				}
				delete(oldConstraintMap, strings.ToLower(constraint.Name))
				continue
			}
			diff.addForeignKeyList = append(diff.addForeignKeyList, &ast.AlterTableStmt{
				Table: newTable.createTable.Table,
				Specs: []*ast.AlterTableSpec{
					{
						Tp:         ast.AlterTableAddConstraint,
						Constraint: constraint,
					},
				},
			})
		case ast.ConstraintCheck:
			if oldConstraint, ok := oldConstraintMap[strings.ToLower(constraint.Name)]; ok {
				if !isCheckConstraintEqual(constraint, oldConstraint) {
					diff.dropConstraintExceptFkList = append(diff.dropConstraintExceptFkList, &ast.AlterTableStmt{
						Table: newTable.createTable.Table,
						Specs: []*ast.AlterTableSpec{
							{
								Tp:         ast.AlterTableDropCheck,
								Constraint: constraint,
							},
						},
					})
					diff.addConstraintExceptFkList = append(diff.addConstraintExceptFkList, &ast.AlterTableStmt{
						Table: newTable.createTable.Table,
						Specs: []*ast.AlterTableSpec{
							{
								Tp:         ast.AlterTableAddConstraint,
								Constraint: constraint,
							},
						},
					})
				}
				delete(oldConstraintMap, strings.ToLower(constraint.Name))
				continue
			}
			diff.addConstraintExceptFkList = append(diff.addConstraintExceptFkList, &ast.AlterTableStmt{
				Table: newTable.createTable.Table,
				Specs: []*ast.AlterTableSpec{
					{
						Tp:         ast.AlterTableAddConstraint,
						Constraint: constraint,
					},
				},
			})
		}
	}

	if oldPrimaryKey != nil {
		diff.dropConstraintExceptFkList = append(diff.dropConstraintExceptFkList, &ast.AlterTableStmt{
			Table: newTable.createTable.Table,
			Specs: []*ast.AlterTableSpec{
				{
					Tp: ast.AlterTableDropPrimaryKey,
				},
			},
		})
	}

	for _, oldConstraint := range oldConstraintMap {
		switch oldConstraint.Tp {
		case ast.ConstraintCheck:
			diff.dropConstraintExceptFkList = append(diff.dropConstraintExceptFkList, &ast.AlterTableStmt{
				Table: newTable.createTable.Table,
				Specs: []*ast.AlterTableSpec{
					{
						Tp:         ast.AlterTableDropCheck,
						Constraint: oldConstraint,
					},
				},
			})
		case ast.ConstraintForeignKey:
			diff.dropForeignKeyList = append(diff.dropForeignKeyList, &ast.AlterTableStmt{
				Table: newTable.createTable.Table,
				Specs: []*ast.AlterTableSpec{
					{
						Tp:   ast.AlterTableDropForeignKey,
						Name: oldConstraint.Name,
					},
				},
			})
		}
	}
}

func (diff *diffNode) diffIndex(oldTable, newTable *tableInfo) {
	for indexName, newIndex := range newTable.indexMap {
		if oldIndex, ok := oldTable.indexMap[indexName]; ok {
			if !isIndexEqual(newIndex.createIndex, oldIndex.createIndex) {
				diff.dropIndexList = append(diff.dropIndexList, &ast.DropIndexStmt{
					IndexName: oldIndex.createIndex.IndexName,
					Table:     oldIndex.createIndex.Table,
				})
				diff.createIndexList = append(diff.createIndexList, newIndex.createIndex)
			}
			delete(oldTable.indexMap, indexName)
			continue
		}
		diff.createIndexList = append(diff.createIndexList, newIndex.createIndex)
	}

	for _, oldIndex := range oldTable.indexMap {
		diff.dropIndexList = append(diff.dropIndexList, &ast.DropIndexStmt{
			IndexName: oldIndex.createIndex.IndexName,
			Table:     oldIndex.createIndex.Table,
		})
	}
}

func (diff *diffNode) diffColumn(oldTable, newTable *tableInfo) {
	// We use a single ALTER TABLE statement to add and modify columns,
	// because we need to maintain a fixed order of these two operations.
	// This approach ensures that we can manipulate the column position as needed.
	addAndModifyColumnStatement := &ast.AlterTableStmt{Table: newTable.createTable.Table}

	oldColumnMap := buildColumnMap(oldTable.createTable.Cols)
	oldColumnPositionMap := buildColumnPositionMap(oldTable.createTable)
	for idx, columnDef := range newTable.createTable.Cols {
		// Column names are always case insensitive.
		newColumnName := columnDef.Name.Name.L
		oldColumnDef, ok := oldColumnMap[newColumnName]
		if !ok {
			columnPosition := &ast.ColumnPosition{Tp: ast.ColumnPositionFirst}
			if idx >= 1 {
				columnPosition.Tp = ast.ColumnPositionAfter
				columnPosition.RelativeColumn = &ast.ColumnName{Name: model.NewCIStr(newTable.createTable.Cols[idx-1].Name.Name.O)}
			}
			addAndModifyColumnStatement.Specs = append(addAndModifyColumnStatement.Specs, &ast.AlterTableSpec{
				Tp:         ast.AlterTableAddColumns,
				NewColumns: []*ast.ColumnDef{columnDef},
				Position:   columnPosition,
			})
			continue
		}

		// Compare the column positions.
		columnPosition := &ast.ColumnPosition{Tp: ast.ColumnPositionNone}
		columnPosInOld := oldColumnPositionMap[newColumnName]
		if hasColumnsIntersection(oldTable.createTable.Cols[:columnPosInOld], newTable.createTable.Cols[idx+1:]) {
			if idx == 0 {
				columnPosition.Tp = ast.ColumnPositionFirst
			} else {
				columnPosition.Tp = ast.ColumnPositionAfter
				columnPosition.RelativeColumn = &ast.ColumnName{Name: model.NewCIStr(newTable.createTable.Cols[idx-1].Name.Name.O)}
			}
		}
		// Compare the column definitions.
		if !isColumnEqual(oldColumnDef, columnDef) || columnPosition.Tp != ast.ColumnPositionNone {
			addAndModifyColumnStatement.Specs = append(addAndModifyColumnStatement.Specs, &ast.AlterTableSpec{
				Tp:         ast.AlterTableModifyColumn,
				NewColumns: []*ast.ColumnDef{columnDef},
				Position:   columnPosition,
			})
		}
		delete(oldColumnMap, newColumnName)
	}
	// TODO(zp): add an option to control whether to drop the excess columns.
	for _, columnDef := range oldColumnMap {
		diff.dropColumnList = append(diff.dropColumnList, &ast.AlterTableStmt{
			Table: newTable.createTable.Table,
			Specs: []*ast.AlterTableSpec{
				{
					Tp: ast.AlterTableDropColumn,
					OldColumnName: &ast.ColumnName{
						Name: model.NewCIStr(columnDef.Name.Name.O),
					},
				},
			},
		})
	}

	if len(addAndModifyColumnStatement.Specs) > 0 {
		diff.addAndModifyColumnList = append(diff.addAndModifyColumnList, addAndModifyColumnStatement)
	}
}

func (diff *diffNode) diffTableOption(oldTable, newTable *tableInfo) {
	if alterTableOptionStmt := diffTableOptions(newTable.createTable.Table, oldTable.createTable.Options, newTable.createTable.Options); alterTableOptionStmt != nil {
		diff.alterTableOptionList = append(diff.alterTableOptionList, alterTableOptionStmt)
	}
}

func (diff *diffNode) deparse() (string, error) {
	var buf bytes.Buffer
	flag := format.DefaultRestoreFlags | format.RestoreStringWithoutCharset | format.RestorePrettyFormat

	sort.Strings(diff.dropUnsupportedStatement)
	for _, statement := range diff.dropUnsupportedStatement {
		if _, err := buf.WriteString(statement); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("\n\n"); err != nil {
			return "", err
		}
	}
	if err := sortAndWriteNodeList(&buf, diff.dropForeignKeyList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.dropConstraintExceptFkList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.dropIndexList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.dropViewList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.dropTableList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.createTableList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.alterTableOptionList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.addAndModifyColumnList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.dropColumnList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.createTempViewList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.createIndexList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.addConstraintExceptFkList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.addForeignKeyList, flag); err != nil {
		return "", err
	}
	if err := sortAndWriteNodeList(&buf, diff.createViewList, flag); err != nil {
		return "", err
	}

	sort.Strings(diff.createUnsupportedStatement)
	for _, statement := range diff.createUnsupportedStatement {
		if _, err := buf.WriteString(statement); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("\n\n"); err != nil {
			return "", err
		}
	}

	sort.Strings(diff.inPlaceDropUnsupportedStatement)
	for _, statement := range diff.inPlaceDropUnsupportedStatement {
		if _, err := buf.WriteString(statement); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("\n\n"); err != nil {
			return "", err
		}
	}

	sort.Strings(diff.inPlaceAddUnsupportedStatement)
	for _, statement := range diff.inPlaceAddUnsupportedStatement {
		if _, err := buf.WriteString(statement); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("\n\n"); err != nil {
			return "", err
		}
	}

	text := buf.String()
	if len(text) > 0 {
		return fmt.Sprintf("%s%s%s", disableFKCheckStmt, buf.String(), enableFKCheckStmt), nil
	}
	return "", nil
}

// constraintMap returns a map of constraint name to constraint.
type constraintMap map[string]*ast.Constraint

// SchemaDiff returns the schema diff.
// It only supports schema information from mysqldump.
func (*SchemaDiffer) SchemaDiff(oldStmt, newStmt string, ignoreCaseSensitive bool) (string, error) {
	// 1. Preprocessing Stage.
	// TiDB parser doesn't support some statements like `CREATE EVENT`, so we need to extract them out and diff them based on string compare.
	oldUnsupportedStmtList, oldSupportedStmt, err := classifyStatement(oldStmt)
	if err != nil {
		return "", err
	}
	newUnsupportedStmtList, newSupportedStmt, err := classifyStatement(newStmt)
	if err != nil {
		return "", err
	}

	diff := &diffNode{
		ignoreCaseSensitive: ignoreCaseSensitive,
	}
	if err := diff.diffSupportedStatement(oldSupportedStmt, newSupportedStmt); err != nil {
		return "", err
	}
	if err := diff.diffUnsupportedStatement(oldUnsupportedStmtList, newUnsupportedStmtList); err != nil {
		return "", err
	}

	return diff.deparse()
}

func classifyStatement(statement string) ([]string, string, error) {
	unsupported, supported, err := bbparser.ExtractTiDBUnsupportStmts(statement)
	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to extract TiDB unsupported statements from statements %q", statement)
	}
	var afterFilter []string
	for _, stmt := range unsupported {
		if !bbparser.IsDelimiter(stmt) {
			afterFilter = append(afterFilter, stmt)
		}
	}
	return afterFilter, supported, nil
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

func getID(node ast.Node) string {
	switch in := node.(type) {
	case *ast.CreateTableStmt:
		return in.Table.Name.String()
	case *ast.DropTableStmt:
		return in.Tables[0].Name.String()
	case *ast.AlterTableStmt:
		for _, spec := range in.Specs {
			switch spec.Tp {
			case ast.AlterTableOption:
				return in.Table.Name.String()
			case ast.AlterTableAddColumns:
				return fmt.Sprintf("%s.%s", in.Table.Name.String(), spec.NewColumns[0].Name)
			case ast.AlterTableDropColumn:
				return fmt.Sprintf("%s.%s", in.Table.Name.String(), spec.OldColumnName.Name.String())
			case ast.AlterTableModifyColumn:
				return fmt.Sprintf("%s.%s", in.Table.Name.String(), spec.NewColumns[0].Name)
			case ast.AlterTableAddConstraint:
				return fmt.Sprintf("%s.%s", in.Table.Name.String(), spec.Constraint.Name)
			case ast.AlterTableDropForeignKey:
				return fmt.Sprintf("%s.%s", in.Table.Name.String(), spec.Name)
			case ast.AlterTableDropPrimaryKey:
				return in.Table.Name.String()
			case ast.AlterTableDropCheck:
				return fmt.Sprintf("%s.%s", in.Table.Name.String(), spec.Constraint.Name)
			}
		}
	case *ast.CreateIndexStmt:
		return fmt.Sprintf("%s.%s", in.Table.Name.String(), in.IndexName)
	case *ast.DropIndexStmt:
		return fmt.Sprintf("%s.%s", in.Table.Name.String(), in.IndexName)
	case *ast.CreateViewStmt:
		return in.ViewName.Name.String()
	}
	return ""
}

func sortAndWriteNodeList(w io.Writer, ns []ast.Node, flags format.RestoreFlags) error {
	sort.Slice(ns, func(i, j int) bool {
		return getID(ns[i]) < getID(ns[j])
	})

	for _, n := range ns {
		if err := writeNodeStatement(w, n, flags); err != nil {
			return err
		}
	}
	return nil
}

type schemaInfo struct {
	tableMap tableMap
	viewMap  viewMap
}

type viewMap map[string]*ast.CreateViewStmt

type indexInfo struct {
	createIndex *ast.CreateIndexStmt
}
type indexMap map[string]*indexInfo

func newIndexInfo(createIndex *ast.CreateIndexStmt) *indexInfo {
	return &indexInfo{
		createIndex: createIndex,
	}
}

type tableInfo struct {
	createTable *ast.CreateTableStmt
	indexMap    indexMap
}
type tableMap map[string]*tableInfo

func newTableInfo(createTable *ast.CreateTableStmt) (*tableInfo, error) {
	result := &tableInfo{
		createTable: createTable,
		indexMap:    make(indexMap),
	}

	var newConstraintList []*ast.Constraint
	for _, constraint := range createTable.Constraints {
		if createIndex := transformConstraintToIndex(createTable.Table, constraint); createIndex != nil {
			if _, exists := result.indexMap[strings.ToLower(createIndex.IndexName)]; exists {
				return nil, errors.Errorf("Try to create index `%s` on table `%s`, but index already exists", createIndex.IndexName, createIndex.Table.Name.String())
			}
			result.indexMap[strings.ToLower(constraint.Name)] = newIndexInfo(createIndex)
		} else {
			newConstraintList = append(newConstraintList, constraint)
		}
	}
	createTable.Constraints = newConstraintList

	return result, nil
}

func transformConstraintToIndex(tableName *ast.TableName, constraint *ast.Constraint) *ast.CreateIndexStmt {
	indexType := ast.IndexKeyTypeNone
	switch constraint.Tp {
	case ast.ConstraintKey, ast.ConstraintIndex:
	case ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex:
		indexType = ast.IndexKeyTypeUnique
	case ast.ConstraintFulltext:
		indexType = ast.IndexKeyTypeFullText
	default:
		return nil
	}
	result := &ast.CreateIndexStmt{
		IndexName:               constraint.Name,
		Table:                   tableName,
		IndexPartSpecifications: constraint.Keys,
		IndexOption:             constraint.Option,
		KeyType:                 indexType,
	}
	if result.IndexOption == nil {
		result.IndexOption = &ast.IndexOption{Tp: model.IndexTypeInvalid}
	}
	return result
}

// buildSchemaInfo returns schema information built by statements.
func (diff *diffNode) buildSchemaInfo(nodes []ast.StmtNode) (*schemaInfo, error) {
	result := &schemaInfo{
		tableMap: make(tableMap),
		viewMap:  make(viewMap),
	}
	for _, node := range nodes {
		switch stmt := node.(type) {
		case *ast.CreateTableStmt:
			var err error
			tableName := stmt.Table.Name.O
			if diff.ignoreCaseSensitive {
				tableName = stmt.Table.Name.L
			}
			if result.tableMap[tableName], err = newTableInfo(stmt); err != nil {
				return nil, err
			}
		case *ast.CreateIndexStmt:
			tableName := stmt.Table.Name.O
			if diff.ignoreCaseSensitive {
				tableName = stmt.Table.Name.L
			}
			table, exists := result.tableMap[tableName]
			if !exists {
				return nil, errors.Errorf("Try to create index `%s` on table `%s`, but table not found", stmt.IndexName, stmt.Table.Name.String())
			}
			// Index names are always case insensitive
			if _, exists := table.indexMap[strings.ToLower(stmt.IndexName)]; exists {
				return nil, errors.Errorf("Try to create index `%s` on table `%s`, but index already exists", stmt.IndexName, stmt.Table.Name.String())
			}
			table.indexMap[strings.ToLower(stmt.IndexName)] = newIndexInfo(stmt)
		case *ast.CreateViewStmt:
			viewName := stmt.ViewName.Name.O
			if diff.ignoreCaseSensitive {
				viewName = stmt.ViewName.Name.L
			}
			result.viewMap[viewName] = stmt
		default:
		}
	}
	return result, nil
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
func buildColumnMap(columnList []*ast.ColumnDef) map[string]*ast.ColumnDef {
	oldColumnMap := make(map[string]*ast.ColumnDef)
	for _, columnDef := range columnList {
		// Column names are always case insensitive.
		oldColumnMap[columnDef.Name.Name.L] = columnDef
	}
	return oldColumnMap
}

// buildColumnPositionMap returns a map of column name to column position.
func buildColumnPositionMap(stmt *ast.CreateTableStmt) map[string]int {
	m := make(map[string]int)
	for i, col := range stmt.Cols {
		// Column names are always case insensitive.
		m[col.Name.Name.L] = i
	}
	return m
}

func buildConstraintMap(stmt *ast.CreateTableStmt) (*ast.Constraint, constraintMap) {
	var primaryKey *ast.Constraint
	constraintMap := make(constraintMap)
	for _, constraint := range stmt.Constraints {
		switch constraint.Tp {
		case ast.ConstraintForeignKey, ast.ConstraintCheck:
			constraintMap[strings.ToLower(constraint.Name)] = constraint
		case ast.ConstraintPrimaryKey:
			primaryKey = constraint
		default:
		}
	}
	return primaryKey, constraintMap
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
	oldCollate, oldNormalizeOptions := normalizeColumnOptions(old)
	newCollate, newNormalizeOptions := normalizeColumnOptions(new)
	if len(oldNormalizeOptions) != len(newNormalizeOptions) {
		return false
	}
	if newCollate != nil {
		if oldCollate == nil {
			return false
		}
		if oldCollate.StrValue != newCollate.StrValue {
			return false
		}
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
func normalizeColumnOptions(options []*ast.ColumnOption) (*ast.ColumnOption, []*ast.ColumnOption) {
	var retOptions []*ast.ColumnOption
	var collateOption *ast.ColumnOption
	for _, option := range options {
		switch option.Tp {
		case ast.ColumnOptionCollate:
			collateOption = option
			continue
		case ast.ColumnOptionNull:
			continue
		case ast.ColumnOptionNoOption:
			continue
		case ast.ColumnOptionDefaultValue:
			if option.Expr.GetType().GetType() == mysql.TypeNull {
				continue
			}
		}
		retOptions = append(retOptions, option)
	}
	sort.Slice(retOptions, func(i, j int) bool {
		return retOptions[i].Tp < retOptions[j].Tp
	})
	return collateOption, retOptions
}

// isIndexEqual returns true if definitions of two indexes are the same.
func isIndexEqual(old, new *ast.CreateIndexStmt) bool {
	// CREATE [UNIQUE | FULLTEXT | SPATIAL] INDEX index_name
	// [index_type]
	// ON tbl_name (key_part,...)
	// [index_option]
	// [algorithm_option | lock_option] ...

	// MySQL index names are case insensitive.
	if !strings.EqualFold(old.IndexName, new.IndexName) {
		return false
	}
	if (old.IndexOption == nil) != (new.IndexOption == nil) {
		return false
	}
	if old.IndexOption != nil && new.IndexOption != nil {
		if old.IndexOption.Tp != new.IndexOption.Tp {
			return false
		}
	}

	if !isKeyPartEqual(old.IndexPartSpecifications, new.IndexPartSpecifications) {
		return false
	}
	if !isIndexOptionEqual(old.IndexOption, new.IndexOption) {
		return false
	}
	return true
}

// isPrimaryKeyEqual returns true if definitions of two indexes are the same.
func isPrimaryKeyEqual(old, new *ast.Constraint) bool {
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
	if !strings.EqualFold(old.Name, new.Name) {
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

// trimParentheses trims outer parentheses.
func trimParentheses(expr ast.ExprNode) ast.ExprNode {
	result := expr
	for {
		if node, yes := result.(*ast.ParenthesesExpr); yes {
			result = node.Expr
		} else {
			break
		}
	}
	return result
}

// isCheckConstraintEqual returns true if two check constraints are the same.
func isCheckConstraintEqual(old, new *ast.Constraint) bool {
	// check_constraint_definition:
	// 		[CONSTRAINT [symbol]] CHECK (expr) [[NOT] ENFORCED]
	if !strings.EqualFold(old.Name, new.Name) {
		return false
	}

	if old.Enforced != new.Enforced {
		return false
	}

	oldExpr, err := toString(trimParentheses(old.Expr))
	if err != nil {
		log.Error("failed to convert old check constraint expression to string", zap.Error(err))
		return false
	}
	newExpr, err := toString(trimParentheses(new.Expr))
	if err != nil {
		log.Error("failed to convert new check constraint expression to string", zap.Error(err))
		return false
	}
	return oldExpr == newExpr
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
			switch oldOption.Tp {
			// For table engine, table charset and table collation, if oldTable has but newTable doesn't,
			// we skip drop them.
			case ast.TableOptionEngine, ast.TableOptionCharset, ast.TableOptionCollate:
				continue
			}
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
		defaultCharset := "utf8mb4"
		if option.StrValue == defaultCharset {
			return nil
		}
		return &ast.TableOption{
			Tp:       ast.TableOptionCharset,
			StrValue: defaultCharset,
		}
	case ast.TableOptionCollate:
		// TODO(zp): default collate is related with the charset.
		defaultCollate := "utf8mb4_general_ci"
		if option.StrValue == defaultCollate {
			return nil
		}
		return &ast.TableOption{
			Tp:       ast.TableOptionCollate,
			StrValue: defaultCollate,
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
func (diff *diffNode) isViewEqual(old, new *ast.CreateViewStmt) bool {
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
	if diff.ignoreCaseSensitive {
		oldViewStr, err := toLowerNameString(old)
		if err != nil {
			log.Error("fail to convert old view to string", zap.Error(err))
			return false
		}
		newViewStr, err := toLowerNameString(new)
		if err != nil {
			log.Error("fail to convert new view to string", zap.Error(err))
			return false
		}
		return oldViewStr == newViewStr
	}
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

// hasColumnsIntersection returns true if two column slices have column name intersection.
func hasColumnsIntersection(a, b []*ast.ColumnDef) bool {
	bMap := make(map[string]bool)
	for _, col := range b {
		// MySQL column name is case insensitive.
		bMap[col.Name.Name.L] = true
	}
	for _, col := range a {
		// MySQL column name is case insensitive.
		if _, ok := bMap[col.Name.Name.L]; ok {
			return true
		}
	}
	return false
}
