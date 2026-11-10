import { create } from "@bufbuild/protobuf";
import { auditLogServiceClientConnect } from "@/api";
import { getTimeForPbTimestampProtoEs } from "@/types";
import {
  type AuditLog,
  SearchAuditLogsRequestSchema,
} from "@/types/proto-es/v1/audit_log_service_pb";

const PAGE_SIZE = 500;
const MAX_EVENTS = 5;
const RECENT_ACTIVITY_WINDOW_MS = 3 * 24 * 60 * 60 * 1000;

const actionByMethod: Record<string, string> = {
  "/bytebase.v1.DatabaseService/CreateDatabase": "created a database",
  "/bytebase.v1.DatabaseService/UpdateDatabase": "updated a database",
  "/bytebase.v1.IssueService/CreateIssue": "created an Issue",
  "/bytebase.v1.IssueService/UpdateIssue": "updated an Issue",
  "/bytebase.v1.IssueService/CreateIssueComment": "commented on an Issue",
  "/bytebase.v1.IssueService/UpdateIssueComment": "updated an Issue comment",
  "/bytebase.v1.IssueService/BatchUpdateIssuesStatus": "updated Issue status",
  "/bytebase.v1.IssueService/ApproveIssue": "approved an Issue",
  "/bytebase.v1.IssueService/RejectIssue": "rejected an Issue",
  "/bytebase.v1.IssueService/RequestIssue": "requested changes to an Issue",
  "/bytebase.v1.IssueService/RunReview": "ran an Issue review",
  "/bytebase.v1.PlanService/CreatePlan": "created a plan",
  "/bytebase.v1.PlanService/UpdatePlan": "updated a plan",
  "/bytebase.v1.PlanService/RunPlanChecks": "ran plan checks",
  "/bytebase.v1.PlanService/CancelPlanCheckRun": "canceled plan checks",
  "/bytebase.v1.ProjectService/UpdateProject": "updated project settings",
  "/bytebase.v1.ProjectService/SetIamPolicy": "updated project access",
  "/bytebase.v1.ProjectService/CreateProject": "created a project",
  "/bytebase.v1.ProjectService/DeleteProject": "deleted a project",
  "/bytebase.v1.ProjectService/UndeleteProject": "restored a project",
  "/bytebase.v1.WorkspaceService/UpdateWorkspace": "updated workspace settings",
  "/bytebase.v1.WorkspaceService/SetIamPolicy": "updated workspace access",
  "/bytebase.v1.SettingService/UpdateSetting": "updated a workspace setting",
  "/bytebase.v1.OrgPolicyService/CreatePolicy": "created a policy",
  "/bytebase.v1.OrgPolicyService/UpdatePolicy": "updated a policy",
  "/bytebase.v1.OrgPolicyService/DeletePolicy": "deleted a policy",
  "/bytebase.v1.UserService/CreateUser": "created a user",
  "/bytebase.v1.UserService/UpdateUser": "updated a user",
  "/bytebase.v1.UserService/DeleteUser": "deleted a user",
  "/bytebase.v1.UserService/UndeleteUser": "restored a user",
  "/bytebase.v1.UserService/UpdateEmail": "updated a user's email",
  "/bytebase.v1.RoleService/CreateRole": "created a role",
  "/bytebase.v1.RoleService/UpdateRole": "updated a role",
  "/bytebase.v1.RoleService/DeleteRole": "deleted a role",
  "/bytebase.v1.GroupService/CreateGroup": "created a group",
  "/bytebase.v1.GroupService/UpdateGroup": "updated a group",
  "/bytebase.v1.GroupService/DeleteGroup": "deleted a group",
  "/bytebase.v1.RolloutService/CreateRollout": "created a rollout",
  "/bytebase.v1.RolloutService/BatchRunTasks": "started rollout tasks",
  "/bytebase.v1.RolloutService/BatchSkipTasks": "skipped rollout tasks",
  "/bytebase.v1.RolloutService/BatchCancelTaskRuns": "canceled task runs",
};

function summarizeAuditLog(log: AuditLog) {
  const timestamp = getTimeForPbTimestampProtoEs(log.createTime);
  return {
    ...(timestamp !== undefined
      ? { occurredAt: new Date(timestamp).toISOString() }
      : {}),
    actor: log.actor,
    action: actionByMethod[log.method] ?? "recorded activity",
    resource: log.resource,
    outcome: log.status?.code ? "failed" : "succeeded",
  };
}

function isMeaningfulAuditLog(log: AuditLog): boolean {
  return actionByMethod[log.method] !== undefined;
}

export async function getRecentActivity(parent: string): Promise<string> {
  try {
    const response = await auditLogServiceClientConnect.searchAuditLogs(
      create(SearchAuditLogsRequestSchema, {
        parent,
        filter: `create_time >= '${new Date(
          Date.now() - RECENT_ACTIVITY_WINDOW_MS
        ).toISOString()}'`,
        orderBy: "create_time desc",
        pageSize: PAGE_SIZE,
      })
    );
    const auditLogs = response.auditLogs ?? [];
    const meaningfulAuditLogs = auditLogs.filter(isMeaningfulAuditLog);
    return JSON.stringify({
      events: meaningfulAuditLogs.slice(0, MAX_EVENTS).map(summarizeAuditLog),
      partial:
        Boolean(response.nextPageToken) ||
        meaningfulAuditLogs.length > MAX_EVENTS,
    });
  } catch {
    return JSON.stringify({ error: "Unable to load recent activity." });
  }
}
