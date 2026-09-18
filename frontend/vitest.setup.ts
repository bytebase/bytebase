import "@testing-library/jest-dom/vitest";
import { afterEach, vi } from "vitest";

// App-wide guardrail against React bugs that only ever surface as a console
// warning, which the default reporter hides — so a broken component would
// otherwise pass its own tests. Every signature below is ALWAYS a real bug:
//   - "getSnapshot should be cached" — a `useSyncExternalStore` whose
//     getSnapshot returns a fresh value every call (this crashed the SQL
//     Editor: GutterBar → useVueRoute → "Maximum update depth").
//   - "Maximum update depth exceeded" — setState/store-write on every render.
//   - "cannot be a descendant of" / "cannot be a child of" — invalid HTML
//     nesting, e.g. a <div> inside a <p>, or a <tr> outside a <tbody>. The
//     parser closes the ancestor early, so the rendered tree is not the one the
//     component describes and hydration mismatches. React emits four such
//     messages; the second string is a substring of the three that name a
//     direct-parent rule, including both text-node cases.
// Any test that renders a component triggering these now fails, so the whole
// existing render-test suite doubles as detection across the app. React dedupes
// the nesting warning per module, and vitest isolates modules per file, so a
// file reports its FIRST invalid nesting and not the rest — fix one and re-run
// rather than assuming the file is clean.
const FORBIDDEN_WARNING_SIGNATURES = [
  "getSnapshot should be cached",
  "Maximum update depth exceeded",
  "cannot be a descendant of",
  "cannot be a child of",
];
let forbiddenWarnings: string[] = [];
const realConsoleError = console.error.bind(console);
console.error = (...args: unknown[]) => {
  const message = args.map((arg) => String(arg)).join(" ");
  if (FORBIDDEN_WARNING_SIGNATURES.some((sig) => message.includes(sig))) {
    forbiddenWarnings.push(message);
  }
  realConsoleError(...args);
};
afterEach(() => {
  if (forbiddenWarnings.length > 0) {
    const captured = forbiddenWarnings;
    forbiddenWarnings = [];
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
