package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestGetVCSUserFromGitHubPullRequest(t *testing.T) {
	eventPath := filepath.Join(t.TempDir(), "event.json")
	require.NoError(t, os.WriteFile(eventPath, []byte(`{"pull_request":{"user":{"id":1001,"login":"alice","name":"Alice","type":"User"}}}`), 0600))
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)

	user := getVCSUser(world.GitHub)
	require.NotNil(t, user)
	require.Equal(t, v1pb.VCSType_GITHUB, user.VcsType)
	require.Equal(t, "1001", user.UserId)
	require.Equal(t, "alice", user.UserName)
	require.Equal(t, "Alice", user.DisplayName)
}

func TestGetVCSUserSkipsGitHubBot(t *testing.T) {
	eventPath := filepath.Join(t.TempDir(), "event.json")
	require.NoError(t, os.WriteFile(eventPath, []byte(`{"pull_request":{"user":{"id":41898282,"login":"github-actions[bot]","type":"Bot"}}}`), 0600))
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)

	require.Nil(t, getVCSUser(world.GitHub))
}

func TestGetVCSUserFromGitHubPullRequestTarget(t *testing.T) {
	eventPath := filepath.Join(t.TempDir(), "event.json")
	require.NoError(t, os.WriteFile(eventPath, []byte(`{"pull_request":{"user":{"id":1001,"login":"alice","name":"Alice","type":"User"}}}`), 0600))
	t.Setenv("GITHUB_EVENT_NAME", "pull_request_target")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)

	user := getVCSUser(world.GitHub)
	require.NotNil(t, user)
	require.Equal(t, v1pb.VCSType_GITHUB, user.VcsType)
	require.Equal(t, "1001", user.UserId)
}

func TestGetVCSUserSkipsGitLabWhenAuthorUnavailable(t *testing.T) {
	t.Setenv("CI_PIPELINE_SOURCE", "merge_request_event")
	t.Setenv("GITLAB_USER_ID", "2002")
	t.Setenv("GITLAB_USER_LOGIN", "bob")
	t.Setenv("GITLAB_USER_NAME", "Bob")

	require.Nil(t, getVCSUser(world.GitLab))
}

func TestGetVCSUserSkipsBitbucketWhenAuthorUnavailable(t *testing.T) {
	t.Setenv("BITBUCKET_PR_ID", "10")
	t.Setenv("BITBUCKET_STEP_TRIGGERER_UUID", "{3003}")
	t.Setenv("BITBUCKET_STEP_TRIGGERER_USERNAME", "carol")

	require.Nil(t, getVCSUser(world.Bitbucket))
}

func TestGetVCSUserSkipsUnsupportedPlatform(t *testing.T) {
	require.Nil(t, getVCSUser(world.LocalPlatform))
}

func TestBuildCheckReleaseRequestIncludesVCSUser(t *testing.T) {
	eventPath := filepath.Join(t.TempDir(), "event.json")
	require.NoError(t, os.WriteFile(eventPath, []byte(`{"pull_request":{"user":{"id":1001,"login":"alice","name":"Alice","type":"User"}}}`), 0600))
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)

	req := buildCheckReleaseRequest(&world.World{
		Project:     "projects/prod",
		Targets:     []string{"instances/prod/databases/app"},
		CustomRules: "must be safe",
	}, world.GitHub, []*v1pb.Release_File{
		{
			Path:      "migrations/001.sql",
			Version:   "001",
			Statement: []byte("SELECT 1;"),
		},
	}, v1pb.Release_VERSIONED)

	require.Equal(t, "projects/prod", req.Parent)
	require.Equal(t, []string{"instances/prod/databases/app"}, req.Targets)
	require.Equal(t, "must be safe", req.CustomRules)
	require.NotNil(t, req.VcsUser)
	require.Equal(t, v1pb.VCSType_GITHUB, req.VcsUser.VcsType)
	require.Equal(t, "1001", req.VcsUser.UserId)
}
