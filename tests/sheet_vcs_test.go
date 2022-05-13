package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/stretchr/testify/require"
)

func TestSheetVCS(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServer(ctx, dataDir, getTestPort(t.Name()))
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
		Name:          "TestVCS",
		Type:          vcs.GitLabSelfHost,
		InstanceURL:   ctl.gitURL,
		APIURL:        ctl.gitAPIURL,
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
	// create a gitlab project.
	ctl.gitlab.CreateProject(gitlabProjectIDStr)
	_, err = ctl.createRepository(api.RepositoryCreate{
		VCSID:              vcs.ID,
		ProjectID:          project.ID,
		Name:               "Test Repository",
		FullPath:           repositoryPath,
		WebURL:             fmt.Sprintf("%s/%s", ctl.gitURL, repositoryPath),
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
	err = ctl.gitlab.AddFiles(gitlabProjectIDStr, files)
	a.NoError(err)

	err = ctl.syncSheet(project.ID)
	a.NoError(err)

	sheets, err := ctl.listSheets(api.SheetFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Equal(1, len(sheets))

	sheetFromVCS := sheets[0]
	a.Equal("all_employee", sheetFromVCS.Name)
	a.Equal(fileContent, sheetFromVCS.Statement)
}
