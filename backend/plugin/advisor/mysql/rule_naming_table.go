package mysql

import (
	"context"
	"fmt"
	"regexp"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NamingTableConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_TABLE, &NamingTableConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_NAMING_TABLE, &NamingTableConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_NAMING_TABLE, &NamingTableConventionAdvisor{})
}

// NamingTableConventionAdvisor is the advisor checking for table naming convention.
type NamingTableConventionAdvisor struct {
}

// Check checks for table naming convention.
func (*NamingTableConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for naming table rule")
	}

	format, err := regexp.Compile(namingPayload.Format)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}

	rule := &namingTableOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		format:    format,
		maxLength: maxLength,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type namingTableOmniRule struct {
	OmniBaseRule
	format    *regexp.Regexp
	maxLength int
}

func (*namingTableOmniRule) Name() string {
	return "NamingTableRule"
}

func (r *namingTableOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		if n.Table != nil {
			r.handleTableName(n.Table.Name, r.LocToLine(n.Loc))
		}
	case *ast.AlterTableStmt:
		for _, cmd := range n.Commands {
			if cmd.Type == ast.ATRenameTable && cmd.NewName != "" {
				r.handleTableName(cmd.NewName, r.LocToLine(n.Loc))
			}
		}
	case *ast.RenameTableStmt:
		for _, pair := range n.Pairs {
			if pair.New != nil {
				r.handleTableName(pair.New.Name, r.LocToLine(n.Loc))
			}
		}
	default:
	}
}

func (r *namingTableOmniRule) handleTableName(tableName string, lineNumber int32) {
	absoluteLine := r.BaseLine + int(lineNumber)
	if !r.format.MatchString(tableName) {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("`%s` mismatches table naming convention, naming format should be %q", tableName, r.format),
			StartPosition: common.ConvertANTLRLineToPosition(absoluteLine),
		})
	}
	if r.maxLength > 0 && len(tableName) > r.maxLength {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("`%s` mismatches table naming convention, its length should be within %d characters", tableName, r.maxLength),
			StartPosition: common.ConvertANTLRLineToPosition(absoluteLine),
		})
	}
}
