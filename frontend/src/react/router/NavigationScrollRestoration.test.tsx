import { act, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  createMemoryRouter,
  Outlet,
  RouterProvider,
} from "react-router-dom";
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

function TestShell() {
  return (
    <NavigationScrollRestoration>
      <div data-scroll-restoration-id={MAIN_SCROLL_RESTORATION_ID}>
        <div data-scroll-restoration-id="panel">
          <Outlet />
        </div>
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
}

function recordScroll(element: HTMLElement, top: number, left = 0): void {
  element.scrollTop = top;
  element.scrollLeft = left;
  element.dispatchEvent(new Event("scroll"));
}

async function renderRouter(firstElement = <div>First</div>) {
  const router = createMemoryRouter(
    [
      {
        path: "/",
        element: <TestShell />,
        children: [
          { path: "first", element: firstElement },
          { path: "second", element: <div>Second</div> },
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
});
