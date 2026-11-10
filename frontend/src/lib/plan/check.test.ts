// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  Plan_SpecSchema,
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
  planCheckRunListForSpec,
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

describe("planCheckRunListForSpec", () => {
  const DB = "instances/i/databases/shop";
  const SHEET_A = "a".repeat(64);
  const SHEET_B = "b".repeat(64);
  const spec = (id: string, sheet: string) =>
    create(Plan_SpecSchema, {
      id,
      config: {
        case: "changeDatabaseConfig",
        value: { targets: [DB], sheet: `projects/p/sheets/${sheet}` },
      },
    });
  const report = (title: string, sheetSha256: string) =>
    create(PlanCheckRun_ResultSchema, {
      type: PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT,
      target: DB,
      title,
      sheetSha256,
    });
  const titles = (
    sheet: string,
    runs = [
      create(PlanCheckRunSchema, {
        status: PlanCheckRun_Status.DONE,
        results: [
          report("a", SHEET_A),
          report("b", SHEET_B),
          report("old", ""),
        ],
      }),
    ]
  ) =>
    planCheckRunListForSpec(runs, spec("spec", sheet)).flatMap((run) =>
      run.results.map((result) => result.title)
    );

  test("gives each change on one database only its own sheet's results", () => {
    expect(titles(SHEET_A)).toEqual(["a", "old"]);
    expect(titles(SHEET_B)).toEqual(["b", "old"]);
  });
});
