import { t } from "@/plugins/i18n";
import { Issue_ApprovalStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { ActionDefinition } from "../types";

export const ROLLOUT_CREATE: ActionDefinition = {
  id: "ROLLOUT_CREATE",
  label: () => t("issue.create-rollout"),
  buttonType: "default",
  category: "primary",
  priority: 55,

  isVisible: (ctx) =>
    !ctx.isIssueOnly &&
    !ctx.plan.hasRollout &&
    ctx.issue !== undefined &&
    ctx.permissions.createRollout &&
    ctx.rolloutPreconditionsMet,

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
    if (!ctx.rollout) return false;
    if (
      ctx.approvalStatus !== Issue_ApprovalStatus.APPROVED &&
      ctx.approvalStatus !== Issue_ApprovalStatus.SKIPPED
    ) {
      return false;
    }
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
    if (
      ctx.approvalStatus !== Issue_ApprovalStatus.APPROVED &&
      ctx.approvalStatus !== Issue_ApprovalStatus.SKIPPED
    ) {
      return false;
    }
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
