package mysql

import (
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
)

// MariaDB system versioning is written as plain MariaDB syntax, not inside a
// /*M! */ comment: the omni MySQL loader that parses schema text for diffs would
// skip the comment and load the table unversioned, so a diff would silently
// recreate it without history. It rejects the plain syntax instead.

// systemVersioningOption is what the MariaDB sync appends to
// TableMetadata.CreateOptions for a system-versioned table. information_schema
// reports versioning only as a TABLE_TYPE, which TableMetadata has no field for.
const systemVersioningOption = "WITH SYSTEM VERSIONING"

func isSystemVersioned(table *metadatapb.TableMetadata) bool {
	return strings.Contains(table.CreateOptions, systemVersioningOption)
}

func writeSystemVersioningOption(buf *strings.Builder, table *metadatapb.TableMetadata) {
	if isSystemVersioned(table) {
		_, _ = buf.WriteString(" " + systemVersioningOption)
	}
}

// isSystemTimeColumn reports a MariaDB column declared GENERATED ALWAYS AS ROW START
// or ROW END, which information_schema reports as a stored generated column with that
// text as its expression.
func isSystemTimeColumn(column *metadatapb.ColumnMetadata) bool {
	generation := column.GetGeneration()
	if generation == nil || generation.Type != metadatapb.GenerationMetadata_TYPE_STORED {
		return false
	}
	return generation.Expression == "ROW START" || generation.Expression == "ROW END"
}

// writeSystemTimeColumnAttribute writes the attribute without the parentheses and
// storage kind of an ordinary generated column, which MariaDB rejects here.
func writeSystemTimeColumnAttribute(buf *strings.Builder, column *metadatapb.ColumnMetadata) {
	_, _ = buf.WriteString(" GENERATED ALWAYS AS " + column.Generation.Expression)
}

// writeSystemTimePeriod writes the PERIOD FOR SYSTEM_TIME table element that MariaDB
// requires alongside explicit ROW START and ROW END columns.
func writeSystemTimePeriod(buf *strings.Builder, table *metadatapb.TableMetadata) {
	var start, end string
	for _, column := range table.Columns {
		if !isSystemTimeColumn(column) {
			continue
		}
		if column.Generation.Expression == "ROW START" {
			start = column.Name
		} else {
			end = column.Name
		}
	}
	if start == "" || end == "" {
		return
	}
	_, _ = buf.WriteString(",\n  PERIOD FOR SYSTEM_TIME (`" + start + "`, `" + end + "`)")
}
