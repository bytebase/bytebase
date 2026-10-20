// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { useAppStore } from "@/stores/app";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  DatabaseGroupSchema,
  DatabaseGroupView,
} from "@/types/proto-es/v1/database_group_service_pb";
import {
  IssueSchema,
  ReviewRun_Status,
  ReviewRunSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  PlanCheckRunSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import {
  RolloutSchema,
  TaskRunSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";

const mocks = vi.hoisted(() => ({
  getCurrentUser: vi.fn(),
  getIssue: vi.fn(),
  getPlan: vi.fn(),
  getPlanCheckRun: vi.fn(),
  getProject: vi.fn(),
  getReviewRun: vi.fn(),
  getRollout: vi.fn(),
  listTaskRuns: vi.fn(),
  readsAIReviewRun: vi.fn(() => false),
}));

vi.mock("../../utils/aiReviewRun", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../utils/aiReviewRun")>()),
  readsAIReviewRun: mocks.readsAIReviewRun,
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
  projectServiceClientConnect: { getProject: mocks.getProject },
  rolloutServiceClientConnect: {
    getRollout: mocks.getRollout,
    listTaskRuns: mocks.listTaskRuns,
  },
  userServiceClientConnect: { getCurrentUser: mocks.getCurrentUser },
}));

import { fetchPlanSnapshot } from "./fetchPlanSnapshot";

const plan = (hasRollout: boolean) =>
  create(PlanSchema, {
    name: "projects/foo/plans/1",
    issue: "projects/foo/issues/1",
    hasRollout,
  });

const reviewRun = (status: ReviewRun_Status) =>
  create(ReviewRunSchema, {
    name: "projects/foo/issues/1/reviewRuns/ai",
    status,
  });

