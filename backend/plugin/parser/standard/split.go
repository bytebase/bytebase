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
		// Find the start position of this SQL in the original statement
		startPos := strings.Index(statement[byteOffset:], strings.TrimLeft(sql, "\n\t "))
		if startPos != -1 {
			startPos += byteOffset
		} else {
			startPos = byteOffset
		}
		endPos := startPos + len(sql)

		list = append(list, base.Statement{
			Text: sql,
			Range: &storepb.Range{
				Start: int32(startPos),
				End:   int32(endPos),
			},
			End:   &storepb.Position{Line: 0, Column: 0},
			Empty: false,
		})
		byteOffset = endPos
		return nil
	})
	return list, err
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
			s = strings.Trim(s, "\n\t ")
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
	s = strings.Trim(s, "\n\t ")
	if s != "" {
		if err := f(s); err != nil {
			return errors.Wrapf(err, "execute query %q failed", s)
		}
	}

	return nil
}
