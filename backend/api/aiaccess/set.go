package aiaccess

import (
	"slices"

	"github.com/bytebase/bytebase/backend/common/permission"
)

// Set is a ceiling's capability set: what an AI session may reach at most,
// before the caller's own IAM is intersected with it.
type Set struct {
	name         string
	capabilities []Capability
	index        map[Capability]bool
}

func newSet(name string, capabilities []Capability) *Set {
	sorted := sortedCapabilities(capabilities)
	index := make(map[Capability]bool, len(sorted))
	for _, capability := range sorted {
		index[capability] = true
	}
	return &Set{name: name, capabilities: sorted, index: index}
}

// Name returns the set's stable name.
func (s *Set) Name() string { return s.name }

// Capabilities returns the set's contents, sorted. This is what the settings
// drawer and the consent page render.
func (s *Set) Capabilities() []Capability { return slices.Clone(s.capabilities) }

// Contains reports membership.
func (s *Set) Contains(capability Capability) bool { return s.index[capability] }

// Allows evaluates required(request) ⊆ resolve(ceiling) for this set.
//
// A forbidden requirement is denied by any set. So is one that demands nothing:
// an empty capability list is a resolution bug, not a grant, and it is the
// shape every error path returns.
func (s *Set) Allows(requirement Requirement) bool {
	if requirement.Forbidden || len(requirement.Capabilities) == 0 {
		return false
	}
	for _, capability := range requirement.Capabilities {
		if !s.Contains(capability) {
			return false
		}
	}
	return true
}

// statementCapabilities are permissions that no method annotation carries: the
// SQL service resolves them per statement inside Query and Export
// (backend/api/v1/sql_service.go accessCheckWithGrantedTargets), against the
// caller's IAM.
//
// They are vocabulary, not yet evaluator input. Required never returns one —
// SQLService/Query resolves to bb.databases.get alone — so containment admits a
// DML statement exactly as it admits a SELECT. Holding bb.sql.select while
// withholding bb.sql.dml and bb.sql.ddl is what the statement clamp will key
// on; until that clamp exists, a read-only ceiling narrows which methods a
// session may call, not which statements it may run.
var statementCapabilities = []Capability{
	Capability(permission.SQLDdl),
	Capability(permission.SQLDml),
	Capability(permission.SQLExplain),
	Capability(permission.SQLInfo),
	Capability(permission.SQLSelect),
}

// ReadOnly is the predefined read-only ceiling.
//
// Membership rule, applied per capability rather than by taste: a capability is
// here when every method that requires it only reads, returns no credential and
// no Bytebase privilege topology, and sends nothing outside the customer's own
// databases. Reading those databases is what read-only exists for, so a live
// connection to them is in scope; an outbound call to a third party, a copy of
// data leaving the product, and another user's stored SQL are not — which is
// why export, the AI proxy, and the workspace-wide query-history and worksheet
// listings are absent even though none of them writes.
var ReadOnly = newSet("READ_ONLY", []Capability{
	Capability(permission.ChangelogsGet),
	Capability(permission.ChangelogsList),
	Capability(permission.DatabaseCatalogsGet),
	Capability(permission.DatabaseGroupsGet),
	Capability(permission.DatabaseGroupsList),
	Capability(permission.DatabasesGet),
	Capability(permission.DatabasesGetSchema),
	Capability(permission.DatabasesList),
	Capability(permission.InstanceRolesList),
	Capability(permission.InstancesGet),
	Capability(permission.InstancesList),
	Capability(permission.IssueCommentsList),
	Capability(permission.IssuesGet),
	Capability(permission.IssuesList),
	Capability(permission.PlanCheckRunsGet),
	Capability(permission.PlansGet),
	Capability(permission.PlansList),
	Capability(permission.ProjectsGet),
	Capability(permission.ProjectsList),
	Capability(permission.ReleasesGet),
	Capability(permission.ReleasesList),
	Capability(permission.ReviewConfigsGet),
	Capability(permission.ReviewConfigsList),
	Capability(permission.RevisionsGet),
	Capability(permission.RevisionsList),
	Capability(permission.RolloutsGet),
	Capability(permission.RolloutsList),
	Capability(permission.SheetsGet),
	Capability(permission.SQLExplain),
	Capability(permission.SQLInfo),
	Capability(permission.SQLSelect),
	Capability(permission.TaskRunsList),

	OpAccessGrantsSearchMine,
	OpCelExpressionsDeparse,
	OpCelExpressionsParse,
	OpIdentityProvidersList,
	OpProjectsSearch,
	OpPurchasePlansList,
	OpQueryHistoriesGet,
	OpQueryHistoriesSearch,
	OpServerInfoGet,
	OpSubscriptionGet,
	OpUsersGetCurrent,
	OpWorksheetFoldersList,
	OpWorksheetsGet,
	OpWorksheetsSearch,
	OpWorkspacesGet,
	OpWorkspacesList,
})

