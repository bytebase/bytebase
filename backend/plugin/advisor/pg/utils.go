package pg

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/bytebase/omni/pg/ast"
	"github.com/pkg/errors"

	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

func getTemplateRegexp(template string, templateList []string, tokens map[string]string) (*regexp.Regexp, error) {
	for _, key := range templateList {
		if token, ok := tokens[key]; ok {
			template = strings.ReplaceAll(template, key, token)
		}
	}

	return regexp.Compile(template)
}

// normalizeSchemaName normalizes empty schema names to "public" (PostgreSQL default schema).
func normalizeSchemaName(schemaName string) string {
	if schemaName == "" {
		return "public"
	}
	return schemaName
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

// hasDataModifyingCTE reports whether a WITH clause contains INSERT, UPDATE, DELETE, or MERGE.
// PostgreSQL only allows data-modifying statements in a top-level WITH.
func hasDataModifyingCTE(with *ast.WithClause) bool {
	if with == nil || with.Ctes == nil {
		return false
	}
	for _, item := range with.Ctes.Items {
		cte, ok := item.(*ast.CommonTableExpr)
		if !ok {
			continue
		}
		switch cte.Ctequery.(type) {
		case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt, *ast.MergeStmt:
			return true
		default:
		}
	}
	return false
}

// nolint:unused
// omniTableName extracts the table name from a RangeVar.
func omniTableName(rv *ast.RangeVar) string {
	if rv == nil {
		return ""
	}
	return rv.Relname
}

// nolint:unused
// omniSchemaName extracts the schema name from a RangeVar, defaulting to "public".
func omniSchemaName(rv *ast.RangeVar) string {
	if rv == nil || rv.Schemaname == "" {
		return "public"
	}
	return rv.Schemaname
}

// nolint:unused
// omniConstraintColumns extracts column names from a Constraint's Keys list.
func omniConstraintColumns(c *ast.Constraint) []string {
	if c == nil || c.Keys == nil {
		return nil
	}
	var cols []string
	for _, item := range c.Keys.Items {
		if s, ok := item.(*ast.String); ok {
			cols = append(cols, s.Str)
		}
	}
	return cols
}

// nolint:unused
// omniColumnConstraints iterates over a ColumnDef's constraint list.
func omniColumnConstraints(col *ast.ColumnDef) []*ast.Constraint {
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

// nolint:unused
// omniTableElements iterates over a CreateStmt's table elements,
// returning column defs and table constraints separately.
func omniTableElements(create *ast.CreateStmt) ([]*ast.ColumnDef, []*ast.Constraint) {
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

// nolint:unused
// omniAlterTableCmds extracts AlterTableCmd items from an AlterTableStmt.
func omniAlterTableCmds(alter *ast.AlterTableStmt) []*ast.AlterTableCmd {
	if alter.Cmds == nil {
		return nil
	}
	var cmds []*ast.AlterTableCmd
	for _, item := range alter.Cmds.Items {
		if cmd, ok := item.(*ast.AlterTableCmd); ok {
			cmds = append(cmds, cmd)
		}
	}
	return cmds
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
		if omniIsRoleOrSearchPathSet(n) {
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

// omniIsRoleOrSearchPathSet checks if a VariableSetStmt sets the role or search path, including a
// RESET ALL.
func omniIsRoleOrSearchPathSet(stmt *ast.VariableSetStmt) bool {
	if stmt == nil {
		return false
	}
	name := strings.ToLower(stmt.Name)
	return name == "role" || name == "search_path" || stmt.Kind == ast.VAR_RESET_ALL
}

// nolint:unused
// omniTypeName extracts the type name string from a TypeName node.
func omniTypeName(tn *ast.TypeName) string {
	if tn == nil || tn.Names == nil {
		return ""
	}
	var parts []string
	for _, item := range tn.Names.Items {
		if s, ok := item.(*ast.String); ok {
			parts = append(parts, s.Str)
		}
	}
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// omniTypeNameFull extracts the full type name with modifiers from a TypeName node.
// For example, "character varying(20)" instead of just "character varying".
func omniTypeNameFull(tn *ast.TypeName) string {
	name := omniTypeName(tn)
	if name == "" {
		return ""
	}
	if tn.Typmods != nil && len(tn.Typmods.Items) > 0 {
		var mods []string
		for _, item := range tn.Typmods.Items {
			switch v := item.(type) {
			case *ast.Integer:
				mods = append(mods, fmt.Sprintf("%d", v.Ival))
			case *ast.A_Const:
				if iv, ok := v.Val.(*ast.Integer); ok {
					mods = append(mods, fmt.Sprintf("%d", iv.Ival))
				}
			default:
			}
		}
		if len(mods) > 0 {
			name += "(" + strings.Join(mods, ",") + ")"
		}
	}
	if tn.ArrayBounds != nil && len(tn.ArrayBounds.Items) > 0 {
		name += "[]"
	}
	return name
}

// omniListStrings extracts string values from an ast.List.
func omniListStrings(list *ast.List) []string {
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

// omniIndexColumns extracts column names from an IndexStmt's IndexParams list.
func omniIndexColumns(idx *ast.IndexStmt) []string {
	if idx == nil || idx.IndexParams == nil {
		return nil
	}
	var cols []string
	for _, item := range idx.IndexParams.Items {
		if elem, ok := item.(*ast.IndexElem); ok && elem.Name != "" {
			cols = append(cols, elem.Name)
		}
	}
	return cols
}

// omniDropObjectNames extracts object names from a DropStmt.
// Returns list of qualified name string slices.
func omniDropObjectNames(drop *ast.DropStmt) [][]string {
	if drop.Objects == nil {
		return nil
	}
	var result [][]string
	for _, item := range drop.Objects.Items {
		list, ok := item.(*ast.List)
		if !ok {
			continue
		}
		var parts []string
		for _, nameItem := range list.Items {
			if s, ok := nameItem.(*ast.String); ok {
				parts = append(parts, s.Str)
			}
		}
		if len(parts) > 0 {
			result = append(result, parts)
		}
	}
	return result
}

// omniCollectFromClauseRangeVars recursively collects RangeVar nodes from a FROM clause list.
func omniCollectFromClauseRangeVars(fromClause *ast.List) []*ast.RangeVar {
	if fromClause == nil {
		return nil
	}
	var result []*ast.RangeVar
	for _, item := range fromClause.Items {
		result = append(result, omniCollectRangeVars(item)...)
	}
	return result
}

// omniCollectRangeVars recursively collects RangeVar nodes from a node,
// including those inside subqueries.
func omniCollectRangeVars(node ast.Node) []*ast.RangeVar {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.RangeVar:
		return []*ast.RangeVar{n}
	case *ast.JoinExpr:
		var result []*ast.RangeVar
		result = append(result, omniCollectRangeVars(n.Larg)...)
		result = append(result, omniCollectRangeVars(n.Rarg)...)
		return result
	case *ast.RangeSubselect:
		// Recurse into subquery
		if sel, ok := n.Subquery.(*ast.SelectStmt); ok {
			return omniCollectFromClauseRangeVars(sel.FromClause)
		}
		return nil
	default:
		return nil
	}
}

// nolint:unused
// omniDropObjects extracts object names from a DropStmt.
// Returns list of (schema, name) pairs.
func omniDropObjects(drop *ast.DropStmt) [][2]string {
	if drop.Objects == nil {
		return nil
	}
	var result [][2]string
	for _, item := range drop.Objects.Items {
		list, ok := item.(*ast.List)
		if !ok {
			continue
		}
		var parts []string
		for _, nameItem := range list.Items {
			if s, ok := nameItem.(*ast.String); ok {
				parts = append(parts, s.Str)
			}
		}
		switch len(parts) {
		case 1:
			result = append(result, [2]string{"public", parts[0]})
		case 2:
			result = append(result, [2]string{parts[0], parts[1]})
		default:
		}
	}
	return result
}
