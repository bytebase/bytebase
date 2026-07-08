import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  type Task,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import type { UseTaskRunLogDataResult } from "./useTaskRunLogData";
import { useTaskRunLogData } from "./useTaskRunLogData";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  fetchRelease: vi.fn(),
  fetchRolloutByName: vi.fn(),
  fetchSheet: vi.fn(),
  getSheetByName: vi.fn(),
  getTaskRunLog: vi.fn(),
  rolloutsByName: {} as Record<string, unknown>,
}));

vi.mock("@/connect", () => ({
  rolloutServiceClientConnect: { getTaskRunLog: mocks.getTaskRunLog },
}));

vi.mock("@/react/stores/app", () => {
  const state = () => ({
    fetchRelease: mocks.fetchRelease,
    fetchRolloutByName: mocks.fetchRolloutByName,
    fetchSheet: mocks.fetchSheet,
    getSheetByName: mocks.getSheetByName,
    rolloutsByName: mocks.rolloutsByName,
  });
  const useAppStore = (selector?: (s: unknown) => unknown) =>
    selector ? selector(state()) : state();
  useAppStore.getState = state;
  return { useAppStore };
});

vi.mock("@/utils", () => ({
  extractRolloutNameFromTaskRunName: (name: string) =>
    name.split("/stages/")[0],
  extractTaskNameFromTaskRunName: (name: string) => name.split("/taskRuns/")[0],
  isReleaseBasedTask: () => false,
  releaseNameOfTaskV1: () => "",
  sheetNameOfTaskV1: (task: Task) =>
    (task as unknown as { sheet: string }).sheet,
}));

const ROLLOUT_NAME = "projects/p/rollouts/r";
const TASK_NAME = `${ROLLOUT_NAME}/stages/s/tasks/t1`;
const SHEET_NAME = "projects/p/sheets/101";

// The real isSheetContentComplete compares content byte length to
// contentSize, so fixtures carry actual encoded content.
const makeSheet = (statement: string, contentSize: number): Sheet =>
  ({
    name: SHEET_NAME,
    content: new TextEncoder().encode(statement),
    contentSize: BigInt(contentSize),
  }) as unknown as Sheet;

const makeRolloutWithTask = () => ({
  stages: [{ tasks: [{ name: TASK_NAME, sheet: SHEET_NAME }] }],
});

const renderHook = (taskRunName: string, taskRunStatus?: TaskRun_Status) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  let latest: UseTaskRunLogDataResult | undefined;
  const Probe = () => {
    latest = useTaskRunLogData(taskRunName, taskRunStatus);
    return null;
  };
  act(() => {
    root.render(<Probe />);
  });
  return {
    result: () => latest as UseTaskRunLogDataResult,
    flush: async () => {
      await act(async () => {
        await Promise.resolve();
        await Promise.resolve();
      });
    },
    unmount: () => act(() => root.unmount()),
  };
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.rolloutsByName = {};
  mocks.getTaskRunLog.mockResolvedValue({ entries: [] });
  mocks.getSheetByName.mockReturnValue(undefined);
  mocks.fetchSheet.mockResolvedValue(undefined);
  mocks.fetchRolloutByName.mockResolvedValue(makeRolloutWithTask());
});

