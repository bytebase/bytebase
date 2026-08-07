package aiaccess

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
)

// forbiddenMethods are the methods no ceiling may admit, mapped to the reason.
//
// Every entry is a credential endpoint: it mints, exchanges, or resets the
// credential that authenticates a caller. They carry no IAM permission and
// never can — you cannot require a credential to obtain one — so denial cannot
// be expressed by leaving a capability out of a set. It has to be unconditional.
//
// This is the whole list. Methods that are merely dangerous (AdminExecute,
// service-account key rotation, IAM policy writes) are not here: they carry
// real permissions, and they are denied by those permissions being in no set.
var forbiddenMethods = map[string]string{
	"/bytebase.v1.AuthService/ExchangeToken":        "mints an access token",
	"/bytebase.v1.AuthService/Login":                "mints a session credential",
	"/bytebase.v1.AuthService/Logout":               "revokes session credentials",
	"/bytebase.v1.AuthService/Refresh":              "mints a fresh access token",
	"/bytebase.v1.AuthService/RequestPasswordReset": "issues a password reset token",
	"/bytebase.v1.AuthService/ResetPassword":        "changes a login credential",
	"/bytebase.v1.AuthService/SendEmailLoginCode":   "issues a login code",
	"/bytebase.v1.AuthService/Signup":               "creates a user and a session",
	"/bytebase.v1.AuthService/SwitchWorkspace":      "mints a token for another workspace",
}

// methodEntry is one explicitly mapped method.
type methodEntry struct {
	capabilities []Capability
	reason       string
}

