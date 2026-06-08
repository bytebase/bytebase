import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { PagedTableFooter, usePagedData } from "./usePagedData";

vi.mock("@/react/hooks/useSessionPageSize", () => ({
  getPageSizeOptions: () => [50, 100],
  useSessionPageSize: () => [50, vi.fn()],
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

describe("usePagedData", () => {
  let root: Root | undefined;
  let container: HTMLDivElement | undefined;

  afterEach(() => {
    vi.useRealTimers();
    sessionStorage.clear();
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
