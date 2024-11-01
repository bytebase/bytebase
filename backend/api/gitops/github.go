package gitops

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/github"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func getGitHubPullRequestInfo(ctx context.Context, vcsProvider *store.VCSProviderMessage, vcsConnector *store.VCSConnectorMessage, body []byte, profile *config.Profile) (*pullRequestInfo, error) {
	var pushEvent github.PullRequestPushEvent
	if err := json.Unmarshal(body, &pushEvent); err != nil {
		return nil, errors.Errorf("failed to unmarshal push event, error %v", err)
	}

	var actionType webhookAction
	switch pushEvent.Action {
	case github.PullRequestEventOpened, github.PullRequestEventSynchronize:
		actionType = webhookActionSQLReview
	case github.PullRequestEventClosed:
		if !pushEvent.PullRequest.Merged {
			return nil, errors.Errorf("skip pull request close action, pull request is not merged")
		}
		if common.IsDev() && profile.DevelopmentVersioned {
			actionType = webhookActionCreateRelease
		} else {
			actionType = webhookActionCreateIssue
		}
	default:
		return nil, errors.Errorf(`skip webhook event action "%s"`, pushEvent.Action)
	}

	if pushEvent.PullRequest.Base.Ref != vcsConnector.Payload.Branch {
		return nil, errors.Errorf("skip branch, got %q, want %q", pushEvent.PullRequest.Base.Ref, vcsConnector.Payload.Branch)
	}

	mrFiles, err := vcs.Get(storepb.VCSType_GITHUB, vcs.ProviderConfig{InstanceURL: vcsProvider.InstanceURL, AuthToken: vcsProvider.AccessToken}).ListPullRequestFile(ctx, vcsConnector.Payload.ExternalId, fmt.Sprintf("%d", pushEvent.Number))
	if err != nil {
		return nil, errors.Errorf("failed to list merge %q request files, error %v", pushEvent.PullRequest.HTMLURL, err)
	}

	prInfo := &pullRequestInfo{
		action: actionType,
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
		file.content = convertFileContentToUTF8String(content)
	}

	if actionType == webhookActionCreateRelease {
		prInfo.getAllFiles = func(ctx context.Context) ([]*fileChange, error) {
			p := vcs.Get(storepb.VCSType_GITHUB, vcs.ProviderConfig{InstanceURL: vcsProvider.InstanceURL, AuthToken: vcsProvider.AccessToken}).(*github.Provider)

			mergeCommitSha, err := p.GetPullRequestMergedCommit(ctx, vcsConnector.Payload.ExternalId, fmt.Sprintf("%d", pushEvent.Number))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get pull request %d", pushEvent.Number)
			}

			dirFiles, err := p.GetDirectoryFiles(ctx, vcsConnector.Payload.ExternalId, mergeCommitSha, vcsConnector.Payload.BaseDirectory)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get directory files")
			}
			var allFiles []*fileChange
			for _, f := range dirFiles {
				f.Content = convertFileContentToUTF8String(f.Content)
				converted, err := convertVcsFile(f)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to convert vcs file")
				}
				allFiles = append(allFiles, converted)
			}
			return allFiles, nil
		}
	}

	return prInfo, nil
}
