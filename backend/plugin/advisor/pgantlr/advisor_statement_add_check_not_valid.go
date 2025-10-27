package pgantlr

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*StatementAddCheckNotValidAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementAddCheckNotValid, &StatementAddCheckNotValidAdvisor{})
}

// StatementAddCheckNotValidAdvisor is the advisor checking for to add check not valid.
type StatementAddCheckNotValidAdvisor struct {
}

// Check checks for to add check not valid.
func (*StatementAddCheckNotValidAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &statementAddCheckNotValidChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type statementAddCheckNotValidChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

// EnterColconstraint handles column-level constraints in ALTER TABLE ADD CONSTRAINT ... CHECK
// This handles the case where CHECK constraint is parsed as a column-level constraint
func (c *statementAddCheckNotValidChecker) EnterColconstraint(ctx *parser.ColconstraintContext) {
	// Check if this is within an ALTER TABLE statement
	parent := ctx.GetParent()
	var alterTableCtx *parser.AltertablestmtContext
	for parent != nil {
		if altCtx, ok := parent.(*parser.AltertablestmtContext); ok {
			alterTableCtx = altCtx
			if !isTopLevel(alterTableCtx.GetParent()) {
				return
			}
			break
		}
		parent = parent.GetParent()
	}

	// Only process if we're within an ALTER TABLE
	if alterTableCtx == nil {
		return
	}

	// Check if this is a CHECK constraint
	if ctx.Colconstraintelem() != nil {
		constraintElem := ctx.Colconstraintelem()
		if constraintElem.CHECK() != nil {
			// For column-level constraints, NOT VALID would be in Constraintattr
			// But since this is parsed as a column constraint, it won't have NOT VALID
			// (when NOT VALID is present, it's parsed as a table constraint)
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  c.level,
				Code:    advisor.StatementAddCheckWithValidation.Int32(),
				Title:   c.title,
				Content: "Adding check constraints with validation will block reads and writes. You can add check constraints not valid and then validate separately",
				StartPosition: &storepb.Position{
					Line:   int32(alterTableCtx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// EnterTableconstraint handles table-level constraints in ALTER TABLE ADD CONSTRAINT ... CHECK
// This handles the case where CHECK constraint is parsed as a table-level constraint (e.g., with NOT VALID)
func (c *statementAddCheckNotValidChecker) EnterTableconstraint(ctx *parser.TableconstraintContext) {
	// Check if this is within an ALTER TABLE statement
	parent := ctx.GetParent()
	var alterTableCtx *parser.AltertablestmtContext
	for parent != nil {
		if altCtx, ok := parent.(*parser.AltertablestmtContext); ok {
			alterTableCtx = altCtx
			if !isTopLevel(alterTableCtx.GetParent()) {
				return
			}
			break
		}
		parent = parent.GetParent()
	}

	// Only process if we're within an ALTER TABLE
	if alterTableCtx == nil {
		return
	}

	// Check if this is a CHECK constraint
	if ctx.Constraintelem() != nil {
		constraintElem := ctx.Constraintelem()
		if constraintElem.CHECK() != nil {
			// Check if NOT VALID is specified
			hasNotValid := false
			if constraintElem.Constraintattributespec() != nil {
				allAttrs := constraintElem.Constraintattributespec().AllConstraintattributeElem()
				for _, attr := range allAttrs {
					// Check for NOT VALID
					if attr.NOT() != nil && attr.VALID() != nil {
						hasNotValid = true
						break
					}
				}
			}

			if !hasNotValid {
				c.adviceList = append(c.adviceList, &storepb.Advice{
					Status:  c.level,
					Code:    advisor.StatementAddCheckWithValidation.Int32(),
					Title:   c.title,
					Content: "Adding check constraints with validation will block reads and writes. You can add check constraints not valid and then validate separately",
					StartPosition: &storepb.Position{
						Line:   int32(alterTableCtx.GetStart().GetLine()),
						Column: 0,
					},
				})
			}
		}
	}
}
