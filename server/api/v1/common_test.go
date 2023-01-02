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
	environmentID, instanceID, err := getEnvironmentAndInstanceID("environments/e1/instances/i2")
	require.NoError(t, err)
	require.Equal(t, "e1", environmentID)
	require.Equal(t, "i2", instanceID)

	_, _, err = getEnvironmentAndInstanceID("environments/e1/instances/i2/databases/d3")
	require.Error(t, err)
}
