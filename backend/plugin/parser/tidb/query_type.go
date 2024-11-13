package tidb

import (
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func getQueryType(node tidbast.Node, allSystem bool) (base.QueryType, bool) {
	switch n := node.(type) {
	case *tidbast.SelectStmt, *tidbast.SetOprStmt:
		if allSystem {
			return base.SelectInfoSchema, false
		}
		return base.Select, false
	case *tidbast.ExplainStmt:
		if n.Analyze {
			t, _ := getQueryType(n.Stmt, allSystem)
			return t, true
		}
		return base.Explain, false
	case *tidbast.ShowStmt:
		return base.SelectInfoSchema, false
	case tidbast.DMLNode:
		// The order between DMLNode and SelectStmt/SetOprStmt is important.
		// Here means all DMLNodes except SelectStmt and SetOprStmt.
		return base.DML, false
	case tidbast.DDLNode:
		return base.DDL, false
	case *tidbast.SetStmt, *tidbast.SetConfigStmt, *tidbast.SetResourceGroupStmt, *tidbast.SetRoleStmt, *tidbast.SetBindingStmt:
		// Treat SAFE SET as select statement.
		return base.Select, false
	}

	return base.QueryTypeUnknown, false
}
