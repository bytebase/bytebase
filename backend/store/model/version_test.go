package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionLessThan(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want bool
	}{
		{
			name: "1.0 < 2.0",
			v1:   "1.0",
			v2:   "2.0",
			want: true,
		},
		{
			name: "2.0 !< 1.0",
			v1:   "2.0",
			v2:   "1.0",
			want: false,
		},
		{
			name: "1.0 < 1.1",
			v1:   "1.0",
			v2:   "1.1",
			want: true,
		},
		{
			name: "1.1 !< 1.0",
			v1:   "1.1",
			v2:   "1.0",
			want: false,
		},
		{
			name: "1.0 < 1.0.1",
			v1:   "1.0",
			v2:   "1.0.1",
			want: true,
		},
		{
			name: "1.0.1 !< 1.0",
			v1:   "1.0.1",
			v2:   "1.0",
			want: false,
		},
		{
			name: "1.0.0 !< 1.0.0",
			v1:   "1.0.0",
			v2:   "1.0.0",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1, err := NewVersion(tt.v1)
			require.NoError(t, err)

			v2, err := NewVersion(tt.v2)
			require.NoError(t, err)

			result := v1.LessThan(v2)
			require.Equal(t, tt.want, result)
		})
	}
}