// explicitMethods maps the methods whose requirement cannot be read off a
// permission annotation, plus the one method whose annotation understates what
// it does.
//
// Two shapes live here:
//
//  1. No annotation to derive from — custom-authorization methods that
//     authorize inside their handler, and methods that run before a credential
//     exists. These mint an operation ID. Note that skipping authentication and
//     passing the ceiling are independent checks: a credential-free method
//     still resolves here, it is not exempt.
//  2. An annotation that under-specifies the operation, because it is shared
//     with siblings of a different risk. Four methods today: Export and
//     DiffSchema share SQLService/Query's "bb.databases.get",
//     GetTaskRunSession shares four store-only task reads' "bb.taskRuns.list",
//     and SetupSample shares CreateProject's "bb.projects.create" while also
//     registering instances. Each entry restates the annotated permission and
//     adds what separates it. For a method the ACL actually enforces the
//     annotation on — auth_method IAM — the lint pins that restatement to the
//     live annotation, so an entry cannot drift into requiring less than the
//     ACL requires.
//
// A shared annotation is the derivation's one blind spot, and nothing in the
// lint can find the next one: a new RPC that reuses an existing permission
// resolves silently. The inventory golden file is what surfaces it, by putting
// every method's resolved capability and set membership into the diff.
var explicitMethods = map[string]methodEntry{
	// Shape 2: annotation under-specifies.
	"/bytebase.v1.SQLService/Export": {
		capabilities: []Capability{Capability(permission.DatabasesGet), OpDatabasesExport},
		reason:       "shares Query's bb.databases.get annotation; the operation ID is what a read-only ceiling withholds",
	},
	"/bytebase.v1.RolloutService/GetTaskRunSession": {
		capabilities: []Capability{Capability(permission.TaskRunsList), OpTaskRunsGetSession},
		reason:       "shares the task-status read permission but opens an admin-credentialed connection returning other sessions' SQL",
	},
	"/bytebase.v1.DatabaseService/DiffSchema": {
		capabilities: []Capability{Capability(permission.DatabasesGet), OpDatabasesDiffSchema},
		reason:       "shares Query's bb.databases.get annotation but returns migration DDL against a caller-supplied target schema",
	},
	"/bytebase.v1.ActuatorService/SetupSample": {
		capabilities: []Capability{Capability(permission.ProjectsCreate), Capability(permission.InstancesCreate)},
		reason:       "shares bb.projects.create but also registers two instances with ADMIN data sources and starts sample Postgres processes",
	},

	// Shape 1: no annotation, or an annotation the ACL does not evaluate.
	"/bytebase.v1.AccessGrantService/SearchMyAccessGrants": {
		capabilities: []Capability{OpAccessGrantsSearchMine},
		reason:       "custom authorization; annotation is not evaluated, the handler scopes to the caller's own grants",
	},
	// Assistant.
	"/bytebase.v1.AIService/Chat": {
		capabilities: []Capability{OpAssistantChat},
		reason:       "custom authorization; forwards caller content to the configured AI provider",
	},

	// CEL utilities: authenticated, no permission, no stored state touched.
	"/bytebase.v1.CelService/BatchDeparse": {
		capabilities: []Capability{OpCelExpressionsDeparse},
		reason:       "no annotation; stateless expression utility",
	},
	"/bytebase.v1.CelService/BatchParse": {
		capabilities: []Capability{OpCelExpressionsParse},
		reason:       "no annotation; stateless expression utility",
	},

	// Issue approval.
	"/bytebase.v1.IssueService/ApproveIssue": {
		capabilities: []Capability{OpIssuesApprove},
		reason:       "custom authorization; advances the approval decision",
	},
	"/bytebase.v1.IssueService/RejectIssue": {
		capabilities: []Capability{OpIssuesReject},
		reason:       "custom authorization; advances the approval decision",
	},
	"/bytebase.v1.IssueService/RequestIssue": {
		capabilities: []Capability{OpIssuesRequestApproval},
		reason:       "custom authorization; re-enters the approval flow",
	},
	"/bytebase.v1.IssueService/RetryIssueApproval": {
		capabilities: []Capability{OpIssuesRetryApproval},
		reason:       "custom authorization; re-runs approval finding",
	},

	// Projects.
	"/bytebase.v1.ProjectService/SearchProjects": {
		capabilities: []Capability{OpProjectsSearch},
		reason:       "custom authorization; caller-scoped project search",
	},

	// Query history: the caller's own executed statements.
	"/bytebase.v1.QueryHistoryService/GetQueryHistory": {
		capabilities: []Capability{OpQueryHistoriesGet},
		reason:       "custom authorization; caller-scoped statement history",
	},
	"/bytebase.v1.QueryHistoryService/SearchQueryHistories": {
		capabilities: []Capability{OpQueryHistoriesSearch},
		reason:       "custom authorization; caller-scoped statement history",
	},
	"/bytebase.v1.SQLService/GetQueryHistory": {
		capabilities: []Capability{OpQueryHistoriesGet},
		reason:       "custom authorization; same operation as QueryHistoryService/GetQueryHistory",
	},
	"/bytebase.v1.SQLService/SearchQueryHistories": {
		capabilities: []Capability{OpQueryHistoriesSearch},
		reason:       "custom authorization; same operation as QueryHistoryService/SearchQueryHistories",
	},

	// Rollout execution.
	"/bytebase.v1.RolloutService/BatchCancelTaskRuns": {
		capabilities: []Capability{OpTaskRunsCancel},
		reason:       "custom authorization; cancels running executions",
	},
	"/bytebase.v1.RolloutService/BatchRunTasks": {
		capabilities: []Capability{OpTasksRun},
		reason:       "custom authorization; executes tasks against databases",
	},
	"/bytebase.v1.RolloutService/BatchSkipTasks": {
		capabilities: []Capability{OpTasksSkip},
		reason:       "custom authorization; skips tasks in a rollout",
	},
	"/bytebase.v1.RolloutService/CreateRollout": {
		capabilities: []Capability{OpRolloutsCreate},
		reason:       "custom authorization; creates the rollout that carries a change",
	},

	// Server and workspace basics.
	"/bytebase.v1.ActuatorService/GetActuatorInfo": {
		capabilities: []Capability{OpServerInfoGet},
		reason:       "credential-free; server version and status counts",
	},
	"/bytebase.v1.IdentityProviderService/ListIdentityProviders": {
		capabilities: []Capability{OpIdentityProvidersList},
		reason:       "credential-free; sign-in page provider list, secrets blanked",
	},
	"/bytebase.v1.WorkspaceService/GetWorkspace": {
		capabilities: []Capability{OpWorkspacesGet},
		reason:       "credential-free; workspace id and title",
	},
	"/bytebase.v1.WorkspaceService/LeaveWorkspace": {
		capabilities: []Capability{OpWorkspacesLeave},
		reason:       "custom authorization; membership change",
	},
	"/bytebase.v1.WorkspaceService/ListWorkspaces": {
		capabilities: []Capability{OpWorkspacesList},
		reason:       "custom authorization; caller's own workspaces",
	},

	// Subscription.
	"/bytebase.v1.SubscriptionService/GetSubscription": {
		capabilities: []Capability{OpSubscriptionGet},
		reason:       "no annotation; license plan and seat status",
	},
	"/bytebase.v1.SubscriptionService/ListPurchasePlans": {
		capabilities: []Capability{OpPurchasePlansList},
		reason:       "no annotation; public pricing catalog",
	},

	// Users.
	"/bytebase.v1.UserService/DeleteUser": {
		capabilities: []Capability{OpUsersDelete},
		reason:       "custom authorization; deactivates a principal",
	},
	"/bytebase.v1.UserService/GetCurrentUser": {
		capabilities: []Capability{OpUsersGetCurrent},
		reason:       "credential-free custom authorization; the caller's own profile",
	},
	"/bytebase.v1.UserService/UndeleteUser": {
		capabilities: []Capability{OpUsersUndelete},
		reason:       "custom authorization; reactivates a principal",
	},
	"/bytebase.v1.UserService/UpdateUser": {
		capabilities: []Capability{OpUsersUpdate},
		reason:       "custom authorization; covers password and MFA changes",
	},

	// Worksheets.
	"/bytebase.v1.WorksheetService/BatchUpdateWorksheetOrganizer": {
		capabilities: []Capability{OpWorksheetOrganizersUpdate},
		reason:       "custom authorization; per-user star and folder state",
	},
	"/bytebase.v1.WorksheetService/CreateWorksheet": {
		capabilities: []Capability{OpWorksheetsCreate},
		reason:       "custom authorization; stores caller-authored SQL",
	},
	"/bytebase.v1.WorksheetService/DeleteWorksheet": {
		capabilities: []Capability{OpWorksheetsDelete},
		reason:       "custom authorization; deletes caller-authored SQL",
	},
	"/bytebase.v1.WorksheetService/GetWorksheet": {
		capabilities: []Capability{OpWorksheetsGet},
		reason:       "custom authorization; returns full SQL content",
	},
	"/bytebase.v1.WorksheetService/ListWorksheetFolders": {
		capabilities: []Capability{OpWorksheetFoldersList},
		reason:       "custom authorization; folder names only",
	},
	"/bytebase.v1.WorksheetService/SearchWorksheets": {
		capabilities: []Capability{OpWorksheetsSearch},
		reason:       "custom authorization; returns SQL content, truncated",
	},
	"/bytebase.v1.WorksheetService/UpdateWorksheet": {
		capabilities: []Capability{OpWorksheetsUpdate},
		reason:       "custom authorization; edits caller-authored SQL",
	},
	"/bytebase.v1.WorksheetService/UpdateWorksheetOrganizer": {
		capabilities: []Capability{OpWorksheetOrganizersUpdate},
		reason:       "custom authorization; per-user star and folder state",
	},
}

