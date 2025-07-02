import type { Plan_Spec, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import { sheetNameForSpec, targetsForSpec } from "./plan";

export const planSpecHasPlanChecks = (spec: Plan_Spec) => {
  if (spec.config?.case === "changeDatabaseConfig") {
    return true;
  }
  return false;
};

export const planCheckRunListForSpec = (
  planCheckRunList: PlanCheckRun[],
  spec: Plan_Spec
) => {
  const targets = targetsForSpec(spec);
  const sheet = sheetNameForSpec(spec);
  return planCheckRunList.filter((check) => {
    if (!targets.includes(check.target)) {
      return false;
    }
    if (sheet && check.sheet) {
      return check.sheet === sheet;
    }
  });
};
