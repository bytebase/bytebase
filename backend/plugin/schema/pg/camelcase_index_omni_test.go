package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/omni/pg/catalog"
)

// TestOmniCamelCaseIndexColumnQuoting_SDL reproduces https://github.com/bytebase/bytebase/issues/19348
// via the omni SDL migration path. CamelCase column names in index expressions must be quoted.
func TestOmniCamelCaseIndexColumnQuoting_SDL(t *testing.T) {
	fromSDL := ""

	toSDL := `
CREATE TABLE "public"."ba_account" (
    "userId" text NOT NULL,
    "accountName" text
);

CREATE INDEX ba_account_userId_idx ON "public"."ba_account" ("userId");
`

	from, err := catalog.LoadSDL(strings.TrimSpace(fromSDL))
	require.NoError(t, err)
	to, err := catalog.LoadSDL(strings.TrimSpace(toSDL))
	require.NoError(t, err)

	diff := catalog.Diff(from, to)
	require.False(t, diff.IsEmpty(), "Diff should not be empty")

	plan := catalog.GenerateMigration(from, to, diff)
	sql := plan.SQL()

	t.Logf("Generated SQL:\n%s", sql)

	require.Contains(t, sql, "ba_account", "Should contain table name")
	require.Contains(t, sql, `"userId"`, "CamelCase column in CREATE INDEX must be quoted")
}

// TestOmniCamelCaseIndexColumnQuoting_AddIndex tests adding an index with CamelCase columns
// to an existing table.
func TestOmniCamelCaseIndexColumnQuoting_AddIndex(t *testing.T) {
	fromSDL := `
CREATE TABLE "public"."ba_account" (
    "userId" text NOT NULL,
    "accountName" text
);
`

	toSDL := `
CREATE TABLE "public"."ba_account" (
    "userId" text NOT NULL,
    "accountName" text
);

CREATE INDEX ba_account_userId_idx ON "public"."ba_account" ("userId");
`

	from, err := catalog.LoadSDL(strings.TrimSpace(fromSDL))
	require.NoError(t, err)
	to, err := catalog.LoadSDL(strings.TrimSpace(toSDL))
	require.NoError(t, err)

	diff := catalog.Diff(from, to)
	require.False(t, diff.IsEmpty(), "Diff should not be empty")

	plan := catalog.GenerateMigration(from, to, diff)
	sql := plan.SQL()

	t.Logf("Generated SQL:\n%s", sql)

	require.Contains(t, sql, "CREATE INDEX")
	require.Contains(t, sql, "ba_account_userid_idx")
	require.Contains(t, sql, `"userId"`, "CamelCase column in CREATE INDEX must be quoted")
}
