import { create } from "@bufbuild/protobuf";
import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  ReviewRun_State,
  ReviewRunSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  PlanCheckRun_Result_Type,
  PlanCheckRun_Status,
  PlanCheckRunSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";

const mocks = vi.hoisted(() => ({
  page: {} as Record<string, unknown>,
  review: undefined as unknown,
  sqlReview: undefined as unknown,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      dbGroupsByName: {},
      dbGroupViewByName: {},
      getIssueComments: () => [],
    }),
}));

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));

vi.mock("./useSqlReviewV2", () => ({
  usePlanReviewRun: () => mocks.review,
  useSqlReviewV2: () => mocks.sqlReview,
}));

import { useAIReviewBesideV1 } from "./usePlanReviewState";

const plan = create(PlanSchema, {
  name: "projects/p/plans/1",
  specs: [
    {
      id: "spec-1",
      config: {
        case: "changeDatabaseConfig",
        value: { targets: ["instances/i/databases/d"] },
      },
    },
  ],
});

const runState = (state: ReviewRun_State) => ({
  run: create(ReviewRunSchema, {
    name: "projects/p/issues/1/reviewRun",
    state,
  }),
  stale: false,
  loadFailed: false,
  reload: vi.fn(),
});

const adviceRun = create(PlanCheckRunSchema, {
  name: "projects/p/plans/1/planCheckRun",
  status: PlanCheckRun_Status.DONE,
  results: [
    {
      type: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
      status: Advice_Level.ERROR,
      target: "instances/i/databases/d",
    },
    {
      type: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
      status: Advice_Level.WARNING,
      target: "instances/i/databases/d",
    },
  ],
});

beforeEach(() => {
  mocks.page = {
    plan,
    issue: { name: "projects/p/issues/1" },
    planCheckRuns: [adviceRun],
  };
  mocks.review = runState(ReviewRun_State.SUCCEEDED);
  mocks.sqlReview = undefined;
});

describe("useAIReviewBesideV1", () => {
  test("is the AI review's row, then the plan checks' rows", () => {
    const { result } = renderHook(() => useAIReviewBesideV1());
    expect(result.current?.rows).toMatchObject([
      { reviewer: "review", conclusion: "passed" },
      { reviewer: "policy", conclusion: "open", warnings: 1 },
    ]);
  });

  test("shows a running review as running", () => {
    mocks.review = runState(ReviewRun_State.RUNNING);
    const { result } = renderHook(() => useAIReviewBesideV1());
    expect(result.current?.rows[0]).toMatchObject({
      reviewer: "review",
      conclusion: "running",
    });
  });

  test.each([
    [
      "under SQL Review V2, which has the rows",
      () => {
        mocks.sqlReview = {};
      },
    ],
    [
      "where the page reads no review run",
      () => {
        mocks.review = undefined;
      },
    ],
  ])("is undefined %s", (_, arrange) => {
    arrange();
    const { result } = renderHook(() => useAIReviewBesideV1());
    expect(result.current).toBeUndefined();
  });
});
