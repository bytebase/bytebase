package pg

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
)

func deparseImpl(context DeparseContext, in ast.Node, buf *strings.Builder) error {
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
	case *ast.ConstraintDef:
		return deparseConstraintDef(context, node, buf)
	case *ast.CreateIndexStmt:
		if err := deparseCreateIndex(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.DropIndexStmt:
		if err := deparseDropIndex(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.CreateSequenceStmt:
		if err := deparseCreateSequence(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.AlterSequenceStmt:
		if err := deparseAlterSequence(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.DropSequenceStmt:
		if err := deparseDropSequence(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.DropExtensionStmt:
		if err := deparseDropExtension(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.DropFunctionStmt:
		if err := deparseDropFunction(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.DropTriggerStmt:
		if err := deparseDropTrigger(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.DropTypeStmt:
		if err := deparseDropType(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.AlterTypeStmt:
		if err := deparseAlterType(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.RenameSchemaStmt:
		if err := deparseRenameSchema(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	case *ast.CommentStmt:
		if err := deparseComment(context, node, buf); err != nil {
			return err
		}
		return buf.WriteByte(';')
	}
	return errors.Errorf("failed to deparse %T", in)
}

func deparseComment(context DeparseContext, in *ast.CommentStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
		return err
	}

	switch in.Type {
	case ast.ObjectTypeTable:
		if _, err := buf.WriteString("COMMENT ON TABLE \""); err != nil {
			return err
		}
		tableDef, ok := in.Object.(*ast.TableDef)
		if !ok {
			return errors.Errorf("expect *ast.TableDef, but got %T", in.Object)
		}
		if tableDef.Schema != "" {
			if _, err := buf.WriteString(tableDef.Schema); err != nil {
				return err
			}
			if _, err := buf.WriteString("\".\""); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(tableDef.Name); err != nil {
			return err
		}
		if _, err := buf.WriteString("\" IS '"); err != nil {
			return err
		}
		escapeComment := strings.ReplaceAll(in.Comment, "'", "''")
		if _, err := buf.WriteString(escapeComment); err != nil {
			return err
		}
		_, err := buf.WriteString("'")
		return err
	case ast.ObjectTypeColumn:
		if _, err := buf.WriteString("COMMENT ON COLUMN \""); err != nil {
			return err
		}
		columnNameDef, ok := in.Object.(*ast.ColumnNameDef)
		if !ok {
			return errors.Errorf("expect *ast.ColumnNameDef, but got %T", in.Object)
		}
		if columnNameDef.Table.Schema != "" {
			if _, err := buf.WriteString(columnNameDef.Table.Schema); err != nil {
				return err
			}
			if _, err := buf.WriteString("\".\""); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(columnNameDef.Table.Name); err != nil {
			return err
		}
		if _, err := buf.WriteString("\".\""); err != nil {
			return err
		}
		if _, err := buf.WriteString(columnNameDef.ColumnName); err != nil {
			return err
		}
		if _, err := buf.WriteString("\" IS '"); err != nil {
			return err
		}
		escapeComment := strings.ReplaceAll(in.Comment, "'", "''")
		if _, err := buf.WriteString(escapeComment); err != nil {
			return err
		}
		_, err := buf.WriteString("'")
		return err
	default:
		return errors.Errorf("failed to deparse comment for type %v", in.Type)
	}
}

func deparseDropIndex(context DeparseContext, in *ast.DropIndexStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
		return err
	}

	if _, err := buf.WriteString("DROP INDEX "); err != nil {
		return err
	}

	if in.IfExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}

	for i, index := range in.IndexList {
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}

		if index.Table != nil && index.Table.Schema != "" {
			if err := writeSurrounding(buf, index.Table.Schema, `"`); err != nil {
				return err
			}
			if err := buf.WriteByte('.'); err != nil {
				return err
			}
		}

		if err := writeSurrounding(buf, index.Name, `"`); err != nil {
			return err
		}
	}

	if in.Behavior == ast.DropBehaviorCascade {
		if _, err := buf.WriteString(" CASCADE"); err != nil {
			return err
		}
	}
	return nil
}

func deparseCreateIndex(context DeparseContext, in *ast.CreateIndexStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
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

	if in.Concurrently {
		if _, err := buf.WriteString("CONCURRENTLY "); err != nil {
			return err
		}
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
		if err := deparseIndexKey(DeparseContext{IndentLevel: 0}, key, buf); err != nil {
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
	case ast.IndexMethodTypeIvfflat:
		_, err = buf.WriteString("ivfflat")
	}
	return err
}

func deparseIndexKey(context DeparseContext, in *ast.IndexKeyDef, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
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

func deparseDropTable(context DeparseContext, in *ast.DropTableStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
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
		if err := deparseTableDef(DeparseContext{IndentLevel: 0}, table, buf); err != nil {
			return err
		}
	}
	return deparseDropBehavior(context, in.Behavior, buf)
}

func deparseAlterTable(context DeparseContext, in *ast.AlterTableStmt, buf *strings.Builder) error {
	if _, err := buf.WriteString("ALTER TABLE "); err != nil {
		return err
	}
	if err := deparseTableDef(context, in.Table, buf); err != nil {
		return err
	}
	itemContext := DeparseContext{
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
		case *ast.RenameColumnStmt:
			if len(in.AlterItemList) != 1 {
				return errors.Errorf("deparse failed, RenameColumnStmt needs to be alone in a ALTER TABLE statement")
			}
			if err := deparseRenameColumn(itemContext, action, buf); err != nil {
				return err
			}
		case *ast.RenameTableStmt:
			if len(in.AlterItemList) != 1 {
				return errors.Errorf("deparse failed, RenameTableStmt needs to be alone in a ALTER TABLE statement")
			}
			if err := deparseRenameTable(itemContext, action, buf); err != nil {
				return err
			}
		}
	}
	return nil
}

func deparseRenameColumn(context DeparseContext, in *ast.RenameColumnStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
		return err
	}
	if _, err := buf.WriteString("RENAME COLUMN "); err != nil {
		return err
	}
	if err := writeSurrounding(buf, in.ColumnName, `"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(" TO "); err != nil {
		return err
	}
	return writeSurrounding(buf, in.NewName, `"`)
}

func deparseRenameTable(context DeparseContext, in *ast.RenameTableStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
		return err
	}
	if _, err := buf.WriteString("RENAME TO "); err != nil {
		return err
	}
	return writeSurrounding(buf, in.NewName, `"`)
}

func deparseSetDefault(context DeparseContext, in *ast.SetDefaultStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
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

func deparseDropDefault(context DeparseContext, in *ast.DropDefaultStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
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

func deparseAddConstraint(context DeparseContext, in *ast.AddConstraintStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
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
	return deparseConstraintDef(context, in.Constraint, buf)
}

func deparseConstraintDef(_ DeparseContext, in *ast.ConstraintDef, buf *strings.Builder) error {
	switch in.Type {
	case ast.ConstraintTypeUniqueUsingIndex:
		if _, err := buf.WriteString("UNIQUE USING INDEX "); err != nil {
			return err
		}
		if err := writeSurrounding(buf, in.IndexName, `"`); err != nil {
			return err
		}

		if in.Initdeferred {
			if _, err := buf.WriteString(" INITIALLY DEFERRED"); err != nil {
				return err
			}
		} else if in.Deferrable {
			if _, err := buf.WriteString(" DEFERRABLE"); err != nil {
				return err
			}
		}
	case ast.ConstraintTypeUnique:
		if _, err := buf.WriteString("UNIQUE ("); err != nil {
			return err
		}
		if err := deparseKeyList(DeparseContext{}, in.KeyList, buf); err != nil {
			return err
		}

		if len(in.Including) > 0 {
			if _, err := buf.WriteString(") INCLUDE ("); err != nil {
				return err
			}
			if err := deparseKeyList(DeparseContext{}, in.Including, buf); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
		if in.IndexTableSpace != "" {
			if _, err := buf.WriteString(" USING INDEX TABLESPACE "); err != nil {
				return err
			}
			if err := writeSurrounding(buf, in.IndexTableSpace, `"`); err != nil {
				return err
			}
		}
	case ast.ConstraintTypePrimary:
		if _, err := buf.WriteString("PRIMARY KEY ("); err != nil {
			return err
		}
		if err := deparseKeyList(DeparseContext{}, in.KeyList, buf); err != nil {
			return err
		}
		if len(in.Including) > 0 {
			if _, err := buf.WriteString(") INCLUDE ("); err != nil {
				return err
			}
			if err := deparseKeyList(DeparseContext{}, in.Including, buf); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
		if in.IndexTableSpace != "" {
			if _, err := buf.WriteString(" USING INDEX TABLESPACE "); err != nil {
				return err
			}
			if err := writeSurrounding(buf, in.IndexTableSpace, `"`); err != nil {
				return err
			}
		}
	case ast.ConstraintTypeCheck:
		if _, err := buf.WriteString("CHECK ("); err != nil {
			return err
		}
		if _, err := buf.WriteString(in.Expression.Text()); err != nil {
			return err
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	case ast.ConstraintTypeExclusion:
		if _, err := buf.WriteString("EXCLUDE USING "); err != nil {
			return err
		}
		if err := deparseIndexMethod(in.AccessMethod, buf); err != nil {
			return err
		}
		if _, err := buf.WriteString(" ("); err != nil {
			return err
		}
		if _, err := buf.WriteString(in.Exclusions); err != nil {
			return err
		}
		if err := buf.WriteByte(')'); err != nil {
			return err
		}
		if in.WhereClause != "" {
			if _, err := buf.WriteString(" WHERE ("); err != nil {
				return err
			}
			if _, err := buf.WriteString(in.WhereClause); err != nil {
				return err
			}
			if err := buf.WriteByte(')'); err != nil {
				return err
			}
		}
	case ast.ConstraintTypeForeign:
		if _, err := buf.WriteString("FOREIGN KEY ("); err != nil {
			return err
		}
		for i, column := range in.KeyList {
			if i != 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if err := writeSurrounding(buf, column, `"`); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(") REFERENCES "); err != nil {
			return err
		}
		if err := deparseTableDef(DeparseContext{}, in.Foreign.Table, buf); err != nil {
			return err
		}
		if _, err := buf.WriteString(" ("); err != nil {
			return err
		}
		for i, column := range in.Foreign.ColumnList {
			if i != 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if err := writeSurrounding(buf, column, `"`); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}

		if err := deparseForeignMatchType(DeparseContext{}, in.Foreign.MatchType, buf); err != nil {
			return err
		}

		if err := deparseReferentialAction(DeparseContext{}, "ON DELETE", in.Foreign.OnDelete, buf); err != nil {
			return err
		}

		if err := deparseReferentialAction(DeparseContext{}, "ON UPDATE", in.Foreign.OnUpdate, buf); err != nil {
			return err
		}
	}
	return nil
}

func deparseReferentialAction(_ DeparseContext, prefix string, in *ast.ReferentialActionDef, buf *strings.Builder) error {
	if in.Type == ast.ReferentialActionTypeNoAction {
		// It's default value, no need to print.
		return nil
	}
	if err := buf.WriteByte(' '); err != nil {
		return err
	}
	if _, err := buf.WriteString(prefix); err != nil {
		return err
	}
	if err := buf.WriteByte(' '); err != nil {
		return err
	}
	switch in.Type {
	case ast.ReferentialActionTypeRestrict:
		if _, err := buf.WriteString("RESTRICT"); err != nil {
			return err
		}
	case ast.ReferentialActionTypeCascade:
		if _, err := buf.WriteString("CASCADE"); err != nil {
			return err
		}
	case ast.ReferentialActionTypeSetNull:
		if _, err := buf.WriteString("SET NULL"); err != nil {
			return err
		}
	case ast.ReferentialActionTypeSetDefault:
		if _, err := buf.WriteString("SET DEFAULT"); err != nil {
			return err
		}
	}
	return nil
}

func deparseForeignMatchType(_ DeparseContext, in ast.ForeignMatchType, buf *strings.Builder) error {
	switch in {
	case ast.ForeignMatchTypeSimple:
		// It's default value, no need to print.
	case ast.ForeignMatchTypeFull:
		if _, err := buf.WriteString(" MATCH FULL"); err != nil {
			return err
		}
	case ast.ForeignMatchTypePartial:
		if _, err := buf.WriteString(" MATCH PARTIAL"); err != nil {
			return err
		}
	}
	return nil
}

func deparseDropConstraint(context DeparseContext, in *ast.DropConstraintStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
		return err
	}
	if _, err := buf.WriteString("DROP CONSTRAINT "); err != nil {
		return err
	}
	if in.IfExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}

	return writeSurrounding(buf, in.ConstraintName, `"`)
}

func deparseDropNotNull(context DeparseContext, in *ast.DropNotNullStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
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

func deparseSetNotNull(context DeparseContext, in *ast.SetNotNullStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
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

func deparseDropColumn(context DeparseContext, in *ast.DropColumnStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
		return err
	}

	if _, err := buf.WriteString("DROP COLUMN "); err != nil {
		return err
	}

	if in.IfExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}

	if err := writeSurrounding(buf, in.ColumnName, `"`); err != nil {
		return err
	}

	return deparseDropBehavior(context, in.Behavior, buf)
}

func deparseAlterColumnType(context DeparseContext, in *ast.AlterColumnTypeStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
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

func deparseAddColumnList(context DeparseContext, in *ast.AddColumnListStmt, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
		return err
	}

	if _, err := buf.WriteString("ADD COLUMN "); err != nil {
		return err
	}

	if in.IfNotExists {
		if _, err := buf.WriteString("IF NOT EXISTS "); err != nil {
			return err
		}
	}

	if len(in.ColumnList) != 1 {
		return errors.Errorf("PostgreSQL doesn't support zero or multi-columns for ALTER TABLE ADD COLUMN statements")
	}

	return deparseColumnDef(DeparseContext{IndentLevel: 0}, in.ColumnList[0], buf)
}

func deparseCreateTable(context DeparseContext, in *ast.CreateTableStmt, buf *strings.Builder) error {
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

	if len(in.ColumnList)+len(in.ConstraintList) != 0 {
		if _, err := buf.WriteString(" ("); err != nil {
			return err
		}
	}
	columnContext := DeparseContext{
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
	for i, constraint := range in.ConstraintList {
		if i != 0 || len(in.ColumnList) > 0 {
			if _, err := buf.WriteString(","); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
		if err := deparseTableConstraint(columnContext, constraint, buf); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n"); err != nil {
		return err
	}
	if len(in.ColumnList)+len(in.ConstraintList) != 0 {
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	}
	if in.PartitionDef != nil {
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
		if err := deparsePartitionDef(columnContext, in.PartitionDef, buf); err != nil {
			return err
		}
	}

	return nil
}

func deparsePartitionDef(_ DeparseContext, in *ast.PartitionDef, buf *strings.Builder) error {
	if _, err := buf.WriteString("PARTITION BY "); err != nil {
		return err
	}
	if _, err := buf.WriteString(strings.ToUpper(in.Strategy)); err != nil {
		return err
	}
	if _, err := buf.WriteString(" ("); err != nil {
		return err
	}
	for i, key := range in.KeyList {
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		switch key.Type {
		case ast.PartitionKeyTypeColumn:
			if err := writeSurrounding(buf, key.Key, `"`); err != nil {
				return err
			}
		case ast.PartitionKeyTypeExpression:
			if _, err := buf.WriteString("("); err != nil {
				return err
			}
			if _, err := buf.WriteString(key.Key); err != nil {
				return err
			}
			if _, err := buf.WriteString(")"); err != nil {
				return err
			}
		}
	}
	if _, err := buf.WriteString(")"); err != nil {
		return err
	}

	return nil
}

func deparseTableConstraint(context DeparseContext, in *ast.ConstraintDef, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
		return err
	}

	if in.Name != "" {
		if _, err := buf.WriteString("CONSTRAINT "); err != nil {
			return err
		}
		if err := writeSurrounding(buf, in.Name, `"`); err != nil {
			return err
		}
		if _, err := buf.WriteString(" "); err != nil {
			return err
		}
	}
	return deparseConstraintDef(context, in, buf)
}

func deparseColumnDef(context DeparseContext, in *ast.ColumnDef, buf *strings.Builder) error {
	if err := context.WriteIndent(buf, deparseIndentString); err != nil {
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

func deparseColumnConstraint(_ DeparseContext, in *ast.ConstraintDef, buf *strings.Builder) error {
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
	case ast.ConstraintTypeNull:
		if _, err := buf.WriteString("NULL"); err != nil {
			return err
		}
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
	case ast.ConstraintTypeGenerated:
		if _, err := buf.WriteString(fmt.Sprintf("GENERATED ALWAYS AS (%s) STORED", in.Expression.Text())); err != nil {
			return err
		}
	default:
		return errors.Errorf("failed to deparse column constraint: not support %d", in.Type)
	}
	return nil
}

func deparseTableDef(_ DeparseContext, in *ast.TableDef, buf *strings.Builder) error {
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

func deparseDataType(_ DeparseContext, in ast.DataType, buf *strings.Builder) error {
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
		if _, err := buf.WriteString(node.Text()); err != nil {
			return err
		}
	default:
		return errors.Errorf("failed to deparse data type %T", in)
	}
	return nil
}

func deparseAlterSequence(ctx DeparseContext, in *ast.AlterSequenceStmt, buf *strings.Builder) error {
	if _, err := buf.WriteString("ALTER SEQUENCE "); err != nil {
		return err
	}
	if in.IfExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}
	if err := deparseSequenceName(ctx, in.Name, buf); err != nil {
		return err
	}
	// Write alter items.
	itemContext := DeparseContext{IndentLevel: ctx.IndentLevel + 1}
	if in.Type != nil {
		if err := buf.WriteByte('\n'); err != nil {
			return err
		}
		if err := itemContext.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}

		if _, err := buf.WriteString("AS "); err != nil {
			return err
		}
		if err := deparseDataType(DeparseContext{}, in.Type, buf); err != nil {
			return err
		}
	}

	if in.IncrementBy != nil {
		if err := buf.WriteByte('\n'); err != nil {
			return err
		}
		if err := itemContext.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}

		if _, err := buf.WriteString(fmt.Sprintf("INCREMENT BY %d", *in.IncrementBy)); err != nil {
			return err
		}
	}

	if in.NoMinValue {
		if err := buf.WriteByte('\n'); err != nil {
			return err
		}
		if err := itemContext.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}

		if _, err := buf.WriteString("NO MINVALUE"); err != nil {
			return err
		}
	} else if in.MinValue != nil {
		if err := buf.WriteByte('\n'); err != nil {
			return err
		}
		if err := itemContext.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}

		if _, err := buf.WriteString(fmt.Sprintf("MINVALUE %d", *in.MinValue)); err != nil {
			return err
		}
	}

	if in.NoMaxValue {
		if err := buf.WriteByte('\n'); err != nil {
			return err
		}
		if err := itemContext.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}

		if _, err := buf.WriteString("NO MAXVALUE"); err != nil {
			return err
		}
	} else if in.MaxValue != nil {
		if err := buf.WriteByte('\n'); err != nil {
			return err
		}
		if err := itemContext.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}

		if _, err := buf.WriteString(fmt.Sprintf("MAXVALUE %d", *in.MaxValue)); err != nil {
			return err
		}
	}

	if in.StartWith != nil {
		if err := buf.WriteByte('\n'); err != nil {
			return err
		}
		if err := itemContext.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}

		if _, err := buf.WriteString(fmt.Sprintf("START WITH %d", *in.StartWith)); err != nil {
			return err
		}
	}

	if in.RestartWith != nil {
		if err := buf.WriteByte('\n'); err != nil {
			return err
		}
		if err := itemContext.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}

		if _, err := buf.WriteString(fmt.Sprintf("RESTART WITH %d", *in.RestartWith)); err != nil {
			return err
		}
	}

	if in.Cache != nil {
		if err := buf.WriteByte('\n'); err != nil {
			return err
		}
		if err := itemContext.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}

		if _, err := buf.WriteString(fmt.Sprintf("CACHE %d", *in.Cache)); err != nil {
			return err
		}
	}

	if in.Cycle != nil {
		if err := buf.WriteByte('\n'); err != nil {
			return err
		}
		if err := itemContext.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}

		if !*in.Cycle {
			if _, err := buf.WriteString("NO "); err != nil {
				return err
			}
		}

		if _, err := buf.WriteString("CYCLE"); err != nil {
			return err
		}
	}

	if in.OwnedByNone {
		if err := buf.WriteByte('\n'); err != nil {
			return err
		}
		if err := itemContext.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}

		if _, err := buf.WriteString("OWNED BY NONE"); err != nil {
			return err
		}
	} else if in.OwnedBy != nil {
		if err := buf.WriteByte('\n'); err != nil {
			return err
		}
		if err := itemContext.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}

		if _, err := buf.WriteString("OWNED BY "); err != nil {
			return err
		}
		if err := deparseColumnNameDef(DeparseContext{}, in.OwnedBy, buf); err != nil {
			return err
		}
	}
	return nil
}

