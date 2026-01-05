package standard

import (
	"bufio"
	"io"
	"strings"
	"unicode"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_CLICKHOUSE, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_SQLITE, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_HIVE, SplitSQL)
	base.RegisterSplitterFunc(storepb.Engine_DATABRICKS, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	var list []base.Statement
	byteOffset := 0
	err := applyMultiStatements(strings.NewReader(statement), func(sql string) error {
		// sql may have trailing newline added by applyMultiStatements
		// Strip trailing whitespace for searching in the original statement
		sqlTrimmed := strings.TrimRight(sql, "\n\r\t ")

		// Find where the SQL content exists in the remaining statement
		sqlPos := strings.Index(statement[byteOffset:], sqlTrimmed)
		if sqlPos == -1 {
			sqlPos = 0
		}

		// startPos includes leading whitespace from where previous statement ended
		startPos := byteOffset
		// endPos is where the SQL content ends
		endPos := byteOffset + sqlPos + len(sqlTrimmed)

		// The actual text includes leading whitespace + SQL content
		text := statement[startPos:endPos]

		// Calculate line and column for Start position
		startLine, startColumn := base.CalculateLineAndColumn(statement, startPos)
		// Calculate line and column for End position
		endLine, endColumn := base.CalculateLineAndColumn(statement, endPos)

		list = append(list, base.Statement{
			Text: text,
			Start: &storepb.Position{
				Line:   int32(startLine + 1),   // 1-based
				Column: int32(startColumn + 1), // 1-based per proto spec
			},
			End: &storepb.Position{
				Line:   int32(endLine + 1),   // 1-based
				Column: int32(endColumn + 1), // 1-based per proto spec
			},
			Range: &storepb.Range{
				Start: int32(startPos),
				End:   int32(endPos),
			},
			Empty: isEmptySQL(text),
		})
		byteOffset = endPos
		return nil
	})
	return list, err
}

// isEmptySQL checks if the SQL contains only whitespace and comments.
func isEmptySQL(sql string) bool {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return true
	}
	// Check if it's only comments
	if strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "/*") {
		// Simple heuristic: if after removing comment markers there's no SQL
		lines := strings.Split(trimmed, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "--") {
				continue
			}
			if strings.HasPrefix(line, "/*") && strings.HasSuffix(line, "*/") {
				continue
			}
			// Found non-comment content
			return false
		}
		return true
	}
	return false
}

// applyMultiStatements will apply the split statements from scanner.
// This function only used for SQLite, snowflake.
// For MySQL and PostgreSQL, use parser.SplitSQL.
// Copy from plugin/db/util/driverutil.go.
func applyMultiStatements(sc io.Reader, f func(string) error) error {
	// TODO(rebelice): use parser/tokenizer to split SQL statements.
	reader := bufio.NewReader(sc)
	var sb strings.Builder
	delimiter := false
	comment := false
	done := false
	for !done {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				done = true
			} else {
				return err
			}
		}
		line = strings.TrimRightFunc(line, unicode.IsSpace)

		execute := false
		switch {
		case strings.HasPrefix(line, "/*"):
			if strings.Contains(line, "*/") {
				if !strings.HasSuffix(line, "*/") {
					return errors.Errorf("`*/` must be the end of the line; new statement should start as a new line")
				}
			} else {
				comment = true
			}
			continue
		case comment && !strings.Contains(line, "*/"):
			// Skip the line when in comment mode.
			continue
		case comment && strings.Contains(line, "*/"):
			if !strings.HasSuffix(line, "*/") {
				return errors.Errorf("`*/` must be the end of the line; new statement should start as a new line")
			}
			comment = false
			continue
		case sb.Len() == 0 && line == "":
			continue
		case strings.HasPrefix(line, "--"):
			continue
		case line == "DELIMITER ;;":
			delimiter = true
			continue
		case line == "DELIMITER ;" && delimiter:
			delimiter = false
			execute = true
		case strings.HasSuffix(line, ";"):
			_, _ = sb.WriteString(line)
			_, _ = sb.WriteString("\n")
			if !delimiter {
				execute = true
			}
		default:
			_, _ = sb.WriteString(line)
			_, _ = sb.WriteString("\n")
			continue
		}
		if execute {
			s := sb.String()
			// Don't trim - include leading whitespace in Text for position consistency
			if s != "" {
				if err := f(s); err != nil {
					return errors.Wrapf(err, "execute query %q failed", s)
				}
			}
			sb.Reset()
		}
	}
	// Apply the remaining content.
	s := sb.String()
	// Don't trim - include leading whitespace in Text for position consistency
	if s != "" {
		if err := f(s); err != nil {
			return errors.Wrapf(err, "execute query %q failed", s)
		}
	}

	return nil
}
