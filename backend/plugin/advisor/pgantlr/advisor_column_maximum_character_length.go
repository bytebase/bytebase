package pgantlr

import (
	"context"
	"fmt"
	"strconv"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*ColumnMaximumCharacterLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleColumnMaximumCharacterLength, &ColumnMaximumCharacterLengthAdvisor{})
}

// ColumnMaximumCharacterLengthAdvisor is the advisor checking for maximum character length.
type ColumnMaximumCharacterLengthAdvisor struct {
}

// Check checks for maximum character length.
func (*ColumnMaximumCharacterLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	if payload.Number <= 0 {
		return nil, nil
	}

	checker := &columnMaximumCharacterLengthChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		maximum:                      payload.Number,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type columnMaximumCharacterLengthChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	maximum    int
}

func (c *columnMaximumCharacterLengthChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	tableName := c.extractTableName(ctx.AllQualified_name())

	// Check all columns
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			if elem.ColumnDef() != nil {
				colDef := elem.ColumnDef()
				if colDef.Colid() != nil && colDef.Typename() != nil {
					columnName := normalizeColid(colDef.Colid())
					charLength := c.getCharLength(colDef.Typename())
					if charLength > c.maximum {
						c.addAdvice(tableName, columnName, ctx.GetStart().GetLine())
						return // Only report first violation
					}
				}
			}
		}
	}
}

func (c *columnMaximumCharacterLengthChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	text := ctx.Relation_expr().Qualified_name().GetText()
	parts := splitIdentifier(text)
	if len(parts) == 0 {
		return
	}

	var tableName string
	// Format: schema.table or just table (always with quotes)
	if len(parts) == 1 {
		tableName = fmt.Sprintf("%q", parts[0])
	} else {
		tableName = fmt.Sprintf("%q.%q", parts[0], parts[1])
	}

	// Check ALTER TABLE commands
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// ADD COLUMN
			if cmd.ADD_P() != nil && cmd.ColumnDef() != nil {
				colDef := cmd.ColumnDef()
				if colDef.Colid() != nil && colDef.Typename() != nil {
					columnName := normalizeColid(colDef.Colid())
					charLength := c.getCharLength(colDef.Typename())
					if charLength > c.maximum {
						c.addAdvice(tableName, columnName, ctx.GetStart().GetLine())
						return
					}
				}
			}

			// ALTER COLUMN TYPE
			if cmd.ALTER() != nil && cmd.TYPE_P() != nil && cmd.Typename() != nil {
				// Get column name
				allColids := cmd.AllColid()
				if len(allColids) > 0 {
					columnName := normalizeColid(allColids[0])
					charLength := c.getCharLength(cmd.Typename())
					if charLength > c.maximum {
						c.addAdvice(tableName, columnName, ctx.GetStart().GetLine())
						return
					}
				}
			}
		}
	}
}

func (*columnMaximumCharacterLengthChecker) extractTableName(qualifiedNames []parser.IQualified_nameContext) string {
	if len(qualifiedNames) == 0 {
		return ""
	}

	text := qualifiedNames[0].GetText()
	parts := splitIdentifier(text)
	if len(parts) == 0 {
		return ""
	}

	// Format: schema.table or just table (always with quotes)
	if len(parts) == 1 {
		return fmt.Sprintf("%q", parts[0])
	}
	return fmt.Sprintf("%q.%q", parts[0], parts[1])
}

func (*columnMaximumCharacterLengthChecker) getCharLength(typename parser.ITypenameContext) int {
	if typename == nil {
		return 0
	}

	// Check if this is a character type
	if typename.Simpletypename() == nil {
		return 0
	}

	simpleType := typename.Simpletypename()

	// Check if it's a character type
	if simpleType.Character() == nil {
		return 0
	}

	character := simpleType.Character()
	if character.Character_c() == nil {
		return 0
	}

	characterC := character.Character_c()

	// Skip VARCHAR - we only check CHAR types
	if characterC.VARCHAR() != nil {
		return 0
	}

	// Skip CHARACTER VARYING, CHAR VARYING, etc.
	// Only check CHAR/CHARACTER/NCHAR without VARYING
	if (characterC.CHARACTER() != nil || characterC.CHAR_P() != nil || characterC.NCHAR() != nil) && characterC.Opt_varying() != nil {
		return 0
	}

	// Now check if it has a size
	if character.Iconst() != nil {
		sizeText := character.Iconst().GetText()
		size, err := strconv.Atoi(sizeText)
		if err != nil {
			return 0
		}
		return size
	}

	return 0
}

func (c *columnMaximumCharacterLengthChecker) addAdvice(tableName, columnName string, line int) {
	c.adviceList = append(c.adviceList, &storepb.Advice{
		Status:  c.level,
		Code:    advisor.CharLengthExceedsLimit.Int32(),
		Title:   c.title,
		Content: fmt.Sprintf("The length of the CHAR column %q in table %s is bigger than %d, please use VARCHAR instead", columnName, tableName, c.maximum),
		StartPosition: &storepb.Position{
			Line:   int32(line),
			Column: 0,
		},
	})
}
