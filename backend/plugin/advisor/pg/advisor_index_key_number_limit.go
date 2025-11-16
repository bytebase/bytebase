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
	_ advisor.Advisor = (*IndexKeyNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleIndexKeyNumberLimit, &IndexKeyNumberLimitAdvisor{})
}

// IndexKeyNumberLimitAdvisor is the advisor checking for index key number limit.
type IndexKeyNumberLimitAdvisor struct {
}

// Check checks for index key number limit.
func (*IndexKeyNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	rule := &indexKeyNumberLimitRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		max: payload.Number,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type indexKeyNumberLimitRule struct {
	BaseRule

	max int
}

func (*indexKeyNumberLimitRule) Name() string {
	return "index_key_number_limit"
}

func (r *indexKeyNumberLimitRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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

func (*indexKeyNumberLimitRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleIndexstmt checks CREATE INDEX statements
func (r *indexKeyNumberLimitRule) handleIndexstmt(ctx antlr.ParserRuleContext) {
	indexstmtCtx, ok := ctx.(*parser.IndexstmtContext)
	if !ok {
		return
	}

	if !isTopLevel(indexstmtCtx.GetParent()) {
		return
	}

	// Count the number of index parameters
	if indexstmtCtx.Index_params() != nil {
		keyCount := r.countIndexKeys(indexstmtCtx.Index_params())
		if r.max > 0 && keyCount > r.max {
			indexName := ""
			if indexstmtCtx.Name() != nil {
				indexName = pg.NormalizePostgreSQLName(indexstmtCtx.Name())
			}

			tableName := ""
			if indexstmtCtx.Relation_expr() != nil && indexstmtCtx.Relation_expr().Qualified_name() != nil {
				tableName = extractTableName(indexstmtCtx.Relation_expr().Qualified_name())
			}

			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    code.IndexKeyNumberExceedsLimit.Int32(),
				Title:   r.title,
				Content: fmt.Sprintf("The number of keys of index %q in table %q should be not greater than %d", indexName, tableName, r.max),
				StartPosition: &storepb.Position{
					Line:   int32(indexstmtCtx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// handleCreatestmt checks CREATE TABLE with inline constraints
func (r *indexKeyNumberLimitRule) handleCreatestmt(ctx antlr.ParserRuleContext) {
	createstmtCtx, ok := ctx.(*parser.CreatestmtContext)
	if !ok {
		return
	}

	if !isTopLevel(createstmtCtx.GetParent()) {
		return
	}

	qualifiedNames := createstmtCtx.AllQualified_name()
	if len(qualifiedNames) == 0 {
		return
	}

	tableName := extractTableName(qualifiedNames[0])
	if tableName == "" {
		return
	}

	// Check table-level constraints
	if createstmtCtx.Opttableelementlist() != nil && createstmtCtx.Opttableelementlist().Tableelementlist() != nil {
		allElements := createstmtCtx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			if elem.Tablelikeclause() != nil {
				continue
			}
			if elem.Tableconstraint() != nil {
				r.checkTableConstraint(elem.Tableconstraint(), tableName, createstmtCtx.GetStart().GetLine())
			}
		}
	}
}

// handleAltertablestmt checks ALTER TABLE ADD CONSTRAINT
func (r *indexKeyNumberLimitRule) handleAltertablestmt(ctx antlr.ParserRuleContext) {
	altertablestmtCtx, ok := ctx.(*parser.AltertablestmtContext)
	if !ok {
		return
	}

	if !isTopLevel(altertablestmtCtx.GetParent()) {
		return
	}

	if altertablestmtCtx.Relation_expr() == nil || altertablestmtCtx.Relation_expr().Qualified_name() == nil {
		return
	}

	tableName := extractTableName(altertablestmtCtx.Relation_expr().Qualified_name())
	if tableName == "" {
		return
	}

	// Check ALTER TABLE ADD CONSTRAINT
	if altertablestmtCtx.Alter_table_cmds() != nil {
		allCmds := altertablestmtCtx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// ADD CONSTRAINT
			if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
				r.checkTableConstraint(cmd.Tableconstraint(), tableName, altertablestmtCtx.GetStart().GetLine())
			}
		}
	}
}

func (r *indexKeyNumberLimitRule) checkTableConstraint(constraint parser.ITableconstraintContext, tableName string, line int) {
	if constraint == nil {
		return
	}

	var keyCount int
	var constraintName string

	// Get constraint name if present
	if constraint.Name() != nil {
		constraintName = pg.NormalizePostgreSQLName(constraint.Name())
	}

	// Check different constraint types
	if constraint.Constraintelem() != nil {
		elem := constraint.Constraintelem()

		// PRIMARY KEY or UNIQUE
		if (elem.PRIMARY() != nil && elem.KEY() != nil) || (elem.UNIQUE() != nil) {
			if elem.Columnlist() != nil {
				keyCount = r.countColumnList(elem.Columnlist())
			}
		}

		// FOREIGN KEY
		if elem.FOREIGN() != nil && elem.KEY() != nil {
			if elem.Columnlist() != nil {
				keyCount = r.countColumnList(elem.Columnlist())
			}
		}
	}

	if r.max > 0 && keyCount > r.max {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.IndexKeyNumberExceedsLimit.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("The number of keys of index %q in table %q should be not greater than %d", constraintName, tableName, r.max),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}

func (*indexKeyNumberLimitRule) countIndexKeys(params parser.IIndex_paramsContext) int {
	if params == nil {
		return 0
	}

	allParams := params.AllIndex_elem()
	return len(allParams)
}

func (*indexKeyNumberLimitRule) countColumnList(columnList parser.IColumnlistContext) int {
	if columnList == nil {
		return 0
	}

	allColumns := columnList.AllColumnElem()
	return len(allColumns)
}
