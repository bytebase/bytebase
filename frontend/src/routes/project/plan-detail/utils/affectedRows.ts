import type { Timestamp } from "@bufbuild/protobuf/wkt";
import {
  type PlanCheckRun,
  PlanCheckRun_Result_Type,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { extractDatabaseResourceName } from "@/utils";
import { latestPlanCheckRun } from "./sqlAdvice";

// "incomplete": some DML statements could not be estimated; the reason gives
// how many and why. "not-estimated": no number at all, as for a syntax error.
export type AffectedRowsStatus = "estimated" | "incomplete" | "not-estimated";

export interface TargetAffectedRows {
  target: string;
  status: AffectedRowsStatus;
  rows: bigint;
  // The server's own words for why the estimate falls short.
  title: string;
  reason: string;
  // Where a syntax error starts, 1-based.
  line?: number;
}

export interface AffectedRowsSummary {
  targets: TargetAffectedRows[];
  time?: Timestamp;
}

export const needsLook = (target: TargetAffectedRows): boolean =>
  target.status !== "estimated";

export const formatRows = (rows: bigint): string => rows.toLocaleString();

export const databaseLabel = (target: string): string =>
  extractDatabaseResourceName(target).databaseName || target;

// The statement report of the latest plan check run, one entry per database.
// A database whose engine has no report is left out; undefined when none has.
export function affectedRowsSummary(
  runs: readonly PlanCheckRun[]
): AffectedRowsSummary | undefined {
  const run = latestPlanCheckRun(runs);
  if (!run) return undefined;
  const byTarget = new Map<string, TargetAffectedRows>();
  for (const result of run.results) {
    if (
      result.type !== PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT ||
      byTarget.has(result.target)
    ) {
      continue;
    }
    const base = {
      target: result.target,
      title: result.title,
      reason: result.content,
    };
    if (result.report.case === "sqlSummaryReport") {
      byTarget.set(result.target, {
        ...base,
        status:
          result.status === Advice_Level.WARNING ? "incomplete" : "estimated",
        rows: result.report.value.affectedRows,
      });
      continue;
    }
    if (
      result.status !== Advice_Level.ERROR &&
      result.status !== Advice_Level.WARNING
    ) {
      continue;
    }
    const line =
      result.report.case === "sqlReviewReport"
        ? result.report.value.startPosition?.line
        : undefined;
    byTarget.set(result.target, {
      ...base,
      status: "not-estimated",
      rows: 0n,
      line: line && line > 0 ? line : undefined,
    });
  }
  if (byTarget.size === 0) return undefined;
  return { targets: [...byTarget.values()], time: run.createTime };
}
