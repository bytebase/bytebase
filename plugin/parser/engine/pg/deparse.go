package pg

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
)

func deparse(context parser.DeparseContext, in ast.Node, buf *strings.Builder) error {
	switch node := in.(type) {
	case ast.DataType:
		return deparseDataType(context, node, buf)
	case *ast.CreateTableStmt:
		if err := deparseCreateTable(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.DropTableStmt:
		if err := deparseDropTable(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.AlterTableStmt:
		if err := deparseAlterTable(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.TableDef:
		return deparseTableDef(context, node, buf)
	case *ast.ColumnDef:
		return deparseColumnDef(context, node, buf)
	case *ast.CreateSchemaStmt:
		if err := deparseCreateSchema(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.DropSchemaStmt:
		if err := deparseDropSchema(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.CreateIndexStmt:
		if err := deparseCreateIndex(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	}
	return errors.Errorf("failed to deparse %T", in)
}

func deparseCreateIndex(context parser.DeparseContext, in *ast.CreateIndexStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, parser.DeparseIndentString); err != nil {
		return err
	}

	if _, err := buf.WriteString("CREATE "); err != nil {
		return err
	}

	if in.Index.Unique {
		if _, err := buf.WriteString("UNIQUE "); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString("INDEX "); err != nil {
		return err
	}

	if in.IfNotExists {
		if _, err := buf.WriteString("IF NOT EXISTS "); err != nil {
			return err
		}
	}

	if err := writeSurrounding(buf, in.Index.Name, `"`); err != nil {
		return err
	}

	if _, err := buf.WriteString(" ON "); err != nil {
		return err
	}

	if err := deparseTableDef(context, in.Index.Table, buf); err != nil {
		return err
	}

	if _, err := buf.WriteString(" USING "); err != nil {
		return err
	}

	if err := deparseIndexMethod(in.Index.Method, buf); err != nil {
		return err
	}

	if _, err := buf.WriteString(" ("); err != nil {
		return err
	}

	for i, key := range in.Index.KeyList {
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if err := deparseIndexKey(parser.DeparseContext{IndentLevel: 0}, key, buf); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString(")"); err != nil {
		return err
	}

	return nil
}

func deparseIndexMethod(method ast.IndexMethodType, buf *strings.Builder) (err error) {
	switch method {
	case ast.IndexMethodTypeBTree:
		_, err = buf.WriteString("btree")
	case ast.IndexMethodTypeHash:
		_, err = buf.WriteString("hash")
	case ast.IndexMethodTypeGiST:
		_, err = buf.WriteString("gist")
	case ast.IndexMethodTypeSpGiST:
		_, err = buf.WriteString("spgist")
	case ast.IndexMethodTypeGin:
		_, err = buf.WriteString("gin")
	case ast.IndexMethodTypeBrin:
		_, err = buf.WriteString("brin")
	}
	return err
}

func deparseIndexKey(context parser.DeparseContext, in *ast.IndexKeyDef, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, parser.DeparseIndentString); err != nil {
		return err
	}

	switch in.Type {
	case ast.IndexKeyTypeColumn:
		if _, err := buf.WriteString(in.Key); err != nil {
			return err
		}
	case ast.IndexKeyTypeExpression:
		if err := buf.WriteByte('('); err != nil {
			return err
		}
		if _, err := buf.WriteString(in.Key); err != nil {
			return err
		}
		if err := buf.WriteByte(')'); err != nil {
			return err
		}
	}

	if in.SortOrder == ast.SortOrderTypeDescending {
		if _, err := buf.WriteString(" DESC"); err != nil {
			return err
		}
	}

	switch in.NullOrder {
	case ast.NullOrderTypeFirst:
		if _, err := buf.WriteString(" NULLS FIRST"); err != nil {
			return err
		}
	case ast.NullOrderTypeLast:
		if _, err := buf.WriteString(" NULLS LAST"); err != nil {
			return err
		}
	}

	return nil
}

func deparseDropTable(context parser.DeparseContext, in *ast.DropTableStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, parser.DeparseIndentString); err != nil {
		return err
	}

	if _, err := buf.WriteString("DROP TABLE "); err != nil {
		return err
	}
	if in.IfExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}
	for i, table := range in.TableList {
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if err := deparseTableDef(parser.DeparseContext{IndentLevel: 0}, table, buf); err != nil {
			return err
		}
	}
	return deparseDropBehavior(context, in.Behavior, buf)
}

func deparseAlterTable(context parser.DeparseContext, in *ast.AlterTableStmt, buf *strings.Builder) error {
	if _, err := buf.WriteString("ALTER TABLE "); err != nil {
		return err
	}
	if err := deparseTableDef(context, in.Table, buf); err != nil {
		return err
	}
	itemContext := parser.DeparseContext{
		IndentLevel: context.IndentLevel + 1,
	}
	for i, item := range in.AlterItemList {
		if i != 0 {
			if _, err := buf.WriteString(","); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
		switch action := item.(type) {
		case *ast.AddColumnListStmt:
			if err := deparseAddColumnList(itemContext, action, buf); err != nil {
				return err
			}
		case *ast.AlterColumnTypeStmt:
			if err := deparseAlterColumnType(itemContext, action, buf); err != nil {
				return err
			}
		case *ast.DropColumnStmt:
			if err := deparseDropColumn(itemContext, action, buf); err != nil {
				return err
			}
		case *ast.SetNotNullStmt:
			if err := deparseSetNotNull(itemContext, action, buf); err != nil {
				return err
			}
		case *ast.DropNotNullStmt:
			if err := deparseDropNotNull(itemContext, action, buf); err != nil {
				return err
			}
		case *ast.AddConstraintStmt:
			if err := deparseAddConstraint(itemContext, action, buf); err != nil {
				return err
			}
		case *ast.DropConstraintStmt:
			if err := deparseDropConstraint(itemContext, action, buf); err != nil {
				return err
			}
		case *ast.SetDefaultStmt:
			if err := deparseSetDefault(itemContext, action, buf); err != nil {
				return err
			}
		case *ast.DropDefaultStmt:
			if err := deparseDropDefault(itemContext, action, buf); err != nil {
				return err
			}
		}
	}
	return nil
}

func deparseSetDefault(context parser.DeparseContext, in *ast.SetDefaultStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, parser.DeparseIndentString); err != nil {
		return err
	}
	if _, err := buf.WriteString("ALTER COLUMN "); err != nil {
		return err
	}
	if err := writeSurrounding(buf, in.ColumnName, `"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(" SET DEFAULT "); err != nil {
		return err
	}
	if _, err := buf.WriteString(in.Expression.Text()); err != nil {
		return err
	}
	return nil
}

func deparseDropDefault(context parser.DeparseContext, in *ast.DropDefaultStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, parser.DeparseIndentString); err != nil {
		return err
	}
	if _, err := buf.WriteString("ALTER COLUMN "); err != nil {
		return err
	}
	if err := writeSurrounding(buf, in.ColumnName, `"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(" DROP DEFAULT"); err != nil {
		return err
	}
	return nil
}

func deparseAddConstraint(context parser.DeparseContext, in *ast.AddConstraintStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, parser.DeparseIndentString); err != nil {
		return err
	}
	if _, err := buf.WriteString("ADD "); err != nil {
		return err
	}
	if in.Constraint.Name != "" {
		if _, err := buf.WriteString("CONSTRAINT "); err != nil {
			return err
		}
		if err := writeSurrounding(buf, in.Constraint.Name, `"`); err != nil {
			return err
		}
		if _, err := buf.WriteString(" "); err != nil {
			return err
		}
	}
	switch in.Constraint.Type {
	case ast.ConstraintTypeUniqueUsingIndex:
		if _, err := buf.WriteString("UNIQUE USING INDEX "); err != nil {
			return err
		}
		if err := writeSurrounding(buf, in.Constraint.IndexName, `"`); err != nil {
			return err
		}

		if in.Constraint.Initdeferred {
			if _, err := buf.WriteString(" INITIALLY DEFERRED"); err != nil {
				return err
			}
		} else if in.Constraint.Deferrable {
			if _, err := buf.WriteString(" DEFERRABLE"); err != nil {
				return err
			}
		}
	case ast.ConstraintTypeUnique:
		if _, err := buf.WriteString("UNIQUE ("); err != nil {
			return err
		}
		for idx, col := range in.Constraint.KeyList {
			if idx >= 1 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if err := writeSurrounding(buf, col, `"`); err != nil {
				return err
			}
		}
		if len(in.Constraint.Including) > 0 {
			if _, err := buf.WriteString(") INCLUDE ("); err != nil {
				return err
			}
			for idx, col := range in.Constraint.Including {
				if idx >= 1 {
					if _, err := buf.WriteString(", "); err != nil {
						return err
					}
				}
				if err := writeSurrounding(buf, col, `"`); err != nil {
					return err
				}
			}
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
		if in.Constraint.IndexTableSpace != "" {
			if _, err := buf.WriteString(" USING INDEX TABLESPACE "); err != nil {
				return err
			}
			if err := writeSurrounding(buf, in.Constraint.IndexTableSpace, `"`); err != nil {
				return err
			}
		}
	case ast.ConstraintTypeCheck:
		if _, err := buf.WriteString("CHECK ("); err != nil {
			return err
		}
		if _, err := buf.WriteString(in.Constraint.Expression.Text()); err != nil {
			return err
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	}
	return nil
}

func deparseDropConstraint(context parser.DeparseContext, in *ast.DropConstraintStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, parser.DeparseIndentString); err != nil {
		return err
	}
	if _, err := buf.WriteString("DROP CONSTRAINT "); err != nil {
		return err
	}

	return writeSurrounding(buf, in.ConstraintName, `"`)
}

func deparseDropNotNull(context parser.DeparseContext, in *ast.DropNotNullStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, parser.DeparseIndentString); err != nil {
		return err
	}

	if _, err := buf.WriteString("ALTER COLUMN "); err != nil {
		return err
	}
	if err := writeSurrounding(buf, in.ColumnName, `"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(" DROP NOT NULL"); err != nil {
		return err
	}
	return nil
}

func deparseSetNotNull(context parser.DeparseContext, in *ast.SetNotNullStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, parser.DeparseIndentString); err != nil {
		return err
	}

	if _, err := buf.WriteString("ALTER COLUMN "); err != nil {
		return err
	}
	if err := writeSurrounding(buf, in.ColumnName, `"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(" SET NOT NULL"); err != nil {
		return err
	}
	return nil
}

func deparseDropColumn(context parser.DeparseContext, in *ast.DropColumnStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, parser.DeparseIndentString); err != nil {
		return err
	}

	if _, err := buf.WriteString("DROP COLUMN "); err != nil {
		return err
	}
	return writeSurrounding(buf, in.ColumnName, `"`)
}

func deparseAlterColumnType(context parser.DeparseContext, in *ast.AlterColumnTypeStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, parser.DeparseIndentString); err != nil {
		return err
	}

	if _, err := buf.WriteString("ALTER COLUMN "); err != nil {
		return err
	}
	if err := writeSurrounding(buf, in.ColumnName, `"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(" SET DATA TYPE "); err != nil {
		return err
	}
	return deparseDataType(context, in.Type, buf)
}

func deparseAddColumnList(context parser.DeparseContext, in *ast.AddColumnListStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, parser.DeparseIndentString); err != nil {
		return err
	}

	if _, err := buf.WriteString("ADD COLUMN "); err != nil {
		return err
	}

	if len(in.ColumnList) != 1 {
		return errors.Errorf("PostgreSQL doesn't support zero or multi-columns for ALTER TABLE ADD COLUMN statements")
	}

	return deparseColumnDef(parser.DeparseContext{IndentLevel: 0}, in.ColumnList[0], buf)
}

func deparseCreateTable(context parser.DeparseContext, in *ast.CreateTableStmt, buf *strings.Builder) error {
	if _, err := buf.WriteString("CREATE TABLE"); err != nil {
		return err
	}
	if in.IfNotExists {
		if _, err := buf.WriteString(" IF NOT EXISTS"); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(" "); err != nil {
		return err
	}
	if err := deparseTableDef(context, in.Name, buf); err != nil {
		return err
	}

	if len(in.ColumnList) != 0 {
		if _, err := buf.WriteString(" ("); err != nil {
			return err
		}
	}
	columnContext := parser.DeparseContext{
		IndentLevel: context.IndentLevel + 1,
	}
	for i, column := range in.ColumnList {
		if i != 0 {
			if _, err := buf.WriteString(","); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
		if err := deparseColumnDef(columnContext, column, buf); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n"); err != nil {
		return err
	}
	if len(in.ColumnList) != 0 {
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	}

	return nil
}

func deparseColumnDef(context parser.DeparseContext, in *ast.ColumnDef, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, parser.DeparseIndentString); err != nil {
		return err
	}

	if err := writeSurrounding(buf, in.ColumnName, "\""); err != nil {
		return err
	}
	if _, err := buf.WriteString(" "); err != nil {
		return err
	}
	if err := deparseDataType(context, in.Type, buf); err != nil {
		return err
	}
	for _, constraint := range in.ConstraintList {
		if _, err := buf.WriteString(" "); err != nil {
			return err
		}
		if err := deparseColumnConstraint(context, constraint, buf); err != nil {
			return err
		}
	}
	return nil
}

func deparseColumnConstraint(_ parser.DeparseContext, in *ast.ConstraintDef, buf *strings.Builder) error {
	if in.Name != "" {
		if _, err := buf.WriteString("CONSTRAINT "); err != nil {
			return err
		}
		if err := writeSurrounding(buf, in.Name, "\""); err != nil {
			return err
		}
		if _, err := buf.WriteString(" "); err != nil {
			return err
		}
	}
	switch in.Type {
	case ast.ConstraintTypeNotNull:
		if _, err := buf.WriteString("NOT NULL"); err != nil {
			return err
		}
	case ast.ConstraintTypeUnique:
		if _, err := buf.WriteString("UNIQUE"); err != nil {
			return err
		}
	case ast.ConstraintTypePrimary:
		if _, err := buf.WriteString("PRIMARY KEY"); err != nil {
			return err
		}
	case ast.ConstraintTypeDefault:
		if _, err := buf.WriteString("DEFAULT "); err != nil {
			return err
		}
		if _, err := buf.WriteString(in.Expression.Text()); err != nil {
			return err
		}
	default:
		return errors.Errorf("failed to deparse column constraint: not support %d", in.Type)
	}
	return nil
}

func deparseTableDef(_ parser.DeparseContext, in *ast.TableDef, buf *strings.Builder) error {
	if in.Schema != "" {
		if err := writeSurrounding(buf, in.Schema, "\""); err != nil {
			return err
		}
		if _, err := buf.WriteString("."); err != nil {
			return err
		}
	}
	return writeSurrounding(buf, in.Name, "\"")
}

func deparseDataType(_ parser.DeparseContext, in ast.DataType, buf *strings.Builder) error {
	switch node := in.(type) {
	case *ast.Integer:
		switch node.Size {
		case 8:
			if _, err := buf.WriteString("bigint"); err != nil {
				return err
			}
		case 4:
			if _, err := buf.WriteString("integer"); err != nil {
				return err
			}
		case 2:
			if _, err := buf.WriteString("smallint"); err != nil {
				return err
			}
		default:
			return errors.Errorf("failed to deparse integer with %d size", node.Size)
		}
	case *ast.Decimal:
		if _, err := buf.WriteString("numeric"); err != nil {
			return err
		}
		if node.Precision != 0 {
			if _, err := buf.WriteString("("); err != nil {
				return err
			}
			if _, err := buf.WriteString(strconv.Itoa(node.Precision)); err != nil {
				return err
			}
		}
		if node.Scale != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
			if _, err := buf.WriteString(strconv.Itoa(node.Scale)); err != nil {
				return err
			}
		}
		if node.Precision != 0 {
			if _, err := buf.WriteString(")"); err != nil {
				return err
			}
		}
	case *ast.Float:
		switch node.Size {
		case 8:
			if _, err := buf.WriteString("double precision"); err != nil {
				return err
			}
		case 4:
			if _, err := buf.WriteString("real"); err != nil {
				return err
			}
		default:
			return errors.Errorf("failed to deparse float with %d size", node.Size)
		}
	case *ast.Serial:
		switch node.Size {
		case 8:
			if _, err := buf.WriteString("bigserial"); err != nil {
				return err
			}
		case 4:
			if _, err := buf.WriteString("serial"); err != nil {
				return err
			}
		case 2:
			if _, err := buf.WriteString("smallserial"); err != nil {
				return err
			}
		default:
			return errors.Errorf("failed to deparse serial with %d size", node.Size)
		}
	case *ast.Character:
		if _, err := buf.WriteString("character("); err != nil {
			return err
		}
		if _, err := buf.WriteString(strconv.Itoa(node.Size)); err != nil {
			return err
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	case *ast.CharacterVarying:
		if _, err := buf.WriteString("character varying("); err != nil {
			return err
		}
		if _, err := buf.WriteString(strconv.Itoa(node.Size)); err != nil {
			return err
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	case *ast.Text:
		if _, err := buf.WriteString("text"); err != nil {
			return err
		}
	case *ast.UnconvertedDataType:
		var nameList []string
		for _, name := range node.Name {
			nameList = append(nameList, fmt.Sprintf(`"%s"`, name))
		}
		if _, err := buf.WriteString(strings.Join(nameList, ".")); err != nil {
			return err
		}
	default:
		return errors.Errorf("failed to deparse data type %T", in)
	}
	return nil
}

func deparseCreateSchema(ctx parser.DeparseContext, in *ast.CreateSchemaStmt, buf *strings.Builder) error {
	if _, err := buf.WriteString("CREATE SCHEMA "); err != nil {
		return err
	}

	if in.IfNotExists {
		if _, err := buf.WriteString("IF NOT EXISTS "); err != nil {
			return err
		}
	}
	if in.Name != "" {
		if err := writeSurrounding(buf, in.Name, `"`); err != nil {
			return err
		}
	}
	if in.RoleSpec != nil && in.RoleSpec.Type != ast.RoleSpecTypeNone {
		if in.Name != "" {
			if _, err := buf.WriteString(" "); err != nil {
				return err
			}
		}
		if err := deparseRoleSpec(ctx, in.RoleSpec, buf); err != nil {
			return err
		}
	}
	for _, ele := range in.SchemaElementList {
		if _, err := buf.WriteString(" "); err != nil {
			return err
		}
		if createTableStmt, ok := ele.(*ast.CreateTableStmt); ok {
			if err := deparseCreateTable(ctx, createTableStmt, buf); err != nil {
				return err
			}
		}
	}
	return nil
}

