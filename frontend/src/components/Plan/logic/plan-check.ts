import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import type { ComposedPlan } from "@/types/v1/issue/plan";
import { sheetNameForSpec, databaseForSpec } from ".";

export const planSpecHasPlanChecks = (spec: Plan_Spec) => {
  if (spec.changeDatabaseConfig !== undefined) {
    return true;
  }
  return false;
};

export const planCheckRunListForSpec = (
  plan: ComposedPlan,
  spec: Plan_Spec
) => {
  const target = databaseForSpec(plan.projectEntity, spec).name;
  const sheet = spec ? sheetNameForSpec(spec) : "";
  return plan.planCheckRunList.filter((check) => {
    if (sheet && check.sheet) {
      return check.sheet === sheet && check.target === target;
    }
    // Otherwise filter by target only
    return check.target === target;
  });
};
