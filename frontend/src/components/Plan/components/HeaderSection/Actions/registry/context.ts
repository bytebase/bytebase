import { first, orderBy } from "lodash-es";
import { candidatesOfApprovalStepV1, extractUserId } from "@/store";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  Issue_ApprovalStatus,
  Issue_Approver_Status,
  Issue_Type,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Rollout, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import {
  Task_Status,
  Task_Type,
  TaskRun_ExportArchiveStatus,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  extractTaskRunUID,
  extractTaskUID,
  hasProjectPermissionV2,
  isUserIncludedInList,
  isValidIssueName,
  isValidPlanName,
} from "@/utils";
import type {
  ActionContext,
  ActionPermissions,
  ActionValidation,
} from "./types";

export interface ContextBuilderInput {
  plan: Plan;
  issue: Issue | undefined;
  rollout: Rollout | undefined;
  project: Project;
  currentUser: User;
  taskRuns: TaskRun[];
  isCreating: boolean;
  planCheckStatus: Advice_Level;
  hasRunningPlanChecks: boolean;
  isSpecEmpty: (spec: Plan["specs"][0]) => boolean;
}

function computeIsApprovalCandidate(
  issue: Issue | undefined,
  currentUser: User
): boolean {
  if (!issue) return false;

  const issueApproved =
    issue.approvalStatus === Issue_ApprovalStatus.APPROVED ||
    issue.approvalStatus === Issue_ApprovalStatus.SKIPPED;
  if (issueApproved) return false;

  const { approvers, approvalTemplate } = issue;
  const hasRejection = approvers.some(
    (app: Issue["approvers"][0]) =>
      app.status === Issue_Approver_Status.REJECTED
  );

  // Cannot approve or reject if already rejected
  if (hasRejection) return false;

  const roles = approvalTemplate?.flow?.roles ?? [];
  if (roles.length === 0) return false;

  const rejectedIndex = approvers.findIndex(
    (ap: Issue["approvers"][0]) => ap.status === Issue_Approver_Status.REJECTED
  );
  const currentRoleIndex =
    rejectedIndex >= 0 ? rejectedIndex : approvers.length;
  const currentRole = roles[currentRoleIndex];
  if (!currentRole) return false;

  const candidates = candidatesOfApprovalStepV1(issue, currentRole);
  return isUserIncludedInList(currentUser.email, candidates);
}

function computeExportArchiveReady(
  rollout: Rollout | undefined,
  taskRuns: TaskRun[],
  issue: Issue | undefined,
  currentUserEmail: string
): boolean {
  if (!issue) return false;
  if (![IssueStatus.OPEN, IssueStatus.DONE].includes(issue.status))
    return false;
  if (currentUserEmail !== extractUserId(issue.creator)) return false;

  const exportTasks =
    rollout?.stages
      .flatMap((stage) => stage.tasks)
      .filter((task) => task.type === Task_Type.DATABASE_EXPORT) || [];

  if (exportTasks.length === 0) return false;

  if (
    !exportTasks.every((task) =>
      [Task_Status.DONE, Task_Status.SKIPPED].includes(task.status)
    )
  ) {
    return false;
  }

  const exportTaskRuns = exportTasks
    .map((task) => {
      const taskRunsForTask = taskRuns.filter(
        (taskRun) => extractTaskUID(taskRun.name) === extractTaskUID(task.name)
      );
      return first(
        orderBy(
          taskRunsForTask,
          (taskRun) => Number(extractTaskRunUID(taskRun.name)),
          "desc"
        )
      );
    })
    .filter(Boolean) as TaskRun[];

  if (
    exportTaskRuns.length === 0 ||
    exportTaskRuns.some(
      (taskRun) =>
        taskRun.status !== TaskRun_Status.DONE ||
        taskRun.exportArchiveStatus ===
          TaskRun_ExportArchiveStatus.EXPORT_ARCHIVE_STATUS_UNSPECIFIED
    )
  ) {
    return false;
  }

  return true;
}

