package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasPrefixes(t *testing.T) {
	type args struct {
		src      string
		prefixes []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "has prefixes",
			args: args{
				src:      "abc",
				prefixes: []string{"a", "b", "c"},
			},
			want: true,
		},
		{
			name: "has no matching prefix",
			args: args{
				src:      "this is a sentence",
				prefixes: []string{"that", "x", "y"},
			},
			want: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got := HasPrefixes(tt.args.src, tt.args.prefixes...)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name      string
		str       string
		limit     int
		want      string
		truncated bool
	}{
		{
			name:      "simple truncate 0",
			str:       "0123",
			limit:     0,
			want:      "",
			truncated: true,
		},
		{
			name:      "simple truncate 2",
			str:       "0123",
			limit:     2,
			want:      "01",
			truncated: true,
		},
		{
			name:      "simple truncate 3",
			str:       "0123",
			limit:     3,
			want:      "012",
			truncated: true,
		},
		{
			name:      "simple truncate 4",
			str:       "0123",
			limit:     4,
			want:      "0123",
			truncated: false,
		},
		{
			name:      "simple truncate 20",
			str:       "0123",
			limit:     20,
			want:      "0123",
			truncated: false,
		},
		{
			name:      "unicode truncate 5",
			str:       "H㐀〾▓朗퐭텟şüöžåйкл¤",
			limit:     5,
			want:      "H㐀〾▓朗",
			truncated: true,
		},
		{
			name:      "unicode truncate 10",
			str:       "H㐀〾▓朗퐭텟şüöžåйкл¤",
			limit:     10,
			want:      "H㐀〾▓朗퐭텟şüö",
			truncated: true,
		},
		{
			name:      "unicode fit",
			str:       "H㐀〾▓朗퐭텟şüöžåйкл¤",
			limit:     16,
			want:      "H㐀〾▓朗퐭텟şüöžåйкл¤",
			truncated: false,
		},
	}
	a := assert.New(t)
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			got, truncated := TruncateString(test.str, test.limit)
			a.Equal(test.want, got)
			a.Equal(test.truncated, truncated)
		})
	}
}
