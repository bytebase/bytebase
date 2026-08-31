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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := require.New(t)
			resources, unmatched, err := resolveBatchGet(tc.names, func(name string) (string, bool, error) {
				switch {
				case name == tc.failOn:
					return "", false, errors.Errorf("store is down")
				case tc.absent[name]:
					return "", false, nil
				default:
					return name, true, nil
				}
			})
			if tc.wantErr {
				a.Error(err)
				return
			}
			a.NoError(err)
			a.Equal(tc.wantResources, resources)
			a.Equal(tc.wantUnmatched, unmatched)
		})
	}
}

// BatchGetDatabases splits the parent check in two: the half decidable from the
// request alone stays an error, and the half that reads the stored row becomes
// an unmatched name, so the response cannot say "it exists, just not here".
func TestDatabaseParentChecks(t *testing.T) {
	a := require.New(t)
	projectA, projectB, instanceX := "project-a", "project-b", "instance-x"
	inA := &store.DatabaseMessage{ProjectID: projectA}

	// Decidable from the name alone: the caller contradicting itself.
	a.Error(validateDatabaseNameParent(&projectB, instanceX, "db", &databaseParent{projectID: &projectA}))
	a.Error(validateDatabaseNameParent(&projectA, "instance-y", "db", &databaseParent{instanceID: &instanceX}))
	a.NoError(validateDatabaseNameParent(&projectA, instanceX, "db", &databaseParent{projectID: &projectA, instanceID: &instanceX}))
	a.NoError(validateDatabaseNameParent(&projectA, instanceX, "db", &databaseParent{}))
	// An instance-only name says nothing about a project, so nothing here
	// contradicts a project parent — that is databaseInParent's call.
	a.NoError(validateDatabaseNameParent(nil, instanceX, "db", &databaseParent{projectID: &projectA}))

	// Read from the stored row.
	a.True(databaseInParent(inA, &databaseParent{}))
	a.True(databaseInParent(inA, &databaseParent{projectID: &projectA}))
	a.False(databaseInParent(inA, &databaseParent{projectID: &projectB}))

	// The write paths keep the strict check: BatchSyncDatabases and
	// BatchUpdateDatabases must not act on a database outside the parent.
	a.Error(validateDatabaseParent(nil, instanceX, "db", &store.DatabaseMessage{ProjectID: projectB}, &databaseParent{projectID: &projectA}))
	a.NoError(validateDatabaseParent(nil, instanceX, "db", inA, &databaseParent{projectID: &projectA}))
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
