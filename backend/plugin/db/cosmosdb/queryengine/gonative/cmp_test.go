package gonative

import (
	"encoding/json"
	"slices"
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

// The service sorts mixed types undefined < null < bool < number < string <
// array < object; measured on a container holding one row of each.
func TestCompareItems_RanksCompositesLast(t *testing.T) {
	for _, tc := range []struct {
		name    string
		a, b    json.RawMessage
		wantCmp int
	}{
		{"array after number", json.RawMessage(`[1,2]`), json.RawMessage(`3`), 1},
		{"array after string", json.RawMessage(`[1,2]`), json.RawMessage(`"zzz"`), 1},
		{"object after array", json.RawMessage(`{"a":1}`), json.RawMessage(`[9,9]`), 1},
		{"object after null", json.RawMessage(`{}`), json.RawMessage(`null`), 1},
		{"array before object", json.RawMessage(`[]`), json.RawMessage(`{}`), -1},
		{"equal arrays", json.RawMessage(`[1,2]`), json.RawMessage(`[1,2]`), 0},
		{"equal objects, keys reordered", json.RawMessage(`{"a":1,"b":2}`), json.RawMessage(`{"b":2,"a":1}`), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := compareItems(tc.a, tc.b)
			require.NoError(t, err)
			assert.Equal(t, tc.wantCmp, got)
			back, err := compareItems(tc.b, tc.a)
			require.NoError(t, err)
			assert.Equal(t, -tc.wantCmp, back, "comparison must be antisymmetric")
		})
	}
}

// Within the array and object classes the service orders by the index's own
// encoding, which the values cannot tell us; the engine orders by canonical
// JSON instead, and what it must guarantee is that the order is total and the
// same on every run.
func TestCompareItems_OrdersCompositesConsistently(t *testing.T) {
	values := []json.RawMessage{
		json.RawMessage(`[1,3]`), json.RawMessage(`[2]`), json.RawMessage(`[1,2]`),
		json.RawMessage(`{"b":1}`), json.RawMessage(`{"a":2}`), json.RawMessage(`{"a":1}`),
	}
	sortOnce := func() []string {
		shuffled := slices.Clone(values)
		slices.SortStableFunc(shuffled, func(a, b json.RawMessage) int {
			order, err := compareItems(a, b)
			require.NoError(t, err)
			return order
		})
		out := make([]string, 0, len(shuffled))
		for _, v := range shuffled {
			out = append(out, string(v))
		}
		return out
	}
	first := sortOnce()
	assert.Equal(t, first, sortOnce())
	// Whatever the order within each class, every array precedes every object.
	for i, v := range first {
		if v[0] == '{' {
			for _, rest := range first[i:] {
				assert.Equal(t, byte('{'), rest[0], "an array sorted after an object: %v", first)
			}
			break
		}
	}
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
		{`[1,2]`, jsonArray},
		{`{"a":1}`, jsonObject},
		{` "pad" `, jsonString},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			assert.Equal(t, tc.want, detectJSONType(json.RawMessage(tc.raw)))
		})
	}
}
