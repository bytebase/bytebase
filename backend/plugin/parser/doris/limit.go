package doris

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/bytebase/omni/doris/ast"
	"github.com/bytebase/omni/doris/parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_DORIS, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for Doris, which
// takes no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	trimmedStatement := strings.TrimSpace(statement)
	if strings.HasPrefix(strings.ToUpper(trimmedStatement), "SHOW") {
		return statement
	}

	query, outfile := splitOutfileClause(statement)
	stmt, err := statementWithResultLimitInline(query, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		if outfile == "" {
			return fmt.Sprintf("SELECT * FROM (%s) result LIMIT %d;", base.TrimStatement(query), limit)
		}
		stmt = fmt.Sprintf("SELECT * FROM (\n%s\n) result LIMIT %d", base.TrimStatement(query), limit)
	}
	if outfile != "" {
		return strings.TrimRight(stmt, " \t\r\n") + "\n" + outfile
	}
	return stmt
}

// The Doris AST cannot represent INTO OUTFILE. Keep this top-level clause outside
// the query while capping it, including when a set operation needs wrapping.
func splitOutfileClause(statement string) (string, string) {
	tokens, errs := parser.Tokenize(statement)
	if len(errs) > 0 {
		return statement, ""
	}
	intoKind, _ := parser.KeywordToken("into")
	depth := 0
	for i, token := range tokens {
		switch token.Kind {
		case '(':
			depth++
		case ')':
			depth--
		case ';':
			return statement, ""
		case intoKind:
			if depth == 0 && i+1 < len(tokens) && tokens[i+1].Kind == outfileTokenKind {
				return statement[:token.Loc.Start], statement[token.Loc.Start:]
			}
		default:
		}
	}
	return statement, ""
}

func statementWithResultLimitInline(statement string, limitCount int) (string, error) {
	if strings.TrimSpace(statement) == "" {
		return "", errors.New("empty statement")
	}

	file, errs := parser.Parse(statement)
	if len(errs) > 0 {
		return "", errors.New(errs[0].Error())
	}
	if file == nil || len(file.Stmts) != 1 {
		stmtCount := 0
		if file != nil {
			stmtCount = len(file.Stmts)
		}
		return "", errors.Errorf("expected exactly one statement, got %d", stmtCount)
	}

	switch stmt := file.Stmts[0].(type) {
	case *ast.SelectStmt:
		return rewriteSelectLimit(statement, stmt, limitCount)
	case *ast.SetOpStmt:
		// SetOpStmt.Limit is unusable: the Doris grammar leaves it nil whether or
		// not the text has a trailing LIMIT (confirmed against omni — a UNION ...
		// LIMIT 5 parses with Limit == nil, its text still inside the node's own
		// Loc). Trusting nil to mean "no LIMIT" would append a second one after
		// the caller's. This is an error, not a pass-through: a set operation is
		// still a query whose rows need capping, so the caller's CTE-wrap
		// fallback must run rather than let it through unbounded.
		return "", errors.New("cannot locate LIMIT on a Doris set operation")
	default:
		// A statement Doris parses to something other than a query — DDL, DML,
		// SHOW, etc. — has no rows to cap; run it unchanged, matching every other
		// engine's treatment of a non-SELECT here.
		return statement, nil
	}
}

func hasUnparsedTail(sql string, locEnd int) bool {
	pos := locEnd
	for pos < len(sql) {
		for pos < len(sql) && (sql[pos] == ' ' || sql[pos] == '\t' || sql[pos] == '\n' || sql[pos] == '\r') {
			pos++
		}
		for pos < len(sql) && sql[pos] == ';' {
			pos++
		}
		for pos < len(sql) && (sql[pos] == ' ' || sql[pos] == '\t' || sql[pos] == '\n' || sql[pos] == '\r') {
			pos++
		}
		if pos >= len(sql) {
			return false
		}
		if strings.HasPrefix(sql[pos:], "--") || strings.HasPrefix(sql[pos:], "#") {
			nl := strings.IndexByte(sql[pos:], '\n')
			if nl < 0 {
				return false
			}
			pos += nl + 1
			continue
		}
		if strings.HasPrefix(sql[pos:], "/*") {
			end := strings.Index(sql[pos:], "*/")
			if end < 0 {
				return false
			}
			pos += end + 2
			continue
		}
		return true
	}
	return false
}

