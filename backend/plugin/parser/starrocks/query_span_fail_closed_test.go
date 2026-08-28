package starrocks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestQuerySpanFailClosed pins the BYT-10085 contract at the bytebase layer
// for StarRocks: junk the parser cannot consume fails the span extraction
// instead of yielding an empty span, and the newly accepted ParenSelect
// statement shapes still report their reads — including clauses that live on
// the wrapper itself.
func TestQuerySpanFailClosed(t *testing.T) {
	a := require.New(t)
	q := newQuerySpanExtractor("db1", base.GetQuerySpanContext{}, false)
	secret := base.ColumnResource{Database: "db1", Table: "secret"}

	_, err := q.getQuerySpan(context.Background(), "SELECT a FROM secret GROUP BY a WITH ROLLUP")
	a.Error(err, "engine-invalid tail (WITH ROLLUP) must fail the span, not truncate")

	span, err := q.getQuerySpan(context.Background(), "((SELECT a FROM secret)) LIMIT 2")
	a.NoError(err)
	a.True(span.SourceColumns[secret], "nested paren query must report its table read")

	// The wrapper's own trailing ORDER BY may carry a subquery read.
	span, err = q.getQuerySpan(context.Background(),
		"(SELECT 1) ORDER BY (SELECT max(a) FROM secret)")
	a.NoError(err)
	a.True(span.SourceColumns[secret], "wrapper clause subquery read must be reported")
}
