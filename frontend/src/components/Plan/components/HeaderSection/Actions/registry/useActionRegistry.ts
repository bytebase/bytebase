import { computed } from "vue";
import { useSpecsValidation } from "@/components/Plan/components/common";
import { usePlanContext } from "@/components/Plan/logic";
import { useEditorState } from "@/components/Plan/logic/useEditorState";
import { usePlanCheckStatus } from "@/components/Plan/logic/usePlanCheckStatus";
import { t } from "@/plugins/i18n";
import { useCurrentProjectV1, useCurrentUserV1 } from "@/store";
import { allActions } from "./actions";
import { buildActionContext } from "./context";
import type { ActionDefinition, ActionRegistryReturn } from "./types";

export function useActionRegistry(): ActionRegistryReturn {
  const currentUser = useCurrentUserV1();
  const { project } = useCurrentProjectV1();
  const { isCreating, plan, issue, rollout, taskRuns } = usePlanContext();
  const editorState = useEditorState();
  const { isSpecEmpty } = useSpecsValidation(computed(() => plan.value.specs));
  const {
    getOverallStatus: planCheckStatus,
    hasRunning: hasRunningPlanChecks,
  } = usePlanCheckStatus(plan);

  // Build context reactively
  const context = computed(() =>
    buildActionContext({
      plan: plan.value,
      issue: issue.value,
      rollout: rollout.value,
      project: project.value,
      currentUser: currentUser.value,
      taskRuns: taskRuns.value,
      isCreating: isCreating.value,
      planCheckStatus: planCheckStatus.value,
      hasRunningPlanChecks: hasRunningPlanChecks.value,
      isSpecEmpty,
    })
  );

  // Global disabled state (editor is editing)
  const globalDisabled = computed(() => editorState.isEditing.value);
  const globalDisabledReason = computed(() =>
    globalDisabled.value
      ? t("plan.editor.save-changes-before-continuing")
      : undefined
  );

  // Helper to resolve category (static or dynamic)
  const getCategory = (action: ActionDefinition) =>
    typeof action.category === "function"
      ? action.category(context.value)
      : action.category;

  // Filter visible actions (no actions during plan creation)
  const visibleActions = computed(() => {
    if (isCreating.value) return [];
    return allActions.filter((action) => action.isVisible(context.value));
  });

  // Primary action (first visible by priority, category=primary)
  const primaryAction = computed(() =>
    visibleActions.value.find((a) => getCategory(a) === "primary")
  );

  // Secondary actions: all secondary-category actions + remaining primary actions (excluding the chosen primary)
  const secondaryActions = computed(() =>
    visibleActions.value.filter(
      (a) =>
        getCategory(a) === "secondary" ||
        (getCategory(a) === "primary" && a.id !== primaryAction.value?.id)
    )
  );

  // Check if action is disabled (global or action-specific)
  const isActionDisabled = (action: ActionDefinition) =>
    globalDisabled.value || action.isDisabled(context.value);

  // Get disabled reason (global takes precedence)
  const getDisabledReason = (action: ActionDefinition) =>
    globalDisabledReason.value || action.disabledReason(context.value);

  return {
    context,
    primaryAction,
    secondaryActions,
    isActionDisabled,
    getDisabledReason,
  };
}
