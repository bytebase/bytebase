import { Code, ConnectError } from "@connectrpc/connect";
import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { ReviewRun_State } from "@/types/proto-es/v1/issue_service_pb";

const ISSUE = "projects/foo/issues/7";
const RUN = `${ISSUE}/reviewRun`;

const mocks = vi.hoisted(() => ({
  fetchIssueCommentThreads: vi.fn(),
  getReviewRun: vi.fn(),
}));

vi.mock("@/api", () => ({
  issueServiceClientConnect: { getReviewRun: mocks.getReviewRun },
}));

vi.mock("@/stores/app", () => {
  const state = {
    fetchIssueCommentThreads: mocks.fetchIssueCommentThreads,
  };
  return { useAppStore: { getState: () => state } };
});

import { useReviewRun } from "./useReviewRun";

const reviewRun = (state: ReviewRun_State) => ({ name: RUN, state });

const flush = () =>
  act(async () => {
    await vi.advanceTimersByTimeAsync(0);
  });

beforeEach(() => {
  vi.useFakeTimers();
  vi.clearAllMocks();
  mocks.getReviewRun.mockReset();
  mocks.fetchIssueCommentThreads.mockResolvedValue(undefined);
  mocks.getReviewRun.mockRejectedValue(
    new ConnectError("not found", Code.NotFound)
  );
});

afterEach(() => {
  vi.useRealTimers();
});

test("reads nothing while disabled", async () => {
  const { result } = renderHook(() =>
    useReviewRun({ enabled: false, issueName: ISSUE, reloadKey: "1" })
  );
  await flush();
  expect(mocks.getReviewRun).not.toHaveBeenCalled();
  expect(result.current.run).toBeUndefined();
});

test("reports a review that never ran", async () => {
  const { result } = renderHook(() =>
    useReviewRun({ enabled: true, issueName: ISSUE, reloadKey: "1" })
  );
  await flush();
  expect(mocks.getReviewRun).toHaveBeenCalledWith(
    expect.objectContaining({ name: RUN }),
    expect.anything()
  );
  expect(result.current.run).toBeNull();
  expect(result.current.loadFailed).toBe(false);
});

test("polls an active run and reloads the threads when it completes", async () => {
  mocks.getReviewRun
    .mockResolvedValueOnce(reviewRun(ReviewRun_State.RUNNING))
    .mockResolvedValueOnce(reviewRun(ReviewRun_State.SUCCEEDED));
  const { result } = renderHook(() =>
    useReviewRun({ enabled: true, issueName: ISSUE, reloadKey: "1" })
  );
  await flush();
  expect(result.current.run?.state).toBe(ReviewRun_State.RUNNING);
  expect(mocks.fetchIssueCommentThreads).not.toHaveBeenCalled();

  await act(async () => {
    await vi.advanceTimersByTimeAsync(2000);
  });
  expect(result.current.run?.state).toBe(ReviewRun_State.SUCCEEDED);
  expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledWith({
    parent: ISSUE,
  });

  await act(async () => {
    await vi.advanceTimersByTimeAsync(10_000);
  });
  expect(mocks.getReviewRun).toHaveBeenCalledTimes(2);
});

test("shows a rerun that started elsewhere after the run settled", async () => {
  const settled = {
    ...reviewRun(ReviewRun_State.SUCCEEDED),
    createTime: { seconds: 1n, nanos: 0 },
    endTime: { seconds: 2n, nanos: 0 },
  };
  mocks.getReviewRun
    .mockResolvedValueOnce(settled)
    .mockResolvedValueOnce({ ...settled })
    .mockResolvedValueOnce({
      ...reviewRun(ReviewRun_State.RUNNING),
      createTime: { seconds: 30n, nanos: 0 },
    });
  const { result } = renderHook(() =>
    useReviewRun({ enabled: true, issueName: ISSUE, reloadKey: "1" })
  );
  await flush();
  const first = result.current.run;
  expect(first?.state).toBe(ReviewRun_State.SUCCEEDED);

  // A quiet poll keeps the run it had, so the page renders nothing.
  await act(async () => {
    await vi.advanceTimersByTimeAsync(15_000);
  });
  expect(mocks.getReviewRun).toHaveBeenCalledTimes(2);
  expect(result.current.run).toBe(first);
  expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledTimes(1);

  await act(async () => {
    await vi.advanceTimersByTimeAsync(15_000);
  });
  expect(result.current.run?.state).toBe(ReviewRun_State.RUNNING);
});

test("reads the run again when the reload key changes", async () => {
  mocks.getReviewRun.mockResolvedValue(reviewRun(ReviewRun_State.SUCCEEDED));
  const { rerender } = renderHook(
    ({ reloadKey }) =>
      useReviewRun({ enabled: true, issueName: ISSUE, reloadKey }),
    { initialProps: { reloadKey: "1" } }
  );
  await flush();
  expect(mocks.getReviewRun).toHaveBeenCalledTimes(1);

  rerender({ reloadKey: "2" });
  await flush();
  expect(mocks.getReviewRun).toHaveBeenCalledTimes(2);
});

test("reports a failed read", async () => {
  mocks.getReviewRun.mockRejectedValue(
    new ConnectError("boom", Code.Unavailable)
  );
  const { result } = renderHook(() =>
    useReviewRun({ enabled: true, issueName: ISSUE, reloadKey: "1" })
  );
  await flush();
  expect(result.current.loadFailed).toBe(true);
});

