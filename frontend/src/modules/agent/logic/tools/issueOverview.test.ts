// @vitest-environment node

import { Code, ConnectError } from "@connectrpc/connect";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  Issue_Type,
  ReviewRun_Status,
} from "@/types/proto-es/v1/issue_service_pb";
import { PlanCheckRun_Status } from "@/types/proto-es/v1/plan_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";

const mocks = vi.hoisted(() => ({
  getIssue: vi.fn(),
  getReviewRun: vi.fn(),
  getPlan: vi.fn(),
  getPlanCheckRun: vi.fn(),
  getRollout: vi.fn(),
  listTaskRuns: vi.fn(),
}));

vi.mock("@/api", () => ({
  issueServiceClientConnect: {
    getIssue: mocks.getIssue,
    getReviewRun: mocks.getReviewRun,
  },
  planServiceClientConnect: {
    getPlan: mocks.getPlan,
    getPlanCheckRun: mocks.getPlanCheckRun,
  },
  rolloutServiceClientConnect: {
    getRollout: mocks.getRollout,
    listTaskRuns: mocks.listTaskRuns,
  },
}));

import {
  collectIssueOverview,
  createIssueOverviewTool,
  extractIssueOverviewEvidence,
} from "./issueOverview";

const issueName = "projects/orders/issues/123";
const planName = "projects/orders/plans/plan-1";

