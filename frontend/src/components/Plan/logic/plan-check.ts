import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import type { ComposedPlan } from "@/types/v1/issue/plan";
import { sheetNameForSpec, targetsForSpec } from "./plan";

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
  const targets = targetsForSpec(spec);
  const sheet = spec ? sheetNameForSpec(spec) : "";
  return plan.planCheckRunList.filter((check) => {
    if (!targets.includes(check.target)) {
      return false;
    }
    if (sheet && check.sheet) {
      return check.sheet === sheet;
    }
  });
};
