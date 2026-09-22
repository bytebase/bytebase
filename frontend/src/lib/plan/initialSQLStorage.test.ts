import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { consumeInitialSQL, saveInitialSQL } from "./initialSQLStorage";

beforeEach(() => {
  localStorage.clear();
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("initial SQL storage", () => {
  test("removes a consumed value from durable storage", () => {
    const key = saveInitialSQL("SELECT 1");

    expect(consumeInitialSQL<string>(key)).toBe("SELECT 1");
    expect(localStorage.getItem(key)).toBeNull();
  });

  test("replays a consumed value for StrictMode initialization", () => {
    const key = saveInitialSQL("SELECT 1");

    expect(consumeInitialSQL<string>(key)).toBe("SELECT 1");
    expect(consumeInitialSQL<string>(key)).toBe("SELECT 1");
  });

  test("removes abandoned values after one day", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-29T00:00:00Z"));
    const abandonedKey = saveInitialSQL({ "databases/one": "ALTER TABLE t" });
    const consumedKey = saveInitialSQL("SELECT 1");
    consumeInitialSQL(consumedKey);

    vi.setSystemTime(new Date("2026-08-30T00:00:00.001Z"));
    saveInitialSQL("SELECT 1");

    expect(localStorage.getItem(abandonedKey)).toBeNull();
    expect(consumeInitialSQL(consumedKey)).toBeUndefined();
  });

  test("keeps only the most recent abandoned values", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-29T00:00:00Z"));
    const keys = Array.from({ length: 25 }, (_, index) => {
      vi.setSystemTime(new Date(Date.UTC(2026, 7, 29, 0, 0, index)));
      return saveInitialSQL(`SELECT ${index}`);
    });

    const storedKeys = Array.from({ length: localStorage.length }, (_, index) =>
      localStorage.key(index)
    ).filter((key) => key?.startsWith("bb.plan.initial-sql."));
    expect(storedKeys).toHaveLength(20);
    expect(localStorage.getItem(keys[0])).toBeNull();
    expect(localStorage.getItem(keys.at(-1)!)).not.toBeNull();
  });

  test("reports local storage failures to the caller", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new DOMException("Storage quota exceeded", "QuotaExceededError");
    });

    expect(() => saveInitialSQL("SELECT 1")).toThrow("Storage quota exceeded");
  });
});
