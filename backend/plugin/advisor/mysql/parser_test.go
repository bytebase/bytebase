package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMysql8WindowFunction(t *testing.T) {
	parser := newParser()
	_, warns, err := parser.Parse("SELECT row_number() OVER ( ORDER BY id ), id FROM xxx;", "utf8mb4", "utf8mb4_general_ci")
	require.NoError(t, err)
	assert.Empty(t, warns)
}
