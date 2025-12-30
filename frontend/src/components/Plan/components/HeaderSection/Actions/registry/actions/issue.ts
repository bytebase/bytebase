import { t } from "@/plugins/i18n";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  Issue_ApprovalStatus,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { ActionDefinition } from "../types";

export const ISSUE_CREATE: ActionDefinition = {
  id: "ISSUE_CREATE",
  label: () => t("plan.ready-for-review"),
  buttonType: "primary",
  category: "primary",
  priority: 5,

  isVisible: (ctx) =>
    !ctx.isIssueOnly &&
    ctx.plan.issue === "" &&
    !ctx.plan.hasRollout &&
    ctx.planState === State.ACTIVE &&
    ctx.permissions.createIssue,

  isDisabled: (ctx) =>
    ctx.validation.hasEmptySpec ||
    ctx.validation.planChecksRunning ||
    (ctx.validation.planChecksFailed && ctx.project.enforceSqlReview),

  disabledReason: (ctx) => {
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
};

export const ISSUE_REVIEW: ActionDefinition = {
  id: "ISSUE_REVIEW",
  label: () => t("issue.review.self"),
  buttonType: "primary",
  category: "primary",
  priority: 30,

  isVisible: (ctx) =>
    ctx.issueStatus === IssueStatus.OPEN &&
    ctx.approvalStatus !== Issue_ApprovalStatus.APPROVED &&
    ctx.approvalStatus !== Issue_ApprovalStatus.SKIPPED &&
    ctx.permissions.isApprovalCandidate,

  isDisabled: () => false,
  disabledReason: () => undefined,

  executeType: "popover:review",
};

export const ISSUE_STATUS_RESOLVE: ActionDefinition = {
  id: "ISSUE_STATUS_RESOLVE",
  label: () => t("issue.batch-transition.resolve"),
  buttonType: "success",
  category: "primary",
  priority: 50,

  isVisible: (ctx) => {
    // Deferred rollout plans auto-resolve when task completes, never show manual resolve
    if (ctx.hasDeferredRollout) return false;
    return (
      ctx.issueStatus === IssueStatus.OPEN &&
      (ctx.approvalStatus === Issue_ApprovalStatus.APPROVED ||
        ctx.approvalStatus === Issue_ApprovalStatus.SKIPPED) &&
      ctx.allTasksFinished &&
      ctx.plan.hasRollout &&
      ctx.permissions.updateIssue
    );
  },

  isDisabled: () => false,
  disabledReason: () => undefined,

  executeType: "panel:issue-status",
};

export const ISSUE_STATUS_CLOSE: ActionDefinition = {
  id: "ISSUE_STATUS_CLOSE",
  label: () => t("issue.batch-transition.close"),
  buttonType: "default",
  category: "secondary",
  priority: 90,

  isVisible: (ctx) =>
    ctx.issueStatus === IssueStatus.OPEN &&
    !ctx.plan.hasRollout &&
    ctx.permissions.updateIssue,

  isDisabled: () => false,
  disabledReason: () => undefined,

  executeType: "panel:issue-status",
};

export const ISSUE_STATUS_REOPEN: ActionDefinition = {
  id: "ISSUE_STATUS_REOPEN",
  label: () => t("issue.batch-transition.reopen"),
  buttonType: "default",
  category: "primary",
  priority: 20,

  // Only show reopen for canceled issues, not for done/resolved issues
  isVisible: (ctx) =>
    ctx.issueStatus === IssueStatus.CANCELED && ctx.permissions.updateIssue,

  isDisabled: () => false,
  disabledReason: () => undefined,

  executeType: "panel:issue-status",
};

export const issueActions: ActionDefinition[] = [
  ISSUE_CREATE,
  ISSUE_REVIEW,
  ISSUE_STATUS_RESOLVE,
  ISSUE_STATUS_CLOSE,
  ISSUE_STATUS_REOPEN,
];
