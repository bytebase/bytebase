import { cleanup, render } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

vi.mock("@/app/router", () => ({
  router: { push: vi.fn(), resolve: () => ({ href: "/projects/shop/plans/674" }) },
}));

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => ({
    bypassLeaveGuardOnce: vi.fn(),
    isEditing: false,
    planId: "674",
    projectId: "shop",
  }),
}));

import { ActivityScope, CommentLinkProvider } from "../shell/CommentLinkContext";
import { useActivityLanding } from "./useActivityLanding";

const NAME = "projects/shop/issues/701/issueComments/c1";
const OTHER = "projects/shop/issues/701/issueComments/c2";

function Entry({ names }: { names: string[] }) {
  const { linked, ref } = useActivityLanding<HTMLDivElement>(names);
  return <div data-linked={linked || undefined} data-testid="entry" ref={ref} />;
}

const scrollIntoView = vi.fn();
const onLanded = vi.fn();
// The scroll pane is 800px tall and its sticky title row covers the top 56px.
const PANE = { top: 0, bottom: 800 };
const HEADER = { top: 0, bottom: 56 };
let rect = { top: 0, bottom: 0 };

beforeEach(() => {
  vi.clearAllMocks();
  rect = { top: 0, bottom: 0 };
  HTMLElement.prototype.scrollIntoView = scrollIntoView;
  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(
    function (this: HTMLElement) {
      if (this.hasAttribute("data-scroll-restoration-id")) {
        return PANE as DOMRect;
      }
      if (this.hasAttribute("data-plan-sticky-header")) {
        return HEADER as DOMRect;
      }
      return rect as DOMRect;
    }
  );
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  Reflect.deleteProperty(HTMLElement.prototype, "scrollIntoView");
});

const inActivity = (focusName: string | undefined, children: ReactNode) => (
  <div data-scroll-restoration-id="main">
    <header data-plan-sticky-header />
    <ActivityScope
      focus={focusName ? { commentName: focusName, nonce: 7 } : undefined}
      onLanded={onLanded}
    >
      {children}
    </ActivityScope>
  </div>
);

describe("useActivityLanding", () => {
  test("an entry off screen is jumped to, focused, and the request is taken", () => {
    rect = { top: 2000, bottom: 2100 };
    const { getByTestId } = render(inActivity(NAME, <Entry names={[NAME]} />));
    const entry = getByTestId("entry");
    expect(scrollIntoView.mock.contexts).toEqual([entry]);
    // No behavior is passed: the page jumps, it does not glide.
    expect(scrollIntoView).toHaveBeenCalledWith({ block: "center" });
    expect(document.activeElement).toBe(entry);
    expect(onLanded).toHaveBeenCalledWith(7);
  });

  test("an entry already in view is focused where it is", () => {
    rect = { top: 200, bottom: 300 };
    const { getByTestId } = render(inActivity(NAME, <Entry names={[NAME]} />));
    expect(scrollIntoView).not.toHaveBeenCalled();
    expect(document.activeElement).toBe(getByTestId("entry"));
    expect(onLanded).toHaveBeenCalledWith(7);
  });

  test("an entry whose first row is under the sticky title row counts as off screen", () => {
    rect = { top: 40, bottom: 140 };
    render(inActivity(NAME, <Entry names={[NAME]} />));
    expect(scrollIntoView).toHaveBeenCalledTimes(1);
  });

  test("an entry just below the sticky title row stays where it is", () => {
    rect = { top: 58, bottom: 158 };
    render(inActivity(NAME, <Entry names={[NAME]} />));
    expect(scrollIntoView).not.toHaveBeenCalled();
  });

  test("a long entry whose first row shows stays where it is, though it runs off the bottom", () => {
    rect = { top: 600, bottom: 1400 };
    render(inActivity(NAME, <Entry names={[NAME]} />));
    expect(scrollIntoView).not.toHaveBeenCalled();
  });

  test("an entry whose first row is cut by the bottom edge is jumped to", () => {
    rect = { top: 790, bottom: 890 };
    render(inActivity(NAME, <Entry names={[NAME]} />));
    expect(scrollIntoView).toHaveBeenCalledTimes(1);
  });

  test("a row that stands for several comments takes the landing of each", () => {
    render(inActivity(OTHER, <Entry names={[NAME, OTHER]} />));
    expect(onLanded).toHaveBeenCalledWith(7);
  });

  test("a request for another comment is left alone", () => {
    render(inActivity(OTHER, <Entry names={[NAME]} />));
    expect(scrollIntoView).not.toHaveBeenCalled();
    expect(onLanded).not.toHaveBeenCalled();
  });

  test("the address marks its comment in Activity and nowhere else", () => {
    const inside = render(
      <CommentLinkProvider linkedCommentName={NAME}>
        {inActivity(undefined, <Entry names={[NAME]} />)}
      </CommentLinkProvider>
    );
    expect(inside.getByTestId("entry").hasAttribute("data-linked")).toBe(true);
    inside.unmount();
    // The statement editor draws the same comment outside the timeline.
    const outside = render(
      <CommentLinkProvider linkedCommentName={NAME}>
        <Entry names={[NAME]} />
      </CommentLinkProvider>
    );
    expect(outside.getByTestId("entry").hasAttribute("data-linked")).toBe(false);
  });
});
