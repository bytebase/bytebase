package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
)

var (
	_ advisor.Advisor = (*EncodingAllowlistAdvisor)(nil)
	_ ast.Visitor     = (*encodingAllowlistChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleCharsetAllowlist, &EncodingAllowlistAdvisor{})
}

// EncodingAllowlistAdvisor is the advisor checking for encoding allowlist.
type EncodingAllowlistAdvisor struct {
}

// Check checks for encoding allowlist.
func (*EncodingAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &encodingAllowlistChecker{
		level:     level,
		title:     string(checkCtx.Rule.Type),
		allowlist: make(map[string]bool),
	}

	for _, encoding := range payload.List {
		checker.allowlist[strings.ToLower(encoding)] = true
	}

	for _, stmt := range stmtList {
		ast.Walk(checker, stmt)
	}

	return checker.adviceList, nil
}

type encodingAllowlistChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	text       string
	line       int
	allowlist  map[string]bool
}

// Visit implements the ast.Visitor interface.
func (checker *encodingAllowlistChecker) Visit(in ast.Node) ast.Visitor {
	code := advisor.Ok
	var disabledEncoding string
	line := checker.line
	switch node := in.(type) {
	case *ast.CreateDatabaseStmt:
		encoding := getDatabaseEncoding(node.OptionList)
		if _, exist := checker.allowlist[encoding]; encoding != "" && !exist {
			code = advisor.DisabledCharset
			disabledEncoding = encoding
		}
	default:
	}

	if code != advisor.Ok {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.level,
			Code:          code.Int32(),
			Title:         checker.title,
			Content:       fmt.Sprintf("\"%s\" used disabled encoding '%s'", checker.text, disabledEncoding),
			StartPosition: common.ConvertPGParserLineToPosition(line),
		})
	}

	return checker
}

func getDatabaseEncoding(optionList []*ast.DatabaseOptionDef) string {
	for _, option := range optionList {
		if option.Type == ast.DatabaseOptionEncoding {
			return strings.ToLower(option.Value)
		}
	}

	return ""
}
