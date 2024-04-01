package gitops

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/github"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	closeAction = "closed"
)

func getGitHubPullRequestInfo(ctx context.Context, vcsProvider *store.VCSProviderMessage, vcsConnector *store.VCSConnectorMessage, body []byte) (*pullRequestInfo, error) {
	var pushEvent github.PullRequestPushEvent
	if err := json.Unmarshal(body, &pushEvent); err != nil {
		return nil, errors.Errorf("failed to unmarshal push event, error %v", err)
	}
	if pushEvent.Action != closeAction {
		return nil, errors.Errorf("invalid webhook event action, got %s, want closed", pushEvent.Action)
	}

	if pushEvent.PullRequest.Base.Ref != vcsConnector.Payload.Branch {
		return nil, errors.Errorf("committed to branch %q, want branch %q", pushEvent.PullRequest.Base.Ref, vcsConnector.Payload.Branch)
	}

	mrFiles, err := vcs.Get(storepb.VCSType_GITHUB, vcs.ProviderConfig{InstanceURL: vcsProvider.InstanceURL, AuthToken: vcsProvider.AccessToken}).ListPullRequestFile(ctx, vcsConnector.Payload.ExternalId, fmt.Sprintf("%d", pushEvent.Number))
	if err != nil {
		return nil, errors.Errorf("failed to list merge %q request files, error %v", pushEvent.PullRequest.HTMLURL, err)
	}

	prInfo := &pullRequestInfo{
		// email. How do we determine the user for GitHub user?
		url:         pushEvent.PullRequest.HTMLURL,
		title:       pushEvent.PullRequest.Title,
		description: pushEvent.PullRequest.Body,
		changes:     getChangesByFileList(mrFiles, vcsConnector.Payload.BaseDirectory),
	}

	for _, file := range prInfo.changes {
		content, err := vcs.Get(storepb.VCSType_GITHUB, vcs.ProviderConfig{InstanceURL: vcsProvider.InstanceURL, AuthToken: vcsProvider.AccessToken}).ReadFileContent(ctx, vcsConnector.Payload.ExternalId, file.path, vcs.RefInfo{RefType: vcs.RefTypeCommit, RefName: pushEvent.PullRequest.Head.SHA})
		if err != nil {
			return nil, errors.Errorf("failed read file content, merge request %q, file %q, error %v", pushEvent.PullRequest.HTMLURL, file.path, err)
		}
		file.content = content
	}
	return prInfo, nil
}
