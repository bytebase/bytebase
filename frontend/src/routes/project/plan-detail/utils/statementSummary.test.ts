// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { expect, test } from "vitest";
import { StatementType } from "@/types/proto-es/v1/common_pb";
import {
  PlanCheckRun_Result_SqlSummaryReportSchema,
  PlanCheckRun_Result_Type,
  PlanCheckRun_ResultSchema,
  PlanCheckRunSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { statementSummary } from "./statementSummary";

const report = (types: StatementType[], affectedRows: bigint) =>
  create(PlanCheckRun_ResultSchema, {
    type: PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT,
    report: {
      case: "sqlSummaryReport",
      value: create(PlanCheckRun_Result_SqlSummaryReportSchema, {
        statementTypes: types,
        affectedRows,
      }),
    },
  });

test("summarizes the statement kinds and the most rows a target affects", () => {
  const types = [
    StatementType.DELETE,
    StatementType.ALTER_TABLE,
    StatementType.UPDATE,
    StatementType.ALTER_TABLE,
  ];
  const summary = statementSummary([
    create(PlanCheckRunSchema, {
      results: [report(types, 600n), report(types, 900n)],
    }),
  ]);
  expect(summary).toEqual({
    types: ["DELETE", "ALTER TABLE", "UPDATE"],
    affectedRows: 900n,
  });
});

test("is undefined without a summary report", () => {
  expect(
    statementSummary([
      create(PlanCheckRunSchema, {
        results: [
          create(PlanCheckRun_ResultSchema, {
            type: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
          }),
        ],
      }),
    ])
  ).toBeUndefined();
});
