package v1

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

func isOwnerOrDBA(role api.Role) bool {
	return role == api.Owner || role == api.DBA
}

// isProjectOwnerOrDeveloper returns whether a principal is a project owner or developer in the project.
func isProjectOwnerOrDeveloper(principalID int, projectPolicy *store.IAMPolicyMessage) bool {
	for _, binding := range projectPolicy.Bindings {
		if binding.Role != api.Owner && binding.Role != api.Developer {
			continue
		}
		for _, member := range binding.Members {
			if member.ID == principalID || member.Email == api.AllUsers {
				return true
			}
		}
	}
	return false
}

func getUserBelongingProjects(ctx context.Context, s *store.Store, userUID int) (map[string]bool, error) {
	projects, err := s.ListProjectV2(ctx, &store.FindProjectMessage{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list projects")
	}

	projectIDs := map[string]bool{}
	for _, project := range projects {
		policy, err := s.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &project.ResourceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get project %q iam policy", project.ResourceID)
		}
		if isProjectMember(userUID, policy) {
			projectIDs[project.ResourceID] = true
		}
	}
	return projectIDs, nil
}

func isUserAtLeastProjectViewer(ctx context.Context, s *store.Store, requestProjectID string) (bool, error) {
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return false, status.Errorf(codes.Internal, "principal ID not found")
	}
	user, err := s.GetUserByID(ctx, principalID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get user %d", principalID)
	}

	if isOwnerOrDBA(user.Role) {
		return true, nil
	}

	policy, err := s.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &requestProjectID})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get project iam policy")
	}

	if isProjectOwnerDeveloperOrViewer(principalID, policy) {
		return true, nil
	}
	return false, nil
}

// isProjectOwnerDeveloperOrViewer returns whether a principal is a project owner or developer in the project.
func isProjectOwnerDeveloperOrViewer(principalID int, projectPolicy *store.IAMPolicyMessage) bool {
	for _, binding := range projectPolicy.Bindings {
		if binding.Role != api.Owner && binding.Role != api.Developer && binding.Role != api.ProjectViewer {
			continue
		}
		for _, member := range binding.Members {
			if member.ID == principalID || member.Email == api.AllUsers {
				return true
			}
		}
	}
	return false
}

func isUserAtLeastProjectDeveloper(ctx context.Context, s *store.Store, requestProjectID string) (bool, error) {
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return false, status.Errorf(codes.Internal, "principal ID not found")
	}
	user, err := s.GetUserByID(ctx, principalID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get user %d", principalID)
	}

	if isOwnerOrDBA(user.Role) {
		return true, nil
	}

	policy, err := s.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &requestProjectID})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get project iam policy")
	}

	if isProjectOwnerOrDeveloper(principalID, policy) {
		return true, nil
	}
	return false, nil
}

func isUserAtLeastProjectMember(ctx context.Context, s *store.Store, requestProjectID string) (bool, error) {
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return false, status.Errorf(codes.Internal, "principal ID not found")
	}
	user, err := s.GetUserByID(ctx, principalID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get user %d", principalID)
	}

	if isOwnerOrDBA(user.Role) {
		return true, nil
	}

	policy, err := s.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &requestProjectID})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get project iam policy")
	}

	if isProjectMember(principalID, policy) {
		return true, nil
	}
	return false, nil
}
