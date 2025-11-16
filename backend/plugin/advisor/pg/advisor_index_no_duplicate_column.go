package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*IndexNoDuplicateColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleIndexNoDuplicateColumn, &IndexNoDuplicateColumnAdvisor{})
}

// IndexNoDuplicateColumnAdvisor is the advisor checking for no duplicate columns in index.
type IndexNoDuplicateColumnAdvisor struct {
}

// Check checks for no duplicate columns in index.
func (*IndexNoDuplicateColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &indexNoDuplicateColumnRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
	}

	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type indexNoDuplicateColumnRule struct {
	BaseRule
}

func (*indexNoDuplicateColumnRule) Name() string {
	return "index_no_duplicate_column"
}

func (r *indexNoDuplicateColumnRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Indexstmt":
		r.handleIndexstmt(ctx)
	case "Createstmt":
		r.handleCreatestmt(ctx)
	case "Altertablestmt":
		r.handleAltertablestmt(ctx)
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*indexNoDuplicateColumnRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *indexNoDuplicateColumnRule) handleIndexstmt(ctx antlr.ParserRuleContext) {
	indexCtx, ok := ctx.(*parser.IndexstmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Get index name
	indexName := ""
	if indexCtx.Name() != nil {
		indexName = pg.NormalizePostgreSQLName(indexCtx.Name())
	}

	// Get table name
	tableName := ""
	if indexCtx.Relation_expr() != nil && indexCtx.Relation_expr().Qualified_name() != nil {
		tableName = extractTableName(indexCtx.Relation_expr().Qualified_name())
	}

	// Check for duplicate columns in index parameters
	if indexCtx.Index_params() != nil {
		columns := r.extractIndexColumns(indexCtx.Index_params())
		if dupCol := findDuplicate(columns); dupCol != "" {
			r.addAdvice("INDEX", indexName, tableName, dupCol, indexCtx.GetStart().GetLine())
		}
	}
}

func (r *indexNoDuplicateColumnRule) handleCreatestmt(ctx antlr.ParserRuleContext) {
	createCtx, ok := ctx.(*parser.CreatestmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	qualifiedNames := createCtx.AllQualified_name()
	if len(qualifiedNames) == 0 {
		return
	}

	tableName := extractTableName(qualifiedNames[0])
	if tableName == "" {
		return
	}

	// Check table-level constraints
	if createCtx.Opttableelementlist() != nil && createCtx.Opttableelementlist().Tableelementlist() != nil {
		allElements := createCtx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			if elem.Tableconstraint() != nil {
				r.checkTableConstraint(elem.Tableconstraint(), tableName, elem.GetStart().GetLine())
			}
		}
	}
}

func (r *indexNoDuplicateColumnRule) handleAltertablestmt(ctx antlr.ParserRuleContext) {
	alterCtx, ok := ctx.(*parser.AltertablestmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if alterCtx.Relation_expr() == nil || alterCtx.Relation_expr().Qualified_name() == nil {
		return
	}

	tableName := extractTableName(alterCtx.Relation_expr().Qualified_name())
	if tableName == "" {
		return
	}

	// Check ALTER TABLE ADD CONSTRAINT
	if alterCtx.Alter_table_cmds() != nil {
		allCmds := alterCtx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// ADD CONSTRAINT
			if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
				r.checkTableConstraint(cmd.Tableconstraint(), tableName, alterCtx.GetStart().GetLine())
			}
		}
	}
}

func (r *indexNoDuplicateColumnRule) checkTableConstraint(constraint parser.ITableconstraintContext, tableName string, line int) {
	if constraint == nil {
		return
	}

	// Get constraint name
	constraintName := ""
	if constraint.Name() != nil {
		constraintName = pg.NormalizePostgreSQLName(constraint.Name())
	}

	// Check different constraint types
	if constraint.Constraintelem() != nil {
		elem := constraint.Constraintelem()

		// PRIMARY KEY
		if elem.PRIMARY() != nil && elem.KEY() != nil {
			if elem.Columnlist() != nil {
				columns := r.extractColumnList(elem.Columnlist())
				if dupCol := findDuplicate(columns); dupCol != "" {
					r.addAdvice("PRIMARY KEY", constraintName, tableName, dupCol, line)
				}
			}
		}

		// UNIQUE
		if elem.UNIQUE() != nil {
			if elem.Columnlist() != nil {
				columns := r.extractColumnList(elem.Columnlist())
				if dupCol := findDuplicate(columns); dupCol != "" {
					r.addAdvice("UNIQUE KEY", constraintName, tableName, dupCol, line)
				}
			}
		}

		// FOREIGN KEY
		if elem.FOREIGN() != nil && elem.KEY() != nil {
			if elem.Columnlist() != nil {
				columns := r.extractColumnList(elem.Columnlist())
				if dupCol := findDuplicate(columns); dupCol != "" {
					r.addAdvice("FOREIGN KEY", constraintName, tableName, dupCol, line)
				}
			}
		}
	}
}

func (*indexNoDuplicateColumnRule) extractIndexColumns(params parser.IIndex_paramsContext) []string {
	if params == nil {
		return nil
	}

	var columns []string
	allParams := params.AllIndex_elem()
	for _, param := range allParams {
		if param.Colid() != nil {
			colName := pg.NormalizePostgreSQLColid(param.Colid())
			columns = append(columns, colName)
		}
	}

	return columns
}

func (*indexNoDuplicateColumnRule) extractColumnList(columnList parser.IColumnlistContext) []string {
	if columnList == nil {
		return nil
	}

	var columns []string
	allColumns := columnList.AllColumnElem()
	for _, col := range allColumns {
		if col.Colid() != nil {
			colName := pg.NormalizePostgreSQLColid(col.Colid())
			columns = append(columns, colName)
		}
	}

	return columns
}

func findDuplicate(columns []string) string {
	seen := make(map[string]bool)
	for _, col := range columns {
		if seen[col] {
			return col
		}
		seen[col] = true
	}
	return ""
}

func (r *indexNoDuplicateColumnRule) addAdvice(constraintType, constraintName, tableName, duplicateColumn string, line int) {
	r.AddAdvice(&storepb.Advice{
		Status:  r.level,
		Code:    code.DuplicateColumnInIndex.Int32(),
		Title:   r.title,
		Content: fmt.Sprintf("%s %q has duplicate column %q.%q", constraintType, constraintName, tableName, duplicateColumn),
		StartPosition: &storepb.Position{
			Line:   int32(line),
			Column: 0,
		},
	})
}
