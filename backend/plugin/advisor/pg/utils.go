package pg

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/bytebase/omni/pg/ast"
	"github.com/pkg/errors"
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

// getAffectedRows extracts the estimated row count from a PostgreSQL EXPLAIN result.
func getAffectedRows(res []any) (int64, error) {
	// the res struct is []any{columnName, columnTable, rowDataList}
	if len(res) != 3 {
		return 0, errors.Errorf("expected 3 but got %d", len(res))
	}
	rowList, ok := res[2].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any but got %t", res[2])
	}
	// EXPLAIN output has at least 2 rows
	if len(rowList) < 2 {
		return 0, errors.Errorf("not found any data")
	}
	// We need row 2
	rowTwo, ok := rowList[1].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any but got %t", rowList[0])
	}
	// PostgreSQL EXPLAIN result has one column
	if len(rowTwo) != 1 {
		return 0, errors.Errorf("expected one but got %d", len(rowTwo))
	}
	// Get the string value
	text, ok := rowTwo[0].(string)
	if !ok {
		return 0, errors.Errorf("expected string but got %t", rowTwo[0])
	}

	rowsRegexp := regexp.MustCompile("rows=([0-9]+)")
	matches := rowsRegexp.FindStringSubmatch(text)
	if len(matches) != 2 {
		return 0, errors.Errorf("failed to find rows in %q", text)
	}
	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, errors.Errorf("failed to get integer from %q", matches[1])
	}
	return value, nil
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

// nolint:unused
// omniIsRoleOrSearchPathSet checks if a VariableSetStmt is SET ROLE or SET search_path.
func omniIsRoleOrSearchPathSet(stmt *ast.VariableSetStmt) bool {
	if stmt == nil {
		return false
	}
	name := strings.ToLower(stmt.Name)
	return name == "role" || name == "search_path"
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
