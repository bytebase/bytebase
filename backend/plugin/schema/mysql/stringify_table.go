package mysql

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	mysqlTypeBlob       = "blob"
	mysqlTypeTinyBob    = "tinyblob"
	mysqlTypeMediumBlob = "mediumblob"
	mysqlTypeLongBlob   = "longblob"
	mysqlTypeJSON       = "json"
	mysqlTypeGeometry   = "geometry"

	mysqlIndexFullText = "FULLTEXT"
	mysqlIndexSpatial  = "SPATIAL"

	mysqlNoAction = "NO ACTION"
)

func init() {
	schema.RegisterStringifyTable(storepb.Engine_MYSQL, StringifyTable)
}

func StringifyTable(metadata *storepb.TableMetadata) (string, error) {
	var buf strings.Builder

	if _, err := fmt.Fprintf(&buf, "CREATE TABLE `%s` (\n", metadata.Name); err != nil {
		return "", err
	}

	for i, column := range metadata.Columns {
		if i != 0 {
			if _, err := fmt.Fprintf(&buf, ",\n"); err != nil {
				return "", err
			}
		}
		if err := printColumnClause(&buf, column); err != nil {
			return "", err
		}
	}

	if err := printPrimaryKeyClause(&buf, metadata.Indexes); err != nil {
		return "", err
	}

	for _, index := range metadata.Indexes {
		if index.Primary {
			continue
		}
		if err := printIndexClause(&buf, index); err != nil {
			return "", err
		}
	}

	for _, fk := range metadata.ForeignKeys {
		if err := printForeignKeyClause(&buf, fk); err != nil {
			return "", err
		}
	}

	for _, check := range metadata.CheckConstraints {
		if err := printCheckClause(&buf, check); err != nil {
			return "", err
		}
	}

	if _, err := fmt.Fprintf(&buf, "\n) ENGINE=%s", metadata.Engine); err != nil {
		return "", err
	}

	if metadata.Charset != "" {
		if _, err := fmt.Fprintf(&buf, " DEFAULT CHARSET=%s", metadata.Charset); err != nil {
			return "", err
		}
	}

	if metadata.Collation != "" {
		if _, err := fmt.Fprintf(&buf, " COLLATE=%s", metadata.Collation); err != nil {
			return "", err
		}
	}

	if metadata.Comment != "" {
		if _, err := fmt.Fprintf(&buf, " COMMENT='%s'", metadata.Comment); err != nil {
			return "", err
		}
	}

	if len(metadata.Partitions) > 0 {
		if err := printPartitionClause(&buf, metadata.Partitions); err != nil {
			return "", err
		}
	}

	if _, err := fmt.Fprintf(&buf, ";\n"); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Copy the logic from backend/plugin/schema/mysql/state.go.
func printPartitionClause(buf *strings.Builder, partitions []*storepb.TablePartitionMetadata) error {
	if len(partitions) == 0 {
		return nil
	}
	vsc := getVersionSpecificComment(partitions)
	curComment := vsc
	if _, err := fmt.Fprintf(buf, "%s PARTITION BY ", curComment); err != nil {
		return err
	}
	switch partitions[0].Type {
	case storepb.TablePartitionMetadata_RANGE:
		if _, err := fmt.Fprintf(buf, "RANGE (%s)", partitions[0].Expression); err != nil {
			return err
		}
	case storepb.TablePartitionMetadata_RANGE_COLUMNS:
		if _, err := fmt.Fprintf(buf, "RANGE COLUMNS (%s)", partitions[0].Expression); err != nil {
			return err
		}
	case storepb.TablePartitionMetadata_LIST:
		if _, err := fmt.Fprintf(buf, "LIST (%s)", partitions[0].Expression); err != nil {
			return err
		}
	case storepb.TablePartitionMetadata_LIST_COLUMNS:
		if _, err := fmt.Fprintf(buf, "LIST COLUMNS (%s)", partitions[0].Expression); err != nil {
			return err
		}
	case storepb.TablePartitionMetadata_HASH:
		if _, err := fmt.Fprintf(buf, "HASH (%s)", partitions[0].Expression); err != nil {
			return err
		}
	case storepb.TablePartitionMetadata_KEY:
		if _, err := fmt.Fprintf(buf, "KEY (%s)", partitions[0].Expression); err != nil {
			return err
		}
	case storepb.TablePartitionMetadata_LINEAR_HASH:
		if _, err := fmt.Fprintf(buf, "LINEAR HASH (%s)", partitions[0].Expression); err != nil {
			return err
		}
	case storepb.TablePartitionMetadata_LINEAR_KEY:
		if _, err := fmt.Fprintf(buf, "LINEAR KEY (%s)", partitions[0].Expression); err != nil {
			return err
		}
	default:
		return errors.Errorf("unknown partition type: %v", partitions[0].Type)
	}

	useDefault := int64(0)
	if partitions[0].UseDefault != "" {
		var err error
		useDefault, err = strconv.ParseInt(partitions[0].UseDefault, 10, 64)
		if err != nil {
			return err
		}
	}
	if useDefault != 0 {
		if _, err := fmt.Fprintf(buf, "\nPARTITIONS %d", useDefault); err != nil {
			return err
		}
	}

	if len(partitions[0].Subpartitions) > 0 {
		if _, err := fmt.Fprintf(buf, "\nSUBPARTITION BY "); err != nil {
			return err
		}
		switch partitions[0].Subpartitions[0].Type {
		case storepb.TablePartitionMetadata_HASH:
			if _, err := fmt.Fprintf(buf, "HASH (%s)", partitions[0].Subpartitions[0].Expression); err != nil {
				return err
			}
		case storepb.TablePartitionMetadata_LINEAR_HASH:
			if _, err := fmt.Fprintf(buf, "LINEAR HASH (%s)", partitions[0].Subpartitions[0].Expression); err != nil {
				return err
			}
		case storepb.TablePartitionMetadata_KEY:
			if _, err := fmt.Fprintf(buf, "KEY (%s)", partitions[0].Subpartitions[0].Expression); err != nil {
				return err
			}
		case storepb.TablePartitionMetadata_LINEAR_KEY:
			if _, err := fmt.Fprintf(buf, "LINEAR KEY (%s)", partitions[0].Subpartitions[0].Expression); err != nil {
				return err
			}
		default:
			return errors.Errorf("invalid subpartition type: %v", partitions[0].Subpartitions[0].Type)
		}
	}

	subUseDefault := 0
	if len(partitions[0].Subpartitions) > 0 && partitions[0].Subpartitions[0].UseDefault != "" {
		var err error
		subUseDefault, err = strconv.Atoi(partitions[0].Subpartitions[0].UseDefault)
		if err != nil {
			return err
		}
	}

	if subUseDefault != 0 {
		if _, err := fmt.Fprintf(buf, "\nSUBPARTITIONS %d", subUseDefault); err != nil {
			return err
		}
	}

	if useDefault == 0 {
		if _, err := fmt.Fprintf(buf, "\n("); err != nil {
			return err
		}
		preposition, err := getPrepositionByType(partitions[0].Type)
		if err != nil {
			return err
		}
		for i, partition := range partitions {
			if i != 0 {
				if _, err := fmt.Fprintf(buf, ",\n "); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(buf, "PARTITION %s", partition.Name); err != nil {
				return err
			}
			if preposition != "" {
				if partition.Value != "MAXVALUE" {
					if _, err := fmt.Fprintf(buf, " VALUES %s (%s)", preposition, partition.Value); err != nil {
						return err
					}
				} else {
					if _, err := fmt.Fprintf(buf, " VALUES %s %s", preposition, partition.Value); err != nil {
						return err
					}
				}
			}

			if subUseDefault == 0 && len(partition.Subpartitions) > 0 {
				if _, err := fmt.Fprintf(buf, "\n ("); err != nil {
					return err
				}
				for j, subPartition := range partition.Subpartitions {
					if _, err := fmt.Fprintf(buf, "SUBPARTITION %s", subPartition.Name); err != nil {
						return err
					}
					if err := writePartitionOptions(buf); err != nil {
						return err
					}
					if j == len(partition.Subpartitions)-1 {
						if _, err := fmt.Fprintf(buf, ")"); err != nil {
							return err
						}
					} else {
						if _, err := fmt.Fprintf(buf, ",\n  "); err != nil {
							return err
						}
					}
				}
			} else {
				if err := writePartitionOptions(buf); err != nil {
					return err
				}
			}

			if i == len(partitions)-1 {
				if _, err := fmt.Fprintf(buf, ")"); err != nil {
					return err
				}
			}
		}
	}

	if _, err := fmt.Fprintf(buf, " */"); err != nil {
		return err
	}

	return nil
}

func getVersionSpecificComment(partitions []*storepb.TablePartitionMetadata) string {
	if len(partitions) == 0 {
		return ""
	}
	partition := partitions[0]
	if partition.Type == storepb.TablePartitionMetadata_RANGE_COLUMNS || partition.Type == storepb.TablePartitionMetadata_LIST_COLUMNS {
		// MySQL introduce columns partitioning in 5.5+
		return "\n/*!50500"
	}
	return "\n/*!50100"
}

func printCheckClause(buf *strings.Builder, check *storepb.CheckConstraintMetadata) error {
	if _, err := fmt.Fprintf(buf, ",\n  CONSTRAINT `%s` CHECK %s", check.Name, check.Expression); err != nil {
		return err
	}
	return nil
}

func printForeignKeyClause(buf *strings.Builder, fk *storepb.ForeignKeyMetadata) error {
	if _, err := fmt.Fprintf(buf, ",\n  CONSTRAINT `%s` FOREIGN KEY (", fk.Name); err != nil {
		return err
	}

	for i, column := range fk.Columns {
		if i != 0 {
			if _, err := fmt.Fprintf(buf, ", "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(buf, "`%s`", column); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(buf, ") REFERENCES `%s` (", fk.ReferencedTable); err != nil {
		return err
	}

	for i, column := range fk.ReferencedColumns {
		if i != 0 {
			if _, err := fmt.Fprintf(buf, ", "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(buf, "`%s`", column); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(buf, ")"); err != nil {
		return err
	}

	if fk.OnDelete != "" && !strings.EqualFold(fk.OnDelete, mysqlNoAction) {
		if _, err := fmt.Fprintf(buf, " ON DELETE %s", fk.OnDelete); err != nil {
			return err
		}
	}

	if fk.OnUpdate != "" && !strings.EqualFold(fk.OnUpdate, mysqlNoAction) {
		if _, err := fmt.Fprintf(buf, " ON UPDATE %s", fk.OnUpdate); err != nil {
			return err
		}
	}

	return nil
}

func printIndexClause(buf *strings.Builder, index *storepb.IndexMetadata) error {
	if index.Primary {
		return nil
	}

	if _, err := fmt.Fprintf(buf, ",\n  "); err != nil {
		return err
	}

	if index.Unique {
		if _, err := fmt.Fprintf(buf, "UNIQUE "); err != nil {
			return err
		}
	} else if strings.EqualFold(index.Type, mysqlIndexFullText) {
		if _, err := fmt.Fprintf(buf, "FULLTEXT "); err != nil {
			return err
		}
	} else if strings.EqualFold(index.Type, mysqlIndexSpatial) {
		if _, err := fmt.Fprintf(buf, "SPATIAL "); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(buf, "KEY `%s` (", index.Name); err != nil {
		return err
	}

	for i, expr := range index.Expressions {
		if i != 0 {
			if _, err := fmt.Fprintf(buf, ", "); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(buf, "%s", expr); err != nil {
			return err
		}

		if len(index.Descending) > i && index.Descending[i] {
			if _, err := fmt.Fprintf(buf, " DESC"); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintf(buf, ")"); err != nil {
		return err
	}

	return nil
}

func printPrimaryKeyClause(buf *strings.Builder, indexes []*storepb.IndexMetadata) error {
	for _, index := range indexes {
		if index.Primary {
			if _, err := fmt.Fprintf(buf, ",\n  PRIMARY KEY ("); err != nil {
				return err
			}
			for i, column := range index.Expressions {
				if i != 0 {
					if _, err := fmt.Fprintf(buf, ", "); err != nil {
						return err
					}
				}
				if _, err := fmt.Fprintf(buf, "`%s`", column); err != nil {
					return err
				}
				if len(index.Descending) > i && index.Descending[i] {
					if _, err := fmt.Fprintf(buf, " DESC"); err != nil {
						return err
					}
				}
			}
			if _, err := fmt.Fprintf(buf, ")"); err != nil {
				return err
			}
			return nil
		}
	}

	return nil
}

func isAutoIncrement(column *storepb.ColumnMetadata) bool {
	return strings.EqualFold(column.GetDefaultExpression(), autoIncrementSymbol)
}

func printColumnClause(buf *strings.Builder, column *storepb.ColumnMetadata) error {
	if _, err := fmt.Fprintf(buf, "  `%s` %s", column.Name, column.Type); err != nil {
		return err
	}

	if column.CharacterSet != "" {
		if _, err := fmt.Fprintf(buf, " CHARACTER SET %s", column.CharacterSet); err != nil {
			return err
		}
	}

	if column.Collation != "" {
		if _, err := fmt.Fprintf(buf, " COLLATE %s", column.Collation); err != nil {
			return err
		}
	}

	if column.Generation != nil && column.Generation.Expression != "" {
		if _, err := fmt.Fprintf(buf, " GENERATED ALWAYS AS (%s) ", column.Generation.Expression); err != nil {
			return err
		}
		switch column.Generation.Type {
		case storepb.GenerationMetadata_TYPE_STORED:
			if _, err := fmt.Fprintf(buf, "STORED"); err != nil {
				return err
			}
		case storepb.GenerationMetadata_TYPE_VIRTUAL:
			if _, err := fmt.Fprintf(buf, "VIRTUAL"); err != nil {
				return err
			}
		}
	}

	if column.Nullable {
		if _, err := fmt.Fprintf(buf, " NULL"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(buf, " NOT NULL"); err != nil {
			return err
		}
	}

	if err := printDefaultClause(buf, column); err != nil {
		return err
	}

	// Handle auto_increment.
	if isAutoIncrement(column) {
		if _, err := fmt.Fprintf(buf, " %s", autoIncrementSymbol); err != nil {
			return err
		}
	}

	if column.OnUpdate != "" {
		if _, err := buf.WriteString(fmt.Sprintf(" ON UPDATE %s", column.OnUpdate)); err != nil {
			return err
		}
	}
	if column.Comment != "" {
		if _, err := fmt.Fprintf(buf, " COMMENT '%s'", column.Comment); err != nil {
			return err
		}
	}
	return nil
}

func printDefaultClause(buf *strings.Builder, column *storepb.ColumnMetadata) error {
	if column.DefaultValue == nil {
		return nil
	}

	if column.GetDefaultNull() {
		if !column.Nullable || !typeSupportsDefaultValue(column.Type) {
			// If the column is not nullable, then the default value should not be null.
			// For this case, we should not print the default clause.
			return nil
		}
		if column.Generation != nil && column.Generation.Expression != "" {
			return nil
		}
		if _, err := fmt.Fprintf(buf, " DEFAULT NULL"); err != nil {
			return err
		}
		return nil
	}

	if column.GetDefaultExpression() != "" {
		if isAutoIncrement(column) {
			// If the default value is auto_increment, then we should not print the default clause.
			// We'll handle this in the following AUTO_INCREMENT clause.
			return nil
		}
		if _, err := fmt.Fprintf(buf, " DEFAULT %s", column.GetDefaultExpression()); err != nil {
			return err
		}
		return nil
	}

	if column.GetDefault() != nil {
		if _, err := fmt.Fprintf(buf, " DEFAULT '%s'", column.GetDefault().GetValue()); err != nil {
			return err
		}
	}

	if column.OnUpdate != "" {
		if _, err := fmt.Fprintf(buf, " ON UPDATE %s", column.OnUpdate); err != nil {
			return err
		}
	}

	return nil
}

func typeSupportsDefaultValue(tp string) bool {
	switch strings.ToLower(tp) {
	case mysqlTypeBlob, mysqlTypeTinyBob, mysqlTypeMediumBlob, mysqlTypeLongBlob, mysqlTypeJSON, mysqlTypeGeometry:
		return false
	default:
		return true
	}
}
