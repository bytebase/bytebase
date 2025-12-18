package oceanbase

import (
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// getANTLRTree extracts MySQL ANTLR parse trees from the advisor context.
// OceanBase uses the MySQL parser.
func getANTLRTree(checkCtx advisor.Context) ([]*base.ParseResult, error) {
	return advisor.GetANTLRParseResults(checkCtx)
}
