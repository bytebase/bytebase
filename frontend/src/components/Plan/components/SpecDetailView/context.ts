import { head } from "lodash-es";
import { computed } from "vue";
import { useRoute } from "vue-router";
import { usePlanContext } from "../../logic";

export const useSelectedSpec = () => {
  const route = useRoute();
  const { plan } = usePlanContext();

  const selectedSpec = computed(() => {
    if (plan.value.specs.length === 0) {
      throw new Error("No specs found in the plan.");
    }
    const specId = route.params.specId as string | undefined;
    if (!specId) {
      throw new Error("Spec ID is required in the route parameters.");
    }
    const foundSpec =
      plan.value.specs.find((spec) => spec.id === specId) ||
      head(plan.value.specs);
    if (!foundSpec) {
      throw new Error(`Spec with ID ${specId} not found in the plan.`);
    }
    return foundSpec;
  });

  return selectedSpec;
};
