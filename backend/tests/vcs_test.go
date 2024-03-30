package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/gitlab"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestVCS(t *testing.T) {
	branchName := "feature/foo"
	pullRequestFiles := []*vcs.PullRequestFile{
		{Path: "bbtest/0001##migrate##😊create_table_book1.sql"},
		{Path: "bbtest/0002##migrate##新建create_table_book2.sql"},
	}
	fileContentMap := map[string]string{
		pullRequestFiles[0].Path: migrationStatement1,
		pullRequestFiles[1].Path: migrationStatement2,
	}
	pullRequestID := 2250
	pullRequestTitle := "TestVCS"
	pullRequestDescription := "TestVCS fun."

	tests := []struct {
		name               string
		vcsProviderCreator fake.VCSProviderCreator
		vcsType            v1pb.VCSProvider_Type
		externalID         string
		repositoryFullPath string
		webhookPushEvent   any
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            v1pb.VCSProvider_GITLAB,
			externalID:         "121",
			repositoryFullPath: "test/vcs",
			webhookPushEvent: gitlab.MergeRequestPushEvent{
				ObjectKind: "merge_request",
				ObjectAttributes: gitlab.EventObjectAttributes{
					IID:          pullRequestID,
					URL:          "https://gitlab.com/test/vcs/-/merge_requests/2250",
					TargetBranch: branchName,
					Action:       "merge",
					Title:        pullRequestTitle,
					Description:  pullRequestDescription,
					LastCommit: gitlab.LastCommit{
						ID: "cc63b0592388a7ab1b05b005ad8c8dc14ce432b1",
					},
				},
			},
		},
	}
	for _, test := range tests {
		// Fix the problem that closure in a for loop will always use the last element.
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			a := require.New(t)
			ctx := context.Background()
			ctl := &controller{}
			ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:            t.TempDir(),
				vcsProviderCreator: test.vcsProviderCreator,
			})
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

			evcs, err := ctl.evcsClient.CreateVCSProvider(ctx, &v1pb.CreateVCSProviderRequest{
				VcsProvider: &v1pb.VCSProvider{
					Title:       t.Name(),
					Type:        test.vcsType,
					Url:         ctl.vcsURL,
					AccessToken: "testApplicationSecret",
				},
				VcsProviderId: strings.ToLower(test.vcsType.String()),
			})
			a.NoError(err)
			ctl.vcsProvider.CreateRepository(test.externalID)
			err = ctl.vcsProvider.CreateBranch(test.externalID, branchName)
			a.NoError(err)

			oldVcsConnector, err := ctl.vcsConnectorServiceClient.CreateVCSConnector(ctx, &v1pb.CreateVCSConnectorRequest{
				Parent: ctl.project.Name,
				VcsConnector: &v1pb.VCSConnector{
					VcsProvider:   evcs.Name,
					Title:         "Test VCS Connector",
					FullPath:      test.repositoryFullPath,
					WebUrl:        fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					Branch:        branchName,
					BaseDirectory: baseDirectory + "+invalid",
					ExternalId:    test.externalID,
				},
				VcsConnectorId: "default",
			})
			a.NoError(err)
			vcsConnector, err := ctl.vcsConnectorServiceClient.UpdateVCSConnector(ctx, &v1pb.UpdateVCSConnectorRequest{
				VcsConnector: &v1pb.VCSConnector{
					Name:          oldVcsConnector.Name,
					BaseDirectory: baseDirectory,
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"base_directory"}},
			})
			a.NoError(err)
			a.Equal(baseDirectory, vcsConnector.BaseDirectory)

			instanceName := "testInstance"
			instanceDir, err := ctl.provisionSQLiteInstance(t.TempDir(), instanceName)
			a.NoError(err)

			instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
				InstanceId: generateRandomString("instance", 10),
				Instance: &v1pb.Instance{
					Title:       instanceName,
					Engine:      v1pb.Engine_SQLITE,
					Environment: "environments/prod",
					Activation:  true,
					DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
				},
			})
			a.NoError(err)

			databaseName := "testVCS"
			err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, databaseName, "", nil /* labelMap */)
			a.NoError(err)
			database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
				Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
			})
			a.NoError(err)

			err = ctl.vcsProvider.AddPullRequest(test.externalID, pullRequestID, pullRequestFiles)
			a.NoError(err)
			err = ctl.vcsProvider.AddFiles(test.externalID, fileContentMap)
			a.NoError(err)

			payload, err := json.Marshal(test.webhookPushEvent)
			a.NoError(err)
			err = ctl.vcsProvider.SendWebhookPush(test.externalID, payload)
			a.NoError(err)

			// Get schema update issue.
			issue, err := ctl.getLastOpenIssue(ctx, ctl.project)
			a.NoError(err)
			a.NotNil(issue)
			err = ctl.waitRollout(ctx, issue.Name, issue.Rollout)
			a.NoError(err)
			// TODO(d): use pull requst.
			a.Equal(pullRequestTitle, issue.Title)
			a.Equal(pullRequestDescription, issue.Description)
			err = ctl.closeIssue(ctx, ctl.project, issue.Name)
			a.NoError(err)

			// Query schema.
			dbMetadata, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{Name: fmt.Sprintf("%s/schema", database.Name)})
			a.NoError(err)
			a.Equal(want2BookSchema, dbMetadata.Schema)

			// Get migration history.
			resp, err := ctl.databaseServiceClient.ListChangeHistories(ctx, &v1pb.ListChangeHistoriesRequest{
				Parent: database.Name,
				View:   v1pb.ChangeHistoryView_CHANGE_HISTORY_VIEW_FULL,
			})
			a.NoError(err)
			histories := resp.ChangeHistories
			wantHistories := []*v1pb.ChangeHistory{
				{
					Type:       v1pb.ChangeHistory_MIGRATE,
					Status:     v1pb.ChangeHistory_DONE,
					Schema:     dumpedSchema2,
					PrevSchema: dumpedSchema,
					Version:    "0002",
				},
				{
					Type:       v1pb.ChangeHistory_MIGRATE,
					Status:     v1pb.ChangeHistory_DONE,
					Schema:     dumpedSchema,
					PrevSchema: "",
					Version:    "0001",
				},
			}
			a.Equal(len(wantHistories), len(histories))
			for i, history := range histories {
				got := &v1pb.ChangeHistory{
					Type:       history.Type,
					Status:     history.Status,
					Schema:     history.Schema,
					PrevSchema: history.PrevSchema,
					Version:    history.Version,
				}
				want := wantHistories[i]
				a.Equal(got, want)
			}

			_, err = ctl.vcsConnectorServiceClient.DeleteVCSConnector(ctx, &v1pb.DeleteVCSConnectorRequest{Name: vcsConnector.Name})
			a.NoError(err)
		})
	}
}
