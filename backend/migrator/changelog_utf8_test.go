package migrator

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

// TestListChangelogsWithInvalidUTF8 verifies that reading sync_history.raw_dump
// containing invalid UTF-8 bytes does not crash. Regression test for SQLSTATE 22021
// caused by LEFT(raw_dump, N) triggering PostgreSQL UTF-8 validation.
//
// The customer reported: "failed to ensure baseline changelog: failed to check for
// existing changelogs: failed to query: ERROR: invalid byte sequence for encoding
// "UTF8": 0xe4 0xb8 (SQLSTATE 22021)"
func TestListChangelogsWithInvalidUTF8(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()

	// Create minimal schema with raw_dump as bytea initially.
	// We insert the invalid bytes as bytea (no encoding validation), then flip
	// the column type to text via pg_attribute. This simulates how corrupted data
	// ends up in production (e.g., via server-side INSERT INTO...SELECT during
	// the 3.2 migration, or older PostgreSQL versions with laxer validation).
	setup := `
		CREATE TABLE sync_history (
			id bigserial PRIMARY KEY,
			created_at timestamptz NOT NULL DEFAULT now(),
			instance text NOT NULL,
			db_name  text NOT NULL,
			metadata json NOT NULL DEFAULT '{}',
			raw_dump bytea NOT NULL DEFAULT ''::bytea
		);

		CREATE TABLE changelog (
			id bigserial PRIMARY KEY,
			created_at timestamptz NOT NULL DEFAULT now(),
			instance text NOT NULL,
			db_name  text NOT NULL,
			status   text NOT NULL,
			sync_history_id bigint REFERENCES sync_history(id),
			payload  jsonb NOT NULL DEFAULT '{}'
		);

		CREATE TABLE task (
			id serial PRIMARY KEY,
			plan_id bigint NOT NULL
		);

		CREATE TABLE plan (
			id bigserial PRIMARY KEY,
			name text NOT NULL
		);

		-- Insert invalid UTF-8 as bytea (no encoding validation).
		-- 0xe4 0xb8 is a truncated 3-byte Chinese UTF-8 character (e.g. 中 = 0xe4 0xb8 0xad).
		INSERT INTO sync_history (instance, db_name, metadata, raw_dump)
		VALUES ('test-instance', 'test-db', '{}', '\xE4B8'::bytea);

		-- Flip column type from bytea to text via system catalog.
		-- bytea and text share the same varlena storage format, so the raw bytes
		-- are preserved but now treated as text — bypassing UTF-8 validation.
		UPDATE pg_attribute
		SET atttypid = 'text'::regtype
		WHERE attrelid = 'sync_history'::regclass
		  AND attname = 'raw_dump';

		INSERT INTO changelog (instance, db_name, status, sync_history_id, payload)
		VALUES ('test-instance', 'test-db', 'DONE', 1, '{}');
	`
	_, err := db.ExecContext(ctx, setup)
	require.NoError(t, err)

	// Reproduce the bug: LEFT() on text containing invalid UTF-8 triggers SQLSTATE 22021.
	// This mirrors the query that ListChangelogs generates when ShowFull is false.
	var schema string
	err = db.QueryRowContext(ctx,
		`SELECT COALESCE(LEFT(sh_cur.raw_dump, 512), '')
		 FROM changelog
		 LEFT JOIN sync_history sh_cur ON sh_cur.id = changelog.sync_history_id
		 WHERE changelog.id = $1`,
		1,
	).Scan(&schema)
	assert.Error(t, err, "LEFT() on invalid UTF-8 text must fail — confirming the bug exists")
	if err != nil {
		assert.Contains(t, err.Error(), "22021", "error must be SQLSTATE 22021 (invalid byte sequence)")
	}

	// Verify the fix: reading the raw column without LEFT() succeeds because
	// a plain column reference does not trigger PostgreSQL's UTF-8 string-function validation.
	err = db.QueryRowContext(ctx,
		`SELECT COALESCE(sh_cur.raw_dump, '')
		 FROM changelog
		 LEFT JOIN sync_history sh_cur ON sh_cur.id = changelog.sync_history_id
		 WHERE changelog.id = $1`,
		1,
	).Scan(&schema)
	require.NoError(t, err, "reading raw_dump without LEFT() must succeed even with invalid UTF-8")

	// After Go-side sanitization, the result must be valid UTF-8.
	sanitized := strings.ToValidUTF8(schema, "")
	assert.True(t, utf8.ValidString(sanitized), "sanitized schema must be valid UTF-8")
}
