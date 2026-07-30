//nolint:revive
package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestGetInstanceDatabaseID(t *testing.T) {
	instanceID, err := GetInstanceID("instances/i2")
	require.NoError(t, err)
	require.Equal(t, "i2", instanceID)

	_, err = GetInstanceID("instances/i2/databases/d3")
	require.Error(t, err)
}

func TestWorkspaceInstanceResourceNames(t *testing.T) {
	instance := FormatInstance("instance-1")
	require.Equal(t, "instances/instance-1", instance)
	instanceID, err := GetInstanceID(instance)
	require.NoError(t, err)
	require.Equal(t, "instance-1", instanceID)

	database := FormatDatabase("instance-1", "database-1")
	require.Equal(t, "instances/instance-1/databases/database-1", database)
	instanceID, databaseID, err := GetInstanceDatabaseID(database)
	require.NoError(t, err)
	require.Equal(t, "instance-1", instanceID)
	require.Equal(t, "database-1", databaseID)

	revision := FormatRevision("instance-1", "database-1", "revision-1")
	require.Equal(t, "instances/instance-1/databases/database-1/revisions/revision-1", revision)
	instanceID, databaseID, revisionID, err := GetInstanceDatabaseRevisionID(revision)
	require.NoError(t, err)
	require.Equal(t, "instance-1", instanceID)
	require.Equal(t, "database-1", databaseID)
	require.Equal(t, "revision-1", revisionID)

	changelog := FormatChangelog("instance-1", "database-1", "changelog-1")
	require.Equal(t, "instances/instance-1/databases/database-1/changelogs/changelog-1", changelog)
	instanceID, databaseID, changelogID, err := GetInstanceDatabaseChangelogID(changelog)
	require.NoError(t, err)
	require.Equal(t, "instance-1", instanceID)
	require.Equal(t, "database-1", databaseID)
	require.Equal(t, "changelog-1", changelogID)
}

func TestProjectInstanceResourceNames(t *testing.T) {
	instance := FormatProjectInstance("project-1", "instance-1")
	require.Equal(t, "projects/project-1/instances/instance-1", instance)
	projectID, instanceID, err := GetProjectIDInstanceID(instance)
	require.NoError(t, err)
	require.Equal(t, "project-1", projectID)
	require.Equal(t, "instance-1", instanceID)

	database := FormatProjectDatabase("project-1", "instance-1", "database-1")
	require.Equal(t, "projects/project-1/instances/instance-1/databases/database-1", database)
	projectID, instanceID, databaseID, err := GetProjectIDInstanceDatabaseID(database)
	require.NoError(t, err)
	require.Equal(t, "project-1", projectID)
	require.Equal(t, "instance-1", instanceID)
	require.Equal(t, "database-1", databaseID)

	revision := FormatProjectRevision("project-1", "instance-1", "database-1", "revision-1")
	require.Equal(t, "projects/project-1/instances/instance-1/databases/database-1/revisions/revision-1", revision)
	projectID, instanceID, databaseID, revisionID, err := GetProjectIDInstanceDatabaseRevisionID(revision)
	require.NoError(t, err)
	require.Equal(t, "project-1", projectID)
	require.Equal(t, "instance-1", instanceID)
	require.Equal(t, "database-1", databaseID)
	require.Equal(t, "revision-1", revisionID)

	changelog := FormatProjectChangelog("project-1", "instance-1", "database-1", "changelog-1")
	require.Equal(t, "projects/project-1/instances/instance-1/databases/database-1/changelogs/changelog-1", changelog)
	projectID, instanceID, databaseID, changelogID, err := GetProjectIDInstanceDatabaseChangelogID(changelog)
	require.NoError(t, err)
	require.Equal(t, "project-1", projectID)
	require.Equal(t, "instance-1", instanceID)
	require.Equal(t, "database-1", databaseID)
	require.Equal(t, "changelog-1", changelogID)
}

