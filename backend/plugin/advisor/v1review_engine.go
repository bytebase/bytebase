package advisor

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

var v1EngineMemoSeq atomic.Int64

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

// V1EngineOptions configures a V1Engine with optional MySQL-family dialect quirks.
type V1EngineOptions[N any] struct {
	Extract             func(Context) ([]V1Stmt[N], error)
	ByteOffsetLine      func(sql string, byteOffset int) int32
	PosMode             V1PosMode
	TiDBQuirks          bool
	IsAlterTable        func(N) bool
	CreateTableLocStart func(N) (int, bool)
}

// V1TemplateNamingOptions configures TemplateNaming formatting.
type V1TemplateNamingOptions[N any] struct {
	BacktickQuote  bool
	InvalidContent func(V1Stmt[N], v1review.Finding) string
}

// V1RowLimitConfig configures a row-limit rule (STATEMENT_INSERT_ROW_LIMIT or
// STATEMENT_AFFECTED_ROW_LIMIT) on a V1Engine.
type V1RowLimitConfig[N any] struct {
	Probes          func([]N) []int
	Judge           func([]N, map[int]v1review.Probe, int) []v1review.Finding
	ExceededCode    code.Code
	ExceededVerb    string
	ExceededContent func(text, count string, maxRow int) string
	PositiveOnly    bool
	TrimEstimate    bool
	StatementText   func(V1Stmt[N], N) string
	Estimate        func(ctx context.Context, checkCtx Context, stmt V1Stmt[N], node N, settings []string) (v1review.Probe, error)
	BudgetPos       func(V1Stmt[N], N) *storepb.Position
	Pos             V1PosFunc[N]
	PreExecutions   func(stmts []V1Stmt[N], nodes []N, want []int, visit func(i int, settings []string))
}

// V1DMLDryRunConfig configures STATEMENT_DML_DRY_RUN on a V1Engine.
type V1DMLDryRunConfig[N any] struct {
	Probes        func([]N) []int
	Judge         func([]N, map[int]v1review.Probe) []v1review.Finding
	Probe         func(ctx context.Context, prober db.Prober, stmt V1Stmt[N], settings []string) v1review.Probe
	FormatContent func(V1Stmt[N], v1review.Finding) string
	Pos           V1PosFunc[N]
	PreExecutions func(stmts []V1Stmt[N], nodes []N, want []int, visit func(i int, settings []string))
}

// V1ColumnRequiredConfig configures COLUMN_REQUIRED on a V1Engine.
type V1ColumnRequiredConfig[N any] struct {
	Fn        func([]N, []string) []v1review.Finding
	Batch     bool
	SameGroup func(a, b v1review.Finding) bool
	TableFmt  string
	QuoteCol  bool
	Pos       V1PosFunc[N]
}

// V1PriorBackupConfig configures BUILTIN_PRIOR_BACKUP_CHECK on a V1Engine.
type V1PriorBackupConfig[N any] struct {
	Engine     storepb.Engine
	Facts      func(ctx context.Context, checkCtx Context) v1review.PriorBackupFacts
	Check      func(checkCtx Context, stmts []V1Stmt[N], nodes []N, facts v1review.PriorBackupFacts) ([]v1review.Finding, error)
	Render     func(stmts []V1Stmt[N], nodes []N, f v1review.Finding) (code.Code, string, *storepb.Position, bool)
	DefaultPos *storepb.Position
	MixedLine  func(V1Stmt[N]) int
}

type v1CachedStmts[N any] struct {
	stmts []V1Stmt[N]
	nodes []N
}

// V1Engine builds Advisor implementations for omni v1review-backed engines
// with shared memoized node extraction, payload validation, and position helpers.
type V1Engine[N any] struct {
	extract             func(Context) ([]V1Stmt[N], error)
	byteOffsetLine      func(sql string, byteOffset int) int32
	memoKey             string
	defaultPos          V1PosFunc[N]
	tidbQuirks          bool
	isAlterTable        func(N) bool
	createTableLocStart func(N) (int, bool)
}

