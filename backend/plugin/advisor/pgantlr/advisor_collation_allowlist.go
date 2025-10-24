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
	_ advisor.Advisor = (*CollationAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleCollationAllowlist, &CollationAllowlistAdvisor{})
}

// CollationAllowlistAdvisor is the advisor checking for collation allowlist.
type CollationAllowlistAdvisor struct {
}

// Check checks for collation allowlist.
func (*CollationAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &collationAllowlistChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		allowlist:                    make(map[string]bool),
		tokens:                       tree.Tokens,
	}
	for _, collation := range payload.List {
		checker.allowlist[collation] = true
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type collationAllowlistChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	allowlist  map[string]bool
	tokens     *antlr.CommonTokenStream
}

// EnterCreatestmt handles CREATE TABLE statements
func (c *collationAllowlistChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Extract collations from column definitions
	if ctx.Opttableelementlist() != nil {
		c.checkTableElementList(ctx.Opttableelementlist(), ctx)
	}
}

// EnterAltertablestmt handles ALTER TABLE statements
func (c *collationAllowlistChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check ALTER TABLE ADD COLUMN
	if ctx.Alter_table_cmds() != nil {
		c.checkAlterTableCmds(ctx.Alter_table_cmds(), ctx)
	}
}

func (c *collationAllowlistChecker) checkTableElementList(listCtx parser.IOpttableelementlistContext, stmtCtx antlr.ParserRuleContext) {
	if listCtx == nil || listCtx.Tableelementlist() == nil {
		return
	}

	allElements := listCtx.Tableelementlist().AllTableelement()
	for _, elem := range allElements {
		if elem.ColumnDef() != nil {
			c.checkColumnDef(elem.ColumnDef(), stmtCtx)
		}
	}
}

func (c *collationAllowlistChecker) checkColumnDef(colDef parser.IColumnDefContext, stmtCtx antlr.ParserRuleContext) {
	if colDef == nil || colDef.Colquallist() == nil {
		return
	}

	// Check column constraints for COLLATE clause
	// colquallist -> colconstraint* -> COLLATE any_name
	allConstraints := colDef.Colquallist().AllColconstraint()
	for _, constraint := range allConstraints {
		// Check if this constraint is a COLLATE constraint
		if constraint.COLLATE() != nil && constraint.Any_name() != nil {
			collName := c.extractCollationNameFromAnyName(constraint.Any_name())
			if collName != "" && !c.allowlist[collName] {
				c.addAdvice(collName, stmtCtx)
			}
		}
	}
}

func (c *collationAllowlistChecker) checkAlterTableCmds(cmds parser.IAlter_table_cmdsContext, stmtCtx antlr.ParserRuleContext) {
	if cmds == nil {
		return
	}

	allCmds := cmds.AllAlter_table_cmd()
	for _, cmd := range allCmds {
		// ADD COLUMN
		if cmd.ADD_P() != nil && cmd.ColumnDef() != nil {
			c.checkColumnDef(cmd.ColumnDef(), stmtCtx)
		}

		// ALTER COLUMN TYPE with COLLATE
		// Check if this is ALTER COLUMN ... TYPE ...
		// Grammar: ALTER opt_column? colid opt_set_data? TYPE_P typename opt_collate_clause? alter_using?
		if cmd.ALTER() != nil && cmd.TYPE_P() != nil && cmd.Opt_collate_clause() != nil && cmd.Opt_collate_clause().Any_name() != nil {
			collName := c.extractCollationNameFromAnyName(cmd.Opt_collate_clause().Any_name())
			if collName != "" && !c.allowlist[collName] {
				c.addAdvice(collName, stmtCtx)
			}
		}
	}
}

func (*collationAllowlistChecker) extractCollationNameFromAnyName(anyName parser.IAny_nameContext) string {
	if anyName == nil {
		return ""
	}

	// any_name can be: colid | colid attrs
	// For collation names like "unknown" or "utf8mb4_0900_ai_ci", we need the text
	// Handle quoted strings by getting the full text and removing quotes if present
	text := anyName.GetText()

	// Remove leading/trailing quotes if present
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		return text[1 : len(text)-1]
	}

	return text
}

func (c *collationAllowlistChecker) addAdvice(collation string, ctx antlr.ParserRuleContext) {
	text := c.tokens.GetTextFromRuleContext(ctx)

	c.adviceList = append(c.adviceList, &storepb.Advice{
		Status:  c.level,
		Code:    advisor.DisabledCollation.Int32(),
		Title:   c.title,
		Content: fmt.Sprintf("Use disabled collation \"%s\", related statement \"%s\"", collation, text),
		StartPosition: &storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: int32(ctx.GetStart().GetColumn()),
		},
	})
}
