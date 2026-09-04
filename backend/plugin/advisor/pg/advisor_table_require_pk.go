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
//
// The rule only guards the net change of a batch: a table created in the batch
// must end with a PRIMARY KEY, and a table that had a PRIMARY KEY before the
// batch must still have one afterwards. ALTER TABLE on a pre-existing table
// without a PRIMARY KEY is out of scope so unrelated changes are not blocked.
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
		originalMetadata: checkCtx.OriginalMetadata,
		finalMetadata:    checkCtx.FinalMetadata,
		searchPath:       []string{"public"},
		tableMentions:    make(map[string]*tableMention),
	}
	if checkCtx.OriginalMetadata != nil {
		// Resolve $user against SessionUser so unqualified names land in the
		// same schema the walkthrough uses when building FinalMetadata.
		if sp := checkCtx.OriginalMetadata.GetSearchPathForCurrentUser(checkCtx.SessionUser); len(sp) > 0 {
			rule.searchPath = sp
		}
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
	originalMetadata *model.DatabaseMetadata
	finalMetadata    *model.DatabaseMetadata
	// searchPath resolves unqualified table names; never empty.
	searchPath []string

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
	schema := r.resolveSchema(n.Relation.Schemaname, tableName)

	key := fmt.Sprintf("%s.%s", schema, tableName)
	r.tableMentions[key] = &tableMention{
		startLine: int(r.ContentStartLine()) + r.BaseLine,
		text:      r.TrimmedStmtText(),
	}
}

// handleAlterTableStmt records ALTER TABLE statements on tables that had a
// PRIMARY KEY before this batch. Tables created in this batch are already
// tracked by handleCreateStmt.
func (r *tableRequirePKRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}
	schema := r.resolveSchema(n.Relation.Schemaname, tableName)

	key := fmt.Sprintf("%s.%s", schema, tableName)
	if _, tracked := r.tableMentions[key]; !tracked {
		if table := r.originalTable(schema, tableName); table == nil || table.GetPrimaryKey() == nil {
			return
		}
	}
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

	if n.Objects == nil {
		return
	}
	for _, item := range n.Objects.Items {
		list, ok := item.(*ast.List)
		if !ok {
			continue
		}
		var parts []string
		for _, nameItem := range list.Items {
			if s, ok := nameItem.(*ast.String); ok {
				parts = append(parts, s.Str)
			}
		}
		var schema, tableName string
		switch len(parts) {
		case 1:
			tableName = parts[0]
		case 2:
			schema, tableName = parts[0], parts[1]
		default:
			continue
		}
		delete(r.tableMentions, fmt.Sprintf("%s.%s", r.resolveSchema(schema, tableName), tableName))
	}
}

// resolveSchema mirrors PostgreSQL name resolution. A qualified name is taken
// as-is. An unqualified name maps to the first search_path schema that already
// holds the table, either created earlier in this batch or present in the
// original metadata; otherwise to the first search_path schema, which is
// where CREATE TABLE places a new table.
func (r *tableRequirePKRule) resolveSchema(schema, tableName string) string {
	if schema != "" {
		return schema
	}
	for _, candidate := range r.searchPath {
		if _, ok := r.tableMentions[fmt.Sprintf("%s.%s", candidate, tableName)]; ok {
			return candidate
		}
		if r.originalTable(candidate, tableName) != nil {
			return candidate
		}
	}
	return r.searchPath[0]
}

// originalTable returns the table from the metadata before this batch, or nil.
func (r *tableRequirePKRule) originalTable(schemaName, tableName string) *model.TableMetadata {
	if r.originalMetadata == nil {
		return nil
	}
	return r.originalMetadata.GetSchemaMetadata(schemaName).GetTable(tableName)
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
