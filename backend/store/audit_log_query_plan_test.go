package store_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestSearchAuditLogsDoesNotCachePreparedPlans(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		parameters map[string]string
		dateStyles []string
	}{
		{
			name:       "default",
			dateStyles: []string{"ISO, MDY", "SQL, DMY"},
		},
		{
			name:       "description cache disabled",
			parameters: map[string]string{"description_cache_capacity": "0"},
			dateStyles: []string{"ISO, MDY", "SQL, DMY"},
		},
		{
			name: "exec without caches",
			parameters: map[string]string{
				"default_query_exec_mode":    "exec",
				"description_cache_capacity": "0",
				"statement_cache_capacity":   "0",
			},
			dateStyles: []string{"ISO, MDY"},
		},
		{
			name: "simple protocol without caches",
			parameters: map[string]string{
				"default_query_exec_mode":    "simple_protocol",
				"description_cache_capacity": "0",
				"statement_cache_capacity":   "0",
			},
			dateStyles: []string{"ISO, MDY"},
		},
		{
			name: "describe exec without caches",
			parameters: map[string]string{
				"default_query_exec_mode":    "describe_exec",
				"description_cache_capacity": "0",
				"statement_cache_capacity":   "0",
			},
			dateStyles: []string{"ISO, MDY", "SQL, DMY"},
		},
	} {
		for _, dateStyle := range tc.dateStyles {
			t.Run(tc.name+"/"+dateStyle, func(t *testing.T) {
				t.Parallel()
				checkAuditLogSearchPlanCaching(t, dateStyle, tc.parameters)
			})
		}
	}
}

func checkAuditLogSearchPlanCaching(t *testing.T, dateStyle string, parameters map[string]string) {
	t.Helper()
	ctx := context.Background()
	db, s, dsn := testcontainer.NewMetadataDB(t)
	if len(parameters) > 0 {
		require.NoError(t, s.Close())
		metadataURL, err := url.Parse(dsn)
		require.NoError(t, err)
		query := metadataURL.Query()
		for key, value := range parameters {
			query.Set(key, value)
		}
		metadataURL.RawQuery = query.Encode()
		configuredStore, err := store.New(ctx, metadataURL.String(), false)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, configuredStore.Close()) })
		s = configuredStore
	}

	// Prepared statements belong to a connection; inspect the one used by every search.
	s.GetDB().SetMaxOpenConns(1)
	s.GetDB().SetMaxIdleConns(1)

	_, err := s.GetDB().ExecContext(ctx, `SELECT set_config('DateStyle', $1, false)`, dateStyle)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('ws-test'), ('ws-other')`)
	require.NoError(t, err)

	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	upperBound := createdAt.Add(2 * time.Second)
	const approveMethod = "/bytebase.v1.IssueService/ApproveIssue"
	const updateMethod = "/bytebase.v1.PlanService/UpdatePlan"
	const project = "projects/target"

	_, err = db.ExecContext(ctx, `
		INSERT INTO audit_log (resource_id, workspace, created_at, payload)
		SELECT 'noise-' || n, 'ws-test', $1,
			jsonb_build_object('parent', 'projects/other', 'method', $2::text)
		FROM generate_series(1, 1000) AS n
	`, createdAt, updateMethod)
	require.NoError(t, err)

	for _, row := range []struct {
		id        string
		workspace string
		parent    string
		method    string
		at        time.Time
	}{
		{"before-lower", "ws-test", project, approveMethod, createdAt.Add(-time.Microsecond)},
		{"at-lower", "ws-test", project, approveMethod, createdAt},
		{"tie-a", "ws-test", project, updateMethod, createdAt.Add(time.Second)},
		{"tie-b", "ws-test", project, approveMethod, createdAt.Add(time.Second)},
		{"at-upper", "ws-test", project, updateMethod, upperBound},
		{"after-upper", "ws-test", project, approveMethod, upperBound.Add(time.Microsecond)},
		{"other-workspace", "ws-other", project, approveMethod, createdAt},
		{"other-method", "ws-test", project, "/bytebase.v1.SQLService/Query", createdAt},
	} {
		payload, err := protojson.Marshal(&storepb.AuditLog{
			Parent: row.parent,
			Method: row.method,
			User:   "users/test@example.com",
		})
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `
			INSERT INTO audit_log (resource_id, workspace, created_at, payload)
			VALUES ($1, $2, $3, $4)
		`, row.id, row.workspace, row.at, payload)
		require.NoError(t, err)
	}

	filterQ, err := store.GetSearchAuditLogsFilter(`
		(method == "/bytebase.v1.IssueService/ApproveIssue" || method == "/bytebase.v1.PlanService/UpdatePlan")
		&& create_time >= "2026-01-02T03:04:05.123456789Z"
	`)
	require.NoError(t, err)
	cutoff := createdAt.Add(789 * time.Nanosecond)
	filterQ = store.ApplyRetentionFilter(filterQ, &cutoff)
	filterQ = store.ApplyCreateTimeUpperBound(filterQ, upperBound.Add(789*time.Nanosecond))
	orders, err := store.GetAuditLogOrders("create_time asc")
	require.NoError(t, err)
	limit := 2
	parent := project
	wantTimes := map[string]time.Time{
		"at-lower": createdAt,
		"tie-b":    createdAt.Add(time.Second),
		"tie-a":    createdAt.Add(time.Second),
		"at-upper": upperBound,
	}

	for range 8 {
		var ids []string
		for offset := 0; offset < 4; offset += limit {
			logs, err := s.SearchAuditLogs(ctx, &store.AuditLogFind{
				Workspace:   "ws-test",
				Project:     &parent,
				FilterQ:     filterQ,
				Limit:       &limit,
				Offset:      &offset,
				OrderByKeys: orders,
			})
			require.NoError(t, err)
			require.Len(t, logs, limit)
			for _, log := range logs {
				ids = append(ids, log.ResourceID)
				require.Equal(t, "ws-test", log.Workspace)
				require.Equal(t, project, log.Payload.Parent)
				require.Equal(t, "users/test@example.com", log.Payload.User)
				require.Contains(t, []string{approveMethod, updateMethod}, log.Payload.Method)
				require.WithinDuration(t, wantTimes[log.ResourceID], log.CreatedAt, 0)
			}
		}
		require.Equal(t, []string{"at-lower", "tie-b", "tie-a", "at-upper"}, ids)
	}

	var preparedSearches int
	err = s.GetDB().QueryRowContext(ctx, `
		SELECT count(*) FROM pg_prepared_statements WHERE statement LIKE $1
	`, "%FROM audit_log%").Scan(&preparedSearches)
	require.NoError(t, err)
	require.Zero(t, preparedSearches, "audit searches must not reuse a prepared statement that can select a generic plan")
}
