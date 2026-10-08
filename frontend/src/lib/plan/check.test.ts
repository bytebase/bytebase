// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  PlanCheckRun_Result_Type,
  PlanCheckRun_ResultSchema,
  PlanCheckRun_Status,
  PlanCheckRunSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import {
  getFilteredResultGroups,
  getPlanCheckSummary,
  getPlanCheckSummaryWithFallback,
} from "./check";

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
      error: 1,
      running: 1,
      success: 1,
      total: 4,
      warning: 1,
    });
  });

  test("counts incomplete runs once even when a timeout has an error result", () => {
    for (const status of [
      PlanCheckRun_Status.FAILED,
      PlanCheckRun_Status.CANCELED,
    ]) {
      const run = create(PlanCheckRunSchema, {
        status,
        results: [
          create(PlanCheckRun_ResultSchema, { status: Advice_Level.ERROR }),
        ],
      });
      const statusName = PlanCheckRun_Status[status];
      expect(getPlanCheckSummary([run]).error).toBe(1);
      expect(
        getPlanCheckSummaryWithFallback([], { [statusName]: 1, ERROR: 1 }).error
      ).toBe(1);
      expect(
        getPlanCheckSummaryWithFallback([], { [statusName]: 1 }).error
      ).toBe(1);
    }
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

  test("shows an incomplete run without duplicating its error result", () => {
    const groups = getFilteredResultGroups({
      includeRunFailure: true,
      planCheckRuns: [
        create(PlanCheckRunSchema, {
          name: "failed-run",
          status: PlanCheckRun_Status.FAILED,
          results: [
            create(PlanCheckRun_ResultSchema, {
              status: Advice_Level.ERROR,
              title: "Plan check run timed out",
            }),
          ],
        }),
        create(PlanCheckRunSchema, {
          name: "canceled-run",
          status: PlanCheckRun_Status.CANCELED,
          error: "Canceled by user",
        }),
      ],
      runCanceledTitle: "Canceled",
      runFailureTitle: "Failed",
    });

    expect(groups).toHaveLength(2);
    expect(groups[0]?.results[0]?.title).toBe("Plan check run timed out");
    expect(groups[1]?.results[0]?.title).toBe("Canceled");
    expect(groups[1]?.results[0]?.content).toBe("Canceled by user");
  });
});
