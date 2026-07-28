import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  PlanCheckRun_Result_Type,
  PlanCheckRun_ResultSchema,
  PlanCheckRunSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { getSQLAdviceMarkers } from "./sqlAdvice";

describe("getSQLAdviceMarkers", () => {
  test("uses latest SQL review warnings and errors with valid positions", () => {
    const markers = getSQLAdviceMarkers([
      create(PlanCheckRunSchema, {
        name: "projects/p/plans/1/planCheckRuns/1",
        results: [
          create(PlanCheckRun_ResultSchema, {
            content: "old",
            report: {
              case: "sqlReviewReport",
              value: { startPosition: { column: 2, line: 3 } },
            },
            status: Advice_Level.ERROR,
            title: "Old",
            type: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
          }),
        ],
      }),
      create(PlanCheckRunSchema, {
        name: "projects/p/plans/1/planCheckRuns/2",
        results: [
          create(PlanCheckRun_ResultSchema, {
            code: 123,
            content: "new warning",
            report: {
              case: "sqlReviewReport",
              value: { startPosition: { column: 4, line: 5 } },
            },
            status: Advice_Level.WARNING,
            title: "Rule",
            type: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
          }),
          create(PlanCheckRun_ResultSchema, {
            content: "summary ignored",
            status: Advice_Level.ERROR,
            type: PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT,
          }),
          create(PlanCheckRun_ResultSchema, {
            content: "line zero ignored",
            report: {
              case: "sqlReviewReport",
              value: { startPosition: { column: 1, line: 0 } },
            },
            status: Advice_Level.ERROR,
            type: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
          }),
        ],
      }),
    ]);

    expect(markers).toEqual([
      {
        endColumn: 4,
        endLineNumber: 5,
        message: "new warning",
        severity: "WARNING",
        source: "Rule (123) L5:C4",
        startColumn: 4,
        startLineNumber: 5,
      },
    ]);
  });
});
