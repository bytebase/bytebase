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
// the walkthrough; statements are only tracked to tell created tables from
// pre-existing ones and to attribute the advice.
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
	// dropped is set when the batch contains a DROP TABLE for this name.
	dropped bool
}

type tableRequirePKRule struct {
	OmniBaseRule
	originalMetadata *model.DatabaseMetadata
	finalMetadata    *model.DatabaseMetadata

	// tableMentions tracks, per table name, what the batch did to the table
	// and the last CREATE/ALTER statement for advice attribution. Keys ignore
	// schema qualification; the final metadata supplies the schema.
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
		if m := r.recordMention(omniTableName(n.Relation)); m != nil {
			m.created = true
		}
	case *ast.AlterTableStmt:
		r.recordMention(omniTableName(n.Relation))
	case *ast.DropStmt:
		if n.RemoveType != int(ast.OBJECT_TABLE) {
			return
		}
		for _, obj := range omniDropObjects(n) {
			r.mention(obj[1]).dropped = true
		}
	case *ast.RenameStmt:
		if n.RenameType != ast.OBJECT_TABLE || n.Relation == nil {
			return
		}
		r.renameMention(n.Relation.Relname, n.Newname)
	default:
	}
}

func (r *tableRequirePKRule) mention(tableName string) *tableMention {
	m := r.tableMentions[tableName]
	if m == nil {
		m = &tableMention{}
		r.tableMentions[tableName] = m
	}
	return m
}

// recordMention points the table's advice attribution at the current statement.
func (r *tableRequirePKRule) recordMention(tableName string) *tableMention {
	if tableName == "" {
		return nil
	}
	m := r.mention(tableName)
	m.startLine = int(r.ContentStartLine()) + r.BaseLine
	m.text = r.TrimmedStmtText()
	return m
}

// renameMention carries the batch state of oldName over to newName.
func (r *tableRequirePKRule) renameMention(oldName, newName string) {
	if oldName == "" || newName == "" {
		return
	}
	old := r.tableMentions[oldName]
	delete(r.tableMentions, oldName)
	m := r.recordMention(newName)
	if old != nil {
		m.created = m.created || old.created
		m.dropped = m.dropped || old.dropped
	}
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
			mention := r.tableMentions[tableName]
			original := r.originalTable(schemaName, tableName)
			switch {
			case original == nil:
				// New in this batch: only tables the batch created are in scope.
				if mention == nil || !mention.created {
					continue
				}
			case original.GetPrimaryKey() == nil:
				// A pre-existing table without a PRIMARY KEY is out of scope
				// unless the batch dropped and recreated it.
				if mention == nil || !mention.created || !mention.dropped {
					continue
				}
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
