package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToStoredVersion(t *testing.T) {
	type test struct {
		useSemanticVersion    bool
		version               string
		semanticVersionSuffix string
		want                  string
		wantErr               string
	}
	tests := []test{
		{false, "hello", "", "0000.0000.0000-hello", ""},
		{false, "hello", "world", "0000.0000.0000-hello", ""},
		{true, "hello", "world", "", "No Major.Minor.Patch elements found"},
		{true, "v1.2.3", "world", "", "Invalid character(s) found in major number"},
		{true, "1.10000.3", "world", "", "major, minor, patch version should be < 10000"},
		{true, "1.2.3", "world", "0001.0002.0003-world", ""},
		{true, "2021.1.13", "world", "2021.0001.0013-world", ""},
	}
	for _, tc := range tests {
		got, err := toStoredVersion(tc.useSemanticVersion, tc.version, tc.semanticVersionSuffix)
		if tc.wantErr != "" {
			require.Contains(t, err.Error(), tc.wantErr)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
}

func TestFromStoredVersion(t *testing.T) {
	type test struct {
		storedVersion             string
		wantUseSemanticVersion    bool
		wantVersion               string
		wantSemanticVersionSuffix string
		wantErr                   string
	}
	tests := []test{
		{"0000.0000.0000-hello", false, "hello", "", ""},
		{"0001.0001.0000-hello", true, "1.1.0", "hello", ""},
		{"2021.0001.0013-world", true, "2021.1.13", "world", ""},
		{"2021.0001.0013-hello-world", true, "2021.1.13", "hello-world", ""},
		{"2021.0001.0013", false, "", "", "should contain '-'"},
		{"2021.0001.0013.1234-hello", false, "", "", "should be in semantic version"},
		{"2021.0001-hello", false, "", "", "should be in semantic version"},
		{"2a21.0001.0013-hello", false, "", "", "should be in semantic version"},
		{"10000.0001.0000-hello", false, "", "", "should be < 10000"},
		{"", false, "", "", "should contain '-'"},
		{"hello", false, "", "", "should contain '-'"},
		{"1.2.3", false, "", "", "should contain '-'"},
	}
	for _, tc := range tests {
		gotUseSemanticVersion, gotVersion, gotSemanticVersionSuffix, err := fromStoredVersion(tc.storedVersion)
		if tc.wantErr != "" {
			require.Contains(t, err.Error(), tc.wantErr)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, tc.wantUseSemanticVersion, gotUseSemanticVersion)
		require.Equal(t, tc.wantVersion, gotVersion)
		require.Equal(t, tc.wantSemanticVersionSuffix, gotSemanticVersionSuffix)
	}
}
