// Package oracle is the advisor for oracle database.
package oracle

import (
	"github.com/bytebase/omni/oracle/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

// BaseRule provides common functionality for Oracle advisor rules.
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

func (r *BaseRule) locLine(loc ast.Loc) int {
	if loc.Start < 0 || r.stmtText == "" {
		return r.baseLine + 1
	}
	return r.baseLine + int(plsqlparser.ByteOffsetToRunePosition(r.stmtText, loc.Start).Line)
}

func (r *BaseRule) rawText(loc ast.Loc) string {
	if loc.Start < 0 || loc.End < loc.Start || r.stmtText == "" || loc.End > len(r.stmtText) {
		return ""
	}
	return r.stmtText[loc.Start:loc.End]
}

// OnStatement is a no-op default for rules while they are migrated to omni.
func (*BaseRule) OnStatement(_ ast.Node) {}
