import Emittery from "emittery";
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_REVIEW_CENTER_DETAIL } from "@/router/dashboard/projectV1";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import type { PlanContext, PlanEvents } from "./context";

export const useBasePlanContext = (
  context: Pick<PlanContext, "isCreating" | "ready" | "plan">
): Partial<PlanContext> => {
  const { plan } = context;
  const route = useRoute();
  const router = useRouter();

  const events: PlanEvents = new Emittery();

  const specs = computed(() => plan.value?.specs || []);

  const selectedSpec = computed((): Plan_Spec | undefined => {
    // Check if spec is selected from URL.
    const specId = route.query.spec as string;
    if (specId) {
      const specFound = specs.value.find((spec) => spec.id === specId);
      if (specFound) {
        return specFound;
      }
    }

    // Fallback to first spec.
    return undefined;
  });

  events.on("select-spec", ({ spec }) => {
    router.replace({
      name: PROJECT_V1_ROUTE_REVIEW_CENTER_DETAIL,
      query: {
        ...route.query,
        spec: spec.id,
      },
      hash: route.hash,
    });
  });

  return {
    events,
    selectedSpec,
  };
};
