package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/tests/fake"
)

func TestSheetVCS(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		vcsProviderCreator fake.VCSProviderCreator
		vcsType            vcs.Type
		externalID         string
		repositoryFullPath string
	}{
		{
			name:               "GitLab",
			vcsProviderCreator: fake.NewGitLab,
			vcsType:            vcs.GitLabSelfHost,
			externalID:         "121",
			repositoryFullPath: "test/schemaUpdate",
		},
		{
			name:               "GitHub",
			vcsProviderCreator: fake.NewGitHub,
			vcsType:            vcs.GitHubCom,
			externalID:         "octocat/Hello-World",
			repositoryFullPath: "octocat/Hello-World",
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
			err := ctl.StartServerWithExternalPg(ctx, &config{
				dataDir:            t.TempDir(),
				vcsProviderCreator: test.vcsProviderCreator,
			})
			a.NoError(err)
			defer func() {
				_ = ctl.Close(ctx)
			}()

			// Create a VCS.
			vcs, err := ctl.createVCS(
				api.VCSCreate{
					Name:          t.Name(),
					Type:          test.vcsType,
					InstanceURL:   ctl.vcsURL,
					APIURL:        ctl.vcsProvider.APIURL(ctl.vcsURL),
					ApplicationID: "testApplicationID",
					Secret:        "testApplicationSecret",
				},
			)
			a.NoError(err)

			// Create a project.
			project, err := ctl.createProject(
				api.ProjectCreate{
					Name: "Test VCS Project",
					Key:  "TestVCSSchemaUpdate",
				},
			)
			a.NoError(err)

			// Create a repository.
			ctl.vcsProvider.CreateRepository(test.externalID)

			// Create the branch
			err = ctl.vcsProvider.CreateBranch(test.externalID, "feature/foo")
			a.NoError(err)

			_, err = ctl.createRepository(
				api.RepositoryCreate{
					VCSID:              vcs.ID,
					ProjectID:          project.ID,
					Name:               "Test Repository",
					FullPath:           test.repositoryFullPath,
					WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, test.repositoryFullPath),
					BranchFilter:       "feature/foo",
					BaseDirectory:      baseDirectory,
					FilePathTemplate:   "{{ENV_NAME}}/{{DB_NAME}}##{{VERSION}}##{{TYPE}}##{{DESCRIPTION}}.sql",
					SchemaPathTemplate: "{{ENV_NAME}}/.{{DB_NAME}}##LATEST.sql",
					SheetPathTemplate:  "sheet/{{NAME}}.sql",
					ExternalID:         test.externalID,
					AccessToken:        "accessToken1",
					RefreshToken:       "refreshToken1",
				},
			)
			a.NoError(err)

			// Initial git files.
			gitFile := "sheet/all_employee.sql"
			fileContent := "SELECT * FROM employee"
			files := map[string]string{
				gitFile: fileContent,
			}
			files[gitFile] = fileContent
			err = ctl.vcsProvider.AddFiles(test.externalID, files)
			a.NoError(err)

			sheetsBefore, err := ctl.listMySheets()
			a.NoError(err)

			err = ctl.syncSheet(project.ID)
			a.NoError(err)

			sheetsAfter, err := ctl.listMySheets()
			a.NoError(err)
			a.Len(sheetsAfter, len(sheetsBefore)+1)

			sheetFromVCS := sheetsAfter[len(sheetsAfter)-1]
			a.Equal("all_employee", sheetFromVCS.Name)
			a.Equal(fileContent, sheetFromVCS.Statement)
		})
	}
}
