package tests

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// TestAuditLogSearchPagesTerminate pins that a search's own audit row does not
// shift the traversal it belongs to. Reading the audit log is audited, and the
// search pages by offset, so the row each request writes lands at the top of
// the set the next page counts from: at page size 1 the traversal would return
// the same row forever.
//
//nolint:tparallel // The traversals share one server and run in sequence.
func TestAuditLogSearchPagesTerminate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	require.NoError(t, err)

	search := func(t *testing.T, orderBy, token string) *v1pb.SearchAuditLogsResponse {
		t.Helper()
		resp, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
			Parent:    workspace.Msg.Name,
			Filter:    `method == "` + v1connect.AuditLogServiceSearchAuditLogsProcedure + `"`,
			OrderBy:   orderBy,
			PageSize:  1,
			PageToken: token,
		}))
		require.NoError(t, err)
		return resp.Msg
	}

	// Two searches to seed the set these traversals page over.
	search(t, "", "")
	search(t, "", "")

	// Newest-first returns the same row forever without the bound; oldest-first
	// grows the set at the rate the offset advances.
	for _, orderBy := range []string{"", "create_time desc", "create_time asc"} {
		t.Run("order_by="+orderBy, func(t *testing.T) {
			seen := map[string]bool{}
			token := ""
			for page := 0; page < 20; page++ {
				resp := search(t, orderBy, token)
				for _, row := range resp.AuditLogs {
					require.False(t, seen[row.Name], "page %d repeated %s", page, row.Name)
					seen[row.Name] = true
				}
				token = resp.NextPageToken
				if token == "" {
					break
				}
			}
			require.Empty(t, token, "the traversal must end, whatever the searches themselves wrote")
			require.NotEmpty(t, seen, "the traversal must read the rows the earlier searches wrote")
		})
	}
}
