package doris

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestQuerySpanFailClosed pins the BYT-10085 contract at the bytebase layer:
// a statement the parser cannot fully consume must FAIL the span extraction,
// not return an empty span. Before strict parsing, trailing junk silently
// truncated to the valid prefix — masking and table-level ACL checks then
// saw nothing to protect. The positive cases cover the Doris-specific
// GroupedQuery shapes (repeated trailing clause groups) that the strict
// parser newly produces.
func TestQuerySpanFailClosed(t *testing.T) {
	a := require.New(t)
	q := newQuerySpanExtractor("db1", base.GetQuerySpanContext{}, false)
	secret := base.ColumnResource{Database: "db1", Table: "secret"}

	_, err := q.getQuerySpan(context.Background(), "SELECT a FROM secret xx yy zz")
	a.Error(err, "trailing junk must fail the span, not truncate silently")

	span, err := q.getQuerySpan(context.Background(), "(SELECT a FROM secret) LIMIT 1")
	a.NoError(err)
	a.True(span.SourceColumns[secret], "paren query must report its table read")

	// Doris keeps repeated clause groups on a GroupedQuery wrapper; the
	// inner group's subquery read must survive.
	span, err = q.getQuerySpan(context.Background(),
		"SELECT 1 ORDER BY (SELECT max(a) FROM secret) ORDER BY 1")
	a.NoError(err)
	a.True(span.SourceColumns[secret], "inner clause group's subquery read must survive the wrap")
}
