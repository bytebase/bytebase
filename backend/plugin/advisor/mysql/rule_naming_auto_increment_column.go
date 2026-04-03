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
	_ advisor.Advisor = (*NamingAutoIncrementColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT, &NamingAutoIncrementColumnAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT, &NamingAutoIncrementColumnAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT, &NamingAutoIncrementColumnAdvisor{})
}

// NamingAutoIncrementColumnAdvisor is the advisor checking for auto-increment naming convention.
type NamingAutoIncrementColumnAdvisor struct {
}

// Check checks for auto-increment naming convention.
func (*NamingAutoIncrementColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for this rule")
	}

	format, err := regexp.Compile(namingPayload.Format)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}

	rule := &namingAutoIncrementColumnOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		format:    format,
		maxLength: maxLength,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type namingAutoIncrementColumnOmniRule struct {
	OmniBaseRule
	format    *regexp.Regexp
	maxLength int
}

func (*namingAutoIncrementColumnOmniRule) Name() string {
	return "NamingAutoIncrementColumnRule"
}

func (r *namingAutoIncrementColumnOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		if n.Table == nil {
			return
		}
		tableName := n.Table.Name
		for _, col := range n.Columns {
			if col == nil || !col.AutoIncrement {
				continue
			}
			r.handleAutoIncrementColumn(tableName, col.Name, r.LocToLine(n.Loc))
		}
	case *ast.AlterTableStmt:
		tableName := ""
		if n.Table != nil {
			tableName = n.Table.Name
		}
		for _, cmd := range n.Commands {
			if cmd == nil {
				continue
			}
			switch cmd.Type {
			case ast.ATAddColumn, ast.ATModifyColumn, ast.ATChangeColumn:
				for _, col := range omniGetColumnsFromCmd(cmd) {
					if col.AutoIncrement {
						r.handleAutoIncrementColumn(tableName, col.Name, r.LocToLine(n.Loc))
					}
				}
			default:
			}
		}
	default:
	}
}

func (r *namingAutoIncrementColumnOmniRule) handleAutoIncrementColumn(tableName, columnName string, lineNumber int32) {
	absoluteLine := r.BaseLine + int(lineNumber)
	if !r.format.MatchString(columnName) {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.NamingAutoIncrementColumnConventionMismatch.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("`%s`.`%s` mismatches auto_increment column naming convention, naming format should be %q", tableName, columnName, r.format),
			StartPosition: common.ConvertANTLRLineToPosition(absoluteLine),
		})
	}
	if r.maxLength > 0 && len(columnName) > r.maxLength {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.NamingAutoIncrementColumnConventionMismatch.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("`%s`.`%s` mismatches auto_increment column naming convention, its length should be within %d characters", tableName, columnName, r.maxLength),
			StartPosition: common.ConvertANTLRLineToPosition(absoluteLine),
		})
	}
}
