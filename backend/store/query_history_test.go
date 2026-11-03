package store_test

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// The payload is protojson, so a filter with the wrong key or enum spelling
// matches nothing instead of failing; only a real query shows it.
func TestListQueryHistoriesExportDestinationFilter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, s, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
 INSERT INTO workspace (resource_id) VALUES ('default');
 INSERT INTO project (resource_id, workspace, name) VALUES ('history', 'default', 'History');
 INSERT INTO instance (resource_id, workspace) VALUES ('i', 'default');
 `)
	require.NoError(t, err)

	project := "history"
	for _, history := range []*store.QueryHistoryMessage{
		{Statement: "query", Type: store.QueryHistoryTypeQuery, Payload: &storepb.QueryHistoryPayload{}},
		{Statement: "download", Type: store.QueryHistoryTypeExport, Payload: &storepb.QueryHistoryPayload{
			Export: &storepb.QueryHistoryPayload_Export{Destination: storepb.ExportDestination_DOWNLOAD},
		}},
		{Statement: "sharepoint", Type: store.QueryHistoryTypeExport, Payload: &storepb.QueryHistoryPayload{
			Export: &storepb.QueryHistoryPayload_Export{Destination: storepb.ExportDestination_SHAREPOINT, Uri: "https://example.sharepoint.com/Exports/1"},
		}},
	} {
		history.Creator = "dev@example.com"
		history.Project = project
		history.Database = "instances/i/databases/d"
		_, err := s.CreateQueryHistory(ctx, history)
		require.NoError(t, err)
	}

	filterQ, err := store.GetListQueryHistoryFilter(`type == "QUERY" || export_destination == "SHAREPOINT"`)
	require.NoError(t, err)
	histories, err := s.ListQueryHistories(ctx, &store.FindQueryHistoryMessage{Project: &project, FilterQ: filterQ})
	require.NoError(t, err)
	var statements []string
	for _, history := range histories {
		statements = append(statements, history.Statement)
	}
	slices.Sort(statements)
	require.Equal(t, []string{"query", "sharepoint"}, statements)
}
