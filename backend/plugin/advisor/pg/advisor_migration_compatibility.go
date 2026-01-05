package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, &CompatibilityAdvisor{})
}

// CompatibilityAdvisor is the advisor checking for schema backward compatibility.
type CompatibilityAdvisor struct {
}

// Check checks schema backward compatibility.
func (*CompatibilityAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var adviceList []*storepb.Advice
	lastCreateTable := ""

	for _, stmtInfo := range checkCtx.ParsedStatements {
		if stmtInfo.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmtInfo.AST)
		if !ok {
			continue
		}
		rule := &compatibilityRule{
			BaseRule: BaseRule{
				level: level,
				title: checkCtx.Rule.Type.String(),
			},
			tokens:          antlrAST.Tokens,
			lastCreateTable: lastCreateTable,
		}

		checker := NewGenericChecker([]Rule{rule})
		rule.SetBaseLine(stmtInfo.BaseLine())
		checker.SetBaseLine(stmtInfo.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)

		adviceList = append(adviceList, checker.GetAdviceList()...)
		lastCreateTable = rule.lastCreateTable
	}

	return adviceList, nil
}

type compatibilityRule struct {
	BaseRule

	tokens          *antlr.CommonTokenStream
	lastCreateTable string
}

func (*compatibilityRule) Name() string {
	return "migration_compatibility"
}

func (r *compatibilityRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		r.handleCreatestmt(ctx)
	case "Dropdbstmt":
		r.handleDropdbstmt(ctx)
	case "Dropstmt":
		r.handleDropstmt(ctx)
	case "Renamestmt":
		r.handleRenamestmt(ctx)
	case "Altertablestmt":
		r.handleAltertablestmt(ctx)
	case "Indexstmt":
		r.handleIndexstmt(ctx)
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*compatibilityRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleCreatestmt tracks CREATE TABLE statements
func (r *compatibilityRule) handleCreatestmt(ctx antlr.ParserRuleContext) {
	createstmtCtx, ok := ctx.(*parser.CreatestmtContext)
	if !ok {
		return
	}

	if !isTopLevel(createstmtCtx.GetParent()) {
		return
	}

	qualifiedNames := createstmtCtx.AllQualified_name()
	if len(qualifiedNames) > 0 {
		r.lastCreateTable = extractTableName(qualifiedNames[0])
	}
}

// handleDropdbstmt handles DROP DATABASE
func (r *compatibilityRule) handleDropdbstmt(ctx antlr.ParserRuleContext) {
	dropdbstmtCtx, ok := ctx.(*parser.DropdbstmtContext)
	if !ok {
		return
	}

	if !isTopLevel(dropdbstmtCtx.GetParent()) {
		return
	}

	r.AddAdvice(&storepb.Advice{
		Status:  r.level,
		Code:    advisorcode.CompatibilityDropDatabase.Int32(),
		Title:   r.title,
		Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, getTextFromTokens(r.tokens, dropdbstmtCtx)),
		StartPosition: &storepb.Position{
			Line:   int32(dropdbstmtCtx.GetStart().GetLine()),
			Column: 0,
		},
	})
}

