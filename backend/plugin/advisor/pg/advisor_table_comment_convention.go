package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*TableCommentConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_COMMENT, &TableCommentConventionAdvisor{})
}

// TableCommentConventionAdvisor is the advisor checking for table comment convention.
type TableCommentConventionAdvisor struct {
}

func (*TableCommentConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	commentPayload := checkCtx.Rule.GetCommentConventionPayload()

	rule := &tableCommentConventionRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		payload:       commentPayload,
		createdTables: make(map[string]*tableInfo),
		tableComments: make(map[string]*tableCommentInfo),
	}

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetStatement(stmt.BaseLine(), stmt.Text)
		rule.OnStatement(node)
	}

	return rule.generateAdvice(), nil
}

type tableInfo struct {
	schema      string
	tableName   string
	displayName string
	line        int32
}

type tableCommentInfo struct {
	comment string
	line    int32
}

type tableCommentConventionRule struct {
	OmniBaseRule

	payload       *storepb.SQLReviewRule_CommentConventionRulePayload
	createdTables map[string]*tableInfo
	tableComments map[string]*tableCommentInfo
}

func (*tableCommentConventionRule) Name() string {
	return "table-comment-convention"
}

func (r *tableCommentConventionRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.CommentStmt:
		r.handleCommentStmt(n)
	default:
	}
}

func (r *tableCommentConventionRule) handleCreateStmt(n *ast.CreateStmt) {
	if n.Relation == nil {
		return
	}

	tableName := n.Relation.Relname
	schemaName := n.Relation.Schemaname
	if schemaName == "" {
		schemaName = "public"
	}

	tableKey := fmt.Sprintf("%s.%s", schemaName, tableName)
	displayName := tableName
	if schemaName != "public" {
		displayName = fmt.Sprintf("%s.%s", schemaName, tableName)
	}

	r.createdTables[tableKey] = &tableInfo{
		schema:      schemaName,
		tableName:   tableName,
		displayName: displayName,
		line:        r.absoluteLine(),
	}
}

func (r *tableCommentConventionRule) handleCommentStmt(n *ast.CommentStmt) {
	if n.Objtype != ast.OBJECT_TABLE {
		return
	}

	var schemaName, tableName string
	switch obj := n.Object.(type) {
	case *ast.List:
		items := obj.Items
		switch len(items) {
		case 1:
			schemaName = "public"
			tableName = omniStringVal(items[0])
		case 2:
			schemaName = omniStringVal(items[0])
			tableName = omniStringVal(items[1])
		default:
			return
		}
	case *ast.String:
		schemaName = "public"
		tableName = obj.Str
	default:
		return
	}

	tableKey := fmt.Sprintf("%s.%s", schemaName, tableName)
	r.tableComments[tableKey] = &tableCommentInfo{
		comment: n.Comment,
		line:    r.absoluteLine(),
	}
}

// absoluteLine returns the absolute 1-based line number for the current statement.
func (r *tableCommentConventionRule) absoluteLine() int32 {
	return r.ContentStartLine() + int32(r.BaseLine)
}

func (r *tableCommentConventionRule) generateAdvice() []*storepb.Advice {
	var adviceList []*storepb.Advice

	for tableKey, ti := range r.createdTables {
		tc, hasComment := r.tableComments[tableKey]

		if !hasComment || tc.comment == "" {
			if r.payload.Required {
				adviceList = append(adviceList, &storepb.Advice{
					Status:  r.Level,
					Code:    code.CommentEmpty.Int32(),
					Title:   r.Title,
					Content: fmt.Sprintf("Comment is required for table `%s`", ti.displayName),
					StartPosition: &storepb.Position{
						Line:   ti.line,
						Column: 0,
					},
				})
			}
		} else {
			if r.payload.MaxLength > 0 && int32(len(tc.comment)) > r.payload.MaxLength {
				adviceList = append(adviceList, &storepb.Advice{
					Status:  r.Level,
					Code:    code.CommentTooLong.Int32(),
					Title:   r.Title,
					Content: fmt.Sprintf("Table `%s` comment is too long. The length of comment should be within %d characters", ti.displayName, r.payload.MaxLength),
					StartPosition: &storepb.Position{
						Line:   tc.line,
						Column: 0,
					},
				})
			}
		}
	}

	return adviceList
}
