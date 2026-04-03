package mysql

import (
	"context"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*RequireAlgorithmOrLockOptionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION, &RequireAlgorithmOrLockOptionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION, &RequireAlgorithmOrLockOptionAdvisor{})

	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION, &RequireAlgorithmOrLockOptionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION, &RequireAlgorithmOrLockOptionAdvisor{})
}

// RequireAlgorithmOrLockOptionAdvisor is the advisor checking for required algorithm or lock options.
type RequireAlgorithmOrLockOptionAdvisor struct {
}

func (*RequireAlgorithmOrLockOptionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	requiredOption, errorCode := "ALGORITHM", code.StatementNoAlgorithmOption
	if checkCtx.Rule.Type == storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION {
		requiredOption, errorCode = "LOCK", code.StatementNoLockOption
	}

	rule := &requireAlgoLockOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		requiredOption: requiredOption,
		errorCode:      errorCode,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type requireAlgoLockOmniRule struct {
	OmniBaseRule
	requiredOption string
	errorCode      code.Code
}

func (*requireAlgoLockOmniRule) Name() string {
	return "RequireAlgorithmOrLockOptionRule"
}

func (r *requireAlgoLockOmniRule) OnStatement(node ast.Node) {
	alter, ok := node.(*ast.AlterTableStmt)
	if !ok {
		return
	}

	hasOption := false
	for _, cmd := range alter.Commands {
		if r.requiredOption == "ALGORITHM" && cmd.Type == ast.ATAlgorithm {
			hasOption = true
			break
		}
		if r.requiredOption == "LOCK" && cmd.Type == ast.ATLock {
			hasOption = true
			break
		}
	}

	if !hasOption {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          int32(r.errorCode),
			Title:         r.Title,
			Content:       "ALTER TABLE statement should include " + r.requiredOption + " option",
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.ContentStartLine())),
		})
	}
}

// nolint:unused
func omniAlterCmdHasOption(cmds []*ast.AlterTableCmd, optName string) bool {
	for _, cmd := range cmds {
		if optName == "ALGORITHM" && cmd.Type == ast.ATAlgorithm {
			return true
		}
		if optName == "LOCK" && cmd.Type == ast.ATLock {
			return true
		}
	}
	return false
}

// nolint:unused
func omniAlterCmdOptionValue(cmd *ast.AlterTableCmd) string {
	if cmd == nil || cmd.Name == "" {
		return ""
	}
	return strings.ToUpper(cmd.Name)
}
