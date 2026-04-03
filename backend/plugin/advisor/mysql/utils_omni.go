package mysql

import (
	"strings"

	"github.com/bytebase/omni/mysql/ast"
)

// nolint:unused
// omniTableName extracts the table name from a TableRef.
func omniTableName(ref *ast.TableRef) string {
	if ref == nil {
		return ""
	}
	return ref.Name
}

// nolint:unused
// omniColumnNames extracts column name list from a table-level constraint.
func omniColumnNames(constraint *ast.Constraint) []string {
	if constraint == nil {
		return nil
	}
	return constraint.Columns
}

// nolint:unused
// omniIndexColumns extracts column names from an index column list.
// For expression-based index columns, it falls back to the column reference name.
func omniIndexColumns(cols []*ast.IndexColumn) []string {
	if len(cols) == 0 {
		return nil
	}
	var names []string
	for _, col := range cols {
		if ref, ok := col.Expr.(*ast.ColumnRef); ok {
			names = append(names, ref.Column)
		}
	}
	return names
}

// nolint:unused
// omniDataTypeName extracts the normalized (upper-case) type name string from a DataType.
func omniDataTypeName(dt *ast.DataType) string {
	if dt == nil {
		return ""
	}
	return strings.ToUpper(dt.Name)
}

// nolint:unused
// omniIsNullable checks if a column allows NULL.
// A column is nullable unless it has a NOT NULL constraint or is a PRIMARY KEY.
func omniIsNullable(col *ast.ColumnDef) bool {
	if col == nil {
		return true
	}
	for _, c := range col.Constraints {
		if c.Type == ast.ColConstrNotNull || c.Type == ast.ColConstrPrimaryKey {
			return false
		}
	}
	return true
}

// nolint:unused
// omniHasDefault checks if a column has a DEFAULT value.
func omniHasDefault(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	if col.DefaultValue != nil {
		return true
	}
	for _, c := range col.Constraints {
		if c.Type == ast.ColConstrDefault {
			return true
		}
	}
	return false
}

// nolint:unused
// omniIsAutoIncrement checks if a column has AUTO_INCREMENT.
func omniIsAutoIncrement(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	return col.AutoIncrement
}

// nolint:unused
// omniColumnComment extracts the COMMENT string from a column definition.
func omniColumnComment(col *ast.ColumnDef) string {
	if col == nil {
		return ""
	}
	return col.Comment
}

// nolint:unused
// omniTableOptionValue extracts a table option value by name (case-insensitive).
func omniTableOptionValue(opts []*ast.TableOption, name string) string {
	for _, opt := range opts {
		if strings.EqualFold(opt.Name, name) {
			return opt.Value
		}
	}
	return ""
}

// nolint:unused
// omniConstraintsByType filters constraints by the given type.
func omniConstraintsByType(constraints []*ast.Constraint, typ ast.ConstraintType) []*ast.Constraint {
	var result []*ast.Constraint
	for _, c := range constraints {
		if c.Type == typ {
			result = append(result, c)
		}
	}
	return result
}

// omniDataTypeNameCompact returns a compact, lowercase type name string.
func omniDataTypeNameCompact(dt *ast.DataType) string {
	if dt == nil {
		return ""
	}
	return strings.ToLower(dt.Name)
}

// omniIsIntegerType checks if the data type is an integer type.
func omniIsIntegerType(dt *ast.DataType) bool {
	if dt == nil {
		return false
	}
	switch strings.ToUpper(dt.Name) {
	case "INT", "TINYINT", "SMALLINT", "MEDIUMINT", "BIGINT", "INTEGER":
		return true
	default:
		return false
	}
}

// omniIsTimeType checks if the data type is DATETIME or TIMESTAMP.
func omniIsTimeType(dt *ast.DataType) bool {
	if dt == nil {
		return false
	}
	switch strings.ToUpper(dt.Name) {
	case "DATETIME", "TIMESTAMP":
		return true
	default:
		return false
	}
}

// omniIsDefaultCurrentTime checks if a column has DEFAULT CURRENT_TIMESTAMP / NOW().
func omniIsDefaultCurrentTime(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	return isCurrentTimeFuncCall(col.DefaultValue)
}

