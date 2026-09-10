import "@testing-library/jest-dom/vitest";
import { afterEach, vi } from "vitest";

// App-wide guardrail against React bugs that only ever surface as a console
// warning, which the default reporter hides — so a broken component would
// otherwise pass its own tests. Every signature below is ALWAYS a real bug:
//   - "getSnapshot should be cached" — a `useSyncExternalStore` whose
//     getSnapshot returns a fresh value every call (this crashed the SQL
//     Editor: GutterBar → useVueRoute → "Maximum update depth").
//   - "Maximum update depth exceeded" — setState/store-write on every render.
//   - "cannot be a descendant of" — invalid HTML nesting, e.g. a <div> inside a
//     <p>. The browser's parser closes the <p> early, so the rendered tree is
//     not the one the component describes and hydration mismatches. This one
//     shipped once already, in a divider that swapped a <span> for a component
//     rendering a <div>, and the suite stayed green.
// Any test that renders a component triggering these now fails, so the whole
// existing render-test suite doubles as detection across the app.
const FORBIDDEN_WARNING_SIGNATURES = [
  "getSnapshot should be cached",
  "Maximum update depth exceeded",
  "cannot be a descendant of",
];
let renderLoopWarnings: string[] = [];
const realConsoleError = console.error.bind(console);
console.error = (...args: unknown[]) => {
  const message = args.map((arg) => String(arg)).join(" ");
  if (FORBIDDEN_WARNING_SIGNATURES.some((sig) => message.includes(sig))) {
    renderLoopWarnings.push(message);
  }
  realConsoleError(...args);
};
afterEach(() => {
  if (renderLoopWarnings.length > 0) {
    const captured = renderLoopWarnings;
    renderLoopWarnings = [];
    throw new Error(
      `React reported a defect this test would otherwise have passed through. ` +
        `A useSyncExternalStore getSnapshot must return a cached/stable value, ` +
        `effects must not setState unconditionally every render, and element ` +
        `nesting must be valid HTML:\n${captured.join("\n")}`
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
