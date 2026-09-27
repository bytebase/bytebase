import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test } from "vitest";
import { useMediaQuery } from "./useMediaQuery";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

type MediaListener = () => void;

// Controllable `matchMedia` mock: each query keeps a live `matches` flag and a
// set of change listeners that the returned setter can flip to emulate a
// viewport crossing the breakpoint.
function installMatchMedia(initialMatches: (query: string) => boolean) {
  const registry = new Map<
    string,
    { matches: boolean; listeners: Set<MediaListener> }
  >();
  const entryFor = (query: string) => {
    let entry = registry.get(query);
    if (!entry) {
      entry = { matches: initialMatches(query), listeners: new Set() };
      registry.set(query, entry);
    }
    return entry;
  };
  window.matchMedia = ((query: string) => {
    const entry = entryFor(query);
    return {
      get matches() {
        return entry.matches;
      },
      media: query,
      onchange: null,
      addEventListener: (_: string, cb: MediaListener) =>
        entry.listeners.add(cb),
      removeEventListener: (_: string, cb: MediaListener) =>
        entry.listeners.delete(cb),
      addListener: (cb: MediaListener) => entry.listeners.add(cb),
      removeListener: (cb: MediaListener) => entry.listeners.delete(cb),
      dispatchEvent: () => true,
    } as unknown as MediaQueryList;
  }) as unknown as typeof window.matchMedia;
  return (query: string, matches: boolean) => {
    const entry = entryFor(query);
    entry.matches = matches;
    entry.listeners.forEach((cb) => cb());
  };
}

describe("useMediaQuery", () => {
  let container: HTMLDivElement;
  let root: Root;
  const realMatchMedia = window.matchMedia;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    window.matchMedia = realMatchMedia;
  });

  test("reflects the initial match state of the query", () => {
    let value: boolean | undefined;
    installMatchMedia(() => true);
    function Probe() {
      value = useMediaQuery("(min-width: 1024px)");
      return null;
    }
    act(() => root.render(<Probe />));
    expect(value).toBe(true);
  });

  test("passes the query through to matchMedia", () => {
    let seenQuery = "";
    installMatchMedia((query) => {
      seenQuery = query;
      return false;
    });
    function Probe() {
      useMediaQuery("(max-width: 639px)");
      return null;
    }
    act(() => root.render(<Probe />));
    expect(seenQuery).toBe("(max-width: 639px)");
  });

  test("re-renders when the query flips", () => {
    let value: boolean | undefined;
    const query = "(max-width: 639px)";
    const setMatches = installMatchMedia(() => false);
    function Probe() {
      value = useMediaQuery(query);
      return null;
    }
    act(() => root.render(<Probe />));
    expect(value).toBe(false);

    act(() => setMatches(query, true));
    expect(value).toBe(true);
  });
});
