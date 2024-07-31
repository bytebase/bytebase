package gitops

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/azure"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func getAzurePullRequestInfo(ctx context.Context, vcsProvider *store.VCSProviderMessage, vcsConnector *store.VCSConnectorMessage, body []byte) (*pullRequestInfo, error) {
	var pushEvent azure.PullRequestEvent
	if err := json.Unmarshal(body, &pushEvent); err != nil {
		return nil, errors.Errorf("failed to unmarshal push event, error %v", err)
	}

	var actionType webhookAction
	switch pushEvent.EventType {
	case azure.PullRequestEventCreated, azure.PullRequestEventUpdated:
		actionType = webhookActionSQLReview
	case azure.PullRequestEventMerged:
		if strings.ToLower(pushEvent.Resource.Status) != "completed" {
			return nil, errors.Errorf("invalid pull request status: %v", pushEvent.Resource.Status)
		}
		if strings.ToLower(pushEvent.Resource.MergeStatus) != "succeeded" {
			return nil, errors.Errorf("invalid pull request merge status: %v", pushEvent.Resource.MergeStatus)
		}
		actionType = webhookActionCreateIssue
	default:
		return nil, errors.Errorf(`skip webhook event action "%v"`, pushEvent.EventType)
	}

	targetBranch, err := vcs.Branch(pushEvent.Resource.TargetRefName)
	if err != nil {
		return nil, errors.Errorf("failed to get target branch, error %v", err)
	}

	if vcsConnector.Payload.Branch != targetBranch {
		return nil, errors.Errorf("committed to branch %q, want branch %q", targetBranch, vcsConnector.Payload.Branch)
	}

	azurePlugin, ok := vcs.Get(
		storepb.VCSType_AZURE_DEVOPS,
		vcs.ProviderConfig{InstanceURL: vcsProvider.InstanceURL, AuthToken: vcsProvider.AccessToken},
	).(*azure.Provider)
	if !ok {
		return nil, errors.Errorf("failed to get azure plugin")
	}

	mrFiles, err := azurePlugin.ListPullRequestFileInCommit(
		ctx,
		vcsConnector.Payload.ExternalId,
		pushEvent.Resource.LastMergeCommit.CommitID,
		pushEvent.Resource.PullRequestID,
	)
	if err != nil {
		return nil, errors.Errorf("failed to list merge request files by commit %v, error %v", pushEvent.Resource.LastMergeCommit.CommitID, err)
	}

	prInfo := &pullRequestInfo{
		action: actionType,
		// TODO(ed): get the email.
		url:         pushEvent.Resource.Links.Web.Href,
		title:       pushEvent.Resource.Title,
		description: pushEvent.Resource.Description,
		changes:     getChangesByFileList(mrFiles, vcsConnector.Payload.BaseDirectory),
	}

	for _, file := range prInfo.changes {
		content, err := azurePlugin.ReadFileContent(
			ctx,
			vcsConnector.Payload.ExternalId,
			file.path,
			vcs.RefInfo{RefType: vcs.RefTypeCommit, RefName: pushEvent.Resource.LastMergeCommit.CommitID})
		if err != nil {
			return nil, errors.Errorf("failed read file content, merge request %q, file %q, error %v", pushEvent.Resource.Links.Web, file.path, err)
		}
		file.content = convertFileContentToUTF8String(content)
	}

	return prInfo, nil
}
