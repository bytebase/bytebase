// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  type PlanCheckRun_Result,
  PlanCheckRun_Result_SqlReviewReportSchema,
  PlanCheckRun_Result_SqlSummaryReportSchema,
  PlanCheckRun_Result_Type,
  PlanCheckRun_ResultSchema,
  PlanCheckRunSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { affectedRowsSummary } from "./affectedRows";

const DB1 = "instances/i/databases/db1";
const DB2 = "instances/i/databases/db2";

const summaryResult = (
  target: string,
  status: Advice_Level,
  rows: bigint,
  content = ""
) =>
  create(PlanCheckRun_ResultSchema, {
    type: PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT,
    status,
    target,
    content,
    report: {
      case: "sqlSummaryReport",
      value: create(PlanCheckRun_Result_SqlSummaryReportSchema, {
        affectedRows: rows,
      }),
    },
  });

const run = (uid: number, results: PlanCheckRun_Result[]) =>
  create(PlanCheckRunSchema, {
    name: `projects/p/plans/1/planCheckRuns/${uid}`,
    results,
  });

describe("affectedRowsSummary", () => {
  test("a warning marks an estimate incomplete and keeps its reason", () => {
    const summary = affectedRowsSummary([
      run(1, [
        summaryResult(DB1, Advice_Level.SUCCESS, 600n),
        summaryResult(DB2, Advice_Level.WARNING, 200170n, "permission denied"),
      ]),
    ]);
    expect(
      summary?.targets.map(({ target, status, rows, reason }) => ({
        target,
        status,
        rows,
        reason,
      }))
    ).toEqual([
      { target: DB1, status: "estimated", rows: 600n, reason: "" },
      {
        target: DB2,
        status: "incomplete",
        rows: 200170n,
        reason: "permission denied",
      },
    ]);
  });

  test("a syntax error is not estimated and points at its line", () => {
    const summary = affectedRowsSummary([
      run(1, [
        create(PlanCheckRun_ResultSchema, {
          type: PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT,
          status: Advice_Level.ERROR,
          target: DB1,
          title: "Syntax error",
          report: {
            case: "sqlReviewReport",
            value: create(PlanCheckRun_Result_SqlReviewReportSchema, {
              startPosition: { line: 12, column: 3 },
            }),
          },
        }),
      ]),
    ]);
    expect(summary?.targets[0]).toMatchObject({
      status: "not-estimated",
      title: "Syntax error",
      line: 12,
    });
  });

  test("reads only the latest run and skips engines without a report", () => {
    const summary = affectedRowsSummary([
      run(1, [summaryResult(DB1, Advice_Level.SUCCESS, 1n)]),
      run(2, [
        summaryResult(DB1, Advice_Level.SUCCESS, 2n),
        create(PlanCheckRun_ResultSchema, {
          type: PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT,
          status: Advice_Level.SUCCESS,
          target: DB2,
          title: "Statement report is not supported for MONGODB",
        }),
        create(PlanCheckRun_ResultSchema, {
          type: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
          status: Advice_Level.ERROR,
          target: DB2,
        }),
      ]),
    ]);
    expect(summary?.targets.map(({ target, rows }) => [target, rows])).toEqual([
      [DB1, 2n],
    ]);
  });

  test("is undefined without a statement report", () => {
    expect(affectedRowsSummary([])).toBeUndefined();
    expect(
      affectedRowsSummary([
        run(1, [
          create(PlanCheckRun_ResultSchema, {
            type: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
            status: Advice_Level.SUCCESS,
            target: DB1,
          }),
        ]),
      ])
    ).toBeUndefined();
  });
});
