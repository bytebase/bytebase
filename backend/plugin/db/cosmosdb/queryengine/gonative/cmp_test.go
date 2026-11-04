package gonative

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareItems_TypeOrder(t *testing.T) {
	// Cosmos order: undefined < null < bool < number < string.
	// We emulate "undefined" with an empty RawMessage.
	cases := []struct {
		name    string
		a, b    json.RawMessage
		wantCmp int
	}{
		{"undefined<null", json.RawMessage(``), json.RawMessage(`null`), -1},
		{"null<bool", json.RawMessage(`null`), json.RawMessage(`false`), -1},
		{"bool<number", json.RawMessage(`true`), json.RawMessage(`0`), -1},
		{"number<string", json.RawMessage(`42`), json.RawMessage(`"a"`), -1},
		{"string>bool", json.RawMessage(`"a"`), json.RawMessage(`true`), 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := compareItems(tc.a, tc.b)
			require.NoError(t, err)
			assert.Equal(t, tc.wantCmp, got)
		})
	}
}

func TestCompareItems_WithinType(t *testing.T) {
	cases := []struct {
		name    string
		a, b    json.RawMessage
		wantCmp int
	}{
		{"bool false<true", json.RawMessage(`false`), json.RawMessage(`true`), -1},
		{"bool equal true", json.RawMessage(`true`), json.RawMessage(`true`), 0},
		{"number ascending", json.RawMessage(`1.5`), json.RawMessage(`2`), -1},
		{"number equal", json.RawMessage(`42`), json.RawMessage(`42`), 0},
		{"number descending", json.RawMessage(`10`), json.RawMessage(`-3`), 1},
		{"string lex", json.RawMessage(`"AD"`), json.RawMessage(`"AE"`), -1},
		{"string equal", json.RawMessage(`"AE"`), json.RawMessage(`"AE"`), 0},
		{"null equal", json.RawMessage(`null`), json.RawMessage(`null`), 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := compareItems(tc.a, tc.b)
			require.NoError(t, err)
			assert.Equal(t, tc.wantCmp, got)
		})
	}
}

func TestCompareItems_UnorderableFails(t *testing.T) {
	_, err := compareItems(json.RawMessage(`[1,2]`), json.RawMessage(`3`))
	require.Error(t, err)
	_, err = compareItems(json.RawMessage(`{}`), json.RawMessage(`null`))
	require.Error(t, err)
}

func TestCompareItems_HandlesLeadingWhitespace(t *testing.T) {
	got, err := compareItems(json.RawMessage(` 1 `), json.RawMessage(`2`))
	require.NoError(t, err)
	assert.Equal(t, -1, got)
}

func TestDetectJSONType(t *testing.T) {
	cases := []struct {
		raw  string
		want jsonType
	}{
		{``, jsonUndefined},
		{`null`, jsonNull},
		{`true`, jsonBool},
		{`false`, jsonBool},
		{`0`, jsonNumber},
		{`42`, jsonNumber},
		{`-1.5`, jsonNumber},
		{`"x"`, jsonString},
		{`[1,2]`, jsonUnorderable},
		{`{"a":1}`, jsonUnorderable},
		{` "pad" `, jsonString},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			assert.Equal(t, tc.want, detectJSONType(json.RawMessage(tc.raw)))
		})
	}
}
