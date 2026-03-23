package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*EncodingAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, &EncodingAllowlistAdvisor{})
}

// EncodingAllowlistAdvisor is the advisor checking for encoding allowlist.
type EncodingAllowlistAdvisor struct {
}

// Check checks for encoding allowlist.
func (*EncodingAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()

	allowlist := make(map[string]bool)
	for _, encoding := range stringArrayPayload.List {
		allowlist[strings.ToLower(encoding)] = true
	}

	rule := &encodingAllowlistRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		allowlist: allowlist,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type encodingAllowlistRule struct {
	OmniBaseRule

	allowlist map[string]bool
}

func (*encodingAllowlistRule) Name() string {
	return "encoding-allowlist"
}

func (r *encodingAllowlistRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.CreatedbStmt)
	if !ok {
		return
	}

	encoding := r.extractEncoding(n)
	if encoding == "" {
		return
	}

	if !r.allowlist[strings.ToLower(encoding)] {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.DisabledCharset.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("\"\" used disabled encoding '%s'", strings.ToLower(encoding)),
			StartPosition: &storepb.Position{
				Line:   0,
				Column: 0,
			},
		})
	}
}

func (*encodingAllowlistRule) extractEncoding(n *ast.CreatedbStmt) string {
	if n.Options == nil {
		return ""
	}
	for _, item := range n.Options.Items {
		defElem, ok := item.(*ast.DefElem)
		if !ok {
			continue
		}
		if strings.EqualFold(defElem.Defname, "encoding") {
			if s, ok := defElem.Arg.(*ast.String); ok {
				return s.Str
			}
		}
	}
	return ""
}
