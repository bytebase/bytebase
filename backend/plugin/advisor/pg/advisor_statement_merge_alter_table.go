package pg

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementMergeAlterTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, &StatementMergeAlterTableAdvisor{})
}

// StatementMergeAlterTableAdvisor is the advisor checking for no redundant ALTER TABLE statements.
type StatementMergeAlterTableAdvisor struct {
}

// Check checks for no redundant ALTER TABLE statements.
func (*StatementMergeAlterTableAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementMergeAlterTableRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		tableMap: make(tableMap),
	}

	RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})

	return rule.generateAdvice(), nil
}

type statementMergeAlterTableRule struct {
	OmniBaseRule
	tableMap tableMap
}

type tableMap map[string]tableStatement

type tableStatement struct {
	schema string
	name   string
	count  int
	line   int
}

func (m tableMap) set(schema string, table string, line int) {
	t := tableStatement{
		schema: schema,
		name:   table,
		count:  1,
		line:   line,
	}
	m[t.key()] = t
}

func (m tableMap) add(schema string, table string, line int) {
	if t, exists := m[fmt.Sprintf("%s.%s", schema, table)]; exists {
		t.count++
		t.line = line
		m[t.key()] = t
	}
}

func (t tableStatement) key() string {
	return fmt.Sprintf("%s.%s", t.schema, t.name)
}

func (*statementMergeAlterTableRule) Name() string {
	return string(storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE)
}

func (r *statementMergeAlterTableRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		if n.Relation != nil {
			tableName := omniTableName(n.Relation)
			schema := omniSchemaName(n.Relation)
			if tableName != "" {
				r.tableMap.set(schema, tableName, int(r.ContentEndLine()))
			}
		}
	case *ast.AlterTableStmt:
		if n.Relation != nil {
			tableName := omniTableName(n.Relation)
			schema := omniSchemaName(n.Relation)
			if tableName != "" {
				r.tableMap.add(schema, tableName, int(r.ContentEndLine()))
			}
		}
	default:
	}
}

func (r *statementMergeAlterTableRule) generateAdvice() []*storepb.Advice {
	var adviceList []*storepb.Advice
	var tableList []tableStatement
	for _, table := range r.tableMap {
		tableList = append(tableList, table)
	}
	slices.SortFunc(tableList, func(i, j tableStatement) int {
		if i.line < j.line {
			return -1
		}
		if i.line > j.line {
			return 1
		}
		return 0
	})
	for _, table := range tableList {
		if table.count > 1 {
			adviceList = append(adviceList, &storepb.Advice{
				Status:  r.Level,
				Code:    code.StatementRedundantAlterTable.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf("There are %d statements to modify table `%s`", table.count, table.name),
				StartPosition: &storepb.Position{
					Line:   int32(table.line),
					Column: 0,
				},
			})
		}
	}

	return adviceList
}