describe("fetchPlanSnapshot", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getProject.mockResolvedValue(
      create(ProjectSchema, { name: "projects/foo", title: "Foo" })
    );
    mocks.getCurrentUser.mockResolvedValue(
      create(UserSchema, { name: "users/me@example.com" })
    );
    mocks.getPlan.mockResolvedValue(plan(true));
    mocks.getIssue.mockResolvedValue(
      create(IssueSchema, {
        name: "projects/foo/issues/1",
        status: IssueStatus.DONE,
      })
    );
    mocks.getPlanCheckRun.mockResolvedValue(create(PlanCheckRunSchema));
    mocks.getRollout.mockResolvedValue(
      create(RolloutSchema, { name: "projects/foo/rollouts/1" })
    );
    mocks.listTaskRuns.mockResolvedValue({
      taskRuns: [create(TaskRunSchema, { name: "taskRuns/1" })],
    });
  });

  test("reconciles a plan fetched before rollout creation with a done issue", async () => {
    mocks.getPlan
      .mockResolvedValueOnce(plan(false))
      .mockResolvedValueOnce(plan(true));

    const patch = await fetchPlanSnapshot("foo", "1");

    expect(mocks.getPlan).toHaveBeenCalledTimes(2);
    expect(mocks.getRollout).toHaveBeenCalledTimes(1);
    expect(patch.plan.hasRollout).toBe(true);
    expect(patch.rollout?.name).toBe("projects/foo/rollouts/1");
    expect(patch.taskRuns).toHaveLength(1);
  });

  test("keeps rollout and task runs out of the patch when either fetch fails", async () => {
    mocks.listTaskRuns.mockRejectedValue(new Error("temporary failure"));

    const patch = await fetchPlanSnapshot("foo", "1", {}, true);

    expect("rollout" in patch).toBe(false);
    expect("taskRuns" in patch).toBe(false);
  });

  test("does not publish stale rollout absence when reconciliation fails", async () => {
    mocks.getPlan
      .mockResolvedValueOnce(plan(false))
      .mockRejectedValueOnce(new Error("temporary failure"));

    const patch = await fetchPlanSnapshot("foo", "1", {}, true);

    expect("rollout" in patch).toBe(false);
    expect("taskRuns" in patch).toBe(false);
  });

  test("keeps an issue fetch failure out of the patch", async () => {
    mocks.getIssue.mockRejectedValue(new Error("temporary failure"));

    const patch = await fetchPlanSnapshot("foo", "1", {}, true);

    expect("issue" in patch).toBe(false);
  });

  describe("with an open issue", () => {
    beforeEach(() => {
      mocks.getPlan.mockResolvedValue(plan(false));
      mocks.getIssue.mockResolvedValue(
        create(IssueSchema, {
          name: "projects/foo/issues/1",
          status: IssueStatus.OPEN,
        })
      );
    });

    test("reads the issue's AI review slot, and a slot that never ran as null", async () => {
      mocks.readsAIReviewRun.mockReturnValue(true);
      mocks.getReviewRun.mockRejectedValueOnce(
        new ConnectError("review run not found", Code.NotFound)
      );

      const patch = await fetchPlanSnapshot("foo", "1");

      expect(mocks.readsAIReviewRun).toHaveBeenCalledExactlyOnceWith(
        expect.objectContaining({
          plan: expect.objectContaining({ name: "projects/foo/plans/1" }),
          project: expect.objectContaining({ name: "projects/foo" }),
        })
      );
      expect(mocks.getReviewRun).toHaveBeenCalledExactlyOnceWith(
        expect.objectContaining({
          name: "projects/foo/issues/1/reviewRuns/ai",
        }),
        expect.anything()
      );
      expect(patch.aiReviewRun).toBeNull();
    });

    test("patches the AI review run it read", async () => {
      mocks.readsAIReviewRun.mockReturnValue(true);
      const run = reviewRun(ReviewRun_Status.RUNNING);
      mocks.getReviewRun.mockResolvedValueOnce(run);

      const patch = await fetchPlanSnapshot("foo", "1", {}, true);

      expect(patch.aiReviewRun).toBe(run);
    });

    test("keeps a failed AI review read out of the patch", async () => {
      mocks.readsAIReviewRun.mockReturnValue(true);
      mocks.getReviewRun.mockRejectedValueOnce(
        new ConnectError("temporary failure", Code.Unavailable)
      );

      const patch = await fetchPlanSnapshot("foo", "1", {}, true);

      expect("aiReviewRun" in patch).toBe(false);
    });

    test("gives the AI review gate the database groups the page has loaded", async () => {
      mocks.readsAIReviewRun.mockReturnValue(false);
      const group = "projects/foo/databaseGroups/empty";
      useAppStore.setState({
        dbGroupsByName: {
          [group]: create(DatabaseGroupSchema, { name: group }),
        },
        dbGroupViewByName: { [group]: DatabaseGroupView.FULL },
      });
      try {
        await fetchPlanSnapshot("foo", "1");
      } finally {
        useAppStore.setState({ dbGroupsByName: {}, dbGroupViewByName: {} });
      }

      const [gate] = mocks.readsAIReviewRun.mock.calls[0] as unknown as [
        { matchedDatabaseCount: (group: string) => number | undefined },
      ];
      expect(gate.matchedDatabaseCount(group)).toBe(0);
      expect(
        gate.matchedDatabaseCount("projects/foo/databaseGroups/unloaded")
      ).toBeUndefined();
    });

    test("does not read the AI review slot when its gate is closed", async () => {
      mocks.readsAIReviewRun.mockReturnValue(false);

      const patch = await fetchPlanSnapshot("foo", "1", {}, true);

      expect(mocks.getReviewRun).not.toHaveBeenCalled();
      expect("aiReviewRun" in patch).toBe(true);
      expect(patch.aiReviewRun).toBeUndefined();
    });
  });

  test.each([IssueStatus.DONE, IssueStatus.CANCELED])(
    "drops a running AI review run once the issue is closed: status %s",
    async (status) => {
      mocks.getPlan.mockResolvedValue(plan(false));
      mocks.getIssue.mockResolvedValue(
        create(IssueSchema, { name: "projects/foo/issues/1", status })
      );
      mocks.readsAIReviewRun.mockReturnValue(true);
      mocks.getReviewRun.mockResolvedValue(reviewRun(ReviewRun_Status.RUNNING));

      const patch = await fetchPlanSnapshot("foo", "1", {}, true);

      expect("aiReviewRun" in patch).toBe(true);
      expect(patch.aiReviewRun).toBeUndefined();
    }
  );
});
