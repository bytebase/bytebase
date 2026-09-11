package pg

import (
	"testing"

	"github.com/bytebase/omni/pg/catalog"
	"github.com/pkg/errors"
)

func TestClassifyAnalyzeError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want fallbackReason
	}{
		{"nil", nil, reasonNone},
		{
			"undefined function",
			&catalog.Error{Code: catalog.CodeUndefinedFunction, Message: "function fn(text) does not exist"},
			reasonExpectedPseudoSemantic,
		},
		{
			"ambiguous function",
			&catalog.Error{Code: catalog.CodeAmbiguousFunction, Message: "function fn is not unique"},
			reasonExpectedPseudoSemantic,
		},
		{
			"datatype mismatch",
			&catalog.Error{Code: catalog.CodeDatatypeMismatch, Message: "UNION types text and integer cannot be matched"},
			reasonExpectedPseudoSemantic,
		},
		{
			"feature not supported",
			&catalog.Error{Code: catalog.CodeFeatureNotSupported, Message: "not supported"},
			reasonExpectedPseudoSemantic,
		},
		{
			"ambiguous column",
			&catalog.Error{Code: catalog.CodeAmbiguousColumn, Message: "column x is ambiguous"},
			reasonExpectedPseudoSemantic,
		},
		{
			"undefined table",
			&catalog.Error{Code: catalog.CodeUndefinedTable, Message: `relation "foo" does not exist`},
			reasonUndefinedReference,
		},
		{
			"undefined column",
			&catalog.Error{Code: catalog.CodeUndefinedColumn, Message: `column "x" does not exist`},
			reasonUndefinedReference,
		},
		{
			"undefined object",
			&catalog.Error{Code: catalog.CodeUndefinedObject, Message: `type "t" does not exist`},
			reasonUndefinedReference,
		},
		{
			"undefined schema",
			&catalog.Error{Code: catalog.CodeUndefinedSchema, Message: `schema "s" does not exist`},
			reasonUndefinedReference,
		},
		{
			"other catalog error",
			&catalog.Error{Code: catalog.CodeDuplicateTable, Message: "duplicate"},
			reasonAnalyzerUnsupported,
		},
		{
			"plain error",
			errors.New("unsupported node type"),
			reasonAnalyzerUnsupported,
		},
		{
			"wrapped catalog error",
			errors.Wrap(&catalog.Error{Code: catalog.CodeDatatypeMismatch, Message: "mismatch"}, "ctx"),
			reasonExpectedPseudoSemantic,
		},
	}
	for _, c := range cases {
		if got := classifyAnalyzeError(c.err); got != c.want {
			t.Errorf("%s: classifyAnalyzeError got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestFallbackReasonString(t *testing.T) {
	cases := map[fallbackReason]string{
		reasonNone:                   "none",
		reasonExpectedPseudoSemantic: "expected_pseudo_semantic",
		reasonUndefinedReference:     "undefined_reference",
		reasonAnalyzerUnsupported:    "analyzer_unsupported",
	}
	for r, want := range cases {
		if got := r.String(); got != want {
			t.Errorf("fallbackReason(%d).String(): got %q, want %q", r, got, want)
		}
	}
}