export function buildActionContext(input: ContextBuilderInput): ActionContext {
  const {
    plan,
    issue,
    rollout,
    project,
    currentUser,
    taskRuns,
    isCreating,
    planCheckStatus,
    hasRunningPlanChecks,
    isSpecEmpty,
  } = input;

  const currentUserEmail = currentUser.email;
  const isIssueOnly =
    !isValidPlanName(plan.name) && !!isValidIssueName(issue?.name);
  const isExportPlan = plan.specs.some(
    (spec) => spec.config?.case === "exportDataConfig"
  );
  // Plans where rollout is created on-demand when user clicks action button
  const hasDeferredRollout = plan.specs.some(
    (spec) =>
      spec.config?.case === "exportDataConfig" ||
      spec.config?.case === "createDatabaseConfig"
  );
  const isCreator =
    currentUserEmail === extractUserId(plan.creator || "") ||
    (issue ? currentUserEmail === extractUserId(issue.creator) : false);

  // Compute permissions
  const isApprovalCandidate = computeIsApprovalCandidate(issue, currentUser);
  const permissions: ActionPermissions = {
    updatePlan:
      currentUserEmail === extractUserId(plan.creator || "") ||
      hasProjectPermissionV2(project, "bb.plans.update"),
    createIssue: hasProjectPermissionV2(project, "bb.issues.create"),
    updateIssue: hasProjectPermissionV2(project, "bb.issues.update"),
    createRollout: hasProjectPermissionV2(project, "bb.rollouts.create"),
    runTasks:
      issue?.type === Issue_Type.DATABASE_EXPORT
        ? currentUserEmail === extractUserId(issue.creator)
        : hasProjectPermissionV2(project, "bb.taskRuns.create"),
    isApprovalCandidate,
    canApprove: isApprovalCandidate,
    canReject: isApprovalCandidate,
  };

  // Compute validation state
  const validation: ActionValidation = {
    hasEmptySpec: plan.specs.some((spec) => isSpecEmpty(spec)),
    planChecksRunning: hasRunningPlanChecks,
    planChecksFailed: planCheckStatus === Advice_Level.ERROR,
  };

  // Compute task-related flags
  const allTasks = rollout?.stages.flatMap((stage) => stage.tasks) || [];

  const allTasksFinished = allTasks.every((task) =>
    [Task_Status.DONE, Task_Status.SKIPPED].includes(task.status)
  );

  const hasDatabaseCreateOrExportTasks = allTasks.some(
    (task) =>
      task.type === Task_Type.DATABASE_CREATE ||
      task.type === Task_Type.DATABASE_EXPORT
  );

  const hasStartableTasks = allTasks.some((task) =>
    [
      Task_Status.NOT_STARTED,
      Task_Status.FAILED,
      Task_Status.CANCELED,
    ].includes(task.status)
  );

  const hasRunningTasks = allTasks.some((task) =>
    [Task_Status.PENDING, Task_Status.RUNNING].includes(task.status)
  );

  // Compute approval status
  const issueApproved =
    issue?.approvalStatus === Issue_ApprovalStatus.APPROVED ||
    issue?.approvalStatus === Issue_ApprovalStatus.SKIPPED;

  return {
    plan,
    issue,
    rollout,
    project,

    planState: plan.state,
    issueStatus: issue?.status,
    approvalStatus: issue?.approvalStatus,

    isCreating,
    isIssueOnly,
    isExportPlan,
    hasDeferredRollout,
    isCreator,
    issueApproved,
    exportArchiveReady: computeExportArchiveReady(
      rollout,
      taskRuns,
      issue,
      currentUserEmail
    ),
    allTasksFinished,
    hasDatabaseCreateOrExportTasks,
    hasStartableTasks,
    hasRunningTasks,

    permissions,
    validation,
  };
}
