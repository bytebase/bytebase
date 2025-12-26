import { t } from "@/plugins/i18n";
import { State } from "@/types/proto-es/v1/common_pb";
import type { ActionDefinition } from "../types";

export const PLAN_CLOSE: ActionDefinition = {
  id: "PLAN_CLOSE",
  label: () => t("common.close"),
  buttonType: "default",
  category: "secondary",
  priority: 100,

  isVisible: (ctx) =>
    !ctx.isIssueOnly &&
    ctx.plan.issue === "" &&
    !ctx.plan.hasRollout &&
    ctx.planState === State.ACTIVE &&
    ctx.permissions.updatePlan,

  isDisabled: () => false,
  disabledReason: () => undefined,

  executeType: "confirm-dialog",
  confirmTitle: () => t("common.close"),
  confirmContent: () => t("plan.state.close-confirm"),
};

export const PLAN_REOPEN: ActionDefinition = {
  id: "PLAN_REOPEN",
  label: () => t("common.reopen"),
  buttonType: "default",
  category: "primary",
  priority: 10,

  isVisible: (ctx) =>
    !ctx.isIssueOnly &&
    ctx.plan.issue === "" &&
    !ctx.plan.hasRollout &&
    ctx.planState === State.DELETED &&
    ctx.permissions.updatePlan,

  isDisabled: () => false,
  disabledReason: () => undefined,

  executeType: "confirm-dialog",
  confirmTitle: () => t("common.reopen"),
  confirmContent: () => t("plan.state.reopen-confirm"),
};

export const planActions: ActionDefinition[] = [PLAN_CLOSE, PLAN_REOPEN];
