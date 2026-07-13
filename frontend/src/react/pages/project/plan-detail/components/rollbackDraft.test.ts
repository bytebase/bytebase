import { create } from "@bufbuild/protobuf";
import { describe, expect, test, vi } from "vitest";
import type { CreateIssueRequest } from "@/types/proto-es/v1/issue_service_pb";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { CreatePlanRequest } from "@/types/proto-es/v1/plan_service_pb";
import { PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import { DraftReviewIssueCreationError } from "../utils/header";
import { createRollbackDraftReview } from "./rollbackDraft";

const preview = {
  statement: "UPDATE t SET active = FALSE;",
  target: "instances/i/databases/d",
};
const localizedTitle = "Localized rollback title";
const localizedDescription = "Localized rollback description";

describe("createRollbackDraftReview", () => {
  test("persists rollback sheets, then creates one Plan and one linked draft Issue", async () => {
    const calls: string[] = [];
    const createSheet = vi.fn(async (sheet) => ({
      ...sheet,
      name: "projects/p1/sheets/sheet-1",
    }));
    const createPlan = vi.fn(async (request: CreatePlanRequest) => {
      calls.push("plan");
      const plan = create(PlanSchema, request.plan);
      plan.name = "projects/p1/plans/plan-1";
      return plan;
    });
    const createIssue = vi.fn(async (request: CreateIssueRequest) => {
      calls.push("issue");
      const issue = create(IssueSchema, request.issue);
      issue.name = "projects/p1/issues/issue-1";
      return issue;
    });

    const result = await createRollbackDraftReview({
      createIssue,
      createPlan,
      createSheet,
      creator: "users/dev@example.com",
      newId: () => "generated",
      parent: "projects/p1",
      previews: [preview],
      title: localizedTitle,
      description: localizedDescription,
    });

    expect(calls).toEqual(["plan", "issue"]);
    expect(createSheet).toHaveBeenCalledWith(
      expect.objectContaining({
        content: new TextEncoder().encode(preview.statement),
        name: "projects/p1/sheets/generated",
      })
    );
    expect(createPlan.mock.calls[0][0].plan).toMatchObject({
      creator: "users/dev@example.com",
      specs: [
        {
          config: {
            case: "changeDatabaseConfig",
            value: {
              sheet: "projects/p1/sheets/sheet-1",
              targets: [preview.target],
            },
          },
        },
      ],
      title: localizedTitle,
      description: localizedDescription,
    });
    expect(createIssue.mock.calls[0][0].issue).toMatchObject({
      creator: "users/dev@example.com",
      draft: true,
      labels: [],
      plan: "projects/p1/plans/plan-1",
    });
    expect(result.plan.name).toBe("projects/p1/plans/plan-1");
  });

  test("does not retry and exposes the persisted malformed Plan when Issue creation fails", async () => {
    const createdPlan = create(PlanSchema, {
      name: "projects/p1/plans/plan-1",
    });
    const createPlan = vi.fn(async () => createdPlan);
    const createIssue = vi.fn(async () => {
      throw new Error("draft issue failed");
    });

    await expect(
      createRollbackDraftReview({
        createIssue,
        createPlan,
        createSheet: vi.fn(async (sheet) => ({
          ...sheet,
          name: "projects/p1/sheets/sheet-1",
        })),
        creator: "users/dev@example.com",
        newId: () => "generated",
        parent: "projects/p1",
        previews: [preview],
        title: localizedTitle,
        description: localizedDescription,
      })
    ).rejects.toMatchObject({
      plan: createdPlan,
    } satisfies Partial<DraftReviewIssueCreationError>);
    expect(createPlan).toHaveBeenCalledOnce();
    expect(createIssue).toHaveBeenCalledOnce();
  });
});
