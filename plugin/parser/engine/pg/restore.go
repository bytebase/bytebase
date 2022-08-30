package pg

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
)

func restore(context parser.RestoreContext, in ast.Node, buf *strings.Builder) error {
	switch node := in.(type) {
	case ast.DataType:
		return restoreDataType(context, node, buf)
	case *ast.CreateTableStmt:
		return restoreCreateTable(context, node, buf)
	case *ast.TableDef:
		return restoreTableDef(context, node, buf)
	case *ast.ColumnDef:
		return restoreColumnDef(context, node, buf)
	}

	return errors.Errorf("failed to restore %T", in)
}

func restoreCreateTable(context parser.RestoreContext, in *ast.CreateTableStmt, buf *strings.Builder) error {
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
	if err := restoreTableDef(context, in.Name, buf); err != nil {
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
		if err := restoreColumnDef(context, column, buf); err != nil {
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

func restoreColumnDef(context parser.RestoreContext, in *ast.ColumnDef, buf *strings.Builder) error {
	if err := writeSurrounding(buf, in.ColumnName, "\""); err != nil {
		return err
	}
	if _, err := buf.WriteString(" "); err != nil {
		return err
	}
	if err := restoreDataType(context, in.Type, buf); err != nil {
		return err
	}
	for _, constraint := range in.ConstraintList {
		if _, err := buf.WriteString(" "); err != nil {
			return err
		}
		if err := restoreColumnConstraint(context, constraint, buf); err != nil {
			return err
		}
	}
	return nil
}

func restoreColumnConstraint(_ parser.RestoreContext, in *ast.ConstraintDef, buf *strings.Builder) error {
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
		return errors.Errorf("failed to restore column constraint: not support %d", in.Type)
	}
	return nil
}

func restoreTableDef(_ parser.RestoreContext, in *ast.TableDef, buf *strings.Builder) error {
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

func restoreDataType(_ parser.RestoreContext, in ast.DataType, buf *strings.Builder) error {
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
	case *ast.UnconvertedDataType:
		var nameList []string
		for _, name := range node.Name {
			nameList = append(nameList, fmt.Sprintf(`"%s"`, name))
		}
		if _, err := buf.WriteString(strings.Join(nameList, ".")); err != nil {
			return err
		}
	default:
		return errors.Errorf("failed to restore data type %T", in)
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
