package pg

import (
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/bytebase/bytebase/plugin/parser/ast"
)

var (
	_ advisor.Advisor = (*EncodingAllowlistAdvisor)(nil)
	_ ast.Visitor     = (*encodingAllowlistChecker)(nil)
)

func init() {
	advisor.Register(db.Postgres, advisor.PostgreSQLEncodingAllowlist, &EncodingAllowlistAdvisor{})
}

// EncodingAllowlistAdvisor is the advisor checking for encoding allowlist.
type EncodingAllowlistAdvisor struct {
}

// Check checks for encoding allowlist.
func (*EncodingAllowlistAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	stmtList, errAdvice := parseStatement(statement)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &encodingAllowlistChecker{
		level:     level,
		title:     string(ctx.Rule.Type),
		allowlist: make(map[string]bool),
	}

	for _, encoding := range payload.List {
		checker.allowlist[strings.ToLower(encoding)] = true
	}

	for _, stmt := range stmtList {
		ast.Walk(checker, stmt)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type encodingAllowlistChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
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
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    code,
			Title:   checker.title,
			Content: fmt.Sprintf("\"%s\" used disabled encoding '%s'", checker.text, disabledEncoding),
			Line:    line,
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
