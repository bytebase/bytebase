import { create } from "@bufbuild/protobuf";
import { describe, expect, test, vi } from "vitest";
import type {
  CreateIssueRequest,
  UpdateIssueRequest,
} from "@/types/proto-es/v1/issue_service_pb";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import type {
  CreatePlanRequest,
  Plan,
} from "@/types/proto-es/v1/plan_service_pb";
import { PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  createPlanWithDraftReview,
  DraftReviewIssueCreationError,
  getCreateIssueBlockingErrors,
  getCreateIssueConfirmErrors,
  getCreatePlanBlockingReasons,
  hasChecksWarning,
  shouldStayOnPlanDetailPage,
  submitDraftReview,
} from "./workflow";

const t = (key: string) => key;

const makePlan = (cases: string[], statusCount = {}): Plan =>
  ({
    planCheckRunStatusCount: statusCount,
    specs: cases.map((caseName) => ({
      config: { case: caseName, value: {} },
    })),
  }) as unknown as Plan;

describe("plan detail header create issue helpers", () => {
  test("keeps database change plans on plan detail after issue creation", () => {
    expect(shouldStayOnPlanDetailPage(makePlan(["changeDatabaseConfig"]))).toBe(
      true
    );
    expect(shouldStayOnPlanDetailPage(makePlan(["createDatabaseConfig"]))).toBe(
      false
    );
  });

  test("flags non-blocking check warnings without adding a confirm error", () => {
    const plan = makePlan(["changeDatabaseConfig"], { ERROR: 1 });
    const project = {
      enforceSqlReview: false,
      forceIssueLabels: false,
    } as Project;
    const blockingErrors = getCreateIssueBlockingErrors({
      emptySpecCount: 0,
      plan,
      project,
      t,
    });

    expect(blockingErrors).toEqual([]);
    expect(hasChecksWarning(plan)).toBe(true);
    expect(
      getCreateIssueConfirmErrors({
        blockingErrors,
        project,
        selectedLabelCount: 0,
        t,
      })
    ).toEqual([]);
  });

  test("keeps SQL review enforcement as a blocking error", () => {
    const plan = makePlan(["changeDatabaseConfig"], { ERROR: 1 });
    const project = {
      enforceSqlReview: true,
      forceIssueLabels: false,
    } as Project;

    expect(
      getCreateIssueBlockingErrors({
        emptySpecCount: 0,
        plan,
        project,
        t,
      })
    ).toContain(
      "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
    );
  });

  test("blocks issue creation while plan checks are queued", () => {
    const project = {
      enforceSqlReview: false,
      forceIssueLabels: false,
    } as Project;

    expect(
      getCreateIssueBlockingErrors({
        emptySpecCount: 0,
        plan: makePlan(["changeDatabaseConfig"], { AVAILABLE: 1 }),
        project,
        t,
      })
    ).toContain(
      "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
    );
  });

  test("blocks data export issue creation", () => {
    const project = {
      enforceSqlReview: false,
      forceIssueLabels: false,
    } as Project;

    expect(
      getCreateIssueBlockingErrors({
        emptySpecCount: 0,
        plan: makePlan(["exportDataConfig"]),
        project,
        t,
      })
    ).toContain("issue.data-export.creation-not-supported");
  });
});

