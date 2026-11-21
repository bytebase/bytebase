package advisor

import (
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// V1PosMode selects the default StartPosition calculation for a V1Engine.
type V1PosMode uint8

const (
	// V1PosFinding places findings at f.Start, falling back to the first non-whitespace character.
	V1PosFinding V1PosMode = iota
	// V1PosContentStart places findings at the statement's first non-whitespace line.
	V1PosContentStart
	// V1PosFindingOrFirstLine places findings at f.Start, falling back to line 1 of the statement.
	V1PosFindingOrFirstLine
)

// V1PosFunc computes the Advice StartPosition for a finding within a statement.
type V1PosFunc[N any] func(V1Stmt[N], v1review.Finding) *storepb.Position

// V1CompatEntry maps a v1review compatibility finding code to its advice code,
// content format, and optional forced WARNING status.
type V1CompatEntry struct {
	Code    code.Code
	Format  string
	Warning bool
}

// V1Engine builds Advisor implementations for omni v1review-backed engines
// with shared memoized node extraction, payload validation, and position helpers.
type V1Engine[N any] struct {
	r          v1Runner[N]
	defaultPos V1PosFunc[N]
}

// NewV1Engine returns a rule builder for an engine whose AST nodes are N.
func NewV1Engine[N any](
	extract func(Context) ([]V1Stmt[N], error),
	byteOffsetLine func(sql string, byteOffset int) int32,
	mode V1PosMode,
) *V1Engine[N] {
	e := &V1Engine[N]{
		r: v1Runner[N]{
			extract:        extract,
			byteOffsetLine: byteOffsetLine,
			memoKey:        fmt.Sprintf("v1engine:%d", v1RunnerMemoSeq.Add(1)),
		},
	}
	switch mode {
	case V1PosContentStart:
		e.defaultPos = e.PosContentStart
	case V1PosFindingOrFirstLine:
		e.defaultPos = e.PosFindingOrFirstLine
	default:
		e.defaultPos = e.PosFinding
	}
	return e
}

// RequireStringArrayPayload returns the string array payload list or an error with msg when absent.
func RequireStringArrayPayload(rule *storepb.SQLReviewRule, msg string) ([]string, error) {
	p := rule.GetStringArrayPayload()
	if p == nil {
		return nil, errors.New(msg)
	}
	return p.List, nil
}

// Prepare validates the rule level and extracts memoized statement wrappers and AST nodes.
func (e *V1Engine[N]) Prepare(checkCtx Context) ([]V1Stmt[N], []N, storepb.Advice_Status, string, error) {
	level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, nil, 0, "", err
	}
	stmts, nodes, err := e.r.extractNodes(checkCtx)
	if err != nil {
		return nil, nil, 0, "", err
	}
	return stmts, nodes, level, checkCtx.Rule.Type.String(), nil
}

// Nodes returns memoized statement wrappers and AST nodes without reading checkCtx.Rule.
func (e *V1Engine[N]) Nodes(checkCtx Context) ([]V1Stmt[N], []N, error) {
	return e.r.extractNodes(checkCtx)
}

func (e *V1Engine[N]) pos(posFn []V1PosFunc[N]) V1PosFunc[N] {
	if len(posFn) > 0 && posFn[0] != nil {
		return posFn[0]
	}
	return e.defaultPos
}

// PosContentStart anchors the advice on the statement's first non-whitespace line.
func (e *V1Engine[N]) PosContentStart(stmt V1Stmt[N], _ v1review.Finding) *storepb.Position {
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(e.r.contentStartLine(stmt.Text)))
}

// PosContentEnd anchors the advice on the statement's last non-whitespace, non-semicolon line.
func (e *V1Engine[N]) PosContentEnd(stmt V1Stmt[N], _ v1review.Finding) *storepb.Position {
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(e.r.contentEndLine(stmt.Text)))
}

// PosFinding anchors the advice at f.Start, falling back to the first non-whitespace character.
func (e *V1Engine[N]) PosFinding(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(e.r.findingLine(stmt.Text, f)))
}

// PosFindingOrFirstLine anchors the advice at f.Start, falling back to line 1 of the statement.
func (e *V1Engine[N]) PosFindingOrFirstLine(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	line := int32(1)
	if f.Start != 0 || f.End != 0 {
		line = e.r.byteOffsetLine(stmt.Text, f.Start)
	}
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(line))
}

// PosSearchOrStart anchors the advice on the line containing f.Search, or contentStartLine.
func (e *V1Engine[N]) PosSearchOrStart(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	return e.r.posSearchOrStart(stmt, f)
}

// PosNameOrStart anchors the advice on the line containing f.Name, or contentStartLine.
func (e *V1Engine[N]) PosNameOrStart(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(e.r.findLineByName(stmt.Text, f.Name)))
}

