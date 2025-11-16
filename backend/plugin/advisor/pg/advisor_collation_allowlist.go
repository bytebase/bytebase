package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
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

	allowlist := make(map[string]bool)
	for _, collation := range payload.List {
		allowlist[collation] = true
	}

	rule := &collationAllowlistRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		allowlist: allowlist,
		tokens:    tree.Tokens,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type collationAllowlistRule struct {
	BaseRule

	allowlist map[string]bool
	tokens    *antlr.CommonTokenStream
}

func (*collationAllowlistRule) Name() string {
	return "collation-allowlist"
}

func (r *collationAllowlistRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		return r.handleCreatestmt(ctx.(*parser.CreatestmtContext))
	case "Altertablestmt":
		return r.handleAltertablestmt(ctx.(*parser.AltertablestmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*collationAllowlistRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleCreatestmt handles CREATE TABLE statements
func (r *collationAllowlistRule) handleCreatestmt(ctx *parser.CreatestmtContext) error {
	if !isTopLevel(ctx.GetParent()) {
		return nil
	}

	// Extract collations from column definitions
	if ctx.Opttableelementlist() != nil {
		r.checkTableElementList(ctx.Opttableelementlist(), ctx)
	}
	return nil
}

// handleAltertablestmt handles ALTER TABLE statements
func (r *collationAllowlistRule) handleAltertablestmt(ctx *parser.AltertablestmtContext) error {
	if !isTopLevel(ctx.GetParent()) {
		return nil
	}

	// Check ALTER TABLE ADD COLUMN
	if ctx.Alter_table_cmds() != nil {
		r.checkAlterTableCmds(ctx.Alter_table_cmds(), ctx)
	}
	return nil
}

func (r *collationAllowlistRule) checkTableElementList(listCtx parser.IOpttableelementlistContext, stmtCtx antlr.ParserRuleContext) {
	if listCtx == nil || listCtx.Tableelementlist() == nil {
		return
	}

	allElements := listCtx.Tableelementlist().AllTableelement()
	for _, elem := range allElements {
		if elem.ColumnDef() != nil {
			r.checkColumnDef(elem.ColumnDef(), stmtCtx)
		}
	}
}

func (r *collationAllowlistRule) checkColumnDef(colDef parser.IColumnDefContext, stmtCtx antlr.ParserRuleContext) {
	if colDef == nil || colDef.Colquallist() == nil {
		return
	}

	// Check column constraints for COLLATE clause
	// colquallist -> colconstraint* -> COLLATE any_name
	allConstraints := colDef.Colquallist().AllColconstraint()
	for _, constraint := range allConstraints {
		// Check if this constraint is a COLLATE constraint
		if constraint.COLLATE() != nil && constraint.Any_name() != nil {
			collName := r.extractCollationNameFromAnyName(constraint.Any_name())
			if collName != "" && !r.allowlist[collName] {
				r.addAdvice(collName, stmtCtx)
			}
		}
	}
}

func (r *collationAllowlistRule) checkAlterTableCmds(cmds parser.IAlter_table_cmdsContext, stmtCtx antlr.ParserRuleContext) {
	if cmds == nil {
		return
	}

	allCmds := cmds.AllAlter_table_cmd()
	for _, cmd := range allCmds {
		// ADD COLUMN
		if cmd.ADD_P() != nil && cmd.ColumnDef() != nil {
			r.checkColumnDef(cmd.ColumnDef(), stmtCtx)
		}

		// ALTER COLUMN TYPE with COLLATE
		// Check if this is ALTER COLUMN ... TYPE ...
		// Grammar: ALTER opt_column? colid opt_set_data? TYPE_P typename opt_collate_clause? alter_using?
		if cmd.ALTER() != nil && cmd.TYPE_P() != nil && cmd.Opt_collate_clause() != nil && cmd.Opt_collate_clause().Any_name() != nil {
			collName := r.extractCollationNameFromAnyName(cmd.Opt_collate_clause().Any_name())
			if collName != "" && !r.allowlist[collName] {
				r.addAdvice(collName, stmtCtx)
			}
		}
	}
}

func (*collationAllowlistRule) extractCollationNameFromAnyName(anyName parser.IAny_nameContext) string {
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

func (r *collationAllowlistRule) addAdvice(collation string, ctx antlr.ParserRuleContext) {
	text := r.tokens.GetTextFromRuleContext(ctx)

	r.AddAdvice(&storepb.Advice{
		Status:  r.level,
		Code:    code.DisabledCollation.Int32(),
		Title:   r.title,
		Content: fmt.Sprintf("Use disabled collation \"%s\", related statement \"%s\"", collation, text),
		StartPosition: &storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: 0,
		},
	})
}
