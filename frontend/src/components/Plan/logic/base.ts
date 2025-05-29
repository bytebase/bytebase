import Emittery from "emittery";
import { first } from "lodash-es";
import { useDialog } from "naive-ui";
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_REVIEW_CENTER_DETAIL } from "@/router/dashboard/projectV1";
import { useUIStateStore } from "@/store";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import { emptyPlanSpec } from "@/types/v1/issue/plan";
import type { PlanContext, PlanEvents } from "./context";

export const useBasePlanContext = (
  context: Pick<PlanContext, "isCreating" | "ready" | "plan">
): Partial<PlanContext> => {
  const { plan } = context;
  const uiStateStore = useUIStateStore();
  const route = useRoute();
  const router = useRouter();
  const dialog = useDialog();

  const events: PlanEvents = new Emittery();

  const specs = computed(() => plan.value?.specs || []);

  const selectedSpec = computed((): Plan_Spec => {
    // Check if spec is selected from URL.
    const specId = route.query.spec as string;
    if (specId) {
      const specFound = specs.value.find((spec) => spec.id === specId);
      if (specFound) {
        return specFound;
      }
    }

    // Fallback to first spec.
    return first(specs.value) || emptyPlanSpec();
  });

  const formatOnSave = computed({
    get: () => uiStateStore.editorFormatStatementOnSave,
    set: (value: boolean) => uiStateStore.setEditorFormatStatementOnSave(value),
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
    formatOnSave,
    dialog,
  };
};
