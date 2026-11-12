package github

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorCommand(t *testing.T) {
	t.Parallel()

	require.Equal(t, "::error::sign in as ci@workload.bytebase.com: refused", ErrorCommand("sign in as ci@workload.bytebase.com: refused"))
	require.Equal(t, "::error::50%25 done%0D%0A::warning::injected", ErrorCommand("50% done\r\n::warning::injected"))
}
