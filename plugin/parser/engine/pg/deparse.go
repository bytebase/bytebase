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
	case *ast.TableDef:
		return deparseTableDef(context, node, buf)
	case *ast.ColumnDef:
		return deparseColumnDef(context, node, buf)
	case *ast.CreateSchemaStmt:
		return deparseCreateSchema(context, node, buf)
	}

	return errors.Errorf("failed to deparse %T", in)
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
		if _, err := buf.WriteString("("); err != nil {
			return err
		}
	}
	for i, column := range in.ColumnList {
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if err := deparseColumnDef(context, column, buf); err != nil {
			return err
		}
	}
	if len(in.ColumnList) != 0 {
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	}

	return nil
}

func deparseColumnDef(context parser.DeparseContext, in *ast.ColumnDef, buf *strings.Builder) error {
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
		if _, err := buf.WriteString("INT"); err != nil {
			return err
		}
		if _, err := buf.WriteString(strconv.Itoa(node.Size)); err != nil {
			return err
		}
	case *ast.Decimal:
		if _, err := buf.WriteString("DECIMAL"); err != nil {
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
		if _, err := buf.WriteString("FLOAT"); err != nil {
			return err
		}
		if _, err := buf.WriteString(strconv.Itoa(node.Size)); err != nil {
			return err
		}
	case *ast.Serial:
		if _, err := buf.WriteString("SERIAL"); err != nil {
			return err
		}
		if _, err := buf.WriteString(strconv.Itoa(node.Size)); err != nil {
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
	if in.RoleSpec != nil && in.RoleSpec.Tp != ast.RoleSpecTypeNone {
		if in.Name != "" {
			if _, err := buf.WriteString(" "); err != nil {
				return err
			}
		}
		if err := deparseRoleSpec(ctx, in.RoleSpec, buf); err != nil {
			return err
		}
	}
	for _, ele := range in.SchemaElements {
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
	if in != nil && in.Tp != ast.RoleSpecTypeNone {
		if _, err := buf.WriteString("AUTHORIZATION "); err != nil {
			return err
		}
		switch in.Tp {
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
