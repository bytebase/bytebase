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
	err := applyMultiStatements(statement, func(text string, start, end int) error {
		startLine, startColumn := base.CalculateLineAndColumn(statement, start)
		endLine, endColumn := base.CalculateLineAndColumn(statement, end)

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
				Start: int32(start),
				End:   int32(end),
			},
			Empty: isEmptySQL(text),
		})
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

// applyMultiStatements splits the statement by semicolons and invokes f for
// each sub-statement with the text slice from the original statement and its
// [start, end) byte offsets. The text for each statement includes any leading
// whitespace from the end of the previous statement, matching the original
// behavior needed for position tracking.
func applyMultiStatements(statement string, f func(text string, start, end int) error) error {
	reader := bufio.NewReader(strings.NewReader(statement))
	delimiter := false
	comment := false
	done := false
	hasContent := false
	// byteOffset tracks our read position in the original statement.
	byteOffset := 0
	// prevEnd tracks where the last emitted statement ended; the next
	// statement's text will start here (to include inter-statement whitespace).
	prevEnd := 0
	// contentEnd is the byte offset just past the last content character of
	// the statement being accumulated (excludes trailing whitespace/newlines).
	contentEnd := 0
	for !done {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				return err
			}
			done = true
		}
		lineLen := len(line)
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
		case comment && !strings.Contains(line, "*/"):
			// skip line in comment mode
		case comment && strings.Contains(line, "*/"):
			if !strings.HasSuffix(line, "*/") {
				return errors.Errorf("`*/` must be the end of the line; new statement should start as a new line")
			}
			comment = false
		case !hasContent && line == "":
			// skip leading blank lines
		case strings.HasPrefix(line, "--"):
			// skip comment lines
		case line == "DELIMITER ;;":
			delimiter = true
		case line == "DELIMITER ;" && delimiter:
			delimiter = false
			execute = true
		case strings.HasSuffix(line, ";"):
			hasContent = true
			contentEnd = byteOffset + len(line)
			if !delimiter {
				execute = true
			}
		default:
			if line != "" {
				hasContent = true
				contentEnd = byteOffset + len(line)
			}
		}
		if execute && hasContent {
			text := statement[prevEnd:contentEnd]
			if err := f(text, prevEnd, contentEnd); err != nil {
				return errors.Wrapf(err, "execute query %q failed", text)
			}
			hasContent = false
			prevEnd = contentEnd
		}
		byteOffset += lineLen
	}
	// Apply the remaining content.
	if hasContent {
		text := statement[prevEnd:contentEnd]
		if err := f(text, prevEnd, contentEnd); err != nil {
			return errors.Wrapf(err, "execute query %q failed", text)
		}
	}
	return nil
}
