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

// NodeExtractor returns an extract function for NewV1Engine / MySQLFamilyOptions.
func NodeExtractor[N any](getNode func(base.AST) (N, bool)) func(Context) ([]V1Stmt[N], error) {
	return func(checkCtx Context) ([]V1Stmt[N], error) {
		return ExtractNodes(checkCtx.ParsedStatements, getNode), nil
	}
}

func v1Nodes[N any](stmts []V1Stmt[N]) []N {
	nodes := make([]N, len(stmts))
	for i, s := range stmts {
		nodes[i] = s.Node
	}
	return nodes
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

// RequireStringArrayPayload returns the string array payload list or an error with msg when absent.
func RequireStringArrayPayload(rule *storepb.SQLReviewRule, msg string) ([]string, error) {
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

// NewPriorBackupFacts builds PriorBackupFacts for engines whose backup database
// existence is checked via checkCtx.ListDatabaseNamesFunc.
func NewPriorBackupFacts(ctx context.Context, checkCtx Context, engine storepb.Engine) v1review.PriorBackupFacts {
	databaseName := common.BackupDatabaseNameOfEngine(engine)
	return v1review.PriorBackupFacts{
		TotalSize:      checkCtx.StatementsTotalSize,
		MaxSize:        common.MaxSheetCheckSize,
		Database:       databaseName,
		DatabaseExists: DatabaseExists(ctx, checkCtx, databaseName),
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

		facts := NewPriorBackupFacts(ctx, checkCtx, engine)
		schema := v1review.Schema{Metadata: checkCtx.DBSchema, ObjectCaseSensitive: checkCtx.IsObjectCaseSensitive}
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