// omniIsOnUpdateCurrentTime checks if a column has ON UPDATE CURRENT_TIMESTAMP / NOW().
func omniIsOnUpdateCurrentTime(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	return isCurrentTimeFuncCall(col.OnUpdate)
}

// isCurrentTimeFuncCall checks if an expression is a CURRENT_TIMESTAMP/NOW() call.
func isCurrentTimeFuncCall(expr ast.ExprNode) bool {
	if expr == nil {
		return false
	}
	fc, ok := expr.(*ast.FuncCallExpr)
	if !ok {
		return false
	}
	switch strings.ToUpper(fc.Name) {
	case "NOW", "CURRENT_TIMESTAMP", "LOCALTIME", "LOCALTIMESTAMP":
		return true
	default:
		return false
	}
}

// omniGetColumnsFromCmd extracts column definitions from an AlterTableCmd.
func omniGetColumnsFromCmd(cmd *ast.AlterTableCmd) []*ast.ColumnDef {
	if cmd == nil {
		return nil
	}
	switch cmd.Type {
	case ast.ATAddColumn:
		if cmd.Column != nil {
			return []*ast.ColumnDef{cmd.Column}
		}
		return cmd.Columns
	case ast.ATModifyColumn, ast.ATChangeColumn:
		if cmd.Column != nil {
			return []*ast.ColumnDef{cmd.Column}
		}
	default:
	}
	return nil
}

// nolint:unused
// omniColumnName returns the effective column name for a command.
func omniColumnName(cmd *ast.AlterTableCmd) string {
	if cmd.Column != nil {
		return cmd.Column.Name
	}
	return cmd.Name
}

// omniIsPrimaryKey checks if a column has a PRIMARY KEY constraint.
func omniIsPrimaryKey(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	for _, c := range col.Constraints {
		if c.Type == ast.ColConstrPrimaryKey {
			return true
		}
	}
	return false
}

// omniPKColumnNames returns a set of column names that are part of a primary key.
func omniPKColumnNames(columns []*ast.ColumnDef, constraints []*ast.Constraint) map[string]bool {
	result := make(map[string]bool)
	for _, col := range columns {
		if omniIsPrimaryKey(col) {
			result[col.Name] = true
		}
	}
	for _, c := range constraints {
		if c.Type == ast.ConstrPrimaryKey {
			for _, name := range c.Columns {
				result[name] = true
			}
		}
	}
	return result
}

// omniColumnNeedDefault checks if a column is NOT NULL and needs a DEFAULT.
// nolint:unused
func omniColumnNeedDefault(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	return !omniIsNullable(col) && !omniHasDefault(col) && !omniIsPrimaryKey(col)
}

// omniIsDefaultExemptType checks if a column type is exempt from requiring DEFAULT
// (BLOB, TEXT, JSON, GEOMETRY and spatial types).
func omniIsDefaultExemptType(col *ast.ColumnDef) bool {
	if col == nil || col.TypeName == nil {
		return false
	}
	switch strings.ToUpper(col.TypeName.Name) {
	case "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB",
		"TEXT", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT",
		"JSON", "GEOMETRY", "POINT", "LINESTRING", "POLYGON",
		"MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON", "GEOMETRYCOLLECTION",
		"GEOMCOLLECTION":
		return true
	default:
		return false
	}
}

// omniColumnNeedDefaultNotNull checks if a NOT NULL column needs a DEFAULT value
// (excludes BLOB, TEXT, JSON, GEOMETRY, and AUTO_INCREMENT columns).
func omniColumnNeedDefaultNotNull(col *ast.ColumnDef) bool {
	if col == nil {
		return false
	}
	if omniIsNullable(col) || omniHasDefault(col) || omniIsPrimaryKey(col) || col.AutoIncrement {
		return false
	}
	if col.TypeName != nil {
		switch strings.ToUpper(col.TypeName.Name) {
		case "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB",
			"TEXT", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT",
			"JSON", "GEOMETRY", "SERIAL":
			return false
		}
	}
	return true
}
