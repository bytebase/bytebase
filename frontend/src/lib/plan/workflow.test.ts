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
import {
  createPlanWithDraftReview,
  DraftReviewIssueCreationError,
  shouldStayOnPlanDetailPage,
  submitDraftReview,
} from "./workflow";

const makePlan = (cases: string[]): Plan =>
  ({
    specs: cases.map((caseName) => ({
      config: { case: caseName, value: {} },
    })),
  }) as unknown as Plan;

describe("shouldStayOnPlanDetailPage", () => {
  test("keeps database change plans on plan detail after issue creation", () => {
    expect(shouldStayOnPlanDetailPage(makePlan(["changeDatabaseConfig"]))).toBe(
      true
    );
    expect(shouldStayOnPlanDetailPage(makePlan(["createDatabaseConfig"]))).toBe(
      false
    );
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
  test("submits the persisted issue by clearing draft only", async () => {
    const draft = create(IssueSchema, {
      name: "projects/p1/issues/456",
      draft: true,
      labels: ["old"],
    });
    const submitted = create(IssueSchema, { ...draft, draft: false });
    const updateIssue = vi.fn(
      async (_request: UpdateIssueRequest) => submitted
    );

    await expect(
      submitDraftReview({ issue: draft, updateIssue })
    ).resolves.toBe(submitted);
    expect(updateIssue.mock.calls[0][0]).toMatchObject({
      issue: { draft: false },
      updateMask: { paths: ["draft"] },
    });
  });

  test("leaves labels to the metadata row rather than writing them back", async () => {
    const updateIssue = vi.fn(async (_request: UpdateIssueRequest) =>
      create(IssueSchema, {})
    );

    await submitDraftReview({
      issue: create(IssueSchema, { draft: true, labels: ["stale"] }),
      updateIssue,
    });

    expect(updateIssue.mock.calls[0][0].updateMask?.paths).not.toContain(
      "labels"
    );
  });

  test("surfaces the submission failure without retrying", async () => {
    const failure = new Error("approval setup failed");
    const updateIssue = vi.fn(async () => {
      throw failure;
    });

    await expect(
      submitDraftReview({
        issue: create(IssueSchema, { draft: true }),
        updateIssue,
      })
    ).rejects.toBe(failure);
    expect(updateIssue).toHaveBeenCalledOnce();
  });
});
