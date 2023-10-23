package masker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMiddle_Byte(t *testing.T) {
	testCases := []struct {
		input []byte
		want  []byte
	}{
		{
			input: []byte("12345678"),
			want:  []byte("3456"),
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		got := middle(tc.input)
		a.Equal(tc.want, got)
	}
}

func TestMiddle_Rune(t *testing.T) {
	testCases := []struct {
		input []rune
		want  []rune
	}{
		{
			input: []rune("🥲🤣🥲🤣"),
			want:  []rune("🤣🥲"),
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		got := middle(tc.input)
		a.Equal(tc.want, got)
	}
}
