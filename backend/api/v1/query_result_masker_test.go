package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/masker"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func strValue(s string) *v1pb.RowValue {
	return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: s}}
}

// tableSource is the table-level span entry every projection over a real table
// carries, even when column resolution failed.
func tableSource() parserbase.SourceColumnSet {
	return parserbase.SourceColumnSet{
		parserbase.ColumnResource{Database: "db", Schema: "public", Table: "t"}: true,
	}
}

// selectSpan builds a plain Select span over one table, with one span result
// per name given.
func selectSpan(names ...string) *parserbase.QuerySpan {
	span := &parserbase.QuerySpan{
		Type:          parserbase.Select,
		SourceColumns: tableSource(),
	}
	for _, name := range names {
		span.Results = append(span.Results, parserbase.QuerySpanResult{
			Name:         name,
			IsPlainField: true,
			SourceColumns: parserbase.SourceColumnSet{
				parserbase.ColumnResource{Database: "db", Schema: "public", Table: "t", Column: name}: true,
			},
		})
	}
	return span
}

func resultWithColumns(names ...string) *v1pb.QueryResult {
	result := &v1pb.QueryResult{ColumnNames: names}
	row := &v1pb.QueryRow{}
	for _, name := range names {
		row.Values = append(row.Values, strValue("raw-"+name))
	}
	result.Rows = []*v1pb.QueryRow{row}
	return result
}

// TestMaskerArityMismatch pins the guard against the span shapes a degraded
// schema snapshot actually produces, and against the shapes that legitimately
// disagree and must not be refused.
func TestMaskerArityMismatch(t *testing.T) {
	testCases := []struct {
		name   string
		span   *parserbase.QuerySpan
		result *v1pb.QueryResult
		want   bool
	}{
		{
			name:   "healthy: span and driver agree",
			span:   selectSpan("id", "email", "ssn"),
			result: resultWithColumns("id", "email", "ssn"),
			want:   false,
		},
		{
			// A table synced with zero columns cannot expand SELECT *, so the
			// span carries no results at all while the driver returns every
			// column. Without the guard every value is copied through raw.
			name:   "table synced with no columns: SELECT * yields no span results",
			span:   &parserbase.QuerySpan{Type: parserbase.Select, SourceColumns: tableSource()},
			result: resultWithColumns("id", "email", "ssn"),
			want:   true,
		},
		{
			// A snapshot missing a leading column shifts every masker onto its
			// neighbour and leaves the last value past the end of the list.
			name:   "snapshot missing a leading column",
			span:   selectSpan("email", "ssn"),
			result: resultWithColumns("id", "email", "ssn"),
			want:   true,
		},
		{
			name:   "span resolves more columns than the driver returned",
			span:   selectSpan("id", "email", "ssn"),
			result: resultWithColumns("id", "email"),
			want:   true,
		},
		{
			// No rows means no data to leak, so an arity disagreement here must
			// not block the query.
			name:   "no rows is never a mismatch",
			span:   &parserbase.QuerySpan{Type: parserbase.Select, SourceColumns: tableSource()},
			result: &v1pb.QueryResult{ColumnNames: []string{"id", "email"}},
			want:   false,
		},
		{
			// `explain analyze` is classified as a plain Select over the target
			// table but produces no span results, while the driver returns a
			// QUERY PLAN column. MaskResults skips EXPLAIN before reaching the
			// guard; this pins that such a shape is otherwise indistinguishable
			// from the zero-column leak above.
			name:   "explain analyze shape is indistinguishable from the leak",
			span:   &parserbase.QuerySpan{Type: parserbase.Select, SourceColumns: tableSource()},
			result: resultWithColumns("QUERY PLAN"),
			want:   true,
		},
		{
			name:   "explain span type is never a mismatch",
			span:   &parserbase.QuerySpan{Type: parserbase.Explain, SourceColumns: tableSource()},
			result: resultWithColumns("QUERY PLAN"),
			want:   false,
		},
		{
			name:   "information schema read is never a mismatch",
			span:   &parserbase.QuerySpan{Type: parserbase.SelectInfoSchema},
			result: resultWithColumns("table_name", "table_type"),
			want:   false,
		},
		{
			name:   "DML is never a mismatch",
			span:   &parserbase.QuerySpan{Type: parserbase.DML},
			result: resultWithColumns("ssn"),
			want:   false,
		},
		{
			name:   "DDL is never a mismatch",
			span:   &parserbase.QuerySpan{Type: parserbase.DDL},
			result: resultWithColumns("result"),
			want:   false,
		},
		{
			// SET / SHOW / SELECT 1 resolve no table, so there is no metadata
			// whose staleness the arity could reveal.
			name:   "projection over no table is never a mismatch",
			span:   &parserbase.QuerySpan{Type: parserbase.Select},
			result: resultWithColumns("search_path"),
			want:   false,
		},
		{
			name:   "nil span",
			span:   nil,
			result: resultWithColumns("id"),
			want:   false,
		},
		{
			name:   "nil result",
			span:   selectSpan("id"),
			result: nil,
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, maskerArityMismatch(tc.span, tc.result))
		})
	}
}

// TestDoMaskResultLeavesExtraColumnsRaw documents the behavior the arity guard
// exists to prevent. doMaskResult applies maskers by position and skips any
// value past the end of the list, so a short masker list silently returns raw
// data. This test must keep passing: it is the hazard, not the fix. If it ever
// fails, doMaskResult started masking defensively on its own and the guard's
// rationale needs revisiting.
func TestDoMaskResultLeavesExtraColumnsRaw(t *testing.T) {
	result := resultWithColumns("id", "email", "ssn")

	// Zero maskers is what a zero-result span produces.
	doMaskResult(nil, nil, result)
	for i, value := range result.Rows[0].Values {
		require.Equal(t, "raw-"+result.ColumnNames[i], value.GetStringValue(),
			"column %d should have been copied through untouched", i)
	}

	// One masker for three columns: only the first is masked.
	result = resultWithColumns("id", "email", "ssn")
	doMaskResult([]masker.Masker{masker.NewDefaultFullMasker()}, nil, result)
	require.NotEqual(t, "raw-id", result.Rows[0].Values[0].GetStringValue())
	require.Equal(t, "raw-email", result.Rows[0].Values[1].GetStringValue())
	require.Equal(t, "raw-ssn", result.Rows[0].Values[2].GetStringValue())
}
