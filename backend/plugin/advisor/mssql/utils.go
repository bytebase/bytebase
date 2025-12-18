package mssql

import (
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// getANTLRTree extracts TSQL ANTLR parse trees from the advisor context.
func getANTLRTree(checkCtx advisor.Context) ([]*base.ParseResult, error) {
	return advisor.GetANTLRParseResults(checkCtx)
}
