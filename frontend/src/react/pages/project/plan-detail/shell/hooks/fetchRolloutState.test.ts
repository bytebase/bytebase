import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getRollout: vi.fn(),
  listTaskRuns: vi.fn(),
  getPlan: vi.fn(),
  getProject: vi.fn(),
  getCurrentUser: vi.fn(),
  getIssue: vi.fn(),
  getPlanCheckRun: vi.fn(),
}));

vi.mock("@/connect", () => ({
  rolloutServiceClientConnect: {
    getRollout: mocks.getRollout,
    listTaskRuns: mocks.listTaskRuns,
  },
  issueServiceClientConnect: { getIssue: mocks.getIssue },
  planServiceClientConnect: {
    getPlan: mocks.getPlan,
    getPlanCheckRun: mocks.getPlanCheckRun,
  },
  projectServiceClientConnect: { getProject: mocks.getProject },
  userServiceClientConnect: { getCurrentUser: mocks.getCurrentUser },
}));

import { fetchPlanSnapshot, fetchRolloutState } from "./fetchPlanSnapshot";

const ROLLOUT = "projects/p/rollouts/1";

describe("fetchRolloutState", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("returns the rollout and its task runs when both resolve", async () => {
    mocks.getRollout.mockResolvedValue({ name: ROLLOUT });
    mocks.listTaskRuns.mockResolvedValue({ taskRuns: [{ name: "tr1" }] });

    const patch = await fetchRolloutState(ROLLOUT);

    expect(patch.rollout).toEqual({ name: ROLLOUT });
    expect(patch.taskRuns).toEqual([{ name: "tr1" }]);

    // Both slim-lane requests must be silent: they poll every ~500ms, so a
    // transient failure must not spam the global error toast.
    const contextArg = expect.objectContaining({
      contextValues: expect.anything(),
    });
    expect(mocks.getRollout).toHaveBeenCalledWith(
      expect.anything(),
      contextArg
    );
    expect(mocks.listTaskRuns).toHaveBeenCalledWith(
      expect.anything(),
      contextArg
    );
  });

  test("returns an empty patch when task runs fail — no partial terminal rollout", async () => {
    // The rollout could report all tasks terminal; applying it without the
    // task-run refresh would stop polling with a stale RUNNING run shown.
    mocks.getRollout.mockResolvedValue({ name: ROLLOUT, stages: [] });
    mocks.listTaskRuns.mockRejectedValue(new Error("boom"));

    const patch = await fetchRolloutState(ROLLOUT);

    expect(patch).toEqual({});
  });

  test("returns an empty patch when the rollout fails", async () => {
    mocks.getRollout.mockRejectedValue(new Error("boom"));
    mocks.listTaskRuns.mockResolvedValue({ taskRuns: [] });

    const patch = await fetchRolloutState(ROLLOUT);

    expect(patch).toEqual({});
  });
});

describe("fetchPlanSnapshot silent flag", () => {
  // hasRollout:false + no issue keeps the fetch to plan/project/user/checkRun.
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getPlan.mockResolvedValue({
      name: "projects/p/plans/1",
      hasRollout: false,
    });
    mocks.getProject.mockResolvedValue({
      title: "P",
      requireIssueApproval: false,
      requirePlanCheckNoError: false,
    });
    mocks.getCurrentUser.mockResolvedValue({ name: "users/me" });
    mocks.getPlanCheckRun.mockResolvedValue({});
  });

  const silentArg = expect.objectContaining({
    contextValues: expect.anything(),
  });

  test("a background poll (silent=true) passes the silent context to every request", async () => {
    await fetchPlanSnapshot("p", "1", {}, true);
    expect(mocks.getPlan).toHaveBeenCalledWith(expect.anything(), silentArg);
    expect(mocks.getProject).toHaveBeenCalledWith(expect.anything(), silentArg);
    expect(mocks.getCurrentUser).toHaveBeenCalledWith({}, silentArg);
    expect(mocks.getPlanCheckRun).toHaveBeenCalledWith(
      expect.anything(),
      silentArg
    );
  });

  test("the initial load (silent=false) stays loud — no silent context", async () => {
    await fetchPlanSnapshot("p", "1", {}, false);
    expect(mocks.getPlan).toHaveBeenCalledWith(expect.anything(), undefined);
    expect(mocks.getProject).toHaveBeenCalledWith(expect.anything(), undefined);
    expect(mocks.getCurrentUser).toHaveBeenCalledWith({}, undefined);
    expect(mocks.getPlanCheckRun).toHaveBeenCalledWith(
      expect.anything(),
      undefined
    );
  });
});
