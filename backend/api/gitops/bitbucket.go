package gitops

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/bitbucket"
	"github.com/bytebase/bytebase/backend/store"
)

func getBitBucketPullRequestInfo(ctx context.Context, vcsProvider *store.VCSProviderMessage, vcsConnector *store.VCSConnectorMessage, body []byte) (*pullRequestInfo, error) {
	var pushEvent bitbucket.PullRequestPushEvent
	if err := json.Unmarshal(body, &pushEvent); err != nil {
		return nil, errors.Errorf("failed to unmarshal push event, error %v", err)
	}

	if pushEvent.PullRequest.Destination.Branch.Name != vcsConnector.Payload.Branch {
		return nil, errors.Errorf("committed to branch %q, want branch %q", pushEvent.PullRequest.Destination.Branch.Name, vcsConnector.Payload.Branch)
	}

	mrFiles, err := vcs.Get(vcs.Bitbucket, vcs.ProviderConfig{InstanceURL: vcsProvider.InstanceURL, AuthToken: vcsProvider.AccessToken}).ListPullRequestFile(ctx, vcsConnector.Payload.ExternalId, fmt.Sprintf("%d", pushEvent.PullRequest.ID))
	if err != nil {
		return nil, errors.Errorf("failed to list merge %q request files, error %v", pushEvent.PullRequest.Links.HTML.Href, err)
	}

	prInfo := &pullRequestInfo{
		// email. How do we determine the user for BitBucket user?
		url:         pushEvent.PullRequest.Links.HTML.Href,
		title:       pushEvent.PullRequest.Title,
		description: pushEvent.PullRequest.Description,
		changes:     getChangesByFileList(mrFiles, vcsConnector.Payload.BaseDirectory),
	}

	for _, file := range prInfo.changes {
		content, err := vcs.Get(vcs.Bitbucket, vcs.ProviderConfig{InstanceURL: vcsProvider.InstanceURL, AuthToken: vcsProvider.AccessToken}).ReadFileContent(ctx, vcsConnector.Payload.ExternalId, file.path, vcs.RefInfo{RefType: vcs.RefTypeCommit, RefName: pushEvent.PullRequest.Source.Commit.Hash})
		if err != nil {
			return nil, errors.Errorf("failed read file content, merge request %q, file %q, error %v", pushEvent.PullRequest.Links.HTML.Href, file.path, err)
		}
		file.content = content
	}
	return prInfo, nil
}
