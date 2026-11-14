package mysql

import (
	"github.com/bytebase/omni/mysql/analysis"
	"github.com/bytebase/omni/mysql/ast"
)

// AffectedRowsQuery returns the statement to EXPLAIN for the rows stmt, parsed
// from statement, modifies.
func AffectedRowsQuery(stmt ast.Node, statement string) string {
	return analysis.AffectedRowsQuery(stmt, statement)
}

// EstimateAffectedRowsFromExplainJSON returns the planner's estimate of the rows
// stmt modifies, read from the EXPLAIN FORMAT=JSON plan of its AffectedRowsQuery.
func EstimateAffectedRowsFromExplainJSON(stmt ast.Node, plan string) (int64, error) {
	return analysis.EstimateAffectedRowsFromExplainJSON(stmt, plan)
}

// CapAffectedRowsByLimit rounds an estimate of the rows stmt modifies and caps it
// by the statement's LIMIT.
func CapAffectedRowsByLimit(stmt ast.Node, rows float64) int64 {
	return analysis.CapAffectedRowsByLimit(stmt, rows)
}

// DMLTargetCount returns how many tables stmt can change for each row its join
// produces, up to joinedTables.
func DMLTargetCount(stmt ast.Node, joinedTables int) int {
	return analysis.DMLTargetCount(stmt, joinedTables)
}
