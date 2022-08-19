package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
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

func TestValidateGitHubWebhookSignature256(t *testing.T) {
	//nolint:misspell
	const payload = `{"ref":"refs/heads/main","before":"07da1e122bdbb81da8499b6d82c6b6302581a5a7","after":"5a96148ac5ef11a53b838b8cc0d9c929420657f3","repository":{"id":470746482,"node_id":"R_kgDOHA8Fcg","name":"bytebase-test","full_name":"unknwon/bytebase-test","private":true,"owner":{"name":"unknwon","email":"jc@unknwon.io","login":"unknwon","id":2946214,"node_id":"MDQ6VXNlcjI5NDYyMTQ=","avatar_url":"https://avatars.githubusercontent.com/u/2946214?v=4","gravatar_id":"","url":"https://api.github.com/users/unknwon","html_url":"https://github.com/unknwon","followers_url":"https://api.github.com/users/unknwon/followers","following_url":"https://api.github.com/users/unknwon/following{/other_user}","gists_url":"https://api.github.com/users/unknwon/gists{/gist_id}","starred_url":"https://api.github.com/users/unknwon/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/unknwon/subscriptions","organizations_url":"https://api.github.com/users/unknwon/orgs","repos_url":"https://api.github.com/users/unknwon/repos","events_url":"https://api.github.com/users/unknwon/events{/privacy}","received_events_url":"https://api.github.com/users/unknwon/received_events","type":"User","site_admin":false},"html_url":"https://github.com/unknwon/bytebase-test","description":null,"fork":false,"url":"https://github.com/unknwon/bytebase-test","forks_url":"https://api.github.com/repos/unknwon/bytebase-test/forks","keys_url":"https://api.github.com/repos/unknwon/bytebase-test/keys{/key_id}","collaborators_url":"https://api.github.com/repos/unknwon/bytebase-test/collaborators{/collaborator}","teams_url":"https://api.github.com/repos/unknwon/bytebase-test/teams","hooks_url":"https://api.github.com/repos/unknwon/bytebase-test/hooks","issue_events_url":"https://api.github.com/repos/unknwon/bytebase-test/issues/events{/number}","events_url":"https://api.github.com/repos/unknwon/bytebase-test/events","assignees_url":"https://api.github.com/repos/unknwon/bytebase-test/assignees{/user}","branches_url":"https://api.github.com/repos/unknwon/bytebase-test/branches{/branch}","tags_url":"https://api.github.com/repos/unknwon/bytebase-test/tags","blobs_url":"https://api.github.com/repos/unknwon/bytebase-test/git/blobs{/sha}","git_tags_url":"https://api.github.com/repos/unknwon/bytebase-test/git/tags{/sha}","git_refs_url":"https://api.github.com/repos/unknwon/bytebase-test/git/refs{/sha}","trees_url":"https://api.github.com/repos/unknwon/bytebase-test/git/trees{/sha}","statuses_url":"https://api.github.com/repos/unknwon/bytebase-test/statuses/{sha}","languages_url":"https://api.github.com/repos/unknwon/bytebase-test/languages","stargazers_url":"https://api.github.com/repos/unknwon/bytebase-test/stargazers","contributors_url":"https://api.github.com/repos/unknwon/bytebase-test/contributors","subscribers_url":"https://api.github.com/repos/unknwon/bytebase-test/subscribers","subscription_url":"https://api.github.com/repos/unknwon/bytebase-test/subscription","commits_url":"https://api.github.com/repos/unknwon/bytebase-test/commits{/sha}","git_commits_url":"https://api.github.com/repos/unknwon/bytebase-test/git/commits{/sha}","comments_url":"https://api.github.com/repos/unknwon/bytebase-test/comments{/number}","issue_comment_url":"https://api.github.com/repos/unknwon/bytebase-test/issues/comments{/number}","contents_url":"https://api.github.com/repos/unknwon/bytebase-test/contents/{+path}","compare_url":"https://api.github.com/repos/unknwon/bytebase-test/compare/{base}...{head}","merges_url":"https://api.github.com/repos/unknwon/bytebase-test/merges","archive_url":"https://api.github.com/repos/unknwon/bytebase-test/{archive_format}{/ref}","downloads_url":"https://api.github.com/repos/unknwon/bytebase-test/downloads","issues_url":"https://api.github.com/repos/unknwon/bytebase-test/issues{/number}","pulls_url":"https://api.github.com/repos/unknwon/bytebase-test/pulls{/number}","milestones_url":"https://api.github.com/repos/unknwon/bytebase-test/milestones{/number}","notifications_url":"https://api.github.com/repos/unknwon/bytebase-test/notifications{?since,all,participating}","labels_url":"https://api.github.com/repos/unknwon/bytebase-test/labels{/name}","releases_url":"https://api.github.com/repos/unknwon/bytebase-test/releases{/id}","deployments_url":"https://api.github.com/repos/unknwon/bytebase-test/deployments","created_at":1647463607,"updated_at":"2022-03-16T20:46:47Z","pushed_at":1658671596,"git_url":"git://github.com/unknwon/bytebase-test.git","ssh_url":"git@github.com:unknwon/bytebase-test.git","clone_url":"https://github.com/unknwon/bytebase-test.git","svn_url":"https://github.com/unknwon/bytebase-test","homepage":null,"size":20,"stargazers_count":0,"watchers_count":0,"language":null,"has_issues":true,"has_projects":true,"has_downloads":true,"has_wiki":true,"has_pages":false,"forks_count":0,"mirror_url":null,"archived":false,"disabled":false,"open_issues_count":0,"license":{"key":"apache-2.0","name":"Apache License 2.0","spdx_id":"Apache-2.0","url":"https://api.github.com/licenses/apache-2.0","node_id":"MDc6TGljZW5zZTI="},"allow_forking":true,"is_template":false,"web_commit_signoff_required":false,"topics":[],"visibility":"private","forks":0,"open_issues":0,"watchers":0,"default_branch":"main","stargazers":0,"master_branch":"main"},"pusher":{"name":"unknwon","email":"jc@unknwon.io"},"sender":{"login":"unknwon","id":2946214,"node_id":"MDQ6VXNlcjI5NDYyMTQ=","avatar_url":"https://avatars.githubusercontent.com/u/2946214?v=4","gravatar_id":"","url":"https://api.github.com/users/unknwon","html_url":"https://github.com/unknwon","followers_url":"https://api.github.com/users/unknwon/followers","following_url":"https://api.github.com/users/unknwon/following{/other_user}","gists_url":"https://api.github.com/users/unknwon/gists{/gist_id}","starred_url":"https://api.github.com/users/unknwon/starred{/owner}{/repo}","subscriptions_url":"https://api.github.com/users/unknwon/subscriptions","organizations_url":"https://api.github.com/users/unknwon/orgs","repos_url":"https://api.github.com/users/unknwon/repos","events_url":"https://api.github.com/users/unknwon/events{/privacy}","received_events_url":"https://api.github.com/users/unknwon/received_events","type":"User","site_admin":false},"created":false,"deleted":false,"forced":false,"base_ref":null,"compare":"https://github.com/unknwon/bytebase-test/compare/07da1e122bdb...5a96148ac5ef","commits":[{"id":"5a96148ac5ef11a53b838b8cc0d9c929420657f3","tree_id":"8a842b23b62886d2ee12c152eda741cf39b1ceef","distinct":true,"message":"Create testdb_dev__202101131000__baseline__create_tablefoo_for_bar.sql","timestamp":"2022-07-24T22:06:36+08:00","url":"https://github.com/unknwon/bytebase-test/commit/5a96148ac5ef11a53b838b8cc0d9c929420657f3","author":{"name":"Joe Chen","email":"jc@unknwon.io","username":"unknwon"},"committer":{"name":"GitHub","email":"noreply@github.com","username":"web-flow"},"added":["Dev/testdb_dev__202101131000__baseline__create_tablefoo_for_bar.sql"],"removed":[],"modified":[]}],"head_commit":{"id":"5a96148ac5ef11a53b838b8cc0d9c929420657f3","tree_id":"8a842b23b62886d2ee12c152eda741cf39b1ceef","distinct":true,"message":"Create testdb_dev__202101131000__baseline__create_tablefoo_for_bar.sql","timestamp":"2022-07-24T22:06:36+08:00","url":"https://github.com/unknwon/bytebase-test/commit/5a96148ac5ef11a53b838b8cc0d9c929420657f3","author":{"name":"Joe Chen","email":"jc@unknwon.io","username":"unknwon"},"committer":{"name":"GitHub","email":"noreply@github.com","username":"web-flow"},"added":["Dev/testdb_dev__202101131000__baseline__create_tablefoo_for_bar.sql"],"removed":[],"modified":[]}}`

	t.Run("wrong key", func(t *testing.T) {
		got, err := validateGitHubWebhookSignature256(
			"sha256=6bf313c917fd04a3c6c85270bab6c2a6ae40b7ab37767107bf80ad5c6a0a0deb",
			"abadkey",
			[]byte(payload),
		)
		assert.False(t, got)
		assert.NoError(t, err)
	})

	t.Run("wrong signature", func(t *testing.T) {
		got, err := validateGitHubWebhookSignature256(
			"sha256=8335bc69262e94b20753316d844e155ae4d7826a6c89f12e98083ed0dce8d057",
			"bZovosSKsJ8QKCG9",
			[]byte(payload),
		)
		assert.False(t, got)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		got, err := validateGitHubWebhookSignature256(
			"sha256=6bf313c917fd04a3c6c85270bab6c2a6ae40b7ab37767107bf80ad5c6a0a0deb",
			"bZovosSKsJ8QKCG9",
			[]byte(payload),
		)
		assert.True(t, got)
		assert.NoError(t, err)
	})
}

func TestParseBranchNameFromGitHubRefs(t *testing.T) {
	tests := []struct {
		refs   string
		expect string
		err    bool
	}{
		{
			refs:   "refs/heads/main",
			expect: "main",
			err:    false,
		},
		{
			refs:   "refs/heads/feat/heads",
			expect: "feat/heads",
			err:    false,
		},
		// Broken
		{
			refs: "ref",
			err:  true,
		},
		{
			refs:   "refs/heads/",
			expect: "",
			err:    true,
		},
	}
	for _, test := range tests {
		branch, err := parseBranchNameFromGitHubRefs(test.refs)
		if test.err {
			assert.Error(t, err)
		} else {
			assert.Equal(t, test.expect, branch)
		}
	}
}
