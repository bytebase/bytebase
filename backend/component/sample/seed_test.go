package sample

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadSeedData(t *testing.T) {
	data, err := LoadSeedData()
	require.NoError(t, err)
	require.Contains(t, data, "CREATE TABLE employee")
	require.Contains(t, data, "INSERT INTO employee")
}