func deparseDropFunction(ctx DeparseContext, in *ast.DropFunctionStmt, buf *strings.Builder) error {
	if _, err := buf.WriteString("DROP FUNCTION "); err != nil {
		return err
	}
	if in.IfExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}
	for i, function := range in.FunctionList {
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if err := deparseFunctionSignature(function, buf); err != nil {
			return err
		}
	}
	return deparseDropBehavior(ctx, in.Behavior, buf)
}

func deparseFunctionSignature(function *ast.FunctionDef, buf *strings.Builder) error {
	if function.Schema != "" {
		if err := writeSurrounding(buf, function.Schema, `"`); err != nil {
			return err
		}
		if err := buf.WriteByte('.'); err != nil {
			return err
		}
	}
	if err := writeSurrounding(buf, function.Name, `"`); err != nil {
		return err
	}
	if err := buf.WriteByte('('); err != nil {
		return err
	}
	total := 0
	for _, parameter := range function.ParameterList {
		if parameter.Mode == ast.FunctionParameterModeOut {
			continue
		}
		if total != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		total++
		if err := deparseDataType(DeparseContext{}, parameter.Type, buf); err != nil {
			return err
		}
	}
	return buf.WriteByte(')')
}

func deparseRenameSchema(_ DeparseContext, in *ast.RenameSchemaStmt, buf *strings.Builder) error {
	if _, err := buf.WriteString("ALTER SCHEMA "); err != nil {
		return err
	}
	if err := writeSurrounding(buf, in.Schema, `"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(" RENAME TO "); err != nil {
		return err
	}
	return writeSurrounding(buf, in.NewName, `"`)
}

func deparseAlterType(ctx DeparseContext, in *ast.AlterTypeStmt, buf *strings.Builder) error {
	if _, err := buf.WriteString("ALTER TYPE "); err != nil {
		return err
	}
	if in.Type.Schema != "" {
		if err := writeSurrounding(buf, in.Type.Schema, `"`); err != nil {
			return err
		}
		if err := buf.WriteByte('.'); err != nil {
			return err
		}
	}
	if err := writeSurrounding(buf, in.Type.Name, `"`); err != nil {
		return err
	}
	if err := buf.WriteByte(' '); err != nil {
		return err
	}

	for _, item := range in.AlterItemList {
		if node, ok := item.(*ast.AddEnumLabelStmt); ok {
			// The ADD ENUM VALUE statement use the cannot share the AlterType statement with other alter items.
			return deparseAddEnumValue(ctx, node, buf)
		}
	}
	return nil
}

func deparseAddEnumValue(_ DeparseContext, in *ast.AddEnumLabelStmt, buf *strings.Builder) error {
	if _, err := buf.WriteString("ADD VALUE "); err != nil {
		return err
	}
	if err := writeSurrounding(buf, in.NewLabel, "'"); err != nil {
		return err
	}
	switch in.Position {
	case ast.PositionTypeEnd:
		return nil
	case ast.PositionTypeBefore:
		if _, err := buf.WriteString(" BEFORE "); err != nil {
			return err
		}
	case ast.PositionTypeAfter:
		if _, err := buf.WriteString(" AFTER "); err != nil {
			return err
		}
	}
	return writeSurrounding(buf, in.NeighborLabel, "'")
}

func deparseDropType(ctx DeparseContext, in *ast.DropTypeStmt, buf *strings.Builder) error {
	if _, err := buf.WriteString("DROP TYPE "); err != nil {
		return err
	}
	if in.IfExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}
	for i, tp := range in.TypeNameList {
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if tp.Schema != "" {
			if err := writeSurrounding(buf, tp.Schema, `"`); err != nil {
				return err
			}
			if err := buf.WriteByte('.'); err != nil {
				return err
			}
		}
		if err := writeSurrounding(buf, tp.Name, `"`); err != nil {
			return err
		}
	}
	return deparseDropBehavior(ctx, in.Behavior, buf)
}

