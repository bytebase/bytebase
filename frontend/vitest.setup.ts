import "@testing-library/jest-dom/vitest";
import { afterEach, vi } from "vitest";

// App-wide guardrail against React infinite-render-loop bugs. Both signatures
// below are ALWAYS real bugs, never benign:
//   - "getSnapshot should be cached" — a `useSyncExternalStore` whose
//     getSnapshot returns a fresh value every call (this crashed the SQL
//     Editor: GutterBar → useVueRoute → "Maximum update depth"). It is only a
//     console warning, so a looping component would otherwise pass its tests.
//   - "Maximum update depth exceeded" — setState/store-write on every render.
// Any test that renders a component triggering these now fails, so the whole
// existing render-test suite doubles as loop detection across the app.
const RENDER_LOOP_SIGNATURES = [
  "getSnapshot should be cached",
  "Maximum update depth exceeded",
];
let renderLoopWarnings: string[] = [];
const realConsoleError = console.error.bind(console);
console.error = (...args: unknown[]) => {
  const message = args.map((arg) => String(arg)).join(" ");
  if (RENDER_LOOP_SIGNATURES.some((sig) => message.includes(sig))) {
    renderLoopWarnings.push(message);
  }
  realConsoleError(...args);
};
afterEach(() => {
  if (renderLoopWarnings.length > 0) {
    const captured = renderLoopWarnings;
    renderLoopWarnings = [];
    throw new Error(
      `React infinite-render-loop detected during this test. A useSyncExternalStore ` +
        `getSnapshot must return a cached/stable value, and effects must not ` +
        `setState unconditionally every render:\n${captured.join("\n")}`
    );
  }
});

// jsdom does not implement ResizeObserver. Components that subscribe to size
// changes (e.g. EllipsisText) blow up on mount without this shim.
if (typeof globalThis.ResizeObserver === "undefined") {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver;
}

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
