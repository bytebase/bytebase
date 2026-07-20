import { afterEach, describe, expect, test, vi } from "vitest";
import {
  clearPagedDataCache,
  readPagedDataCache,
  writePagedDataCache,
} from "./pagedDataCache";

describe("pagedDataCache", () => {
  afterEach(() => {
    clearPagedDataCache();
    vi.useRealTimers();
  });

  test("returns an isolated copy of a cached view", () => {
    writePagedDataCache("issues", {
      dataList: [{ name: "issues/1" }],
      hasMore: true,
      nextPageToken: "page-2",
    });

    const first = readPagedDataCache<{ name: string }>("issues");
    first?.dataList.push({ name: "issues/local-only" });

    expect(readPagedDataCache("issues")).toEqual({
      dataList: [{ name: "issues/1" }],
      hasMore: true,
      nextPageToken: "page-2",
    });
  });

  test("expires five minutes after write even when reads refresh LRU order", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-07-20T00:00:00Z"));
    writePagedDataCache("issues", {
      dataList: [{ name: "issues/1" }],
      hasMore: false,
      nextPageToken: "",
    });

    vi.advanceTimersByTime(4 * 60 * 1000);
    expect(readPagedDataCache("issues")).toBeDefined();

    vi.advanceTimersByTime(60 * 1000 + 1);

    expect(readPagedDataCache("issues")).toBeUndefined();
  });

  test("evicts the least recently used view after twenty entries", () => {
    for (let i = 0; i < 20; i++) {
      writePagedDataCache(`issues-${i}`, {
        dataList: [{ name: `issues/${i}` }],
        hasMore: false,
        nextPageToken: "",
      });
    }
    expect(readPagedDataCache("issues-0")).toBeDefined();

    writePagedDataCache("issues-20", {
      dataList: [{ name: "issues/20" }],
      hasMore: false,
      nextPageToken: "",
    });

    expect(readPagedDataCache("issues-1")).toBeUndefined();
    expect(readPagedDataCache("issues-0")).toBeDefined();
  });
});
