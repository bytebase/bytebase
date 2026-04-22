import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  PlanCheckRun_Result_Type,
  PlanCheckRun_ResultSchema,
  PlanCheckRun_Status,
  PlanCheckRunSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { getFilteredResultGroups, getPlanCheckSummary } from "./planCheck";

describe("plan check helpers", () => {
  test("summarizes run-level and result-level statuses", () => {
    const summary = getPlanCheckSummary([
      create(PlanCheckRunSchema, {
        status: PlanCheckRun_Status.RUNNING,
      }),
      create(PlanCheckRunSchema, {
        results: [
          create(PlanCheckRun_ResultSchema, {
            status: Advice_Level.ERROR,
          }),
          create(PlanCheckRun_ResultSchema, {
            status: Advice_Level.WARNING,
          }),
          create(PlanCheckRun_ResultSchema, {
            status: Advice_Level.SUCCESS,
          }),
        ],
        status: PlanCheckRun_Status.FAILED,
      }),
    ]);

    expect(summary).toEqual({
      error: 2,
      running: 1,
      success: 1,
      total: 5,
      warning: 1,
    });
  });

  test("groups filtered results by type and target", () => {
    const groups = getFilteredResultGroups({
      planCheckRuns: [
        create(PlanCheckRunSchema, {
          results: [
            create(PlanCheckRun_ResultSchema, {
              status: Advice_Level.ERROR,
              target: "databases/db1",
              type: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
            }),
            create(PlanCheckRun_ResultSchema, {
              status: Advice_Level.WARNING,
              target: "databases/db1",
              type: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
            }),
            create(PlanCheckRun_ResultSchema, {
              status: Advice_Level.ERROR,
              target: "databases/db2",
              type: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
            }),
          ],
        }),
      ],
      selectedStatus: Advice_Level.ERROR,
    });

    expect(groups).toHaveLength(2);
    expect(groups.map((group) => group.target)).toEqual([
      "databases/db1",
      "databases/db2",
    ]);
    expect(groups[0]?.results).toHaveLength(1);
  });
});
