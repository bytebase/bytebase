import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { useLocalStorageBoolean } from "./useLocalStorageBoolean";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const KEY = "bb.test.preference";

// The hook's value and setter, captured from a render so each case can drive it
// the way a component would.
const mount = (defaultValue: boolean) => {
  const seen: { value: boolean; set: (next: boolean) => void }[] = [];
  function Probe() {
    const [value, set] = useLocalStorageBoolean(KEY, defaultValue);
    seen.push({ value, set });
    return null;
  }
  const container = document.createElement("div");
  const root = createRoot(container);
  act(() => root.render(<Probe />));
  return {
    latest: () => seen[seen.length - 1],
    unmount: () => act(() => root.unmount()),
  };
};

beforeEach(() => {
  localStorage.clear();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useLocalStorageBoolean", () => {
  test.each([
    ["true", true],
    ["false", false],
  ])("honors the stored %s", (raw, expected) => {
    localStorage.setItem(KEY, raw);
    const { latest, unmount } = mount(!expected);
    expect(latest().value).toBe(expected);
    unmount();
  });

  // A value another build wrote, or a half-written entry. Coercing it would let
  // any garbage pin the preference off.
  test.each(["", "1", "TRUE", "{}", "null"])(
    "falls back to the default for %o",
    (raw) => {
      localStorage.setItem(KEY, raw);
      const { latest, unmount } = mount(true);
      expect(latest().value).toBe(true);
      unmount();
    }
  );

  test("writes the new value back", () => {
    const { latest, unmount } = mount(false);
    act(() => latest().set(true));
    expect(latest().value).toBe(true);
    expect(localStorage.getItem(KEY)).toBe("true");
    unmount();
  });

  test("does not write on mount, so a default is not a preference", () => {
    const { unmount } = mount(true);
    expect(localStorage.getItem(KEY)).toBeNull();
    unmount();
  });

  // A locked-down profile throws on both accessors, and a preference is never
  // worth failing a render over.
  test("survives storage that throws on read", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("denied");
    });
    const { latest, unmount } = mount(true);
    expect(latest().value).toBe(true);
    unmount();
  });

  test("survives storage that throws on write", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("denied");
    });
    const { latest, unmount } = mount(false);
    act(() => latest().set(true));
    expect(latest().value).toBe(true);
    unmount();
  });
});
