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

func TestSheetVCS_GitLab(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServer(ctx, dataDir, fake.NewGitLab, getTestPort(t.Name()))
	a.NoError(err)
	defer ctl.Close(ctx)
	err = ctl.Login()
	a.NoError(err)
	err = ctl.setLicense()
	a.NoError(err)

	// Create a test VCS.
	applicationID := "testApplicationID"
	applicationSecret := "testApplicationSecret"
	vcs, err := ctl.createVCS(api.VCSCreate{
		Name:          "TestVCS_GitLab",
		Type:          vcs.GitLabSelfHost,
		InstanceURL:   ctl.vcsURL,
		APIURL:        ctl.vcsProvider.APIURL(ctl.vcsURL),
		ApplicationID: applicationID,
		Secret:        applicationSecret,
	})
	a.NoError(err)

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name: "Test VCS Project",
		Key:  "TestVCSSyncSheet",
	})
	a.NoError(err)

	// Create a repository.
	repositoryPath := "test/sync-sheet"
	accessToken := "accessToken"
	refreshToken := "refreshToken"
	gitlabProjectID := 121
	gitlabProjectIDStr := fmt.Sprintf("%d", gitlabProjectID)
	// Create a GitLab project.
	ctl.vcsProvider.CreateRepository(gitlabProjectIDStr)
	_, err = ctl.createRepository(api.RepositoryCreate{
		VCSID:              vcs.ID,
		ProjectID:          project.ID,
		Name:               "Test Repository",
		FullPath:           repositoryPath,
		WebURL:             fmt.Sprintf("%s/%s", ctl.vcsURL, repositoryPath),
		BranchFilter:       "feature/foo",
		BaseDirectory:      "bbtest",
		FilePathTemplate:   "{{ENV_NAME}}/{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql",
		SchemaPathTemplate: "{{ENV_NAME}}/.{{DB_NAME}}__LATEST.sql",
		SheetPathTemplate:  "sheet/{{NAME}}.sql",
		ExternalID:         gitlabProjectIDStr,
		AccessToken:        accessToken,
		ExpiresTs:          0,
		RefreshToken:       refreshToken,
	})
	a.NoError(err)

	// Initial git files.
	files := map[string]string{}
	gitFile := "sheet/all_employee.sql"
	fileContent := "SELECT * FROM employee"
	files[gitFile] = fileContent
	err = ctl.vcsProvider.AddFiles(gitlabProjectIDStr, files)
	a.NoError(err)

	err = ctl.syncSheet(project.ID)
	a.NoError(err)

	sheets, err := ctl.listSheets(api.SheetFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Len(sheets, 1)

	sheetFromVCS := sheets[0]
	a.Equal("all_employee", sheetFromVCS.Name)
	a.Equal(fileContent, sheetFromVCS.Statement)
}

func TestSheetVCS_GitHub(t *testing.T) {
	t.Parallel()

	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	err := ctl.StartServer(ctx, t.TempDir(), fake.NewGitHub, getTestPort(t.Name()))
	a.NoError(err)
	defer func() { _ = ctl.Close(ctx) }()

	err = ctl.Login()
	a.NoError(err)
	err = ctl.setLicense()
	a.NoError(err)

	// Create a test VCS.
	clientID := "testClientID"
	clientSecret := "testClientSecret"
	vcs, err := ctl.createVCS(
		api.VCSCreate{
			Name:          "TestVCS_GitHub",
			Type:          vcs.GitHubCom,
			InstanceURL:   ctl.vcsURL,
			APIURL:        ctl.vcsProvider.APIURL(ctl.vcsURL),
			ApplicationID: clientID,
			Secret:        clientSecret,
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

	// Create a GitHub repository.
	accessToken := "accessToken1"
	refreshToken := "refreshToken1"
	repositoryFullName := "octocat/Hello-World"
	repositoryHTMLURL := fmt.Sprintf("%s/%s", ctl.vcsURL, repositoryFullName)
	ctl.vcsProvider.CreateRepository(repositoryFullName)
	_, err = ctl.createRepository(
		api.RepositoryCreate{
			VCSID:              vcs.ID,
			ProjectID:          project.ID,
			Name:               "Test Repository",
			FullPath:           repositoryFullName,
			WebURL:             repositoryHTMLURL,
			BranchFilter:       "main",
			BaseDirectory:      baseDirectory,
			FilePathTemplate:   "{{ENV_NAME}}/{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql",
			SchemaPathTemplate: "{{ENV_NAME}}/.{{DB_NAME}}__LATEST.sql",
			SheetPathTemplate:  "sheet/{{NAME}}.sql",
			ExternalID:         repositoryFullName,
			AccessToken:        accessToken,
			RefreshToken:       refreshToken,
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
	err = ctl.vcsProvider.AddFiles(repositoryFullName, files)
	a.NoError(err)

	err = ctl.syncSheet(project.ID)
	a.NoError(err)

	sheets, err := ctl.listSheets(api.SheetFind{ProjectID: &project.ID})
	a.NoError(err)
	a.Len(sheets, 1)

	sheetFromVCS := sheets[0]
	a.Equal("all_employee", sheetFromVCS.Name)
	a.Equal(fileContent, sheetFromVCS.Statement)
}
