package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeSavedQueryFolder(t *testing.T) {
	testCases := []struct {
		name    string
		folder  string
		want    string
		wantErr bool
	}{
		{name: "unfiled", folder: "", want: ""},
		{name: "already normalized", folder: "team/reports", want: "team/reports"},
		{name: "leading and trailing slashes", folder: "/team/", want: "team"},
		{name: "nested with boundary slashes", folder: "/team/reports/", want: "team/reports"},
		{name: "slashes only", folder: "///", want: ""},
		{name: "empty segment rejected", folder: "team//reports", wantErr: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeSavedQueryFolder(tc.folder)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
