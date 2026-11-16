package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

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
//
// The rule guards the net change of a batch by comparing the metadata before
// and after the walkthrough: a table created in the batch must end with a
// PRIMARY KEY, and a table that had one must still have one. Pre-existing
// tables without a PRIMARY KEY are out of scope, so unrelated changes to them
// are not blocked. The judgment lives in omni (pg/v1review.TableRequirePK);
// this adapter passes both snapshots and renders the findings as advice.
type TableRequirePKAdvisor struct {
}

// Check parses the given statement and checks for errors.
func (*TableRequirePKAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var nodes []ast.Node
	var stmts []base.ParsedStatement
	var startLines []int
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		nodes = append(nodes, node)
		stmts = append(stmts, stmt)
		startLines = append(startLines, stmt.BaseLine()+int((&BaseRule{StmtText: stmt.Text}).ContentStartLine()))
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range pgv1review.TableRequirePK(nodes, startLines, requirePKSchema(checkCtx.OriginalMetadata), requirePKSchema(checkCtx.FinalMetadata)) {
		if f.Code != v1review.TablePrimaryKeyMissing {
			continue
		}
		content := fmt.Sprintf("Table %q.%q requires PRIMARY KEY", f.Value, f.Name)
		line := 0
		if f.Statement >= 0 {
			line = startLines[f.Statement]
			if text := strings.TrimRight(strings.TrimSpace(stmts[f.Statement].Text), ";"); text != "" {
				content = fmt.Sprintf("%s, related statement: %q", content, text)
			}
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Code:    code.TableNoPK.Int32(),
			Title:   title,
			Content: content,
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
	return adviceList, nil
}

func requirePKSchema(m *model.DatabaseMetadata) *v1review.Schema {
	if m == nil {
		return nil
	}
	return &v1review.Schema{Metadata: m.GetProto(), ObjectCaseSensitive: m.GetIsObjectCaseSensitive()}
}
