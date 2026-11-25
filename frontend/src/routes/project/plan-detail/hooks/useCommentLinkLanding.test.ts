import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  comments: {} as Record<string, { name: string }[]>,
  expandPhase: vi.fn(),
  fetchThreads: vi.fn(async (_request: { parent: string }) => []),
  requestActivityFocus: vi.fn(),
}));

vi.mock("@/stores/app", () => {
  const state = () => ({
    fetchIssueCommentThreads: mocks.fetchThreads,
    issueCommentsByIssue: mocks.comments,
  });
  return {
    useAppStore: Object.assign(
      (selector: (state: unknown) => unknown) => selector(state()),
      { getState: state }
    ),
  };
});

vi.mock("../shared/stores/usePlanDetailStore", () => ({
  usePlanDetailStoreApi: () => ({
    getState: () => ({ requestActivityFocus: mocks.requestActivityFocus }),
  }),
}));

import { useCommentLinkLanding } from "./useCommentLinkLanding";

const ISSUE = "projects/shop/issues/701";
const NAME = `${ISSUE}/issueComments/c1`;

const land = (
  routeHash: string | undefined,
  issueName: string | null = ISSUE
) =>
  renderHook(
    (props: { routeHash: string | undefined }) =>
      useCommentLinkLanding({
        expandPhase: mocks.expandPhase,
        issueName: issueName ?? undefined,
        routeHash: props.routeHash,
      }),
    { initialProps: { routeHash } }
  );

beforeEach(() => {
  vi.clearAllMocks();
  mocks.comments = {};
});

describe("useCommentLinkLanding", () => {
  test("waits for the issue's comments, then expands Review and asks for the comment", () => {
    const { result, rerender } = land("#issueComments/c1");
    expect(result.current).toBeUndefined();
    expect(mocks.requestActivityFocus).not.toHaveBeenCalled();
    mocks.comments = { [ISSUE]: [{ name: NAME }] };
    rerender({ routeHash: "#issueComments/c1" });
    expect(mocks.expandPhase).toHaveBeenCalledWith("review");
    expect(mocks.requestActivityFocus).toHaveBeenCalledWith(NAME);
    expect(result.current).toBe(NAME);
  });

  test("a refresh of the comments does not move the page again", () => {
    mocks.comments = { [ISSUE]: [{ name: NAME }] };
    const { rerender } = land("#issueComments/c1");
    mocks.comments = {
      [ISSUE]: [{ name: NAME }, { name: `${ISSUE}/issueComments/c2` }],
    };
    rerender({ routeHash: "#issueComments/c1" });
    expect(mocks.requestActivityFocus).toHaveBeenCalledTimes(1);
  });

  test("another address lands again", () => {
    mocks.comments = {
      [ISSUE]: [{ name: NAME }, { name: `${ISSUE}/issueComments/c2` }],
    };
    const { rerender } = land("#issueComments/c1");
    rerender({ routeHash: "#issueComments/c2" });
    expect(mocks.requestActivityFocus).toHaveBeenLastCalledWith(
      `${ISSUE}/issueComments/c2`
    );
    expect(mocks.requestActivityFocus).toHaveBeenCalledTimes(2);
  });

  test("an address that names no comment of this issue opens the plan as it is", async () => {
    mocks.comments = { [ISSUE]: [{ name: NAME }] };
    const { result, rerender } = land("#issueComments/gone");
    // The comments are listed once more, in case the comment is newer.
    expect(mocks.fetchThreads).toHaveBeenCalledWith({ parent: ISSUE });
    await act(async () => {});
    rerender({ routeHash: "#issueComments/gone" });
    expect(mocks.fetchThreads).toHaveBeenCalledTimes(1);
    expect(result.current).toBeUndefined();
    expect(mocks.expandPhase).not.toHaveBeenCalled();
    expect(mocks.requestActivityFocus).not.toHaveBeenCalled();
  });

  test("a comment newer than the loaded ones is found by listing them once more", async () => {
    const newer = `${ISSUE}/issueComments/new`;
    mocks.comments = { [ISSUE]: [{ name: NAME }] };
    mocks.fetchThreads.mockImplementationOnce(async () => {
      mocks.comments = { [ISSUE]: [{ name: NAME }, { name: newer }] };
      return [];
    });
    const { result, rerender } = land("#issueComments/new");
    expect(mocks.requestActivityFocus).not.toHaveBeenCalled();
    await act(async () => {});
    rerender({ routeHash: "#issueComments/new" });
    expect(mocks.requestActivityFocus).toHaveBeenCalledWith(newer);
    expect(mocks.requestActivityFocus).toHaveBeenCalledTimes(1);
    expect(result.current).toBe(newer);
  });

  test("a comment that turns up only after the plan was opened as it is does not move the page", async () => {
    const late = `${ISSUE}/issueComments/late`;
    mocks.comments = { [ISSUE]: [{ name: NAME }] };
    const { rerender } = land("#issueComments/late");
    await act(async () => {});
    mocks.comments = { [ISSUE]: [{ name: NAME }, { name: late }] };
    rerender({ routeHash: "#issueComments/late" });
    expect(mocks.requestActivityFocus).not.toHaveBeenCalled();
  });

  test.each([
    ["no fragment", undefined, ISSUE],
    ["another fragment", "#plan-phase-review", ISSUE],
    ["a plan without an issue", "#issueComments/c1", null],
  ])("%s lands nowhere", (_, routeHash, issueName) => {
    mocks.comments = { [ISSUE]: [{ name: NAME }] };
    const { result } = land(routeHash, issueName);
    expect(result.current).toBeUndefined();
    expect(mocks.requestActivityFocus).not.toHaveBeenCalled();
  });
});
