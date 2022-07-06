package server

import (
	"testing"
	"time"

	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
	"github.com/stretchr/testify/assert"
)

func TestDedupMigrationFiles(t *testing.T) {
	timestamp1 := "2021-01-13T00:00:00+00:00"
	timestamp2 := "2021-01-14T00:00:00+00:00"
	timestamp3 := "2021-01-15T00:00:00+00:00"
	time1, _ := time.Parse(time.RFC3339, timestamp1)
	time2, _ := time.Parse(time.RFC3339, timestamp2)
	time3, _ := time.Parse(time.RFC3339, timestamp3)

	tests := []struct {
		name       string
		commitList []gitlab.WebhookCommit
		want       []distinctFileItem
	}{
		{
			name:       "Empty",
			commitList: []gitlab.WebhookCommit{},
			want:       nil,
		},
		{
			name: "Single commit, single file",
			commitList: []gitlab.WebhookCommit{
				{
					ID:        "1",
					Title:     "Commit 1",
					Message:   "Update 1",
					Timestamp: timestamp1,
					URL:       "example.com",
					Author: gitlab.WebhookCommitAuthor{
						Name: "bob",
					},
					AddedList: []string{
						"v1.sql",
					},
				},
			},
			want: []distinctFileItem{
				{
					createdTime: time1,
					commit: gitlab.WebhookCommit{
						ID:        "1",
						Title:     "Commit 1",
						Message:   "Update 1",
						Timestamp: timestamp1,
						URL:       "example.com",
						Author: gitlab.WebhookCommitAuthor{
							Name: "bob",
						},
						AddedList: []string{
							"v1.sql",
						},
					},
					fileName: "v1.sql",
				},
			},
		},
		{
			name: "Single commit, multiple files",
			commitList: []gitlab.WebhookCommit{
				{
					ID:        "1",
					Title:     "Commit 1",
					Message:   "Update 1",
					Timestamp: timestamp1,
					URL:       "example.com",
					Author: gitlab.WebhookCommitAuthor{
						Name: "bob",
					},
					AddedList: []string{
						"v1.sql",
						"v2.sql",
					},
				},
			},
			want: []distinctFileItem{
				{
					createdTime: time1,
					commit: gitlab.WebhookCommit{
						ID:        "1",
						Title:     "Commit 1",
						Message:   "Update 1",
						Timestamp: timestamp1,
						URL:       "example.com",
						Author: gitlab.WebhookCommitAuthor{
							Name: "bob",
						},
						AddedList: []string{
							"v1.sql",
							"v2.sql",
						},
					},
					fileName: "v1.sql",
				},
				{
					createdTime: time1,
					commit: gitlab.WebhookCommit{
						ID:        "1",
						Title:     "Commit 1",
						Message:   "Update 1",
						Timestamp: timestamp1,
						URL:       "example.com",
						Author: gitlab.WebhookCommitAuthor{
							Name: "bob",
						},
						AddedList: []string{
							"v1.sql",
							"v2.sql",
						},
					},
					fileName: "v2.sql",
				},
			},
		},
		{
			name: "Multi commits, single file",
			commitList: []gitlab.WebhookCommit{
				{
					ID:        "1",
					Title:     "Commit 1",
					Message:   "Update 1",
					Timestamp: timestamp1,
					URL:       "example.com",
					Author: gitlab.WebhookCommitAuthor{
						Name: "bob",
					},
					AddedList: []string{
						"v1.sql",
					},
				},
				{
					ID:        "2",
					Title:     "Merge branch",
					Message:   "Merge update",
					Timestamp: timestamp2,
					URL:       "example.com",
					Author: gitlab.WebhookCommitAuthor{
						Name: "bob",
					},
					AddedList: []string{
						"v1.sql",
					},
				},
			},
			want: []distinctFileItem{
				{
					createdTime: time2,
					commit: gitlab.WebhookCommit{
						ID:        "2",
						Title:     "Merge branch",
						Message:   "Merge update",
						Timestamp: timestamp2,
						URL:       "example.com",
						Author: gitlab.WebhookCommitAuthor{
							Name: "bob",
						},
						AddedList: []string{
							"v1.sql",
						},
					},
					fileName: "v1.sql",
				},
			},
		},
		{
			name: "Multi commits, multi files",
			commitList: []gitlab.WebhookCommit{
				{
					ID:        "1",
					Title:     "Commit 1",
					Message:   "Update 1",
					Timestamp: timestamp1,
					URL:       "example.com",
					Author: gitlab.WebhookCommitAuthor{
						Name: "bob",
					},
					AddedList: []string{
						"v1.sql",
						"v2.sql",
					},
				},
				{
					ID:        "2",
					Title:     "Commit 2",
					Message:   "Update 2",
					Timestamp: timestamp1,
					URL:       "example.com",
					Author: gitlab.WebhookCommitAuthor{
						Name: "bob",
					},
					AddedList: []string{
						"v3.sql",
					},
				},
				{
					ID:        "3",
					Title:     "Merge branch",
					Message:   "Merge update",
					Timestamp: timestamp3,
					URL:       "example.com",
					Author: gitlab.WebhookCommitAuthor{
						Name: "bob",
					},
					AddedList: []string{
						"v1.sql",
						"v2.sql",
						"v3.sql",
					},
				},
			},
			want: []distinctFileItem{
				{
					createdTime: time3,
					commit: gitlab.WebhookCommit{
						ID:        "3",
						Title:     "Merge branch",
						Message:   "Merge update",
						Timestamp: timestamp3,
						URL:       "example.com",
						Author: gitlab.WebhookCommitAuthor{
							Name: "bob",
						},
						AddedList: []string{
							"v1.sql",
							"v2.sql",
							"v3.sql",
						},
					},
					fileName: "v1.sql",
				},
				{
					createdTime: time3,
					commit: gitlab.WebhookCommit{
						ID:        "3",
						Title:     "Merge branch",
						Message:   "Merge update",
						Timestamp: timestamp3,
						URL:       "example.com",
						Author: gitlab.WebhookCommitAuthor{
							Name: "bob",
						},
						AddedList: []string{
							"v1.sql",
							"v2.sql",
							"v3.sql",
						},
					},
					fileName: "v2.sql",
				},
				{
					createdTime: time3,
					commit: gitlab.WebhookCommit{
						ID:        "3",
						Title:     "Merge branch",
						Message:   "Merge update",
						Timestamp: timestamp3,
						URL:       "example.com",
						Author: gitlab.WebhookCommitAuthor{
							Name: "bob",
						},
						AddedList: []string{
							"v1.sql",
							"v2.sql",
							"v3.sql",
						},
					},
					fileName: "v3.sql",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupMigrationFilesFromCommitList(tt.commitList)
			assert.Equal(t, got, tt.want)
		})
	}
}
