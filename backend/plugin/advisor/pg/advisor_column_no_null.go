package pg

import (
	"context"
	"fmt"
	"strings"

	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_NO_NULL, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value. The
// judgment lives in omni (pg/v1review.ColumnNoNull); this adapter passes the
// statements, their text and the pre-change schema, and renders the findings.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	texts := make([]string, len(stmts))
	for i, stmt := range stmts {
		texts[i] = stmt.Text
	}
	var schema v1review.Schema
	if md := checkCtx.OriginalMetadata; md != nil {
		schema = v1review.Schema{Metadata: md.GetProto(), ObjectCaseSensitive: md.GetIsObjectCaseSensitive()}
	}

	var adviceList []*storepb.Advice
	for _, f := range pgv1review.ColumnNoNull(nodes, texts, schema) {
		stmt := stmts[f.Statement]
		var line int32
		if f.Start == 0 && f.End == 0 {
			line = (&BaseRule{StmtText: stmt.Text}).ContentStartLine()
		} else {
			line = int32(1 + strings.Count(stmt.Text[:f.Start], "\n"))
		}
		table := fmt.Sprintf("%q.%q", f.Schema, f.Table)
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Code:    code.ColumnCannotNull.Int32(),
			Title:   checkCtx.Rule.Type.String(),
			Content: fmt.Sprintf("Column %q in %s cannot have NULL value", f.Name, table),
			StartPosition: &storepb.Position{
				Line:   line + int32(stmt.BaseLine()),
				Column: 0,
			},
		})
	}
	return adviceList, nil
}
