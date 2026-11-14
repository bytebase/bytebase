package pg

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/store/model"
)

// originalSchema returns the pre-change schema the omni v1review rules look
// names up in. A nil m gives an empty schema, in which every lookup misses.
func originalSchema(m *model.DatabaseMetadata) v1review.Schema {
	if m == nil {
		return v1review.Schema{}
	}
	return v1review.Schema{Metadata: m.GetProto(), ObjectCaseSensitive: m.GetIsObjectCaseSensitive()}
}

// getExplainSQL returns the query whose result getExplainPlan reads.
func getExplainSQL(statement string) string {
	return fmt.Sprintf("EXPLAIN (FORMAT JSON) %s", statement)
}

// getExplainPlan returns the JSON plan from the advisor.Query result of a getExplainSQL query.
func getExplainPlan(res []any) (string, error) {
	// the res struct is []any{columnName, columnTable, rowDataList}
	if len(res) != 3 {
		return "", errors.Errorf("expected 3 but got %d", len(res))
	}
	rowList, ok := res[2].([]any)
	if !ok {
		return "", errors.Errorf("expected []any but got %T", res[2])
	}
	if len(rowList) != 1 {
		return "", errors.Errorf("expected one plan row but got %d", len(rowList))
	}
	row, ok := rowList[0].([]any)
	if !ok || len(row) != 1 {
		return "", errors.Errorf("expected one plan column but got %v", rowList[0])
	}
	plan, ok := row[0].(string)
	if !ok {
		return "", errors.Errorf("expected string but got %T", row[0])
	}
	return plan, nil
}

// getAffectedRows returns the estimated rows a statement and its data-modifying CTEs modify, from
// the advisor.Query result of its getExplainSQL query.
func getAffectedRows(res []any) (int64, error) {
	plan, err := getExplainPlan(res)
	if err != nil {
		return 0, err
	}
	return pgparser.GetEstimatedAffectedRowsFromExplainJSON(plan)
}

// getInsertedRows returns the estimated rows an INSERT adds, without the rows its data-modifying
// CTEs change, from the advisor.Query result of its getExplainSQL query.
func getInsertedRows(res []any) (int64, error) {
	plan, err := getExplainPlan(res)
	if err != nil {
		return 0, err
	}
	return pgparser.GetEstimatedInsertedRowsFromExplainJSON(plan)
}

// rangeVarTableName extracts the table name from a RangeVar.
func rangeVarTableName(rv *ast.RangeVar) string {
	if rv == nil {
		return ""
	}
	return rv.Relname
}

// columnConstraints iterates over a ColumnDef's constraint list.
func columnConstraints(col *ast.ColumnDef) []*ast.Constraint {
	if col == nil || col.Constraints == nil {
		return nil
	}
	var result []*ast.Constraint
	for _, item := range col.Constraints.Items {
		if c, ok := item.(*ast.Constraint); ok {
			result = append(result, c)
		}
	}
	return result
}

// tableElements iterates over a CreateStmt's table elements,
// returning column defs and table constraints separately.
func tableElements(create *ast.CreateStmt) ([]*ast.ColumnDef, []*ast.Constraint) {
	if create.TableElts == nil {
		return nil, nil
	}
	var cols []*ast.ColumnDef
	var cons []*ast.Constraint
	for _, item := range create.TableElts.Items {
		switch n := item.(type) {
		case *ast.ColumnDef:
			cols = append(cols, n)
		case *ast.Constraint:
			cons = append(cons, n)
		default:
		}
	}
	return cols, cons
}

// sessionSettings holds, in order, the statements that set the role or search path, which an EXPLAIN
// replays. A SET LOCAL setting lasts until its transaction ends, ROLLBACK also drops the settings of
// its transaction, and DISCARD ALL drops every setting.
type sessionSettings struct {
	settings []sessionSetting
	// transactionStart is the number of settings when the current transaction began.
	transactionStart int
	inTransaction    bool
}

type sessionSetting struct {
	name  string
	text  string
	local bool
}

// add records node, whose text is text, when it changes the settings.
func (s *sessionSettings) add(node ast.Node, text string) {
	switch n := node.(type) {
	case *ast.VariableSetStmt:
		if isRoleOrSearchPathSet(n) {
			if n.Kind == ast.VAR_SET_CURRENT && !n.IsLocal {
				// SET ... FROM CURRENT keeps the value a SET LOCAL gave the setting after the transaction ends.
				for i := range s.settings {
					if strings.EqualFold(s.settings[i].name, n.Name) {
						s.settings[i].local = false
					}
				}
			}
			s.settings = append(s.settings, sessionSetting{name: n.Name, text: text, local: n.IsLocal})
		}
	case *ast.DiscardStmt:
		if n.Target == ast.DISCARD_ALL {
			s.settings, s.transactionStart = nil, 0
		}
	case *ast.TransactionStmt:
		switch n.Kind {
		case ast.TRANS_STMT_BEGIN, ast.TRANS_STMT_START:
			// A BEGIN inside a transaction only warns.
			if !s.inTransaction {
				s.transactionStart, s.inTransaction = len(s.settings), true
			}
		case ast.TRANS_STMT_COMMIT, ast.TRANS_STMT_ROLLBACK:
			// A ROLLBACK outside a transaction only warns.
			if n.Kind == ast.TRANS_STMT_ROLLBACK && s.inTransaction {
				s.settings = s.settings[:s.transactionStart]
			}
			s.settings = slices.DeleteFunc(s.settings, func(setting sessionSetting) bool { return setting.local })
			// COMMIT AND CHAIN and ROLLBACK AND CHAIN start the next transaction from here.
			s.transactionStart, s.inTransaction = len(s.settings), n.Chain
		default:
		}
	default:
	}
}

func (s *sessionSettings) statements() []string {
	var statements []string
	for _, setting := range s.settings {
		statements = append(statements, setting.text)
	}
	return statements
}

// isRoleOrSearchPathSet checks if a VariableSetStmt sets the role or search path, including a
// RESET ALL.
func isRoleOrSearchPathSet(stmt *ast.VariableSetStmt) bool {
	if stmt == nil {
		return false
	}
	name := strings.ToLower(stmt.Name)
	return name == "role" || name == "search_path" || stmt.Kind == ast.VAR_RESET_ALL
}

// listStrings extracts string values from an ast.List.
func listStrings(list *ast.List) []string {
	if list == nil {
		return nil
	}
	var result []string
	for _, item := range list.Items {
		if s, ok := item.(*ast.String); ok {
			result = append(result, s.Str)
		}
	}
	return result
}
