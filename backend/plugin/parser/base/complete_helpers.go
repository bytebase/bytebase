package base

import (
	"slices"
	"strings"
	"unicode/utf8"
)

// CaretToByteOffset converts a (1-based line, 0-based UTF-16 code-unit column) caret
// position into a byte offset within statement.
func CaretToByteOffset(statement string, caretLine, caretColumn int) int {
	line := 1
	col := 0
	for i := 0; i < len(statement); {
		if line == caretLine && col >= caretColumn {
			return i
		}
		r, size := utf8.DecodeRuneInString(statement[i:])
		if r == '\n' {
			if line == caretLine {
				return i
			}
			line++
			col = 0
		} else if r <= 0xFFFF {
			col++
		} else {
			col += 2
		}
		i += size
	}
	return len(statement)
}

// StatementAtCaret isolates the SQL slice around the caret from pre-split statements.
// When includeFollowing is false, only the single statement containing the caret is returned;
// otherwise that statement and all following statements are concatenated.
func StatementAtCaret(list []Statement, statement string, caretLine, caretOffset int, includeFollowing bool) (string, int, int) {
	if len(FilterEmptyStatements(list)) <= 1 {
		return statement, caretLine, caretOffset
	}

	caretLineZeroBased := caretLine - 1
	start := 0
	newCaretLine, newCaretOffset := caretLine, caretOffset
	for i, sql := range list {
		sqlEndLine := int(sql.End.GetLine()) - 1
		sqlEndColumn := int(sql.End.GetColumn())
		if sqlEndLine > caretLineZeroBased || (sqlEndLine == caretLineZeroBased && sqlEndColumn >= caretOffset) {
			start = i
			if i == 0 {
				break
			}
			previousEndLine := int(list[i-1].End.GetLine()) - 1
			previousEndColumn := int(list[i-1].End.GetColumn())
			newCaretLine = caretLineZeroBased - previousEndLine + 1
			if caretLineZeroBased == previousEndLine {
				newCaretOffset = caretOffset - previousEndColumn + 1
			}
			break
		}
	}

	if !includeFollowing {
		return list[start].Text, newCaretLine, newCaretOffset
	}
	var buf strings.Builder
	for i := start; i < len(list); i++ {
		if _, err := buf.WriteString(list[i].Text); err != nil {
			return statement, caretLine, caretOffset
		}
	}
	return buf.String(), newCaretLine, newCaretOffset
}

// DeduplicateCandidates removes duplicate candidates keyed by Candidate.String()
// while preserving first occurrence, then sorts with compare.
func DeduplicateCandidates[T any](items []T, convert func(T) (Candidate, bool), compare func(a, b Candidate) int) []Candidate {
	result := make([]Candidate, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		c, ok := convert(item)
		if !ok {
			continue
		}
		key := c.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, c)
	}
	if compare != nil {
		slices.SortFunc(result, compare)
	}
	return result
}

// CompareCandidateByTypeText compares candidates by Type string then Text.
func CompareCandidateByTypeText(a, b Candidate) int {
	if a.Type != b.Type {
		return strings.Compare(string(a.Type), string(b.Type))
	}
	return strings.Compare(a.Text, b.Text)
}
