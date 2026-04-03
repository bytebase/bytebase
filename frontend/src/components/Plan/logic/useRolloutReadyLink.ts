import { computed } from "vue";
import { useRoute } from "vue-router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { usePlanContext } from "./context";
import { isDatabaseChangeSpec } from "./plan";

/**
 * Composable that determines whether the "Plan" link should be shown.
 * This link appears on the Issue page when:
 * - The plan has an existing rollout
 * - All specs are database-change related
 */
export const useRolloutReadyLink = () => {
  const route = useRoute();
  const { plan } = usePlanContext();

  const shouldShow = computed(() => {
    // Only show on Issue page
    if (route.name !== PROJECT_V1_ROUTE_ISSUE_DETAIL) return false;

    if (!plan.value.hasRollout) return false;

    return (
      plan.value.specs.length > 0 &&
      plan.value.specs.every(isDatabaseChangeSpec)
    );
  });

  return {
    shouldShow,
  };
};
