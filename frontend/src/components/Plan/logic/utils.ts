import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import type { ComposedPlan } from "@/types/v1/issue/plan";

export const stepForSpec = (plan: ComposedPlan, spec: Plan_Spec) => {
  return plan.steps.find((step) => step.specs.find((s) => s.id === spec.id));
};
