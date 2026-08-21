package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// pgcryptoMetadata is a database carrying an installed extension the way the sync reports
// one: the extension itself is listed, and the functions it owns arrive with SkipDump set
// (they are the pg_depend deptype='e' dependents).
func pgcryptoMetadata() *storepb.DatabaseSchemaMetadata {
	return &storepb.DatabaseSchemaMetadata{
		Extensions: []*storepb.ExtensionMetadata{
			{Name: "pgcrypto", Schema: "public", Version: "1.3", Description: "cryptographic functions"},
		},
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{Name: "users", Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer"}}},
				},
				Functions: []*storepb.FunctionMetadata{
					{
						Name:       "armor",
						Signature:  "armor(bytea)",
						SkipDump:   true,
						Definition: "CREATE FUNCTION armor(bytea) RETURNS text LANGUAGE c AS '$libdir/pgcrypto', 'pg_armor';",
					},
				},
			},
		},
	}
}

// TestExtensionSDL_DumpIsRoundTripClean proves the SDL dump of a database with an extension
// is usable as an SDL file: it declares the extension, omits the objects the extension owns,
// and diffs against itself to nothing. Before the gate accepted CREATE EXTENSION, feeding
// Export Schema output back in as SDL failed the check outright.
func TestExtensionSDL_DumpIsRoundTripClean(t *testing.T) {
	meta := model.NewDatabaseMetadata(pgcryptoMetadata(), nil, nil, storepb.Engine_POSTGRES, false)

	sdl, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, meta)
	require.NoError(t, err)

	require.Contains(t, sdl, `CREATE EXTENSION IF NOT EXISTS "pgcrypto"`)
	require.Contains(t, sdl, "users")
	require.NotContains(t, sdl, "armor", "extension-owned functions must stay out of the SDL")

	require.Empty(t, strings.TrimSpace(omniSDLMigration(t, sdl, sdl)),
		"the exported SDL must diff against itself to nothing")
}

// TestExtensionSDL_DeclaredExtensionProducesNoFunctionDrops covers the differ half of the
// deadlock. Loading CREATE EXTENSION materializes the bundled extension's functions in the
// catalog, so the declaration has to be present on BOTH sides for them to cancel out.
// With it declared, adding a table is the only change — no DROP FUNCTION.
func TestExtensionSDL_DeclaredExtensionProducesNoFunctionDrops(t *testing.T) {
	source := `CREATE EXTENSION IF NOT EXISTS "pgcrypto" WITH SCHEMA "public" VERSION '1.3';`
	target := source + "\n" + `CREATE TABLE "public"."users" ("id" integer NOT NULL);`

	sql := omniSDLMigration(t, source, target)

	require.Contains(t, sql, "users")
	require.NotContains(t, sql, "DROP FUNCTION",
		"a declared extension must never yield drops for the functions it owns")
}

// TestExtensionSDL_UndeclaredExtensionIsAnError pins the guard for the state the reporter
// was pushed into: an SDL that omits an installed extension used to generate DROP FUNCTION
// for each of the extension's functions, which PostgreSQL refuses ("cannot drop function
// armor(bytea) because extension pgcrypto requires it", SQLSTATE 2BP01) — so the rollout
// could never succeed. The check now fails with an actionable ERROR instead.
func TestExtensionSDL_UndeclaredExtensionIsAnError(t *testing.T) {
	current := model.NewDatabaseMetadata(pgcryptoMetadata(), nil, nil, storepb.Engine_POSTGRES, false)
	userSDL := `CREATE TABLE "public"."users" ("id" integer NOT NULL);`

	advices, err := pgSDLDropAdvices(userSDL, current, "")
	require.NoError(t, err)

	var found bool
	for _, advice := range advices {
		if advice.GetCode() != code.SDLUndeclaredExtension.Int32() {
			continue
		}
		found = true
		require.Equal(t, storepb.Advice_ERROR, advice.GetStatus())
		require.Contains(t, advice.GetContent(), `CREATE EXTENSION IF NOT EXISTS "pgcrypto"`)
	}
	require.True(t, found, "an installed but undeclared extension must fail the SDL check")
}
