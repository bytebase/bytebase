package pg

import (
	"strings"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

// OmniRule defines the interface for omni-based SQL validation rules.
type OmniRule interface {
	// OnStatement is called for each top-level statement AST node.
	OnStatement(node ast.Node)

	// Name returns the rule name for logging/debugging.
	Name() string

	// GetAdviceList returns the accumulated advice from this rule.
	GetAdviceList() []*storepb.Advice
}

// OmniBaseRule provides common functionality for omni-based rules.
type OmniBaseRule struct {
	Level    storepb.Advice_Status
	Title    string
	Advice   []*storepb.Advice
	BaseLine int
	StmtText string
}

// SetStatement sets the statement context for position calculations.
func (r *OmniBaseRule) SetStatement(baseLine int, stmtText string) {
	r.BaseLine = baseLine
	r.StmtText = stmtText
}

// GetAdviceList returns the accumulated advice.
func (r *OmniBaseRule) GetAdviceList() []*storepb.Advice {
	return r.Advice
}

// AddAdvice adds advice. The BaseLine offset is added automatically to StartPosition.Line.
func (r *OmniBaseRule) AddAdvice(advice *storepb.Advice) {
	if advice.StartPosition != nil {
		advice.StartPosition.Line += int32(r.BaseLine)
	}
	r.Advice = append(r.Advice, advice)
}

// AddAdviceAbsolute adds advice with an already-absolute line number.
func (r *OmniBaseRule) AddAdviceAbsolute(advice *storepb.Advice) {
	r.Advice = append(r.Advice, advice)
}

// LocToLine converts an omni Loc byte offset to a 1-based line number
// relative to the current statement (suitable for AddAdvice which adds BaseLine).
func (r *OmniBaseRule) LocToLine(loc ast.Loc) int32 {
	if loc.Start < 0 || r.StmtText == "" {
		return r.ContentStartLine()
	}
	pos := pgparser.ByteOffsetToRunePosition(r.StmtText, loc.Start)
	return pos.Line
}

// TrimmedStmtText returns the statement text with leading/trailing whitespace
// and trailing semicolons removed. This matches the behavior of the ANTLR
// getTextFromTokens helper that did not include semicolons.
func (r *OmniBaseRule) TrimmedStmtText() string {
	return strings.TrimRight(strings.TrimSpace(r.StmtText), ";")
}

// ContentStartLine returns the 1-based line number of the first non-whitespace
// character in StmtText. This accounts for leading newlines that SplitSQL may
// include in the statement text. Returns 1 if the text is empty or has no
// leading newlines.
func (r *OmniBaseRule) ContentStartLine() int32 {
	idx := strings.IndexFunc(r.StmtText, func(c rune) bool {
		return c != ' ' && c != '\t' && c != '\n' && c != '\r'
	})
	if idx <= 0 {
		return 1
	}
	pos := pgparser.ByteOffsetToRunePosition(r.StmtText, idx)
	return pos.Line
}

// ContentEndLine returns the 1-based line number of the last non-whitespace
// character in StmtText. This matches ANTLR's GetStop().GetLine() behavior.
func (r *OmniBaseRule) ContentEndLine() int32 {
	if r.StmtText == "" {
		return 1
	}
	// Find last non-whitespace character
	idx := -1
	for i := len(r.StmtText) - 1; i >= 0; i-- {
		c := r.StmtText[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != ';' {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return 1
	}
	pos := pgparser.ByteOffsetToRunePosition(r.StmtText, idx)
	return pos.Line
}

// FindLineByName searches for an identifier name in the statement text and returns
// its 1-based line number. Falls back to ContentStartLine() if not found.
// This is useful when the AST node's Loc is unknown (-1) but we need a line number.
func (r *OmniBaseRule) FindLineByName(name string) int32 {
	if name == "" || r.StmtText == "" {
		return r.ContentStartLine()
	}
	idx := strings.Index(r.StmtText, name)
	if idx < 0 {
		return r.ContentStartLine()
	}
	pos := pgparser.ByteOffsetToRunePosition(r.StmtText, idx)
	return pos.Line
}

// RunOmniRules iterates over parsed statements and dispatches each omni AST node to all rules.
// Returns combined advice from all rules. Skips statements without omni AST.
func RunOmniRules(stmts []base.ParsedStatement, rules []OmniRule) []*storepb.Advice {
	for _, stmt := range stmts {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		baseLine := stmt.BaseLine()
		for _, rule := range rules {
			if br, ok := rule.(interface{ SetStatement(int, string) }); ok {
				br.SetStatement(baseLine, stmt.Text)
			}
			rule.OnStatement(node)
		}
	}
	var allAdvice []*storepb.Advice
	for _, rule := range rules {
		allAdvice = append(allAdvice, rule.GetAdviceList()...)
	}
	return allAdvice
}