func deparseRoleSpec(_ parser.DeparseContext, in *ast.RoleSpec, buf *strings.Builder) error {
	if in != nil && in.Type != ast.RoleSpecTypeNone {
		if _, err := buf.WriteString("AUTHORIZATION "); err != nil {
			return err
		}
		switch in.Type {
		case ast.RoleSpecTypeUser:
			if err := writeSurrounding(buf, in.Value, `"`); err != nil {
				return err
			}
		case ast.RoleSpecTypeCurrentRole:
			if _, err := buf.WriteString("CURRENT_ROLE"); err != nil {
				return err
			}
		case ast.RoleSpecTypeCurrentUser:
			if _, err := buf.WriteString("CURRENT_USER"); err != nil {
				return err
			}
		case ast.RoleSpecTypeSessionUser:
			if _, err := buf.WriteString("SESSION_USER"); err != nil {
				return err
			}
		}
	}
	return nil
}

func deparseDropSchema(_ parser.DeparseContext, in *ast.DropSchemaStmt, buf *strings.Builder) error {
	if in == nil {
		return nil
	}
	if _, err := buf.WriteString("DROP SCHEMA"); err != nil {
		return err
	}
	if in.IfExists {
		if _, err := buf.WriteString(" IF EXISTS"); err != nil {
			return err
		}
	}
	for idx, schema := range in.SchemaList {
		if idx >= 1 {
			if _, err := buf.WriteString(","); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(" "); err != nil {
			return err
		}
		if err := writeSurrounding(buf, schema, `"`); err != nil {
			return err
		}
	}
	return deparseDropBehavior(parser.DeparseContext{}, in.Behavior, buf)
}

func deparseDropBehavior(_ parser.DeparseContext, behavior ast.DropBehavior, buf *strings.Builder) error {
	if behavior == ast.DropBehaviorCascade {
		if _, err := buf.WriteString(" CASCADE"); err != nil {
			return err
		}
	}
	return nil
}

func writeSurrounding(buf *strings.Builder, s string, enclosure string) error {
	if _, err := buf.WriteString(enclosure); err != nil {
		return err
	}
	if _, err := buf.WriteString(s); err != nil {
		return err
	}
	if _, err := buf.WriteString(enclosure); err != nil {
		return err
	}
	return nil
}
