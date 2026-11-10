import type { TFunction } from "i18next";
import type { ScopeOption } from "@/components/AdvancedSearch";
import type { PlanStatusScopeValue } from "@/stores/app/plan";

export const planStateScopeOption = (t: TFunction): ScopeOption => ({
  id: "state",
  title: t("common.state"),
  description: t("issue.advanced-search.scope.state.description"),
  options: [
    {
      value: "ACTIVE",
      keywords: ["active", "ACTIVE"],
      render: () => <span>{t("common.active")}</span>,
    },
    {
      value: "DELETED",
      keywords: ["deleted", "DELETED", "closed", "CLOSED"],
      render: () => <span>{t("common.closed")}</span>,
    },
  ],
});

export const planStatusScopeOption = (t: TFunction): ScopeOption => {
  const labels: Record<PlanStatusScopeValue, string> = {
    FAILED: t("task.status.failed"),
    RUNNING: t("task.status.running"),
    WAITING: t("plan.status-scope.waiting"),
    DONE: t("task.status.done"),
  };
  return {
    id: "status",
    title: t("common.status"),
    description: t("plan.status-scope.description"),
    allowMultiple: true,
    options: (Object.entries(labels) as [PlanStatusScopeValue, string][]).map(
      ([value, label]) => ({
        value,
        keywords: [value.toLowerCase(), label.toLowerCase()],
        render: () => <span>{label}</span>,
      })
    ),
  };
};
