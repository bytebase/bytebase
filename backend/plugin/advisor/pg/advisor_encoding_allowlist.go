package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*EncodingAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleCharsetAllowlist, &EncodingAllowlistAdvisor{})
}

// EncodingAllowlistAdvisor is the advisor checking for encoding allowlist.
type EncodingAllowlistAdvisor struct {
}

// Check checks for encoding allowlist.
func (*EncodingAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Convert allowlist to lowercase for case-insensitive comparison
	allowlist := make(map[string]bool)
	for _, encoding := range payload.List {
		allowlist[strings.ToLower(encoding)] = true
	}

	rule := &encodingAllowlistRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		allowlist: allowlist,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type encodingAllowlistRule struct {
	BaseRule

	allowlist map[string]bool
}

func (*encodingAllowlistRule) Name() string {
	return "encoding-allowlist"
}

func (r *encodingAllowlistRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createdbstmt":
		r.handleCreatedbstmt(ctx.(*parser.CreatedbstmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*encodingAllowlistRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *encodingAllowlistRule) handleCreatedbstmt(ctx *parser.CreatedbstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Extract encoding from createdb_opt_list
	// Check both with and without WITH keyword
	var encoding string
	if ctx.Createdb_opt_list() != nil {
		encoding = r.extractEncoding(ctx.Createdb_opt_list())
	}

	if encoding != "" {
		// Check if encoding is in allowlist
		if !r.allowlist[strings.ToLower(encoding)] {
			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    code.DisabledCharset.Int32(),
				Title:   r.title,
				Content: fmt.Sprintf("\"\" used disabled encoding '%s'", strings.ToLower(encoding)),
				StartPosition: &storepb.Position{
					Line:   0,
					Column: 0,
				},
			})
		}
	}
}

func (*encodingAllowlistRule) extractEncoding(optList parser.ICreatedb_opt_listContext) string {
	if optList == nil {
		return ""
	}

	// Iterate through all createdb_opt_items
	if optList.Createdb_opt_items() == nil {
		return ""
	}

	for _, item := range optList.Createdb_opt_items().AllCreatedb_opt_item() {
		if item == nil {
			continue
		}

		// Check if this is an ENCODING option
		if item.Createdb_opt_name() != nil && item.Createdb_opt_name().ENCODING() != nil {
			// Get the encoding value from Opt_boolean_or_string
			if item.Opt_boolean_or_string() != nil {
				// Could be a string constant or identifier
				text := item.Opt_boolean_or_string().GetText()
				// Remove quotes if present
				if len(text) >= 2 && text[0] == '\'' && text[len(text)-1] == '\'' {
					return text[1 : len(text)-1]
				}
				return text
			}
		}
	}

	return ""
}
