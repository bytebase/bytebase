package mssql

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NamingTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_NAMING_TABLE, &NamingTableAdvisor{})
}

// NamingTableAdvisor is the advisor checking for table naming convention..
type NamingTableAdvisor struct {
}

// Check checks for table naming convention..
func (*NamingTableAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	rule := &namingTableRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
		format:       format,
		maxLength:    maxLength,
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type namingTableRule struct {
	OmniBaseRule
	format    *regexp.Regexp
	maxLength int
}

func (*namingTableRule) Name() string {
	return "NamingTableRule"
}

func (r *namingTableRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		if n.Name == nil {
			return
		}
		r.checkTableName(n.Name.Object, n.Loc)
	case *ast.ExecStmt:
		r.checkSpRename(n)
	default:
	}
}

func (r *namingTableRule) checkTableName(tableName string, loc ast.Loc) {
	if tableName == "" {
		return
	}
	if !r.format.MatchString(tableName) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf(`%s mismatches table naming convention, naming format should be %q`, tableName, r.format),
			StartPosition: &storepb.Position{Line: r.LocToLine(loc)},
		})
	}
	if r.maxLength > 0 && len(tableName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.NamingTableConventionMismatch.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf(`%s mismatches table naming convention, its length should be within %d characters`, tableName, r.maxLength),
			StartPosition: &storepb.Position{Line: r.LocToLine(loc)},
		})
	}
}

func (r *namingTableRule) checkSpRename(execStmt *ast.ExecStmt) {
	if execStmt.Name == nil {
		return
	}
	// Check if the procedure is sp_rename.
	if execStmt.Name.Schema != "" {
		return
	}
	if !strings.EqualFold(execStmt.Name.Object, "sp_rename") {
		return
	}

	if execStmt.Args == nil || execStmt.Args.Len() < 2 {
		return
	}

	// Get the second argument (new name).
	arg1, ok := execStmt.Args.Items[1].(*ast.ExecArg)
	if !ok || arg1 == nil {
		return
	}
	lit, ok := arg1.Value.(*ast.Literal)
	if !ok || lit == nil {
		return
	}
	newTableName := lit.Str
	if strings.HasPrefix(newTableName, "'") && strings.HasSuffix(newTableName, "'") {
		newTableName = newTableName[1 : len(newTableName)-1]
	}

	r.checkTableName(newTableName, execStmt.Loc)
}
