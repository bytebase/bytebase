import { StatementType } from "@/types/proto-es/v1/common_pb";
import {
  type PlanCheckRun,
  PlanCheckRun_Result_Type,
} from "@/types/proto-es/v1/plan_service_pb";

export interface StatementSummary {
  // Distinct statement kinds in order of first appearance: "ALTER TABLE".
  // Some engines' reports list each kind once, so their number is not a
  // statement count.
  types: string[];
  // The most rows any target expects the change to affect.
  affectedRows: bigint;
}

// What the statement summary report says about a spec's SQL. The report
// describes the statement rather than judging it, so it belongs beside the
// statement, not among the reviewers. Undefined until a report exists.
export function statementSummary(
  runs: readonly PlanCheckRun[]
): StatementSummary | undefined {
  const reports = runs.flatMap((run) =>
    run.results.flatMap((result) =>
      result.type === PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT &&
      result.report.case === "sqlSummaryReport"
        ? [result.report.value]
        : []
    )
  );
  if (reports.length === 0) return undefined;
  const first = reports[0];
  const types = [
    ...new Set(
      first.statementTypes.map((type) =>
        (StatementType[type] ?? "").replaceAll("_", " ")
      )
    ),
  ].filter(Boolean);
  const affectedRows = reports.reduce(
    (max, report) => (report.affectedRows > max ? report.affectedRows : max),
    0n
  );
  return { types, affectedRows };
}
