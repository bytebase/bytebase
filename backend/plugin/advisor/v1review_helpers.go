package advisor

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// V1Stmt pairs an engine-specific AST node with the statement text, 0-based
// line offset, and original splitter start position.
type V1Stmt[N any] struct {
	Node     N
	Text     string
	BaseLine int
	Start    *storepb.Position
}

// extractNodes unwraps parsed statements into V1Stmt values using getNode.
func extractNodes[N any](parsed []base.ParsedStatement, getNode func(base.AST) (N, bool)) []V1Stmt[N] {
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
			Start:    stmt.Start,
		})
	}
	return stmts
}

// FmtFindingName returns a finding content formatter substituting f.Name into format.
func FmtFindingName(format string) func(v1review.Finding) string {
	return func(f v1review.Finding) string { return fmt.Sprintf(format, f.Name) }
}

// BacktickName wraps f.Name in backticks.
func BacktickName(f v1review.Finding) string {
	return fmt.Sprintf("`%s`", f.Name)
}

// DoubleQuotedName wraps f.Name in double quotes without Go escaping.
func DoubleQuotedName(f v1review.Finding) string {
	return fmt.Sprintf(`"%s"`, f.Name)
}

// NodeExtractor returns an extract function for NewV1Engine / MySQLFamilyOptions.
func NodeExtractor[N any](getNode func(base.AST) (N, bool)) func(Context) ([]V1Stmt[N], error) {
	return func(checkCtx Context) ([]V1Stmt[N], error) {
		return extractNodes(checkCtx.ParsedStatements, getNode), nil
	}
}

func v1Nodes[N any](stmts []V1Stmt[N]) []N {
	nodes := make([]N, len(stmts))
	for i, s := range stmts {
		nodes[i] = s.Node
	}
	return nodes
}

func defaultByteOffsetLine(sql string, byteOffset int) int32 {
	return base.ByteOffsetToRunePosition(sql, byteOffset).Line
}

func contentStartIndex(text string) int {
	for i, c := range text {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return i
		}
	}
	return 0
}

func firstNonSpaceIndex(text string) int {
	for i, ch := range text {
		if !unicode.IsSpace(ch) {
			return i
		}
	}
	return 0
}

func trimmedStmtText(text string) string {
	return strings.TrimRight(strings.TrimSpace(text), ";")
}

func dryRunText(text string) string {
	return trimmedStmtText(text) + ";"
}

func newAdvice(status storepb.Advice_Status, c code.Code, title, content string, pos *storepb.Position) *storepb.Advice {
	return &storepb.Advice{Status: status, Code: c.Int32(), Title: title, Content: content, StartPosition: pos}
}

// V1SchemaOf converts a DatabaseMetadata snapshot to a v1review.Schema.
func V1SchemaOf(m *model.DatabaseMetadata) v1review.Schema {
	if m == nil {
		return v1review.Schema{}
	}
	return v1review.Schema{Metadata: m.GetProto(), ObjectCaseSensitive: m.GetIsObjectCaseSensitive()}
}

// V1SchemaPtrOf converts a DatabaseMetadata snapshot to *v1review.Schema, returning nil for nil metadata.
func V1SchemaPtrOf(m *model.DatabaseMetadata) *v1review.Schema {
	if m == nil {
		return nil
	}
	s := V1SchemaOf(m)
	return &s
}

// readMissingSchema repeats an unguarded read when omni reports ErrSchemaMissing.
func readMissingSchema(m *model.DatabaseMetadata, err error) {
	if err == v1review.ErrSchemaMissing {
		m.GetSchemaMetadata("")
	}
}

// requireNumberPayload returns the integer number payload or an error with msg when absent.
func requireNumberPayload(rule *storepb.SQLReviewRule, msg string) (int, error) {
	p := rule.GetNumberPayload()
	if p == nil {
		return 0, errors.New(msg)
	}
	return int(p.Number), nil
}

// requireStringArrayPayload returns the string array payload list or an error with msg when absent.
func requireStringArrayPayload(rule *storepb.SQLReviewRule, msg string) ([]string, error) {
	p := rule.GetStringArrayPayload()
	if p == nil {
		return nil, errors.New(msg)
	}
	return p.List, nil
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

// newPriorBackupFacts builds PriorBackupFacts for engines whose backup database
// existence is checked via checkCtx.ListDatabaseNamesFunc.
func newPriorBackupFacts(ctx context.Context, checkCtx Context, engine storepb.Engine) v1review.PriorBackupFacts {
	databaseName := common.BackupDatabaseNameOfEngine(engine)
	return v1review.PriorBackupFacts{
		TotalSize:      checkCtx.StatementsTotalSize,
		MaxSize:        common.MaxSheetCheckSize,
		Database:       databaseName,
		DatabaseExists: DatabaseExists(ctx, checkCtx, databaseName),
	}
}

// NewMySQLPriorBackupAdvisor returns a Func for BUILTIN_PRIOR_BACKUP_CHECK
// shared by MySQL and MariaDB, reusing e's memoized statement extraction.
func NewMySQLPriorBackupAdvisor[N, P any](
	e *V1Engine[N],
	engine storepb.Engine,
	makeStmt func(N, string) P,
	checkFn func([]P, v1review.Schema, v1review.PriorBackupFacts) ([]v1review.Finding, error),
) Func {
	return e.PriorBackup(V1PriorBackupConfig[N]{
		Engine: engine,
		Check: func(checkCtx Context, stmts []V1Stmt[N], _ []N, facts v1review.PriorBackupFacts) ([]v1review.Finding, error) {
			pStmts := make([]P, len(stmts))
			for i, s := range stmts {
				pStmts[i] = makeStmt(s.Node, s.Text)
			}
			findings, err := checkFn(pStmts, e.SchemaOf(checkCtx), facts)
			if err != nil {
				return nil, errors.Wrap(err, "failed to prepare transformation")
			}
			return findings, nil
		},
		MixedLine: func(stmt V1Stmt[N]) int { return stmt.BaseLine },
	})
}

func renderPriorBackupFinding(
	f v1review.Finding,
	defaultPos *storepb.Position,
	mixedLine func(int) int,
) (code.Code, string, *storepb.Position, bool) {
	switch f.Code {
	case v1review.PriorBackupScriptTooLarge:
		return code.BuiltinPriorBackupCheck, fmt.Sprintf("The size of the SQL statements exceeds the maximum limit of %s bytes for backup", f.Value), defaultPos, true
	case v1review.PriorBackupMixedDDLAndDML:
		pos := defaultPos
		if mixedLine != nil {
			pos = base.LinePosition(mixedLine(f.Statement))
		}
		return code.BuiltinPriorBackupCheck, "Prior backup cannot deal with mixed DDL and DML statements", pos, true
	case v1review.PriorBackupDDLInBatch:
		return code.BuiltinPriorBackupCheck, "Prior backup cannot deal with mixed DDL and DML statements", defaultPos, true
	case v1review.PriorBackupDatabaseMissing:
		return code.DatabaseNotExists, fmt.Sprintf("Need database %q to do prior backup but it does not exist", f.Name), defaultPos, true
	case v1review.PriorBackupMixedDMLOnTable:
		return code.BuiltinPriorBackupCheck, fmt.Sprintf("Prior backup cannot handle mixed DML statements on the same table %q", f.Table), defaultPos, true
	case v1review.PriorBackupTooManyDMLStatements:
		return code.BuiltinPriorBackupCheck, fmt.Sprintf("Prior backup cannot handle more than %s DML statements across different tables", f.Value), defaultPos, true
	default:
		return 0, "", nil, false
	}
}
