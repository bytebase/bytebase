package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetInstanceDatabaseID(t *testing.T) {
	instanceID, err := GetInstanceID("instances/i2")
	require.NoError(t, err)
	require.Equal(t, "i2", instanceID)

	_, err = GetInstanceID("instances/i2/databases/d3")
	require.Error(t, err)
}

func TestGetRiskID(t *testing.T) {
	tests := []struct {
		name string
		want int64
	}{
		{
			name: "risks/1234",
			want: 1234,
		},
		{
			name: "risks/12345678901",
			want: 12345678901,
		},
	}

	a := require.New(t)
	for _, test := range tests {
		got, err := GetRiskID(test.name)
		a.NoError(err)
		a.Equal(test.want, got)
	}
}
