package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCreateIssueDoesNotDeadlockWithProjectDeletion(t *testing.T) {
	err := runWithConcurrentProjectDeletion(t, `
		INSERT INTO plan (id, creator, project, name, description) VALUES (101, 'creator@example.com', 'project-a', 'Plan A', '');
	`, "plan", 9903, func(ctx context.Context, s *store.Store) error {
		planUID := int64(101)
		_, err := s.CreateIssue(ctx, &store.IssueMessage{
			ProjectID:    "project-a",
			CreatorEmail: "creator@example.com",
			Title:        "Issue A",
			Type:         storepb.Issue_DATABASE_CHANGE,
			Payload:      &storepb.Issue{},
			PlanUID:      &planUID,
		})
		return err
	})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "deadlock detected")
}
