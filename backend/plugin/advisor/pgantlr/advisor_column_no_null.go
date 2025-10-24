package pgantlr

import (
	"context"
	"fmt"
	"slices"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleColumnNotNull, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &columnNoNullChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		catalog:                      checkCtx.Catalog,
		nullableColumns:              make(columnMap),
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.generateAdviceList(), nil
}

type columnName struct {
	schema string
	table  string
	column string
}

func (c columnName) normalizeTableName() string {
	if c.schema == "" || c.schema == "public" {
		return fmt.Sprintf("%q.%q", "public", c.table)
	}
	return fmt.Sprintf("%q.%q", c.schema, c.table)
}

type columnMap map[columnName]int

type columnNoNullChecker struct {
	*parser.BasePostgreSQLParserListener

	level           storepb.Advice_Status
	title           string
	catalog         *catalog.Finder
	nullableColumns columnMap
}

func (c *columnNoNullChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	tableName := c.extractTableName(ctx.AllQualified_name())
	if tableName == "" {
		return
	}

	// Track all columns and their line numbers
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			// Column definition
			if elem.ColumnDef() != nil {
				colDef := elem.ColumnDef()
				if colDef.Colid() != nil {
					columnName := normalizeColid(colDef.Colid())
					// Add column as nullable by default
					c.addColumn("public", tableName, columnName, elem.GetStart().GetLine())

					// Check column constraints for NOT NULL or PRIMARY KEY
					c.removeColumnByColConstraints("public", tableName, colDef)
				}
			}

			// Table constraint (like PRIMARY KEY (id))
			if elem.Tableconstraint() != nil {
				c.removeColumnByTableConstraint("public", tableName, elem.Tableconstraint())
			}
		}
	}
}

func (c *columnNoNullChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	tableName := ctx.Relation_expr().Qualified_name().GetText()

	// Check ALTER TABLE commands
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// ADD COLUMN
			if cmd.ADD_P() != nil && cmd.ColumnDef() != nil {
				colDef := cmd.ColumnDef()
				if colDef.Colid() != nil {
					columnName := normalizeColid(colDef.Colid())
					c.addColumn("public", tableName, columnName, ctx.GetStart().GetLine())
					c.removeColumnByColConstraints("public", tableName, colDef)
				}
			}

			// ALTER COLUMN SET NOT NULL
			if cmd.ALTER() != nil && cmd.SET() != nil && cmd.NOT() != nil && cmd.NULL_P() != nil {
				allColids := cmd.AllColid()
				if len(allColids) > 0 {
					columnName := normalizeColid(allColids[0])
					c.removeColumn("public", tableName, columnName)
				}
			}

			// ALTER COLUMN DROP NOT NULL
			if cmd.ALTER() != nil && cmd.DROP() != nil && cmd.NOT() != nil && cmd.NULL_P() != nil {
				allColids := cmd.AllColid()
				if len(allColids) > 0 {
					columnName := normalizeColid(allColids[0])
					c.addColumn("public", tableName, columnName, ctx.GetStart().GetLine())
				}
			}

			// ADD table constraint
			if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
				c.removeColumnByTableConstraint("public", tableName, cmd.Tableconstraint())
			}
		}
	}
}

func (*columnNoNullChecker) extractTableName(qualifiedNames []parser.IQualified_nameContext) string {
	if len(qualifiedNames) == 0 {
		return ""
	}

	text := qualifiedNames[0].GetText()
	parts := splitIdentifier(text)
	if len(parts) == 0 {
		return ""
	}

	return parts[len(parts)-1]
}

func (c *columnNoNullChecker) addColumn(schema, table, column string, line int) {
	if schema == "" {
		schema = "public"
	}
	c.nullableColumns[columnName{schema: schema, table: table, column: column}] = line
}

func (c *columnNoNullChecker) removeColumn(schema, table, column string) {
	if schema == "" {
		schema = "public"
	}
	delete(c.nullableColumns, columnName{schema: schema, table: table, column: column})
}

func (c *columnNoNullChecker) removeColumnByColConstraints(schema, table string, colDef parser.IColumnDefContext) {
	if colDef.Colquallist() == nil {
		return
	}

	columnName := normalizeColid(colDef.Colid())
	allConstraints := colDef.Colquallist().AllColconstraint()
	for _, constraint := range allConstraints {
		if constraint.Colconstraintelem() == nil {
			continue
		}

		elem := constraint.Colconstraintelem()

		// NOT NULL constraint
		if elem.NOT() != nil && elem.NULL_P() != nil {
			c.removeColumn(schema, table, columnName)
			return
		}

		// PRIMARY KEY constraint
		if elem.PRIMARY() != nil && elem.KEY() != nil {
			c.removeColumn(schema, table, columnName)
			return
		}
	}
}

func (c *columnNoNullChecker) removeColumnByTableConstraint(schema, table string, constraint parser.ITableconstraintContext) {
	if constraint.Constraintelem() == nil {
		return
	}

	elem := constraint.Constraintelem()

	// PRIMARY KEY (col1, col2, ...)
	if elem.PRIMARY() != nil && elem.KEY() != nil && elem.Columnlist() != nil {
		allColumnElems := elem.Columnlist().AllColumnElem()
		for _, columnElem := range allColumnElems {
			if columnElem.Colid() != nil {
				c.removeColumn(schema, table, normalizeColid(columnElem.Colid()))
			}
		}
		return
	}

	// PRIMARY KEY USING INDEX
	if elem.PRIMARY() != nil && elem.KEY() != nil && elem.Existingindex() != nil {
		existingIndex := elem.Existingindex()
		if existingIndex.Name() != nil {
			indexName := normalizeName(existingIndex.Name())
			// Try to find index in catalog
			if c.catalog != nil {
				_, index := c.catalog.Origin.FindIndex(&catalog.IndexFind{
					SchemaName: schema,
					TableName:  table,
					IndexName:  indexName,
				})
				if index != nil {
					for _, expression := range index.ExpressionList() {
						c.removeColumn(schema, table, expression)
					}
				}
			}
		}
	}
}

func (c *columnNoNullChecker) generateAdviceList() []*storepb.Advice {
	var adviceList []*storepb.Advice
	var columnList []columnName

	for column := range c.nullableColumns {
		columnList = append(columnList, column)
	}

	if len(columnList) > 0 {
		// Order it cause the random iteration order in Go
		slices.SortFunc(columnList, func(i, j columnName) int {
			if i.schema != j.schema {
				if i.schema < j.schema {
					return -1
				}
				return 1
			}
			if i.table != j.table {
				if i.table < j.table {
					return -1
				}
				return 1
			}
			if i.column < j.column {
				return -1
			}
			if i.column > j.column {
				return 1
			}
			return 0
		})
	}

	for _, column := range columnList {
		adviceList = append(adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.ColumnCannotNull.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("Column %q in %s cannot have NULL value", column.column, column.normalizeTableName()),
			StartPosition: &storepb.Position{
				Line:   int32(c.nullableColumns[column]),
				Column: 0,
			},
		})
	}

	return adviceList
}