describe("createPlanWithDraftReview", () => {
  const plan = create(PlanSchema, {
    name: "projects/p1/plans/placeholder",
    title: "Add index",
    description: "Reduce query latency",
  });
  const createdPlan = create(PlanSchema, {
    ...plan,
    name: "projects/p1/plans/123",
  });
  const draftIssue = create(IssueSchema, {
    name: "projects/p1/issues/456",
    draft: true,
    plan: createdPlan.name,
  });

  test("creates the plan first and then one linked draft issue with initial labels", async () => {
    const calls: string[] = [];
    const createPlan = vi.fn(async (_request: CreatePlanRequest) => {
      calls.push("plan");
      return createdPlan;
    });
    const createIssue = vi.fn(async (_request: CreateIssueRequest) => {
      calls.push("issue");
      return draftIssue;
    });

    const result = await createPlanWithDraftReview({
      createIssue,
      createPlan,
      creator: "users/dev@example.com",
      labels: ["prod", "security"],
      parent: "projects/p1",
      plan,
    });

    expect(calls).toEqual(["plan", "issue"]);
    expect(result).toEqual({ issue: draftIssue, plan: createdPlan });
    expect(createIssue).toHaveBeenCalledOnce();
    expect(createIssue.mock.calls[0][0].issue).toMatchObject({
      creator: "users/dev@example.com",
      description: "Reduce query latency",
      draft: true,
      labels: ["prod", "security"],
      plan: "projects/p1/plans/123",
      title: "Add index",
    });
  });

  test("reports the persisted malformed plan and never retries when draft issue creation fails", async () => {
    const createPlan = vi.fn(async () => createdPlan);
    const failure = new Error("issue creation denied");
    const createIssue = vi.fn(async () => {
      throw failure;
    });

    const promise = createPlanWithDraftReview({
      createIssue,
      createPlan,
      creator: "users/dev@example.com",
      labels: [],
      parent: "projects/p1",
      plan,
    });

    await expect(promise).rejects.toMatchObject({
      cause: failure,
      plan: createdPlan,
    } satisfies Partial<DraftReviewIssueCreationError>);
    expect(createPlan).toHaveBeenCalledOnce();
    expect(createIssue).toHaveBeenCalledOnce();
  });
});

describe("submitDraftReview", () => {
  test("submits the persisted issue by clearing draft and retaining selected labels", async () => {
    const draft = create(IssueSchema, {
      name: "projects/p1/issues/456",
      draft: true,
      labels: ["old"],
    });
    const submitted = create(IssueSchema, {
      ...draft,
      draft: false,
      labels: ["prod"],
    });
    const updateIssue = vi.fn(
      async (_request: UpdateIssueRequest) => submitted
    );

    await expect(
      submitDraftReview({ issue: draft, labels: ["prod"], updateIssue })
    ).resolves.toBe(submitted);
    expect(updateIssue.mock.calls[0][0]).toMatchObject({
      issue: { draft: false, labels: ["prod"] },
      updateMask: { paths: ["draft", "labels"] },
    });
  });

  test("surfaces the submission failure without retrying", async () => {
    const failure = new Error("approval setup failed");
    const updateIssue = vi.fn(async () => {
      throw failure;
    });

    await expect(
      submitDraftReview({
        issue: create(IssueSchema, { draft: true }),
        labels: [],
        updateIssue,
      })
    ).rejects.toBe(failure);
    expect(updateIssue).toHaveBeenCalledOnce();
  });
});

describe("getCreatePlanBlockingReasons", () => {
  test("flags an empty title", () => {
    expect(
      getCreatePlanBlockingReasons({ title: "", emptySpecCount: 0, t })
    ).toEqual(["plan.title-required"]);
  });

  test("flags a whitespace-only title", () => {
    expect(
      getCreatePlanBlockingReasons({ title: "   ", emptySpecCount: 0, t })
    ).toEqual(["plan.title-required"]);
  });

  test("flags empty statements", () => {
    expect(
      getCreatePlanBlockingReasons({
        title: "Add column",
        emptySpecCount: 2,
        t,
      })
    ).toEqual(["plan.navigator.statement-empty"]);
  });

  test("lists both blockers, title first", () => {
    expect(
      getCreatePlanBlockingReasons({ title: "", emptySpecCount: 1, t })
    ).toEqual(["plan.title-required", "plan.navigator.statement-empty"]);
  });

  test("returns no reasons when valid", () => {
    expect(
      getCreatePlanBlockingReasons({
        title: "Add column",
        emptySpecCount: 0,
        t,
      })
    ).toEqual([]);
  });
});
