package migrator

import (
	"context"
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/store"
)

// TestListChangelogsWithInvalidUTF8 verifies that ListChangelogs handles
// invalid UTF-8 in sync_history.raw_dump without crashing.
func TestListChangelogsWithInvalidUTF8(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	pgURL := fmt.Sprintf("postgres://postgres:root-password@%s:%s/postgres", container.GetHost(), container.GetPort())
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, s.Close())
	})

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
			resource_id text NOT NULL DEFAULT 'changelogs/101',
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

		INSERT INTO changelog (resource_id, instance, db_name, status, sync_history_id, payload)
		VALUES ('changelogs/101', 'test-instance', 'test-db', 'DONE', 1, '{}');
	`
	_, err = db.ExecContext(ctx, setup)
	require.NoError(t, err)

	// BASIC view should not read raw_dump at all, so the query must succeed.
	changelogs, err := s.ListChangelogs(ctx, &store.FindChangelogMessage{
		ShowFull: false,
	})
	require.NoError(t, err)
	require.Len(t, changelogs, 1)
	require.Empty(t, changelogs[0].Schema)

	// FULL view fetches raw_dump directly and sanitizes it in Go.
	changelogs, err = s.ListChangelogs(ctx, &store.FindChangelogMessage{
		ShowFull: true,
	})
	require.NoError(t, err)
	require.Len(t, changelogs, 1)
	require.True(t, utf8.ValidString(changelogs[0].Schema), "sanitized schema must be valid UTF-8")
}
