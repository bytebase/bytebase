// Package oracle is the advisor for oracle database.
package oracle

import (
	"github.com/bytebase/omni/oracle/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

type BaseRule struct {
	level      storepb.Advice_Status
	title      string
	adviceList []*storepb.Advice
	baseLine   int
	stmtText   string
}

// NewBaseRule creates a new BaseRule.
func NewBaseRule(level storepb.Advice_Status, title string, baseLine int) BaseRule {
	return BaseRule{
		level:    level,
		title:    title,
		baseLine: baseLine,
	}
}

// AddAdvice adds an advice to the rule.
func (r *BaseRule) AddAdvice(status storepb.Advice_Status, code int32, content string, position *storepb.Position) {
	r.adviceList = append(r.adviceList, &storepb.Advice{
		Status:        status,
		Code:          code,
		Title:         r.title,
		Content:       content,
		StartPosition: position,
	})
}

// GetAdviceList returns the advice list.
func (r *BaseRule) GetAdviceList() ([]*storepb.Advice, error) {
	return r.adviceList, nil
}

// SetBaseLine sets the base line for the rule.
func (r *BaseRule) SetBaseLine(baseLine int) {
	r.baseLine = baseLine
}

// SetStatement sets the current statement context for omni-based rules.
func (r *BaseRule) SetStatement(baseLine int, stmtText string) {
	r.baseLine = baseLine
	r.stmtText = stmtText
}

// OnStatement is a no-op default for rules while they are migrated to omni.
func (*BaseRule) OnStatement(_ ast.Node) {}

// Rule defines the Oracle omni SQL review rule interface.
type Rule interface {
	OnStatement(node ast.Node)
	Name() string
	GetAdviceList() ([]*storepb.Advice, error)
}

// RunRules dispatches parsed Oracle omni AST nodes to rules.
func RunRules(stmts []base.ParsedStatement, rules []Rule) ([]*storepb.Advice, error) {
	for _, stmt := range stmts {
		if stmt.AST == nil {
			continue
		}
		node, ok := plsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, rule := range rules {
			if br, ok := rule.(interface{ SetStatement(int, string) }); ok {
				br.SetStatement(stmt.BaseLine(), stmt.Text)
			}
			rule.OnStatement(node)
		}
	}

	var adviceList []*storepb.Advice
	for _, rule := range rules {
		list, err := rule.GetAdviceList()
		if err != nil {
			return nil, err
		}
		adviceList = append(adviceList, list...)
	}
	return adviceList, nil
}
