import { act, useLayoutEffect, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import { createMemoryRouter, Outlet } from "react-router";
import { RouterProvider } from "react-router/dom";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { setAppRouter } from "./navigation";
import {
  MAIN_SCROLL_RESTORATION_ID,
  NavigationScrollRestoration,
  useScrollRestorationLoadMore,
} from "./NavigationScrollRestoration";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

function TestShell({ routeInsidePanel = true }: { routeInsidePanel?: boolean }) {
  return (
    <NavigationScrollRestoration>
      <div data-scroll-restoration-id={MAIN_SCROLL_RESTORATION_ID}>
        <div data-scroll-restoration-id="panel">
          {routeInsidePanel ? <Outlet /> : null}
        </div>
        {routeInsidePanel ? null : <Outlet />}
      </div>
    </NavigationScrollRestoration>
  );
}

function makeScrollable(element: HTMLElement): void {
  Object.defineProperties(element, {
    clientHeight: { configurable: true, value: 200 },
    clientWidth: { configurable: true, value: 200 },
    scrollHeight: { configurable: true, value: 1000 },
    scrollWidth: { configurable: true, value: 1000 },
  });
  element.getBoundingClientRect = () => ({
    x: 0,
    y: 0,
    top: 0,
    bottom: 200,
    left: 0,
    right: 200,
    width: 200,
    height: 200,
    toJSON: () => ({}),
  });
}

function recordScroll(element: HTMLElement, top: number, left = 0): void {
  element.scrollTop = top;
  element.scrollLeft = left;
  element.dispatchEvent(new Event("scroll"));
}

function RouteScrollMetrics({
  scrollHeight,
  resetScrollTop = false,
}: {
  scrollHeight: number;
  resetScrollTop?: boolean;
}) {
  useLayoutEffect(() => {
    const main = document.querySelector<HTMLElement>(
      `[data-scroll-restoration-id='${MAIN_SCROLL_RESTORATION_ID}']`
    );
    if (!main) throw new Error("Missing main scroll target");
    Object.defineProperty(main, "scrollHeight", {
      configurable: true,
      value: scrollHeight,
    });
    if (resetScrollTop) main.scrollTop = 0;
  }, [resetScrollTop, scrollHeight]);

  return null;
}

function AnchorPage({
  anchorTop,
  scrollHeight,
}: {
  anchorTop: () => number;
  scrollHeight?: () => number;
}) {
  const anchorRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const anchor = anchorRef.current;
    const main = anchor?.closest<HTMLElement>(
      `[data-scroll-restoration-id='${MAIN_SCROLL_RESTORATION_ID}']`
    );
    if (!anchor || !main) throw new Error("Missing anchor test elements");
    if (scrollHeight) {
      Object.defineProperty(main, "scrollHeight", {
        configurable: true,
        value: scrollHeight(),
      });
    }

    anchor.getBoundingClientRect = () => {
      const top = anchorTop() - main.scrollTop;
      return {
        x: 0,
        y: top,
        top,
        bottom: top + 40,
        left: 0,
        right: 200,
        width: 200,
        height: 40,
        toJSON: () => ({}),
      };
    };
  }, [anchorTop, scrollHeight]);

  return (
    <div
      ref={anchorRef}
      data-scroll-restoration-anchor="issues/anchor"
    >
      Anchor
    </div>
  );
}

async function renderRouter(
  firstElement = <div>First</div>,
  secondElement = <div>Second</div>,
  { routeInsidePanel = true }: { routeInsidePanel?: boolean } = {}
) {
  const router = createMemoryRouter(
    [
      {
        path: "/",
        element: <TestShell routeInsidePanel={routeInsidePanel} />,
        children: [
          { path: "first", element: firstElement },
          { path: "second", element: secondElement },
        ],
      },
    ],
    {
      initialEntries: [{ pathname: "/first", key: "first-entry" }],
    }
  );
  setAppRouter(router);

  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  await act(async () => {
    root.render(<RouterProvider router={router} />);
  });

  const main = container.querySelector<HTMLElement>(
    `[data-scroll-restoration-id='${MAIN_SCROLL_RESTORATION_ID}']`
  );
  const panel = container.querySelector<HTMLElement>(
    "[data-scroll-restoration-id='panel']"
  );
  if (!main || !panel) throw new Error("Missing test scroll targets");
  makeScrollable(main);
  makeScrollable(panel);

  return {
    main,
    panel,
    router,
    unmount: () => {
      act(() => {
        root.unmount();
        container.remove();
      });
      router.dispose();
    },
  };
}

