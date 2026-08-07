package aiaccess

// Minted operation capabilities.
//
// These are public API surface the moment custom ceiling sets ship: they appear
// in the set-contents API, in the consent page, and in Terraform state. Renaming
// one later is a breaking change, so they are named like proto fields — stable
// collection noun, consistent verb, no abbreviations — and they mirror the
// shape of the IAM permissions they sit beside in a set ("bb.rollouts.get" ->
// "ai.op.rollouts.create"): plural collection, singular for a singleton
// resource, lowerCamelCase throughout.
const (
	// Access grants. SearchMyAccessGrants annotates bb.accessGrants.get but is
	// auth_method CUSTOM, so the ACL never evaluates that permission
	// (backend/api/v1/acl.go doIAMPermissionCheck); the handler scopes the
	// query to the caller's own grants instead. Gating it on the workspace-wide
	// grant-read permission would describe access the method does not have.
	OpAccessGrantsSearchMine Capability = "ai.op.accessGrants.searchMine"

	// Assistant.
	OpAssistantChat Capability = "ai.op.assistant.chat"

	// CEL utilities.
	OpCelExpressionsDeparse Capability = "ai.op.celExpressions.deparse"
	OpCelExpressionsParse   Capability = "ai.op.celExpressions.parse"

	// Data movement and schema authoring. Both share SQLService/Query's
	// "bb.databases.get" annotation: this pair is what separates reading a
	// result set from taking a copy of it out of the product, and from
	// generating migration DDL against a caller-supplied target schema.
	OpDatabasesDiffSchema Capability = "ai.op.databases.diffSchema"
	OpDatabasesExport     Capability = "ai.op.databases.export"

	// Identity providers. The listing method is credential-free (it feeds the
	// sign-in page), so it carries no annotation to derive from.
	OpIdentityProvidersList Capability = "ai.op.identityProviders.list"

	// Issue approval. These four drive the approval state machine and belong to
	// no predefined set.
	OpIssuesApprove         Capability = "ai.op.issues.approve"
	OpIssuesReject          Capability = "ai.op.issues.reject"
	OpIssuesRequestApproval Capability = "ai.op.issues.requestApproval"
	OpIssuesRetryApproval   Capability = "ai.op.issues.retryApproval"

	// Projects.
	OpProjectsSearch Capability = "ai.op.projects.search"

	// Purchasing.
	OpPurchasePlansList Capability = "ai.op.purchasePlans.list"

	// Query history.
	OpQueryHistoriesGet    Capability = "ai.op.queryHistories.get"
	OpQueryHistoriesSearch Capability = "ai.op.queryHistories.search"

	// Rollout execution. GetTaskRunSession shares the plain "bb.taskRuns.list"
	// read permission with four store-only reads, but it alone opens an
	// admin-credentialed connection to the customer database and returns other
	// sessions' in-flight SQL; this ID is what separates it from task status.
	OpRolloutsCreate     Capability = "ai.op.rollouts.create"
	OpTaskRunsCancel     Capability = "ai.op.taskRuns.cancel"
	OpTaskRunsGetSession Capability = "ai.op.taskRuns.getSession"
	OpTasksRun           Capability = "ai.op.tasks.run"
	OpTasksSkip          Capability = "ai.op.tasks.skip"

	// Server.
	OpServerInfoGet Capability = "ai.op.serverInfo.get"

	// Subscription.
	OpSubscriptionGet Capability = "ai.op.subscription.get"

	// Users.
	OpUsersDelete     Capability = "ai.op.users.delete"
	OpUsersGetCurrent Capability = "ai.op.users.getCurrent"
	OpUsersUndelete   Capability = "ai.op.users.undelete"
	OpUsersUpdate     Capability = "ai.op.users.update"

	// Worksheets.
	OpWorksheetFoldersList      Capability = "ai.op.worksheetFolders.list"
	OpWorksheetOrganizersUpdate Capability = "ai.op.worksheetOrganizers.update"
	OpWorksheetsCreate          Capability = "ai.op.worksheets.create"
	OpWorksheetsDelete          Capability = "ai.op.worksheets.delete"
	OpWorksheetsGet             Capability = "ai.op.worksheets.get"
	OpWorksheetsSearch          Capability = "ai.op.worksheets.search"
	OpWorksheetsUpdate          Capability = "ai.op.worksheets.update"

	// Workspaces.
	OpWorkspacesGet   Capability = "ai.op.workspaces.get"
	OpWorkspacesLeave Capability = "ai.op.workspaces.leave"
	OpWorkspacesList  Capability = "ai.op.workspaces.list"
)

// operations is every minted operation ID, sorted. A capability that is not a
// permission and not in here does not exist.
var operations = sortedCapabilities([]Capability{
	OpAccessGrantsSearchMine,
	OpAssistantChat,
	OpCelExpressionsDeparse,
	OpCelExpressionsParse,
	OpDatabasesDiffSchema,
	OpDatabasesExport,
	OpIdentityProvidersList,
	OpIssuesApprove,
	OpIssuesReject,
	OpIssuesRequestApproval,
	OpIssuesRetryApproval,
	OpProjectsSearch,
	OpPurchasePlansList,
	OpQueryHistoriesGet,
	OpQueryHistoriesSearch,
	OpRolloutsCreate,
	OpServerInfoGet,
	OpSubscriptionGet,
	OpTaskRunsCancel,
	OpTaskRunsGetSession,
	OpTasksRun,
	OpTasksSkip,
	OpUsersDelete,
	OpUsersGetCurrent,
	OpUsersUndelete,
	OpUsersUpdate,
	OpWorksheetFoldersList,
	OpWorksheetOrganizersUpdate,
	OpWorksheetsCreate,
	OpWorksheetsDelete,
	OpWorksheetsGet,
	OpWorksheetsSearch,
	OpWorksheetsUpdate,
	OpWorkspacesGet,
	OpWorkspacesLeave,
	OpWorkspacesList,
})
