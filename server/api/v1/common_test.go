package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidResourceID(t *testing.T) {
	tests := []struct {
		resourceID string
		want       bool
	}{
		{
			resourceID: "hello123",
			want:       true,
		},
		{
			resourceID: "hello-123",
			want:       true,
		},
		{
			resourceID: "你好",
			want:       false,
		},
		{
			resourceID: "123abc",
			want:       false,
		},
		{
			resourceID: "a1234567890123456789012345678901234567890123456789012345678901234567890",
			want:       false,
		},
	}

	for _, test := range tests {
		got := isValidResourceID(test.resourceID)
		require.Equal(t, test.want, got, test.resourceID)
	}
}
