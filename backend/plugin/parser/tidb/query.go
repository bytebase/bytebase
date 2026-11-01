package tidb

import (
	"github.com/pingcap/tidb/pkg/parser/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_TIDB, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
// Only SELECT, EXPLAIN, SHOW, SET and DESCRIBE are allowed in read-only mode.
// EXPLAIN ANALYZE executes what it wraps, so it is allowed only over a read.
func validateQuery(statement string) (bool, bool, error) {
	stmtList, err := ParseTiDB(statement, "", "")
	if err != nil {
		return false, false, err
	}
	hasExecute := false
	readOnly := true
	for _, stmt := range stmtList {
		switch stmt := stmt.(type) {
		case *ast.SelectStmt, *ast.SetOprStmt:
			// INTO OUTFILE writes a file on the database server, so the
			// statement is not a read whatever the query itself does — pg
			// refuses its own SELECT ... INTO on the same grounds.
			if hasSelectInto(stmt) {
				return false, false, nil
			}
		case *ast.SetStmt:
			hasExecute = true
		case *ast.ShowStmt:
		case *ast.ExplainStmt:
			if !explainReads(stmt) {
				readOnly = false
			}
		default:
			return false, false, nil
		}
	}
	return readOnly, !hasExecute, nil
}

// ExtractMySQLTableList extracts all the TableNames from node.
// If asName is true, extract AsName prior to OrigName.
func ExtractMySQLTableList(in ast.Node, asName bool) []*ast.TableName {
	input := []*ast.TableName{}
	return extractTableList(in, input, asName)
}

// Authorization needs every base table, including tables nested inside expressions
// and derived queries. The alias-oriented extractor below can collapse those tables.
type accessTableVisitor struct {
	tables        []*ast.TableName
	resolveCTEs   bool
	cteScopes     []map[string]bool
	lockingTables map[*ast.TableName]bool
}

func (v *accessTableVisitor) Enter(node ast.Node) (ast.Node, bool) {
	if selectStmt, ok := node.(*ast.SelectStmt); ok && selectStmt.LockInfo != nil {
		if v.lockingTables == nil {
			v.lockingTables = make(map[*ast.TableName]bool)
		}
		// Locking clauses reference FROM aliases, whose physical sources are visited separately.
		for _, table := range selectStmt.LockInfo.Tables {
			v.lockingTables[table] = true
		}
	}
	if v.resolveCTEs {
		switch n := node.(type) {
		case *ast.SelectStmt, *ast.SetOprStmt, *ast.SetOprSelectList, *ast.UpdateStmt, *ast.DeleteStmt:
			v.cteScopes = append(v.cteScopes, make(map[string]bool))
		case *ast.CommonTableExpression:
			if n.IsRecursive {
				v.cteScopes[len(v.cteScopes)-1][n.Name.L] = true
			}
		default:
		}
	}
	if table, ok := node.(*ast.TableName); ok {
		if v.lockingTables[table] {
			return node, true
		}
		if v.resolveCTEs && table.Schema.O == "" {
			for _, scope := range v.cteScopes {
				if scope[table.Name.L] {
					return node, true
				}
			}
		}
		v.tables = append(v.tables, table)
	}
	return node, false
}

func (v *accessTableVisitor) Leave(node ast.Node) (ast.Node, bool) {
	if v.resolveCTEs {
		switch n := node.(type) {
		case *ast.SelectStmt, *ast.SetOprStmt, *ast.SetOprSelectList, *ast.UpdateStmt, *ast.DeleteStmt:
			v.cteScopes = v.cteScopes[:len(v.cteScopes)-1]
		case *ast.CommonTableExpression:
			// Non-recursive CTEs become visible only after their definition.
			v.cteScopes[len(v.cteScopes)-1][n.Name.L] = true
		default:
		}
	}
	return node, true
}

// -------------------------------------------- DO NOT TOUCH --------------------------------------------

// extractTableList extracts all the TableNames from node.
// If asName is true, extract AsName prior to OrigName.
// Privilege check should use OrigName, while expression may use AsName.
// WARNING: copy from TiDB core code, do NOT touch!
func extractTableList(node ast.Node, input []*ast.TableName, asName bool) []*ast.TableName {
	switch x := node.(type) {
	case *ast.SelectStmt:
		if x.From != nil {
			input = extractTableList(x.From.TableRefs, input, asName)
		}
		if x.Where != nil {
			input = extractTableList(x.Where, input, asName)
		}
		if x.With != nil {
			for _, cte := range x.With.CTEs {
				input = extractTableList(cte.Query, input, asName)
			}
		}
		for _, f := range x.Fields.Fields {
			if s, ok := f.Expr.(*ast.SubqueryExpr); ok {
				input = extractTableList(s, input, asName)
			}
		}
	case *ast.DeleteStmt:
		input = extractTableList(x.TableRefs.TableRefs, input, asName)
		if x.IsMultiTable {
			for _, t := range x.Tables.Tables {
				input = extractTableList(t, input, asName)
			}
		}
		if x.Where != nil {
			input = extractTableList(x.Where, input, asName)
		}
		if x.With != nil {
			for _, cte := range x.With.CTEs {
				input = extractTableList(cte.Query, input, asName)
			}
		}
	case *ast.UpdateStmt:
		input = extractTableList(x.TableRefs.TableRefs, input, asName)
		for _, e := range x.List {
			input = extractTableList(e.Expr, input, asName)
		}
		if x.Where != nil {
			input = extractTableList(x.Where, input, asName)
		}
		if x.With != nil {
			for _, cte := range x.With.CTEs {
				input = extractTableList(cte.Query, input, asName)
			}
		}
	case *ast.InsertStmt:
		input = extractTableList(x.Table.TableRefs, input, asName)
		input = extractTableList(x.Select, input, asName)
	case *ast.SetOprStmt:
		l := &ast.SetOprSelectList{}
		unfoldSelectList(x.SelectList, l)
		for _, s := range l.Selects {
			input = extractTableList(s.(ast.ResultSetNode), input, asName)
		}
	case *ast.PatternInExpr:
		if s, ok := x.Sel.(*ast.SubqueryExpr); ok {
			input = extractTableList(s, input, asName)
		}
	case *ast.ExistsSubqueryExpr:
		if s, ok := x.Sel.(*ast.SubqueryExpr); ok {
			input = extractTableList(s, input, asName)
		}
	case *ast.BinaryOperationExpr:
		if s, ok := x.R.(*ast.SubqueryExpr); ok {
			input = extractTableList(s, input, asName)
		}
	case *ast.SubqueryExpr:
		input = extractTableList(x.Query, input, asName)
	case *ast.Join:
		input = extractTableList(x.Left, input, asName)
		input = extractTableList(x.Right, input, asName)
	case *ast.TableSource:
		if s, ok := x.Source.(*ast.TableName); ok {
			if x.AsName.L != "" && asName {
				newTableName := *s
				newTableName.Name = x.AsName
				newTableName.Schema = ast.NewCIStr("")
				input = append(input, &newTableName)
			} else {
				input = append(input, s)
			}
		} else if s, ok := x.Source.(*ast.SelectStmt); ok {
			if s.From != nil {
				var innerList []*ast.TableName
				innerList = extractTableList(s.From.TableRefs, innerList, asName)
				if len(innerList) > 0 {
					innerTableName := innerList[0]
					if x.AsName.L != "" && asName {
						newTableName := *innerList[0]
						newTableName.Name = x.AsName
						newTableName.Schema = ast.NewCIStr("")
						innerTableName = &newTableName
					}
					input = append(input, innerTableName)
				}
			}
		} else if s, ok := x.Source.(*ast.SetOprStmt); ok {
			// BYT-8415: Handle UNION ALL and other set operations in FROM subqueries
			input = extractTableList(s, input, asName)
		}
	default:
	}
	return input
}

// WARNING: copy from TiDB core code, do NOT touch!
func unfoldSelectList(list *ast.SetOprSelectList, unfoldList *ast.SetOprSelectList) {
	for _, sel := range list.Selects {
		switch s := sel.(type) {
		case *ast.SelectStmt:
			unfoldList.Selects = append(unfoldList.Selects, s)
		case *ast.SetOprSelectList:
			unfoldSelectList(s, unfoldList)
		default:
		}
	}
}

// explainReads reports whether an EXPLAIN is a read. The two EXPLAIN forms are
// decided here together so the export gate below is the only one either goes
// through: a second gate on the ANALYZE path could only drift from this one.
//
// The parser gives every EXPLAIN form this handles a statement to wrap, DESCRIBE
// included; statement-less `EXPLAIN FOR CONNECTION` parses to ExplainForStmt,
// which the validator refuses by default. So the nil guard is for safety here,
// not a policy.
func explainReads(explain *ast.ExplainStmt) bool {
	if explain.Stmt == nil {
		return false
	}
	// An export is refused under either form. A plain EXPLAIN does not run it,
	// but the editor must not label as a read what it refuses when asked
	// directly.
	if hasSelectInto(explain.Stmt) {
		return false
	}
	if !explain.Analyze {
		// A plain EXPLAIN only plans, so it reads whatever it wraps, DML
		// included — the editor has always allowed that.
		return true
	}
	// EXPLAIN ANALYZE runs the statement to report actual row counts and
	// timings, so only the SELECT family is a read. Postgres gates its own
	// EXPLAIN ANALYZE the same way. The refusal matters most for DML: EXPLAIN
	// ANALYZE DELETE really deletes, and a plain EXPLAIN does not.
	switch explain.Stmt.(type) {
	case *ast.SelectStmt, *ast.SetOprStmt:
		return true
	}
	return false
}

// hasSelectInto reports whether any SELECT in node carries an INTO clause.
//
// The whole tree is searched rather than the node the caller holds, because the
// parser attaches INTO to an arm of a set operation and not to the set
// operation itself: `SELECT 1 UNION SELECT a FROM t INTO OUTFILE '/x'` is a
// SetOprStmt whose second arm carries the export. The parser also accepts INTO
// inside a subquery or a CTE, so the search reaches statements whose root is an
// ordinary SELECT; refusing those is the conservative answer for a clause that
// writes a file on the server, and it is what the MySQL validator already does.
//
// Every INTO form counts, not just OUTFILE: the parser rejects DUMPFILE and
// INTO <variable> outright, so OUTFILE is the only form that reaches here
// today and any form a later parser accepts is refused by default.
func hasSelectInto(node ast.Node) bool {
	v := &selectIntoVisitor{}
	node.Accept(v)
	return v.found
}

// selectIntoVisitor finds a SELECT carrying an INTO clause.
type selectIntoVisitor struct {
	found bool
}

func (v *selectIntoVisitor) Enter(node ast.Node) (ast.Node, bool) {
	if sel, ok := node.(*ast.SelectStmt); ok && sel.SelectIntoOpt != nil {
		v.found = true
	}
	return node, v.found
}

func (*selectIntoVisitor) Leave(node ast.Node) (ast.Node, bool) {
	return node, true
}
