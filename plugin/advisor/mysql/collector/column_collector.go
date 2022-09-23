package collector

import (
	"fmt"
	"sort"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
)

type Column interface {
	Table() string
	Name() string
	Comment() string
	Line() int
}

type ColumnCollector interface {
	Collect([]ast.StmtNode)
	GenerateAdvice(ColumnChecker) []advisor.Advice
}

func NewColumnCollector(ctx CollectionContext, finder *catalog.Finder) ColumnCollector {
	if finder.Exists() {
		return &catalogColumnCollector{
			ctx:    ctx,
			finder: finder,
			set:    make(catalogColumnKeyMap),
		}
	}
	return &stmtColumnCollector{ctx: ctx}
}

// define stmtColumnCollector

type stmtColumnCollector struct {
	ctx        CollectionContext
	columnList []*stmtColumn
	adviceList []advisor.Advice
}

func (c *stmtColumnCollector) Collect(nodeList []ast.StmtNode) {
	for _, node := range nodeList {
		c.collectNode(node)
	}
}

func (c *stmtColumnCollector) getComent(column *ast.ColumnDef, text string) string {
	for _, option := range column.Options {
		if option.Tp == ast.ColumnOptionComment {
			comment, err := restoreNode(option.Expr, format.RestoreStringWithoutCharset)
			if err != nil {
				comment = ""
				c.adviceList = append(c.adviceList, advisor.Advice{
					Status:  c.ctx.Level,
					Code:    advisor.Internal,
					Title:   "Internal error for parsing column comment",
					Content: fmt.Sprintf("\"%s\" meet internal error %s", text, err),
				})
			}
			return comment
		}
	}
	return ""
}

func (c *stmtColumnCollector) collectNode(in ast.Node) {
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		if c.ctx.Flag.hasFlag(CollectionFlagCreateTable) {
			for _, column := range node.Cols {
				c.columnList = append(c.columnList, &stmtColumn{
					table:   node.Table.Name.O,
					name:    column.Name.Name.O,
					comment: c.getComent(column, in.Text()),
					line:    column.OriginTextPosition(),
				})
			}
		}
	// ALTER TABLE
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			switch spec.Tp {
			// ADD COLUMNS
			case ast.AlterTableAddColumns:
				if c.ctx.Flag.hasFlag(CollectionFlagAddColumn) {
					for _, column := range spec.NewColumns {
						c.columnList = append(c.columnList, &stmtColumn{
							table:   node.Table.Name.O,
							name:    column.Name.Name.O,
							comment: c.getComent(column, in.Text()),
							line:    node.OriginTextPosition(),
						})
					}
				}
			// CHANGE COLUMN
			case ast.AlterTableChangeColumn:
				if c.ctx.Flag.hasFlag(CollectionFlagChangeColumn) {
					c.columnList = append(c.columnList, &stmtColumn{
						table:   node.Table.Name.O,
						name:    spec.NewColumns[0].Name.Name.O,
						comment: c.getComent(spec.NewColumns[0], in.Text()),
						line:    node.OriginTextPosition(),
					})
				}
			// MODIFY COLUMN
			case ast.AlterTableModifyColumn:
				if c.ctx.Flag.hasFlag(CollectionFlagModifyColumn) {
					c.columnList = append(c.columnList, &stmtColumn{
						table:   node.Table.Name.O,
						name:    spec.NewColumns[0].Name.Name.O,
						comment: c.getComent(spec.NewColumns[0], in.Text()),
						line:    node.OriginTextPosition(),
					})
				}
			// ALTER COLUMN
			case ast.AlterTableAlterColumn:
				if c.ctx.Flag.hasFlag(CollectionFlagAlterColumn) {
					c.columnList = append(c.columnList, &stmtColumn{
						table:   node.Table.Name.O,
						name:    spec.NewColumns[0].Name.Name.O,
						comment: c.getComent(spec.NewColumns[0], in.Text()),
						line:    node.OriginTextPosition(),
					})
				}
			// RENAME COLUMN
			case ast.AlterTableRenameColumn:
				if c.ctx.Flag.hasFlag(CollectionFlagRenameColumn) {
					c.columnList = append(c.columnList, &stmtColumn{
						table:   node.Table.Name.O,
						name:    spec.NewColumnName.Name.O,
						comment: "",
						line:    node.OriginTextPosition(),
					})
				}
			}
		}
	}
}

func (c *stmtColumnCollector) GenerateAdvice(f ColumnChecker) []advisor.Advice {
	for _, column := range c.columnList {
		c.adviceList = append(c.adviceList, f(column)...)
	}
	return c.adviceList
}

type stmtColumn struct {
	table   string
	name    string
	comment string
	line    int
}

func (col *stmtColumn) Table() string {
	return col.table
}

func (col *stmtColumn) Name() string {
	return col.name
}

func (col *stmtColumn) Comment() string {
	return col.comment
}