func TestInstanceResourceNamesRejectWrongScopeAndMalformedNames(t *testing.T) {
	projectInstance := "projects/project-1/instances/instance-1"
	projectDatabase := projectInstance + "/databases/database-1"
	projectRevision := projectDatabase + "/revisions/revision-1"
	projectChangelog := projectDatabase + "/changelogs/changelog-1"

	_, err := GetInstanceID(projectInstance)
	require.Error(t, err)
	_, _, err = GetInstanceDatabaseID(projectDatabase)
	require.Error(t, err)
	_, _, _, err = GetInstanceDatabaseRevisionID(projectRevision)
	require.Error(t, err)
	_, _, _, err = GetInstanceDatabaseChangelogID(projectChangelog)
	require.Error(t, err)

	_, _, err = GetProjectIDInstanceID("instances/instance-1")
	require.Error(t, err)
	_, _, _, err = GetProjectIDInstanceDatabaseID("instances/instance-1/databases/database-1")
	require.Error(t, err)
	_, _, _, _, err = GetProjectIDInstanceDatabaseRevisionID("instances/instance-1/databases/database-1/revisions/revision-1")
	require.Error(t, err)
	_, _, _, _, err = GetProjectIDInstanceDatabaseChangelogID("instances/instance-1/databases/database-1/changelogs/changelog-1")
	require.Error(t, err)

	for _, name := range []string{
		"projects/project-1/instances",
		"projects//instances/instance-1",
		"projects/project-1/instances/instance-1/",
		"projects/project-1/instances/instance-1/databases/database-1/revisions",
		"projects/project-1/instances/instance-1/databases/database-1/changelogs/",
	} {
		_, _, err := GetProjectIDInstanceID(name)
		require.Error(t, err, name)
		_, _, _, err = GetProjectIDInstanceDatabaseID(name)
		require.Error(t, err, name)
		_, _, _, _, err = GetProjectIDInstanceDatabaseRevisionID(name)
		require.Error(t, err, name)
		_, _, _, _, err = GetProjectIDInstanceDatabaseChangelogID(name)
		require.Error(t, err, name)
	}
}

func TestGetPolicyResourceTypeAndResourceForInstanceAndDatabase(t *testing.T) {
	for _, test := range []struct {
		name         string
		resourceType storepb.Policy_Resource
	}{
		{
			name:         "instances/instance-1",
			resourceType: storepb.Policy_INSTANCE,
		},
		{
			name:         "instances/instance-1/databases/database-1",
			resourceType: storepb.Policy_DATABASE,
		},
		{
			name:         "projects/project-1/instances/instance-1",
			resourceType: storepb.Policy_INSTANCE,
		},
		{
			name:         "projects/project-1/instances/instance-1/databases/database-1",
			resourceType: storepb.Policy_DATABASE,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			resourceType, resource, err := GetPolicyResourceTypeAndResource(test.name)
			require.NoError(t, err)
			require.Equal(t, test.resourceType, resourceType)
			require.Equal(t, test.name, *resource)
		})
	}

	for _, name := range []string{
		"instances/instance-1/databases",
		"instances//databases/database-1",
		"projects/project-1/instances/instance-1/databases",
		"projects/project-1/instances//databases/database-1",
		"projects/project-1/instances/instance-1/databases/database-1/revisions/revision-1",
	} {
		_, _, err := GetPolicyResourceTypeAndResource(name)
		require.Error(t, err, name)
	}
}

func TestGetProjectIDQueryHistoryID(t *testing.T) {
	projectID, historyID, err := GetProjectIDQueryHistoryID("projects/p1/queryHistories/550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	require.Equal(t, "p1", projectID)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", historyID)

	_, _, err = GetProjectIDQueryHistoryID("projects/p1/queryHistories")
	require.Error(t, err)

	_, _, err = GetProjectIDQueryHistoryID("queryHistories/h1")
	require.Error(t, err)
}
