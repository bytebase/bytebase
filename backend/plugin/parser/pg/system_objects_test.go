package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolarDBSystemSchema(t *testing.T) {
	require.True(t, IsSystemSchema("pg_bitmapindex"))
	require.Contains(t, strings.Split(SystemSchemaWhereClause, ","), "'pg_bitmapindex'")
}