// NewV1Engine returns a rule builder for an engine whose AST nodes are N.
// A nil byteOffsetLine defaults to base.ByteOffsetToRunePosition(sql, offset).Line.
func NewV1Engine[N any](
	extract func(Context) ([]V1Stmt[N], error),
	byteOffsetLine func(sql string, byteOffset int) int32,
	mode V1PosMode,
) *V1Engine[N] {
	return NewV1EngineWithOptions(V1EngineOptions[N]{
		Extract:        extract,
		ByteOffsetLine: byteOffsetLine,
		PosMode:        mode,
	})
}

// NewV1EngineWithOptions returns a rule builder configured with opt.
func NewV1EngineWithOptions[N any](opt V1EngineOptions[N]) *V1Engine[N] {
	byteOffsetLine := opt.ByteOffsetLine
	if byteOffsetLine == nil {
		byteOffsetLine = defaultByteOffsetLine
	}
	e := &V1Engine[N]{
		extract:             opt.Extract,
		byteOffsetLine:      byteOffsetLine,
		memoKey:             fmt.Sprintf("v1engine:%d", v1EngineMemoSeq.Add(1)),
		tidbQuirks:          opt.TiDBQuirks,
		isAlterTable:        opt.IsAlterTable,
		createTableLocStart: opt.CreateTableLocStart,
	}
	switch opt.PosMode {
	case V1PosContentStart:
		e.defaultPos = e.PosContentStart
	case V1PosFindingOrFirstLine:
		e.defaultPos = e.PosFindingOrFirstLine
	default:
		e.defaultPos = e.PosFinding
	}
	return e
}

// Prepare validates the rule level and extracts memoized statement wrappers and AST nodes.
func (e *V1Engine[N]) Prepare(checkCtx Context) ([]V1Stmt[N], []N, storepb.Advice_Status, string, error) {
	level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, nil, 0, "", err
	}
	stmts, nodes, err := e.Nodes(checkCtx)
	if err != nil {
		return nil, nil, 0, "", err
	}
	return stmts, nodes, level, checkCtx.Rule.Type.String(), nil
}

// Nodes returns memoized statement wrappers and AST nodes without reading checkCtx.Rule.
func (e *V1Engine[N]) Nodes(checkCtx Context) ([]V1Stmt[N], []N, error) {
	if e.memoKey != "" {
		if cached, ok := checkCtx.Memo(e.memoKey); ok {
			if c, typeOK := cached.(v1CachedStmts[N]); typeOK {
				return c.stmts, c.nodes, nil
			}
		}
	}
	stmts, err := e.extract(checkCtx)
	if err != nil {
		return nil, nil, err
	}
	nodes := v1Nodes(stmts)
	if e.memoKey != "" {
		checkCtx.SetMemo(e.memoKey, v1CachedStmts[N]{stmts: stmts, nodes: nodes})
	}
	return stmts, nodes, nil
}

func (e *V1Engine[N]) pos(posFn []V1PosFunc[N]) V1PosFunc[N] {
	if len(posFn) > 0 && posFn[0] != nil {
		return posFn[0]
	}
	return e.defaultPos
}

func (e *V1Engine[N]) absLine(stmt V1Stmt[N], byteOffset int) int {
	return stmt.BaseLine + int(e.byteOffsetLine(stmt.Text, byteOffset))
}

// FirstTokenLine returns the 1-based absolute line of the first non-whitespace rune in stmt.Text.
func (e *V1Engine[N]) FirstTokenLine(stmt V1Stmt[N]) int {
	return e.absLine(stmt, firstNonSpaceIndex(stmt.Text))
}

func (e *V1Engine[N]) findingLine(text string, f v1review.Finding) int32 {
	offset := f.Start
	if f.Start == 0 && f.End == 0 {
		offset = contentStartIndex(text)
	}
	return e.byteOffsetLine(text, offset)
}

// ContentStartLine returns the 1-based line of the first non-whitespace character in text.
func (e *V1Engine[N]) ContentStartLine(text string) int32 {
	idx := contentStartIndex(text)
	if idx <= 0 {
		return 1
	}
	return e.byteOffsetLine(text, idx)
}