beforeEach(() => {
  sessionStorage.clear();
  vi.spyOn(window, "scrollTo").mockImplementation(() => {});
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("NavigationScrollRestoration", () => {
  test("restores every scroll target on Back and Forward", async () => {
    const { main, panel, router, unmount } = await renderRouter();

    recordScroll(main, 240, 30);
    recordScroll(panel, 120, 10);

    await act(async () => {
      await router.navigate("/second");
    });
    expect(main.scrollTop).toBe(0);
    expect(panel.scrollTop).toBe(0);

    recordScroll(main, 360, 40);
    recordScroll(panel, 180, 20);

    await act(async () => {
      await router.navigate(-1);
    });
    expect(main.scrollTop).toBe(240);
    expect(main.scrollLeft).toBe(30);
    expect(panel.scrollTop).toBe(120);
    expect(panel.scrollLeft).toBe(10);

    await act(async () => {
      await router.navigate(1);
    });
    expect(main.scrollTop).toBe(360);
    expect(main.scrollLeft).toBe(40);
    expect(panel.scrollTop).toBe(180);
    expect(panel.scrollLeft).toBe(20);

    unmount();
  });

  test("snapshots the outgoing entry without waiting for a scroll event", async () => {
    const { main, router, unmount } = await renderRouter();

    main.scrollTop = 240;
    await act(async () => {
      await router.navigate("/second");
    });
    await act(async () => {
      await router.navigate(-1);
    });

    expect(main.scrollTop).toBe(240);
    unmount();
  });

  test("does not overwrite the outgoing position after destination layout commits", async () => {
    const { main, router, unmount } = await renderRouter(
      <RouteScrollMetrics scrollHeight={1000} />,
      <RouteScrollMetrics scrollHeight={200} resetScrollTop />
    );

    main.scrollTop = 640;
    await act(async () => {
      await router.navigate("/second");
    });
    expect(main.scrollTop).toBe(0);

    await act(async () => {
      await router.navigate(-1);
    });

    expect(main.scrollTop).toBe(640);
    unmount();
  });

  test("restores after repeated Push and Back navigations", async () => {
    const { main, router, unmount } = await renderRouter();

    recordScroll(main, 240);
    for (let i = 0; i < 3; i++) {
      await act(async () => {
        await router.navigate("/second");
      });
      await act(async () => {
        await router.navigate(-1);
      });
      expect(main.scrollTop).toBe(240);
    }

    unmount();
  });

  test("preserves the inherited position in a new history entry", async () => {
    const { main, router, unmount } = await renderRouter();

    recordScroll(main, 240);
    await act(async () => {
      await router.navigate("/first?tab=activity", {
        preventScrollReset: true,
        replace: true,
      });
    });
    expect(main.scrollTop).toBe(240);

    await act(async () => {
      await router.navigate("/second");
    });
    await act(async () => {
      await router.navigate(-1);
    });

    expect(router.state.location.search).toBe("?tab=activity");
    expect(main.scrollTop).toBe(240);
    unmount();
  });

  test("asks paged content to grow until the saved position is reachable", async () => {
    const onLoadMore = vi.fn();
    const PagedPage = () => {
      useScrollRestorationLoadMore({
        hasMore: true,
        isFetchingMore: false,
        isLoading: false,
        dataList: [],
        loadMore: onLoadMore,
      });
      return <div>First</div>;
    };
    const { main, router, unmount } = await renderRouter(<PagedPage />);

    recordScroll(main, 700);
    await act(async () => {
      await router.navigate("/second");
    });
    Object.defineProperty(main, "scrollHeight", {
      configurable: true,
      value: 300,
    });

    await act(async () => {
      await router.navigate(-1);
    });

    expect(onLoadMore).toHaveBeenCalledOnce();
    unmount();
  });

  test("stops growing paged content after the user cancels restoration", async () => {
    const onLoadMore = vi.fn();
    let appendItem: (() => void) | undefined;
    const PagedPage = () => {
      const [dataList, setDataList] = useState<unknown[]>([]);
      appendItem = () => setDataList((list) => [...list, {}]);
      useScrollRestorationLoadMore({
        hasMore: true,
        isFetchingMore: false,
        isLoading: false,
        dataList,
        loadMore: onLoadMore,
      });
      return <div>First</div>;
    };
    const { main, router, unmount } = await renderRouter(<PagedPage />);

    recordScroll(main, 700);
    await act(async () => {
      await router.navigate("/second");
    });
    Object.defineProperty(main, "scrollHeight", {
      configurable: true,
      value: 300,
    });
    await act(async () => {
      await router.navigate(-1);
    });
    expect(onLoadMore).toHaveBeenCalledOnce();

    await act(async () => {
      document.dispatchEvent(new Event("pointerdown"));
    });
    act(() => appendItem?.());

    expect(onLoadMore).toHaveBeenCalledOnce();
    unmount();
  });

  test("keeps growing while a page request exceeds the idle timeout", async () => {
    vi.useFakeTimers();
    const onLoadMore = vi.fn();
    let finishPage: (() => void) | undefined;
    const PagedPage = () => {
      const [dataList, setDataList] = useState<unknown[]>([]);
      const [isFetchingMore, setIsFetchingMore] = useState(false);
      finishPage = () => {
        setDataList((list) => [...list, {}]);
        setIsFetchingMore(false);
      };
      useScrollRestorationLoadMore({
        hasMore: true,
        isFetchingMore,
        isLoading: false,
        dataList,
        loadMore: () => {
          onLoadMore();
          setIsFetchingMore(true);
        },
      });
      return <div>First</div>;
    };
    const { main, router, unmount } = await renderRouter(<PagedPage />);

    recordScroll(main, 700);
    await act(async () => {
      await router.navigate("/second");
    });
    Object.defineProperty(main, "scrollHeight", {
      configurable: true,
      value: 300,
    });
    await act(async () => {
      await router.navigate(-1);
    });
    expect(onLoadMore).toHaveBeenCalledOnce();

    await act(async () => {
      vi.advanceTimersByTime(30001);
    });
    act(() => finishPage?.());

    expect(onLoadMore).toHaveBeenCalledTimes(2);
    unmount();
  });

  test("keeps the same semantic anchor offset when row geometry changes", async () => {
    let anchorTop = 400;
    const { main, router, unmount } = await renderRouter(
      <AnchorPage anchorTop={() => anchorTop} />,
      <div>Second</div>,
      { routeInsidePanel: false }
    );

    recordScroll(main, 380);
    await act(async () => {
      await router.navigate("/second");
    });

    anchorTop = 460;
    await act(async () => {
      await router.navigate(-1);
    });

    expect(main.scrollTop).toBe(440);
    unmount();
  });

  test("uses the semantic anchor when the old coordinate is no longer reachable", async () => {
    let anchorTop = 620;
    let scrollHeight = 1000;
    const { main, router, unmount } = await renderRouter(
      <AnchorPage
        anchorTop={() => anchorTop}
        scrollHeight={() => scrollHeight}
      />,
      <div>Second</div>,
      { routeInsidePanel: false }
    );

    recordScroll(main, 600);
    await act(async () => {
      await router.navigate("/second");
    });

    anchorTop = 350;
    scrollHeight = 550;
    await act(async () => {
      await router.navigate(-1);
    });

    expect(main.scrollTop).toBe(330);
    unmount();
  });

  test("grows paged content when a saved anchor is not loaded yet", async () => {
    const onLoadMore = vi.fn();
    let showAnchor = true;
    const PagedAnchorPage = () => {
      useScrollRestorationLoadMore({
        hasMore: true,
        isFetchingMore: false,
        isLoading: false,
        dataList: [{}],
        loadMore: onLoadMore,
      });
      return showAnchor ? (
        <AnchorPage anchorTop={() => 320} />
      ) : (
        <div>First page</div>
      );
    };
    const { main, router, unmount } = await renderRouter(
      <PagedAnchorPage />,
      <div>Second</div>,
      { routeInsidePanel: false }
    );

    recordScroll(main, 300);
    await act(async () => {
      await router.navigate("/second");
    });

    showAnchor = false;
    await act(async () => {
      await router.navigate(-1);
    });

    const restoredScrollTop = main.scrollTop;
    unmount();

    expect(restoredScrollTop).toBe(300);
    expect(onLoadMore).toHaveBeenCalledOnce();
  });

  test("keeps restoring while an anchor correction is temporarily clamped", async () => {
    vi.useFakeTimers();
    let anchorTop = 320;
    let scrollHeight = 1000;
    const { main, router, unmount } = await renderRouter(
      <AnchorPage
        anchorTop={() => anchorTop}
        scrollHeight={() => scrollHeight}
      />,
      <div>Second</div>,
      { routeInsidePanel: false }
    );
    let scrollTop = 0;
    let maxScrollTop = 800;
    Object.defineProperty(main, "scrollTop", {
      configurable: true,
      get: () => scrollTop,
      set: (value: number) => {
        scrollTop = Math.max(0, Math.min(value, maxScrollTop));
      },
    });

    recordScroll(main, 300);
    await act(async () => {
      await router.navigate("/second");
    });

    anchorTop = 500;
    scrollHeight = 650;
    maxScrollTop = 450;
    await act(async () => {
      await router.navigate(-1);
    });
    expect(main.scrollTop).toBe(450);

    Object.defineProperty(main, "scrollHeight", {
      configurable: true,
      value: 800,
    });
    maxScrollTop = 600;
    act(() => vi.advanceTimersByTime(100));

    expect(main.scrollTop).toBe(480);
    unmount();
  });

  test("grows paged content while an anchor correction is clamped", async () => {
    vi.useFakeTimers();
    const onLoadMore = vi.fn();
    let finishPage: (() => void) | undefined;
    let anchorTop = 320;
    let scrollHeight = 1000;
    const PagedAnchorPage = () => {
      const [dataList, setDataList] = useState<unknown[]>([]);
      const [hasMore, setHasMore] = useState(true);
      const [isFetchingMore, setIsFetchingMore] = useState(false);
      finishPage = () => {
        setDataList([{}]);
        setHasMore(false);
        setIsFetchingMore(false);
      };
      useScrollRestorationLoadMore({
        hasMore,
        isFetchingMore,
        isLoading: false,
        dataList,
        loadMore: () => {
          onLoadMore();
          setIsFetchingMore(true);
        },
      });
      return (
        <AnchorPage
          anchorTop={() => anchorTop}
          scrollHeight={() => scrollHeight}
        />
      );
    };
    const { main, router, unmount } = await renderRouter(
      <PagedAnchorPage />,
      <div>Second</div>,
      { routeInsidePanel: false }
    );
    let scrollTop = 0;
    let maxScrollTop = 800;
    Object.defineProperty(main, "scrollTop", {
      configurable: true,
      get: () => scrollTop,
      set: (value: number) => {
        scrollTop = Math.max(0, Math.min(value, maxScrollTop));
      },
    });

    recordScroll(main, 300);
    await act(async () => {
      await router.navigate("/second");
    });

    anchorTop = 500;
    scrollHeight = 650;
    maxScrollTop = 450;
    await act(async () => {
      await router.navigate(-1);
    });

    expect(main.scrollTop).toBe(450);
    expect(onLoadMore).toHaveBeenCalledOnce();

    Object.defineProperty(main, "scrollHeight", {
      configurable: true,
      value: 800,
    });
    maxScrollTop = 600;
    act(() => finishPage?.());

    expect(main.scrollTop).toBe(480);
    expect(vi.getTimerCount()).toBe(0);
    unmount();
  });

  test("accepts the best clamped anchor when paged content cannot grow", async () => {
    vi.useFakeTimers();
    let anchorTop = 320;
    let scrollHeight = 1000;
    const PagedAnchorPage = () => {
      useScrollRestorationLoadMore({
        hasMore: false,
        isFetchingMore: false,
        isLoading: false,
        dataList: [],
        loadMore: vi.fn(),
      });
      return (
        <AnchorPage
          anchorTop={() => anchorTop}
          scrollHeight={() => scrollHeight}
        />
      );
    };
    const { main, router, unmount } = await renderRouter(
      <PagedAnchorPage />,
      <div>Second</div>,
      { routeInsidePanel: false }
    );
    let scrollTop = 0;
    let maxScrollTop = 800;
    Object.defineProperty(main, "scrollTop", {
      configurable: true,
      get: () => scrollTop,
      set: (value: number) => {
        scrollTop = Math.max(0, Math.min(value, maxScrollTop));
      },
    });

    recordScroll(main, 300);
    await act(async () => {
      await router.navigate("/second");
    });

    anchorTop = 500;
    scrollHeight = 650;
    maxScrollTop = 450;
    await act(async () => {
      await router.navigate(-1);
    });
    expect(main.scrollTop).toBe(450);
    expect(vi.getTimerCount()).toBe(0);

    recordScroll(main, 200);
    await act(async () => {
      await router.navigate("/second");
      await router.navigate(-1);
    });

    expect(main.scrollTop).toBe(200);
    unmount();
  });
});