// Required returns what a request demands of the ceiling.
//
// Resolution is exactly-once by construction: a method is either forbidden,
// explicitly mapped, or derived from its permission annotation — checked in
// that order, and a method that matches none of the three is a denial plus an
// error, never an allow. The lint proves the three paths partition every v1 RPC.
//
// The derivation reads the annotation through auth.GetAuthContext — the
// function the auth interceptors resolve every request through, and the origin
// of the permission the ACL then checks — and applies the ACL's two
// per-request raises through auth.PermissionForRequest and
// auth.HasAllowMissingEnabled. There is no second permission table. The one
// caveat, recorded per row in the inventory: on a custom-authorization method
// the ACL skips the IAM check, so the annotation describes what the handler
// enforces rather than what an interceptor does.
func Required(procedure string, request any) (Requirement, error) {
	if reason, ok := forbiddenMethods[procedure]; ok {
		return denied(reason), nil
	}
	if entry, ok := explicitMethods[procedure]; ok {
		if len(entry.capabilities) == 0 {
			return denied("explicit mapping is empty"), errors.Errorf("method %q has an explicit mapping with no capabilities", procedure)
		}
		return Requirement{
			Capabilities: sortedCapabilities(entry.capabilities),
			Reason:       entry.reason,
		}, nil
	}
	authContext, err := auth.GetAuthContext(procedure)
	if err != nil {
		return denied("unresolvable method"), errors.Wrapf(err, "failed to resolve auth context for %q", procedure)
	}
	granted := auth.PermissionForRequest(request, authContext.Permission)
	if granted == "" {
		// A new RPC with no annotation lands here, and the lint turns that into
		// a CI failure rather than a runtime surprise.
		return denied("unclassified method"), errors.Errorf("method %q has no permission annotation and no explicit capability mapping", procedure)
	}

	capabilities := []Capability{Capability(granted)}
	// The ACL's second per-request raise: allow_missing turns an update into a
	// create, and it demands the create permission on top of the update one
	// (backend/api/v1/acl.go). Read from the same helpers rather than restated,
	// so a ceiling holding bb.plans.update without bb.plans.create cannot admit
	// an UpdatePlan that creates.
	if auth.HasAllowMissingEnabled(request) {
		capabilities = append(capabilities, Capability(auth.AllowMissingCreatePermission(granted)))
	}

	reason := "derived from the method's permission annotation"
	if authContext.AuthMethod != common.AuthMethodIAM {
		// The ACL skips the IAM check for these, so the annotation describes
		// what the handler enforces rather than what an interceptor does. It is
		// still the method's declared requirement, and it is what the OpenAPI
		// index publishes, so it stays the ceiling vocabulary — but the
		// inventory says so out loud.
		reason = "derived from the method's permission annotation (custom authorization: enforced by the handler, not the ACL)"
	}
	return Requirement{Capabilities: sortedCapabilities(capabilities), Reason: reason}, nil
}

// denied builds the fail-closed requirement. Every error path returns one, so a
// caller that logs the error and proceeds still denies rather than admits.
func denied(reason string) Requirement {
	return Requirement{Forbidden: true, Reason: reason}
}