// ReadWrite is the predefined read-write ceiling: everything in the vocabulary
// except the administration capabilities, the approval capabilities, and the
// forbidden methods (decision 7). It is a narrowing of today's behavior, where
// an AI session reaches the entire API; the delta is exactly adminCapabilities
// plus approvalCapabilities plus forbiddenMethods.
//
// It is written out in full rather than derived from ReadOnly so the drawer
// renders stable data, so adding a capability to a shipped preset is a reviewed
// edit, and so the containment of one set by the other is a property a test can
// break rather than a consequence of how they are built. The lint holds the
// tables in agreement: a capability that is neither administration nor approval
// fails CI until it is listed here.
var ReadWrite = newSet("READ_WRITE", []Capability{
	// Everything read-only holds.
	Capability(permission.ChangelogsGet),
	Capability(permission.ChangelogsList),
	Capability(permission.DatabaseCatalogsGet),
	Capability(permission.DatabaseGroupsGet),
	Capability(permission.DatabaseGroupsList),
	Capability(permission.DatabasesGet),
	Capability(permission.DatabasesGetSchema),
	Capability(permission.DatabasesList),
	Capability(permission.InstanceRolesList),
	Capability(permission.InstancesGet),
	Capability(permission.InstancesList),
	Capability(permission.IssueCommentsList),
	Capability(permission.IssuesGet),
	Capability(permission.IssuesList),
	Capability(permission.PlanCheckRunsGet),
	Capability(permission.PlansGet),
	Capability(permission.PlansList),
	Capability(permission.ProjectsGet),
	Capability(permission.ProjectsList),
	Capability(permission.ReleasesGet),
	Capability(permission.ReleasesList),
	Capability(permission.ReviewConfigsGet),
	Capability(permission.ReviewConfigsList),
	Capability(permission.RevisionsGet),
	Capability(permission.RevisionsList),
	Capability(permission.RolloutsGet),
	Capability(permission.RolloutsList),
	Capability(permission.SheetsGet),
	Capability(permission.SQLExplain),
	Capability(permission.SQLInfo),
	Capability(permission.SQLSelect),
	Capability(permission.TaskRunsList),

	OpAccessGrantsSearchMine,
	OpCelExpressionsDeparse,
	OpCelExpressionsParse,
	OpIdentityProvidersList,
	OpProjectsSearch,
	OpPurchasePlansList,
	OpQueryHistoriesGet,
	OpQueryHistoriesSearch,
	OpServerInfoGet,
	OpSubscriptionGet,
	OpUsersGetCurrent,
	OpWorksheetFoldersList,
	OpWorksheetsGet,
	OpWorksheetsSearch,
	OpWorkspacesGet,
	OpWorkspacesList,

	// Plus the change-authoring half. Two entries are read-shaped but author
	// changes rather than describe them: diffMetadata and the
	// ai.op.databases.diffSchema operation both generate migration DDL from a
	// caller-supplied target schema, and no read-only role holds diffMetadata
	// (backend/store/predefined_roles.go grants it to the schema-change roles).
	Capability(permission.DatabaseGroupsCreate),
	Capability(permission.DatabaseGroupsDelete),
	Capability(permission.DatabaseGroupsUpdate),
	Capability(permission.DatabasesDiffMetadata),
	Capability(permission.DatabasesSync),
	Capability(permission.DatabasesUpdate),
	Capability(permission.InstancesSync),
	Capability(permission.IssueCommentsCreate),
	Capability(permission.IssueCommentsUpdate),
	Capability(permission.IssuesCreate),
	Capability(permission.IssuesUpdate),
	Capability(permission.PlanCheckRunsRun),
	Capability(permission.PlansCreate),
	Capability(permission.PlansUpdate),
	Capability(permission.ProjectsCreate),
	Capability(permission.ReleasesCheck),
	Capability(permission.ReleasesCreate),
	Capability(permission.ReleasesDelete),
	Capability(permission.ReleasesUndelete),
	Capability(permission.ReleasesUpdate),
	Capability(permission.RevisionsCreate),
	Capability(permission.RevisionsDelete),
	Capability(permission.SheetsCreate),
	Capability(permission.SQLDdl),
	Capability(permission.SQLDml),

	OpDatabasesDiffSchema,
	OpDatabasesExport,
	OpRolloutsCreate,
	OpTaskRunsCancel,
	OpTasksRun,
	OpTasksSkip,
	OpWorksheetOrganizersUpdate,
	OpWorksheetsCreate,
	OpWorksheetsDelete,
	OpWorksheetsUpdate,
})

