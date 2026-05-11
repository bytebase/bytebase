import { ApprovalStatus, State } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";

type T = (key: string, options?: Record<string, unknown>) => string;

const TASK_STATUS_FILTERS: Task_Status[] = [
  Task_Status.RUNNING,
  Task_Status.FAILED,
  Task_Status.PENDING,
  Task_Status.NOT_STARTED,
  Task_Status.CANCELED,
  Task_Status.DONE,
  Task_Status.SKIPPED,
];

export interface SidebarStatusInfo {
  dotClass: string;
  label: string;
}

export const getPlanDetailSidebarStatusInfo = ({
  isCreating,
  issue,
  planState,
  rollout,
  t,
}: {
  isCreating: boolean;
  issue?: Pick<Issue, "approvalStatus" | "status">;
  planState: State;
  rollout?: Rollout;
  t: T;
}): SidebarStatusInfo => {
  if (isCreating) {
    return { dotClass: "bg-control-placeholder", label: t("common.creating") };
  }
  if (planState === State.DELETED) {
    return { dotClass: "bg-control-placeholder", label: t("common.closed") };
  }
  if (issue?.status === IssueStatus.CANCELED) {
    return { dotClass: "bg-control-placeholder", label: t("common.closed") };
  }
  if (rollout) {
    return getRolloutStatusInfo(rollout, t);
  }
  return getReviewStatusInfo(issue, t);
};

const getReviewStatusInfo = (
  issue: Pick<Issue, "approvalStatus" | "status"> | undefined,
  t: T
): SidebarStatusInfo => {
  if (!issue) {
    return { dotClass: "bg-control-placeholder", label: t("common.draft") };
  }
  if (issue.status === IssueStatus.DONE) {
    return { dotClass: "bg-success", label: t("common.approved") };
  }

  switch (issue.approvalStatus) {
    case ApprovalStatus.APPROVED:
    case ApprovalStatus.SKIPPED:
      return { dotClass: "bg-success", label: t("common.approved") };
    case ApprovalStatus.REJECTED:
      return { dotClass: "bg-warning", label: t("common.rejected") };
    default:
      return { dotClass: "bg-accent", label: t("common.in-review") };
  }
};

const getRolloutStatusInfo = (rollout: Rollout, t: T): SidebarStatusInfo => {
  const hasCompletedTasks = rollout.stages
    .flatMap((stage) => stage.tasks)
    .some(
      (task) =>
        task.status === Task_Status.DONE || task.status === Task_Status.SKIPPED
    );

  switch (getRolloutStatus(rollout)) {
    case Task_Status.DONE:
    case Task_Status.SKIPPED:
      return { dotClass: "bg-success", label: t("common.done") };
    case Task_Status.FAILED:
      return { dotClass: "bg-error", label: t("common.failed") };
    case Task_Status.RUNNING:
    case Task_Status.PENDING:
      return { dotClass: "bg-accent", label: t("common.deploying") };
    case Task_Status.CANCELED:
      return {
        dotClass: "bg-control-placeholder",
        label: t("common.canceled"),
      };
    case Task_Status.NOT_STARTED:
      if (hasCompletedTasks) {
        return { dotClass: "bg-accent", label: t("common.deploying") };
      }
      return {
        dotClass: "bg-control-placeholder",
        label: t("common.not-started"),
      };
    default:
      return {
        dotClass: "bg-control-placeholder",
        label: t("common.not-started"),
      };
  }
};

const getStageStatus = (tasks: Array<{ status: Task_Status }>): Task_Status => {
  if (tasks.length === 0) return Task_Status.NOT_STARTED;
  for (const status of TASK_STATUS_FILTERS) {
    if (tasks.some((task) => task.status === status)) {
      return status;
    }
  }
  return Task_Status.NOT_STARTED;
};

const getRolloutStatus = (rollout: Rollout): Task_Status => {
  if (rollout.stages.length === 0) return Task_Status.NOT_STARTED;
  for (const status of TASK_STATUS_FILTERS) {
    if (
      rollout.stages.some((stage) => getStageStatus(stage.tasks) === status)
    ) {
      return status;
    }
  }
  return Task_Status.NOT_STARTED;
};
