package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseElasticsearchRESTSeam(t *testing.T) {
	res, err := ParseElasticsearchREST("GET /users/_search\n{\"query\":{\"match_all\":{}}}")
	require.NoError(t, err)
	require.Empty(t, res.Errors)
	require.Len(t, res.Requests, 1)
	require.Equal(t, "GET", res.Requests[0].Method)
	require.Equal(t, "/users/_search", res.Requests[0].URL)
	require.Equal(t, []string{`{"query":{"match_all":{}}}`}, res.Requests[0].Data)

	invalid, err := ParseElasticsearchREST("db.users.find({})")
	require.NoError(t, err)
	require.NotEmpty(t, invalid.Errors)
	require.NotNil(t, invalid.Errors[0].Position)
	require.Equal(t, int32(0), invalid.Errors[0].Position.Line)
	require.Equal(t, int32(3), invalid.Errors[0].Position.Column)
}
