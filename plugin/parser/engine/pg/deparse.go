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
		return deparseCreateTable(context, node, buf)
	case *ast.AlterTableStmt:
		return deparseAlterTable(context, node, buf)
	case *ast.TableDef:
		return deparseTableDef(context, node, buf)
	case *ast.ColumnDef:
		return deparseColumnDef(context, node, buf)
	}

	return errors.Errorf("failed to deparse %T", in)
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
		}
	}
	if _, err := buf.WriteString(";\n"); err != nil {
		return err
	}
	return nil
}

func deparseAlterColumnType(context parser.DeparseContext, in *ast.AlterColumnTypeStmt, buf *strings.Builder) error {
	for i := 0; i < context.IndentLevel; i++ {
		if _, err := buf.WriteString(parser.DeparseIndentString); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString("ALTER COLUMN "); err != nil {
		return err
	}
	if err := deparseTableDef(context, in.Table, buf); err != nil {
		return err
	}
	if _, err := buf.WriteString(" SET DATA TYPE "); err != nil {
		return err
	}
	return deparseDataType(context, in.Type, buf)
}

func deparseAddColumnList(context parser.DeparseContext, in *ast.AddColumnListStmt, buf *strings.Builder) error {
	for i := 0; i < context.IndentLevel; i++ {
		if _, err := buf.WriteString(parser.DeparseIndentString); err != nil {
			return err
		}
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
	for i := 0; i < context.IndentLevel; i++ {
		if _, err := buf.WriteString(parser.DeparseIndentString); err != nil {
			return err
		}
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
