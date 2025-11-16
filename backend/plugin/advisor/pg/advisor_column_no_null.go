package pg

import (
	"context"
	"fmt"
	"slices"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
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

	rule := &columnNoNullRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		originCatalog:   checkCtx.OriginCatalog,
		nullableColumns: make(columnMap),
	}

	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
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

type columnNoNullRule struct {
	BaseRule

	originCatalog   *catalog.DatabaseState
	nullableColumns columnMap
}

func (*columnNoNullRule) Name() string {
	return "CreatestmtAltertablestmt"
}

func (r *columnNoNullRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		if createStmt, ok := ctx.(*parser.CreatestmtContext); ok {
			r.handleCreatestmt(createStmt)
		}
	case "Altertablestmt":
		if alterStmt, ok := ctx.(*parser.AltertablestmtContext); ok {
			r.handleAltertablestmt(alterStmt)
		}
	default:
		// Do nothing for other node types
	}
	return nil
}

func (r *columnNoNullRule) OnExit(ctx antlr.ParserRuleContext, _ string) error {
	// Generate advice when we exit the root node
	if _, ok := ctx.(*parser.RootContext); ok {
		var columnList []columnName
		for column := range r.nullableColumns {
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
			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    code.ColumnCannotNull.Int32(),
				Title:   r.title,
				Content: fmt.Sprintf("Column %q in %s cannot have NULL value", column.column, column.normalizeTableName()),
				StartPosition: &storepb.Position{
					Line:   int32(r.nullableColumns[column]),
					Column: 0,
				},
			})
		}
	}
	return nil
}

func (r *columnNoNullRule) handleCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	tableName := r.extractTableName(ctx.AllQualified_name())
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
					columnName := pg.NormalizePostgreSQLColid(colDef.Colid())
					// Add column as nullable by default
					r.addColumn("public", tableName, columnName, colDef.GetStart().GetLine())

					// Check column constraints for NOT NULL or PRIMARY KEY
					r.removeColumnByColConstraints("public", tableName, colDef)
				}
			}

			// Table constraint (like PRIMARY KEY (id))
			if elem.Tableconstraint() != nil {
				r.removeColumnByTableConstraint("public", tableName, elem.Tableconstraint())
			}
		}
	}
}

func (r *columnNoNullRule) handleAltertablestmt(ctx *parser.AltertablestmtContext) {
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
					columnName := pg.NormalizePostgreSQLColid(colDef.Colid())
					r.addColumn("public", tableName, columnName, colDef.GetStart().GetLine())
					r.removeColumnByColConstraints("public", tableName, colDef)
				}
			}

			// ALTER COLUMN SET NOT NULL
			if cmd.ALTER() != nil && cmd.SET() != nil && cmd.NOT() != nil && cmd.NULL_P() != nil {
				allColids := cmd.AllColid()
				if len(allColids) > 0 {
					columnName := pg.NormalizePostgreSQLColid(allColids[0])
					r.removeColumn("public", tableName, columnName)
				}
			}

			// ALTER COLUMN DROP NOT NULL
			if cmd.ALTER() != nil && cmd.DROP() != nil && cmd.NOT() != nil && cmd.NULL_P() != nil {
				allColids := cmd.AllColid()
				if len(allColids) > 0 {
					columnName := pg.NormalizePostgreSQLColid(allColids[0])
					r.addColumn("public", tableName, columnName, cmd.GetStart().GetLine())
				}
			}

			// ADD table constraint
			if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
				r.removeColumnByTableConstraint("public", tableName, cmd.Tableconstraint())
			}
		}
	}
}

func (*columnNoNullRule) extractTableName(qualifiedNames []parser.IQualified_nameContext) string {
	if len(qualifiedNames) == 0 {
		return ""
	}

	return extractTableName(qualifiedNames[0])
}

func (r *columnNoNullRule) addColumn(schema, table, column string, line int) {
	if schema == "" {
		schema = "public"
	}
	r.nullableColumns[columnName{schema: schema, table: table, column: column}] = line
}

func (r *columnNoNullRule) removeColumn(schema, table, column string) {
	if schema == "" {
		schema = "public"
	}
	delete(r.nullableColumns, columnName{schema: schema, table: table, column: column})
}

func (r *columnNoNullRule) removeColumnByColConstraints(schema, table string, colDef parser.IColumnDefContext) {
	if colDef.Colquallist() == nil {
		return
	}

	columnName := pg.NormalizePostgreSQLColid(colDef.Colid())
	allConstraints := colDef.Colquallist().AllColconstraint()
	for _, constraint := range allConstraints {
		if constraint.Colconstraintelem() == nil {
			continue
		}

		elem := constraint.Colconstraintelem()

		// NOT NULL constraint
		if elem.NOT() != nil && elem.NULL_P() != nil {
			r.removeColumn(schema, table, columnName)
			return
		}

		// PRIMARY KEY constraint
		if elem.PRIMARY() != nil && elem.KEY() != nil {
			r.removeColumn(schema, table, columnName)
			return
		}
	}
}

func (r *columnNoNullRule) removeColumnByTableConstraint(schema, table string, constraint parser.ITableconstraintContext) {
	if constraint.Constraintelem() == nil {
		return
	}

	elem := constraint.Constraintelem()

	// PRIMARY KEY (col1, col2, ...)
	if elem.PRIMARY() != nil && elem.KEY() != nil && elem.Columnlist() != nil {
		allColumnElems := elem.Columnlist().AllColumnElem()
		for _, columnElem := range allColumnElems {
			if columnElem.Colid() != nil {
				r.removeColumn(schema, table, pg.NormalizePostgreSQLColid(columnElem.Colid()))
			}
		}
		return
	}

	// PRIMARY KEY USING INDEX
	if elem.PRIMARY() != nil && elem.KEY() != nil && elem.Existingindex() != nil {
		existingIndex := elem.Existingindex()
		if existingIndex.Name() != nil {
			indexName := pg.NormalizePostgreSQLName(existingIndex.Name())
			// Try to find index in catalog
			if r.originCatalog != nil {
				_, index := r.originCatalog.GetIndex(schema, table, indexName)
				if index != nil {
					for _, expression := range index.ExpressionList() {
						r.removeColumn(schema, table, expression)
					}
				}
			}
		}
	}
}
