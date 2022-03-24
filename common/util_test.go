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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasPrefixes(tt.args.src, tt.args.prefixes...)
			assert.Equal(t, got, tt.want)
		})
	}
}
