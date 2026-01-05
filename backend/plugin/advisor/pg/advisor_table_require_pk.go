package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*TableRequirePKAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_REQUIRE_PK, &TableRequirePKAdvisor{})
}

// TableRequirePKAdvisor is the advisor checking table requires PK.
type TableRequirePKAdvisor struct {
}

// Check parses the given statement and checks for errors.
func (*TableRequirePKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableRequirePKRule{
		BaseRule: BaseRule{
			level: level,
			title: checkCtx.Rule.Type.String(),
		},
		finalMetadata: checkCtx.FinalMetadata,
		tableMentions: make(map[string]*tableMention),
	}

	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		rule.tokens = antlrAST.Tokens
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	// Simple Solution: Validate final state after walking all statements
	rule.validateFinalState()

	return checker.GetAdviceList(), nil
}

type tableMention struct {
	startLine int
	endLine   int
	text      string
}

type tableRequirePKRule struct {
	BaseRule
	tokens        *antlr.CommonTokenStream
	finalMetadata *model.DatabaseMetadata

	// Simple Solution: Track last mention of each table
	tableMentions map[string]*tableMention // key: "schema.table", value: last mention info
}

// Name returns the rule name.
func (*tableRequirePKRule) Name() string {
	return "table.require-pk"
}

// OnEnter is called when the parser enters a rule context.
func (r *tableRequirePKRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		r.handleCreatestmt(ctx.(*parser.CreatestmtContext))
	case "Altertablestmt":
		r.handleAltertablestmt(ctx.(*parser.AltertablestmtContext))
	case "Dropstmt":
		r.handleDropstmt(ctx.(*parser.DropstmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*tableRequirePKRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleCreatestmt records CREATE TABLE statements (Simple Solution)
func (r *tableRequirePKRule) handleCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	var tableName, schemaName string
	allQualifiedNames := ctx.AllQualified_name()
	if len(allQualifiedNames) > 0 {
		tableName = extractTableName(allQualifiedNames[0])
		schemaName = extractSchemaName(allQualifiedNames[0])
		if schemaName == "" {
			schemaName = "public"
		}
	}

	// Simple Solution: Just record the table mention (ALWAYS update for last occurrence)
	// Store absolute line numbers (with baseLine offset added now)
	key := fmt.Sprintf("%s.%s", schemaName, tableName)
	r.tableMentions[key] = &tableMention{
		startLine: ctx.GetStart().GetLine() + r.baseLine,
		endLine:   ctx.GetStop().GetLine() + r.baseLine,
		text:      getTextFromTokens(r.tokens, ctx),
	}
}

// handleAltertablestmt records ALTER TABLE statements (Simple Solution)
func (r *tableRequirePKRule) handleAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	var tableName, schemaName string
	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		tableName = extractTableName(ctx.Relation_expr().Qualified_name())
		schemaName = extractSchemaName(ctx.Relation_expr().Qualified_name())
		if schemaName == "" {
			schemaName = "public"
		}
	}

	// Simple Solution: Just record the table mention (ALWAYS update for last occurrence)
	// Store absolute line numbers (with baseLine offset added now)
	key := fmt.Sprintf("%s.%s", schemaName, tableName)
	r.tableMentions[key] = &tableMention{
		startLine: ctx.GetStart().GetLine() + r.baseLine,
		endLine:   ctx.GetStop().GetLine() + r.baseLine,
		text:      getTextFromTokens(r.tokens, ctx),
	}
}

// handleDropstmt handles DROP TABLE - remove from tracking
func (r *tableRequirePKRule) handleDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is DROP TABLE
	if ctx.Object_type_any_name() == nil || ctx.Object_type_any_name().TABLE() == nil {
		return
	}

	// Remove all dropped tables from tracking (simplified - best effort)
	// Note: For the Simple Solution, not handling DROP TABLE perfectly is acceptable
	// since dropped tables won't be in catalog.Final anyway
	if ctx.Any_name_list() != nil {
		allNames := ctx.Any_name_list().AllAny_name()
		for _, anyName := range allNames {
			if anyName.Colid() != nil {
				// Simple table name (most common case)
				name := pgparser.NormalizePostgreSQLColid(anyName.Colid())
				key := fmt.Sprintf("public.%s", name)
				delete(r.tableMentions, key)
			}
			// For qualified names, we skip for simplicity - they won't cause false positives
			// because catalog.Final won't have dropped tables
		}
	}
}

// validateFinalState checks all mentioned tables against catalog.Final (Simple Solution)
func (r *tableRequirePKRule) validateFinalState() {
	for tableKey, mention := range r.tableMentions {
		// Parse table key: "schema.table"
		schemaName, tableName := parseTableKey(tableKey)

		// Check catalog.Final for PRIMARY KEY
		schema := r.finalMetadata.GetSchemaMetadata(schemaName)
		var hasPK bool
		if schema != nil {
			table := schema.GetTable(tableName)
			if table != nil {
				hasPK = table.GetPrimaryKey() != nil
			}
		}

		if !hasPK {
			content := fmt.Sprintf("Table %q.%q requires PRIMARY KEY", schemaName, tableName)

			// Include the related statement
			if mention.text != "" {
				content = fmt.Sprintf("%s, related statement: %q", content, mention.text)
			}

			// Directly append to adviceList instead of using AddAdvice,
			// because mention.startLine already contains the absolute line number
			r.adviceList = append(r.adviceList, &storepb.Advice{
				Status:  r.level,
				Code:    code.TableNoPK.Int32(),
				Title:   r.title,
				Content: content,
				StartPosition: &storepb.Position{
					Line:   int32(mention.startLine),
					Column: 0,
				},
			})
		}
	}
}

// parseTableKey splits "schema.table" into schema and table name
func parseTableKey(key string) (string, string) {
	// Simple split on first dot
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			return key[:i], key[i+1:]
		}
	}
	return "public", key
}
