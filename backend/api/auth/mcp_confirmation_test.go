package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMCPStatementDigestKeepsTheFieldsApart(t *testing.T) {
	digest := MCPStatementDigest("DELETE FROM t", "instances/a/databases/b", "app", "")
	require.Equal(t, digest, MCPStatementDigest("DELETE FROM t", "instances/a/databases/b", "app", ""))
	require.NotEqual(t, digest, MCPStatementDigest("DELETE FROM t", "instances/a/databases/b", "", ""))
	require.NotEqual(t, digest, MCPStatementDigest("DELETE FROM t", "instances/a/databases/b", "app", "orders"))
	require.NotEqual(t, MCPStatementDigest("DELETE FROM tapp", "", "", ""), MCPStatementDigest("DELETE FROM t", "app", "", ""))
	require.NotEqual(t, MCPStatementDigest("DELETE FROM t", "", "app", ""), MCPStatementDigest("DELETE FROM t", "", "", "app"))
}
