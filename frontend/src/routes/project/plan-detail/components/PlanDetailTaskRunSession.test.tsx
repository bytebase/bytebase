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

vi.mock("@/api", () => ({
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

  test("a live-poll response is not masked by a slow initial fetch", async () => {
    let resolveInitial: () => void = () => {};
    mocks.getTaskRunSession
      // #1 initial load — held in flight (slow / contended DB).
      .mockReturnValueOnce(
        new Promise((resolve) => {
          resolveInitial = () => resolve(responseWithPid("111"));
        })
      )
      // #2 poll tick — resolves promptly with fresh data.
      .mockResolvedValueOnce(responseWithPid("222"))
      .mockReturnValue(new Promise(() => {}));

    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<PlanDetailTaskRunSession taskRun={runningTaskRun} />);
    });
    // While the initial fetch is still pending, the spinner is shown.
    expect(container.querySelector(".animate-spin")).not.toBeNull();

    // A poll tick resolves with data before the initial fetch settles.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(SESSION_POLL_INTERVAL_MS);
      await Promise.resolve();
    });

    // The fresh session must render now, not stay hidden behind the spinner
    // until the slow initial fetch settles.
    expect(container.textContent).toContain("222");
    expect(container.querySelector(".animate-spin")).toBeNull();

    // The initial fetch is superseded by the newer poll seq, so its late
    // resolution can't rewind the panel.
    await act(async () => {
      resolveInitial();
      await Promise.resolve();
    });
    expect(container.textContent).toContain("222");

    act(() => root.unmount());
  });

  test("does not fetch on mount while the stage is hidden (inactive)", async () => {
    mocks.getTaskRunSession.mockResolvedValue(responseWithPid("111"));
    const container = document.createElement("div");
    const root = createRoot(container);

    // Mount RUNNING but inactive (a hidden kept-alive stage): no request fires.
    await act(async () => {
      root.render(
        <PlanDetailTaskRunSession taskRun={runningTaskRun} active={false} />
      );
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(15000);
    });
    expect(mocks.getTaskRunSession).not.toHaveBeenCalled();

    // Becoming active loads it lazily.
    await act(async () => {
      root.render(<PlanDetailTaskRunSession taskRun={runningTaskRun} active />);
    });
    await act(async () => {
      await Promise.resolve();
    });
    expect(mocks.getTaskRunSession).toHaveBeenCalled();

    act(() => root.unmount());
  });

  test("pauses the session poll while the stage is hidden (inactive)", async () => {
    mocks.getTaskRunSession.mockResolvedValue(responseWithPid("111"));
    const container = document.createElement("div");
    const root = createRoot(container);

    // Active + running: the initial load plus at least one 5s poll tick fire.
    await act(async () => {
      root.render(<PlanDetailTaskRunSession taskRun={runningTaskRun} active />);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    const callsWhileActive = mocks.getTaskRunSession.mock.calls.length;
    expect(callsWhileActive).toBeGreaterThanOrEqual(2);

    // Switch stages: the card stays mounted but goes inactive — the poll stops.
    await act(async () => {
      root.render(
        <PlanDetailTaskRunSession taskRun={runningTaskRun} active={false} />
      );
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(15000);
    });
    expect(mocks.getTaskRunSession.mock.calls.length).toBe(callsWhileActive);

    act(() => root.unmount());
  });
});
