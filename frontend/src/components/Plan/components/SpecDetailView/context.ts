import { head } from "lodash-es";
import {
  computed,
  inject,
  provide,
  watchEffect,
  type InjectionKey,
  type Ref,
} from "vue";
import { useRoute, useRouter } from "vue-router";
import { type Plan } from "@/types/proto-es/v1/plan_service_pb";
import { gotoSpec } from "../common/utils";

const KEY = Symbol("bb.plan.spec.detail") as InjectionKey<PlanSpecContext>;

export const usePlanSpecContext = () => {
  return inject(KEY)!;
};

export const providePlanSpecContext = (refs: {
  isCreating: Ref<boolean>;
  plan: Ref<Plan>;
}) => {
  const route = useRoute();
  const router = useRouter();
  const { plan } = refs;

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

  // Auto-redirect if the specId in the route does not match any spec in the plan.
  watchEffect(() => {
    const specNotFound = !plan.value.specs.some(
      (spec) => spec.id === route.params.specId
    );
    if (specNotFound) {
      const foundSpec = head(plan.value.specs);
      if (!foundSpec) {
        throw new Error("No specs available in the plan to redirect to.");
      }
      gotoSpec(router, plan.value, foundSpec.id);
    }
  });

  const context = {
    selectedSpec,
  };

  provide(KEY, context);

  return context;
};

type PlanSpecContext = ReturnType<typeof providePlanSpecContext>;