describe("issueOverview", () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset());
  });

  test("collects a bounded view of an Issue, its Plan, checks, reviews, and rollout", async () => {
    mocks.getIssue.mockResolvedValue({
      name: issueName,
      title: "Add customer table",
      type: Issue_Type.DATABASE_CHANGE,
      status: IssueStatus.OPEN,
      approvalStatus: ApprovalStatus.PENDING,
      plan: planName,
    });
    mocks.getPlan.mockResolvedValue({
      name: planName,
      title: "Add customer table",
      hasRollout: true,
    });
    mocks.getPlanCheckRun.mockResolvedValue({
      name: `${planName}/planCheckRun`,
      status: PlanCheckRun_Status.DONE,
      results: [
        { status: Advice_Level.SUCCESS },
        { status: Advice_Level.WARNING },
        { status: Advice_Level.ERROR },
      ],
    });
    mocks.getReviewRun.mockImplementation((request) =>
      Promise.resolve({
        name: request.name,
        status:
          request.name === `${issueName}/reviewRuns/rule`
            ? ReviewRun_Status.DONE
            : ReviewRun_Status.RUNNING,
      })
    );
    mocks.getRollout.mockResolvedValue({
      name: `${planName}/rollout`,
      stages: [{}, {}],
    });
    mocks.listTaskRuns.mockResolvedValue({
      taskRuns: [
        {
          name: `${planName}/rollout/stages/prod/tasks/task-1/taskRuns/run-1`,
          status: TaskRun_Status.DONE,
        },
        {
          name: `${planName}/rollout/stages/prod/tasks/task-2/taskRuns/run-2`,
          status: TaskRun_Status.RUNNING,
        },
      ],
      nextPageToken: "next-page",
    });

    await expect(
      collectIssueOverview("projects/orders", issueName)
    ).resolves.toEqual({
      project: "projects/orders",
      issue: {
        status: "available",
        name: issueName,
        title: "Add customer table",
        type: "DATABASE_CHANGE",
        issueStatus: "OPEN",
        approvalStatus: "PENDING",
      },
      plan: {
        status: "available",
        name: planName,
        title: "Add customer table",
        hasRollout: true,
      },
      planCheck: {
        status: "available",
        runStatus: "DONE",
        resultCounts: { success: 1, warning: 1, error: 1 },
      },
      reviews: {
        rule: { status: "available", runStatus: "DONE" },
        ai: { status: "available", runStatus: "RUNNING" },
      },
      rollout: {
        status: "available",
        name: `${planName}/rollout`,
        stageCount: 2,
      },
      taskRuns: {
        status: "available",
        partial: true,
        taskRuns: [
          {
            name: `${planName}/rollout/stages/prod/tasks/task-1/taskRuns/run-1`,
            status: "DONE",
          },
          {
            name: `${planName}/rollout/stages/prod/tasks/task-2/taskRuns/run-2`,
            status: "RUNNING",
          },
        ],
      },
      evidence: {
        project: "projects/orders",
        references: [
          { kind: "issue", name: issueName, title: "Add customer table" },
        ],
      },
    });
    expect(mocks.getIssue).toHaveBeenCalledWith(
      expect.objectContaining({ name: issueName })
    );
    expect(mocks.getPlan).toHaveBeenCalledWith(
      expect.objectContaining({ name: planName })
    );
    expect(mocks.getPlanCheckRun).toHaveBeenCalledWith(
      expect.objectContaining({ name: `${planName}/planCheckRun` })
    );
    expect(mocks.getReviewRun).toHaveBeenCalledWith(
      expect.objectContaining({ name: `${issueName}/reviewRuns/rule` })
    );
    expect(mocks.getReviewRun).toHaveBeenCalledWith(
      expect.objectContaining({ name: `${issueName}/reviewRuns/ai` })
    );
    expect(mocks.getRollout).toHaveBeenCalledWith(
      expect.objectContaining({ name: `${planName}/rollout` })
    );
    expect(mocks.listTaskRuns).toHaveBeenCalledWith(
      expect.objectContaining({
        parent: `${planName}/rollout/stages/-/tasks/-`,
        pageSize: 10,
      })
    );
  });

  test("keeps independent Issue facets explicit when a read is unavailable", async () => {
    mocks.getIssue.mockResolvedValue({
      name: issueName,
      title: "Add customer table",
      type: Issue_Type.DATABASE_CHANGE,
      status: IssueStatus.OPEN,
      approvalStatus: ApprovalStatus.PENDING,
      plan: planName,
    });
    mocks.getPlan.mockRejectedValue(new Error("plan unavailable"));
    mocks.getPlanCheckRun.mockRejectedValue(new Error("checks unavailable"));
    mocks.getReviewRun.mockRejectedValue(new Error("reviews unavailable"));

    await expect(
      collectIssueOverview("projects/orders", issueName)
    ).resolves.toMatchObject({
      issue: { status: "available", name: issueName },
      plan: { status: "unavailable" },
      planCheck: { status: "unavailable" },
      reviews: {
        rule: { status: "unavailable" },
        ai: { status: "unavailable" },
      },
      rollout: { status: "unavailable" },
      taskRuns: { status: "unavailable" },
    });
    expect(mocks.getRollout).not.toHaveBeenCalled();
    expect(mocks.listTaskRuns).not.toHaveBeenCalled();
  });

  test("treats missing dependent resources as not configured", async () => {
    mocks.getIssue.mockResolvedValue({
      name: issueName,
      title: "Add customer table",
      type: Issue_Type.DATABASE_CHANGE,
      status: IssueStatus.OPEN,
      approvalStatus: ApprovalStatus.PENDING,
      plan: planName,
    });
    mocks.getPlan.mockResolvedValue({
      name: planName,
      title: "Add customer table",
      hasRollout: false,
    });
    mocks.getPlanCheckRun.mockRejectedValue(
      new ConnectError("not found", Code.NotFound)
    );
    mocks.getReviewRun.mockRejectedValue(
      new ConnectError("not found", Code.NotFound)
    );

    await expect(
      collectIssueOverview("projects/orders", issueName)
    ).resolves.toMatchObject({
      plan: { status: "available", hasRollout: false },
      planCheck: { status: "not_configured" },
      reviews: {
        rule: { status: "not_configured" },
        ai: { status: "not_configured" },
      },
      rollout: { status: "not_started" },
      taskRuns: { status: "not_started" },
    });
  });

  test("does not expose a mismatched dependent resource", async () => {
    mocks.getIssue.mockResolvedValue({
      name: issueName,
      title: "Add customer table",
      type: Issue_Type.DATABASE_CHANGE,
      status: IssueStatus.OPEN,
      approvalStatus: ApprovalStatus.PENDING,
      plan: planName,
    });
    mocks.getPlan.mockResolvedValue({
      name: planName,
      title: "Add customer table",
      hasRollout: true,
    });
    mocks.getPlanCheckRun.mockResolvedValue({
      name: "projects/other/plans/plan-2/planCheckRun",
      status: PlanCheckRun_Status.DONE,
      results: [],
    });
    mocks.getReviewRun.mockImplementation((request) =>
      Promise.resolve({
        name:
          request.name === `${issueName}/reviewRuns/rule`
            ? "projects/other/issues/5/reviewRuns/rule"
            : request.name,
        status: ReviewRun_Status.DONE,
      })
    );
    mocks.getRollout.mockResolvedValue({
      name: "projects/other/plans/plan-2/rollout",
      stages: [],
    });

    await expect(
      collectIssueOverview("projects/orders", issueName)
    ).resolves.toMatchObject({
      planCheck: { status: "unavailable" },
      reviews: {
        rule: { status: "unavailable" },
        ai: { status: "available" },
      },
      rollout: { status: "unavailable" },
      taskRuns: { status: "unavailable" },
    });
    expect(mocks.listTaskRuns).not.toHaveBeenCalled();
  });

  test("does not expose task runs outside the selected rollout", async () => {
    mocks.getIssue.mockResolvedValue({
      name: issueName,
      title: "Add customer table",
      type: Issue_Type.DATABASE_CHANGE,
      status: IssueStatus.OPEN,
      approvalStatus: ApprovalStatus.PENDING,
      plan: planName,
    });
    mocks.getPlan.mockResolvedValue({
      name: planName,
      title: "Add customer table",
      hasRollout: true,
    });
    mocks.getPlanCheckRun.mockResolvedValue({
      name: `${planName}/planCheckRun`,
      status: PlanCheckRun_Status.DONE,
      results: [],
    });
    mocks.getReviewRun.mockImplementation((request) =>
      Promise.resolve({ name: request.name, status: ReviewRun_Status.DONE })
    );
    mocks.getRollout.mockResolvedValue({
      name: `${planName}/rollout`,
      stages: [],
    });
    mocks.listTaskRuns.mockResolvedValue({
      taskRuns: [
        {
          name: "projects/other/plans/plan-2/rollout/stages/prod/tasks/task-1/taskRuns/run-1",
          status: TaskRun_Status.DONE,
        },
      ],
    });

    await expect(
      collectIssueOverview("projects/orders", issueName)
    ).resolves.toMatchObject({
      rollout: { status: "available", name: `${planName}/rollout` },
      taskRuns: { status: "unavailable" },
    });
  });

  test("does not read an Issue outside the chat-bound project", async () => {
    await expect(
      collectIssueOverview("projects/orders", "projects/other/issues/123")
    ).resolves.toBeNull();
    await expect(
      collectIssueOverview(
        "projects/orders.v2",
        "projects/ordersXv2/issues/123"
      )
    ).resolves.toBeNull();
    expect(mocks.getIssue).not.toHaveBeenCalled();
    expect(mocks.getPlan).not.toHaveBeenCalled();
    expect(mocks.getPlanCheckRun).not.toHaveBeenCalled();
    expect(mocks.getReviewRun).not.toHaveBeenCalled();
    expect(mocks.getRollout).not.toHaveBeenCalled();
    expect(mocks.listTaskRuns).not.toHaveBeenCalled();
  });

  test("makes a missing Plan, review run, and rollout explicit without treating them as errors", async () => {
    mocks.getIssue.mockResolvedValue({
      name: issueName,
      title: "Request production access",
      type: Issue_Type.ACCESS_GRANT,
      status: IssueStatus.OPEN,
      approvalStatus: ApprovalStatus.PENDING,
      plan: "",
    });

    await expect(
      collectIssueOverview("projects/orders", issueName)
    ).resolves.toMatchObject({
      plan: { status: "not_configured" },
      planCheck: { status: "not_configured" },
      reviews: {
        rule: { status: "not_configured" },
        ai: { status: "not_configured" },
      },
      rollout: { status: "not_started" },
      taskRuns: { status: "not_started" },
    });
    expect(mocks.getPlan).not.toHaveBeenCalled();
    expect(mocks.getPlanCheckRun).not.toHaveBeenCalled();
    expect(mocks.getReviewRun).not.toHaveBeenCalled();
  });

  test("returns a project-bound JSON snapshot and only accepts scoped evidence", async () => {
    mocks.getIssue.mockResolvedValue({
      name: issueName,
      title: "Add customer table",
      type: Issue_Type.DATABASE_CHANGE,
      status: IssueStatus.OPEN,
      approvalStatus: ApprovalStatus.PENDING,
      plan: "",
    });

    await expect(
      createIssueOverviewTool({ projectName: "projects/orders" })({
        issueName,
      })
    ).resolves.toContain('"project":"projects/orders"');
    expect(
      extractIssueOverviewEvidence(
        JSON.stringify({
          evidence: {
            project: "projects/orders",
            references: [
              {
                kind: "issue",
                name: "projects/other/issues/123",
                title: "Other project",
              },
            ],
          },
        })
      )
    ).toBeUndefined();
  });
});
