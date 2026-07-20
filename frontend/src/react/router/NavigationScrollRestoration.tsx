import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { type Location, useLocation, useNavigationType } from "react-router";
import type { PagedDataResult } from "@/react/hooks/usePagedData";
import { getAppRouterState, subscribeRoute } from "@/react/router/navigation";
import { minmax } from "@/utils/math";

const STORAGE_KEY = "bb.navigation-scroll-restoration";
const MAX_SAVED_ENTRIES = 50;
const RESTORE_IDLE_TIMEOUT_MS = 30000;
const RESTORE_MAX_TIMEOUT_MS = 5 * 60 * 1000;
const WINDOW_TARGET_ID = "window";
const CUSTOM_TARGET_PREFIX = "custom:";
const TARGET_ATTRIBUTE = "data-scroll-restoration-id";
/** The `data-scroll-restoration-id` value carried by the main layout pane. */
export const MAIN_SCROLL_RESTORATION_ID = "main";
const SCROLL_KEYS = new Set([
  "ArrowDown",
  "ArrowUp",
  "End",
  "Home",
  "PageDown",
  "PageUp",
  " ",
]);

type ScrollPosition = {
  x: number;
  y: number;
};

type SavedPositions = Record<string, Record<string, ScrollPosition>>;
type ScrollTarget = Window | HTMLElement;

type ActiveRestoration = {
  locationKey: string;
  positions: Record<string, ScrollPosition>;
};

type PendingRestore = {
  cancel: () => void;
  keepAlive: () => void;
  setBusy: (busy: boolean) => void;
};

type ScrollRestorationContextValue = {
  positions: Record<string, ScrollPosition>;
  keepAlive: (id: string) => void;
  setBusy: (id: string, busy: boolean) => void;
};

const ScrollRestorationContext = createContext<
  ScrollRestorationContextValue | undefined
>(undefined);

function isWindowTarget(target: ScrollTarget): target is Window {
  return target === window;
}

function locationStorageKey(location: Location): string {
  if (location.key !== "default") {
    return location.key;
  }
  return `default:${location.pathname}${location.search}${location.hash}`;
}

function loadSavedPositions(): SavedPositions {
  try {
    const saved = sessionStorage.getItem(STORAGE_KEY);
    return saved ? (JSON.parse(saved) as SavedPositions) : {};
  } catch {
    return {};
  }
}

function persistSavedPositions(savedPositions: SavedPositions): void {
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(savedPositions));
  } catch {
    // Scroll restoration is best-effort when storage is unavailable.
  }
}

function savePosition(
  savedPositions: SavedPositions,
  locationKey: string,
  targetId: string,
  position: ScrollPosition
): void {
  if (!savedPositions[locationKey]) {
    const savedKeys = Object.keys(savedPositions);
    if (savedKeys.length >= MAX_SAVED_ENTRIES) {
      delete savedPositions[savedKeys[0]];
    }
    savedPositions[locationKey] = {};
  }
  savedPositions[locationKey][targetId] = position;
}

function copyPositions(
  savedPositions: SavedPositions,
  fromLocationKey: string,
  toLocationKey: string
): void {
  const positions = savedPositions[fromLocationKey];
  if (!positions || fromLocationKey === toLocationKey) return;
  for (const [id, position] of Object.entries(positions)) {
    savePosition(savedPositions, toLocationKey, id, { ...position });
  }
}

function customTargetId(value: string): string {
  return `${CUSTOM_TARGET_PREFIX}${value}`;
}

function targetId(target: EventTarget | null): string | undefined {
  if (target === document) {
    return WINDOW_TARGET_ID;
  }
  if (!(target instanceof HTMLElement)) {
    return undefined;
  }
  const id = target.getAttribute(TARGET_ATTRIBUTE);
  return id ? customTargetId(id) : undefined;
}

function findTarget(id: string): ScrollTarget | null {
  if (id === WINDOW_TARGET_ID) {
    return window;
  }
  if (!id.startsWith(CUSTOM_TARGET_PREFIX)) {
    return null;
  }
  const value = id.slice(CUSTOM_TARGET_PREFIX.length);
  return document.querySelector<HTMLElement>(
    `[${TARGET_ATTRIBUTE}="${CSS.escape(value)}"]`
  );
}

