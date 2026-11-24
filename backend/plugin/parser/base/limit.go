package base

import (
	"fmt"
	"log/slog"
	"strings"
	"unicode"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// ResultLimitFunc returns statement rewritten so its outermost query returns at
// most limit rows. engineVersion is the connected server's version string; every
// engine but Oracle ignores it (Oracle's clause depends on it — ROWNUM before
// version 12, FETCH NEXT from 12c on).
//
// A function here never fails outward: each implementation tries a precise
// rewrite and falls back to a wrapping subquery on any error, logging why. That
// mirrors every registered function's actual shape, so the registry does not
// carry an error return only every driver would immediately discard.
type ResultLimitFunc func(statement string, limit int, engineVersion string) string

// StatementWithResultLimit returns statement rewritten to return at most limit
// rows, for engines that enforce the cap in SQL text. An engine with nothing
// registered — because its driver caps rows another way (Hive trims by result
// size; MongoDB, Redis and the other non-SQL stores have no row cap here at
// all) — returns statement unchanged; callers do not need to ask first.
func StatementWithResultLimit(engine storepb.Engine, statement string, limit int, engineVersion string) string {
	f := engines[engine].ResultLimit
	if f == nil {
		return statement
	}
	return f(statement, limit, engineVersion)
}

// TrimStatement trims the whitespace and trailing semicolons a limit rewrite's
// fallback wrapper (a subquery or CTE) would otherwise leave stray inside itself.
func TrimStatement(statement string) string {
	return strings.TrimLeftFunc(strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon), unicode.IsSpace)
}

// RewriteResultLimit runs rewrite and, on error, logs the failure and formats
// fallbackFmt with TrimStatement(statement) and limit.
func RewriteResultLimit(statement string, limit int, rewrite func(string, int) (string, error), fallbackFmt string) string {
	stmt, err := rewrite(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		return fmt.Sprintf(fallbackFmt, TrimStatement(statement), limit)
	}
	return stmt
}
