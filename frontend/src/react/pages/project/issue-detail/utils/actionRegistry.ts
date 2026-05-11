import type { TFunction } from "i18next";
import { first, orderBy } from "lodash-es";
import { candidatesOfApprovalStepV1, extractUserEmail } from "@/store";
import { ApprovalStatus, State } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
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
import { isApprovalCompleted } from "./approval";

export type IssueReviewAction = "ISSUE_REVIEW";
export type IssueStatusAction = "ISSUE_STATUS_CLOSE" | "ISSUE_STATUS_REOPEN";
export type IssueAction =
  | IssueReviewAction
  | IssueStatusAction
  | "ISSUE_CREATE";
export type RolloutAction = "ROLLOUT_START" | "ROLLOUT_CANCEL";
export type ExportAction = "EXPORT_DOWNLOAD";
export type UnifiedAction = IssueAction | RolloutAction | ExportAction;
export type ExecuteType =
  | "immediate"
  | "popover:labels"
  | "popover:review"
  | "panel:issue-status"
  | "panel:rollout";
export type ActionCategory = "primary" | "secondary";

export interface ActionPermissions {
  updatePlan: boolean;
  createIssue: boolean;
  updateIssue: boolean;
  createRollout: boolean;
  runTasks: boolean;
  isApprovalCandidate: boolean;
}

export interface ActionValidation {
  hasEmptySpec: boolean;
  planChecksRunning: boolean;
  planChecksFailed: boolean;
}

export interface ActionContext {
  plan: Plan;
  issue: Issue | undefined;
  rollout: Rollout | undefined;
  project: Project;
  planState: State;
  issueStatus: IssueStatus | undefined;
  approvalStatus: ApprovalStatus | undefined;
  isCreating: boolean;
  isIssueOnly: boolean;
  isExportPlan: boolean;
  isReleasePlan: boolean;
  hasDeferredRollout: boolean;
  isCreator: boolean;
  issueApproved: boolean;
  exportArchiveReady: boolean;
  allTasksFinished: boolean;
  hasDatabaseCreateOrExportTasks: boolean;
  hasStartableTasks: boolean;
  hasRunningTasks: boolean;
  permissions: ActionPermissions;
  validation: ActionValidation;
}

export interface ActionDefinition {
  id: UnifiedAction;
  label: (ctx: ActionContext) => string;
  buttonType: "primary" | "success" | "default";
  category: ActionCategory | ((ctx: ActionContext) => ActionCategory);
  priority: number;
  isVisible: (ctx: ActionContext) => boolean;
  isDisabled: (ctx: ActionContext) => boolean;
  disabledReason: (ctx: ActionContext) => string | undefined;
  executeType: ExecuteType;
}

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

const computeIsApprovalCandidate = (
  issue: Issue | undefined,
  currentUser: User,
  project: Project
): boolean => {
  if (!issue) return false;
  if (isApprovalCompleted(issue)) return false;

  const { approvers, approvalTemplate } = issue;
  const hasRejection = approvers.some(
    (app) => app.status === Issue_Approver_Status.REJECTED
  );
  if (hasRejection) return false;

  const roles = approvalTemplate?.flow?.roles ?? [];
  if (roles.length === 0) return false;

  const rejectedIndex = approvers.findIndex(
    (approver) => approver.status === Issue_Approver_Status.REJECTED
  );
  const currentRoleIndex =
    rejectedIndex >= 0 ? rejectedIndex : approvers.length;
  const currentRole = roles[currentRoleIndex];
  if (!currentRole) return false;

  const candidates = candidatesOfApprovalStepV1(issue, currentRole);
  if (!isUserIncludedInList(currentUser.email, candidates)) return false;

  if (
    !project.allowSelfApproval &&
    currentUser.email === extractUserEmail(issue.creator)
  ) {
    return false;
  }
  return true;
};

