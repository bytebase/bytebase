export * from "./types";
export * from "./context";
export * from "./useActionRegistry";
export * from "./actions";
export { ActionButton, ActionDropdown } from "./components";

// Standalone helper to get action label without full registry
import { t } from "@/plugins/i18n";
import type { UnifiedAction } from "./types";

export function getActionLabel(
  actionId: UnifiedAction,
  isExportPlan = false
): string {
  switch (actionId) {
    case "ISSUE_REVIEW":
      return t("issue.review.self");
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
    case "ROLLOUT_CREATE":
      return t("issue.create-rollout");
    case "ROLLOUT_START":
      return isExportPlan ? t("common.export") : t("common.rollout");
    case "ROLLOUT_CANCEL":
      return t("common.cancel");
    case "EXPORT_DOWNLOAD":
      return t("common.download");
  }
}
