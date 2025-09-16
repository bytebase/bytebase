import { flatten } from "lodash-es";
import { useDBGroupStore } from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type {
  Plan_Spec,
  PlanCheckRun,
} from "@/types/proto-es/v1/plan_service_pb";
import { sheetNameForSpec } from "./plan";

export const planSpecHasPlanChecks = (spec: Plan_Spec) => {
  if (spec.config?.case === "changeDatabaseConfig") {
    return true;
  }
  return false;
};

// flattenTargetsOfSpec flattens the targets of a plan spec, which can be a mix of database names and database group names.
export const flattenTargetsOfSpec = (spec: Plan_Spec): string[] => {
  let rawTargets: string[] = [];
  if (spec.config.case === "changeDatabaseConfig") {
    rawTargets = spec.config.value.targets || [];
  }
  if (spec.config.case === "exportDataConfig") {
    rawTargets = spec.config.value.targets || [];
  }
  const targets = flatten(
    rawTargets.map((target) => {
      if (isValidDatabaseName(target)) {
        return target;
      }
      if (isValidDatabaseGroupName(target)) {
        const dbGroup = useDBGroupStore().getDBGroupByName(
          target,
          DatabaseGroupView.MATCHED
        );
        return dbGroup?.matchedDatabases.map((db) => db.name);
      }
      return target;
    })
  ).filter(Boolean) as string[];
  return targets;
};

export const planCheckRunListForSpec = (
  planCheckRunList: PlanCheckRun[],
  spec: Plan_Spec
) => {
  const targets = flattenTargetsOfSpec(spec);
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
