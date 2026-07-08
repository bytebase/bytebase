import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getRollout: vi.fn(),
  listTaskRuns: vi.fn(),
}));

vi.mock("@/connect", () => ({
  rolloutServiceClientConnect: {
    getRollout: mocks.getRollout,
    listTaskRuns: mocks.listTaskRuns,
  },
  issueServiceClientConnect: {},
  planServiceClientConnect: {},
  projectServiceClientConnect: {},
  userServiceClientConnect: {},
}));

import { fetchRolloutState } from "./fetchPlanSnapshot";

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
