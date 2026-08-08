package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestPostgresSDLAcceptsCreateExtension pins the gate half of the SDL/extension deadlock:
// the PostgreSQL SDL dump emits CREATE EXTENSION (plus COMMENT ON EXTENSION) for every
// installed extension, so the release gate must accept both. While CREATE EXTENSION was
// rejected, Export Schema output was not valid SDL, and the only way to pass the check was
// to delete the declaration — which left the extension declared on the source side of the
// diff alone and made the differ emit DROP FUNCTION for the extension's own functions.
//
// The PostgreSQL ParseStatements func is registered by the parser/pg package, which
// release_service_check.go already imports in this package.
func TestPostgresSDLAcceptsCreateExtension(t *testing.T) {
	sql := `CREATE EXTENSION IF NOT EXISTS "pgcrypto" WITH SCHEMA "public" VERSION '1.3';
COMMENT ON EXTENSION "pgcrypto" IS 'cryptographic functions';
CREATE TABLE "public"."users" ("id" integer NOT NULL);
`
	stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, sql)
	require.NoError(t, err)

	got, err := getStatementTypesWithPositionsForEngine(storepb.Engine_POSTGRES, base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.Len(t, got, 3)

	require.Equal(t, storepb.StatementType_CREATE_EXTENSION, got[0].Type)
	require.Equal(t, storepb.StatementType_COMMENT, got[1].Type)
	require.Equal(t, storepb.StatementType_CREATE_TABLE, got[2].Type)
	for _, stmt := range got {
		require.True(t, isAllowedInSDL(storepb.Engine_POSTGRES, stmt.Type), "%s must be allowed in PostgreSQL SDL", stmt.Type)
	}

	// CREATE EXTENSION stays a PostgreSQL-only overlay: MySQL has no extensions, so its
	// gate must not widen.
	require.False(t, isAllowedInSDL(storepb.Engine_MYSQL, storepb.StatementType_CREATE_EXTENSION),
		"CREATE EXTENSION should be disallowed in MySQL SDL")
}
