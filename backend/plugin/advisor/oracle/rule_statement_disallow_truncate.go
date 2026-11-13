package oracle

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oracle/ast"
	oraclev1review "github.com/bytebase/omni/oracle/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var _ advisor.Advisor = (*StatementDisallowTruncateAdvisor)(nil)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE, &StatementDisallowTruncateAdvisor{})
}

// StatementDisallowTruncateAdvisor is the advisor disallowing TRUNCATE. The
// judgment lives in omni (oracle/v1review.StatementDisallowTruncate); this
// adapter renders its findings as advice.
type StatementDisallowTruncateAdvisor struct{}

func (*StatementDisallowTruncateAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := plsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range oraclev1review.StatementDisallowTruncate([]ast.Node{node}, stmt.Text) {
			var content string
			switch f.Code {
			case v1review.Truncate:
				content = fmt.Sprintf(`TRUNCATE TABLE %q is not allowed: it issues an implicit COMMIT and cannot be rolled back. Any prior uncommitted work in the same transaction is also committed. Prior-backup treats this as DDL and does not produce row-level snapshots.`, f.Table)
			case v1review.TruncatePartition:
				content = fmt.Sprintf(`ALTER TABLE %q TRUNCATE %s %q is not allowed: partition truncate shares the implicit-commit gap of TRUNCATE TABLE on Oracle. Prior-backup treats this as DDL and does not produce row-level snapshots.`, f.Table, f.Value, f.Name)
			default:
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementDisallowTruncate.Int32(),
				Title:         title,
				Content:       content,
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + findingLine(stmt.Text, f)),
			})
		}
	}
	return adviceList, nil
}