const deferred = <T>() => {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((r) => {
    resolve = r;
  });
  return { promise, resolve };
};

test("marks the run stale until the plan's new run is read", async () => {
  const next = deferred<ReturnType<typeof reviewRun>>();
  mocks.getReviewRun
    .mockResolvedValueOnce(reviewRun(ReviewRun_State.SUCCEEDED))
    .mockReturnValueOnce(next.promise);
  const { result, rerender } = renderHook(
    ({ reloadKey }) =>
      useReviewRun({ enabled: true, issueName: ISSUE, reloadKey }),
    { initialProps: { reloadKey: "1" } }
  );
  await flush();
  expect(result.current.stale).toBe(false);

  rerender({ reloadKey: "2" });
  expect(result.current.run?.state).toBe(ReviewRun_State.SUCCEEDED);
  expect(result.current.stale).toBe(true);

  next.resolve(reviewRun(ReviewRun_State.RUNNING));
  await flush();
  expect(result.current.run?.state).toBe(ReviewRun_State.RUNNING);
  expect(result.current.stale).toBe(false);
});

test("keeps the latest read when an earlier one answers last", async () => {
  const earlier = deferred<ReturnType<typeof reviewRun>>();
  mocks.getReviewRun
    .mockReturnValueOnce(earlier.promise)
    .mockResolvedValueOnce(reviewRun(ReviewRun_State.RUNNING));
  const { result } = renderHook(() =>
    useReviewRun({ enabled: true, issueName: ISSUE, reloadKey: "1" })
  );
  await act(async () => {
    void result.current.reload();
    await vi.advanceTimersByTimeAsync(0);
  });
  expect(result.current.run?.state).toBe(ReviewRun_State.RUNNING);

  earlier.resolve(reviewRun(ReviewRun_State.SUCCEEDED));
  await flush();
  expect(result.current.run?.state).toBe(ReviewRun_State.RUNNING);
});

test("keeps a completed run stale until its findings are listed", async () => {
  const listed = deferred<undefined>();
  mocks.fetchIssueCommentThreads.mockReturnValueOnce(listed.promise);
  mocks.getReviewRun
    .mockResolvedValueOnce(reviewRun(ReviewRun_State.RUNNING))
    .mockResolvedValueOnce(reviewRun(ReviewRun_State.SUCCEEDED));
  const { result } = renderHook(() =>
    useReviewRun({ enabled: true, issueName: ISSUE, reloadKey: "1" })
  );
  await flush();
  expect(result.current.stale).toBe(false);

  await act(async () => {
    await vi.advanceTimersByTimeAsync(2000);
  });
  expect(result.current.run?.state).toBe(ReviewRun_State.SUCCEEDED);
  expect(result.current.stale).toBe(true);

  listed.resolve(undefined);
  await flush();
  expect(result.current.stale).toBe(false);
});

test("keeps a completed run stale when its findings fail to list", async () => {
  mocks.fetchIssueCommentThreads.mockRejectedValueOnce(new Error("boom"));
  mocks.getReviewRun
    .mockResolvedValueOnce(reviewRun(ReviewRun_State.RUNNING))
    .mockResolvedValueOnce(reviewRun(ReviewRun_State.SUCCEEDED));
  const { result } = renderHook(() =>
    useReviewRun({ enabled: true, issueName: ISSUE, reloadKey: "1" })
  );
  await flush();
  await act(async () => {
    await vi.advanceTimersByTimeAsync(2000);
  });
  expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledTimes(1);
  expect(result.current.stale).toBe(true);
});

test("lists the findings of a run first read as succeeded", async () => {
  const listed = deferred<undefined>();
  mocks.fetchIssueCommentThreads.mockReturnValueOnce(listed.promise);
  mocks.getReviewRun.mockResolvedValue(reviewRun(ReviewRun_State.SUCCEEDED));
  const { result, rerender } = renderHook(
    ({ reloadKey }) =>
      useReviewRun({ enabled: true, issueName: ISSUE, reloadKey }),
    { initialProps: { reloadKey: "1" } }
  );
  await flush();
  expect(result.current.run?.state).toBe(ReviewRun_State.SUCCEEDED);
  expect(result.current.stale).toBe(true);
  expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledTimes(1);

  listed.resolve(undefined);
  await flush();
  expect(result.current.stale).toBe(false);

  // Reading the same execution again lists nothing.
  rerender({ reloadKey: "2" });
  await flush();
  expect(mocks.getReviewRun).toHaveBeenCalledTimes(2);
  expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledTimes(1);
  expect(result.current.stale).toBe(false);
});

test("keeps an outdated run stale and polls it until its replacement starts", async () => {
  mocks.getReviewRun
    .mockResolvedValueOnce({
      ...reviewRun(ReviewRun_State.SUCCEEDED),
      outdated: true,
    })
    .mockResolvedValueOnce(reviewRun(ReviewRun_State.RUNNING));
  const { result } = renderHook(() =>
    useReviewRun({ enabled: true, issueName: ISSUE, reloadKey: "1" })
  );
  await flush();
  expect(result.current.run?.state).toBe(ReviewRun_State.SUCCEEDED);
  expect(result.current.stale).toBe(true);

  await act(async () => {
    await vi.advanceTimersByTimeAsync(2000);
  });
  expect(mocks.getReviewRun).toHaveBeenCalledTimes(2);
  expect(result.current.run?.state).toBe(ReviewRun_State.RUNNING);
  expect(result.current.stale).toBe(false);
});
