import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  type TaskRun,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { PlanDetailTaskRunSession } from "./PlanDetailTaskRunSession";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  getTaskRunSession: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/connect", () => ({
  rolloutServiceClientConnect: { getTaskRunSession: mocks.getTaskRunSession },
}));

// The component's live-session poll interval (module-private constant).
const SESSION_POLL_INTERVAL_MS = 5000;

// A minimal getTaskRunSession response carrying one Postgres session with the
// given pid, enough for the table to render and to tell responses apart.
const responseWithPid = (pid: string) => ({
  session: {
    case: "postgres" as const,
    value: {
      session: {
        pid,
        query: `select ${pid}`,
        state: "active",
        blockedByPids: [],
      },
      blockingSessions: [],
      blockedSessions: [],
    },
  },
});

const runningTaskRun = {
  name: "projects/p/rollouts/1/stages/s/tasks/t/taskRuns/1",
  status: TaskRun_Status.RUNNING,
} as unknown as TaskRun;

describe("PlanDetailTaskRunSession", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  test("an out-of-order session response cannot overwrite fresher data", async () => {
    let resolveFirst: () => void = () => {};
    let resolveSecond: () => void = () => {};
    mocks.getTaskRunSession
      .mockReturnValueOnce(
        new Promise((resolve) => {
          resolveFirst = () => resolve(responseWithPid("111"));
        })
      )
      .mockReturnValueOnce(
        new Promise((resolve) => {
          resolveSecond = () => resolve(responseWithPid("222"));
        })
      )
      .mockReturnValue(new Promise(() => {}));

    const container = document.createElement("div");
    const root = createRoot(container);

    // Mount RUNNING: the initial-load fetch (#1) is in flight.
    await act(async () => {
      root.render(<PlanDetailTaskRunSession taskRun={runningTaskRun} />);
    });

    // A live-poll tick fires the second fetch (#2) while #1 is still pending.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(SESSION_POLL_INTERVAL_MS);
    });

    // The newer fetch resolves first with the fresh session…
    await act(async () => {
      resolveSecond();
      await Promise.resolve();
    });
    // …then the older, superseded fetch resolves late (also clearing loading).
    await act(async () => {
      resolveFirst();
      await Promise.resolve();
    });

    // The stale response must be dropped: the panel shows the fresh session.
    expect(container.textContent).toContain("222");
    expect(container.textContent).not.toContain("111");

    act(() => root.unmount());
  });
});
