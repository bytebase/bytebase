package mariadb

import (
	"context"

	"github.com/bytebase/omni/mariadb/ast"
	mariadbv1review "github.com/bytebase/omni/mariadb/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mariadbparser "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
)

var (
	_ advisor.Advisor = (*RequireAlgorithmOrLockOptionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION, &RequireAlgorithmOrLockOptionAdvisor{})

	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION, &RequireAlgorithmOrLockOptionAdvisor{})
}

// RequireAlgorithmOrLockOptionAdvisor is the advisor checking for required algorithm or lock options.
// The judgment lives in omni (mariadb/v1review.StatementRequireAlgorithmOption
// and StatementRequireLockOption); this adapter renders its findings as
// advice.
type RequireAlgorithmOrLockOptionAdvisor struct {
}

func (*RequireAlgorithmOrLockOptionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	requiredOption, errorCode, judge := "ALGORITHM", code.StatementNoAlgorithmOption, mariadbv1review.StatementRequireAlgorithmOption
	if checkCtx.Rule.Type == storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION {
		requiredOption, errorCode, judge = "LOCK", code.StatementNoLockOption, mariadbv1review.StatementRequireLockOption
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mariadbparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range judge([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          int32(errorCode),
				Title:         title,
				Content:       "ALTER TABLE statement should include " + requiredOption + " option",
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
			})
		}
	}
	return adviceList, nil
}
