package spanner

import (
	googlesqlanalysis "github.com/bytebase/omni/googlesql/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_SPANNER, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for Spanner, which
// takes no engineVersion.
//
// It is omni's RewriteResultLimit: the cap on the outermost query only, never a
// subquery wrapper, where Spanner rejects a WITH clause. Keeping the rule in one
// place matters for the pipe syntax, whose cap is a `|> LIMIT` operator rather
// than a query-level clause — a LIMIT appended after a pipeline is a syntax
// error.
func statementWithResultLimit(stmt string, limit int, _ string) string {
	return googlesqlanalysis.RewriteResultLimit(stmt, limit)
}
