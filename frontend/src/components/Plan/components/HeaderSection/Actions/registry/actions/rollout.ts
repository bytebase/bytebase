import { t } from "@/plugins/i18n";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { ActionDefinition } from "../types";

export const ROLLOUT_CREATE: ActionDefinition = {
  id: "ROLLOUT_CREATE",
  label: () => t("issue.create-rollout"),
  buttonType: "default",
  category: (ctx) => {
    // Show as primary only when issue approved and plan checks done without errors
    if (
      ctx.issueApproved &&
      !ctx.validation.planChecksRunning &&
      !ctx.validation.planChecksFailed
    ) {
      return "primary";
    }
    // For force creating rollout, show as secondary
    return "secondary";
  },
  priority: 55,

  isVisible: (ctx) => {
    // Deferred rollout plans use ROLLOUT_START to create rollout and run tasks together
    if (ctx.hasDeferredRollout) return false;
    if (ctx.isIssueOnly) return false;
    if (ctx.plan.hasRollout) return false;
    if (!ctx.issue) return false;
    // Don't show create rollout when issue is closed
    if (ctx.issueStatus === IssueStatus.CANCELED) return false;
    if (!ctx.permissions.createRollout) return false;

    // Project setting validations are handled in RolloutCreatePanel
    return true;
  },

  isDisabled: () => false,
  disabledReason: () => undefined,

  executeType: "immediate",
};

export const ROLLOUT_START: ActionDefinition = {
  id: "ROLLOUT_START",
  label: (ctx) => (ctx.isExportPlan ? t("common.export") : t("common.rollout")),
  buttonType: "primary",
  category: "primary",
  priority: 60,

  isVisible: (ctx) => {
    // Deferred rollout plans: rollout is created on-demand when user clicks action button
    if (ctx.hasDeferredRollout) {
      if (!ctx.issue || !ctx.issueApproved) return false;
      // Show if no rollout yet (will create it) or has startable tasks
      if (!ctx.rollout) return ctx.permissions.runTasks;
      return ctx.hasStartableTasks && ctx.permissions.runTasks;
    }
    // Regular plans: rollout must already exist
    if (!ctx.rollout) return false;
    if (!ctx.issueApproved) return false;
    if (!ctx.hasDatabaseCreateOrExportTasks) return false;
    if (!ctx.hasStartableTasks) return false;
    return ctx.permissions.runTasks;
  },

  isDisabled: () => false,
  disabledReason: () => undefined,

  executeType: "panel:rollout",
};

export const ROLLOUT_CANCEL: ActionDefinition = {
  id: "ROLLOUT_CANCEL",
  label: () => t("common.cancel"),
  buttonType: "default",
  category: "secondary",
  priority: 80,

  isVisible: (ctx) => {
    if (!ctx.rollout) return false;
    if (!ctx.issueApproved) return false;
    if (!ctx.hasRunningTasks) return false;
    return ctx.permissions.runTasks;
  },

  isDisabled: () => false,
  disabledReason: () => undefined,

  executeType: "panel:rollout",
};

export const EXPORT_DOWNLOAD: ActionDefinition = {
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
};

export const rolloutActions: ActionDefinition[] = [
  ROLLOUT_CREATE,
  ROLLOUT_START,
  ROLLOUT_CANCEL,
  EXPORT_DOWNLOAD,
];
