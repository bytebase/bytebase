package pgantlr

import (
	"context"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*IndexConcurrentlyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleCreateIndexConcurrently, &IndexConcurrentlyAdvisor{})
}

// IndexConcurrentlyAdvisor is the advisor checking for to create index concurrently.
type IndexConcurrentlyAdvisor struct {
}

// Check checks for to create index concurrently.
func (*IndexConcurrentlyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &indexCreateConcurrentlyChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		newlyCreatedTables:           make(map[string]bool),
	}

	// First pass: collect all newly created tables
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type indexCreateConcurrentlyChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList         []*storepb.Advice
	level              storepb.Advice_Status
	title              string
	newlyCreatedTables map[string]bool
}

// EnterCreatestmt collects newly created tables
func (c *indexCreateConcurrentlyChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Extract table name
	qualifiedNames := ctx.AllQualified_name()
	if len(qualifiedNames) > 0 {
		tableName := extractTableName(qualifiedNames[0])
		if tableName != "" {
			c.newlyCreatedTables[tableName] = true
		}
	}
}

// EnterIndexstmt checks CREATE INDEX statements
func (c *indexCreateConcurrentlyChecker) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if CONCURRENTLY is used (via Opt_concurrently)
	hasConcurrently := ctx.Opt_concurrently() != nil && ctx.Opt_concurrently().CONCURRENTLY() != nil

	if !hasConcurrently {
		// Check if the index is being created on a newly created table
		if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
			tableName := extractTableName(ctx.Relation_expr().Qualified_name())
			// Skip the check if the table is newly created
			if c.newlyCreatedTables[tableName] {
				return
			}
		}

		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.CreateIndexUnconcurrently.Int32(),
			Title:   c.title,
			Content: "Creating indexes will block writes on the table, unless use CONCURRENTLY",
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// EnterDropstmt checks DROP INDEX statements
func (c *indexCreateConcurrentlyChecker) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is a DROP INDEX statement
	if ctx.INDEX() != nil {
		// Check if CONCURRENTLY is used
		if ctx.CONCURRENTLY() == nil {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  c.level,
				Code:    advisor.DropIndexUnconcurrently.Int32(),
				Title:   c.title,
				Content: "Droping indexes will block writes on the table, unless use CONCURRENTLY",
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}