func deparseDropTrigger(ctx DeparseContext, in *ast.DropTriggerStmt, buf *strings.Builder) error {
	if _, err := buf.WriteString("DROP TRIGGER "); err != nil {
		return err
	}
	if in.IfExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}
	if err := writeSurrounding(buf, in.Trigger.Name, `"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(" ON "); err != nil {
		return err
	}
	if err := deparseTableDef(DeparseContext{}, in.Trigger.Table, buf); err != nil {
		return err
	}
	return deparseDropBehavior(ctx, in.Behavior, buf)
}

func deparseDropExtension(ctx DeparseContext, in *ast.DropExtensionStmt, buf *strings.Builder) error {
	if _, err := buf.WriteString("DROP EXTENSION "); err != nil {
		return err
	}
	if in.IfExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}
	for i, name := range in.NameList {
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if err := writeSurrounding(buf, name, `"`); err != nil {
			return err
		}
	}
	return deparseDropBehavior(ctx, in.Behavior, buf)
}

func deparseDropSequence(ctx DeparseContext, in *ast.DropSequenceStmt, buf *strings.Builder) error {
	if _, err := buf.WriteString("DROP SEQUENCE "); err != nil {
		return err
	}
	if in.IfExists {
		if _, err := buf.WriteString("IF EXISTS "); err != nil {
			return err
		}
	}
	for i, sequence := range in.SequenceNameList {
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if err := deparseSequenceName(ctx, sequence, buf); err != nil {
			return err
		}
	}
	return deparseDropBehavior(ctx, in.Behavior, buf)
}

