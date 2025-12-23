import { computed } from "vue";
import { usePlanContext } from "@/components/Plan/logic";
import { t } from "@/plugins/i18n";
import {
  candidatesOfApprovalStepV1,
  extractUserId,
  useCurrentProjectV1,
  useCurrentUserV1,
} from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  Issue_ApprovalStatus,
  Issue_Approver_Status,
  Issue_Type,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { Task_Status, Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import {
  hasProjectPermissionV2,
  isUserIncludedInList,
  isValidIssueName,
  isValidPlanName,
} from "@/utils";
import { type UnifiedAction } from "./types";

export const usePlanAction = () => {
  const currentUser = useCurrentUserV1();
  const { project } = useCurrentProjectV1();

  const { isCreating, plan, issue, rollout } = usePlanContext();

  /**
   * Compute available actions based on issue state and user permissions.
   *
   * Action priority order:
   * 1. Plan-level actions (for draft plans): PLAN_REOPEN, PLAN_CLOSE, ISSUE_CREATE
   * 2. Issue review actions: ISSUE_REVIEW_APPROVE, ISSUE_REVIEW_REJECT, ISSUE_REVIEW_RE_REQUEST
   * 3. Issue status actions: ISSUE_STATUS_RESOLVE, ISSUE_STATUS_CLOSE, ISSUE_STATUS_REOPEN
   * 4. Rollout actions: ROLLOUT_START, ROLLOUT_CANCEL
   *
   * Special cases:
   * - Grant requests: Issue-only (no plan), only show issue-related actions
   * - Draft plans: Plans without issues, show plan management + issue creation
   * - Deleted plans: Only show PLAN_REOPEN
   * - Cancelled/Done issues: Only show ISSUE_STATUS_REOPEN
   */
  const availableActions = computed(() => {
    const actions: UnifiedAction[] = [];

    if (isCreating.value) return actions;

    const currentUserEmail = currentUser.value.email;
    const isIssueOnly =
      !isValidPlanName(plan.value.name) && isValidIssueName(issue.value?.name);

    // For issue-only cases (grant requests, etc.), skip plan-specific actions
    // and go directly to issue-related actions
    if (isIssueOnly) {
      // Issue-only cases should not show plan actions
      // Continue to issue-related actions below
    }
    // If no issue exists, show create issue action or close plan action.
    else if (plan.value.issue === "") {
      // If rollout exists, no actions are available.
      if (plan.value.rollout !== "") {
        return actions;
      }

      const canUpdatePlan =
        currentUserEmail === extractUserId(plan.value.creator || "") ||
        hasProjectPermissionV2(project.value, "bb.plans.update");

      // Check if user can reopen the plan
      if (plan.value.state === State.DELETED) {
        if (canUpdatePlan) {
          actions.push("PLAN_REOPEN");
        }
        return actions; // For deleted plans, only show reopen action
      }

      // Check if user can close the plan
      if (canUpdatePlan) {
        actions.push("PLAN_CLOSE");
      }

      // Check if user can create an issue
      if (hasProjectPermissionV2(project.value, "bb.issues.create")) {
        actions.push("ISSUE_CREATE");
      }

      return actions;
    }

    const issueValue = issue?.value;
    // Should not reach here.
    if (!issueValue) return actions;
    const issueCreator = extractUserId(issueValue.creator);
    const isCanceled = issueValue.status === IssueStatus.CANCELED;
    const isDone = issueValue.status === IssueStatus.DONE;
    const canUpdateIssue = hasProjectPermissionV2(
      project.value,
      "bb.issues.update"
    );

    // If issue is canceled, check for re-open action
    if (isCanceled) {
      if (canUpdateIssue) {
        actions.push("ISSUE_STATUS_REOPEN");
      }
      return actions;
    }

    // If issue is done, check for reopen action
    if (isDone) {
      if (canUpdateIssue) {
        actions.push("ISSUE_STATUS_REOPEN");
      }
      return actions;
    }

    // Check for review actions
    const issueApproved =
      issueValue.approvalStatus === Issue_ApprovalStatus.APPROVED ||
      issueValue.approvalStatus === Issue_ApprovalStatus.SKIPPED;
    if (!issueApproved) {
      const { approvers, approvalTemplate } = issueValue;

      // Check if issue has been rejected
      const hasRejection = approvers.some(
        (app) => app.status === Issue_Approver_Status.REJECTED
      );

      // RE_REQUEST is only available to the issue creator when rejected
      if (hasRejection && currentUserEmail === issueCreator) {
        actions.push("ISSUE_REVIEW_RE_REQUEST");
      } else {
        // Check if user can approve/reject
        const roles = approvalTemplate?.flow?.roles ?? [];
        if (roles.length > 0) {
          const rejectedIndex = approvers.findIndex(
            (ap) => ap.status === Issue_Approver_Status.REJECTED
          );
          const currentRoleIndex =
            rejectedIndex >= 0 ? rejectedIndex : approvers.length;
          const currentRole = roles[currentRoleIndex];

          if (currentRole) {
            const candidates = candidatesOfApprovalStepV1(
              issueValue,
              currentRole
            );
            if (isUserIncludedInList(currentUserEmail, candidates)) {
              actions.push("ISSUE_REVIEW_APPROVE");

              // Only show REJECT if no one has rejected yet.
              if (!hasRejection) {
                actions.push("ISSUE_REVIEW_REJECT");
              }
            }
          }
        }
      }
    }

    if (canUpdateIssue) {
      // Check if issue can be resolved (all tasks must be finished)
      if (rollout.value) {
        const allTasksFinished = rollout.value.stages
          .flatMap((stage) => stage.tasks)
          .every((task) =>
            [Task_Status.DONE, Task_Status.SKIPPED].includes(task.status)
          );
        if (allTasksFinished && issueApproved) {
          actions.push("ISSUE_STATUS_RESOLVE");
        }
      }

      actions.push("ISSUE_STATUS_CLOSE");
    }

    // Check for rollout actions when rollout has database creation/export tasks
    // Only allow rollout actions when approval is ready (APPROVED or SKIPPED)
    // For database change tasks, we'll handle in other places
    if (rollout.value && issueApproved) {
      // Check if there are any database creation or export tasks
      const hasDatabaseCreateOrExportTasks = rollout.value.stages.some(
        (stage) =>
          stage.tasks.some(
            (task) =>
              task.type === Task_Type.DATABASE_CREATE ||
              task.type === Task_Type.DATABASE_EXPORT
          )
      );
      // Different permission checks based on issue type
      // For export data issues: only the creator can run tasks
      // For other issues: need bb.taskRuns.create permission
      const canRunTasks =
        issueValue.type === Issue_Type.DATABASE_EXPORT
          ? currentUserEmail === extractUserId(issueValue.creator)
          : hasProjectPermissionV2(project.value, "bb.taskRuns.create");

      if (hasDatabaseCreateOrExportTasks && canRunTasks) {
        // Show ROLLOUT_START if there are actionable database creation/export tasks
        // This includes both normal rollout and force rollout scenarios
        const hasStartableTasks = rollout.value.stages
          .flatMap((stage) => stage.tasks)
          .some((task) =>
            [
              Task_Status.NOT_STARTED,
              Task_Status.FAILED,
              Task_Status.CANCELED,
            ].includes(task.status)
          );
        if (hasStartableTasks) {
          actions.push("ROLLOUT_START");
        }

        // Check for cancel action on running/pending tasks
        const runningTask = rollout.value.stages
          .flatMap((stage) => stage.tasks)
          .find((task) =>
            [Task_Status.PENDING, Task_Status.RUNNING].includes(task.status)
          );

        if (runningTask) {
          actions.push("ROLLOUT_CANCEL");
        }
      }
    }

    return actions;
  });

  const isExportPlan = computed(() => {
    return plan.value.specs.some(
      (spec) => spec.config?.case === "exportDataConfig"
    );
  });

  const actionDisplayName = (action: UnifiedAction): string => {
    switch (action) {
      case "ISSUE_REVIEW_APPROVE":
        return t("common.approve");
      case "ISSUE_REVIEW_REJECT":
        return t("custom-approval.issue-review.send-back");
      case "ISSUE_REVIEW_RE_REQUEST":
        return t("custom-approval.issue-review.re-request-review");
      case "ISSUE_STATUS_CLOSE":
        return t("issue.batch-transition.close");
      case "ISSUE_STATUS_REOPEN":
        return t("issue.batch-transition.reopen");
      case "ISSUE_STATUS_RESOLVE":
        return t("issue.batch-transition.resolve");
      case "ISSUE_CREATE":
        return t("plan.ready-for-review");
      case "PLAN_CLOSE":
        return t("common.close");
      case "PLAN_REOPEN":
        return t("common.reopen");
      case "ROLLOUT_START":
        return isExportPlan.value ? t("common.export") : t("common.rollout");
      case "ROLLOUT_CANCEL":
        return t("common.cancel");
    }
  };

  return {
    availableActions,
    actionDisplayName,
  };
};
