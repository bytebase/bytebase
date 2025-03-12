package mysql

import (
	"fmt"
	"strings"

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
