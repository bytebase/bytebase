import { renderHook } from "@testing-library/react";
import type { MouseEvent, ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  bypassLeaveGuardOnce: vi.fn(),
  isEditing: false,
  push: vi.fn(),
}));

vi.mock("@/app/router", () => ({
  router: {
    push: mocks.push,
    resolve: ({ params }: { params: { planId: string; projectId: string } }) => ({
      href: `/projects/${params.projectId}/plans/${params.planId}`,
    }),
  },
}));

vi.mock("./PlanDetailContext", () => ({
  usePlanDetailContext: () => ({
    bypassLeaveGuardOnce: mocks.bypassLeaveGuardOnce,
    isEditing: mocks.isEditing,
    planId: "674",
    projectId: "shop",
  }),
}));

import {
  CommentLinkProvider,
  useCommentLinks,
  useCommentTimestampLink,
} from "./CommentLinkContext";

const NAME = "projects/shop/issues/701/issueComments/c1";

const wrapper = ({ children }: { children: ReactNode }) => (
  <CommentLinkProvider linkedCommentName={undefined}>
    {children}
  </CommentLinkProvider>
);

const click = (init: Partial<MouseEvent<HTMLAnchorElement>> = {}) =>
  ({
    altKey: false,
    button: 0,
    ctrlKey: false,
    defaultPrevented: false,
    metaKey: false,
    preventDefault: vi.fn(),
    shiftKey: false,
    ...init,
  }) as unknown as MouseEvent<HTMLAnchorElement>;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.isEditing = false;
  window.history.replaceState({}, "", "/projects/shop/plans/674/specs/s1?x=1");
});

describe("comment links", () => {
  test("the href is the fragment alone, so the reader's path is kept", () => {
    const { result } = renderHook(() => useCommentLinks(), { wrapper });
    expect(result.current?.hrefOf(NAME)).toBe("#issueComments/c1");
  });

  test("the copied address is on the plan's root path, whatever is selected", () => {
    const { result } = renderHook(() => useCommentLinks(), { wrapper });
    expect(result.current?.urlOf(NAME)).toBe(
      `${window.location.origin}/projects/shop/plans/674#issueComments/c1`
    );
  });

  test("a plain click changes only the fragment and leaves the scroll alone", () => {
    const { result } = renderHook(() => useCommentLinks(), { wrapper });
    const event = click();
    result.current?.follow(event, NAME);
    expect(event.preventDefault).toHaveBeenCalled();
    expect(mocks.push).toHaveBeenCalledWith(
      { fullPath: "/projects/shop/plans/674/specs/s1?x=1#issueComments/c1" },
      { preventScrollReset: true }
    );
    expect(mocks.bypassLeaveGuardOnce).not.toHaveBeenCalled();
  });

  test.each([
    ["a meta click", { metaKey: true }],
    ["a control click", { ctrlKey: true }],
    ["a shift click", { shiftKey: true }],
    ["a middle click", { button: 1 }],
  ])("%s is left to the browser", (_, init) => {
    const { result } = renderHook(() => useCommentLinks(), { wrapper });
    const event = click(init);
    result.current?.follow(event, NAME);
    expect(event.preventDefault).not.toHaveBeenCalled();
    expect(mocks.push).not.toHaveBeenCalled();
  });

  test("a click on the comment the address already names adds no history", () => {
    window.history.replaceState({}, "", "/projects/shop/plans/674#issueComments/c1");
    const { result } = renderHook(() => useCommentLinks(), { wrapper });
    const event = click();
    result.current?.follow(event, NAME);
    expect(event.preventDefault).toHaveBeenCalled();
    expect(mocks.push).not.toHaveBeenCalled();
  });

  test("an unsaved edit does not hold a change of fragment", () => {
    mocks.isEditing = true;
    const { result } = renderHook(() => useCommentLinks(), { wrapper });
    result.current?.follow(click(), NAME);
    expect(mocks.bypassLeaveGuardOnce).toHaveBeenCalledTimes(1);
    expect(mocks.push).toHaveBeenCalledTimes(1);
  });

  test("a timestamp has no link outside a plan page, or without a comment", () => {
    expect(
      renderHook(() => useCommentTimestampLink(NAME)).result.current
    ).toBeUndefined();
    expect(
      renderHook(() => useCommentTimestampLink(undefined), { wrapper }).result
        .current
    ).toBeUndefined();
    expect(
      renderHook(() => useCommentTimestampLink(NAME), { wrapper }).result
        .current?.href
    ).toBe("#issueComments/c1");
  });
});
