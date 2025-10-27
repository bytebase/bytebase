package pgantlr

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*CompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleSchemaBackwardCompatibility, &CompatibilityAdvisor{})
}

// CompatibilityAdvisor is the advisor checking for schema backward compatibility.
type CompatibilityAdvisor struct {
}

// Check checks schema backward compatibility.
func (*CompatibilityAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &compatibilityChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		statementsText:               checkCtx.Statements,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type compatibilityChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList      []*storepb.Advice
	level           storepb.Advice_Status
	title           string
	statementsText  string
	lastCreateTable string
}

// EnterCreatestmt tracks CREATE TABLE statements
func (c *compatibilityChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	qualifiedNames := ctx.AllQualified_name()
	if len(qualifiedNames) > 0 {
		c.lastCreateTable = extractTableName(qualifiedNames[0])
	}
}

// EnterDropdbstmt handles DROP DATABASE
func (c *compatibilityChecker) EnterDropdbstmt(ctx *parser.DropdbstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
	c.adviceList = append(c.adviceList, &storepb.Advice{
		Status:  c.level,
		Code:    advisor.CompatibilityDropDatabase.Int32(),
		Title:   c.title,
		Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, stmtText),
		StartPosition: &storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: 0,
		},
	})
}

// EnterDropstmt handles DROP TABLE/VIEW
func (c *compatibilityChecker) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is DROP TABLE or DROP VIEW
	if ctx.Object_type_any_name() != nil {
		objType := ctx.Object_type_any_name()
		if objType.TABLE() != nil || objType.VIEW() != nil {
			stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  c.level,
				Code:    advisor.CompatibilityDropTable.Int32(),
				Title:   c.title,
				Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, stmtText),
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// EnterRenamestmt handles ALTER TABLE RENAME and RENAME COLUMN
func (c *compatibilityChecker) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	code := advisor.Ok

	// Check if this is a column rename
	if ctx.Opt_column() != nil && ctx.Opt_column().COLUMN() != nil {
		// RENAME COLUMN - check if not on last created table
		if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
			tableName := extractTableName(ctx.Relation_expr().Qualified_name())
			if c.lastCreateTable != tableName {
				code = advisor.CompatibilityRenameColumn
			}
		}
	} else {
		// RENAME TABLE/VIEW
		code = advisor.CompatibilityRenameTable
	}

	if code != advisor.Ok {
		stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    code.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, stmtText),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// EnterAltertablestmt handles various ALTER TABLE commands
func (c *compatibilityChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}
	tableName := extractTableName(ctx.Relation_expr().Qualified_name())

	// Skip if this is the table we just created
	if c.lastCreateTable == tableName {
		return
	}

	if ctx.Alter_table_cmds() == nil {
		return
	}

	allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
	for _, cmd := range allCmds {
		code := advisor.Ok

		// DROP COLUMN
		if cmd.DROP() != nil && cmd.COLUMN() != nil {
			code = advisor.CompatibilityDropColumn
		}

		// ALTER COLUMN TYPE
		if cmd.ALTER() != nil && cmd.TYPE_P() != nil {
			code = advisor.CompatibilityAlterColumn
		}

		// ADD CONSTRAINT
		if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
			constraint := cmd.Tableconstraint()
			if constraint.Constraintelem() != nil {
				elem := constraint.Constraintelem()

				// PRIMARY KEY
				if elem.PRIMARY() != nil && elem.KEY() != nil {
					code = advisor.CompatibilityAddPrimaryKey
				}

				// UNIQUE
				if elem.UNIQUE() != nil {
					code = advisor.CompatibilityAddUniqueKey
				}

				// FOREIGN KEY
				if elem.FOREIGN() != nil && elem.KEY() != nil {
					code = advisor.CompatibilityAddForeignKey
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
						code = advisor.CompatibilityAddCheck
					}
				}
			}
		}

		if code != advisor.Ok {
			stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  c.level,
				Code:    code.Int32(),
				Title:   c.title,
				Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, stmtText),
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStart().GetLine()),
					Column: 0,
				},
			})
			return
		}
	}
}

// EnterIndexstmt handles CREATE UNIQUE INDEX
func (c *compatibilityChecker) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is CREATE UNIQUE INDEX
	if ctx.Opt_unique() == nil || ctx.Opt_unique().UNIQUE() == nil {
		return
	}

	// Get table name
	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}
	tableName := extractTableName(ctx.Relation_expr().Qualified_name())

	// Skip if this is the table we just created
	if c.lastCreateTable == tableName {
		return
	}

	stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())
	c.adviceList = append(c.adviceList, &storepb.Advice{
		Status:  c.level,
		Code:    advisor.CompatibilityAddUniqueKey.Int32(),
		Title:   c.title,
		Content: fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, stmtText),
		StartPosition: &storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: 0,
		},
	})
}
