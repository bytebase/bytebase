package oracle

import (
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterGetDatabaseDefinition(storepb.Engine_ORACLE, GetDatabaseDefinition)
	schema.RegisterGetTableDefinition(storepb.Engine_ORACLE, GetTableDefinition)
}

func GetDatabaseDefinition(_ schema.GetDefinitionContext, to *storepb.DatabaseSchemaMetadata) (string, error) {
	if len(to.Schemas) == 0 {
		return "", nil
	}

	var buf strings.Builder

	schema := to.Schemas[0]
	for _, table := range schema.Tables {
		if err := writeTable(&buf, schema.Name, table); err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}

func GetTableDefinition(schemaName string, table *storepb.TableMetadata, _ []*storepb.SequenceMetadata) (string, error) {
	var buf strings.Builder
	if err := writeTable(&buf, schemaName, table); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func writeTable(buf *strings.Builder, schema string, table *storepb.TableMetadata) error {
	if _, err := buf.WriteString(`CREATE TABLE "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(table.Name); err != nil {
		return err
	}
	if _, err := buf.WriteString("\" (\n"); err != nil {
		return err
	}
	for i, column := range table.Columns {
		if i > 0 {
			if _, err := buf.WriteString(",\n"); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`  `); err != nil {
			return err
		}
		if err := writeColumn(buf, column); err != nil {
			return err
		}
	}

	constraints := []*storepb.IndexMetadata{}
	for _, constraint := range table.Indexes {
		if constraint.IsConstraint {
			constraints = append(constraints, constraint)
		}
	}

	for i, constraint := range constraints {
		if i+len(table.Indexes) > 0 {
			if _, err := buf.WriteString(",\n"); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`  `); err != nil {
			return err
		}
		if err := writeConstraint(buf, constraint); err != nil {
			return err
		}
	}

	for i, check := range table.CheckConstraints {
		if i+len(table.Indexes)+len(constraints) > 0 {
			if _, err := buf.WriteString(",\n"); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`  `); err != nil {
			return err
		}
		if err := writeCheckConstraint(buf, check); err != nil {
			return err
		}
	}

	for i, fk := range table.ForeignKeys {
		if i+len(table.Indexes)+len(constraints)+len(table.CheckConstraints) > 0 {
			if _, err := buf.WriteString(",\n"); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`  `); err != nil {
			return err
		}
		if err := writeForeignKey(buf, schema, fk); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n);\n\n"); err != nil {
		return err
	}

	for _, index := range table.Indexes {
		if index.IsConstraint {
			continue
		}
		if err := writeIndex(buf, table.Name, index); err != nil {
			return err
		}
	}

	return nil
}

func writeIndex(buf *strings.Builder, table string, index *storepb.IndexMetadata) error {
	if _, err := buf.WriteString(`CREATE`); err != nil {
		return err
	}

	switch index.Type {
	case "BITMAP", "FUNCTION-BASED BITMAP":
		if _, err := buf.WriteString(` BITMAP`); err != nil {
			return err
		}
	}

	if index.Unique {
		if _, err := buf.WriteString(` UNIQUE`); err != nil {
			return err
		}
	}

	if _, err := buf.WriteString(` INDEX "`); err != nil {
		return err
	}

	if _, err := buf.WriteString(index.Name); err != nil {
		return err
	}

	if _, err := buf.WriteString(`" ON "`); err != nil {
		return err
	}

	if _, err := buf.WriteString(table); err != nil {
		return err
	}

	if _, err := buf.WriteString(`" (`); err != nil {
		return err
	}

	if strings.Contains(index.Type, "FUNCTION-BASED") {
		for i, expression := range index.Expressions {
			if i != 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString(expression); err != nil {
				return err
			}

			if i < len(index.Descending) && index.Descending[i] {
				if _, err := buf.WriteString(` DESC`); err != nil {
					return err
				}
			} else {
				if _, err := buf.WriteString(` ASC`); err != nil {
					return err
				}
			}
		}
	} else {
		for i, column := range index.Expressions {
			if i != 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString(`"`); err != nil {
				return err
			}
			if _, err := buf.WriteString(column); err != nil {
				return err
			}
			if _, err := buf.WriteString(`"`); err != nil {
				return err
			}
			if i < len(index.Descending) && index.Descending[i] {
				if _, err := buf.WriteString(` DESC`); err != nil {
					return err
				}
			} else {
				if _, err := buf.WriteString(` ASC`); err != nil {
					return err
				}
			}
		}
	}

	_, err := buf.WriteString(`);\n\n`)
	return err
}

func writeForeignKey(buf *strings.Builder, schema string, fk *storepb.ForeignKeyMetadata) error {
	if _, err := buf.WriteString(`CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(fk.Name); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(` FOREIGN KEY (`); err != nil {
		return err
	}
	for i, column := range fk.Columns {
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`"`); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"`); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(`) REFERENCES "`); err != nil {
		return err
	}
	if fk.ReferencedSchema != schema {
		if _, err := buf.WriteString(fk.ReferencedSchema); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"."`); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(fk.ReferencedTable); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" (`); err != nil {
		return err
	}
	for i, column := range fk.ReferencedColumns {
		if i != 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`"`); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"`); err != nil {
			return err
		}
	}
	_, err := buf.WriteString(`)`)
	return err
}

