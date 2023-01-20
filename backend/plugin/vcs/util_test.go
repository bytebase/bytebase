package vcs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranch(t *testing.T) {
	tests := []struct {
		ref     string
		want    string
		wantErr bool
	}{
		{
			ref:     "refs/heads/master",
			want:    "master",
			wantErr: false,
		},
		{
			ref:     "refs/heads/feature/foo",
			want:    "feature/foo",
			wantErr: false,
		},
		{
			ref:     "refs/heads/feature/foo",
			want:    "feature/foo",
			wantErr: false,
		},
	}

	for _, test := range tests {
		result, err := Branch(test.ref)
		if test.wantErr {
			require.Error(t, err)
		}
		assert.Equal(t, result, test.want)
	}
}

func TestGetDistinctFileList(t *testing.T) {
	timestamp1 := "2021-01-13T00:00:00+00:00"
	timestamp2 := "2021-01-14T00:00:00+00:00"
	timestamp3 := "2021-01-15T00:00:00+00:00"
	time1, _ := time.Parse(time.RFC3339, timestamp1)
	time2, _ := time.Parse(time.RFC3339, timestamp2)
	time3, _ := time.Parse(time.RFC3339, timestamp3)
	ts1 := time1.Unix()
	ts2 := time2.Unix()
	ts3 := time3.Unix()

	tests := []struct {
		name       string
		commitList []Commit
		want       []DistinctFileItem
	}{
		{
			name:       "Empty",
			commitList: []Commit{},
			want:       nil,
		},
		{
			name: "Single commit, single file",
			commitList: []Commit{
				{
					ID:         "1",
					Title:      "Commit 1",
					Message:    "Update 1",
					CreatedTs:  ts1,
					URL:        "example.com",
					AuthorName: "bob",
					AddedList: []string{
						"v1.sql",
					},
				},
			},
			want: []DistinctFileItem{
				{
					CreatedTs: ts1,
					Commit: Commit{
						ID:         "1",
						Title:      "Commit 1",
						Message:    "Update 1",
						CreatedTs:  ts1,
						URL:        "example.com",
						AuthorName: "bob",
						AddedList: []string{
							"v1.sql",
						},
					},
					FileName: "v1.sql",
					ItemType: FileItemTypeAdded,
				},
			},
		},
		{
			name: "Single commit, multiple files",
			commitList: []Commit{
				{
					ID:         "1",
					Title:      "Commit 1",
					Message:    "Update 1",
					CreatedTs:  ts1,
					URL:        "example.com",
					AuthorName: "bob",
					AddedList: []string{
						"v1.sql",
						"v2.sql",
					},
				},
			},
			want: []DistinctFileItem{
				{
					CreatedTs: ts1,
					Commit: Commit{
						ID:         "1",
						Title:      "Commit 1",
						Message:    "Update 1",
						CreatedTs:  ts1,
						URL:        "example.com",
						AuthorName: "bob",
						AddedList: []string{
							"v1.sql",
							"v2.sql",
						},
					},
					FileName: "v1.sql",
					ItemType: FileItemTypeAdded,
				},
				{
					CreatedTs: ts1,
					Commit: Commit{
						ID:         "1",
						Title:      "Commit 1",
						Message:    "Update 1",
						CreatedTs:  ts1,
						URL:        "example.com",
						AuthorName: "bob",
						AddedList: []string{
							"v1.sql",
							"v2.sql",
						},
					},
					FileName: "v2.sql",
					ItemType: FileItemTypeAdded,
				},
			},
		},
		{
			name: "Multi commits, single file",
			commitList: []Commit{
				{
					ID:         "1",
					Title:      "Commit 1",
					Message:    "Update 1",
					CreatedTs:  ts1,
					URL:        "example.com",
					AuthorName: "bob",
					AddedList: []string{
						"v1.sql",
					},
				},
				{
					ID:         "2",
					Title:      "Merge branch",
					Message:    "Merge update",
					CreatedTs:  ts2,
					URL:        "example.com",
					AuthorName: "bob",
					AddedList: []string{
						"v1.sql",
					},
				},
			},
			want: []DistinctFileItem{
				{
					CreatedTs: ts2,
					Commit: Commit{
						ID:         "2",
						Title:      "Merge branch",
						Message:    "Merge update",
						CreatedTs:  ts2,
						URL:        "example.com",
						AuthorName: "bob",
						AddedList: []string{
							"v1.sql",
						},
					},
					FileName: "v1.sql",
					ItemType: FileItemTypeAdded,
				},
			},
		},
		{
			name: "Multi commits, single file, direct push",
			commitList: []Commit{
				{
					ID:         "1",
					Title:      "Commit 1",
					Message:    "Update 1",
					CreatedTs:  ts1,
					URL:        "example.com",
					AuthorName: "bob",
					AddedList: []string{
						"v1.sql",
					},
				},
				{
					ID:         "2",
					Title:      "Commit 2",
					Message:    "Update 2",
					CreatedTs:  ts2,
					URL:        "example.com",
					AuthorName: "bob",
					ModifiedList: []string{
						"v1.sql",
					},
				},
			},
			want: []DistinctFileItem{
				{
					CreatedTs: ts2,
					Commit: Commit{
						ID:         "2",
						Title:      "Commit 2",
						Message:    "Update 2",
						CreatedTs:  ts2,
						URL:        "example.com",
						AuthorName: "bob",
						ModifiedList: []string{
							"v1.sql",
						},
					},
					FileName: "v1.sql",
					ItemType: FileItemTypeAdded,
				},
			},
		},
		{
			name: "Multi commits, multi files",
			commitList: []Commit{
				{
					ID:         "1",
					Title:      "Commit 1",
					Message:    "Update 1",
					CreatedTs:  ts1,
					URL:        "example.com",
					AuthorName: "bob",
					AddedList: []string{
						"v1.sql",
						"v2.sql",
					},
				},
				{
					ID:         "2",
					Title:      "Commit 2",
					Message:    "Update 2",
					CreatedTs:  ts1,
					URL:        "example.com",
					AuthorName: "bob",
					AddedList: []string{
						"v3.sql",
					},
				},
				{
					ID:         "3",
					Title:      "Merge branch",
					Message:    "Merge update",
					CreatedTs:  ts3,
					URL:        "example.com",
					AuthorName: "bob",
					AddedList: []string{
						"v1.sql",
						"v2.sql",
						"v3.sql",
					},
				},
			},
			want: []DistinctFileItem{
				{
					CreatedTs: ts3,
					Commit: Commit{
						ID:         "3",
						Title:      "Merge branch",
						Message:    "Merge update",
						CreatedTs:  ts3,
						URL:        "example.com",
						AuthorName: "bob",
						AddedList: []string{
							"v1.sql",
							"v2.sql",
							"v3.sql",
						},
					},
					FileName: "v1.sql",
					ItemType: FileItemTypeAdded,
				},
				{
					CreatedTs: ts3,
					Commit: Commit{
						ID:         "3",
						Title:      "Merge branch",
						Message:    "Merge update",
						CreatedTs:  ts3,
						URL:        "example.com",
						AuthorName: "bob",
						AddedList: []string{
							"v1.sql",
							"v2.sql",
							"v3.sql",
						},
					},
					FileName: "v2.sql",
					ItemType: FileItemTypeAdded,
				},
				{
					CreatedTs: ts3,
					Commit: Commit{
						ID:         "3",
						Title:      "Merge branch",
						Message:    "Merge update",
						CreatedTs:  ts3,
						URL:        "example.com",
						AuthorName: "bob",
						AddedList: []string{
							"v1.sql",
							"v2.sql",
							"v3.sql",
						},
					},
					FileName: "v3.sql",
					ItemType: FileItemTypeAdded,
				},
			},
		},
		{
			name: "Multi commits, multi files, include modified",
			commitList: []Commit{
				{
					ID:         "1",
					Title:      "Commit 1",
					Message:    "Update 1",
					CreatedTs:  ts1,
					URL:        "example.com",
					AuthorName: "bob",
					AddedList: []string{
						"v1.sql",
						"v2.sql",
					},
				},
				{
					ID:         "2",
					Title:      "Commit 2",
					Message:    "Update 2",
					CreatedTs:  ts1,
					URL:        "example.com",
					AuthorName: "bob",
					AddedList: []string{
						"v3.sql",
					},
				},
				{
					ID:         "3",
					Title:      "Commit 3",
					Message:    "Update 3",
					CreatedTs:  ts2,
					URL:        "example.com",
					AuthorName: "bob",
					ModifiedList: []string{
						"v3.sql",
						"v4.sql",
					},
				},
				{
					ID:         "4",
					Title:      "Merge branch",
					Message:    "Merge update",
					CreatedTs:  ts3,
					URL:        "example.com",
					AuthorName: "bob",
					AddedList: []string{
						"v1.sql",
						"v2.sql",
						// This file is both added and modified in the commits above.
						// GitLab will treat this as added in the merge commit.
						"v3.sql",
					},
					ModifiedList: []string{"v4.sql"},
				},
			},
			want: []DistinctFileItem{
				{
					CreatedTs: ts3,
					Commit: Commit{
						ID:         "4",
						Title:      "Merge branch",
						Message:    "Merge update",
						CreatedTs:  ts3,
						URL:        "example.com",
						AuthorName: "bob",
						AddedList: []string{
							"v1.sql",
							"v2.sql",
							"v3.sql",
						},
						ModifiedList: []string{"v4.sql"},
					},
					FileName: "v1.sql",
					ItemType: FileItemTypeAdded,
				},
				{
					CreatedTs: ts3,
					Commit: Commit{
						ID:         "4",
						Title:      "Merge branch",
						Message:    "Merge update",
						CreatedTs:  ts3,
						URL:        "example.com",
						AuthorName: "bob",
						AddedList: []string{
							"v1.sql",
							"v2.sql",
							"v3.sql",
						},
						ModifiedList: []string{"v4.sql"},
					},
					FileName: "v2.sql",
					ItemType: FileItemTypeAdded,
				},
				{
					CreatedTs: ts3,
					Commit: Commit{
						ID:         "4",
						Title:      "Merge branch",
						Message:    "Merge update",
						CreatedTs:  ts3,
						URL:        "example.com",
						AuthorName: "bob",
						AddedList: []string{
							"v1.sql",
							"v2.sql",
							"v3.sql",
						},
						ModifiedList: []string{"v4.sql"},
					},
					FileName: "v3.sql",
					ItemType: FileItemTypeAdded,
				},
				{
					CreatedTs: ts3,
					Commit: Commit{
						ID:         "4",
						Title:      "Merge branch",
						Message:    "Merge update",
						CreatedTs:  ts3,
						URL:        "example.com",
						AuthorName: "bob",
						AddedList: []string{
							"v1.sql",
							"v2.sql",
							"v3.sql",
						},
						ModifiedList: []string{"v4.sql"},
					},
					FileName: "v4.sql",
					ItemType: FileItemTypeModified,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := PushEvent{CommitList: tt.commitList}
			got := p.GetDistinctFileList()
			assert.Equal(t, tt.want, got)
		})
	}
}
