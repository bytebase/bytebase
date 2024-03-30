package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetTextLength(t *testing.T) {
	tests := []struct {
		s    string
		want int64
	}{
		{
			s:    "CHAR",
			want: 1,
		},
		{
			s:    "longtext",
			want: 4_294_967_295,
		},
		{
			s:    "int",
			want: 0,
		},
		{
			s:    "char(123)",
			want: 123,
		},
		{
			s:    "varchar(123)",
			want: 123,
		},
	}
	for _, tc := range tests {
		got := getTextLength(tc.s)
		require.Equal(t, tc.want, got, tc.s)
	}
}
