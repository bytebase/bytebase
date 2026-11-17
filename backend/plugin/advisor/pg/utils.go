package pg

import (
	"fmt"
	"slices"
	"strings"

	pganalysis "github.com/bytebase/omni/pg/analysis"
	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/plugin/db"
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

// getExplainSQL returns the query whose JSON plan getAffectedRows and getInsertedRows read.
func getExplainSQL(statement string) string {
	return fmt.Sprintf("EXPLAIN (FORMAT JSON) %s", statement)
}

// getAffectedRows returns the estimated rows a statement and its data-modifying CTEs modify, from
// the probe result of its getExplainSQL query.
func getAffectedRows(res db.ProbeResult) (int64, error) {
	plan, err := res.Plan()
	if err != nil {
		return 0, err
	}
	return pganalysis.EstimatedAffectedRowsFromExplainJSON(plan)
}

// getInsertedRows returns the estimated rows an INSERT adds, without the rows its data-modifying
// CTEs change, from the probe result of its getExplainSQL query.
func getInsertedRows(res db.ProbeResult) (int64, error) {
	plan, err := res.Plan()
	if err != nil {
		return 0, err
	}
	return pganalysis.EstimatedInsertedRowsFromExplainJSON(plan)
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