const computeExportArchiveReady = (
  rollout: Rollout | undefined,
  taskRuns: TaskRun[],
  issue: Issue | undefined,
  currentUserEmail: string
): boolean => {
  if (!issue) return false;
  if (![IssueStatus.OPEN, IssueStatus.DONE].includes(issue.status))
    return false;
  if (currentUserEmail !== extractUserEmail(issue.creator)) return false;

  const exportTasks =
    rollout?.stages
      .flatMap((stage) => stage.tasks)
      .filter((task) => task.type === Task_Type.DATABASE_EXPORT) ?? [];
  if (exportTasks.length === 0) return false;
  if (
    !exportTasks.every((task) =>
      [Task_Status.DONE, Task_Status.SKIPPED].includes(task.status)
    )
  ) {
    return false;
  }

  const doneTasks = exportTasks.filter(
    (task) => task.status === Task_Status.DONE
  );
  if (doneTasks.length === 0) return false;

  const exportTaskRuns = doneTasks
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
};

export const buildIssueDetailActionContext = (
  input: ContextBuilderInput
): ActionContext => {
  const {
    currentUser,
    hasRunningPlanChecks,
    isCreating,
    isSpecEmpty,
    issue,
    plan,
    planCheckStatus,
    project,
    rollout,
    taskRuns,
  } = input;

  const currentUserEmail = currentUser.email;
  const isIssueOnly =
    !isValidPlanName(plan.name) && Boolean(isValidIssueName(issue?.name));
  const isExportPlan = plan.specs.some(
    (spec) => spec.config?.case === "exportDataConfig"
  );
  const isReleasePlan = plan.specs.some(
    (spec) =>
      spec.config?.case === "changeDatabaseConfig" &&
      Boolean(spec.config.value.release)
  );
  const hasDeferredRollout = plan.specs.some(
    (spec) =>
      spec.config?.case === "exportDataConfig" ||
      spec.config?.case === "createDatabaseConfig"
  );
  const isCreator =
    currentUserEmail === extractUserEmail(plan.creator || "") ||
    (issue ? currentUserEmail === extractUserEmail(issue.creator) : false);

  const permissions: ActionPermissions = {
    updatePlan:
      currentUserEmail === extractUserEmail(plan.creator || "") ||
      hasProjectPermissionV2(project, "bb.plans.update"),
    createIssue: hasProjectPermissionV2(project, "bb.issues.create"),
    updateIssue: hasProjectPermissionV2(project, "bb.issues.update"),
    createRollout: hasProjectPermissionV2(project, "bb.rollouts.create"),
    runTasks:
      issue?.type === Issue_Type.DATABASE_EXPORT
        ? currentUserEmail === extractUserEmail(issue.creator)
        : hasProjectPermissionV2(project, "bb.taskRuns.create"),
    isApprovalCandidate: computeIsApprovalCandidate(
      issue,
      currentUser,
      project
    ),
  };

  const validation: ActionValidation = {
    hasEmptySpec: plan.specs.some((spec) => isSpecEmpty(spec)),
    planChecksRunning: hasRunningPlanChecks,
    planChecksFailed: planCheckStatus === Advice_Level.ERROR,
  };

  const allTasks = rollout?.stages.flatMap((stage) => stage.tasks) ?? [];
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
    isReleasePlan,
    hasDeferredRollout,
    isCreator,
    issueApproved: isApprovalCompleted(issue),
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
};