// ContentEndLine returns the 1-based line of the last non-whitespace, non-semicolon character in text.
func (e *V1Engine[N]) ContentEndLine(text string) int32 {
	if text == "" {
		return 1
	}
	idx := -1
	for i := len(text) - 1; i >= 0; i-- {
		c := text[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != ';' {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return 1
	}
	return e.byteOffsetLine(text, idx)
}

// FindLineByName returns the 1-based line of name in text, or ContentStartLine(text).
func (e *V1Engine[N]) FindLineByName(text, name string) int32 {
	if name == "" || text == "" {
		return e.ContentStartLine(text)
	}
	idx := strings.Index(text, name)
	if idx < 0 {
		return e.ContentStartLine(text)
	}
	return e.byteOffsetLine(text, idx)
}

// PosContentStart anchors the advice on the statement's first non-whitespace line.
func (e *V1Engine[N]) PosContentStart(stmt V1Stmt[N], _ v1review.Finding) *storepb.Position {
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(e.ContentStartLine(stmt.Text)))
}

// PosContentEnd anchors the advice on the statement's last non-whitespace, non-semicolon line.
func (e *V1Engine[N]) PosContentEnd(stmt V1Stmt[N], _ v1review.Finding) *storepb.Position {
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(e.ContentEndLine(stmt.Text)))
}

// PosFinding anchors the advice at f.Start, falling back to the first non-whitespace character.
func (e *V1Engine[N]) PosFinding(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	if e.tidbQuirks {
		if f.Start == 0 && f.End == 0 {
			return base.ConvertANTLRLineToPosition(e.FirstTokenLine(stmt))
		}
		return base.ConvertANTLRLineToPosition(e.absLine(stmt, f.Start))
	}
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(e.findingLine(stmt.Text, f)))
}

// PosFindingOrFirstLine anchors the advice at f.Start, falling back to line 1 of the statement.
func (e *V1Engine[N]) PosFindingOrFirstLine(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	line := int32(1)
	if f.Start != 0 || f.End != 0 {
		line = e.byteOffsetLine(stmt.Text, f.Start)
	}
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(line))
}

// PosSearchOrStart anchors the advice on the line containing f.Search, or ContentStartLine.
func (e *V1Engine[N]) PosSearchOrStart(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	line := e.ContentStartLine(stmt.Text)
	if f.Search != "" {
		line = e.FindLineByName(stmt.Text, f.Search)
	}
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(line))
}

// PosNameOrStart anchors the advice on the line containing f.Name, or ContentStartLine.
func (e *V1Engine[N]) PosNameOrStart(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(e.FindLineByName(stmt.Text, f.Name)))
}

// PosNil returns nil for rules whose advice carries no StartPosition.
func (*V1Engine[N]) PosNil(V1Stmt[N], v1review.Finding) *storepb.Position {
	return nil
}

func (e *V1Engine[N]) posAbsOrFinding(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	if e.tidbQuirks {
		return base.ConvertANTLRLineToPosition(e.absLine(stmt, f.Start))
	}
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(e.findingLine(stmt.Text, f)))
}

func (e *V1Engine[N]) posStatement(stmt V1Stmt[N]) *storepb.Position {
	if e.tidbQuirks {
		return base.ConvertANTLRLineToPosition(e.FirstTokenLine(stmt))
	}
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(e.findingLine(stmt.Text, v1review.Finding{})))
}

func (e *V1Engine[N]) posStatementFinding(stmt V1Stmt[N], _ v1review.Finding) *storepb.Position {
	return e.posStatement(stmt)
}

func (e *V1Engine[N]) posStatementAbsOrStmt(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	if e.tidbQuirks {
		return base.ConvertANTLRLineToPosition(e.absLine(stmt, f.Start))
	}
	return e.posStatement(stmt)
}

func (e *V1Engine[N]) posFindingOrTiDBFirstToken(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	if e.tidbQuirks {
		return e.posStatement(stmt)
	}
	return e.PosFinding(stmt, f)
}

func (e *V1Engine[N]) posBaseLineTwice(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	if e.tidbQuirks {
		return base.ConvertANTLRLineToPosition(e.absLine(stmt, f.Start))
	}
	p := base.ConvertANTLRLineToPosition(stmt.BaseLine + int(e.findingLine(stmt.Text, f)))
	p.Line += int32(stmt.BaseLine)
	return p
}

