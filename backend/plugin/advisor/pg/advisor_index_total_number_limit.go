package pg

import (
	"context"
	"fmt"
	"slices"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*IndexTotalNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, &IndexTotalNumberLimitAdvisor{})
}

// IndexTotalNumberLimitAdvisor is the advisor checking for index total number limit.
type IndexTotalNumberLimitAdvisor struct {
}

// Check checks for index total number limit.
func (*IndexTotalNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	maxCount := int(numberPayload.Number)
	finalMetadata := checkCtx.FinalMetadata
	tableLine := make(tableLineMap)

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		baseLine := stmt.BaseLine()
		stmtText := stmt.Text
		handleIndexTotalStmt(node, baseLine, stmtText, tableLine)
	}

	var adviceList []*storepb.Advice
	generateIndexTotalAdvice(tableLine, finalMetadata, maxCount, level, checkCtx.Rule.Type.String(), &adviceList)
	return adviceList, nil
}

type tableLine struct {
	schema string
	table  string
	line   int
}

type tableLineMap map[string]tableLine

func (m tableLineMap) set(schema string, table string, line int) {
	if schema == "" {
		schema = "public"
	}
	m[fmt.Sprintf("%q.%q", schema, table)] = tableLine{
		schema: schema,
		table:  table,
		line:   line,
	}
}

func handleIndexTotalStmt(node ast.Node, baseLine int, stmtText string, tl tableLineMap) {
	// Helper to compute absolute end line (matching ANTLR GetStop().GetLine() behavior).
	absEndLine := func() int {
		endLine := int32(1)
		if stmtText != "" {
			for i := len(stmtText) - 1; i >= 0; i-- {
				c := stmtText[i]
				if c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != ';' {
					pos := pgparser.ByteOffsetToRunePosition(stmtText, i)
					endLine = pos.Line
					break
				}
			}
		}
		return baseLine + int(endLine)
	}

	switch n := node.(type) {
	case *ast.CreateStmt:
		tableName := omniTableName(n.Relation)
		if tableName == "" {
			return
		}
		schemaName := omniSchemaName(n.Relation)
		cols, constraints := omniTableElements(n)
		// Check column-level constraints
		for _, col := range cols {
			for _, c := range omniColumnConstraints(col) {
				if c.Contype == ast.CONSTR_PRIMARY || c.Contype == ast.CONSTR_UNIQUE {
					tl.set(schemaName, tableName, absEndLine())
					return
				}
			}
		}
		// Check table-level constraints
		for _, c := range constraints {
			if c.Contype == ast.CONSTR_PRIMARY || c.Contype == ast.CONSTR_UNIQUE {
				tl.set(schemaName, tableName, absEndLine())
				return
			}
		}

	case *ast.AlterTableStmt:
		tableName := omniTableName(n.Relation)
		if tableName == "" {
			return
		}
		schemaName := omniSchemaName(n.Relation)
		for _, cmd := range omniAlterTableCmds(n) {
			// ADD COLUMN with PK/UNIQUE constraint
			if cmd.Subtype == int(ast.AT_AddColumn) {
				if colDef, ok := cmd.Def.(*ast.ColumnDef); ok {
					for _, c := range omniColumnConstraints(colDef) {
						if c.Contype == ast.CONSTR_PRIMARY || c.Contype == ast.CONSTR_UNIQUE {
							tl.set(schemaName, tableName, absEndLine())
							return
						}
					}
				}
			}
			// ADD CONSTRAINT (PK or UNIQUE)
			if cmd.Subtype == int(ast.AT_AddConstraint) {
				if c, ok := cmd.Def.(*ast.Constraint); ok {
					if c.Contype == ast.CONSTR_PRIMARY || c.Contype == ast.CONSTR_UNIQUE {
						tl.set(schemaName, tableName, absEndLine())
						return
					}
				}
			}
		}

	case *ast.IndexStmt:
		tableName := omniTableName(n.Relation)
		if tableName == "" {
			return
		}
		schemaName := omniSchemaName(n.Relation)
		tl.set(schemaName, tableName, absEndLine())
	default:
	}
}

func generateIndexTotalAdvice(tl tableLineMap, finalMetadata *model.DatabaseMetadata, maxCount int, level storepb.Advice_Status, title string, adviceList *[]*storepb.Advice) {
	var tableList []tableLine
	for _, table := range tl {
		tableList = append(tableList, table)
	}
	slices.SortFunc(tableList, func(i, j tableLine) int {
		if i.line < j.line {
			return -1
		}
		if i.line > j.line {
			return 1
		}
		return 0
	})

	for _, table := range tableList {
		schema := finalMetadata.GetSchemaMetadata(table.schema)
		if schema == nil {
			continue
		}
		tableInfo := schema.GetTable(table.table)
		if tableInfo == nil {
			continue
		}
		if len(tableInfo.GetProto().Indexes) > maxCount {
			*adviceList = append(*adviceList, &storepb.Advice{
				Status:  level,
				Code:    code.IndexCountExceedsLimit.Int32(),
				Title:   title,
				Content: fmt.Sprintf("The count of index in table %q.%q should be no more than %d, but found %d", table.schema, table.table, maxCount, len(tableInfo.GetProto().Indexes)),
				StartPosition: &storepb.Position{
					Line:   int32(table.line),
					Column: 0,
				},
			})
		}
	}
}