function collectTargets(): Map<string, ScrollTarget> {
  const targets = new Map<string, ScrollTarget>([[WINDOW_TARGET_ID, window]]);
  for (const element of document.querySelectorAll<HTMLElement>(
    `[${TARGET_ATTRIBUTE}]`
  )) {
    const id = element.getAttribute(TARGET_ATTRIBUTE);
    if (id) {
      targets.set(customTargetId(id), element);
    }
  }
  return targets;
}

function readPosition(target: ScrollTarget): ScrollPosition {
  if (isWindowTarget(target)) {
    return { x: window.scrollX, y: window.scrollY };
  }
  return { x: target.scrollLeft, y: target.scrollTop };
}

function scrollRange(target: ScrollTarget): ScrollPosition {
  if (isWindowTarget(target)) {
    const root = document.documentElement;
    const body = document.body;
    return {
      x: Math.max(
        0,
        Math.max(root.scrollWidth, body?.scrollWidth ?? 0) - window.innerWidth
      ),
      y: Math.max(
        0,
        Math.max(root.scrollHeight, body?.scrollHeight ?? 0) -
          window.innerHeight
      ),
    };
  }
  return {
    x: Math.max(0, target.scrollWidth - target.clientWidth),
    y: Math.max(0, target.scrollHeight - target.clientHeight),
  };
}

function applyPosition(
  target: ScrollTarget,
  position: ScrollPosition
): { range: ScrollPosition; reached: boolean } {
  const range = scrollRange(target);
  const x = minmax(position.x, 0, range.x);
  const y = minmax(position.y, 0, range.y);
  if (isWindowTarget(target)) {
    if (window.scrollX !== x || window.scrollY !== y) {
      window.scrollTo(x, y);
    }
  } else {
    if (target.scrollLeft !== x) target.scrollLeft = x;
    if (target.scrollTop !== y) target.scrollTop = y;
  }
  return {
    range,
    reached: range.x >= position.x && range.y >= position.y,
  };
}

function restoreWhenReady(
  id: string,
  position: ScrollPosition,
  onDone: () => void
): PendingRestore {
  let stopped = false;
  let busy = false;
  let animationFrameId: number | undefined;
  let intervalId: number | undefined;
  let timeoutId: number | undefined;
  let maxTimeoutId: number | undefined;
  let resizeObserver: ResizeObserver | undefined;
  let observedTarget: ScrollTarget | null = null;
  let lastRange: ScrollPosition | undefined;

  const stop = () => {
    if (stopped) return;
    stopped = true;
    if (animationFrameId !== undefined) {
      window.cancelAnimationFrame(animationFrameId);
    }
    if (intervalId !== undefined) window.clearInterval(intervalId);
    if (timeoutId !== undefined) window.clearTimeout(timeoutId);
    if (maxTimeoutId !== undefined) window.clearTimeout(maxTimeoutId);
    resizeObserver?.disconnect();
    onDone();
  };

  const keepAlive = () => {
    if (stopped) return;
    if (timeoutId !== undefined) window.clearTimeout(timeoutId);
    timeoutId = window.setTimeout(() => {
      if (busy) {
        keepAlive();
        return;
      }
      stop();
    }, RESTORE_IDLE_TIMEOUT_MS);
  };

  const attempt = () => {
    if (stopped) return;
    if (
      !observedTarget ||
      (!isWindowTarget(observedTarget) && !observedTarget.isConnected)
    ) {
      const target = findTarget(id);
      if (!target) return;
      resizeObserver?.disconnect();
      resizeObserver = new ResizeObserver(attempt);
      const observedElement = isWindowTarget(target)
        ? document.documentElement
        : target;
      resizeObserver.observe(observedElement);
      if (
        !isWindowTarget(target) &&
        target.firstElementChild instanceof HTMLElement
      ) {
        resizeObserver.observe(target.firstElementChild);
      }
      observedTarget = target;
      lastRange = undefined;
    }
    const { range, reached } = applyPosition(observedTarget, position);
    if (reached) {
      stop();
      return;
    }
    if (!lastRange || range.x > lastRange.x || range.y > lastRange.y) {
      keepAlive();
    }
    lastRange = range;
  };

  attempt();
  if (!stopped) {
    animationFrameId = window.requestAnimationFrame(attempt);
    intervalId = window.setInterval(attempt, 100);
    maxTimeoutId = window.setTimeout(stop, RESTORE_MAX_TIMEOUT_MS);
    keepAlive();
  }
  return {
    cancel: stop,
    keepAlive,
    setBusy: (value) => {
      busy = value;
      keepAlive();
    },
  };
}