// adminCapabilities is the administration class: identity, credentials, access
// control, governance policy, billing, workspace and instance configuration,
// project lifecycle, and the operator's own audit trail. No predefined set
// contains them, and 1b-5's custom sets cannot reach them either, because the
// authorable universe is the union of the predefined sets.
//
// Reading is administration too where the read exposes who-can-do-what, a
// stored secret, or another user's content: bb.*.getIamPolicy returns the
// privilege topology; bb.accessGrants.get/list return every user's granted SQL
// plus the grant topology, unscoped; bb.settings.list returns every setting
// type with no per-type check (backend/api/v1/setting_service.go), including
// the AI endpoint, SMTP host and username, and the masking salts;
// bb.queryHistories.list returns every user's raw statements project-wide, or
// workspace-wide under the "projects/-" wildcard; bb.worksheets.list bypasses
// per-worksheet visibility and returns other users' private SQL, and no
// predefined role holds it at all.
//
// bb.projects.delete is here because one capability covers both halves of
// DeleteProject: archiving, and purge:true, which unrecoverably deletes the
// project's issues, plans, task runs, query history and policies
// (backend/store/project.go). bb.projects.undelete follows it as the inverse of
// the same lifecycle.
//
// Three entries are operations rather than permissions, and are the rows this
// class stretches furthest to cover — all flagged for sign-off:
// ai.op.assistant.chat spends the workspace's AI provider credential on an
// outbound call that no IAM permission and no audit annotation covers;
// ai.op.taskRuns.getSession is live admin-driver diagnostics over other
// sessions' SQL; ai.op.workspaces.leave changes membership.
var adminCapabilities = sortedCapabilities([]Capability{
	Capability(permission.AccessGrantsActivate),
	Capability(permission.AccessGrantsCreate),
	Capability(permission.AccessGrantsGet),
	Capability(permission.AccessGrantsList),
	Capability(permission.AccessGrantsRevoke),
	Capability(permission.AuditLogsExport),
	Capability(permission.AuditLogsSearch),
	Capability(permission.DatabaseCatalogsUpdate),
	Capability(permission.GroupsCreate),
	Capability(permission.GroupsDelete),
	Capability(permission.GroupsGet),
	Capability(permission.GroupsList),
	Capability(permission.GroupsUpdate),
	Capability(permission.IdentityProvidersCreate),
	Capability(permission.IdentityProvidersDelete),
	Capability(permission.IdentityProvidersGet),
	Capability(permission.IdentityProvidersUpdate),
	Capability(permission.InstancesCreate),
	Capability(permission.InstancesDelete),
	Capability(permission.InstancesUndelete),
	Capability(permission.InstancesUpdate),
	Capability(permission.PoliciesCreate),
	Capability(permission.PoliciesDelete),
	Capability(permission.PoliciesGet),
	Capability(permission.PoliciesList),
	Capability(permission.PoliciesUpdate),
	Capability(permission.ProjectsDelete),
	Capability(permission.ProjectsGetIAMPolicy),
	Capability(permission.ProjectsSetIAMPolicy),
	Capability(permission.ProjectsUndelete),
	Capability(permission.ProjectsUpdate),
	Capability(permission.QueryHistoriesList),
	Capability(permission.ReviewConfigsCreate),
	Capability(permission.ReviewConfigsDelete),
	Capability(permission.ReviewConfigsUpdate),
	Capability(permission.RolesCreate),
	Capability(permission.RolesDelete),
	Capability(permission.RolesGet),
	Capability(permission.RolesList),
	Capability(permission.RolesUpdate),
	Capability(permission.ServiceAccountsCreate),
	Capability(permission.ServiceAccountsDelete),
	Capability(permission.ServiceAccountsGet),
	Capability(permission.ServiceAccountsList),
	Capability(permission.ServiceAccountsUndelete),
	Capability(permission.ServiceAccountsUpdate),
	Capability(permission.SettingsGet),
	Capability(permission.SettingsList),
	Capability(permission.SettingsSet),
	Capability(permission.SQLAdmin),
	Capability(permission.SubscriptionManage),
	Capability(permission.UsersCreate),
	Capability(permission.UsersGet),
	Capability(permission.UsersList),
	Capability(permission.UsersUpdateEmail),
	Capability(permission.WorkloadIdentitiesCreate),
	Capability(permission.WorkloadIdentitiesDelete),
	Capability(permission.WorkloadIdentitiesGet),
	Capability(permission.WorkloadIdentitiesList),
	Capability(permission.WorkloadIdentitiesUndelete),
	Capability(permission.WorkloadIdentitiesUpdate),
	Capability(permission.WorksheetsList),
	Capability(permission.WorkspacesDelete),
	Capability(permission.WorkspacesGetIamPolicy),
	Capability(permission.WorkspacesRotateDirectorySyncToken),
	Capability(permission.WorkspacesSetIamPolicy),
	Capability(permission.WorkspacesUpdate),

	OpAssistantChat,
	OpTaskRunsGetSession,
	OpUsersDelete,
	OpUsersUndelete,
	OpUsersUpdate,
	OpWorkspacesLeave,
})

// approvalCapabilities is the approval class. A mutation is not automatically a
// write: approving, rejecting, re-requesting and retrying an approval move the
// human decision itself, so they belong to no set — an AI session may compose a
// change and may execute an approved one, but may never be the approver.
var approvalCapabilities = sortedCapabilities([]Capability{

	OpIssuesApprove,
	OpIssuesReject,
	OpIssuesRequestApproval,
	OpIssuesRetryApproval,
})