func (e *V1Engine[N]) posAllowlist(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	if e.tidbQuirks {
		if e.createTableLocStart != nil {
			if start, ok := e.createTableLocStart(stmt.Node); ok && f.Start != max(start, 0) {
				return base.ConvertANTLRLineToPosition(e.absLine(stmt, f.Start))
			}
		}
		return base.ConvertANTLRLineToPosition(e.FirstTokenLine(stmt))
	}
	return e.posBaseLineTwice(stmt, f)
}

func (e *V1Engine[N]) posAlterFirstToken(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	if e.tidbQuirks {
		if e.isAlterTable != nil && e.isAlterTable(stmt.Node) {
			return base.ConvertANTLRLineToPosition(e.FirstTokenLine(stmt))
		}
		return base.ConvertANTLRLineToPosition(e.absLine(stmt, f.Start))
	}
	return e.PosFinding(stmt, f)
}

func (e *V1Engine[N]) posSearchOrFinding(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	line := e.findingLine(stmt.Text, f)
	if f.Search != "" {
		line = e.FindLineByName(stmt.Text, f.Search)
	}
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(line))
}

func (e *V1Engine[N]) stmtText(stmt V1Stmt[N], rawOnTiDB bool) string {
	if e.tidbQuirks && rawOnTiDB {
		return stmt.Text
	}
	return strings.TrimSpace(stmt.Text)
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
	return e.StmtNodes(fn, func(stmt V1Stmt[N], f v1review.Finding) (code.Code, string, bool) {
		return c, content(stmt, f), true
	}, posFn...)
}

// StmtNodes builds a zero-alloc per-statement rule passing node slices to fn and rendering each finding.
func (e *V1Engine[N]) StmtNodes(
	fn func([]N) []v1review.Finding,
	render func(V1Stmt[N], v1review.Finding) (code.Code, string, bool),
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

// StmtCtx builds a per-statement rule whose prepareFn resolves per-review state once
// before iterating over statements.
func (e *V1Engine[N]) StmtCtx(
	prepareFn func(Context, []N) (func([]N) []v1review.Finding, error),
	render func(Context, V1Stmt[N], v1review.Finding) (code.Code, string, bool),
	posFn ...V1PosFunc[N],
) Func {
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		fn, err := prepareFn(checkCtx, nodes)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			for _, f := range fn(nodes[i : i+1 : i+1]) {
				c, content, ok := render(checkCtx, stmt, f)
				if !ok {
					continue
				}
				adviceList = append(adviceList, newAdvice(status, c, title, content, pos(stmt, f)))
			}
		}
		return adviceList, nil
	}
}

// BatchStmts builds a whole-script rule passing both statements and nodes to fn.
func (e *V1Engine[N]) BatchStmts(
	fn func(Context, []V1Stmt[N], []N) []v1review.Finding,
	render func([]V1Stmt[N], v1review.Finding) (code.Code, string, *storepb.Position, bool),
) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range fn(checkCtx, stmts, nodes) {
			c, content, p, ok := render(stmts, f)
			if !ok {
				continue
			}
			adviceList = append(adviceList, newAdvice(status, c, title, content, p))
		}
		return adviceList, nil
	}
}

// Batch builds a whole-script rule whose render callback chooses the advice code, content, and position.
func (e *V1Engine[N]) Batch(
	fn func(Context, []N) []v1review.Finding,
	render func([]V1Stmt[N], v1review.Finding) (code.Code, string, *storepb.Position, bool),
) Func {
	return e.BatchStmts(func(checkCtx Context, _ []V1Stmt[N], nodes []N) []v1review.Finding {
		return fn(checkCtx, nodes)
	}, render)
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
	return e.Batch(fn, func(stmts []V1Stmt[N], f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
		if filterCode != "" && f.Code != filterCode {
			return 0, "", nil, false
		}
		var p *storepb.Position
		if f.Statement >= 0 && f.Statement < len(stmts) {
			p = pos(stmts[f.Statement], f)
		}
		return c, content(f), p, true
	})
}