// Resetting needs no clamping, so skip the forced-layout reads of
// `applyPosition` — this runs on every forward navigation.
function resetTargets(): void {
  for (const target of collectTargets().values()) {
    if (isWindowTarget(target)) {
      window.scrollTo(0, 0);
    } else {
      target.scrollLeft = 0;
      target.scrollTop = 0;
    }
  }
}

function scrollToHash(hash: string): boolean {
  try {
    const element = document.getElementById(decodeURIComponent(hash.slice(1)));
    if (!element) return false;
    element.scrollIntoView();
    return true;
  } catch {
    return false;
  }
}

type NavigationScrollRestorationProps = {
  children: ReactNode;
};

export function useScrollRestorationLoadMore(
  paged: Pick<
    PagedDataResult<unknown>,
    "hasMore" | "isFetchingMore" | "dataList" | "loadMore"
  >,
  /** The value of `data-scroll-restoration-id`; omit for the main layout pane. */
  restorationId: string = MAIN_SCROLL_RESTORATION_ID
): void {
  const { hasMore, isFetchingMore, dataList, loadMore } = paged;
  const restoration = useContext(ScrollRestorationContext);
  const id = customTargetId(restorationId);
  // Depend on the primitive: the saved-position record is replaced on every
  // scroll event, so its identity churns without the requested `y` changing.
  const requestedY = restoration?.positions[id]?.y;
  const itemCount = dataList.length;

  useEffect(() => {
    if (requestedY === undefined) return;
    restoration?.setBusy(id, isFetchingMore);
    return () => restoration?.setBusy(id, false);
  }, [id, isFetchingMore, requestedY, restoration]);

  useEffect(() => {
    if (requestedY === undefined || !hasMore || isFetchingMore) return;
    const target = findTarget(id);
    if (!target || scrollRange(target).y >= requestedY) return;
    restoration?.keepAlive(id);
    loadMore();
  }, [
    hasMore,
    id,
    isFetchingMore,
    itemCount,
    loadMore,
    requestedY,
    restoration,
  ]);
}

/**
 * Restores registered scroll containers by browser history entry.
 * Nested containers opt in with `data-scroll-restoration-id="stable-id"`
 * (the main layout pane registers as `MAIN_SCROLL_RESTORATION_ID`).
 *
 * Hand-rolled instead of react-router's `<ScrollRestoration>`, which only
 * handles window scroll: this app scrolls in nested containers and needs the
 * `useScrollRestorationLoadMore` growth protocol for paged lists. Never mount
 * the built-in alongside — both manage `history.scrollRestoration` and would
 * fight over the window position.
 */
