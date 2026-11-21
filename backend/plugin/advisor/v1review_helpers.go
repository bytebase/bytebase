package advisor

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"unicode"

	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// V1Stmt pairs an engine-specific AST node with the statement text and 0-based
// line offset from which it was parsed.
type V1Stmt[N any] struct {
	Node     N
	Text     string
	BaseLine int
}

// ExtractNodes unwraps parsed statements into V1Stmt values using getNode.
func ExtractNodes[N any](parsed []base.ParsedStatement, getNode func(base.AST) (N, bool)) []V1Stmt[N] {
	stmts := make([]V1Stmt[N], 0, len(parsed))
	for _, stmt := range parsed {
		if stmt.AST == nil {
			continue
		}
		node, ok := getNode(stmt.AST)
		if !ok {
			continue
		}
		stmts = append(stmts, V1Stmt[N]{
			Node:     node,
			Text:     stmt.Text,
			BaseLine: stmt.BaseLine(),
		})
	}
	return stmts
}

var v1RunnerMemoSeq atomic.Int64

// v1Runner holds the engine-specific extraction and V1 position helpers shared
// across the MySQL-family v1review rule adapters.
type v1Runner[N any] struct {
	extract             func(Context) ([]V1Stmt[N], error)
	byteOffsetLine      func(sql string, byteOffset int) int32
	memoKey             string
	tidbQuirks          bool
	isAlterTable        func(N) bool
	createTableLocStart func(N) (int, bool)
}

type v1CachedStmts[N any] struct {
	stmts []V1Stmt[N]
	nodes []N
}

func v1Nodes[N any](stmts []V1Stmt[N]) []N {
	nodes := make([]N, len(stmts))
	for i, s := range stmts {
		nodes[i] = s.Node
	}
	return nodes
}

func (r v1Runner[N]) extractNodes(checkCtx Context) ([]V1Stmt[N], []N, error) {
	if r.memoKey != "" {
		if cached, ok := checkCtx.Memo(r.memoKey); ok {
			if c, typeOK := cached.(v1CachedStmts[N]); typeOK {
				return c.stmts, c.nodes, nil
			}
		}
	}
	stmts, err := r.extract(checkCtx)
	if err != nil {
		return nil, nil, err
	}
	nodes := v1Nodes(stmts)
	if r.memoKey != "" {
		checkCtx.SetMemo(r.memoKey, v1CachedStmts[N]{stmts: stmts, nodes: nodes})
	}
	return stmts, nodes, nil
}

func contentStartIndex(text string) int {
	for i, c := range text {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return i
		}
	}
	return 0
}

func (r v1Runner[N]) contentStartLine(text string) int32 {
	idx := contentStartIndex(text)
	if idx <= 0 {
		return 1
	}
	return r.byteOffsetLine(text, idx)
}

func (r v1Runner[N]) contentEndLine(text string) int32 {
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
	return r.byteOffsetLine(text, idx)
}

func (r v1Runner[N]) findLineByName(text, name string) int32 {
	if name == "" || text == "" {
		return r.contentStartLine(text)
	}
	idx := strings.Index(text, name)
	if idx < 0 {
		return r.contentStartLine(text)
	}
	return r.byteOffsetLine(text, idx)
}

func (r v1Runner[N]) findingLine(text string, f v1review.Finding) int32 {
	offset := f.Start
	if f.Start == 0 && f.End == 0 {
		offset = contentStartIndex(text)
	}
	return r.byteOffsetLine(text, offset)
}

func (r v1Runner[N]) absLine(stmt V1Stmt[N], byteOffset int) int {
	return stmt.BaseLine + int(r.byteOffsetLine(stmt.Text, byteOffset))
}

func (r v1Runner[N]) firstTokenLine(stmt V1Stmt[N]) int {
	for i, ch := range stmt.Text {
		if !unicode.IsSpace(ch) {
			return r.absLine(stmt, i)
		}
	}
	return r.absLine(stmt, 0)
}

func (r v1Runner[N]) posFinding(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	if r.tidbQuirks {
		if f.Start == 0 && f.End == 0 {
			return base.ConvertANTLRLineToPosition(r.firstTokenLine(stmt))
		}
		return base.ConvertANTLRLineToPosition(r.absLine(stmt, f.Start))
	}
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(r.findingLine(stmt.Text, f)))
}