// SchemaBatch builds a whole-script schema rule that triggers ReadMissingSchema
// when eval returns v1review.ErrSchemaMissing.
func (e *V1Engine[N]) SchemaBatch(
	adviceCode code.Code,
	contentFn func(v1review.Finding) string,
	eval func(Context, []N) ([]v1review.Finding, *model.DatabaseMetadata, error),
	posFn ...V1PosFunc[N],
) Func {
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		findings, missingMD, err := eval(checkCtx, nodes)
		if err != nil {
			ReadMissingSchema(missingMD, err)
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range findings {
			adviceList = append(adviceList, newAdvice(status, adviceCode, title, contentFn(f), pos(stmts[f.Statement], f)))
		}
		return adviceList, nil
	}
}

// IndexTotalNumberLimit builds INDEX_TOTAL_NUMBER_LIMIT against FinalMetadata.
func (e *V1Engine[N]) IndexTotalNumberLimit(
	fn func([]N, int, v1review.Schema) ([]v1review.Finding, error),
	contentFn func(v1review.Finding, int) string,
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
		stmts, nodes, err := e.Nodes(checkCtx)
		if err != nil {
			return nil, err
		}
		findings, err := fn(nodes, limit, V1SchemaOf(checkCtx.FinalMetadata))
		if err != nil {
			ReadMissingSchema(checkCtx.FinalMetadata, err)
			return nil, err
		}
		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		for _, f := range findings {
			adviceList = append(adviceList, newAdvice(level, code.IndexCountExceedsLimit, title, contentFn(f, limit), pos(stmts[f.Statement], f)))
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
	return e.NumberStmtWithMsg(fn, "", positiveOnly, c, content, posFn...)
}

// NumberStmtWithMsg builds a per-statement rule parameterized by number_payload with a custom missing-payload error.
func (e *V1Engine[N]) NumberStmtWithMsg(
	fn func([]N, int) []v1review.Finding,
	missingMsg string,
	positiveOnly bool,
	c code.Code,
	content func(V1Stmt[N], v1review.Finding, int) string,
	posFn ...V1PosFunc[N],
) Func {
	if missingMsg == "" {
		missingMsg = "number_payload is required for this rule"
	}
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		limit, err := RequireNumberPayload(checkCtx.Rule, missingMsg)
		if err != nil {
			return nil, err
		}
		if positiveOnly && limit <= 0 {
			return nil, nil
		}
		stmts, nodes, err := e.Nodes(checkCtx)
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
		stmts, nodes, err := e.Nodes(checkCtx)
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

// ColumnRequired builds COLUMN_REQUIRED with grouped missing-column rendering.
func (e *V1Engine[N]) ColumnRequired(cfg V1ColumnRequiredConfig[N]) Func {
	pos := cfg.Pos
	if pos == nil {
		pos = e.defaultPos
	}
	tableFmt := cfg.TableFmt
	if tableFmt == "" {
		tableFmt = "Table %q requires columns: %s"
	}
	renderGroups := func(level storepb.Advice_Status, title string, stmts []V1Stmt[N], stmtIdx int, findings []v1review.Finding, adviceList []*storepb.Advice) []*storepb.Advice {
		for j := 0; j < len(findings); {
			f := findings[j]
			var missing []string
			for ; j < len(findings) && (cfg.SameGroup == nil || cfg.SameGroup(f, findings[j])); j++ {
				col := findings[j].Name
				if cfg.QuoteCol {
					col = fmt.Sprintf("%q", col)
				}
				missing = append(missing, col)
			}
			idx := stmtIdx
			if idx < 0 {
				idx = f.Statement
			}
			adviceList = append(adviceList, newAdvice(level, code.NoRequiredColumn, title, fmt.Sprintf(tableFmt, f.Table, strings.Join(missing, ", ")), pos(stmts[idx], f)))
		}
		return adviceList
	}
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		list, err := RequireStringArrayPayload(checkCtx.Rule, "string_array_payload is required for column required rule")
		if err != nil {
			return nil, err
		}
		stmts, nodes, err := e.Nodes(checkCtx)
		if err != nil {
			return nil, err
		}
		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		if cfg.Batch {
			return renderGroups(level, title, stmts, -1, cfg.Fn(nodes, list), adviceList), nil
		}
		for i := range stmts {
			adviceList = renderGroups(level, title, stmts, i, cfg.Fn(nodes[i:i+1:i+1], list), adviceList)
		}
		return adviceList, nil
	}
}

// NamingRule builds a regex naming convention rule with the standard mismatch
// and optional max-length advice messages.
func (e *V1Engine[N]) NamingRule(
	fn func([]N, v1review.NamingConfig) ([]v1review.Finding, error),
	adviceCode code.Code,
	missingMsg string,
	kind string,
	checkLength bool,
	subject func(v1review.Finding) string,
	posFn ...V1PosFunc[N],
) Func {
	mismatch := func(f v1review.Finding, format string) string {
		return fmt.Sprintf("%s mismatches %s naming convention, naming format should be %q", subject(f), kind, format)
	}
	var tooLong func(v1review.Finding, int) string
	if checkLength {
		tooLong = func(f v1review.Finding, maxLength int) string {
			return fmt.Sprintf("%s mismatches %s naming convention, its length should be within %d characters", subject(f), kind, maxLength)
		}
	}
	return e.NamingRegex(fn, adviceCode, missingMsg, mismatch, tooLong, posFn...)
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
		stmts, nodes, err := e.Nodes(checkCtx)
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

// NamingIdentifierCase builds NAMING_IDENTIFIER_CASE. When alwaysUpper is true,
// a missing payload is accepted and findings always report upper case.
func (e *V1Engine[N]) NamingIdentifierCase(
	fn func([]N, v1review.CaseConfig) []v1review.Finding,
	alwaysUpper bool,
	posFn ...V1PosFunc[N],
) Func {
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		payload := checkCtx.Rule.GetNamingCasePayload()
		if !alwaysUpper && payload == nil {
			return nil, errors.New("naming_case_payload is required for this rule")
		}
		cfg := v1review.CaseConfig{Upper: payload.GetUpper()}
		stmts, nodes, err := e.Nodes(checkCtx)
		if err != nil {
			return nil, err
		}
		caseLabel := "lower"
		if alwaysUpper || cfg.Upper {
			caseLabel = "upper"
		}
		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			for _, f := range fn(nodes[i:i+1:i+1], cfg) {
				content := fmt.Sprintf("Identifier %q should be %s case", f.Name, caseLabel)
				adviceList = append(adviceList, newAdvice(level, code.NamingCaseMismatch, title, content, pos(stmt, f)))
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
	return e.TemplateNamingWithOptions(fn, adviceCode, errTitle, kindLabel, invalidFmt, nilInvalidPos, V1TemplateNamingOptions[N]{}, posFn...)
}

// TemplateNamingWithOptions builds a per-statement template naming rule with dialect formatting options.
func (e *V1Engine[N]) TemplateNamingWithOptions(
	fn func([]N, v1review.TemplateConfig, v1review.Schema) ([]v1review.Finding, error),
	adviceCode code.Code,
	errTitle string,
	kindLabel string,
	invalidFmt string,
	nilInvalidPos bool,
	opts V1TemplateNamingOptions[N],
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
		stmts, nodes, err := e.Nodes(checkCtx)
		if err != nil {
			return nil, err
		}
		cfg := v1review.TemplateConfig{Format: namingPayload.Format, MaxLength: maxLength}
		schema := V1SchemaOf(checkCtx.OriginalMetadata)
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
					content := ""
					if opts.InvalidContent != nil {
						content = opts.InvalidContent(stmt, f)
					} else {
						content = fmt.Sprintf(invalidFmt, f.Detail)
					}
					adviceList = append(adviceList, newAdvice(level, code.Internal, errTitle, content, p))
				case v1review.NameMismatch:
					content := fmt.Sprintf("%s in table \"%s\" mismatches the naming convention, expect %q but found \"%s\"", kindLabel, f.Table, f.Pattern, f.Name)
					if opts.BacktickQuote {
						content = fmt.Sprintf("%s in table `%s` mismatches the naming convention, expect %q but found `%s`", kindLabel, f.Table, f.Pattern, f.Name)
					}
					adviceList = append(adviceList, newAdvice(level, adviceCode, title, content, p))
				case v1review.NameTooLong:
					content := fmt.Sprintf("%s \"%s\" in table \"%s\" mismatches the naming convention, its length should be within %d characters", kindLabel, f.Name, f.Table, maxLength)
					if opts.BacktickQuote {
						content = fmt.Sprintf("%s `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", kindLabel, f.Name, f.Table, maxLength)
					}
					adviceList = append(adviceList, newAdvice(level, adviceCode, title, content, p))
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
// prepareFn resolves any per-review state (such as schema and search_path) once
// before iterating over statements.
func (e *V1Engine[N]) WhereDisallowFuncAndCalc(
	prepareFn func(Context) func([]N) []v1review.Finding,
	dedup bool,
	posFn ...V1PosFunc[N],
) Func {
	pos := e.pos(posFn)
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		stmts, nodes, status, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		fn := prepareFn(checkCtx)
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			for _, f := range fn(nodes[i : i+1 : i+1]) {
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

// PriorBackup builds BUILTIN_PRIOR_BACKUP_CHECK with shared enable-check, facts
// construction, and default finding rendering.
func (e *V1Engine[N]) PriorBackup(cfg V1PriorBackupConfig[N]) Func {
	return func(ctx context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		if !checkCtx.EnablePriorBackup {
			return nil, nil
		}
		stmts, nodes, level, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		var facts v1review.PriorBackupFacts
		if cfg.Facts != nil {
			facts = cfg.Facts(ctx, checkCtx)
		} else {
			facts = NewPriorBackupFacts(ctx, checkCtx, cfg.Engine)
		}
		findings, err := cfg.Check(checkCtx, stmts, nodes, facts)
		if err != nil {
			return nil, err
		}
		var mixedLine func(int) int
		if cfg.MixedLine != nil {
			mixedLine = func(idx int) int { return cfg.MixedLine(stmts[idx]) }
		}
		var adviceList []*storepb.Advice
		for _, f := range findings {
			var (
				adviceCode code.Code
				content    string
				p          *storepb.Position
				ok         bool
			)
			if cfg.Render != nil {
				adviceCode, content, p, ok = cfg.Render(stmts, nodes, f)
			}
			if !ok {
				adviceCode, content, p, ok = renderPriorBackupFinding(f, cfg.DefaultPos, mixedLine)
			}
			if !ok {
				continue
			}
			adviceList = append(adviceList, newAdvice(level, adviceCode, title, content, p))
		}
		return adviceList, nil
	}
}

// RowLimit builds STATEMENT_INSERT_ROW_LIMIT or STATEMENT_AFFECTED_ROW_LIMIT
// with shared payload validation, ExplainBudget accounting, and finding rendering.
func (e *V1Engine[N]) RowLimit(cfg V1RowLimitConfig[N]) Func {
	pos := cfg.Pos
	if pos == nil {
		pos = e.defaultPos
	}
	stmtText := cfg.StatementText
	if stmtText == nil {
		stmtText = func(stmt V1Stmt[N], _ N) string { return dryRunText(stmt.Text) }
	}
	budgetPos := cfg.BudgetPos
	if budgetPos == nil {
		budgetPos = func(stmt V1Stmt[N], _ N) *storepb.Position { return pos(stmt, v1review.Finding{}) }
	}
	estimate := cfg.Estimate
	if estimate == nil {
		estimate = func(ctx context.Context, checkCtx Context, stmt V1Stmt[N], _ N, _ []string) (v1review.Probe, error) {
			text := stmt.Text
			if cfg.TrimEstimate {
				text = dryRunText(text)
			}
			return Estimate(ctx, checkCtx.DBType, checkCtx.Prober, text)
		}
	}
	return func(ctx context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		maxRow, err := RequireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
		if err != nil {
			return nil, err
		}
		if cfg.PositiveOnly && maxRow <= 0 {
			return nil, nil
		}
		stmts, nodes, err := e.Nodes(checkCtx)
		if err != nil {
			return nil, err
		}
		title := checkCtx.Rule.Type.String()
		var explains ExplainBudget
		probes := make(map[int]v1review.Probe)
		var probeErr error
		visit := func(i int, settings []string) {
			if probeErr != nil {
				return
			}
			if !explains.Spend(budgetPos(stmts[i], nodes[i])) {
				return
			}
			probe, err := estimate(ctx, checkCtx, stmts[i], nodes[i], settings)
			if err != nil {
				probeErr = err
				return
			}
			probes[i] = probe
		}
		want := cfg.Probes(nodes)
		if cfg.PreExecutions != nil {
			cfg.PreExecutions(stmts, nodes, want, visit)
		} else {
			for _, i := range want {
				visit(i, nil)
				if probeErr != nil {
					break
				}
			}
		}
		if probeErr != nil {
			return nil, probeErr
		}

		suffix := ""
		if cfg.ExceededCode == code.StatementAffectedRowExceedsLimit {
			suffix = " (estimated)"
		}
		var adviceList []*storepb.Advice
		for _, f := range cfg.Judge(nodes, probes, maxRow) {
			stmt := stmts[f.Statement]
			text := stmtText(stmt, nodes[f.Statement])
			adviceCode, content := cfg.ExceededCode, ""
			switch f.Code {
			case v1review.DryRunFailed:
				content = fmt.Sprintf("\"%s\" dry runs failed: %s", text, f.Value)
			case v1review.RowCountFailed:
				adviceCode, content = code.Internal, fmt.Sprintf("failed to get row count for \"%s\": %s", text, f.Value)
			case v1review.RowLimitExceeded:
				if cfg.ExceededContent != nil {
					content = cfg.ExceededContent(text, f.Value, maxRow)
				} else {
					content = fmt.Sprintf("\"%s\" %s %s rows%s. The count exceeds %d.", text, cfg.ExceededVerb, f.Value, suffix, maxRow)
				}
			default:
				continue
			}
			adviceList = append(adviceList, newAdvice(level, adviceCode, title, content, pos(stmt, f)))
		}
		return explains.AppendSkippedAdvice(adviceList, title, cfg.ExceededCode), nil
	}
}

// DMLDryRun builds STATEMENT_DML_DRY_RUN with shared DDL-skip, prober checks,
// explain cap, and finding rendering.
func (e *V1Engine[N]) DMLDryRun(cfg V1DMLDryRunConfig[N]) Func {
	pos := cfg.Pos
	if pos == nil {
		pos = e.defaultPos
	}
	probeFn := cfg.Probe
	if probeFn == nil {
		probeFn = func(ctx context.Context, prober db.Prober, stmt V1Stmt[N], settings []string) v1review.Probe {
			if _, err := prober.Probe(ctx, db.ProbeContext{PreExecutions: settings}, fmt.Sprintf("EXPLAIN %s", dryRunText(stmt.Text))); err != nil {
				return v1review.Probe{Error: err.Error()}
			}
			return v1review.Probe{}
		}
	}
	formatContent := cfg.FormatContent
	if formatContent == nil {
		formatContent = func(stmt V1Stmt[N], f v1review.Finding) string {
			return fmt.Sprintf("\"%s\" dry runs failed: %s", dryRunText(stmt.Text), f.Value)
		}
	}
	return func(ctx context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		// BYT-8855: Skip DML dry run if DDL statements are mixed in, because DML
		// statements often reference objects created by DDL statements, causing false positives.
		if ContainsDDL(checkCtx.DBType, checkCtx.ParsedStatements) {
			return nil, nil
		}
		prober, err := ProbeFor(checkCtx)
		if err != nil {
			return nil, err
		}
		stmts, nodes, err := e.Nodes(checkCtx)
		if err != nil {
			return nil, err
		}
		probes := make(map[int]v1review.Probe)
		explained := 0
		visit := func(i int, settings []string) {
			if explained >= common.MaximumLintExplainSize {
				return
			}
			explained++
			probes[i] = probeFn(ctx, prober, stmts[i], settings)
		}
		want := cfg.Probes(nodes)
		if cfg.PreExecutions != nil {
			cfg.PreExecutions(stmts, nodes, want, visit)
		} else {
			for _, i := range want {
				visit(i, nil)
				if explained >= common.MaximumLintExplainSize {
					break
				}
			}
		}

		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		for _, f := range cfg.Judge(nodes, probes) {
			stmt := stmts[f.Statement]
			adviceList = append(adviceList, newAdvice(level, code.StatementDMLDryRunFailed, title, formatContent(stmt, f), pos(stmt, f)))
		}
		return adviceList, nil
	}
}
