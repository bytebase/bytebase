package gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/gitlab"
	"github.com/bytebase/bytebase/backend/store"
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
		return nil, errors.Errorf("invalid webhook event type, got %s, want push", pushEvent.ObjectKind)
	}
	if pushEvent.ObjectAttributes.Action != mergeAction {
		return nil, errors.Errorf("invalid webhook event action, got %s, want merge", pushEvent.ObjectAttributes.Action)
	}

	if pushEvent.ObjectAttributes.TargetBranch != vcsConnector.Payload.Branch {
		return nil, errors.Errorf("committed to branch %q, want branch %q", pushEvent.ObjectAttributes.TargetBranch, vcsConnector.Payload.Branch)
	}
	oauthContext := &common.OauthContext{
		AccessToken: vcsProvider.AccessToken,
	}

	mrFiles, err := vcs.Get(vcs.GitLab, vcs.ProviderConfig{}).ListPullRequestFile(ctx, oauthContext, vcsProvider.InstanceURL, vcsConnector.Payload.ExternalId, fmt.Sprintf("%d", pushEvent.ObjectAttributes.IID))
	if err != nil {
		return nil, errors.Errorf("failed to list merge %q request files, error %v", pushEvent.ObjectAttributes.URL, err)
	}

	prInfo := &pullRequestInfo{
		email:       pushEvent.User.Email,
		url:         pushEvent.ObjectAttributes.URL,
		title:       pushEvent.ObjectAttributes.Title,
		description: pushEvent.ObjectAttributes.Description,
	}
	for _, v := range mrFiles {
		if v.IsDeleted {
			continue
		}
		if filepath.Dir(v.Path) != vcsConnector.Payload.BaseDirectory {
			continue
		}
		change, err := getFileChange(v.Path)
		if err != nil {
			slog.Error("failed to get file change info", slog.String("path", v.Path), log.BBError(err))
		}
		if change != nil {
			change.path = v.Path
			prInfo.changes = append(prInfo.changes, change)
		}
	}
	for _, file := range prInfo.changes {
		content, err := vcs.Get(vcs.GitLab, vcs.ProviderConfig{}).ReadFileContent(ctx, oauthContext, vcsProvider.InstanceURL, vcsConnector.Payload.ExternalId, file.path, vcs.RefInfo{RefType: vcs.RefTypeCommit, RefName: pushEvent.ObjectAttributes.LastCommit.ID})
		if err != nil {
			return nil, errors.Errorf("failed read file content, merge request %q, file %q, error %v", pushEvent.ObjectAttributes.URL, file.path, err)
		}
		file.content = content
	}
	return prInfo, nil
}
