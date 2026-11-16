package bigquery

import (
	"strings"

	"github.com/bytebase/omni/v1review"
)

// statementFindingLine is the 1-based line within the statement text that
// holds a statement finding's start: the first line when the finding anchors
// nothing, and the last line when its start lies past the text.
func statementFindingLine(text string, f v1review.Finding) int {
	offset := min(f.Start, len(text))
	return strings.Count(text[:offset], "\n") + 1
}
