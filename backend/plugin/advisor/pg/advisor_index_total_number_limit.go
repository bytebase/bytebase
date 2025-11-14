package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
)

var (
	_ advisor.Advisor = (*IndexTotalNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleIndexTotalNumberLimit, &IndexTotalNumberLimitAdvisor{})
}

// IndexTotalNumberLimitAdvisor is the advisor checking for index total number limit.
type IndexTotalNumberLimitAdvisor struct {
}

// Check checks for index total number limit.
func (*IndexTotalNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	rule := &indexTotalNumberLimitRule{
		BaseRule:     BaseRule{level: level, title: string(checkCtx.Rule.Type)},
		max:          payload.Number,
		tableIndexes: make(map[string]map[string]bool),
		tableLines:   make(map[string]int),
		catalog:      checkCtx.Catalog,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	// Check all tables after processing all statements
	rule.checkAllTables()

	return checker.GetAdviceList(), nil
}

type indexTotalNumberLimitRule struct {
	BaseRule

	max          int
	tableIndexes map[string]map[string]bool // tableKey -> indexName -> exists
	tableLines   map[string]int             // tableKey -> last line number
	catalog      *catalog.Finder
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

func (r *indexTotalNumberLimitRule) checkAllTables() {
	for tableKey, indexes := range r.tableIndexes {
		// Parse table key to get schema and table name
		schema, table := parseTableKey(tableKey)

		// Get the number of indexes created in these statements
		newIndexes := len(indexes)

		// Get the number of indexes that already exist in catalog.Origin
		existingIndexes := 0
		if tableInfo := r.catalog.Origin.FindTable(&catalog.TableFind{
			SchemaName: schema,
			TableName:  table,
		}); tableInfo != nil {
			existingIndexes = tableInfo.CountIndex()
		}

		totalCount := existingIndexes + newIndexes
		if totalCount > r.max {
			line := r.tableLines[tableKey]
			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    advisor.IndexCountExceedsLimit.Int32(),
				Title:   r.title,
				Content: fmt.Sprintf("The count of index in table %q.%q should be no more than %d, but found %d", schema, table, r.max, totalCount),
				StartPosition: &storepb.Position{
					Line:   int32(line),
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
	tableKey := makeTableKey(schemaName, tableName)

	// Initialize table indexes map
	if r.tableIndexes[tableKey] == nil {
		r.tableIndexes[tableKey] = make(map[string]bool)
	}

	// Check if this CREATE TABLE statement creates any indexes
	// (PRIMARY KEY or UNIQUE constraints)
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			// Check column-level constraints
			if elem.ColumnDef() != nil {
				if hasIndexConstraint(elem.ColumnDef()) {
					indexName := fmt.Sprintf("__inline_index_%d__", len(r.tableIndexes[tableKey]))
					r.tableIndexes[tableKey][indexName] = true
				}
			}

			// Check table-level constraints
			if elem.Tableconstraint() != nil && hasTableIndexConstraint(elem.Tableconstraint()) {
				indexName := fmt.Sprintf("__index_%d__", len(r.tableIndexes[tableKey]))
				r.tableIndexes[tableKey][indexName] = true
			}
		}
	}

	// Track last line for this table
	r.tableLines[makeTableKey(schemaName, tableName)] = ctx.GetStop().GetLine()
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
	tableKey := makeTableKey(schemaName, tableName)

	// Initialize table indexes map
	if r.tableIndexes[tableKey] == nil {
		r.tableIndexes[tableKey] = make(map[string]bool)
	}

	// Check ALTER TABLE commands that create indexes
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// ADD COLUMN with PRIMARY KEY or UNIQUE
			if cmd.ADD_P() != nil && cmd.ColumnDef() != nil {
				if hasIndexConstraint(cmd.ColumnDef()) {
					indexName := fmt.Sprintf("__inline_index_%d__", len(r.tableIndexes[tableKey]))
					r.tableIndexes[tableKey][indexName] = true
				}
			}

			// ADD CONSTRAINT (PRIMARY KEY or UNIQUE)
			if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
				if hasTableIndexConstraint(cmd.Tableconstraint()) {
					indexName := fmt.Sprintf("__index_%d__", len(r.tableIndexes[tableKey]))
					r.tableIndexes[tableKey][indexName] = true
				}
			}
		}
	}

	// Track last line for this table
	r.tableLines[makeTableKey(schemaName, tableName)] = ctx.GetStop().GetLine()
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
	tableKey := makeTableKey(schemaName, tableName)

	// Initialize table indexes map
	if r.tableIndexes[tableKey] == nil {
		r.tableIndexes[tableKey] = make(map[string]bool)
	}

	// Add the index
	indexName := fmt.Sprintf("__index_%d__", len(r.tableIndexes[tableKey]))
	r.tableIndexes[tableKey][indexName] = true

	// Track last line for this table
	r.tableLines[makeTableKey(schemaName, tableName)] = ctx.GetStop().GetLine()
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
