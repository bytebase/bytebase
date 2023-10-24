// Package vcs provides the utilities for VCS plugins.
package vcs

import (
	"log/slog"
	"strings"

	"github.com/pkg/errors"
)

// Branch is the helper function returns the branch name from reference name.
// For now, this method only supports branch reference.
// https://git-scm.com/book/en/v2/Git-Internals-Git-References
func Branch(ref string) (string, error) {
	if strings.HasPrefix(ref, "refs/heads/") {
		return strings.TrimPrefix(ref, "refs/heads/"), nil
	}

	return "", errors.Errorf("invalid Git ref: %s", ref)
}

// FileItemType is the type of a file item.
type FileItemType string

// The list of file item types.
const (
	FileItemTypeAdded    FileItemType = "added"
	FileItemTypeModified FileItemType = "modified"
)

// DistinctFileItem is an item for distinct file in push event commits.
// We are observing the push webhook event so that we will receive the event either when:
// 1. A commit is directly pushed to a branch.
// 2. One or more commits are merged to a branch.
//
// There is a complication to deal with the 2nd type. A typical workflow is a developer first
// commits the migration file to the feature branch, and at a later point, she creates a merge
// request to merge the commit to the main branch. Even the developer only creates a single commit
// on the feature branch, that merge request may contain multiple commits (unless both squash and fast-forward merge are used):
// 1. The original commit on the feature branch.
// 2. The merge request commit.
//
// And both commits would include that added migration file. Since we create an issue per migration file,
// we need to filter the commit list to prevent creating a duplicated issue. GitLab has a limitation to distinguish
// whether the commit is a merge commit (https://gitlab.com/gitlab-org/gitlab/-/issues/30914), so we need to dedup
// ourselves. Below is the filtering algorithm:
//  1. If we observe the same migration file multiple times, then we should use the latest migration file. This does not matter
//     for change-based migration since a developer would always create different migration file with incremental names, while it
//     will be important for the state-based migration, since the file name is always the same and we need to use the latest snapshot.
//  2. Maintain the relative commit order between different migration files. If migration file A happens before migration file B,
//     then we should create an issue for migration file A first.
type DistinctFileItem struct {
	CreatedTs int64
	Commit    Commit
	FileName  string
	ItemType  FileItemType
	IsYAML    bool
}

// GetDistinctFileList gets the distinct files from push event commits.
// The caller should ensure the commit list is logically organized.
func (p PushEvent) GetDistinctFileList() []DistinctFileItem {
	// Use list instead of map because we need to maintain the relative commit order in the source branch.
	var distinctFileList []DistinctFileItem
	for _, commit := range p.CommitList {
		slog.Debug("Pre-processing commit to dedup migration files...",
			slog.String("id", commit.ID),
			slog.String("title", commit.Title),
		)

		addDistinctFile := func(fileName string, itemType FileItemType) {
			item := DistinctFileItem{
				CreatedTs: commit.CreatedTs,
				Commit:    commit,
				FileName:  fileName,
				ItemType:  itemType,
				IsYAML:    strings.HasSuffix(fileName, ".yml"),
			}
			for i, file := range distinctFileList {
				// For the migration file with the same name, keep the one from the latest commit
				if item.FileName != file.FileName {
					continue
				}
				isPreviousCommit := file.CreatedTs >= commit.CreatedTs
				if isPreviousCommit {
					// The VCS may reverse the commit order in the commit list, so the index of modified file commit is less than the added file commit.
					// In this case, we should modified the existing distinctFileList item's ItemType to modified.
					if item.ItemType == FileItemTypeAdded {
						distinctFileList[i].ItemType = FileItemTypeAdded
					}
				} else {
					// A file can be added and then modified in a later commit. We should consider the item as added.
					if file.ItemType == FileItemTypeAdded {
						item.ItemType = FileItemTypeAdded
					}
					distinctFileList[i] = item
				}
				// The fileName should be unique within the distinctFileList.
				return
			}
			distinctFileList = append(distinctFileList, item)
		}

		for _, added := range commit.AddedList {
			addDistinctFile(added, FileItemTypeAdded)
		}
		for _, modified := range commit.ModifiedList {
			addDistinctFile(modified, FileItemTypeModified)
		}
	}
	return distinctFileList
}