// PosNil returns nil for rules whose advice carries no StartPosition.
func (*V1Engine[N]) PosNil(V1Stmt[N], v1review.Finding) *storepb.Position {
	return nil
}

// ContentStartLine returns the 1-based line of the first non-whitespace character in text.
func (e *V1Engine[N]) ContentStartLine(text string) int32 {
	return e.r.contentStartLine(text)
}

// ContentEndLine returns the 1-based line of the last non-whitespace, non-semicolon character in text.
func (e *V1Engine[N]) ContentEndLine(text string) int32 {
	return e.r.contentEndLine(text)
}

// FindLineByName returns the 1-based line of name in text, or ContentStartLine(text).
func (e *V1Engine[N]) FindLineByName(text, name string) int32 {
	return e.r.findLineByName(text, name)
}

// TrimmedText returns text with surrounding whitespace and trailing semicolons removed.
func (*V1Engine[N]) TrimmedText(text string) string {
	return trimmedStmtText(text)
}

// SchemaOf converts a Context's DBSchema snapshot to a v1review.Schema.
func (*V1Engine[N]) SchemaOf(m Context) v1review.Schema {
	if m.DBSchema == nil {
		return v1review.Schema{}
	}
	return v1review.Schema{Metadata: m.DBSchema, ObjectCaseSensitive: m.IsObjectCaseSensitive}
}

// SimpleStmt builds a per-statement rule with a fixed advice code and content formatter.
func (e *V1Engine[N]) SimpleStmt(
	fn func([]N) []v1review.Finding,
	c code.Code,
	content func(V1Stmt[N], v1review.Finding) string,
	posFn ...V1PosFunc[N],
) Func {
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			for _, f := range fn(nodes[i : i+1 : i+1]) {
				adviceList = append(adviceList, newAdvice(status, c, title, content(stmt, f), pos(stmt, f)))
			}
		}
		return adviceList, nil
	}
}

// Stmt builds a per-statement rule that passes the full V1Stmt to fn and filters/formats findings.
func (e *V1Engine[N]) Stmt(
	fn func(V1Stmt[N]) []v1review.Finding,
	render func(V1Stmt[N], v1review.Finding) (code.Code, string, bool),
	posFn ...V1PosFunc[N],
) Func {
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, _, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, stmt := range stmts {
			for _, f := range fn(stmt) {
				c, content, ok := render(stmt, f)
				if !ok {
					continue
				}
				adviceList = append(adviceList, newAdvice(status, c, title, content, pos(stmt, f)))
			}
		}
		return adviceList, nil
	}
}

// SimpleBatch builds a whole-script rule with a fixed advice code and optional f.Code filter.
func (e *V1Engine[N]) SimpleBatch(
	fn func(Context, []N) []v1review.Finding,
	c code.Code,
	filterCode v1review.Code,
	content func(v1review.Finding) string,
	posFn ...V1PosFunc[N],
) Func {
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range fn(checkCtx, nodes) {
			if filterCode != "" && f.Code != filterCode {
				continue
			}
			var p *storepb.Position
			if f.Statement >= 0 && f.Statement < len(stmts) {
				p = pos(stmts[f.Statement], f)
			}
			adviceList = append(adviceList, newAdvice(status, c, title, content(f), p))
		}
		return adviceList, nil
	}
}

// NumberStmt builds a per-statement rule parameterized by number_payload.
func (e *V1Engine[N]) NumberStmt(
	fn func([]N, int) []v1review.Finding,
	positiveOnly bool,
	c code.Code,
	content func(V1Stmt[N], v1review.Finding, int) string,
	posFn ...V1PosFunc[N],
) Func {
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		limit, err := RequireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
		if err != nil {
			return nil, err
		}
		if positiveOnly && limit <= 0 {
			return nil, nil
		}
		stmts, nodes, err := e.r.extractNodes(checkCtx)
		if err != nil {
			return nil, err
		}
		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			for _, f := range fn(nodes[i:i+1:i+1], limit) {
				adviceList = append(adviceList, newAdvice(level, c, title, content(stmt, f, limit), pos(stmt, f)))
			}
		}
		return adviceList, nil
	}
}

// StringArrayStmt builds a per-statement rule parameterized by string_array_payload.
func (e *V1Engine[N]) StringArrayStmt(
	fn func([]N, []string) []v1review.Finding,
	missingMsg string,
	c code.Code,
	filterCode v1review.Code,
	content func(V1Stmt[N], v1review.Finding) string,
	posFn ...V1PosFunc[N],
) Func {
	if missingMsg == "" {
		missingMsg = "string_array_payload is required for this rule"
	}
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		list, err := RequireStringArrayPayload(checkCtx.Rule, missingMsg)
		if err != nil {
			return nil, err
		}
		stmts, nodes, err := e.r.extractNodes(checkCtx)
		if err != nil {
			return nil, err
		}
		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			for _, f := range fn(nodes[i:i+1:i+1], list) {
				if filterCode != "" && f.Code != filterCode {
					continue
				}
				adviceList = append(adviceList, newAdvice(level, c, title, content(stmt, f), pos(stmt, f)))
			}
		}
		return adviceList, nil
	}
}

