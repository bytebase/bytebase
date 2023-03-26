package server

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// map from project ID to the map of <role, principal ID>.
var testProjectMemberMap = map[int]map[common.ProjectRole]int{
	100: {
		common.ProjectOwner:     200,
		common.ProjectDeveloper: 201,
	},
	101: {
		common.ProjectOwner: 202,
	},
}

// map from sheet ID to the project ID.
var testSheetProjectMap = map[int]int{
	1000: 100,
	1001: 100,
	1002: 100,
}

// map from sheet ID to the map of <string, principal ID>
// string can be one of the project roles or "CREATOR" which represents the sheet creator.
var testSheetMemberMap = map[int]map[string]int{
	1000: {
		"OWNER":     200,
		"DEVELOPER": 201,
		"CREATOR":   202,
	},
	1001: {
		"OWNER":     200,
		"DEVELOPER": 201,
		"CREATOR":   202,
	},
	1002: {
		"OWNER":     200,
		"DEVELOPER": 201,
		"CREATOR":   202,
	},
}

func testFindPrincipalIDFromProject(projectID int, role common.ProjectRole) int {
	m, ok := testProjectMemberMap[projectID]
	if !ok {
		return 999
	}

	id, ok := m[role]
	if !ok {
		return 999
	}
	return id
}

func testFindPrincipalIDFromSheet(sheetID int, v string) int {
	m, ok := testSheetMemberMap[sheetID]
	if !ok {
		return 999
	}

	id, ok := m[v]
	if !ok {
		return 999
	}
	return id
}

var roleFinder = func(projectID int, principalID int) (common.ProjectRole, error) {
	m, ok := testProjectMemberMap[projectID]
	if !ok {
		return "", nil
	}

	for role, id := range m {
		if id == principalID {
			return role, nil
		}
	}
	return "", nil
}

var sheetFinder = func(sheetID int) (*api.Sheet, error) {
	switch sheetID {
	case 1000:
		return &api.Sheet{
			ID:         1000,
			CreatorID:  testFindPrincipalIDFromSheet(1000, "CREATOR"),
			ProjectID:  testSheetProjectMap[1000],
			Visibility: api.PrivateSheet,
		}, nil
	case 1001:
		return &api.Sheet{
			ID:         1001,
			CreatorID:  testFindPrincipalIDFromSheet(1001, "CREATOR"),
			ProjectID:  testSheetProjectMap[1001],
			Visibility: api.ProjectSheet,
		}, nil
	case 1002:
		return &api.Sheet{
			ID:         1002,
			CreatorID:  testFindPrincipalIDFromSheet(1002, "CREATOR"),
			ProjectID:  testSheetProjectMap[1002],
			Visibility: api.PublicSheet,
		}, nil
	}
	return nil, nil
}

var testProjectDatabaseMemberMap = struct {
	// projectToPolicy is a map from project ID to the map of <principal ID, role>.
	projectToPolicy map[int]map[int]common.ProjectRole
	// projectOwnsDatabase is a map from project ID to the map of <database ID, true>.
	projectOwnsDatabase map[int]map[int]bool
}{
	projectToPolicy: map[int]map[int]common.ProjectRole{
		// Default Project contains no member.
		1: {},
		// Project 101 contains no member.
		101: {},
		// Project 102 contains member 201 as developer.
		102: {
			201: common.ProjectDeveloper,
		},
	},
	projectOwnsDatabase: map[int]map[int]bool{
		// Default Project owns database 301.
		1: {
			301: true,
		},
		// Project 101 owns database 302.
		101: {
			302: true,
		},
		// Project 102 owns database 303.
		102: {
			303: true,
		},
	},
}

var isMemberOfAnyProjectOwnsDatabase = func(principalID int, databaseID int) (bool, error) {
	// Get the ID of the project owns the database.
	var projectID int
	for id, databaseMap := range testProjectDatabaseMemberMap.projectOwnsDatabase {
		if _, ok := databaseMap[databaseID]; ok {
			projectID = id
			break
		}
	}

	// Default project contains no member.
	if projectID == 1 {
		return false, nil
	}

	// Check if the principal is a member of the project.
	m, ok := testProjectDatabaseMemberMap.projectToPolicy[projectID]
	if !ok {
		return false, nil
	}
	for id := range m {
		if id == principalID {
			return true, nil
		}
	}
	return false, nil
}

var testMemberIssueMap = struct {
	// principalIDToProjectID is a map from principal ID to the project ID.
	// We assume that a principal can only be a member of one project (actually it can be a member of multiple projects).
	principalIDToProjectID map[int]int
	// issueIDToProjectID is a map from issue ID to the project ID.
	issueIDToProjectID map[int]int
	// issueIDToCreatorID is a map from issue ID to the creator ID.
	issueIDToCreatorID map[int]int
}{
	principalIDToProjectID: map[int]int{
		// User 202 is a member of project 102.
		202: 102,
		// User 203 is a member of project 102.
		203: 102,
		// User 203 is a member of project 103.
		204: 103,
	},
	issueIDToProjectID: map[int]int{
		// Issue 401 belongs to project 102.
		401: 102,
		// Issue 402 belongs to project 102.
		402: 102,
		// Issue 403 belongs to project 103.
		403: 103,
	},
	issueIDToCreatorID: map[int]int{
		// Issue 401 is created by user 202.
		401: 202,
		// Issue 402 is created by user 203.
		402: 203,
		// Issue 403 is created by user 204.
		403: 204,
	},
}

// If the workspace developer principal is one of the following, it can update the issue:
// 1. The creator of the issue.
// 2. The member of the project that the issue belongs to.
var canWorkspaceDeveloperUpdateIssue = func(issueID int, principalID int) error {
	if creatorID, ok := testMemberIssueMap.issueIDToCreatorID[issueID]; !ok {
		return errors.Errorf("issue %d does not exist", issueID)
	} else if creatorID == principalID {
		return nil
	}

	issueProjectIDBelongTo, ok := testMemberIssueMap.issueIDToProjectID[issueID]
	if !ok {
		return errors.Errorf("issue %d does not belong to any project", issueID)
	}

	principalProjectIDBelongTo, ok := testMemberIssueMap.principalIDToProjectID[principalID]
	if !ok {
		return errors.Errorf("user %d is not a member of any project", principalID)
	}
	if issueProjectIDBelongTo != principalProjectIDBelongTo {
		return errors.Errorf("user %d is not a member of issue %d", principalID, issueID)
	}
	return nil
}
