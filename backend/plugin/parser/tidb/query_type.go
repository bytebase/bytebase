package tidb

import (
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func getQueryType(node tidbast.Node, allSystem bool) base.QueryType {
	switch n := node.(type) {
	case *tidbast.SelectStmt, *tidbast.SetOprStmt:
		if allSystem {
			return base.SelectInfoSchema
		}
		return base.Select
	case *tidbast.ExplainStmt:
		if n.Analyze {
			return base.ExplainAnalyze
		}
		return base.Explain
	case *tidbast.ShowStmt:
		return base.SelectInfoSchema
	case tidbast.DMLNode:
		// The order between DMLNode and SelectStmt/SetOprStmt is important.
		// Here means all DMLNodes except SelectStmt and SetOprStmt.
		return base.DML
	case tidbast.DDLNode:
		return base.DDL
	}

	return base.QueryTypeUnknown
}
