package pgantlr

import (
	"context"
	"fmt"

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

	checker := &indexKeyNumberLimitChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		max:                          payload.Number,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type indexKeyNumberLimitChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	max        int
}

// EnterIndexstmt checks CREATE INDEX statements
func (c *indexKeyNumberLimitChecker) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Count the number of index parameters
	if ctx.Index_params() != nil {
		keyCount := c.countIndexKeys(ctx.Index_params())
		if c.max > 0 && keyCount > c.max {
			indexName := ""
			if ctx.Name() != nil {
				indexName = pg.NormalizePostgreSQLName(ctx.Name())
			}

			tableName := ""
			if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
				tableName = extractTableName(ctx.Relation_expr().Qualified_name())
			}

			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  c.level,
				Code:    advisor.IndexKeyNumberExceedsLimit.Int32(),
				Title:   c.title,
				Content: fmt.Sprintf("The number of keys of index %q in table %q should be not greater than %d", indexName, tableName, c.max),
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// EnterCreatestmt checks CREATE TABLE with inline constraints
func (c *indexKeyNumberLimitChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
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

	// Check table-level constraints
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			if elem.Tablelikeclause() != nil {
				continue
			}
			if elem.Tableconstraint() != nil {
				c.checkTableConstraint(elem.Tableconstraint(), tableName, ctx.GetStart().GetLine())
			}
		}
	}
}

// EnterAltertablestmt checks ALTER TABLE ADD CONSTRAINT
func (c *indexKeyNumberLimitChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
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

	// Check ALTER TABLE ADD CONSTRAINT
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// ADD CONSTRAINT
			if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
				c.checkTableConstraint(cmd.Tableconstraint(), tableName, ctx.GetStart().GetLine())
			}
		}
	}
}

func (c *indexKeyNumberLimitChecker) checkTableConstraint(constraint parser.ITableconstraintContext, tableName string, line int) {
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
				keyCount = c.countColumnList(elem.Columnlist())
			}
		}

		// FOREIGN KEY
		if elem.FOREIGN() != nil && elem.KEY() != nil {
			if elem.Columnlist() != nil {
				keyCount = c.countColumnList(elem.Columnlist())
			}
		}
	}

	if c.max > 0 && keyCount > c.max {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.IndexKeyNumberExceedsLimit.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("The number of keys of index %q in table %q should be not greater than %d", constraintName, tableName, c.max),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}

func (*indexKeyNumberLimitChecker) countIndexKeys(params parser.IIndex_paramsContext) int {
	if params == nil {
		return 0
	}

	allParams := params.AllIndex_elem()
	return len(allParams)
}

func (*indexKeyNumberLimitChecker) countColumnList(columnList parser.IColumnlistContext) int {
	if columnList == nil {
		return 0
	}

	allColumns := columnList.AllColumnElem()
	return len(allColumns)
}
