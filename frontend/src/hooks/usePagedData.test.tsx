import { act, StrictMode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  clearPagedDataCache,
  invalidatePagedDataCacheScope,
  readPagedDataCache,
  writePagedDataCache,
} from "./pagedDataCache";
import { PagedTableFooter, usePagedData } from "./usePagedData";

const pageSizeState = vi.hoisted(() => ({ value: 50 }));

vi.mock("@/hooks/useSessionPageSize", () => ({
  getPageSizeOptions: () => [50, 100],
  useSessionPageSize: () => [pageSizeState.value, vi.fn()],
}));

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

type Item = {
  name: string;
};

type FetchList = (params: {
  pageSize: number;
  pageToken: string;
}) => Promise<{ list: Item[]; nextPageToken?: string }>;

function Harness({ fetchList }: { fetchList: FetchList }) {
  const paged = usePagedData<Item>({
    sessionKey: "test-use-paged-data",
    fetchList,
  });

  if (paged.isLoading) {
    return <div data-state="loading">loading</div>;
  }
  if (paged.dataList.length === 0) {
    return <div data-state="empty">empty</div>;
  }
  return <div data-state="data">{paged.dataList[0].name}</div>;
}

function CacheHarness({
  cacheKey,
  cacheScope,
  cacheRestoreToken,
  fetchList,
}: {
  cacheKey: string;
  cacheScope?: string;
  cacheRestoreToken?: string;
  fetchList: FetchList;
}) {
  const paged = usePagedData<Item>({
    sessionKey: "test-cached-paged-data",
    cacheKey,
    cacheScope,
    cacheRestoreToken,
    fetchList,
  });

  return (
    <div>
      <div data-state={paged.isLoading ? "loading" : "ready"}>
        {paged.dataList.map((item) => item.name).join(",")}
      </div>
      <button type="button" onClick={paged.loadMore}>
        load more
      </button>
    </div>
  );
}