// NamingRegex builds a per-statement regex naming rule (NAMING_TABLE, NAMING_COLUMN,
// TABLE_DROP_NAMING_CONVENTION). When tooLongContent is nil, MaxLength defaults to 0
// and NameTooLong findings are ignored.
func (e *V1Engine[N]) NamingRegex(
	fn func([]N, v1review.NamingConfig) ([]v1review.Finding, error),
	adviceCode code.Code,
	missingMsg string,
	mismatchContent func(v1review.Finding, string) string,
	tooLongContent func(v1review.Finding, int) string,
	posFn ...V1PosFunc[N],
) Func {
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		namingPayload := checkCtx.Rule.GetNamingPayload()
		if namingPayload == nil {
			return nil, errors.New(missingMsg)
		}
		maxLength := 0
		if tooLongContent != nil {
			maxLength = int(namingPayload.MaxLength)
			if maxLength == 0 {
				maxLength = DefaultNameLengthLimit
			}
		}
		cfg := v1review.NamingConfig{Pattern: namingPayload.Format, MaxLength: maxLength}
		if _, err := cfg.Compile(); err != nil {
			return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
		}
		stmts, nodes, err := e.r.extractNodes(checkCtx)
		if err != nil {
			return nil, err
		}
		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			findings, err := fn(nodes[i:i+1:i+1], cfg)
			if err != nil {
				return nil, err
			}
			for _, f := range findings {
				var content string
				switch f.Code {
				case v1review.NameMismatch:
					content = mismatchContent(f, namingPayload.Format)
				case v1review.NameTooLong:
					if tooLongContent == nil {
						continue
					}
					content = tooLongContent(f, maxLength)
				default:
					continue
				}
				adviceList = append(adviceList, newAdvice(level, adviceCode, title, content, pos(stmt, f)))
			}
		}
		return adviceList, nil
	}
}

// TemplateNaming builds a per-statement template naming rule (NAMING_INDEX_*).
func (e *V1Engine[N]) TemplateNaming(
	fn func([]N, v1review.TemplateConfig, v1review.Schema) ([]v1review.Finding, error),
	adviceCode code.Code,
	errTitle string,
	kindLabel string,
	invalidFmt string,
	nilInvalidPos bool,
	posFn ...V1PosFunc[N],
) Func {
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		namingPayload, maxLength, err := requireNamingPayload(checkCtx.Rule, "naming_payload is required for this rule")
		if err != nil {
			return nil, err
		}
		templateList, _ := ParseTemplateTokens(namingPayload.Format)
		for _, key := range templateList {
			if _, ok := TemplateNamingTokens[checkCtx.Rule.Type][key]; !ok {
				return nil, errors.Errorf("invalid template %s for rule %s", key, checkCtx.Rule.Type)
			}
		}
		stmts, nodes, err := e.r.extractNodes(checkCtx)
		if err != nil {
			return nil, err
		}
		cfg := v1review.TemplateConfig{Format: namingPayload.Format, MaxLength: maxLength}
		schema := v1SchemaOf(checkCtx.OriginalMetadata)
		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			findings, err := fn(nodes[i:i+1:i+1], cfg, schema)
			if err != nil {
				return nil, err
			}
			for _, f := range findings {
				p := pos(stmt, f)
				switch f.Code {
				case v1review.PatternInvalid:
					if nilInvalidPos {
						p = nil
					}
					adviceList = append(adviceList, newAdvice(level, code.Internal, errTitle, fmt.Sprintf(invalidFmt, f.Detail), p))
				case v1review.NameMismatch:
					adviceList = append(adviceList, newAdvice(level, adviceCode, title, fmt.Sprintf("%s in table \"%s\" mismatches the naming convention, expect %q but found \"%s\"", kindLabel, f.Table, f.Pattern, f.Name), p))
				case v1review.NameTooLong:
					adviceList = append(adviceList, newAdvice(level, adviceCode, title, fmt.Sprintf("%s \"%s\" in table \"%s\" mismatches the naming convention, its length should be within %d characters", kindLabel, f.Name, f.Table, maxLength), p))
				default:
				}
			}
		}
		return adviceList, nil
	}
}

