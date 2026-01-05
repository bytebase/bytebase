package pg

import (
	"context"
	"fmt"
	"slices"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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

	rule := &indexTotalNumberLimitRule{
		BaseRule:      BaseRule{level: level, title: checkCtx.Rule.Type.String()},
		max:           int(numberPayload.Number),
		finalMetadata: checkCtx.FinalMetadata,
		tableLine:     make(tableLineMap),
	}

	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	rule.generateAdvice()
	return checker.GetAdviceList(), nil
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

type indexTotalNumberLimitRule struct {
	BaseRule

	max           int
	finalMetadata *model.DatabaseMetadata
	tableLine     tableLineMap
}

func (*indexTotalNumberLimitRule) Name() string {
	return "index_total_number_limit"
}

func (r *indexTotalNumberLimitRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		r.handleCreatestmt(ctx.(*parser.CreatestmtContext))
	case "Altertablestmt":
		r.handleAltertablestmt(ctx.(*parser.AltertablestmtContext))
	case "Indexstmt":
		r.handleIndexstmt(ctx.(*parser.IndexstmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*indexTotalNumberLimitRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *indexTotalNumberLimitRule) generateAdvice() {
	var tableList []tableLine
	for _, table := range r.tableLine {
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
		schema := r.finalMetadata.GetSchemaMetadata(table.schema)
		if schema == nil {
			continue
		}
		tableInfo := schema.GetTable(table.table)
		if tableInfo == nil {
			continue
		}
		if len(tableInfo.GetProto().Indexes) > r.max {
			// Directly append to adviceList instead of using AddAdvice because
			// table.line already stores absolute line numbers (baseLine + line at time of encounter).
			// Using AddAdvice would incorrectly add the current baseLine again.
			r.adviceList = append(r.adviceList, &storepb.Advice{
				Status:  r.level,
				Code:    code.IndexCountExceedsLimit.Int32(),
				Title:   r.title,
				Content: fmt.Sprintf("The count of index in table %q.%q should be no more than %d, but found %d", table.schema, table.table, r.max, len(tableInfo.GetProto().Indexes)),
				StartPosition: &storepb.Position{
					Line:   int32(table.line),
					Column: 0,
				},
			})
		}
	}
}

func (r *indexTotalNumberLimitRule) handleCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	qualifiedNames := ctx.AllQualified_name()
	if len(qualifiedNames) == 0 {
		return
	}

	tableName := extractTableName(qualifiedNames[0])
	if tableName == "" {
		return
	}

	schemaName := extractSchemaName(qualifiedNames[0])

	// Check if this CREATE TABLE statement creates any indexes
	// (PRIMARY KEY or UNIQUE constraints)
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			// Check column-level constraints
			if elem.ColumnDef() != nil {
				if hasIndexConstraint(elem.ColumnDef()) {
					r.tableLine.set(schemaName, tableName, r.baseLine+ctx.GetStop().GetLine())
					return
				}
			}

			// Check table-level constraints
			if elem.Tableconstraint() != nil && hasTableIndexConstraint(elem.Tableconstraint()) {
				r.tableLine.set(schemaName, tableName, r.baseLine+ctx.GetStop().GetLine())
				return
			}
		}
	}
}

func (r *indexTotalNumberLimitRule) handleAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	tableName := extractTableName(ctx.Relation_expr().Qualified_name())
	if tableName == "" {
		return
	}

	schemaName := extractSchemaName(ctx.Relation_expr().Qualified_name())

	// Check ALTER TABLE commands that create indexes
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// ADD COLUMN with PRIMARY KEY or UNIQUE
			if cmd.ADD_P() != nil && cmd.ColumnDef() != nil {
				if hasIndexConstraint(cmd.ColumnDef()) {
					r.tableLine.set(schemaName, tableName, r.baseLine+ctx.GetStop().GetLine())
					return
				}
			}

			// ADD CONSTRAINT (PRIMARY KEY or UNIQUE)
			if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
				if hasTableIndexConstraint(cmd.Tableconstraint()) {
					r.tableLine.set(schemaName, tableName, r.baseLine+ctx.GetStop().GetLine())
					return
				}
			}
		}
	}
}

func (r *indexTotalNumberLimitRule) handleIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	tableName := extractTableName(ctx.Relation_expr().Qualified_name())
	if tableName == "" {
		return
	}

	schemaName := extractSchemaName(ctx.Relation_expr().Qualified_name())
	r.tableLine.set(schemaName, tableName, r.baseLine+ctx.GetStop().GetLine())
}

// hasIndexConstraint checks if a column definition has PRIMARY KEY or UNIQUE constraint
func hasIndexConstraint(colDef parser.IColumnDefContext) bool {
	if colDef.Colquallist() == nil {
		return false
	}

	allConstraints := colDef.Colquallist().AllColconstraint()
	for _, constraint := range allConstraints {
		if constraint.Colconstraintelem() != nil {
			elem := constraint.Colconstraintelem()
			// PRIMARY KEY creates an index
			if elem.PRIMARY() != nil && elem.KEY() != nil {
				return true
			}
			// UNIQUE creates an index
			if elem.UNIQUE() != nil {
				return true
			}
		}
	}

	return false
}

// hasTableIndexConstraint checks if a table constraint is PRIMARY KEY or UNIQUE
func hasTableIndexConstraint(constraint parser.ITableconstraintContext) bool {
	if constraint == nil || constraint.Constraintelem() == nil {
		return false
	}

	elem := constraint.Constraintelem()

	// PRIMARY KEY creates an index
	if elem.PRIMARY() != nil && elem.KEY() != nil {
		return true
	}

	// UNIQUE creates an index
	if elem.UNIQUE() != nil {
		return true
	}

	return false
}
