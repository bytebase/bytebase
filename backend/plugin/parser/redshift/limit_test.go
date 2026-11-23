package redshift

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestStatementWithResultLimitSeam(t *testing.T) {
	require.Equal(t, "SELECT * FROM users LIMIT 10", base.StatementWithResultLimit(storepb.Engine_REDSHIFT, "SELECT * FROM users", 10, ""))
	require.Equal(t, "WITH result AS (\nSELECT * FROM t LIMIT (1+2)\n) SELECT * FROM result LIMIT 5;", base.StatementWithResultLimit(storepb.Engine_REDSHIFT, "SELECT * FROM t LIMIT (1+2)", 5, ""))
	require.Equal(t, "INSERT INTO users (name) VALUES ('John')", base.StatementWithResultLimit(storepb.Engine_REDSHIFT, "INSERT INTO users (name) VALUES ('John')", 10, ""))
}