describe("usePagedData", () => {
  let root: Root | undefined;
  let container: HTMLDivElement | undefined;

  afterEach(() => {
    pageSizeState.value = 50;
    vi.useRealTimers();
    sessionStorage.clear();
    clearPagedDataCache();
    act(() => {
      root?.unmount();
    });
    root = undefined;
    container = undefined;
    document.body.innerHTML = "";
  });

  test("keeps loading while a debounced refresh is pending", async () => {
    vi.useFakeTimers();
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    const emptyFetch = vi.fn<FetchList>(async () => ({ list: [] }));
    const dataFetch = vi.fn<FetchList>(async () => ({
      list: [{ name: "items/1" }],
    }));

    await act(async () => {
      root!.render(<Harness fetchList={emptyFetch} />);
      await Promise.resolve();
    });

    expect(container.querySelector("[data-state]")?.textContent).toBe("empty");

    await act(async () => {
      root!.render(<Harness fetchList={dataFetch} />);
      await Promise.resolve();
    });

    expect(container.querySelector("[data-state]")?.textContent).toBe(
      "loading"
    );
    expect(dataFetch).not.toHaveBeenCalled();

    await act(async () => {
      vi.advanceTimersByTime(300);
      await Promise.resolve();
    });

    expect(container.querySelector("[data-state]")?.textContent).toBe(
      "items/1"
    );
  });

  test("restores every loaded page and cached token for history POP", async () => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    const initialFetch = vi
      .fn<FetchList>()
      .mockResolvedValueOnce({
        list: [{ name: "items/1" }],
        nextPageToken: "page-2",
      })
      .mockResolvedValueOnce({
        list: [{ name: "items/2" }],
        nextPageToken: "page-3",
      });

    await act(async () => {
      root!.render(
        <CacheHarness cacheKey="issues" fetchList={initialFetch} />
      );
      await Promise.resolve();
    });
    await act(async () => {
      container!.querySelector("button")?.click();
      await Promise.resolve();
    });
    expect(container.querySelector("[data-state]")?.textContent).toBe(
      "items/1,items/2"
    );

    act(() => root?.unmount());
    root = createRoot(container);
    const resumedFetch = vi.fn<FetchList>(async () => ({
      list: [{ name: "items/3" }],
    }));

    act(() => {
      root!.render(
        <CacheHarness
          cacheKey="issues"
          cacheRestoreToken="entry-1"
          fetchList={resumedFetch}
        />
      );
    });
    expect(
      container.querySelector("[data-state]")?.getAttribute("data-state")
    ).toBe("ready");
    expect(container.querySelector("[data-state]")?.textContent).toBe(
      "items/1,items/2"
    );
    expect(resumedFetch).not.toHaveBeenCalled();

    await act(async () => {
      container!.querySelector("button")?.click();
      await Promise.resolve();
    });
    expect(resumedFetch).toHaveBeenCalledWith({
      pageSize: 50,
      pageToken: "page-3",
    });
  });

  test("does not refresh a hydrated history view during Strict Mode effect replay", async () => {
    vi.useFakeTimers();
    writePagedDataCache("issues:page-size=50", {
      dataList: [{ name: "items/cached" }],
      hasMore: false,
      nextPageToken: "",
    });
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    const fetchList = vi.fn<FetchList>(async () => ({
      list: [{ name: "items/refreshed" }],
    }));

    await act(async () => {
      root!.render(
        <StrictMode>
          <CacheHarness
            cacheKey="issues"
            cacheRestoreToken="entry-1"
            fetchList={fetchList}
          />
        </StrictMode>
      );
      await Promise.resolve();
    });

    expect(container.querySelector("[data-state]")?.textContent).toBe(
      "items/cached"
    );
    expect(fetchList).not.toHaveBeenCalled();

    await act(async () => {
      vi.advanceTimersByTime(300);
      await Promise.resolve();
    });

    expect(fetchList).not.toHaveBeenCalled();
    expect(container.querySelector("[data-state]")?.textContent).toBe(
      "items/cached"
    );
  });

  test("fetches current data for a push-style remount", async () => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    const initialFetch = vi.fn<FetchList>(async () => ({
      list: [{ name: "items/cached" }],
    }));
    await act(async () => {
      root!.render(
        <CacheHarness cacheKey="issues" fetchList={initialFetch} />
      );
      await Promise.resolve();
    });

    act(() => root?.unmount());
    root = createRoot(container);
    const currentFetch = vi.fn<FetchList>(async () => ({
      list: [{ name: "items/current" }],
    }));
    await act(async () => {
      root!.render(
        <CacheHarness cacheKey="issues" fetchList={currentFetch} />
      );
      await Promise.resolve();
    });

    expect(currentFetch).toHaveBeenCalledOnce();
    expect(container.querySelector("[data-state]")?.textContent).toBe(
      "items/current"
    );
  });

  test("persists the cache scope used for later invalidation", async () => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    const fetchList = vi.fn<FetchList>(async () => ({
      list: [{ name: "items/1" }],
    }));

    await act(async () => {
      root!.render(
        <CacheHarness
          cacheKey="issues"
          cacheScope="project-a-issues"
          fetchList={fetchList}
        />
      );
      await Promise.resolve();
    });
    expect(readPagedDataCache("issues:page-size=50")).toBeDefined();

    invalidatePagedDataCacheScope("project-a-issues");

    expect(readPagedDataCache("issues:page-size=50")).toBeUndefined();
  });

  test("restores an entry again after the mounted page visits another view", async () => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    const fetchList = vi.fn<FetchList>(async () => ({
      list: [{ name: "items/entry-1" }],
    }));

    await act(async () => {
      root!.render(
        <CacheHarness
          cacheKey="issues"
          cacheRestoreToken="entry-1"
          fetchList={fetchList}
        />
      );
      await Promise.resolve();
    });
    expect(container.querySelector("[data-state]")?.textContent).toBe(
      "items/entry-1"
    );

    act(() => {
      root!.render(<CacheHarness cacheKey="plans" fetchList={fetchList} />);
    });
    act(() => {
      root!.render(
        <CacheHarness
          cacheKey="issues"
          cacheRestoreToken="entry-1"
          fetchList={fetchList}
        />
      );
    });

    expect(container.querySelector("[data-state]")?.textContent).toBe(
      "items/entry-1"
    );
    expect(fetchList).toHaveBeenCalledOnce();
  });

  test("consumes cache restoration once per history entry", async () => {
    vi.useFakeTimers();
    writePagedDataCache("issues:page-size=50", {
      dataList: [{ name: "items/cached-50" }],
      hasMore: false,
      nextPageToken: "",
    });
    writePagedDataCache("issues:page-size=100", {
      dataList: [{ name: "items/cached-100" }],
      hasMore: false,
      nextPageToken: "",
    });
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    const fetchList = vi.fn<FetchList>(async () => ({
      list: [{ name: "items/current-100" }],
    }));

    act(() => {
      root!.render(
        <CacheHarness
          cacheKey="issues"
          cacheRestoreToken="entry-1"
          fetchList={fetchList}
        />
      );
    });
    expect(container.querySelector("[data-state]")?.textContent).toBe(
      "items/cached-50"
    );

    pageSizeState.value = 100;
    await act(async () => {
      root!.render(
        <CacheHarness
          cacheKey="issues"
          cacheRestoreToken="entry-1"
          fetchList={fetchList}
        />
      );
      await Promise.resolve();
    });
    expect(container.querySelector("[data-state]")?.textContent).not.toBe(
      "items/cached-100"
    );

    await act(async () => {
      vi.advanceTimersByTime(300);
      await Promise.resolve();
    });
    expect(fetchList).toHaveBeenCalledWith({ pageSize: 100, pageToken: "" });
    expect(container.querySelector("[data-state]")?.textContent).toBe(
      "items/current-100"
    );
  });

  test("does not extend absolute expiry when a history view is hydrated", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-07-20T00:00:00Z"));
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    const initialFetch = vi.fn<FetchList>(async () => ({
      list: [{ name: "items/cached" }],
    }));
    await act(async () => {
      root!.render(
        <CacheHarness cacheKey="issues" fetchList={initialFetch} />
      );
      await Promise.resolve();
    });
    act(() => root?.unmount());

    act(() => vi.advanceTimersByTime(4 * 60 * 1000));
    root = createRoot(container);
    const warmFetch = vi.fn<FetchList>(async () => ({ list: [] }));
    act(() => {
      root!.render(
        <CacheHarness
          cacheKey="issues"
          cacheRestoreToken="entry-1"
          fetchList={warmFetch}
        />
      );
    });
    expect(warmFetch).not.toHaveBeenCalled();
    expect(container.querySelector("[data-state]")?.textContent).toBe(
      "items/cached"
    );
    act(() => root?.unmount());

    act(() => vi.advanceTimersByTime(60 * 1000 + 1));
    root = createRoot(container);
    const expiredFetch = vi.fn<FetchList>(async () => ({
      list: [{ name: "items/current" }],
    }));
    await act(async () => {
      root!.render(
        <CacheHarness
          cacheKey="issues"
          cacheRestoreToken="entry-1"
          fetchList={expiredFetch}
        />
      );
      await Promise.resolve();
    });

    expect(expiredFetch).toHaveBeenCalledOnce();
    expect(container.querySelector("[data-state]")?.textContent).toBe(
      "items/current"
    );
  });
});

describe("PagedTableFooter", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("uses the shared select trigger for page size", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <PagedTableFooter
          pageSize={10}
          pageSizeOptions={[10, 20]}
          onPageSizeChange={vi.fn()}
          hasMore={false}
          isFetchingMore={false}
          onLoadMore={vi.fn()}
        />
      );
    });

    expect(container.querySelector("select")).toBeNull();
    expect(container.querySelector("button")?.className).toContain(
      "cursor-pointer"
    );

    await act(async () => {
      root.unmount();
    });
  });
});
