import { create } from "@bufbuild/protobuf";
import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";

const mocks = vi.hoisted(() => ({
  fetchIssueCommentThreads: vi.fn(async () => undefined),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      fetchIssueCommentThreads: mocks.fetchIssueCommentThreads,
    }),
  },
}));

import { useIssueCommentThreadsSync } from "./useIssueCommentThreadsSync";

const issue = (seconds: number) =>
  create(IssueSchema, {
    name: "projects/p/issues/1",
    updateTime: { seconds: BigInt(seconds), nanos: 0 },
  });

beforeEach(() => {
  vi.useFakeTimers();
  mocks.fetchIssueCommentThreads.mockClear();
});
afterEach(() => {
  vi.restoreAllMocks();
  vi.useRealTimers();
});

describe("useIssueCommentThreadsSync", () => {
  test("loads on mount, when the issue's update time moves, and on a visible cadence", () => {
    const { rerender } = renderHook(
      (props) => useIssueCommentThreadsSync(props),
      {
        initialProps: issue(1),
      }
    );
    expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledTimes(1);
    expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledWith({
      parent: "projects/p/issues/1",
    });

    // A poll that leaves updateTime alone (comment writes never move it)
    // does not refetch by itself.
    rerender(issue(1));
    expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledTimes(1);
    rerender(issue(2));
    expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledTimes(2);

    vi.advanceTimersByTime(15_000);
    expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledTimes(3);

    const hidden = vi.spyOn(document, "hidden", "get").mockReturnValue(true);
    vi.advanceTimersByTime(15_000);
    expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledTimes(3);
    hidden.mockReturnValue(false);
    document.dispatchEvent(new Event("visibilitychange"));
    expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledTimes(4);
  });

  test("does nothing without an issue and stops after unmount", () => {
    const { rerender, unmount } = renderHook(
      (props) => useIssueCommentThreadsSync(props),
      {
        initialProps: undefined as ReturnType<typeof issue> | undefined,
      }
    );
    expect(mocks.fetchIssueCommentThreads).not.toHaveBeenCalled();
    rerender(issue(1));
    expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledTimes(1);
    unmount();
    vi.advanceTimersByTime(60_000);
    expect(mocks.fetchIssueCommentThreads).toHaveBeenCalledTimes(1);
  });
});