// handleDropstmt handles DROP TABLE/VIEW
func (r *compatibilityRule) handleDropstmt(ctx antlr.ParserRuleContext) {
	dropstmtCtx, ok := ctx.(*parser.DropstmtContext)
	if !ok {
		return
	}

	if !isTopLevel(dropstmtCtx.GetParent()) {
		return
	}

	// Check if this is DROP TABLE or DROP VIEW
	if dropstmtCtx.Object_type_any_name() != nil {
		objType := dropstmtCtx.Object_type_any_name()
		if objType.TABLE() != nil || objType.VIEW() != nil {
			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    advisorcode.CompatibilityDropTable.Int32(),
				Title:   r.title,
				Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, getTextFromTokens(r.tokens, dropstmtCtx)),
				StartPosition: &storepb.Position{
					Line:   int32(dropstmtCtx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// handleRenamestmt handles ALTER TABLE RENAME and RENAME COLUMN
func (r *compatibilityRule) handleRenamestmt(ctx antlr.ParserRuleContext) {
	renamestmtCtx, ok := ctx.(*parser.RenamestmtContext)
	if !ok {
		return
	}

	if !isTopLevel(renamestmtCtx.GetParent()) {
		return
	}

	code := advisorcode.Ok

	// Check if this is a column rename
	if renamestmtCtx.Opt_column() != nil && renamestmtCtx.Opt_column().COLUMN() != nil {
		// RENAME COLUMN - check if not on last created table
		if renamestmtCtx.Relation_expr() != nil && renamestmtCtx.Relation_expr().Qualified_name() != nil {
			tableName := extractTableName(renamestmtCtx.Relation_expr().Qualified_name())
			if r.lastCreateTable != tableName {
				code = advisorcode.CompatibilityRenameColumn
			}
		}
	} else {
		// RENAME TABLE/VIEW
		code = advisorcode.CompatibilityRenameTable
	}

	if code != advisorcode.Ok {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, getTextFromTokens(r.tokens, renamestmtCtx)),
			StartPosition: &storepb.Position{
				Line:   int32(renamestmtCtx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// handleAltertablestmt handles various ALTER TABLE commands
func (r *compatibilityRule) handleAltertablestmt(ctx antlr.ParserRuleContext) {
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

	// Skip if this is the table we just created
	if r.lastCreateTable == tableName {
		return
	}

	if altertablestmtCtx.Alter_table_cmds() == nil {
		return
	}

	allCmds := altertablestmtCtx.Alter_table_cmds().AllAlter_table_cmd()
	for _, cmd := range allCmds {
		code := advisorcode.Ok

		// DROP COLUMN
		if cmd.DROP() != nil && cmd.COLUMN() != nil {
			code = advisorcode.CompatibilityDropColumn
		}

		// ALTER COLUMN TYPE
		if cmd.ALTER() != nil && cmd.TYPE_P() != nil {
			code = advisorcode.CompatibilityAlterColumn
		}

		// ADD CONSTRAINT
		if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
			constraint := cmd.Tableconstraint()
			if constraint.Constraintelem() != nil {
				elem := constraint.Constraintelem()

				// PRIMARY KEY
				if elem.PRIMARY() != nil && elem.KEY() != nil {
					code = advisorcode.CompatibilityAddPrimaryKey
				}

				// UNIQUE
				if elem.UNIQUE() != nil {
					code = advisorcode.CompatibilityAddUniqueKey
				}

				// FOREIGN KEY
				if elem.FOREIGN() != nil && elem.KEY() != nil {
					code = advisorcode.CompatibilityAddForeignKey
				}

				// CHECK - only if NOT VALID is not present
				if elem.CHECK() != nil {
					// Check if NOT VALID is present in constraint attributes
					hasNotValid := false
					if elem.Constraintattributespec() != nil {
						allAttrs := elem.Constraintattributespec().AllConstraintattributeElem()
						for _, attr := range allAttrs {
							if attr.NOT() != nil && attr.VALID() != nil {
								hasNotValid = true
								break
							}
						}
					}
					if !hasNotValid {
						code = advisorcode.CompatibilityAddCheck
					}
				}
			}
		}

		if code != advisorcode.Ok {
			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    code.Int32(),
				Title:   r.title,
				Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, getTextFromTokens(r.tokens, altertablestmtCtx)),
				StartPosition: &storepb.Position{
					Line:   int32(altertablestmtCtx.GetStart().GetLine()),
					Column: 0,
				},
			})
			return
		}
	}
}

// handleIndexstmt handles CREATE UNIQUE INDEX
func (r *compatibilityRule) handleIndexstmt(ctx antlr.ParserRuleContext) {
	indexstmtCtx, ok := ctx.(*parser.IndexstmtContext)
	if !ok {
		return
	}

	if !isTopLevel(indexstmtCtx.GetParent()) {
		return
	}

	// Check if this is CREATE UNIQUE INDEX
	if indexstmtCtx.Opt_unique() == nil || indexstmtCtx.Opt_unique().UNIQUE() == nil {
		return
	}

	// Get table name
	if indexstmtCtx.Relation_expr() == nil || indexstmtCtx.Relation_expr().Qualified_name() == nil {
		return
	}
	tableName := extractTableName(indexstmtCtx.Relation_expr().Qualified_name())

	// Skip if this is the table we just created
	if r.lastCreateTable == tableName {
		return
	}

	r.AddAdvice(&storepb.Advice{
		Status:  r.level,
		Code:    advisorcode.CompatibilityAddUniqueKey.Int32(),
		Title:   r.title,
		Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, getTextFromTokens(r.tokens, indexstmtCtx)),
		StartPosition: &storepb.Position{
			Line:   int32(indexstmtCtx.GetStart().GetLine()),
			Column: 0,
		},
	})
}
