package gitops

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/gitlab"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	mergeRequestObjectKind = "merge_request"
	mergeAction            = "merge"
)

func getGitLabPullRequestInfo(ctx context.Context, vcsProvider *store.VCSProviderMessage, vcsConnector *store.VCSConnectorMessage, body []byte) (*pullRequestInfo, error) {
	var pushEvent gitlab.MergeRequestPushEvent
	if err := json.Unmarshal(body, &pushEvent); err != nil {
		return nil, errors.Errorf("failed to unmarshal push event, error %v", err)
	}
	if pushEvent.ObjectKind != mergeRequestObjectKind {
		return nil, errors.Errorf("skip webhook event type, got %s, want push", pushEvent.ObjectKind)
	}
	if pushEvent.ObjectAttributes.Action != mergeAction {
		return nil, errors.Errorf("skip webhook event action, got %s, want merge", pushEvent.ObjectAttributes.Action)
	}

	if pushEvent.ObjectAttributes.TargetBranch != vcsConnector.Payload.Branch {
		return nil, errors.Errorf("skip branch got %q, want %q", pushEvent.ObjectAttributes.TargetBranch, vcsConnector.Payload.Branch)
	}

	mrFiles, err := vcs.Get(storepb.VCSType_GITLAB, vcs.ProviderConfig{InstanceURL: vcsProvider.InstanceURL, AuthToken: vcsProvider.AccessToken}).ListPullRequestFile(ctx, vcsConnector.Payload.ExternalId, fmt.Sprintf("%d", pushEvent.ObjectAttributes.IID))
	if err != nil {
		return nil, errors.Errorf("failed to list merge %q request files, error %v", pushEvent.ObjectAttributes.URL, err)
	}

	prInfo := &pullRequestInfo{
		email:       pushEvent.User.Email,
		url:         pushEvent.ObjectAttributes.URL,
		title:       pushEvent.ObjectAttributes.Title,
		description: pushEvent.ObjectAttributes.Description,
		changes:     getChangesByFileList(mrFiles, vcsConnector.Payload.BaseDirectory),
	}

	for _, file := range prInfo.changes {
		content, err := vcs.Get(storepb.VCSType_GITLAB, vcs.ProviderConfig{InstanceURL: vcsProvider.InstanceURL, AuthToken: vcsProvider.AccessToken}).ReadFileContent(ctx, vcsConnector.Payload.ExternalId, file.path, vcs.RefInfo{RefType: vcs.RefTypeCommit, RefName: pushEvent.ObjectAttributes.LastCommit.ID})
		if err != nil {
			return nil, errors.Errorf("failed read file content, merge request %q, file %q, error %v", pushEvent.ObjectAttributes.URL, file.path, err)
		}
		file.content = content
	}
	return prInfo, nil
}
