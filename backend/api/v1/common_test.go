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

func TestGetFilter(t *testing.T) {
	tests := []struct {
		filter    string
		filterKey string
		want      string
		wantErr   bool
	}{
		{
			filter:    "",
			filterKey: "",
			want:      "",
			wantErr:   true,
		},
		{
			filter:    `project= "projects/abc".`,
			filterKey: "project",
			want:      "projects/abc",
			wantErr:   false,
		},
		{
			filter:    `project= "projects/abc".`,
			filterKey: "instance",
			want:      "",
			wantErr:   true,
		},
		{
			filter:    `project= abc.`,
			filterKey: "project",
			want:      "",
			wantErr:   true,
		},
		{
			filter:    `project= abc`,
			filterKey: "project",
			want:      "",
			wantErr:   true,
		},
		{
			filter:    `project= "projects/abc"`,
			filterKey: "project",
			want:      "",
			wantErr:   true,
		},
	}

	for _, test := range tests {
		value, err := getFilter(test.filter, test.filterKey)
		if test.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.want, value)
		}
	}
}

func TestGetEnvironmentInstanceDatabaseID(t *testing.T) {
	environmentID, instanceID, err := getEnvironmentInstanceID("environments/e1/instances/i2")
	require.NoError(t, err)
	require.Equal(t, "e1", environmentID)
	require.Equal(t, "i2", instanceID)

	_, _, err = getEnvironmentInstanceID("environments/e1/instances/i2/databases/d3")
	require.Error(t, err)
}

func TestGetEBNFTokens(t *testing.T) {
	testCases := []struct {
		input   string
		key     string
		want    []string
		wantErr bool
	}{
		{
			input: `resource="environments/e1/instances/i2".`,
			key:   "resource",
			want: []string{
				"environments/e1/instances/i2",
			},
			wantErr: false,
		},
		{
			input: `resource="environments/e1/instances/i2/databases/db3".`,
			key:   "resource",
			want: []string{
				"environments/e1/instances/i2/databases/db3",
			},
			wantErr: false,
		},
		{
			input: `type="DATABASE_BACKUP_MISSING" | "DATABASE_BACKUP_FAILED".`,
			key:   "type",
			want: []string{
				"DATABASE_BACKUP_MISSING",
				"DATABASE_BACKUP_FAILED",
			},
			wantErr: false,
		},
		{
			input: `a="1" | "2". b="3" | "4".`,
			key:   "b",
			want: []string{
				"3",
				"4",
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		got, err := getEBNFTokens(tc.input, tc.key)
		if tc.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		}
	}
}
