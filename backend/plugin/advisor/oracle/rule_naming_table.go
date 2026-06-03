// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"regexp"

	"github.com/bytebase/omni/oracle/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NamingTableAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_NAMING_TABLE, &NamingTableAdvisor{})
}

// NamingTableAdvisor is the advisor checking for table naming convention.
type NamingTableAdvisor struct {
}

// Check checks for table naming convention.
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

	rule := NewNamingTableRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase, format, maxLength)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// NamingTableRule is the rule implementation for table naming convention.
type NamingTableRule struct {
	BaseRule

	currentDatabase string
	format          *regexp.Regexp
	maxLength       int
}

// NewNamingTableRule creates a new NamingTableRule.
func NewNamingTableRule(level storepb.Advice_Status, title string, currentDatabase string, format *regexp.Regexp, maxLength int) *NamingTableRule {
	return &NamingTableRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		format:          format,
		maxLength:       maxLength,
	}
}

// Name returns the rule name.
func (*NamingTableRule) Name() string {
	return "naming.table"
}

// OnStatement checks table names in omni DDL statements.
func (r *NamingTableRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkTableName(omniLastObjectName(n.Name), n.Loc)
	case *ast.AlterTableStmt:
		for _, cmd := range omniAlterTableCmds(n) {
			if cmd.Action == ast.AT_RENAME && cmd.NewName != "" {
				r.checkTableName(cmd.NewName, cmd.Loc)
			}
		}
	default:
	}
}

func (r *NamingTableRule) checkTableName(tableName string, loc ast.Loc) {
	if tableName == "" {
		return
	}
	if !r.format.MatchString(tableName) {
		r.AddAdvice(
			r.level,
			code.NamingTableConventionMismatch.Int32(),
			fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, r.format),
			common.ConvertANTLRLineToPosition(r.locLine(loc)),
		)
	}
	if r.maxLength > 0 && len(tableName) > r.maxLength {
		r.AddAdvice(
			r.level,
			code.NamingTableConventionMismatch.Int32(),
			fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, r.maxLength),
			common.ConvertANTLRLineToPosition(r.locLine(loc)),
		)
	}
}

// OnEnter is called when the parser enters a rule context.

// OnExit is called when the parser exits a rule context.
