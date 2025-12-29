package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseOrderBy(t *testing.T) {
	testCases := []struct {
		input string
		want  []*OrderByKey
		err   error
	}{
		{
			input: "start_time",
			want: []*OrderByKey{
				{
					Key:       "start_time",
					SortOrder: ASC,
				},
			},
		},
		{
			input: "start_time desc",
			want: []*OrderByKey{
				{
					Key:       "start_time",
					SortOrder: DESC,
				},
			},
		},
		{
			input: "start_time desc, count",
			want: []*OrderByKey{
				{
					Key:       "start_time",
					SortOrder: DESC,
				},
				{
					Key:       "count",
					SortOrder: ASC,
				},
			},
		},
	}

	for _, test := range testCases {
		got, err := parseOrderBy(test.input)
		if test.err != nil {
			require.EqualError(t, err, test.err.Error())
		} else {
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		}
	}
}
