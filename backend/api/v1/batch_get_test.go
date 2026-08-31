package v1

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestResolveBatchGet(t *testing.T) {
	testCases := []struct {
		name          string
		names         []string
		absent        map[string]bool
		failOn        string
		wantResources []string
		wantUnmatched []string
		wantErr       bool
	}{
		{
			name:          "returns resources in request order, not lookup order",
			names:         []string{"c", "a", "b"},
			wantResources: []string{"c", "a", "b"},
		},
		{
			name:          "names that resolve to nothing are reported, not dropped",
			names:         []string{"a", "gone", "b", "hidden"},
			absent:        map[string]bool{"gone": true, "hidden": true},
			wantResources: []string{"a", "b"},
			wantUnmatched: []string{"gone", "hidden"},
		},
		{
			name:          "a repeated name is served once, from its first occurrence",
			names:         []string{"a", "b", "a"},
			wantResources: []string{"a", "b"},
		},
		{
			name:          "a repeated unmatched name is reported once",
			names:         []string{"gone", "a", "gone"},
			absent:        map[string]bool{"gone": true},
			wantResources: []string{"a"},
			wantUnmatched: []string{"gone"},
		},
		{
			name:    "a lookup failure fails the batch instead of reading as absent",
			names:   []string{"a", "boom", "b"},
			failOn:  "boom",
			wantErr: true,
		},
		{
			name:  "no names",
			names: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := require.New(t)
			var looked []string
			resources, unmatched, err := resolveBatchGet(tc.names, func(name string) (string, bool, error) {
				looked = append(looked, name)
				if name == tc.failOn {
					return "", false, errors.Errorf("store is down")
				}
				if tc.absent[name] {
					return "", false, nil
				}
				return name, true, nil
			})
			if tc.wantErr {
				a.Error(err)
				a.Nil(resources)
				a.Nil(unmatched)
				return
			}
			a.NoError(err)
			a.Equal(tc.wantResources, emptyToNil(resources))
			a.Equal(tc.wantUnmatched, unmatched)
			// Every distinct name is looked up exactly once.
			a.Len(looked, len(uniqueNames(tc.names)))
		})
	}
}

func emptyToNil(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

func uniqueNames(names []string) map[string]bool {
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		seen[name] = true
	}
	return seen
}

func TestValidateDatabaseNameParent(t *testing.T) {
	projectA, projectB, instanceX := "project-a", "project-b", "instance-x"

	testCases := []struct {
		name       string
		projectID  *string
		instanceID string
		parent     *databaseParent
		wantErr    bool
	}{
		{
			name:       "wildcard parent accepts any name",
			projectID:  &projectA,
			instanceID: instanceX,
			parent:     &databaseParent{},
		},
		{
			name:       "name matching its parent",
			projectID:  &projectA,
			instanceID: instanceX,
			parent:     &databaseParent{projectID: &projectA, instanceID: &instanceX},
		},
		{
			name:       "name naming a different project than the parent",
			projectID:  &projectB,
			instanceID: instanceX,
			parent:     &databaseParent{projectID: &projectA},
			wantErr:    true,
		},
		{
			name:       "name naming a different instance than the parent",
			projectID:  &projectA,
			instanceID: "instance-y",
			parent:     &databaseParent{instanceID: &instanceX},
			wantErr:    true,
		},
		{
			// The name says nothing about a project, so nothing in the request
			// contradicts itself. Whether the stored database sits under the
			// parent is databaseInParent's call, and an unmatched name there.
			name:       "instance-only name under a project parent",
			projectID:  nil,
			instanceID: instanceX,
			parent:     &databaseParent{projectID: &projectA},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := require.New(t)
			err := validateDatabaseNameParent(tc.projectID, tc.instanceID, "db", tc.parent)
			if tc.wantErr {
				a.Error(err)
				return
			}
			a.NoError(err)
		})
	}
}

func TestDatabaseInParent(t *testing.T) {
	a := require.New(t)
	projectA, projectB := "project-a", "project-b"
	database := &store.DatabaseMessage{ProjectID: projectA}

	a.True(databaseInParent(database, &databaseParent{}), "wildcard parent matches")
	a.True(databaseInParent(database, &databaseParent{projectID: &projectA}))
	a.False(databaseInParent(database, &databaseParent{projectID: &projectB}))
}

func TestValidateDatabaseParentStillRejectsAForeignDatabase(t *testing.T) {
	// The write paths keep the strict check: BatchSyncDatabases and
	// BatchUpdateDatabases must not act on a database outside the parent.
	a := require.New(t)
	projectA, projectB, instanceX := "project-a", "project-b", "instance-x"

	err := validateDatabaseParent(nil, instanceX, "db", &store.DatabaseMessage{ProjectID: projectB}, &databaseParent{projectID: &projectA})
	a.Error(err)

	err = validateDatabaseParent(nil, instanceX, "db", &store.DatabaseMessage{ProjectID: projectA}, &databaseParent{projectID: &projectA})
	a.NoError(err)
}

func TestBatchGetLookupNames(t *testing.T) {
	a := require.New(t)

	// The names a BatchGet looks up are the ones it may report as unmatched, so
	// resource resolution must not fail the batch on them.
	a.Equal(
		map[string]bool{"projects/a": true, "projects/b": true},
		batchGetLookupNames(
			&v1pb.BatchGetProjectsRequest{Names: []string{"projects/a", "projects/b"}},
			"/bytebase.v1.ProjectService/BatchGetProjects",
		),
	)

	// The parent is the scope of the call, not one of the looked-up names: a
	// parent that does not resolve stays the caller's error.
	lookup := batchGetLookupNames(
		&v1pb.BatchGetDatabasesRequest{
			Parent: "projects/p",
			Names:  []string{"instances/i/databases/d"},
		},
		"/bytebase.v1.DatabaseService/BatchGetDatabases",
	)
	a.True(lookup["instances/i/databases/d"])
	a.False(lookup["projects/p"])

	// Nothing else gets the leniency.
	a.Nil(batchGetLookupNames(
		&v1pb.BatchDeleteProjectsRequest{Names: []string{"projects/a"}},
		"/bytebase.v1.ProjectService/BatchDeleteProjects",
	))
	a.Nil(batchGetLookupNames(
		&v1pb.GetProjectRequest{Name: "projects/a"},
		"/bytebase.v1.ProjectService/GetProject",
	))
}
