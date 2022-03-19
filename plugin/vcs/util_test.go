package vcs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranch(t *testing.T) {
	tests := []struct {
		ref     string
		want    string
		wantErr bool
	}{
		{
			ref:     "refs/heads/master",
			want:    "master",
			wantErr: false,
		},
		{
			ref:     "refs/heads/feature/foo",
			want:    "feature/foo",
			wantErr: false,
		},
		{
			ref:     "refs/heads/feature/foo",
			want:    "feature/foo",
			wantErr: false,
		},
	}

	for _, test := range tests {
		result, err := Branch(test.ref)
		if test.wantErr {
			require.Error(t, err)
		}
		assert.Equal(t, result, test.want)
	}
}