func (r v1Runner[N]) posAbsOrFinding(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	if r.tidbQuirks {
		return base.ConvertANTLRLineToPosition(r.absLine(stmt, f.Start))
	}
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(r.findingLine(stmt.Text, f)))
}

func (r v1Runner[N]) posStatementAbsOrStmt(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	if r.tidbQuirks {
		return base.ConvertANTLRLineToPosition(r.absLine(stmt, f.Start))
	}
	return r.posStatement(stmt)
}

func (r v1Runner[N]) posFindingOrTiDBFirstToken(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	if r.tidbQuirks {
		return r.posStatement(stmt)
	}
	return r.posFinding(stmt, f)
}

func (r v1Runner[N]) posStatement(stmt V1Stmt[N]) *storepb.Position {
	if r.tidbQuirks {
		return base.ConvertANTLRLineToPosition(r.firstTokenLine(stmt))
	}
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(r.findingLine(stmt.Text, v1review.Finding{})))
}

func (r v1Runner[N]) posBaseLineTwice(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	if r.tidbQuirks {
		return base.ConvertANTLRLineToPosition(r.absLine(stmt, f.Start))
	}
	p := base.ConvertANTLRLineToPosition(stmt.BaseLine + int(r.findingLine(stmt.Text, f)))
	p.Line += int32(stmt.BaseLine)
	return p
}

func (r v1Runner[N]) posAllowlist(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	if r.tidbQuirks {
		if r.createTableLocStart != nil {
			if start, ok := r.createTableLocStart(stmt.Node); ok && f.Start != max(start, 0) {
				return base.ConvertANTLRLineToPosition(r.absLine(stmt, f.Start))
			}
		}
		return base.ConvertANTLRLineToPosition(r.firstTokenLine(stmt))
	}
	return r.posBaseLineTwice(stmt, f)
}

func (r v1Runner[N]) posAlterFirstToken(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	if r.tidbQuirks {
		if r.isAlterTable != nil && r.isAlterTable(stmt.Node) {
			return base.ConvertANTLRLineToPosition(r.firstTokenLine(stmt))
		}
		return base.ConvertANTLRLineToPosition(r.absLine(stmt, f.Start))
	}
	return r.posFinding(stmt, f)
}

func (r v1Runner[N]) posSearchOrStart(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	line := r.contentStartLine(stmt.Text)
	if f.Search != "" {
		line = r.findLineByName(stmt.Text, f.Search)
	}
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(line))
}

func (r v1Runner[N]) posSearchOrFinding(stmt V1Stmt[N], f v1review.Finding) *storepb.Position {
	line := r.findingLine(stmt.Text, f)
	if f.Search != "" {
		line = r.findLineByName(stmt.Text, f.Search)
	}
	return base.ConvertANTLRLineToPosition(stmt.BaseLine + int(line))
}

func (r v1Runner[N]) stmtText(stmt V1Stmt[N], rawOnTiDB bool) string {
	if r.tidbQuirks && rawOnTiDB {
		return stmt.Text
	}
	return strings.TrimSpace(stmt.Text)
}

func trimmedStmtText(text string) string {
	return strings.TrimRight(strings.TrimSpace(text), ";")
}

func dryRunText(text string) string {
	return trimmedStmtText(text) + ";"
}

func v1SchemaOf(m *model.DatabaseMetadata) v1review.Schema {
	if m == nil {
		return v1review.Schema{}
	}
	return v1review.Schema{Metadata: m.GetProto(), ObjectCaseSensitive: m.GetIsObjectCaseSensitive()}
}

func v1SchemaPtrOf(m *model.DatabaseMetadata) *v1review.Schema {
	if m == nil {
		return nil
	}
	s := v1SchemaOf(m)
	return &s
}

// ReadMissingSchema repeats an unguarded read when omni reports ErrSchemaMissing.
func ReadMissingSchema(m *model.DatabaseMetadata, err error) {
	if err == v1review.ErrSchemaMissing {
		m.GetSchemaMetadata("")
	}
}

// RequireNumberPayload returns the integer number payload or an error with msg when absent.
func RequireNumberPayload(rule *storepb.SQLReviewRule, msg string) (int, error) {
	p := rule.GetNumberPayload()
	if p == nil {
		return 0, errors.New(msg)
	}
	return int(p.Number), nil
}

