import "@testing-library/jest-dom/vitest";
import { vi } from "vitest";

// Vitest 4.1's `populateGlobal` does not copy jsdom's `localStorage` /
// `sessionStorage` onto the test global because Node 22+ already defines
// them as experimental globals (which resolve to `undefined` unless
// `--localstorage-file` is set). Pull them off the original jsdom Window
// (exposed as `globalThis.jsdom`) so bare `localStorage` / `sessionStorage`
// access in tests works.
{
  const jsdomInstance = (globalThis as { jsdom?: { window: Window } }).jsdom;
  if (jsdomInstance?.window) {
    const jsdomWindow = jsdomInstance.window;
    for (const key of ["localStorage", "sessionStorage"] as const) {
      Object.defineProperty(globalThis, key, {
        configurable: true,
        get: () => jsdomWindow[key],
      });
    }
  }
}

vi.mock("pouchdb", () => {
  class MockPouchDB {
    static plugin = vi.fn();

    get = vi.fn(async () => undefined);
    put = vi.fn(async () => undefined);
    remove = vi.fn(async () => undefined);
  }

  return {
    default: MockPouchDB,
  };
});

vi.mock("pouchdb-find", () => ({
  default: {},
}));