export const createIssueDetailActions = (t: TFunction): ActionDefinition[] => {
  const issueActions: ActionDefinition[] = [
    {
      id: "ISSUE_CREATE",
      label: () => t("plan.ready-for-review"),
      buttonType: "primary",
      category: "primary",
      priority: 5,
      isVisible: (ctx) =>
        !ctx.isIssueOnly &&
        !ctx.isReleasePlan &&
        ctx.plan.issue === "" &&
        !ctx.plan.hasRollout &&
        ctx.planState === State.ACTIVE,
      isDisabled: (ctx) =>
        !ctx.permissions.createIssue ||
        ctx.validation.hasEmptySpec ||
        ctx.validation.planChecksRunning ||
        (ctx.validation.planChecksFailed && ctx.project.enforceSqlReview),
      disabledReason: (ctx) => {
        if (!ctx.permissions.createIssue) {
          return t("common.missing-required-permission", {
            permissions: "bb.issues.create",
          });
        }
        if (ctx.validation.hasEmptySpec) {
          return t("plan.navigator.statement-empty");
        }
        if (ctx.validation.planChecksRunning) {
          return t(
            "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
          );
        }
        if (ctx.validation.planChecksFailed && ctx.project.enforceSqlReview) {
          return t(
            "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
          );
        }
        return undefined;
      },
      executeType: "popover:labels",
    },
    {
      id: "ISSUE_REVIEW",
      label: () => t("issue.review.self"),
      buttonType: "primary",
      category: "primary",
      priority: 30,
      isVisible: (ctx) =>
        ctx.issueStatus === IssueStatus.OPEN &&
        ctx.approvalStatus !== ApprovalStatus.APPROVED &&
        ctx.approvalStatus !== ApprovalStatus.SKIPPED &&
        ctx.permissions.isApprovalCandidate,
      isDisabled: () => false,
      disabledReason: () => undefined,
      executeType: "popover:review",
    },
    {
      id: "ISSUE_STATUS_CLOSE",
      label: () => t("issue.batch-transition.close"),
      buttonType: "default",
      category: "secondary",
      priority: 90,
      isVisible: (ctx) =>
        ctx.issueStatus === IssueStatus.OPEN && !ctx.plan.hasRollout,
      isDisabled: (ctx) => !ctx.permissions.updateIssue,
      disabledReason: (ctx) => {
        if (!ctx.permissions.updateIssue) {
          return t("common.missing-required-permission", {
            permissions: "bb.issues.update",
          });
        }
        return undefined;
      },
      executeType: "panel:issue-status",
    },
    {
      id: "ISSUE_STATUS_REOPEN",
      label: () => t("issue.batch-transition.reopen"),
      buttonType: "default",
      category: "primary",
      priority: 20,
      isVisible: (ctx) => ctx.issueStatus === IssueStatus.CANCELED,
      isDisabled: (ctx) => !ctx.permissions.updateIssue,
      disabledReason: (ctx) => {
        if (!ctx.permissions.updateIssue) {
          return t("common.missing-required-permission", {
            permissions: "bb.issues.update",
          });
        }
        return undefined;
      },
      executeType: "panel:issue-status",
    },
  ];

  const rolloutActions: ActionDefinition[] = [
    {
      id: "ROLLOUT_START",
      label: (ctx) =>
        ctx.isExportPlan ? t("common.export") : t("common.rollout"),
      buttonType: "primary",
      category: "primary",
      priority: 60,
      isVisible: (ctx) => {
        if (!ctx.hasDeferredRollout) return false;
        if (!ctx.issue || !ctx.issueApproved) return false;
        if (!ctx.rollout) return true;
        return ctx.hasStartableTasks;
      },
      isDisabled: (ctx) => !ctx.permissions.runTasks,
      disabledReason: (ctx) => {
        if (!ctx.permissions.runTasks) {
          if (ctx.isExportPlan) {
            return t("common.only-creator-allowed-export");
          }
          return t("common.missing-required-permission", {
            permissions: "bb.taskRuns.create",
          });
        }
        return undefined;
      },
      executeType: "panel:rollout",
    },
    {
      id: "ROLLOUT_CANCEL",
      label: () => t("common.cancel"),
      buttonType: "default",
      category: "secondary",
      priority: 80,
      isVisible: (ctx) => {
        if (!ctx.hasDeferredRollout) return false;
        if (!ctx.rollout) return false;
        if (!ctx.issueApproved) return false;
        if (!ctx.hasRunningTasks) return false;
        return true;
      },
      isDisabled: (ctx) => !ctx.permissions.runTasks,
      disabledReason: (ctx) => {
        if (!ctx.permissions.runTasks) {
          return t("common.missing-required-permission", {
            permissions: "bb.taskRuns.create",
          });
        }
        return undefined;
      },
      executeType: "panel:rollout",
    },
    {
      id: "EXPORT_DOWNLOAD",
      label: () => t("common.download"),
      buttonType: "primary",
      category: "primary",
      priority: 0,
      isVisible: (ctx) =>
        ctx.isExportPlan && ctx.exportArchiveReady && ctx.isCreator,
      isDisabled: () => false,
      disabledReason: () => undefined,
      executeType: "immediate",
    },
  ];

  return [...issueActions, ...rolloutActions].sort(
    (left, right) => left.priority - right.priority
  );
};
