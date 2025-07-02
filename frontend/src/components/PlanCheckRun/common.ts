import { maxBy } from "lodash-es";
import {
  PlanCheckRun_Result_Status,
  PlanCheckRun_Status,
  PlanCheckRun_Type,
  type PlanCheckRun,
} from "@/types/proto-es/v1/plan_service_pb";
import { extractPlanCheckRunUID } from "@/utils";

export const planCheckRunResultStatus = (checkRun: PlanCheckRun) => {
  let status = PlanCheckRun_Result_Status.SUCCESS;

  for (const result of checkRun.results) {
    if (result.status === PlanCheckRun_Result_Status.ERROR) {
      return PlanCheckRun_Result_Status.ERROR;
    }
    if (result.status === PlanCheckRun_Result_Status.WARNING) {
      status = PlanCheckRun_Result_Status.WARNING;
    }
  }
  return status;
};

export const HiddenPlanCheckTypes = new Set<PlanCheckRun_Type>([
  PlanCheckRun_Type.DATABASE_STATEMENT_SUMMARY_REPORT,
]);

export type PlanCheckRunSummary = {
  runningCount: number;
  successCount: number;
  warnCount: number;
  errorCount: number;
};

export const planCheckRunSummaryForCheckRunList = (
  planCheckRuns: PlanCheckRun[]
) => {
  const summary: PlanCheckRunSummary = {
    runningCount: 0,
    successCount: 0,
    warnCount: 0,
    errorCount: 0,
  };

  planCheckRuns = planCheckRuns.filter(
    (check) => !HiddenPlanCheckTypes.has(check.type)
  );

  const listGroupByTypeTargetSheet = planCheckRuns.reduce(
    (acc, checkRun) => {
      const key = `${checkRun.type}-${checkRun.target}-${checkRun.sheet}`;
      if (!acc[key]) {
        acc[key] = [];
      }
      acc[key].push(checkRun);
      return acc;
    },
    {} as { [key: string]: PlanCheckRun[] }
  );

  const latestCheckRunOfEachTypeTargetSheet = Object.keys(
    listGroupByTypeTargetSheet
  ).map((k) => {
    const listOfTypeTargetSheet = listGroupByTypeTargetSheet[k];
    const latest = maxBy(listOfTypeTargetSheet, (checkRun) =>
      Number(extractPlanCheckRunUID(checkRun.name))
    )!;
    return latest;
  });

  for (const checkRun of latestCheckRunOfEachTypeTargetSheet) {
    switch (checkRun.status) {
      case PlanCheckRun_Status.CANCELED:
        // nothing todo
        break;
      case PlanCheckRun_Status.FAILED:
        summary.errorCount++;
        break;
      case PlanCheckRun_Status.RUNNING:
        summary.runningCount++;
        break;
      case PlanCheckRun_Status.DONE:
        switch (planCheckRunResultStatus(checkRun)) {
          case PlanCheckRun_Result_Status.SUCCESS:
            summary.successCount++;
            break;
          case PlanCheckRun_Result_Status.WARNING:
            summary.warnCount++;
            break;
          case PlanCheckRun_Result_Status.ERROR:
            summary.errorCount++;
        }
    }
  }

  return summary;
};
