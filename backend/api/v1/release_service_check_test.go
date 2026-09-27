package v1

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

// TestCheckReleaseDatabaseTarget pins how a release check confines its
// database targets to the project it runs in. A target naming another project
// is malformed for this request; everything else that is not the project's
// own live database under its canonical name is simply not found, so the
// answer never says what lives in a project the caller cannot see. That the
// check runs end to end on a project instance is TestProjectInstanceWorkflowTargets
// in backend/tests.
func TestCheckReleaseDatabaseTarget(t *testing.T) {
	projectA, projectB := "project-a", "project-b"
	canonical := common.FormatProjectDatabase(projectA, "project-instance", "app")
	owned := func() *store.DatabaseMessage {
		return &store.DatabaseMessage{ProjectID: projectA, InstanceID: "project-instance", DatabaseName: "app", InstanceProjectID: &projectA}
	}
	live := &store.InstanceMessage{ResourceID: "project-instance", ProjectID: &projectA}

	require.Equal(t, connect.CodeInvalidArgument,
		connect.CodeOf(checkReleaseTargetProject(projectA, common.FormatProjectDatabase(projectB, "project-instance", "app"), &projectB)))
	require.NoError(t, checkReleaseTargetProject(projectA, canonical, &projectA))
	require.NoError(t, checkReleaseTargetProject(projectA, common.FormatDatabase("workspace-instance", "shared"), nil))

	require.NoError(t, checkReleaseDatabase(projectA, canonical, owned()))
	require.NoError(t, checkReleaseDatabaseInstance(canonical, owned(), live))

	for _, tc := range []struct {
		name     string
		target   string
		database *store.DatabaseMessage
		instance *store.InstanceMessage
	}{
		{"the workspace form of a project instance's database", common.FormatDatabase("project-instance", "app"), owned(), live},
		{"a database in another project", canonical, &store.DatabaseMessage{ProjectID: projectB, InstanceID: "project-instance", DatabaseName: "app", InstanceProjectID: &projectA}, live},
		{"an archived instance", canonical, owned(), &store.InstanceMessage{ResourceID: "project-instance", ProjectID: &projectA, Deleted: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkReleaseDatabase(projectA, tc.target, tc.database)
			if err == nil {
				err = checkReleaseDatabaseInstance(tc.target, tc.database, tc.instance)
			}
			require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
			require.EqualError(t, err, "not_found: database "+tc.target+" not found")
		})
	}
}
