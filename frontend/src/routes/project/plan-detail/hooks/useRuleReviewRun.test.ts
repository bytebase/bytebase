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
