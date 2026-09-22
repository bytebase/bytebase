package doris

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestQuerySpanFailClosed pins the BYT-10085 contract at the bytebase layer:
// a statement the parser cannot fully consume must FAIL the span extraction,
// not return an empty span. Before strict parsing, "SELECT a FROM secret
// xx yy" silently truncated to "SELECT a FROM secret" upstream and variants
// with a junked FROM clause could yield zero AccessTables — masking and
// table-level ACL checks then saw nothing to protect.
func TestQuerySpanFailClosed(t *testing.T) {
	a := require.New(t)
	q := newQuerySpanExtractor("db1", base.GetQuerySpanContext{}, false)

	_, err := q.getQuerySpan(context.Background(), "SELECT a FROM secret xx yy zz")
	a.Error(err, "trailing junk must fail the span, not truncate silently")

	// The newly accepted paren/grouped shapes still extract their reads.
	span, err := q.getQuerySpan(context.Background(), "(SELECT a FROM secret) LIMIT 1")
	a.NoError(err)
	a.True(span.SourceColumns[base.ColumnResource{Database: "db1", Table: "secret"}],
		"paren query must report its table read")

	span, err = q.getQuerySpan(context.Background(), "(SELECT a FROM secret ORDER BY 1) ORDER BY 1")
	a.NoError(err)
	a.True(span.SourceColumns[base.ColumnResource{Database: "db1", Table: "secret"}],
		"grouped query must report its table read")
}