func requireNamingPayload(rule *storepb.SQLReviewRule, msg string) (*storepb.SQLReviewRule_NamingRulePayload, int, error) {
	p := rule.GetNamingPayload()
	if p == nil {
		return nil, 0, errors.New(msg)
	}
	maxLength := int(p.MaxLength)
	if maxLength == 0 {
		maxLength = DefaultNameLengthLimit
	}
	return p, maxLength, nil
}

var primaryKeyNameRe = regexp.MustCompile(`(?i)PRIMARY\s+KEY\s+` + "`?" + `(\w+)` + "`?" + `\s*\(`)

func keyName(stmtText string, f v1review.Finding, tidbQuirks bool) string {
	if !tidbQuirks && f.Name == "" && f.Value == v1review.KeyPrimary {
		if m := primaryKeyNameRe.FindStringSubmatch(stmtText); len(m) >= 2 {
			return m[1]
		}
	}
	return f.Name
}

// NewMySQLRowLimitAdvisor returns a Func for STATEMENT_INSERT_ROW_LIMIT
// and STATEMENT_AFFECTED_ROW_LIMIT using the shared driver RowEstimator seam.
func NewMySQLRowLimitAdvisor[N any](
	extract func(Context) ([]V1Stmt[N], error),
	byteOffsetLine func(string, int) int32,
	probesFn func([]N) []int,
	judgeFn func([]N, map[int]v1review.Probe, int) []v1review.Finding,
	exceededCode code.Code,
	exceededVerb string,
	trimEstimate bool,
) Func {
	r := v1Runner[N]{extract: extract, byteOffsetLine: byteOffsetLine}
	return func(ctx context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		maxRow, err := RequireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
		if err != nil {
			return nil, err
		}
		title := checkCtx.Rule.Type.String()
		var explains ExplainBudget

		stmts, nodes, err := r.extractNodes(checkCtx)
		if err != nil {
			return nil, err
		}
		probes := make(map[int]v1review.Probe)
		for _, i := range probesFn(nodes) {
			if !explains.Spend(r.posStatement(stmts[i])) {
				continue
			}
			stmtText := stmts[i].Text
			if trimEstimate {
				stmtText = dryRunText(stmtText)
			}
			probe, err := Estimate(ctx, checkCtx.DBType, checkCtx.Prober, stmtText)
			if err != nil {
				return nil, err
			}
			probes[i] = probe
		}

		var advice []*storepb.Advice
		for _, f := range judgeFn(nodes, probes, maxRow) {
			stmt := stmts[f.Statement]
			text := dryRunText(stmt.Text)
			adviceCode, content := exceededCode, ""
			switch f.Code {
			case v1review.DryRunFailed:
				content = fmt.Sprintf("\"%s\" dry runs failed: %s", text, f.Value)
			case v1review.RowCountFailed:
				adviceCode, content = code.Internal, fmt.Sprintf("failed to get row count for \"%s\": %s", text, f.Value)
			case v1review.RowLimitExceeded:
				content = fmt.Sprintf("\"%s\" %s %s rows%s. The count exceeds %d.", text, exceededVerb, f.Value, estimatedSuffix(exceededCode), maxRow)
			default:
				continue
			}
			advice = append(advice, &storepb.Advice{
				Status:        level,
				Code:          adviceCode.Int32(),
				Title:         title,
				Content:       content,
				StartPosition: r.posStatement(stmt),
			})
		}
		return explains.AppendSkippedAdvice(advice, title, exceededCode), nil
	}
}

func estimatedSuffix(c code.Code) string {
	if c == code.StatementAffectedRowExceedsLimit {
		return " (estimated)"
	}
	return ""
}

