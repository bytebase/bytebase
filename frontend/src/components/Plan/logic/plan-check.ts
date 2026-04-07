import { flatten } from "lodash-es";
import { useDBGroupStore } from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import {
  type Plan_Spec,
  type PlanCheckRun,
  PlanCheckRun_Status,
} from "@/types/proto-es/v1/plan_service_pb";

/** Spec targets and slicing consolidated plan check runs per change. Counts: `plan-check-status.ts`. */

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

/**
 * Returns plan check runs scoped to `spec`: each run keeps only `results` whose
 * `target` is in this spec’s flattened database list. Handles consolidated
 * in-memory runs (one API row per plan with all databases).
 */
export const planCheckRunListForSpec = (
  planCheckRunList: PlanCheckRun[],
  spec: Plan_Spec
): PlanCheckRun[] => {
  const targets = flattenTargetsOfSpec(spec);
  if (targets.length === 0) {
    return [];
  }

  // Set lookup: spec can list many DBs; consolidated runs can have huge `results`.
  const targetSet = new Set(targets);

  return planCheckRunList.flatMap((run) => {
    const matchingResults = run.results.filter((result) =>
      targetSet.has(result.target)
    );

    if (matchingResults.length > 0) {
      return [{ ...run, results: matchingResults }];
    }

    // In progress: no per-target rows yet.
    if (
      run.status === PlanCheckRun_Status.RUNNING &&
      run.results.length === 0
    ) {
      return [{ ...run, results: [] }];
    }

    // Run-level failure with no per-target breakdown.
    if (run.status === PlanCheckRun_Status.FAILED && run.results.length === 0) {
      return [{ ...run, results: [] }];
    }

    return [];
  });
};