// BackwardCompatibility builds SCHEMA_BACKWARD_COMPATIBILITY across the full statement batch.
func (e *V1Engine[N]) BackwardCompatibility(
	fn func(Context, []N) []v1review.Finding,
	entries map[v1review.Code]V1CompatEntry,
	content func(V1Stmt[N], v1review.Finding, string) string,
	posFn ...V1PosFunc[N],
) Func {
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range fn(checkCtx, nodes) {
			entry, ok := entries[f.Code]
			if !ok {
				continue
			}
			s := status
			if entry.Warning {
				s = storepb.Advice_WARNING
			}
			stmt := stmts[f.Statement]
			adviceList = append(adviceList, newAdvice(s, entry.Code, title, content(stmt, f, entry.Format), pos(stmt, f)))
		}
		return adviceList, nil
	}
}

// WhereDisallowFuncAndCalc builds STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS.
func (e *V1Engine[N]) WhereDisallowFuncAndCalc(
	fn func(Context, []N) []v1review.Finding,
	dedup bool,
	posFn ...V1PosFunc[N],
) Func {
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			for _, f := range fn(checkCtx, nodes[i:i+1:i+1]) {
				var content string
				switch f.Code {
				case v1review.FunctionOnIndexedColumn:
					content = fmt.Sprintf("Function %q is applied to indexed column %q in the WHERE clause, which prevents index usage", f.Value, f.Name)
				case v1review.CalculationOnIndexedColumn:
					content = fmt.Sprintf("Calculation is applied to indexed column %q in the WHERE clause, which prevents index usage", f.Name)
				default:
					continue
				}
				p := pos(stmt, f)
				if dedup && slices.ContainsFunc(adviceList, func(a *storepb.Advice) bool {
					return a.Content == content && a.GetStartPosition().GetLine() == p.GetLine()
				}) {
					continue
				}
				adviceList = append(adviceList, newAdvice(status, code.StatementDisallowFunctionsAndCalculations, title, content, p))
			}
		}
		return adviceList, nil
	}
}

// ColumnComment builds COLUMN_COMMENT for engines whose omni v1review.ColumnComment
// judges all statements together and requires comment_convention_payload when findings occur.
func (e *V1Engine[N]) ColumnComment(
	fn func(Context, []N, bool, int) []v1review.Finding,
	nilMaxLength int,
	nonPositiveIsMaxInt bool,
	formatCol func(v1review.Finding) string,
	posFn ...V1PosFunc[N],
) Func {
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		payload := checkCtx.Rule.GetCommentConventionPayload()
		required, maxLength := true, nilMaxLength
		if payload != nil {
			required, maxLength = payload.Required, int(payload.MaxLength)
			if maxLength <= 0 {
				if nonPositiveIsMaxInt {
					maxLength = math.MaxInt
				} else {
					maxLength = -1
				}
			}
		}
		var adviceList []*storepb.Advice
		for _, f := range fn(checkCtx, nodes, required, maxLength) {
			if payload == nil {
				return nil, errors.New("comment_convention_payload is required for column comment rule")
			}
			col := formatCol(f)
			var adviceCode code.Code
			var content string
			switch f.Code {
			case v1review.ColumnCommentMissing:
				adviceCode = code.CommentEmpty
				content = "Comment is required for column " + col
			case v1review.ColumnCommentTooLong:
				adviceCode = code.CommentTooLong
				content = fmt.Sprintf("Column %s comment is too long. The length of comment should be within %d characters", col, payload.MaxLength)
			default:
				continue
			}
			adviceList = append(adviceList, newAdvice(status, adviceCode, title, content, pos(stmts[f.Statement], f)))
		}
		return adviceList, nil
	}
}

// TableComment builds TABLE_COMMENT for engines whose omni v1review.TableComment
// judges all statements together.
func (e *V1Engine[N]) TableComment(
	fn func(Context, []N, *v1review.TableCommentConfig) []v1review.Finding,
	formatTable func(v1review.Finding) string,
	posFn ...V1PosFunc[N],
) Func {
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		var cfg *v1review.TableCommentConfig
		if payload := checkCtx.Rule.GetCommentConventionPayload(); payload != nil {
			cfg = &v1review.TableCommentConfig{Required: payload.Required, MaxLength: int(payload.MaxLength)}
		}
		var adviceList []*storepb.Advice
		for _, f := range fn(checkCtx, nodes, cfg) {
			if cfg == nil {
				return nil, errors.New("comment_convention_payload is required for table comment rule")
			}
			tbl := formatTable(f)
			var adviceCode code.Code
			var content string
			switch f.Code {
			case v1review.TableCommentMissing:
				adviceCode = code.CommentEmpty
				content = "Comment is required for table " + tbl
			case v1review.TableCommentTooLong:
				adviceCode = code.CommentTooLong
				content = fmt.Sprintf("Table %s comment is too long. The length of comment should be within %d characters", tbl, cfg.MaxLength)
			default:
				continue
			}
			adviceList = append(adviceList, newAdvice(status, adviceCode, title, content, pos(stmts[f.Statement], f)))
		}
		return adviceList, nil
	}
}
