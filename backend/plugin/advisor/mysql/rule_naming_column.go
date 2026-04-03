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

var _ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_COLUMN, &NamingColumnConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_NAMING_COLUMN, &NamingColumnConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_NAMING_COLUMN, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column naming convention.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (*NamingColumnConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for naming column rule")
	}

	format, err := regexp.Compile(namingPayload.Format)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}

	rule := &namingColumnOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		format:    format,
		maxLength: maxLength,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type namingColumnOmniRule struct {
	OmniBaseRule
	format    *regexp.Regexp
	maxLength int
}

func (*namingColumnOmniRule) Name() string {
	return "NamingColumnRule"
}

func (r *namingColumnOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		if n.Table == nil {
			return
		}
		tableName := n.Table.Name
		for _, col := range n.Columns {
			if col == nil {
				continue
			}
			r.handleColumn(tableName, col.Name, r.LocToLine(col.Loc))
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
			case ast.ATAddColumn:
				for _, col := range omniGetColumnsFromCmd(cmd) {
					r.handleColumn(tableName, col.Name, r.LocToLine(n.Loc))
				}
			case ast.ATRenameColumn:
				if cmd.NewName != "" {
					r.handleColumn(tableName, cmd.NewName, r.LocToLine(n.Loc))
				}
			case ast.ATChangeColumn:
				if cmd.Column != nil {
					r.handleColumn(tableName, cmd.Column.Name, r.LocToLine(n.Loc))
				}
			default:
			}
		}
	default:
	}
}

func (r *namingColumnOmniRule) handleColumn(tableName, columnName string, lineNumber int32) {
	absoluteLine := r.BaseLine + int(lineNumber)
	if !r.format.MatchString(columnName) {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.NamingColumnConventionMismatch.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("`%s`.`%s` mismatches column naming convention, naming format should be %q", tableName, columnName, r.format),
			StartPosition: common.ConvertANTLRLineToPosition(absoluteLine),
		})
	}
	if r.maxLength > 0 && len(columnName) > r.maxLength {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.NamingColumnConventionMismatch.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("`%s`.`%s` mismatches column naming convention, its length should be within %d characters", tableName, columnName, r.maxLength),
			StartPosition: common.ConvertANTLRLineToPosition(absoluteLine),
		})
	}
}