func deparseCreateSequence(ctx DeparseContext, in *ast.CreateSequenceStmt, buf *strings.Builder) error {
	if _, err := buf.WriteString("CREATE SEQUENCE "); err != nil {
		return err
	}
	if in.IfNotExists {
		if _, err := buf.WriteString("IF NOT EXISTS "); err != nil {
			return err
		}
	}
	if err := deparseSequenceName(ctx, in.SequenceDef.SequenceName, buf); err != nil {
		return err
	}
	return depraseSequenceDef(DeparseContext{IndentLevel: ctx.IndentLevel + 1}, &in.SequenceDef, buf)
}

func depraseSequenceDef(ctx DeparseContext, in *ast.SequenceDef, buf *strings.Builder) error {
	if in.SequenceDataType != nil {
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
		if err := ctx.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}
		if _, err := buf.WriteString("AS "); err != nil {
			return err
		}
		if err := deparseDataType(ctx, in.SequenceDataType, buf); err != nil {
			return err
		}
	}
	if in.IncrementBy != nil {
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
		if err := ctx.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}
		if _, err := buf.WriteString(fmt.Sprintf("INCREMENT BY %d", *in.IncrementBy)); err != nil {
			return err
		}
	}
	if in.MinValue != nil {
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
		if err := ctx.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}
		if _, err := buf.WriteString(fmt.Sprintf("MINVALUE %d", *in.MinValue)); err != nil {
			return err
		}
	} else {
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
		if err := ctx.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}
		if _, err := buf.WriteString("NO MINVALUE"); err != nil {
			return err
		}
	}
	if in.MaxValue != nil {
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
		if err := ctx.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}
		if _, err := buf.WriteString(fmt.Sprintf("MAXVALUE %d", *in.MaxValue)); err != nil {
			return err
		}
	} else {
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
		if err := ctx.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}
		if _, err := buf.WriteString("NO MAXVALUE"); err != nil {
			return err
		}
	}
	if in.StartWith != nil {
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
		if err := ctx.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}
		if _, err := buf.WriteString(fmt.Sprintf("START WITH %d", *in.StartWith)); err != nil {
			return err
		}
	}
	if in.Cache != nil {
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
		if err := ctx.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}
		if _, err := buf.WriteString(fmt.Sprintf("CACHE %d", *in.Cache)); err != nil {
			return err
		}
	}
	if in.Cycle {
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
		if err := ctx.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}
		if _, err := buf.WriteString("CYCLE"); err != nil {
			return err
		}
	}
	if in.OwnedBy != nil {
		if _, err := buf.WriteString("\n"); err != nil {
			return err
		}
		if err := ctx.WriteIndent(buf, deparseIndentString); err != nil {
			return err
		}
		if _, err := buf.WriteString("OWNED BY "); err != nil {
			return err
		}
		if err := deparseColumnNameDef(ctx, in.OwnedBy, buf); err != nil {
			return err
		}
	}
	return nil
}

