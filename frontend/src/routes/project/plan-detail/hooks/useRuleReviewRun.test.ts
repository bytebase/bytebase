import { Code, ConnectError } from "@connectrpc/connect";
import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import {
  IssueComment_ThreadState,
  ReviewRun_Status,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";

const ISSUE = "projects/foo/issues/7";
const RULE_RUN = `${ISSUE}/reviewRuns/rule`;

const mocks = vi.hoisted(() => ({
  comments: [] as unknown[],
  fetchIssueCommentThreads: vi.fn(),
  getReviewRun: vi.fn(),
}));

vi.mock("@/api", () => ({
  issueServiceClientConnect: { getReviewRun: mocks.getReviewRun },
}));

vi.mock("@/stores/app", () => {
  const state = {
    fetchIssueCommentThreads: mocks.fetchIssueCommentThreads,
    getIssueComments: () => mocks.comments,
  };
  return {
    useAppStore: Object.assign(
      (selector: (s: typeof state) => unknown) => selector(state),
      { getState: () => state }
    ),
  };
});

import { useRuleReviewRun } from "./useRuleReviewRun";

const reviewRun = (status: ReviewRun_Status) => ({
  name: RULE_RUN,
  type: ReviewRun_Type.RULE,
  status,
  error: "",
});

const finding = (
  runType: ReviewRun_Type,
  threadState: IssueComment_ThreadState
) => ({ reviewMetadata: { runType }, threadState });

const flush = () =>
  act(async () => {
    await vi.advanceTimersByTimeAsync(0);
  });

beforeEach(() => {
  vi.useFakeTimers();
  vi.clearAllMocks();
  mocks.getReviewRun.mockReset();
  mocks.comments = [];
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
    useRuleReviewRun({ enabled: false, issueName: ISSUE, reloadKey: "1" })
  );
  await flush();
  expect(mocks.getReviewRun).not.toHaveBeenCalled();
  expect(result.current.run).toBeUndefined();
});

test("reports a review that never ran", async () => {
  const { result } = renderHook(() =>
    useRuleReviewRun({ enabled: true, issueName: ISSUE, reloadKey: "1" })
  );
  await flush();
  expect(mocks.getReviewRun).toHaveBeenCalledWith(
    expect.objectContaining({ name: RULE_RUN }),
    expect.anything()
  );
  expect(result.current.run).toBeNull();
  expect(result.current.loadFailed).toBe(false);
});

test("polls an active run and reloads the threads when it completes", async () => {
  mocks.comments = [
    finding(ReviewRun_Type.RULE, IssueComment_ThreadState.OPEN),
    finding(ReviewRun_Type.RULE, IssueComment_ThreadState.OPEN),
    finding(ReviewRun_Type.RULE, IssueComment_ThreadState.RESOLVED),
    finding(ReviewRun_Type.AI, IssueComment_ThreadState.OPEN),
  ];
  mocks.getReviewRun
    .mockResolvedValueOnce(reviewRun(ReviewRun_Status.RUNNING))
    .mockResolvedValueOnce(reviewRun(ReviewRun_Status.DONE));
  const { result } = renderHook(() =>
    useRuleReviewRun({ enabled: true, issueName: ISSUE, reloadKey: "1" })
  );
  await flush();
  expect(result.current.run?.status).toBe(ReviewRun_Status.RUNNING);
  expect(mocks.fetchIssueCommentThreads).not.toHaveBeenCalled();

  await act(async () => {
    await vi.advanceTimersByTimeAsync(2000);
  });
  expect(result.current.run?.status).toBe(ReviewRun_Status.DONE);
  expect(result.current.openFindings).toBe(2);
  expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledWith({
    parent: ISSUE,
  });

  await act(async () => {
    await vi.advanceTimersByTimeAsync(10_000);
  });
  expect(mocks.getReviewRun).toHaveBeenCalledTimes(2);
});

test("reads the run again when the reload key changes", async () => {
  mocks.getReviewRun.mockResolvedValue(reviewRun(ReviewRun_Status.DONE));
  const { rerender } = renderHook(
    ({ reloadKey }) =>
      useRuleReviewRun({ enabled: true, issueName: ISSUE, reloadKey }),
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
    useRuleReviewRun({ enabled: true, issueName: ISSUE, reloadKey: "1" })
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
    .mockResolvedValueOnce(reviewRun(ReviewRun_Status.DONE))
    .mockReturnValueOnce(next.promise);
  const { result, rerender } = renderHook(
    ({ reloadKey }) =>
      useRuleReviewRun({ enabled: true, issueName: ISSUE, reloadKey }),
    { initialProps: { reloadKey: "1" } }
  );
  await flush();
  expect(result.current.stale).toBe(false);

  rerender({ reloadKey: "2" });
  expect(result.current.run?.status).toBe(ReviewRun_Status.DONE);
  expect(result.current.stale).toBe(true);

  next.resolve(reviewRun(ReviewRun_Status.AVAILABLE));
  await flush();
  expect(result.current.run?.status).toBe(ReviewRun_Status.AVAILABLE);
  expect(result.current.stale).toBe(false);
});

test("keeps the latest read when an earlier one answers last", async () => {
  const earlier = deferred<ReturnType<typeof reviewRun>>();
  mocks.getReviewRun
    .mockReturnValueOnce(earlier.promise)
    .mockResolvedValueOnce(reviewRun(ReviewRun_Status.AVAILABLE));
  const { result } = renderHook(() =>
    useRuleReviewRun({ enabled: true, issueName: ISSUE, reloadKey: "1" })
  );
  await act(async () => {
    void result.current.reload();
    await vi.advanceTimersByTimeAsync(0);
  });
  expect(result.current.run?.status).toBe(ReviewRun_Status.AVAILABLE);

  earlier.resolve(reviewRun(ReviewRun_Status.DONE));
  await flush();
  expect(result.current.run?.status).toBe(ReviewRun_Status.AVAILABLE);
});

test("keeps a completed run stale until its findings are listed", async () => {
  const listed = deferred<undefined>();
  mocks.fetchIssueCommentThreads.mockReturnValueOnce(listed.promise);
  mocks.getReviewRun
    .mockResolvedValueOnce(reviewRun(ReviewRun_Status.RUNNING))
    .mockResolvedValueOnce(reviewRun(ReviewRun_Status.DONE));
  const { result } = renderHook(() =>
    useRuleReviewRun({ enabled: true, issueName: ISSUE, reloadKey: "1" })
  );
  await flush();
  expect(result.current.stale).toBe(false);

  await act(async () => {
    await vi.advanceTimersByTimeAsync(2000);
  });
  expect(result.current.run?.status).toBe(ReviewRun_Status.DONE);
  expect(result.current.stale).toBe(true);

  listed.resolve(undefined);
  await flush();
  expect(result.current.stale).toBe(false);
});

test("keeps a completed run stale when its findings fail to list", async () => {
  mocks.fetchIssueCommentThreads.mockRejectedValueOnce(new Error("boom"));
  mocks.getReviewRun
    .mockResolvedValueOnce(reviewRun(ReviewRun_Status.RUNNING))
    .mockResolvedValueOnce(reviewRun(ReviewRun_Status.DONE));
  const { result } = renderHook(() =>
    useRuleReviewRun({ enabled: true, issueName: ISSUE, reloadKey: "1" })
  );
  await flush();
  await act(async () => {
    await vi.advanceTimersByTimeAsync(2000);
  });
  expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledTimes(1);
  expect(result.current.stale).toBe(true);
});
