package store

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildListChangelogsQuerySchemaSnapshotBefore(t *testing.T) {
	t.Parallel()
	before, err := time.Parse(time.RFC3339Nano, "2024-12-31T23:59:59.123456789Z")
	require.NoError(t, err)
	database := "db"
	limit := 1

	query, args, err := buildListChangelogsQuery(&FindChangelogMessage{
		InstanceID:              "instance",
		DatabaseName:            &database,
		HasSyncHistory:          true,
		CreatedAtStrictlyBefore: &before,
		Limit:                   &limit,
	})
	require.NoError(t, err)
	require.Contains(t, strings.Join(strings.Fields(query), " "), "changelog.sync_history IS NOT NULL")
	require.Contains(t, strings.Join(strings.Fields(query), " "), "changelog.created_at < $3")
	require.Contains(t, strings.Join(strings.Fields(query), " "), "ORDER BY changelog.created_at DESC, changelog.resource_id DESC LIMIT $4")
	require.Equal(t, []any{"instance", "db", before, 1}, args)
}
