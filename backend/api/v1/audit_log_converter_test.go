package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// The resource name is built from the payload's parent, not the row's
// workspace. Split out of the retention test that moved to backend/store.
func TestConvertToAuditLogsNamesRowsByPayloadParent(t *testing.T) {
	logs := convertToAuditLogs([]*store.AuditLog{
		{ResourceID: "after-cutoff", CreatedAt: time.Unix(2, 0), Payload: &storepb.AuditLog{Parent: "projects/project-a"}},
		{ResourceID: "at-cutoff", CreatedAt: time.Unix(1, 0), Payload: &storepb.AuditLog{Parent: "projects/project-a"}},
	})

	var names []string
	for _, l := range logs {
		names = append(names, l.Name)
	}
	require.Equal(t, []string{
		"projects/project-a/auditLogs/after-cutoff",
		"projects/project-a/auditLogs/at-cutoff",
	}, names)
}