export function NavigationScrollRestoration({
  children,
}: NavigationScrollRestorationProps) {
  const location = useLocation();
  const navigationType = useNavigationType();
  // Lazy state, not `useRef(loadSavedPositions())`: a ref initializer argument
  // is evaluated (storage read + JSON parse) on every render.
  const [savedPositions] = useState(loadSavedPositions);
  const locationKey = locationStorageKey(location);
  const currentLocationKeyRef = useRef(locationKey);
  const [pendingRestores] = useState(() => new Map<string, PendingRestore>());
  const restorationGenerationRef = useRef(0);
  const [activeRestoration, setActiveRestoration] =
    useState<ActiveRestoration>();

  const cancelPendingRestores = useCallback(() => {
    restorationGenerationRef.current++;
    for (const pending of pendingRestores.values()) {
      pending.cancel();
    }
    pendingRestores.clear();
    setActiveRestoration(undefined);
  }, [pendingRestores]);

  const keepRestorationAlive = useCallback(
    (id: string) => pendingRestores.get(id)?.keepAlive(),
    [pendingRestores]
  );

  const setRestorationBusy = useCallback(
    (id: string, busy: boolean) => pendingRestores.get(id)?.setBusy(busy),
    [pendingRestores]
  );

  const recordTarget = useCallback(
    (target: EventTarget | null) => {
      const id = targetId(target);
      if (!id || pendingRestores.has(id)) return;
      // Scroll events fire at frame rate: read the scrolled element directly
      // instead of re-querying the DOM for it, and defer persistence to
      // pagehide/unmount (SPA restores read the in-memory map).
      savePosition(
        savedPositions,
        currentLocationKeyRef.current,
        id,
        readPosition(target === document ? window : (target as HTMLElement))
      );
    },
    [pendingRestores, savedPositions]
  );

  const recordAllTargets = useCallback(
    (locationKey?: string) => {
      const key = locationKey ?? currentLocationKeyRef.current;
      for (const [id, target] of collectTargets()) {
        if (pendingRestores.has(id)) continue;
        savePosition(savedPositions, key, id, readPosition(target));
      }
    },
    [pendingRestores, savedPositions]
  );

  const flushToStorage = useCallback(() => {
    recordAllTargets();
    persistSavedPositions(savedPositions);
  }, [recordAllTargets, savedPositions]);

  useEffect(() => subscribeRoute(recordAllTargets), [recordAllTargets]);

  useEffect(() => {
    // Window scrolls arrive here too, targeting `document` — no separate
    // window listener needed.
    const handleElementScroll = (event: Event) => recordTarget(event.target);
    const handleUserScrollIntent = () => cancelPendingRestores();
    const handleKeyDown = (event: KeyboardEvent) => {
      if (SCROLL_KEYS.has(event.key)) {
        cancelPendingRestores();
      }
    };

    document.addEventListener("scroll", handleElementScroll, {
      capture: true,
      passive: true,
    });
    document.addEventListener("pointerdown", handleUserScrollIntent, true);
    document.addEventListener("touchstart", handleUserScrollIntent, {
      capture: true,
      passive: true,
    });
    document.addEventListener("wheel", handleUserScrollIntent, {
      capture: true,
      passive: true,
    });
    document.addEventListener("keydown", handleKeyDown, true);
    window.addEventListener("pagehide", flushToStorage);

    return () => {
      document.removeEventListener("scroll", handleElementScroll, true);
      document.removeEventListener("pointerdown", handleUserScrollIntent, true);
      document.removeEventListener("touchstart", handleUserScrollIntent, true);
      document.removeEventListener("wheel", handleUserScrollIntent, true);
      document.removeEventListener("keydown", handleKeyDown, true);
      window.removeEventListener("pagehide", flushToStorage);
    };
  }, [cancelPendingRestores, flushToStorage, recordTarget]);

  useEffect(() => {
    const previous = window.history.scrollRestoration;
    window.history.scrollRestoration = "manual";
    return () => {
      window.history.scrollRestoration = previous;
      flushToStorage();
    };
  }, [flushToStorage]);

  useLayoutEffect(() => {
    const previousLocationKey = currentLocationKeyRef.current;
    if (previousLocationKey !== locationKey) {
      recordAllTargets(previousLocationKey);
    }
    cancelPendingRestores();
    currentLocationKeyRef.current = locationKey;
    const savedForLocation = savedPositions[locationKey];
    const preventScrollReset = getAppRouterState()?.preventScrollReset === true;

    if (preventScrollReset) {
      copyPositions(savedPositions, previousLocationKey, locationKey);
    }

    if (navigationType === "POP" && savedForLocation) {
      const currentTargets = collectTargets();
      const targetIds = new Set([
        ...currentTargets.keys(),
        ...Object.keys(savedForLocation),
      ]);
      const generation = restorationGenerationRef.current;
      let remaining = targetIds.size;
      setActiveRestoration({ locationKey, positions: savedForLocation });

      for (const id of targetIds) {
        const position = savedForLocation[id] ?? { x: 0, y: 0 };
        // `restoreWhenReady` may finish synchronously; only register the
        // canceler while the restore is still pending.
        let settled = false;
        const pending = restoreWhenReady(id, position, () => {
          settled = true;
          pendingRestores.delete(id);
          if (restorationGenerationRef.current !== generation) return;
          remaining--;
          if (remaining === 0) {
            setActiveRestoration(undefined);
          }
        });
        if (!settled) {
          pendingRestores.set(id, pending);
        }
      }
      return cancelPendingRestores;
    }

    if (location.hash && scrollToHash(location.hash)) {
      return;
    }

    if (preventScrollReset) {
      return;
    }

    resetTargets();
  }, [
    cancelPendingRestores,
    location,
    locationKey,
    navigationType,
    pendingRestores,
    recordAllTargets,
    savedPositions,
  ]);

  const requestedRestoration = useMemo(
    () =>
      activeRestoration?.locationKey === locationKey
        ? {
            positions: activeRestoration.positions,
            keepAlive: keepRestorationAlive,
            setBusy: setRestorationBusy,
          }
        : undefined,
    [activeRestoration, keepRestorationAlive, locationKey, setRestorationBusy]
  );

  return (
    <ScrollRestorationContext value={requestedRestoration}>
      {children}
    </ScrollRestorationContext>
  );
}
