package pg

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_NO_NULL, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnNoNullRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		originalMetadata: checkCtx.OriginalMetadata,
		nullableColumns:  make(columnMap),
	}

	// Manually iterate statements instead of using RunOmniRules because
	// generateAdvice must be called AFTER all statements have been processed.
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetStatement(stmt.BaseLine(), stmt.Text)
		rule.OnStatement(node)
	}

	rule.generateAdvice()

	return rule.GetAdviceList(), nil
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

type columnMap map[columnName]int32

type columnNoNullRule struct {
	OmniBaseRule

	originalMetadata *model.DatabaseMetadata
	nullableColumns  columnMap
}

func (*columnNoNullRule) Name() string {
	return string(storepb.SQLReviewRule_COLUMN_NO_NULL)
}

func (r *columnNoNullRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	default:
	}
}

// columnLine finds the 1-based line number of a column name within the current
// statement text, searching from the given byte offset. Returns the line number
// and the updated search offset (past the found occurrence).
// Falls back to ContentStartLine() if not found.
func (r *columnNoNullRule) columnLine(colName string, searchFrom int) (int32, int) {
	text := r.StmtText
	if searchFrom < len(text) {
		idx := strings.Index(text[searchFrom:], colName)
		if idx >= 0 {
			pos := searchFrom + idx
			line := int32(1)
			for i := 0; i < pos; i++ {
				if text[i] == '\n' {
					line++
				}
			}
			return line, pos + len(colName)
		}
	}
	return r.ContentStartLine(), searchFrom
}

// generateAdvice generates advice for all nullable columns.
// This should be called AFTER processing all statements to avoid duplicates.
func (r *columnNoNullRule) generateAdvice() {
	var columnList []columnName
	for column := range r.nullableColumns {
		columnList = append(columnList, column)
	}

	if len(columnList) > 0 {
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
		lineNumber := r.nullableColumns[column]
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:  r.Level,
			Code:    code.ColumnCannotNull.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Column %q in %s cannot have NULL value", column.column, column.normalizeTableName()),
			StartPosition: &storepb.Position{
				Line:   lineNumber,
				Column: 0,
			},
		})
	}
}

func (r *columnNoNullRule) handleCreateStmt(n *ast.CreateStmt) {
	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}
	schema := omniSchemaName(n.Relation)

	cols, constraints := omniTableElements(n)

	searchFrom := 0
	for _, col := range cols {
		var line int32
		line, searchFrom = r.columnLine(col.Colname, searchFrom)
		r.addColumn(schema, tableName, col.Colname, line)

		// Check column-level constraints for NOT NULL or PRIMARY KEY
		if col.IsNotNull {
			r.removeColumn(schema, tableName, col.Colname)
			continue
		}
		for _, c := range omniColumnConstraints(col) {
			if c.Contype == ast.CONSTR_NOTNULL || c.Contype == ast.CONSTR_PRIMARY {
				r.removeColumn(schema, tableName, col.Colname)
				break
			}
		}
	}

	// Table-level constraints (e.g., PRIMARY KEY (col1, col2))
	for _, c := range constraints {
		if c.Contype == ast.CONSTR_PRIMARY {
			for _, colName := range omniConstraintColumns(c) {
				r.removeColumn(schema, tableName, colName)
			}
		}
	}
}

func (r *columnNoNullRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}
	schema := omniSchemaName(n.Relation)

	for _, cmd := range omniAlterTableCmds(n) {
		switch ast.AlterTableType(cmd.Subtype) {
		case ast.AT_AddColumn:
			colDef, ok := cmd.Def.(*ast.ColumnDef)
			if !ok || colDef == nil {
				continue
			}
			line, _ := r.columnLine(colDef.Colname, 0)
			r.addColumn(schema, tableName, colDef.Colname, line)

			// Check column-level constraints
			if colDef.IsNotNull {
				r.removeColumn(schema, tableName, colDef.Colname)
				continue
			}
			for _, c := range omniColumnConstraints(colDef) {
				if c.Contype == ast.CONSTR_NOTNULL || c.Contype == ast.CONSTR_PRIMARY {
					r.removeColumn(schema, tableName, colDef.Colname)
					break
				}
			}

		case ast.AT_SetNotNull:
			r.removeColumn(schema, tableName, cmd.Name)

		case ast.AT_DropNotNull:
			r.addColumn(schema, tableName, cmd.Name, r.ContentStartLine())

		case ast.AT_AddConstraint:
			constraint, ok := cmd.Def.(*ast.Constraint)
			if !ok || constraint == nil {
				continue
			}
			if constraint.Contype == ast.CONSTR_PRIMARY {
				for _, colName := range omniConstraintColumns(constraint) {
					r.removeColumn(schema, tableName, colName)
				}

				// PRIMARY KEY USING INDEX
				if constraint.Indexname != "" && r.originalMetadata != nil {
					schemaMetadata := r.originalMetadata.GetSchemaMetadata(schema)
					if schemaMetadata != nil {
						dbTable := schemaMetadata.GetTable(tableName)
						if dbTable != nil {
							index := dbTable.GetIndex(constraint.Indexname)
							if index != nil {
								for _, expression := range index.GetProto().GetExpressions() {
									r.removeColumn(schema, tableName, expression)
								}
							}
						}
					}
				}
			}
		default:
		}
	}
}

func (r *columnNoNullRule) addColumn(schema, table, column string, line int32) {
	if schema == "" {
		schema = "public"
	}
	// Store absolute line number (BaseLine + relative line)
	r.nullableColumns[columnName{schema: schema, table: table, column: column}] = line + int32(r.BaseLine)
}

func (r *columnNoNullRule) removeColumn(schema, table, column string) {
	if schema == "" {
		schema = "public"
	}
	delete(r.nullableColumns, columnName{schema: schema, table: table, column: column})
}
