package mysql

import (
	"encoding/json"
	"strconv"
	"strings"
	"unicode"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// AffectedRowsQuery returns the statement to EXPLAIN for the rows stmt, parsed from statement,
// modifies. MySQL and MariaDB estimate the rows a single-table UPDATE or DELETE scans rather than
// the rows its WHERE clause keeps, so such a statement is explained as a SELECT of those rows.
func AffectedRowsQuery(stmt ast.Node, statement string) string {
	var loc ast.Loc
	var table *ast.TableRef
	// clausesStart is where the WHERE, ORDER BY, and LIMIT clauses begin.
	var clausesStart int
	switch s := stmt.(type) {
	case *ast.UpdateStmt:
		table = singleTableRef(s.Tables)
		if table == nil || len(s.SetList) == 0 {
			return statement
		}
		loc, clausesStart = s.Loc, s.SetList[len(s.SetList)-1].Loc.End
	case *ast.DeleteStmt:
		table = singleTableRef(s.Tables)
		// DELETE writes an alias before PARTITION, which a SELECT writes after it.
		if table == nil || len(s.Using) > 0 || (table.Alias != "" && len(table.Partitions) > 0) {
			return statement
		}
		loc, clausesStart = s.Loc, table.Loc.End
	default:
		return statement
	}
	if loc.Start < 0 || loc.Start > table.Loc.Start || table.Loc.End > clausesStart || clausesStart > loc.End || loc.End > len(statement) {
		return statement
	}
	// omni positions the text after an executable comment as if the comment's markers were removed, so
	// no SELECT is cut from a statement with one. The SELECT would also leave out an optimizer hint
	// before the table, which can change the plan.
	if strings.Contains(statement, "/*!") || strings.Contains(statement, "/*M!") || strings.Contains(statement[loc.Start:table.Loc.Start], "/*+") {
		return statement
	}
	clauses := strings.TrimSpace(statement[clausesStart:loc.End])
	if rest := trimLeadingComments(clauses); rest != "" && !startsWithKeyword(rest, "WHERE", "ORDER", "LIMIT") {
		return statement
	}
	// The text before the statement holds its WITH clause.
	query := statement[:loc.Start] + "SELECT 1 FROM " + statement[table.Loc.Start:table.Loc.End]
	if clauses != "" {
		query += " " + clauses
	}
	return query
}

func singleTableRef(tables []ast.TableExpr) *ast.TableRef {
	if len(tables) != 1 {
		return nil
	}
	if table, ok := tables[0].(*ast.TableRef); ok {
		return table
	}
	return nil
}

// trimLeadingComments removes the whitespace and ordinary comments at the start of text, where -- opens
// a comment only before whitespace or a control character. Executable comments and optimizer hints
// stay, because they can change the statement.
func trimLeadingComments(text string) string {
	for {
		text = strings.TrimLeftFunc(text, unicode.IsSpace)
		switch {
		case strings.HasPrefix(text, "/*!") || strings.HasPrefix(text, "/*M!") || strings.HasPrefix(text, "/*+"):
			return text
		case strings.HasPrefix(text, "/*"):
			end := strings.Index(text[2:], "*/")
			if end < 0 {
				return text
			}
			text = text[2+end+2:]
		case (strings.HasPrefix(text, "--") && (len(text) == 2 || text[2] <= ' ')) || strings.HasPrefix(text, "#"):
			end := strings.IndexByte(text, '\n')
			if end < 0 {
				return ""
			}
			text = text[end+1:]
		default:
			return text
		}
	}
}

func startsWithKeyword(text string, keywords ...string) bool {
	word := text
	if i := strings.IndexFunc(text, func(r rune) bool { return !unicode.IsLetter(r) }); i >= 0 {
		word = text[:i]
	}
	for _, keyword := range keywords {
		if strings.EqualFold(word, keyword) {
			return true
		}
	}
	return false
}

// EstimateAffectedRowsFromExplainJSON returns the planner's estimate of the rows an INSERT,
// REPLACE, UPDATE, or DELETE statement modifies, capped by the statement's LIMIT. The plan is
// MySQL `EXPLAIN FORMAT=JSON` output in explain_json_format_version 1, or MariaDB's, for the
// statement's AffectedRowsQuery.
func EstimateAffectedRowsFromExplainJSON(stmt ast.Node, plan string) (int64, error) {
	var root map[string]any
	if err := json.Unmarshal([]byte(plan), &root); err != nil {
		return 0, errors.Wrap(err, "failed to parse EXPLAIN FORMAT=JSON output")
	}
	block, ok := root["query_block"].(map[string]any)
	if !ok {
		return 0, errors.New("unsupported EXPLAIN FORMAT=JSON output: no query_block")
	}

	var rows float64
	var err error
	switch s := stmt.(type) {
	case *ast.InsertStmt:
		// MySQL prints no source plan for a SELECT that reads no table, such as
		// SELECT ... FROM DUAL WHERE NOT EXISTS (...).
		if tableFreeRows, ok := tableFreeSelectRows(s.Select); ok {
			rows = float64(tableFreeRows)
		} else {
			rows, err = queryBlockRows(block)
		}
	case *ast.UpdateStmt:
		rows, err = targetRows(block, updateTargets(s))
	case *ast.DeleteStmt:
		rows, err = targetRows(block, deleteTargets(s))
	default:
		return 0, errors.Errorf("unsupported statement type %T", stmt)
	}
	if err != nil {
		return 0, err
	}
	return CapAffectedRowsByLimit(stmt, rows), nil
}

// CapAffectedRowsByLimit rounds an estimate of the rows an INSERT, REPLACE, UPDATE, or DELETE
// statement modifies and caps it by the statement's LIMIT.
func CapAffectedRowsByLimit(stmt ast.Node, rows float64) int64 {
	var limit *ast.Limit
	switch s := stmt.(type) {
	case *ast.InsertStmt:
		if s.Select != nil {
			limit = s.Select.Limit
		} else if s.TableSource != nil {
			limit = s.TableSource.Limit
		}
	case *ast.UpdateStmt:
		limit = s.Limit
	case *ast.DeleteStmt:
		limit = s.Limit
	default:
	}
	count := common.RoundRows(rows)
	if limitRows, ok := limitCount(limit); ok && limitRows < count {
		return limitRows
	}
	return count
}

// EstimateAffectedRowsFromOceanBaseExplainJSON returns the EST.ROWS estimate from OceanBase
// `EXPLAIN FORMAT=JSON` output: the root operator's, or the largest child operator's when the
// root has no operator.
func EstimateAffectedRowsFromOceanBaseExplainJSON(plan string) (int64, error) {
	var root map[string]any
	if err := json.Unmarshal([]byte(plan), &root); err != nil {
		return 0, errors.Wrap(err, "failed to parse EXPLAIN FORMAT=JSON output")
	}
	if operator, ok := root["OPERATOR"].(string); ok && operator != "" {
		rows, ok := planNumber(root["EST.ROWS"])
		if !ok {
			return 0, errors.Errorf("operator %q has no EST.ROWS", operator)
		}
		return common.RoundRows(rows), nil
	}
	var maxRows float64
	found := false
	for key, value := range root {
		child, ok := value.(map[string]any)
		if !ok || !strings.HasPrefix(key, "CHILD_") {
			continue
		}
		operator, ok := child["OPERATOR"].(string)
		if !ok || operator == "" {
			continue
		}
		rows, ok := planNumber(child["EST.ROWS"])
		if !ok {
			return 0, errors.Errorf("operator %q has no EST.ROWS", operator)
		}
		if !found || rows > maxRows {
			maxRows = rows
			found = true
		}
	}
	if !found {
		return 0, errors.New("the plan has no operator")
	}
	return common.RoundRows(maxRows), nil
}

// queryBlockRows returns the planner's estimate of the rows a query block produces. A UNION sums its
// branches, an INTERSECT takes the fewest, and an EXCEPT, or a parenthesized query with its own
// ORDER BY or LIMIT, takes its first.
func queryBlockRows(block map[string]any) (float64, error) {
	for _, operation := range []string{"union_result", "intersect_result", "except_result", "unary_result"} {
		result, ok := block[operation].(map[string]any)
		if !ok {
			continue
		}
		branches, ok := result["query_specifications"].([]any)
		if !ok || len(branches) == 0 {
			return 0, errors.Errorf("%s has no query_specifications", operation)
		}
		var total float64
		for i, branch := range branches {
			specification, ok := branch.(map[string]any)
			if !ok {
				return 0, errors.New("unexpected query_specifications element")
			}
			// MySQL nests a set operation as a branch without a query_block of its own.
			branchBlock, ok := specification["query_block"].(map[string]any)
			if !ok {
				branchBlock = specification
			}
			rows, err := queryBlockRows(branchBlock)
			if err != nil {
				return 0, err
			}
			switch {
			case i == 0 || operation == "union_result":
				total += rows
			case operation == "intersect_result":
				total = min(total, rows)
			default:
			}
		}
		return total, nil
	}
	_, cumulative, err := planJoin(block)
	if err != nil {
		return 0, err
	}
	return cumulative[len(cumulative)-1], nil
}

// dmlTargets holds the names a plan uses for the tables an UPDATE or DELETE modifies.
type dmlTargets struct {
	names []string
	// unqualified counts the multi-table UPDATE assignments to a column without a table qualifier.
	unqualified int
}

// DMLTargetCount returns how many tables stmt can change for each row its join produces: the targets
// of a multi-table UPDATE or DELETE, counting each unqualified assignment as another joined table, up
// to joinedTables, and one for any other statement.
func DMLTargetCount(stmt ast.Node, joinedTables int) int {
	var targets dmlTargets
	switch s := stmt.(type) {
	case *ast.UpdateStmt:
		targets = updateTargets(s)
	case *ast.DeleteStmt:
		targets = deleteTargets(s)
	default:
		return 1
	}
	return max(min(len(targets.names)+targets.unqualified, joinedTables), 1)
}

func updateTargets(stmt *ast.UpdateStmt) dmlTargets {
	if len(stmt.Tables) == 1 {
		if table, ok := stmt.Tables[0].(*ast.TableRef); ok {
			return dmlTargets{names: []string{tableRefPlanName(table)}}
		}
	}
	var targets dmlTargets
	for _, assignment := range stmt.SetList {
		if assignment.Column == nil || assignment.Column.Table == "" {
			targets.unqualified++
			continue
		}
		if !containsFold(targets.names, assignment.Column.Table) {
			targets.names = append(targets.names, assignment.Column.Table)
		}
	}
	return targets
}

func deleteTargets(stmt *ast.DeleteStmt) dmlTargets {
	var targets dmlTargets
	for _, expr := range stmt.Tables {
		table, ok := expr.(*ast.TableRef)
		if !ok {
			continue
		}
		// The multi-table forms list names declared in FROM or USING, so only the single-table
		// form can declare an alias here.
		name := table.Name
		if len(stmt.Using) == 0 {
			name = tableRefPlanName(table)
		}
		targets.names = append(targets.names, name)
	}
	return targets
}

func targetRows(block map[string]any, targets dmlTargets) (float64, error) {
	tables, cumulative, err := planJoin(block)
	if err != nil {
		return 0, err
	}
	final := cumulative[len(cumulative)-1]
	// A plan that produces no rows, such as "Impossible WHERE", prints no target.
	if final == 0 {
		return 0, nil
	}

	// Each target counts min(its cumulative estimate, the join's final estimate): the rows
	// reaching the modification, bounded by the rows read from the target table.
	var total float64
	flagged := false
	for i, table := range tables {
		if planFlag(table["update"]) || planFlag(table["delete"]) {
			total += min(cumulative[i], final)
			flagged = true
		}
	}
	if flagged {
		return total, nil
	}

	// MariaDB flags only single-table targets, and the SELECT a single-table UPDATE or DELETE is
	// explained as flags none, so find the targets by name. A name that no plan table has, such as a
	// view's, or that several have, as when a subquery reads another table of that name, counts the
	// final estimate.
	if len(targets.names) == 0 && targets.unqualified == 0 {
		return 0, errors.New("the plan has no UPDATE or DELETE target")
	}
	for _, name := range targets.names {
		index, matches := -1, 0
		for i, table := range tables {
			if tableName, ok := table["table_name"].(string); ok && strings.EqualFold(tableName, name) {
				index = i
				matches++
			}
		}
		if matches == 1 {
			total += min(cumulative[index], final)
		} else {
			total += final
		}
	}
	// An unqualified column may belong to any joined table, so each counts the final estimate, up to
	// one per joined table.
	total += final * float64(min(targets.unqualified, len(tables)))
	return total, nil
}

// planWrappers are plan members that wrap a query body without changing the tables it joins.
var planWrappers = []string{
	// MySQL.
	"ordering_operation", "grouping_operation", "duplicates_removal", "windowing", "buffer_result",
	// MariaDB.
	"block-nl-join", "read_sorted_file", "filesort", "temporary_table", "window_functions_computation", "buffer",
}

// planJoin returns the tables a query block joins, in join order, with the planner's estimate of
// the rows produced after joining each one. It does not descend into subqueries or derived tables.
func planJoin(block map[string]any) ([]map[string]any, []float64, error) {
	var tables []map[string]any
	var collect func(node any) error
	collect = func(node any) error {
		switch n := node.(type) {
		case []any:
			for _, element := range n {
				if err := collect(element); err != nil {
					return err
				}
			}
			return nil
		case map[string]any:
			// A MySQL INSERT plan puts the target table beside insert_from, which holds the source.
			for _, key := range []string{"insert_from", "nested_loop"} {
				if body, ok := n[key]; ok {
					return collect(body)
				}
			}
			if table, ok := n["table"].(map[string]any); ok {
				tables = append(tables, table)
				return nil
			}
			for _, key := range planWrappers {
				if body, ok := n[key]; ok {
					return collect(body)
				}
			}
			if _, ok := n["message"].(string); ok {
				tables = append(tables, n)
				return nil
			}
			return errors.New("unrecognized EXPLAIN FORMAT=JSON plan node")
		default:
			return errors.Errorf("unexpected EXPLAIN FORMAT=JSON plan value %v", node)
		}
	}
	if err := collect(block); err != nil {
		return nil, nil, err
	}
	if len(tables) == 0 {
		return nil, nil, errors.New("the plan has no tables")
	}

	cumulative := make([]float64, len(tables))
	previous, fanout := 1.0, 1.0
	for i, table := range tables {
		filtered, ok := planNumber(table["filtered"])
		if !ok {
			filtered = 100
		}
		produced, hasProduced := planNumber(table["rows_produced_per_join"])
		scanned, hasScanned := planNumber(table["rows_examined_per_scan"])
		if !hasScanned {
			scanned, hasScanned = planNumber(table["rows"])
		}
		materialized, isMaterialized := table["materialized_from_subquery"].(map[string]any)
		message, hasMessage := table["message"].(string)
		switch {
		case hasProduced:
			cumulative[i] = produced * fanout
		case hasScanned:
			// MariaDB prints loops; otherwise every row produced so far drives one scan.
			loops, ok := planNumber(table["loops"])
			if !ok {
				loops = previous
			}
			cumulative[i] = loops * scanned * filtered / 100
		case isMaterialized:
			// A MySQL semi-join materialization scan has no row estimate of its own, and the
			// per-join estimates of the tables after it count rows per materialized row.
			subquery, ok := materialized["query_block"].(map[string]any)
			if !ok {
				return nil, nil, errors.New("materialized_from_subquery has no query_block")
			}
			rows, err := queryBlockRows(subquery)
			if err != nil {
				return nil, nil, err
			}
			fanout *= rows
			cumulative[i] = previous * rows
		case hasMessage:
			rows, ok := planMessageRows(message)
			if !ok {
				return nil, nil, errors.Errorf("the plan has no row estimate: %s", message)
			}
			cumulative[i] = previous * rows
		default:
			if tableName, ok := table["table_name"].(string); ok {
				return nil, nil, errors.Errorf("table %q in the plan has no row estimate", tableName)
			}
			return nil, nil, errors.New("the plan has a table without a row estimate")
		}
		previous = cumulative[i]
	}
	return tables, cumulative, nil
}

// planMessageRows returns the rows produced where the plan prints an optimizer message in place
// of a table.
func planMessageRows(message string) (float64, bool) {
	switch message {
	case "Impossible WHERE", "Impossible WHERE noticed after reading const tables",
		"Impossible HAVING", "Impossible HAVING noticed after reading const tables",
		"no matching row in const table", "No matching rows after partition pruning", "Zero limit":
		return 0, true
	case "No tables used", "Select tables optimized away":
		return 1, true
	default:
		return 0, false
	}
}

// planFlag reads an update/delete marker, which MySQL prints as true and MariaDB as 1.
func planFlag(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case float64:
		return v != 0
	default:
		return false
	}
}