describe("useTaskRunLogData caching", () => {
  test("resolves the task from the cached rollout without refetching it", async () => {
    mocks.rolloutsByName = { [ROLLOUT_NAME]: makeRolloutWithTask() };
    mocks.getSheetByName.mockReturnValue(makeSheet("select 1", 8));

    const hook = renderHook(`${TASK_NAME}/taskRuns/cached-rollout`);
    await hook.flush();

    expect(mocks.fetchRolloutByName).not.toHaveBeenCalled();
    expect(hook.result().metadataFetch.status).toBe("success");
    hook.unmount();
  });

  test("falls back to fetching the rollout when the cache misses", async () => {
    const hook = renderHook(`${TASK_NAME}/taskRuns/rollout-miss`);
    await hook.flush();

    expect(mocks.fetchRolloutByName).toHaveBeenCalledWith(ROLLOUT_NAME, true);
    hook.unmount();
  });

  test("reuses a complete cached sheet instead of refetching raw", async () => {
    mocks.rolloutsByName = { [ROLLOUT_NAME]: makeRolloutWithTask() };
    mocks.getSheetByName.mockReturnValue(makeSheet("select 1", 8));

    const hook = renderHook(`${TASK_NAME}/taskRuns/sheet-cached`);
    await hook.flush();

    expect(mocks.fetchSheet).not.toHaveBeenCalled();
    expect(hook.result().sheetFetch.status).toBe("success");
    expect(hook.result().sheet).toBeDefined();
    hook.unmount();
  });

  test("refetches raw when the cached sheet is a truncated preview", async () => {
    mocks.rolloutsByName = { [ROLLOUT_NAME]: makeRolloutWithTask() };
    mocks.getSheetByName.mockReturnValue(makeSheet("select", 9999));
    mocks.fetchSheet.mockResolvedValue(makeSheet("select 1 -- full", 16));

    const hook = renderHook(`${TASK_NAME}/taskRuns/sheet-truncated`);
    await hook.flush();

    expect(mocks.fetchSheet).toHaveBeenCalledWith(SHEET_NAME, true);
    expect(hook.result().sheetFetch.status).toBe("success");
    hook.unmount();
  });

  test("serves cached log entries synchronously on remount", async () => {
    mocks.rolloutsByName = { [ROLLOUT_NAME]: makeRolloutWithTask() };
    mocks.getSheetByName.mockReturnValue(makeSheet("select 1", 8));
    const taskRunName = `${TASK_NAME}/taskRuns/log-cache`;
    const entries = [{ deployId: "a" }, { deployId: "b" }];
    mocks.getTaskRunLog.mockResolvedValue({ entries });

    const first = renderHook(taskRunName);
    await first.flush();
    expect(first.result().entries).toHaveLength(2);
    first.unmount();

    // Remount with the request hanging: cached entries must paint on the very
    // first render, before any effect or request resolves.
    mocks.getTaskRunLog.mockReturnValue(new Promise(() => {}));
    const second = renderHook(taskRunName);
    expect(second.result().entries).toHaveLength(2);
    expect(second.result().logFetch.status).toBe("success");
    second.unmount();
  });

  test("skips the log refetch entirely for a cached terminal run", async () => {
    mocks.rolloutsByName = { [ROLLOUT_NAME]: makeRolloutWithTask() };
    mocks.getSheetByName.mockReturnValue(makeSheet("select 1", 8));
    const taskRunName = `${TASK_NAME}/taskRuns/terminal`;
    mocks.getTaskRunLog.mockResolvedValue({ entries: [{ deployId: "a" }] });

    const first = renderHook(taskRunName, TaskRun_Status.DONE);
    await first.flush();
    expect(mocks.getTaskRunLog).toHaveBeenCalledTimes(1);
    first.unmount();

    const second = renderHook(taskRunName, TaskRun_Status.DONE);
    await second.flush();
    expect(mocks.getTaskRunLog).toHaveBeenCalledTimes(1);
    expect(second.result().entries).toHaveLength(1);
    expect(second.result().logFetch.status).toBe("success");
    second.unmount();
  });

  test("still revalidates a cached log for a running task", async () => {
    mocks.rolloutsByName = { [ROLLOUT_NAME]: makeRolloutWithTask() };
    mocks.getSheetByName.mockReturnValue(makeSheet("select 1", 8));
    const taskRunName = `${TASK_NAME}/taskRuns/running`;
    mocks.getTaskRunLog.mockResolvedValue({ entries: [{ deployId: "a" }] });

    const first = renderHook(taskRunName, TaskRun_Status.RUNNING);
    await first.flush();
    expect(mocks.getTaskRunLog).toHaveBeenCalledTimes(1);
    first.unmount();

    mocks.getTaskRunLog.mockResolvedValue({
      entries: [{ deployId: "a" }, { deployId: "b" }],
    });
    const second = renderHook(taskRunName, TaskRun_Status.RUNNING);
    // Cached entry paints first…
    expect(second.result().entries).toHaveLength(1);
    await second.flush();
    // …then the revalidation replaces it.
    expect(mocks.getTaskRunLog).toHaveBeenCalledTimes(2);
    expect(second.result().entries).toHaveLength(2);
    second.unmount();
  });

  test("revalidates once when a cached running log remounts as done", async () => {
    mocks.rolloutsByName = { [ROLLOUT_NAME]: makeRolloutWithTask() };
    mocks.getSheetByName.mockReturnValue(makeSheet("select 1", 8));
    const taskRunName = `${TASK_NAME}/taskRuns/run-to-done`;
    mocks.getTaskRunLog.mockResolvedValue({ entries: [{ deployId: "a" }] });

    const running = renderHook(taskRunName, TaskRun_Status.RUNNING);
    await running.flush();
    running.unmount();

    // First mount at DONE: cache was recorded at RUNNING → revalidate once.
    const done = renderHook(taskRunName, TaskRun_Status.DONE);
    await done.flush();
    expect(mocks.getTaskRunLog).toHaveBeenCalledTimes(2);
    done.unmount();

    // Subsequent DONE mounts hit the terminal cache.
    const again = renderHook(taskRunName, TaskRun_Status.DONE);
    await again.flush();
    expect(mocks.getTaskRunLog).toHaveBeenCalledTimes(2);
    again.unmount();
  });

  test("a late log response from an unmounted instance cannot rewind the cache", async () => {
    mocks.rolloutsByName = { [ROLLOUT_NAME]: makeRolloutWithTask() };
    mocks.getSheetByName.mockReturnValue(makeSheet("select 1", 8));
    const taskRunName = `${TASK_NAME}/taskRuns/late-clobber`;

    // The RUNNING instance's log request stays in flight.
    let resolveRunning: (value: { entries: { deployId: string }[] }) => void =
      () => {};
    mocks.getTaskRunLog.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveRunning = resolve;
      })
    );
    const running = renderHook(taskRunName, TaskRun_Status.RUNNING);
    await running.flush();

    // Status flips RUNNING→DONE: the viewer remounts under a new key. The DONE
    // instance fetches and caches the terminal log.
    running.unmount();
    mocks.getTaskRunLog.mockResolvedValueOnce({
      entries: [{ deployId: "done-1" }, { deployId: "done-2" }],
    });
    const done = renderHook(taskRunName, TaskRun_Status.DONE);
    await done.flush();
    expect(done.result().entries).toHaveLength(2);

    // The old RUNNING request resolves late, after the terminal cache write. Its
    // partial entries must not overwrite the cached DONE log.
    await act(async () => {
      resolveRunning({ entries: [{ deployId: "running-partial" }] });
      await Promise.resolve();
      await Promise.resolve();
    });
    done.unmount();

    // A fresh DONE mount seeds synchronously from the cache; it must still be the
    // 2-entry terminal log, not the stale 1-entry running snapshot.
    mocks.getTaskRunLog.mockReturnValue(new Promise(() => {}));
    const reopened = renderHook(taskRunName, TaskRun_Status.DONE);
    expect(reopened.result().entries).toHaveLength(2);
    reopened.unmount();
  });

  test("seeds a complete cached sheet on the first render", () => {
    mocks.rolloutsByName = { [ROLLOUT_NAME]: makeRolloutWithTask() };
    mocks.getSheetByName.mockReturnValue(makeSheet("select 1", 8));
    mocks.getTaskRunLog.mockReturnValue(new Promise(() => {}));

    const hook = renderHook(`${TASK_NAME}/taskRuns/sheet-seed`);
    // No flush: the sheet must come from the synchronous render-time seed.
    expect(hook.result().sheet).toBeDefined();
    expect(hook.result().sheetFetch.status).toBe("success");
    expect(hook.result().metadataFetch.status).toBe("success");
    hook.unmount();
  });
});
