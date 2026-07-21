import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { IssueSchema, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
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
  getRollout: vi.fn(),
  listTaskRuns: vi.fn(),
}));

vi.mock("@/api", () => ({
  issueServiceClientConnect: { getIssue: mocks.getIssue },
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
});
