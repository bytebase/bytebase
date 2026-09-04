package pg

import (
	"context"
	"fmt"
	"slices"

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
// The rule guards the net change of a batch by comparing the metadata before
// and after the walkthrough: a table created in the batch must end with a
// PRIMARY KEY, and a table that had one must still have one. Pre-existing
// tables without a PRIMARY KEY are out of scope, so unrelated changes to them
// are not blocked. Name resolution (search_path, SET ROLE, drops) is left to
// the walkthrough; statements are only tracked to attribute the advice.
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
		tableMentions:    make(map[string]*tableMention),
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
	// created is set when the batch contains a CREATE TABLE for this name.
	created bool
}

type tableRequirePKRule struct {
	OmniBaseRule
	originalMetadata *model.DatabaseMetadata
	finalMetadata    *model.DatabaseMetadata

	// tableMentions records the last CREATE/ALTER TABLE per name for advice
	// attribution. The key is "schema.table" with the schema as written, so an
	// unqualified name is keyed ".table".
	tableMentions map[string]*tableMention
}

// Name returns the rule name.
func (*tableRequirePKRule) Name() string {
	return "table.require-pk"
}

// OnStatement is called for each top-level statement AST node.
func (r *tableRequirePKRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.recordMention(n.Relation, true)
	case *ast.AlterTableStmt:
		r.recordMention(n.Relation, false)
	default:
	}
}

func (r *tableRequirePKRule) recordMention(rv *ast.RangeVar, created bool) {
	tableName := omniTableName(rv)
	if tableName == "" {
		return
	}
	key := mentionKey(rv.Schemaname, tableName)
	if prev := r.tableMentions[key]; prev != nil && prev.created {
		created = true
	}
	r.tableMentions[key] = &tableMention{
		startLine: int(r.ContentStartLine()) + r.BaseLine,
		text:      r.TrimmedStmtText(),
		created:   created,
	}
}

func mentionKey(schema, table string) string {
	return schema + "." + table
}

// mentionFor returns the mention for a table in the final metadata, trying
// the qualified name first and then the unqualified one.
func (r *tableRequirePKRule) mentionFor(schema, table string) *tableMention {
	if m := r.tableMentions[mentionKey(schema, table)]; m != nil {
		return m
	}
	return r.tableMentions[mentionKey("", table)]
}

// originalTable returns the table from the metadata before this batch, or nil.
func (r *tableRequirePKRule) originalTable(schemaName, tableName string) *model.TableMetadata {
	if r.originalMetadata == nil {
		return nil
	}
	return r.originalMetadata.GetSchemaMetadata(schemaName).GetTable(tableName)
}

// validateFinalState flags tables that end the batch without a PRIMARY KEY and
// either were created by the batch or had a PRIMARY KEY before it.
func (r *tableRequirePKRule) validateFinalState() {
	if r.finalMetadata == nil {
		return
	}
	schemaNames := r.finalMetadata.ListSchemaNames()
	slices.Sort(schemaNames)
	for _, schemaName := range schemaNames {
		schema := r.finalMetadata.GetSchemaMetadata(schemaName)
		tableNames := schema.ListTableNames()
		slices.Sort(tableNames)
		for _, tableName := range tableNames {
			if schema.GetTable(tableName).GetPrimaryKey() != nil {
				continue
			}
			mention := r.mentionFor(schemaName, tableName)
			original := r.originalTable(schemaName, tableName)
			switch {
			case original == nil:
				// New in this batch: only tables the batch created are in scope.
				if mention == nil || !mention.created {
					continue
				}
			case original.GetPrimaryKey() == nil:
				// Pre-existing table without a PRIMARY KEY is out of scope.
				continue
			default:
			}
			r.addAdvice(schemaName, tableName, mention)
		}
	}
}

func (r *tableRequirePKRule) addAdvice(schemaName, tableName string, mention *tableMention) {
	content := fmt.Sprintf("Table %q.%q requires PRIMARY KEY", schemaName, tableName)
	line := 0
	if mention != nil {
		line = mention.startLine
		if mention.text != "" {
			content = fmt.Sprintf("%s, related statement: %q", content, mention.text)
		}
	}
	r.AddAdviceAbsolute(&storepb.Advice{
		Status:  r.Level,
		Code:    code.TableNoPK.Int32(),
		Title:   r.Title,
		Content: content,
		StartPosition: &storepb.Position{
			Line:   int32(line),
			Column: 0,
		},
	})
}