func (col *stmtColumn) Line() int {
	return col.line
}

// define catalogColumnCollector

type ColumnChecker func(Column) []advisor.Advice

type catalogColumn struct {
	table  string
	column *catalog.Column
	line   int
}

func (col *catalogColumn) Table() string {
	return col.table
}

func (col *catalogColumn) Name() string {
	return col.column.Name
}

func (col *catalogColumn) Comment() string {
	return col.column.Comment
}

func (col *catalogColumn) Line() int {
	return col.line
}

type catalogColumnKey struct {
	id         int
	tableName  string
	columnName string
	line       int
}

func (c catalogColumnKey) name() string {
	return fmt.Sprintf("%s.%s", c.tableName, c.columnName)
}

type catalogColumnKeyMap map[string]catalogColumnKey

func (s catalogColumnKeyMap) getList() []catalogColumnKey {
	var columnList []catalogColumnKey
	for _, column := range s {
		columnList = append(columnList, column)
	}
	sort.Slice(columnList, func(i, j int) bool {
		return columnList[i].id < columnList[j].id
	})
	return columnList
}

func (s catalogColumnKeyMap) checkFinalCatalog(finder *catalog.Finder, f ColumnChecker) []advisor.Advice {
	var res []advisor.Advice
	list := s.getList()
	for _, column := range list {
		final := finder.Final.FindColumn(&catalog.ColumnFind{
			TableName:  column.tableName,
			ColumnName: column.columnName,
		})
		if final != nil {
			res = append(res, f(&catalogColumn{
				table:  column.tableName,
				column: final,
				line:   column.line,
			})...)
		}
	}

	if len(res) == 0 {
		res = append(res, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return res
}

func (s catalogColumnKeyMap) set(column catalogColumnKey, replace bool) {
	key := column.name()
	if _, exists := s[key]; !exists || replace {
		column.id = len(s)
		s[key] = column
	}
}

func (s catalogColumnKeyMap) collectColumn(ctx CollectionContext, in ast.Node) {
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		if ctx.Flag.hasFlag(CollectionFlagCreateTable) {
			for _, column := range node.Cols {
				col := catalogColumnKey{
					tableName:  node.Table.Name.O,
					columnName: column.Name.Name.O,
					line:       column.OriginTextPosition(),
				}
				s.set(col, ctx.Replace)
			}
		}
	// ALTER TABLE
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			switch spec.Tp {
			// ADD COLUMNS
			case ast.AlterTableAddColumns:
				if ctx.Flag.hasFlag(CollectionFlagAddColumn) {
					for _, column := range spec.NewColumns {
						col := catalogColumnKey{
							tableName:  node.Table.Name.O,
							columnName: column.Name.Name.O,
							line:       node.OriginTextPosition(),
						}
						s.set(col, ctx.Replace)
					}
				}
			// CHANGE COLUMN
			case ast.AlterTableChangeColumn:
				if ctx.Flag.hasFlag(CollectionFlagChangeColumn) {
					col := catalogColumnKey{
						tableName:  node.Table.Name.O,
						columnName: spec.NewColumns[0].Name.Name.O,
						line:       node.OriginTextPosition(),
					}
					s.set(col, ctx.Replace)
				}
			// MODIFY COLUMN
			case ast.AlterTableModifyColumn:
				if ctx.Flag.hasFlag(CollectionFlagModifyColumn) {
					col := catalogColumnKey{
						tableName:  node.Table.Name.O,
						columnName: spec.NewColumns[0].Name.Name.O,
						line:       node.OriginTextPosition(),
					}
					s.set(col, ctx.Replace)
				}
			// ALTER COLUMN
			case ast.AlterTableAlterColumn:
				if ctx.Flag.hasFlag(CollectionFlagAlterColumn) {
					col := catalogColumnKey{
						tableName:  node.Table.Name.O,
						columnName: spec.NewColumns[0].Name.Name.O,
						line:       node.OriginTextPosition(),
					}
					s.set(col, ctx.Replace)
				}
			// RENAME COLUMN
			case ast.AlterTableRenameColumn:
				if ctx.Flag.hasFlag(CollectionFlagRenameColumn) {
					col := catalogColumnKey{
						tableName:  node.Table.Name.O,
						columnName: spec.NewColumnName.Name.O,
						line:       node.OriginTextPosition(),
					}
					s.set(col, ctx.Replace)
				}
			}
		}
	}
}

type catalogColumnCollector struct {
	ctx    CollectionContext
	finder *catalog.Finder
	set    catalogColumnKeyMap
}

func (c *catalogColumnCollector) Collect(nodeList []ast.StmtNode) {
	for _, node := range nodeList {
		c.set.collectColumn(c.ctx, node)
	}
}

func (c *catalogColumnCollector) GenerateAdvice(f ColumnChecker) []advisor.Advice {
	return c.set.checkFinalCatalog(c.finder, f)
}