// planNumber reads a plan number, which MySQL sometimes prints as a string such as "10.00".
func planNumber(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// tableFreeSelectRows returns the most rows a SELECT produces when none of its branches reads a
// table: one for each branch, combined as queryBlockRows combines set operations.
func tableFreeSelectRows(sel *ast.SelectStmt) (int, bool) {
	if sel == nil {
		return 0, false
	}
	if sel.ParenSource != nil {
		return tableFreeSelectRows(sel.ParenSource)
	}
	if sel.SetOp != ast.SetOpNone {
		left, leftOK := tableFreeSelectRows(sel.Left)
		right, rightOK := tableFreeSelectRows(sel.Right)
		if !leftOK || !rightOK {
			return 0, false
		}
		switch sel.SetOp {
		case ast.SetOpIntersect:
			return min(left, right), true
		case ast.SetOpExcept:
			return left, true
		default:
			return left + right, true
		}
	}
	if sel.TableSource != nil || sel.ValuesSource != nil {
		return 0, false
	}
	for _, from := range sel.From {
		table, ok := from.(*ast.TableRef)
		if !ok || table.Schema != "" || !strings.EqualFold(table.Name, "dual") {
			return 0, false
		}
	}
	return 1, true
}

func tableRefPlanName(table *ast.TableRef) string {
	if table.Alias != "" {
		return table.Alias
	}
	return table.Name
}

func containsFold(names []string, name string) bool {
	for _, n := range names {
		if strings.EqualFold(n, name) {
			return true
		}
	}
	return false
}

func limitCount(limit *ast.Limit) (int64, bool) {
	if limit == nil || limit.Count == nil {
		return 0, false
	}
	lit, ok := limit.Count.(*ast.IntLit)
	if !ok || lit.Value < 0 {
		return 0, false
	}
	return lit.Value, true
}
