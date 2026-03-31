import { t } from "@/plugins/i18n";
import type { ActionDefinition } from "../types";

export const ROLLOUT_START: ActionDefinition = {
  id: "ROLLOUT_START",
  label: (ctx) => (ctx.isExportPlan ? t("common.export") : t("common.rollout")),
  buttonType: "primary",
  category: "primary",
  priority: 60,

  isVisible: (ctx) => {
    // Only for deferred rollout plans (export/create DB) — rollout created on-demand
    // Change database plans: rollout managed in Plan Detail Page's Deploy section
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
      return t("common.missing-required-permission");
    }
    return undefined;
  },

  executeType: "panel:rollout",
};

export const ROLLOUT_CANCEL: ActionDefinition = {
  id: "ROLLOUT_CANCEL",
  label: () => t("common.cancel"),
  buttonType: "default",
  category: "secondary",
  priority: 80,

  isVisible: (ctx) => {
    // Only for deferred rollout plans — change database rollouts managed on Plan Detail Page
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
  ROLLOUT_START,
  ROLLOUT_CANCEL,
  EXPORT_DOWNLOAD,
];
