import { act, renderHook } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { usePaginatedSelect } from "./usePaginatedSelect";

describe("usePaginatedSelect", () => {
  test("resets on search and appends the next page", async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({
        items: [{ name: "one" }],
        nextPageToken: "page-2",
      })
      .mockResolvedValueOnce({
        items: [{ name: "two" }],
        nextPageToken: "",
      })
      .mockResolvedValueOnce({
        items: [{ name: "search" }],
        nextPageToken: "search-page-2",
      });
    const { result } = renderHook(() => usePaginatedSelect({ fetchPage }));

    await act(() => result.current.search(""));
    expect(result.current.items.map((item) => item.name)).toEqual(["one"]);
    expect(result.current.hasMore).toBe(true);

    await act(() => result.current.loadMore());
    expect(fetchPage).toHaveBeenLastCalledWith("", "page-2");
    expect(result.current.items.map((item) => item.name)).toEqual([
      "one",
      "two",
    ]);

    await act(() => result.current.search("search"));
    expect(fetchPage).toHaveBeenLastCalledWith("search", "");
    expect(result.current.items.map((item) => item.name)).toEqual(["search"]);
    expect(result.current.hasMore).toBe(true);
  });

  test("ignores stale first-page responses", async () => {
    let resolveOld: (value: {
      items: { name: string }[];
      nextPageToken: string;
    }) => void = () => {};
    const fetchPage = vi
      .fn()
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveOld = resolve;
          })
      )
      .mockResolvedValueOnce({
        items: [{ name: "new" }],
        nextPageToken: "",
      });
    const { result } = renderHook(() => usePaginatedSelect({ fetchPage }));

    let oldRequest: Promise<void>;
    act(() => {
      oldRequest = result.current.search("old");
    });
    await act(() => result.current.search("new"));
    await act(async () => {
      resolveOld({ items: [{ name: "old" }], nextPageToken: "" });
      await oldRequest;
    });

    expect(result.current.items.map((item) => item.name)).toEqual(["new"]);
  });
});
