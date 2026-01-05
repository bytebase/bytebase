package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*TableNoFKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoFKAdvisor{})
}

// TableNoFKAdvisor is the advisor checking table disallow foreign key.
type TableNoFKAdvisor struct {
}

// Check checks table disallow foreign key.
func (*TableNoFKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var adviceList []*storepb.Advice
	for _, stmtInfo := range checkCtx.ParsedStatements {
		if stmtInfo.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmtInfo.AST)
		if !ok {
			continue
		}
		rule := &tableNoFKRule{
			BaseRule: BaseRule{
				level: level,
				title: checkCtx.Rule.Type.String(),
			},
			tokens: antlrAST.Tokens,
		}

		checker := NewGenericChecker([]Rule{rule})
		checker.SetBaseLine(stmtInfo.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
		adviceList = append(adviceList, checker.GetAdviceList()...)
	}

	return adviceList, nil
}

type tableNoFKRule struct {
	BaseRule

	tokens *antlr.CommonTokenStream
}

func (*tableNoFKRule) Name() string {
	return "table_no_fk"
}

func (r *tableNoFKRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		r.handleCreatestmt(ctx)
	case "Altertablestmt":
		r.handleAltertablestmt(ctx)
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*tableNoFKRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleCreatestmt handles CREATE TABLE with FK constraints
func (r *tableNoFKRule) handleCreatestmt(ctx antlr.ParserRuleContext) {
	createstmtCtx, ok := ctx.(*parser.CreatestmtContext)
	if !ok {
		return
	}

	if !isTopLevel(createstmtCtx.GetParent()) {
		return
	}

	var tableName, schemaName string
	allQualifiedNames := createstmtCtx.AllQualified_name()
	if len(allQualifiedNames) > 0 {
		tableName = extractTableName(allQualifiedNames[0])
		schemaName = extractSchemaName(allQualifiedNames[0])
		if schemaName == "" {
			schemaName = "public"
		}
	}

	// Check table-level constraints
	if createstmtCtx.Opttableelementlist() != nil && createstmtCtx.Opttableelementlist().Tableelementlist() != nil {
		allElements := createstmtCtx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			// Check if this is a table constraint
			if elem.Tableconstraint() != nil {
				constraint := elem.Tableconstraint()
				if constraint.Constraintelem() != nil {
					constraintElem := constraint.Constraintelem()
					// Check for FOREIGN KEY
					if constraintElem.FOREIGN() != nil && constraintElem.KEY() != nil {
						r.addFKAdvice(schemaName, tableName, createstmtCtx)
						return
					}
				}
			}

			// Check column-level constraints
			if elem.ColumnDef() != nil {
				columnDef := elem.ColumnDef()
				if columnDef.Colquallist() != nil {
					allQuals := columnDef.Colquallist().AllColconstraint()
					for _, qual := range allQuals {
						if qual.Colconstraintelem() != nil {
							constraintElem := qual.Colconstraintelem()
							// Check for REFERENCES (column-level FK)
							if constraintElem.REFERENCES() != nil {
								r.addFKAdvice(schemaName, tableName, createstmtCtx)
								return
							}
						}
					}
				}
			}
		}
	}
}

// handleAltertablestmt handles ALTER TABLE ADD CONSTRAINT with FK
func (r *tableNoFKRule) handleAltertablestmt(ctx antlr.ParserRuleContext) {
	altertablestmtCtx, ok := ctx.(*parser.AltertablestmtContext)
	if !ok {
		return
	}

	if !isTopLevel(altertablestmtCtx.GetParent()) {
		return
	}

	var tableName, schemaName string
	if altertablestmtCtx.Relation_expr() != nil && altertablestmtCtx.Relation_expr().Qualified_name() != nil {
		tableName = extractTableName(altertablestmtCtx.Relation_expr().Qualified_name())
		schemaName = extractSchemaName(altertablestmtCtx.Relation_expr().Qualified_name())
		if schemaName == "" {
			schemaName = "public"
		}
	}

	// Check all alter table commands for ADD CONSTRAINT with FOREIGN KEY
	if altertablestmtCtx.Alter_table_cmds() != nil {
		allCmds := altertablestmtCtx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
				constraint := cmd.Tableconstraint()
				if constraint.Constraintelem() != nil {
					constraintElem := constraint.Constraintelem()
					// Check for FOREIGN KEY
					if constraintElem.FOREIGN() != nil && constraintElem.KEY() != nil {
						r.addFKAdvice(schemaName, tableName, altertablestmtCtx)
						return
					}
				}
			}
		}
	}
}

func (r *tableNoFKRule) addFKAdvice(schemaName, tableName string, ctx antlr.ParserRuleContext) {
	r.AddAdvice(&storepb.Advice{
		Status:  r.level,
		Code:    code.TableHasFK.Int32(),
		Title:   r.title,
		Content: fmt.Sprintf("Foreign key is not allowed in the table %q.%q, related statement: \"%s\"", schemaName, tableName, getTextFromTokens(r.tokens, ctx)),
		StartPosition: &storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: 0,
		},
	})
}
