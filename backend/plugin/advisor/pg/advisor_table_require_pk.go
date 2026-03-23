package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
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
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		finalMetadata: checkCtx.FinalMetadata,
		tableMentions: make(map[string]*tableMention),
	}

	// Manually iterate statements instead of using RunOmniRules because
	// validateFinalState must be called AFTER all statements have been processed.
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

	rule.validateFinalState()

	return rule.GetAdviceList(), nil
}

type tableMention struct {
	startLine int
	text      string
}

type tableRequirePKRule struct {
	OmniBaseRule
	finalMetadata *model.DatabaseMetadata

	// Track last mention of each table
	tableMentions map[string]*tableMention // key: "schema.table", value: last mention info
}

// Name returns the rule name.
func (*tableRequirePKRule) Name() string {
	return "table.require-pk"
}

// OnStatement is called for each top-level statement AST node.
func (r *tableRequirePKRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	case *ast.DropStmt:
		r.handleDropStmt(n)
	default:
	}
}

// handleCreateStmt records CREATE TABLE statements.
func (r *tableRequirePKRule) handleCreateStmt(n *ast.CreateStmt) {
	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}
	schema := omniSchemaName(n.Relation)

	key := fmt.Sprintf("%s.%s", schema, tableName)
	r.tableMentions[key] = &tableMention{
		startLine: int(r.ContentStartLine()) + r.BaseLine,
		text:      r.TrimmedStmtText(),
	}
}

// handleAlterTableStmt records ALTER TABLE statements.
func (r *tableRequirePKRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}
	schema := omniSchemaName(n.Relation)

	key := fmt.Sprintf("%s.%s", schema, tableName)
	r.tableMentions[key] = &tableMention{
		startLine: int(r.ContentStartLine()) + r.BaseLine,
		text:      r.TrimmedStmtText(),
	}
}

// handleDropStmt handles DROP TABLE - remove from tracking.
func (r *tableRequirePKRule) handleDropStmt(n *ast.DropStmt) {
	if n.RemoveType != int(ast.OBJECT_TABLE) {
		return
	}

	for _, obj := range omniDropObjects(n) {
		key := fmt.Sprintf("%s.%s", obj[0], obj[1])
		delete(r.tableMentions, key)
	}
}

// validateFinalState checks all mentioned tables against FinalMetadata for PRIMARY KEY.
func (r *tableRequirePKRule) validateFinalState() {
	for tableKey, mention := range r.tableMentions {
		schemaName, tableName := parseTableKey(tableKey)

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

			if mention.text != "" {
				content = fmt.Sprintf("%s, related statement: %q", content, mention.text)
			}

			r.AddAdviceAbsolute(&storepb.Advice{
				Status:  r.Level,
				Code:    code.TableNoPK.Int32(),
				Title:   r.Title,
				Content: content,
				StartPosition: &storepb.Position{
					Line:   int32(mention.startLine),
					Column: 0,
				},
			})
		}
	}
}

// parseTableKey splits "schema.table" into schema and table name.
func parseTableKey(key string) (string, string) {
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			return key[:i], key[i+1:]
		}
	}
	return "public", key
}
