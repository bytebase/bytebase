// Static registry of common Bytebase API operations for keyword search.
// All Bytebase APIs use Connect protocol: POST to /bytebase.v1.ServiceName/MethodName

interface ApiOperation {
  id: string;
  path: string;
  description: string;
}

const API_OPERATIONS: ApiOperation[] = [
  // Project
  {
    id: "bytebase.v1.ProjectService.ListProjects",
    path: "/bytebase.v1.ProjectService/ListProjects",
    description: "List all projects",
  },
  {
    id: "bytebase.v1.ProjectService.GetProject",
    path: "/bytebase.v1.ProjectService/GetProject",
    description: "Get a project by name",
  },
  {
    id: "bytebase.v1.ProjectService.CreateProject",
    path: "/bytebase.v1.ProjectService/CreateProject",
    description: "Create a new project",
  },
  {
    id: "bytebase.v1.ProjectService.DeleteProject",
    path: "/bytebase.v1.ProjectService/DeleteProject",
    description: "Delete a project",
  },
  {
    id: "bytebase.v1.ProjectService.SearchProjects",
    path: "/bytebase.v1.ProjectService/SearchProjects",
    description: "Search projects with filters",
  },
  {
    id: "bytebase.v1.ProjectService.GetIamPolicy",
    path: "/bytebase.v1.ProjectService/GetIamPolicy",
    description: "Get project IAM policy (roles and members)",
  },

  // Instance
  {
    id: "bytebase.v1.InstanceService.ListInstances",
    path: "/bytebase.v1.InstanceService/ListInstances",
    description: "List all database instances",
  },
  {
    id: "bytebase.v1.InstanceService.GetInstance",
    path: "/bytebase.v1.InstanceService/GetInstance",
    description: "Get a database instance",
  },
  {
    id: "bytebase.v1.InstanceService.CreateInstance",
    path: "/bytebase.v1.InstanceService/CreateInstance",
    description: "Create a new database instance",
  },
  {
    id: "bytebase.v1.InstanceService.UpdateInstance",
    path: "/bytebase.v1.InstanceService/UpdateInstance",
    description: "Update a database instance",
  },
  {
    id: "bytebase.v1.InstanceService.DeleteInstance",
    path: "/bytebase.v1.InstanceService/DeleteInstance",
    description: "Delete a database instance",
  },
  {
    id: "bytebase.v1.InstanceService.SyncInstance",
    path: "/bytebase.v1.InstanceService/SyncInstance",
    description: "Sync instance metadata and databases",
  },

  // Database
  {
    id: "bytebase.v1.DatabaseService.ListDatabases",
    path: "/bytebase.v1.DatabaseService/ListDatabases",
    description: "List databases in an instance",
  },
  {
    id: "bytebase.v1.DatabaseService.GetDatabase",
    path: "/bytebase.v1.DatabaseService/GetDatabase",
    description: "Get a database by name",
  },
  {
    id: "bytebase.v1.DatabaseService.UpdateDatabase",
    path: "/bytebase.v1.DatabaseService/UpdateDatabase",
    description: "Update database properties",
  },
  {
    id: "bytebase.v1.DatabaseService.GetDatabaseMetadata",
    path: "/bytebase.v1.DatabaseService/GetDatabaseMetadata",
    description: "Get database metadata (tables, columns, indexes)",
  },
  {
    id: "bytebase.v1.DatabaseService.GetDatabaseSchema",
    path: "/bytebase.v1.DatabaseService/GetDatabaseSchema",
    description: "Get database schema as SQL",
  },
  {
    id: "bytebase.v1.DatabaseService.DiffSchema",
    path: "/bytebase.v1.DatabaseService/DiffSchema",
    description: "Diff two database schemas",
  },
  {
    id: "bytebase.v1.DatabaseService.ListChangelogs",
    path: "/bytebase.v1.DatabaseService/ListChangelogs",
    description: "List database change history",
  },
  {
    id: "bytebase.v1.DatabaseService.SyncDatabase",
    path: "/bytebase.v1.DatabaseService/SyncDatabase",
    description: "Sync database metadata",
  },

  // Issue
  {
    id: "bytebase.v1.IssueService.ListIssues",
    path: "/bytebase.v1.IssueService/ListIssues",
    description: "List issues in a project",
  },
  {
    id: "bytebase.v1.IssueService.GetIssue",
    path: "/bytebase.v1.IssueService/GetIssue",
    description: "Get an issue by name",
  },
  {
    id: "bytebase.v1.IssueService.CreateIssue",
    path: "/bytebase.v1.IssueService/CreateIssue",
    description: "Create a new issue (database change request)",
  },
  {
    id: "bytebase.v1.IssueService.UpdateIssue",
    path: "/bytebase.v1.IssueService/UpdateIssue",
    description: "Update an issue",
  },
  {
    id: "bytebase.v1.IssueService.SearchIssues",
    path: "/bytebase.v1.IssueService/SearchIssues",
    description: "Search issues with filters",
  },
  {
    id: "bytebase.v1.IssueService.ApproveIssue",
    path: "/bytebase.v1.IssueService/ApproveIssue",
    description: "Approve an issue",
  },
  {
    id: "bytebase.v1.IssueService.RejectIssue",
    path: "/bytebase.v1.IssueService/RejectIssue",
    description: "Reject an issue",
  },
  {
    id: "bytebase.v1.IssueService.ListIssueComments",
    path: "/bytebase.v1.IssueService/ListIssueComments",
    description: "List comments on an issue",
  },
  {
    id: "bytebase.v1.IssueService.CreateIssueComment",
    path: "/bytebase.v1.IssueService/CreateIssueComment",
    description: "Add a comment to an issue",
  },

  // Plan
  {
    id: "bytebase.v1.PlanService.ListPlans",
    path: "/bytebase.v1.PlanService/ListPlans",
    description: "List plans in a project",
  },
  {
    id: "bytebase.v1.PlanService.GetPlan",
    path: "/bytebase.v1.PlanService/GetPlan",
    description: "Get a plan by name",
  },
  {
    id: "bytebase.v1.PlanService.CreatePlan",
    path: "/bytebase.v1.PlanService/CreatePlan",
    description: "Create a deployment plan",
  },
  {
    id: "bytebase.v1.PlanService.UpdatePlan",
    path: "/bytebase.v1.PlanService/UpdatePlan",
    description: "Update a deployment plan",
  },

  // Rollout
  {
    id: "bytebase.v1.RolloutService.GetRollout",
    path: "/bytebase.v1.RolloutService/GetRollout",
    description: "Get a rollout by name",
  },
  {
    id: "bytebase.v1.RolloutService.ListRollouts",
    path: "/bytebase.v1.RolloutService/ListRollouts",
    description: "List rollouts in a project",
  },
  {
    id: "bytebase.v1.RolloutService.ListTaskRuns",
    path: "/bytebase.v1.RolloutService/ListTaskRuns",
    description: "List task runs in a rollout",
  },

  // SQL
  {
    id: "bytebase.v1.SQLService.Query",
    path: "/bytebase.v1.SQLService/Query",
    description: "Execute a SQL query against a database",
  },
  {
    id: "bytebase.v1.SQLService.Export",
    path: "/bytebase.v1.SQLService/Export",
    description: "Export query results",
  },
  {
    id: "bytebase.v1.SQLService.SearchQueryHistories",
    path: "/bytebase.v1.SQLService/SearchQueryHistories",
    description: "Search SQL query history",
  },

  // Sheet
  {
    id: "bytebase.v1.SheetService.CreateSheet",
    path: "/bytebase.v1.SheetService/CreateSheet",
    description: "Create a SQL sheet",
  },
  {
    id: "bytebase.v1.SheetService.GetSheet",
    path: "/bytebase.v1.SheetService/GetSheet",
    description: "Get a SQL sheet by name",
  },

  // User
  {
    id: "bytebase.v1.UserService.ListUsers",
    path: "/bytebase.v1.UserService/ListUsers",
    description: "List all users",
  },
  {
    id: "bytebase.v1.UserService.GetUser",
    path: "/bytebase.v1.UserService/GetUser",
    description: "Get a user by name",
  },
  {
    id: "bytebase.v1.UserService.GetCurrentUser",
    path: "/bytebase.v1.UserService/GetCurrentUser",
    description: "Get the currently authenticated user",
  },
  {
    id: "bytebase.v1.UserService.CreateUser",
    path: "/bytebase.v1.UserService/CreateUser",
    description: "Create a new user",
  },
  {
    id: "bytebase.v1.UserService.UpdateUser",
    path: "/bytebase.v1.UserService/UpdateUser",
    description: "Update a user",
  },

  // Setting
  {
    id: "bytebase.v1.SettingService.ListSettings",
    path: "/bytebase.v1.SettingService/ListSettings",
    description: "List workspace settings",
  },
  {
    id: "bytebase.v1.SettingService.GetSetting",
    path: "/bytebase.v1.SettingService/GetSetting",
    description: "Get a workspace setting by name",
  },
  {
    id: "bytebase.v1.SettingService.UpdateSetting",
    path: "/bytebase.v1.SettingService/UpdateSetting",
    description: "Update a workspace setting",
  },

  // Policy
  {
    id: "bytebase.v1.OrgPolicyService.ListPolicies",
    path: "/bytebase.v1.OrgPolicyService/ListPolicies",
    description: "List policies for a resource",
  },
  {
    id: "bytebase.v1.OrgPolicyService.GetPolicy",
    path: "/bytebase.v1.OrgPolicyService/GetPolicy",
    description: "Get a policy by name",
  },
  {
    id: "bytebase.v1.OrgPolicyService.CreatePolicy",
    path: "/bytebase.v1.OrgPolicyService/CreatePolicy",
    description: "Create a policy",
  },
  {
    id: "bytebase.v1.OrgPolicyService.UpdatePolicy",
    path: "/bytebase.v1.OrgPolicyService/UpdatePolicy",
    description: "Update a policy",
  },

  // Worksheet
  {
    id: "bytebase.v1.WorksheetService.CreateWorksheet",
    path: "/bytebase.v1.WorksheetService/CreateWorksheet",
    description: "Create a worksheet (saved SQL)",
  },
  {
    id: "bytebase.v1.WorksheetService.GetWorksheet",
    path: "/bytebase.v1.WorksheetService/GetWorksheet",
    description: "Get a worksheet",
  },
  {
    id: "bytebase.v1.WorksheetService.SearchWorksheets",
    path: "/bytebase.v1.WorksheetService/SearchWorksheets",
    description: "Search worksheets",
  },
  {
    id: "bytebase.v1.WorksheetService.UpdateWorksheet",
    path: "/bytebase.v1.WorksheetService/UpdateWorksheet",
    description: "Update a worksheet",
  },
  {
    id: "bytebase.v1.WorksheetService.DeleteWorksheet",
    path: "/bytebase.v1.WorksheetService/DeleteWorksheet",
    description: "Delete a worksheet",
  },

  // Database Group
  {
    id: "bytebase.v1.DatabaseGroupService.ListDatabaseGroups",
    path: "/bytebase.v1.DatabaseGroupService/ListDatabaseGroups",
    description: "List database groups in a project",
  },
  {
    id: "bytebase.v1.DatabaseGroupService.GetDatabaseGroup",
    path: "/bytebase.v1.DatabaseGroupService/GetDatabaseGroup",
    description: "Get a database group",
  },

  // Workspace IAM
  {
    id: "bytebase.v1.WorkspaceService.GetIamPolicy",
    path: "/bytebase.v1.WorkspaceService/GetIamPolicy",
    description: "Get workspace IAM policy",
  },
  {
    id: "bytebase.v1.WorkspaceService.SetIamPolicy",
    path: "/bytebase.v1.WorkspaceService/SetIamPolicy",
    description: "Set workspace IAM policy",
  },

  // Audit Log
  {
    id: "bytebase.v1.AuditLogService.SearchAuditLogs",
    path: "/bytebase.v1.AuditLogService/SearchAuditLogs",
    description: "Search audit logs",
  },

  // Access Grant
  {
    id: "bytebase.v1.AccessGrantService.ListAccessGrants",
    path: "/bytebase.v1.AccessGrantService/ListAccessGrants",
    description: "List access grants in a project",
  },
  {
    id: "bytebase.v1.AccessGrantService.CreateAccessGrant",
    path: "/bytebase.v1.AccessGrantService/CreateAccessGrant",
    description: "Create an access grant (request database permission)",
  },

  // Subscription
  {
    id: "bytebase.v1.SubscriptionService.GetSubscription",
    path: "/bytebase.v1.SubscriptionService/GetSubscription",
    description: "Get workspace subscription and license info",
  },

  // Actuator
  {
    id: "bytebase.v1.ActuatorService.GetActuatorInfo",
    path: "/bytebase.v1.ActuatorService/GetActuatorInfo",
    description: "Get server version and system info",
  },

  // Instance Role
  {
    id: "bytebase.v1.InstanceRoleService.ListInstanceRoles",
    path: "/bytebase.v1.InstanceRoleService/ListInstanceRoles",
    description: "List database roles in an instance",
  },

  // Group
  {
    id: "bytebase.v1.GroupService.ListGroups",
    path: "/bytebase.v1.GroupService/ListGroups",
    description: "List user groups",
  },
  {
    id: "bytebase.v1.GroupService.GetGroup",
    path: "/bytebase.v1.GroupService/GetGroup",
    description: "Get a user group",
  },
  {
    id: "bytebase.v1.GroupService.CreateGroup",
    path: "/bytebase.v1.GroupService/CreateGroup",
    description: "Create a user group",
  },
];

export function getApiOperations(): ApiOperation[] {
  return API_OPERATIONS;
}

export async function searchApi(args: { query: string }): Promise<string> {
  const query = args.query.toLowerCase();
  const tokens = query.split(/\s+/).filter(Boolean);

  const scored = API_OPERATIONS.map((op) => {
    let score = 0;
    const haystack = `${op.description} ${op.id} ${op.path}`.toLowerCase();
    for (const token of tokens) {
      if (haystack.includes(token)) {
        score += 1;
      }
    }
    return { op, score };
  });

  const matches = scored
    .filter((s) => s.score > 0)
    .sort((a, b) => b.score - a.score)
    .slice(0, 10)
    .map((s) => s.op);

  return JSON.stringify(matches);
}
