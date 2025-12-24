import {
  type PlanCheckRun,
  PlanCheckRun_Result_Type,
  PlanCheckRun_Status,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";

export const planCheckRunResultStatus = (checkRun: PlanCheckRun) => {
  let status = Advice_Level.SUCCESS;

  for (const result of checkRun.results) {
    if (result.status === Advice_Level.ERROR) {
      return Advice_Level.ERROR;
    }
    if (result.status === Advice_Level.WARNING) {
      status = Advice_Level.WARNING;
    }
  }
  return status;
};

export const HiddenPlanCheckTypes = new Set<PlanCheckRun_Result_Type>([
  PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT,
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

  for (const checkRun of planCheckRuns) {
    if (checkRun.status === PlanCheckRun_Status.RUNNING) {
      summary.runningCount++;
      continue;
    }
    if (checkRun.status === PlanCheckRun_Status.FAILED) {
      summary.errorCount++;
      continue;
    }
    if (checkRun.status === PlanCheckRun_Status.CANCELED) {
      continue;
    }

    // For DONE status, analyze results by type-target
    // Group results by type-target to count unique checks
    const resultsByTypeTarget = new Map<string, Advice_Level>();
    for (const result of checkRun.results) {
      if (HiddenPlanCheckTypes.has(result.type)) {
        continue;
      }
      const key = `${result.type}-${result.target}`;
      const existing = resultsByTypeTarget.get(key);
      // Keep worst status (ERROR > WARNING > SUCCESS)
      if (!existing || result.status > existing) {
        resultsByTypeTarget.set(key, result.status);
      }
    }

    for (const status of resultsByTypeTarget.values()) {
      switch (status) {
        case Advice_Level.SUCCESS:
          summary.successCount++;
          break;
        case Advice_Level.WARNING:
          summary.warnCount++;
          break;
        case Advice_Level.ERROR:
          summary.errorCount++;
          break;
      }
    }
  }

  return summary;
};
