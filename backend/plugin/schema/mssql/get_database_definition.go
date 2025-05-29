package mssql

import (
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	defaultSchema = "dbo"
)

func init() {
	schema.RegisterGetDatabaseDefinition(storepb.Engine_MSSQL, GetDatabaseDefinition)
	schema.RegisterGetTableDefinition(storepb.Engine_MSSQL, GetTableDefinition)
}

func GetDatabaseDefinition(_ schema.GetDefinitionContext, to *storepb.DatabaseSchemaMetadata) (string, error) {
	if to == nil {
		return "", nil
	}

	var buf strings.Builder
	for _, schema := range to.Schemas {
		writeSchema(&buf, schema)
	}

	return buf.String(), nil
}

func GetTableDefinition(schemaName string, table *storepb.TableMetadata, _ []*storepb.SequenceMetadata) (string, error) {
	var buf strings.Builder
	writeTable(&buf, schemaName, table)
	return buf.String(), nil
}

func writeSchema(out *strings.Builder, schema *storepb.SchemaMetadata) {
	if schema.Name != defaultSchema {
		_, _ = fmt.Fprintf(out, "CREATE SCHEMA [%s];\nGO\n\n", schema.Name)
	}

	for _, table := range schema.Tables {
		writeTable(out, schema.Name, table)
	}
}

func writeTable(out *strings.Builder, schemaName string, table *storepb.TableMetadata) {
	_, _ = fmt.Fprintf(out, "CREATE TABLE [%s].[%s] (\n", schemaName, table.Name)
	for i, column := range table.Columns {
		if i != 0 {
			_, _ = out.WriteString(",\n")
		}
		writeColumn(out, column)
	}

	for _, key := range table.Indexes {
		if !key.IsConstraint {
			continue
		}

		_, _ = out.WriteString(",\n")
		writeKey(out, key)
	}

	for _, fk := range table.ForeignKeys {
		_, _ = out.WriteString(",\n")
		writeForeignKey(out, fk)
	}

	for _, check := range table.CheckConstraints {
		_, _ = out.WriteString(",\n")
		writeCheck(out, check)
	}
	_, _ = fmt.Fprintf(out, "\n);\n\n")

	for _, index := range table.Indexes {
		if index.IsConstraint {
			continue
		}
		writeIndex(out, schemaName, table.Name, index)
	}
}

func writeClusteredColumnStoreIndex(out *strings.Builder, schemaName string, tableName string, index *storepb.IndexMetadata) {
	_, _ = fmt.Fprintf(out, "CREATE CLUSTERED COLUMNSTORE INDEX [%s] ON [%s].[%s];\n\n", index.Name, schemaName, tableName)
}

func writeNonClusteredColumnStoreIndex(out *strings.Builder, schemaName string, tableName string, index *storepb.IndexMetadata) {
	_, _ = fmt.Fprintf(out, "CREATE NONCLUSTERED COLUMNSTORE INDEX [%s] ON [%s].[%s] (\n", index.Name, schemaName, tableName)
	for i, column := range index.Expressions {
		if i != 0 {
			_, _ = out.WriteString(",\n")
		}
		_, _ = fmt.Fprintf(out, "    [%s]", column)
	}
	_, _ = out.WriteString("\n);\n\n")
}

func writeNormalIndex(out *strings.Builder, schemaName string, tableName string, index *storepb.IndexMetadata) {
	_, _ = out.WriteString("CREATE")
	if index.Type != "" {
		_, _ = fmt.Fprintf(out, " %s", index.Type)
	}
	_, _ = fmt.Fprintf(out, " INDEX [%s] ON\n[%s].[%s] (\n", index.Name, schemaName, tableName)
	for i, column := range index.Expressions {
		if i != 0 {
			_, _ = out.WriteString(",\n")
		}
		_, _ = fmt.Fprintf(out, "    [%s]", column)
		if index.Descending[i] {
			_, _ = out.WriteString(" DESC")
		} else {
			_, _ = out.WriteString(" ASC")
		}
	}
	_, _ = out.WriteString("\n);\n\n")
}

func writeIndex(out *strings.Builder, schemaName string, tableName string, index *storepb.IndexMetadata) {
	switch strings.ToUpper(index.Type) {
	case "CLUSTERED COLUMNSTORE":
		writeClusteredColumnStoreIndex(out, schemaName, tableName, index)
	case "NONCLUSTERED COLUMNSTORE":
		writeNonClusteredColumnStoreIndex(out, schemaName, tableName, index)
	default:
		writeNormalIndex(out, schemaName, tableName, index)
	}
}

func writeCheck(out *strings.Builder, check *storepb.CheckConstraintMetadata) {
	_, _ = fmt.Fprintf(out, "    CONSTRAINT [%s] CHECK %s", check.Name, check.Expression)
}

func writeForeignKey(out *strings.Builder, fk *storepb.ForeignKeyMetadata) {
	_, _ = fmt.Fprintf(out, "    CONSTRAINT [%s] FOREIGN KEY (", fk.Name)
	for i, column := range fk.Columns {
		if i != 0 {
			_, _ = out.WriteString(", ")
		}
		_, _ = fmt.Fprintf(out, "[%s]", column)
	}
	_, _ = fmt.Fprintf(out, ") REFERENCES [%s].[%s] (", fk.ReferencedSchema, fk.ReferencedTable)
	for i, column := range fk.ReferencedColumns {
		if i != 0 {
			_, _ = out.WriteString(", ")
		}
		_, _ = fmt.Fprintf(out, "[%s]", column)
	}
	_, _ = out.WriteString(")")
	if fk.OnDelete != "" {
		_, _ = fmt.Fprintf(out, " ON DELETE %s", fk.OnDelete)
	}
	if fk.OnUpdate != "" {
		_, _ = fmt.Fprintf(out, " ON UPDATE %s", fk.OnUpdate)
	}
}

func writeKey(out *strings.Builder, key *storepb.IndexMetadata) {
	_, _ = fmt.Fprintf(out, "    CONSTRAINT [%s]", key.Name)
	if key.Primary {
		_, _ = out.WriteString(" PRIMARY KEY")
	} else if key.Unique {
		_, _ = out.WriteString(" UNIQUE")
	}

	if key.Type != "" {
		_, _ = fmt.Fprintf(out, " %s", key.Type)
	}
	_, _ = out.WriteString(" (")
	for i, column := range key.Expressions {
		if i != 0 {
			_, _ = out.WriteString(", ")
		}
		_, _ = fmt.Fprintf(out, "[%s]", column)
		if key.Descending[i] {
			_, _ = out.WriteString(" DESC")
		} else {
			_, _ = out.WriteString(" ASC")
		}
	}
	_, _ = out.WriteString(")")
}

func writeColumn(out *strings.Builder, column *storepb.ColumnMetadata) {
	_, _ = fmt.Fprintf(out, "    [%s] %s", column.Name, column.Type)
	if column.IsIdentity {
		_, _ = fmt.Fprintf(out, " IDENTITY(%d,%d)", column.IdentitySeed, column.IdentityIncrement)
	}
	if column.Collation != "" {
		_, _ = fmt.Fprintf(out, " COLLATE %s", column.Collation)
	}
	if column.GetDefaultExpression() != "" {
		_, _ = fmt.Fprintf(out, " DEFAULT %s", column.GetDefaultExpression())
	}
	if !column.Nullable {
		_, _ = out.WriteString(" NOT NULL")
	}
}