func writeCheckConstraint(buf *strings.Builder, check *storepb.CheckConstraintMetadata) error {
	if _, err := buf.WriteString(`CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(check.Name); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(` CHECK (`); err != nil {
		return err
	}
	if _, err := buf.WriteString(check.Expression); err != nil {
		return err
	}
	_, err := buf.WriteString(`)`)
	return err
}

func writeConstraint(buf *strings.Builder, constraint *storepb.IndexMetadata) error {
	if _, err := buf.WriteString(`CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := buf.WriteString(constraint.Name); err != nil {
		return err
	}
	if _, err := buf.WriteString(`"`); err != nil {
		return err
	}

	switch {
	case constraint.Primary:
		if _, err := buf.WriteString(` PRIMARY KEY (`); err != nil {
			return err
		}
		for i, column := range constraint.Expressions {
			if i != 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString(`"`); err != nil {
				return err
			}
			if _, err := buf.WriteString(column); err != nil {
				return err
			}
			if _, err := buf.WriteString(`"`); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`)`); err != nil {
			return err
		}
	case !constraint.Primary && constraint.Unique:
		if _, err := buf.WriteString(` UNIQUE (`); err != nil {
			return err
		}
		for i, column := range constraint.Expressions {
			if i != 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString(`"`); err != nil {
				return err
			}
			if _, err := buf.WriteString(column); err != nil {
				return err
			}
			if _, err := buf.WriteString(`"`); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(`)`); err != nil {
			return err
		}
	}
	return nil
}

func writeColumn(buf *strings.Builder, column *storepb.ColumnMetadata) error {
	if _, err := buf.WriteString(`"`); err != nil {
		return err
	}
	if _, err := buf.WriteString(column.Name); err != nil {
		return err
	}
	if _, err := buf.WriteString(`" `); err != nil {
		return err
	}
	if _, err := buf.WriteString(column.Type); err != nil {
		return err
	}
	if column.Collation != "" {
		if _, err := buf.WriteString(` COLLATE "`); err != nil {
			return err
		}
		if _, err := buf.WriteString(column.Collation); err != nil {
			return err
		}
		if _, err := buf.WriteString(`"`); err != nil {
			return err
		}
	}
	if column.DefaultValue != nil {
		if _, err := buf.WriteString(` DEFAULT `); err != nil {
			return err
		}
		if column.DefaultOnNull {
			if _, err := buf.WriteString(`ON NULL `); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(column.GetDefaultExpression()); err != nil {
			return err
		}
	}
	if !column.Nullable {
		if _, err := buf.WriteString(` NOT NULL`); err != nil {
			return err
		}
	}
	return nil
}
