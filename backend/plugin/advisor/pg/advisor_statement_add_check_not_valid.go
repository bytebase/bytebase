package pg

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
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

	rule := &statementAddCheckNotValidRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
	}

	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type statementAddCheckNotValidRule struct {
	BaseRule
}

func (*statementAddCheckNotValidRule) Name() string {
	return "statement_add_check_not_valid"
}

func (r *statementAddCheckNotValidRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Colconstraint":
		r.handleColconstraint(ctx)
	case "Tableconstraint":
		r.handleTableconstraint(ctx)
	default:
	}
	return nil
}

func (*statementAddCheckNotValidRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleColconstraint handles column-level constraints in ALTER TABLE ADD CONSTRAINT ... CHECK
// This handles the case where CHECK constraint is parsed as a column-level constraint
func (r *statementAddCheckNotValidRule) handleColconstraint(ctx antlr.ParserRuleContext) {
	colCtx, ok := ctx.(*parser.ColconstraintContext)
	if !ok {
		return
	}

	// Check if this is within an ALTER TABLE statement
	parent := colCtx.GetParent()
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
	if colCtx.Colconstraintelem() != nil {
		constraintElem := colCtx.Colconstraintelem()
		if constraintElem.CHECK() != nil {
			// For column-level constraints, NOT VALID would be in Constraintattr
			// But since this is parsed as a column constraint, it won't have NOT VALID
			// (when NOT VALID is present, it's parsed as a table constraint)
			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    code.StatementAddCheckWithValidation.Int32(),
				Title:   r.title,
				Content: "Adding check constraints with validation will block reads and writes. You can add check constraints not valid and then validate separately",
				StartPosition: &storepb.Position{
					Line:   int32(alterTableCtx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// handleTableconstraint handles table-level constraints in ALTER TABLE ADD CONSTRAINT ... CHECK
// This handles the case where CHECK constraint is parsed as a table-level constraint (e.g., with NOT VALID)
func (r *statementAddCheckNotValidRule) handleTableconstraint(ctx antlr.ParserRuleContext) {
	tableCtx, ok := ctx.(*parser.TableconstraintContext)
	if !ok {
		return
	}

	// Check if this is within an ALTER TABLE statement
	parent := tableCtx.GetParent()
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
	if tableCtx.Constraintelem() != nil {
		constraintElem := tableCtx.Constraintelem()
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
				r.AddAdvice(&storepb.Advice{
					Status:  r.level,
					Code:    code.StatementAddCheckWithValidation.Int32(),
					Title:   r.title,
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
