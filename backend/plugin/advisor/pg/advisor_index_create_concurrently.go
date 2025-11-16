package pg

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

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

	rule := &indexCreateConcurrentlyRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		newlyCreatedTables: make(map[string]bool),
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type indexCreateConcurrentlyRule struct {
	BaseRule

	newlyCreatedTables map[string]bool
}

func (*indexCreateConcurrentlyRule) Name() string {
	return "index_create_concurrently"
}

func (r *indexCreateConcurrentlyRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		if c, ok := ctx.(*parser.CreatestmtContext); ok {
			r.handleCreatestmt(c)
		}
	case "Indexstmt":
		if c, ok := ctx.(*parser.IndexstmtContext); ok {
			r.handleIndexstmt(c)
		}
	case "Dropstmt":
		if c, ok := ctx.(*parser.DropstmtContext); ok {
			r.handleDropstmt(c)
		}
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*indexCreateConcurrentlyRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleCreatestmt collects newly created tables
func (r *indexCreateConcurrentlyRule) handleCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Extract table name
	qualifiedNames := ctx.AllQualified_name()
	if len(qualifiedNames) > 0 {
		tableName := extractTableName(qualifiedNames[0])
		if tableName != "" {
			r.newlyCreatedTables[tableName] = true
		}
	}
}

// handleIndexstmt checks CREATE INDEX statements
func (r *indexCreateConcurrentlyRule) handleIndexstmt(ctx *parser.IndexstmtContext) {
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
			if r.newlyCreatedTables[tableName] {
				return
			}
		}

		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.CreateIndexUnconcurrently.Int32(),
			Title:   r.title,
			Content: "Creating indexes will block writes on the table, unless use CONCURRENTLY",
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// handleDropstmt checks DROP INDEX statements
func (r *indexCreateConcurrentlyRule) handleDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is a DROP INDEX statement
	if ctx.INDEX() != nil {
		// Check if CONCURRENTLY is used
		if ctx.CONCURRENTLY() == nil {
			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    code.DropIndexUnconcurrently.Int32(),
				Title:   r.title,
				Content: "Droping indexes will block writes on the table, unless use CONCURRENTLY",
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}