func deparseColumnNameDef(ctx DeparseContext, in *ast.ColumnNameDef, buf *strings.Builder) error {
	if err := deparseTableDef(ctx, in.Table, buf); err != nil {
		return err
	}
	if in.ColumnName != "" {
		if _, err := buf.WriteString("."); err != nil {
			return err
		}
		if err := writeSurrounding(buf, in.ColumnName, `"`); err != nil {
			return err
		}
	}
	return nil
}

func deparseSequenceName(_ DeparseContext, in *ast.SequenceNameDef, buf *strings.Builder) error {
	if in.Schema != "" {
		if err := writeSurrounding(buf, in.Schema, `"`); err != nil {
			return err
		}
		if _, err := buf.WriteString("."); err != nil {
			return err
		}
	}
	return writeSurrounding(buf, in.Name, "\"")
}

func deparseCreateSchema(ctx DeparseContext, in *ast.CreateSchemaStmt, buf *strings.Builder) error {
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

func deparseRoleSpec(_ DeparseContext, in *ast.RoleSpec, buf *strings.Builder) error {
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

func deparseDropSchema(_ DeparseContext, in *ast.DropSchemaStmt, buf *strings.Builder) error {
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
	return deparseDropBehavior(DeparseContext{}, in.Behavior, buf)
}

func deparseDropBehavior(_ DeparseContext, behavior ast.DropBehavior, buf *strings.Builder) error {
	if behavior == ast.DropBehaviorCascade {
		if _, err := buf.WriteString(" CASCADE"); err != nil {
			return err
		}
	}
	return nil
}

func deparseKeyList(_ DeparseContext, in []string, buf *strings.Builder) error {
	if len(in) == 0 {
		return nil
	}
	for idx, key := range in {
		if idx >= 1 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if err := writeSurrounding(buf, key, `"`); err != nil {
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