// NewMySQLDMLDryRunAdvisor returns a Func for STATEMENT_DML_DRY_RUN
// that EXPLAINs selected DML statements through checkCtx.Prober.
func NewMySQLDMLDryRunAdvisor[N any](
	extract func(Context) ([]V1Stmt[N], error),
	byteOffsetLine func(string, int) int32,
	probesFn func([]N) []int,
	judgeFn func([]N, map[int]v1review.Probe) []v1review.Finding,
) Func {
	r := v1Runner[N]{extract: extract, byteOffsetLine: byteOffsetLine}
	return func(ctx context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		if ContainsDDL(checkCtx.DBType, checkCtx.ParsedStatements) {
			return nil, nil
		}
		prober, err := ProbeFor(checkCtx)
		if err != nil {
			return nil, err
		}

		stmts, nodes, err := r.extractNodes(checkCtx)
		if err != nil {
			return nil, err
		}
		probes := make(map[int]v1review.Probe)
		for n, i := range probesFn(nodes) {
			if _, err := prober.Probe(ctx, db.ProbeContext{}, fmt.Sprintf("EXPLAIN %s", dryRunText(stmts[i].Text))); err != nil {
				probes[i] = v1review.Probe{Error: err.Error()}
			} else {
				probes[i] = v1review.Probe{}
			}
			if n+1 >= common.MaximumLintExplainSize {
				break
			}
		}

		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		for _, f := range judgeFn(nodes, probes) {
			stmt := stmts[f.Statement]
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementDMLDryRunFailed.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("\"%s\" dry runs failed: %s", dryRunText(stmt.Text), f.Value),
				StartPosition: r.posStatement(stmt),
			})
		}
		return adviceList, nil
	}
}

// NewMySQLPriorBackupAdvisor returns a Func for BUILTIN_PRIOR_BACKUP_CHECK
// shared by MySQL and MariaDB.
func NewMySQLPriorBackupAdvisor[N, P any](
	engine storepb.Engine,
	extract func(Context) ([]V1Stmt[N], error),
	makeStmt func(N, string) P,
	checkFn func([]P, v1review.Schema, v1review.PriorBackupFacts) ([]v1review.Finding, error),
) Func {
	return func(ctx context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		if !checkCtx.EnablePriorBackup {
			return nil, nil
		}
		level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		stmts, err := extract(checkCtx)
		if err != nil {
			return nil, err
		}
		pStmts := make([]P, len(stmts))
		for i, s := range stmts {
			pStmts[i] = makeStmt(s.Node, s.Text)
		}

		databaseName := common.BackupDatabaseNameOfEngine(engine)
		schema := v1review.Schema{Metadata: checkCtx.DBSchema, ObjectCaseSensitive: checkCtx.IsObjectCaseSensitive}
		facts := v1review.PriorBackupFacts{
			TotalSize:      checkCtx.StatementsTotalSize,
			MaxSize:        common.MaxSheetCheckSize,
			Database:       databaseName,
			DatabaseExists: DatabaseExists(ctx, checkCtx, databaseName),
		}
		findings, err := checkFn(pStmts, schema, facts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to prepare transformation")
		}

		return RenderPriorBackupFindings(checkCtx.Rule.Type.String(), level, findings, func(idx int) int {
			return stmts[idx].BaseLine
		}), nil
	}
}

// RenderPriorBackupFindings renders v1review.PriorBackupCheck findings as Advice.
func RenderPriorBackupFindings(title string, level storepb.Advice_Status, findings []v1review.Finding, lineOf func(int) int) []*storepb.Advice {
	var adviceList []*storepb.Advice
	for _, f := range findings {
		switch f.Code {
		case v1review.PriorBackupScriptTooLarge:
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Title:   title,
				Content: fmt.Sprintf("The size of the SQL statements exceeds the maximum limit of %s bytes for backup", f.Value),
				Code:    code.BuiltinPriorBackupCheck.Int32(),
			})
		case v1review.PriorBackupMixedDDLAndDML:
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Title:         title,
				Content:       "Prior backup cannot deal with mixed DDL and DML statements",
				Code:          code.BuiltinPriorBackupCheck.Int32(),
				StartPosition: base.ConvertANTLRLineToPosition(lineOf(f.Statement)),
			})
		case v1review.PriorBackupDatabaseMissing:
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Title:   title,
				Content: fmt.Sprintf("Need database %q to do prior backup but it does not exist", f.Name),
				Code:    code.DatabaseNotExists.Int32(),
			})
		case v1review.PriorBackupMixedDMLOnTable:
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Title:   title,
				Content: fmt.Sprintf("Prior backup cannot handle mixed DML statements on the same table %q", f.Table),
				Code:    code.BuiltinPriorBackupCheck.Int32(),
			})
		case v1review.PriorBackupTooManyDMLStatements:
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Title:   title,
				Content: fmt.Sprintf("Prior backup cannot handle more than %s DML statements across different tables", f.Value),
				Code:    code.BuiltinPriorBackupCheck.Int32(),
			})
		default:
		}
	}
	return adviceList
}
