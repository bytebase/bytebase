import { flatten } from "lodash-es";
import { useDBGroupStore } from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type {
  Plan_Spec,
  PlanCheckRun,
} from "@/types/proto-es/v1/plan_service_pb";

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
          DatabaseGroupView.FULL
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
): PlanCheckRun[] => {
  const targets = flattenTargetsOfSpec(spec);

  // With consolidated model, filter runs that have results matching our targets
  return planCheckRunList.filter((run) => {
    return run.results.some((result) => targets.includes(result.target));
  });
};
