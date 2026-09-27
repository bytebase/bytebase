import { act, renderHook } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import {
  getLocalSheetByName,
  getLocalSheetsVersion,
  setLocalSheetStatement,
  subscribeLocalSheets,
  useLocalSheetsVersion,
} from "./localSheet";

describe("localSheet subscription", () => {
  test("setLocalSheetStatement bumps the version and notifies subscribers", () => {
    const before = getLocalSheetsVersion();
    const listener = vi.fn();
    const unsubscribe = subscribeLocalSheets(listener);

    setLocalSheetStatement(getLocalSheetByName("local/1"), "select 1");
    expect(getLocalSheetsVersion()).toBe(before + 1);
    expect(listener).toHaveBeenCalledTimes(1);

    unsubscribe();
    setLocalSheetStatement(getLocalSheetByName("local/1"), "select 2");
    // No longer notified after unsubscribe.
    expect(listener).toHaveBeenCalledTimes(1);
  });

  test("useLocalSheetsVersion re-renders a consumer on a local sheet edit", () => {
    const { result } = renderHook(() => useLocalSheetsVersion());
    const before = result.current;

    act(() => setLocalSheetStatement(getLocalSheetByName("local/2"), "x"));
    expect(result.current).toBe(before + 1);
  });
});