// rewriteExistingLimit caps an already-present LIMIT clause: keep it when it
// is a constant no larger than limitCount, replace the count otherwise.
func rewriteExistingLimit(sql string, limitNode ast.Node, limitCount int) (string, error) {
	limitLoc := literalLoc(limitNode)
	if limitLoc.Start < 0 {
		return "", errors.New("cannot rewrite non-constant LIMIT expression")
	}

	if countStart, countEnd, ok := findCommaLimitCount(sql, limitLoc); ok {
		// A literal 0 count (LIMIT offset,0) is a valid, stricter limit and must
		// be kept, matching the non-comma path below.
		existingCount, _ := strconv.Atoi(sql[countStart:countEnd])
		if existingCount >= 0 && existingCount <= limitCount {
			return sql, nil
		}
		return sql[:countStart] + fmt.Sprintf("%d", limitCount) + sql[countEnd:], nil
	}

	existingLimit := extractLimitValue(limitNode)
	if existingLimit >= 0 && existingLimit <= limitCount {
		return sql, nil
	}
	return sql[:limitLoc.Start] + fmt.Sprintf("%d", limitCount) + sql[limitLoc.End:], nil
}

func rewriteSelectLimit(sql string, stmt *ast.SelectStmt, limitCount int) (string, error) {
	if stmt.Limit != nil {
		return rewriteExistingLimit(sql, stmt.Limit, limitCount)
	}
	if hasUnparsedTail(sql, stmt.Loc.End) {
		return "", errors.New("statement has unparsed tail content")
	}
	return sql[:stmt.Loc.End] + fmt.Sprintf(" LIMIT %d", limitCount) + sql[stmt.Loc.End:], nil
}

func extractLimitValue(node ast.Node) int {
	lit, ok := node.(*ast.Literal)
	if !ok || lit.Kind != ast.LitInt {
		return -1
	}
	val, err := strconv.Atoi(lit.Value)
	if err != nil {
		return -1
	}
	return val
}

func literalLoc(node ast.Node) ast.Loc {
	lit, ok := node.(*ast.Literal)
	if !ok {
		return ast.Loc{Start: -1, End: -1}
	}
	return lit.Loc
}

func findCommaLimitCount(sql string, limitLoc ast.Loc) (countStart, countEnd int, found bool) {
	pos := limitLoc.End
	pos = skipWhitespaceAndComments(sql, pos)
	if pos >= len(sql) || sql[pos] != ',' {
		return 0, 0, false
	}
	pos++
	pos = skipWhitespaceAndComments(sql, pos)
	countStart = pos
	for pos < len(sql) && sql[pos] >= '0' && sql[pos] <= '9' {
		pos++
	}
	if pos == countStart {
		return 0, 0, false
	}
	return countStart, pos, true
}

func skipWhitespaceAndComments(sql string, pos int) int {
	for pos < len(sql) {
		for pos < len(sql) && (sql[pos] == ' ' || sql[pos] == '\t' || sql[pos] == '\n' || sql[pos] == '\r') {
			pos++
		}
		if strings.HasPrefix(sql[pos:], "--") || strings.HasPrefix(sql[pos:], "#") {
			nl := strings.IndexByte(sql[pos:], '\n')
			if nl < 0 {
				return len(sql)
			}
			pos += nl + 1
			continue
		}
		if strings.HasPrefix(sql[pos:], "/*") {
			end := strings.Index(sql[pos:], "*/")
			if end < 0 {
				return len(sql)
			}
			pos += end + 2
			continue
		}
		return pos
	}
	return pos
}
