# Dashboard Scroll Restoration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make browser Back/Forward restore paginated dashboard lists immediately and accurately, with Issues and Plans as required consumers, including when a header banner changes the available viewport height, without moving dashboard scrolling to `window`.

**Architecture:** Keep the existing app-shell model: `DashboardFrameShell` owns the full viewport, the banner and dashboard header remain outside route scrolling, and `#bb-layout-main` remains the only route-level vertical scroll owner. The restoration engine and `usePagedData` cache are resource-agnostic: each paginated page supplies only a stable view key and each row supplies only a stable resource name. On browser-history POP, hydrate the matching bounded page snapshot, restore the saved coordinate, then correct it against the semantic row anchor; PUSH/REPLACE fetch normally and expired/missing snapshots retain `useScrollRestorationLoadMore` as the cold-cache fallback.

**Tech Stack:** React, TypeScript, React Router data router, Vitest/jsdom, Playwright, Tailwind CSS v4.

## Global Constraints

- Dashboard only. SQL Editor is explicitly out of scope.
- Work directly on `main` and leave every change uncommitted and unstaged.
- No backend, proto, schema, API, or dependency changes.
- Do not replace React Router or add TanStack Router/Query solely for this fix.
- Do not migrate dashboard scrolling to `window`.
- Do not virtualize the issue list in this change. The cache and anchor contracts should remain compatible with later virtualization.
- Keep `DashboardFrameShell.tsx` and `DashboardBodyShell.tsx` structurally unchanged; their current banner/header/main separation is the desired architecture.
- The cache and restoration lifecycle must not contain Issue- or Plan-specific logic. Project Issues, My Issues, and Project Plans are required consumers; Databases, Instances, Projects, and other paginated pages can opt in later by supplying the same generic view-key and row-anchor contract.
- The page cache is a tab-local history snapshot, not a source of truth. Only POP may hydrate it; PUSH/REPLACE and explicit `refresh()` fetch normally.
- Cache entries expire five minutes after the successful fetch/mutation that wrote them. Reads and hydrations may update LRU order but must not extend this absolute lifetime or write the same snapshot back.

---

## Root cause and target behavior

PR #20920 introduced history-entry-keyed restoration, but the current `useLayoutEffect` snapshots the previous history key after the destination route has committed. When the destination resets `#bb-layout-main` to `0`, that late snapshot overwrites the correct outgoing value already captured by the router subscription. The failure is most visible when the list and detail routes have different layout/scroll behavior.

The target behavior follows the modern app-shell pattern observed in Linear:

1. `window.scrollY` stays `0`; the shell owns viewport sizing.
2. The banner and dashboard header consume natural height above the route scroller.
3. `#bb-layout-main` is the one dashboard route-level vertical scroller.
4. Browser history stores the scroll state for each history entry, not just each pathname.
5. A warm Back/Forward POP restores previously loaded pages synchronously, then restores the scroll position; PUSH/REPLACE fetches normally.
6. A semantic resource-row anchor corrects coordinate drift caused by variable-height content.
7. A cold Back can still grow paged content until the saved coordinate is reachable.

## File map

**Create:**

- `frontend/src/react/hooks/pagedDataCache.ts` — bounded tab-memory cache for complete paged-list snapshots with absolute expiry.
- `frontend/src/react/hooks/pagedDataCache.test.ts` — cloning, non-sliding expiry, and LRU eviction coverage.
- `frontend/tests/e2e/scroll-restoration/dashboard-scroll-restoration.spec.ts` — browser Back regression coverage with and without the workspace banner.

**Modify:**

- `frontend/src/react/router/NavigationScrollRestoration.tsx` — remove the late outgoing snapshot; capture and apply semantic anchors.
- `frontend/src/react/router/NavigationScrollRestoration.test.tsx` — reproduce the timing regression and variable-height anchor drift.
- `frontend/src/react/hooks/usePagedData.tsx` — add explicit history-only synchronous hydration while preserving PUSH/REPLACE fetch and Load More behavior.
- `frontend/src/react/hooks/usePagedData.test.tsx` — verify POP paints all loaded pages immediately, resumes with the cached token, and PUSH bypasses the snapshot.
- `frontend/src/react/pages/project/ProjectIssueDashboardPage.tsx` — provide a filter/order-aware project issue cache key.
- `frontend/src/react/pages/workspace/MyIssuesPage.tsx` — provide a filter/order-aware workspace issue cache key.
- `frontend/src/react/components/IssueTable.tsx` — expose each issue row as a stable scroll-restoration anchor and E2E seam.
- `frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx` — make Plan filters URL-backed, provide the Plans view key/POP token, and expose Plan rows as stable anchors.
- `frontend/src/react/pages/project/ProjectPlanDashboardPage.test.tsx` — verify the Plans page consumes the same generic pagination-restoration contract as Issues.
- `frontend/src/react/pages/project/plan-detail/ProjectPlanDetailPage.tsx` — make the dashboard main pane, not the page host, own vertical scrolling.
- `frontend/src/react/pages/project/plan-detail/ProjectPlanDetailPage.test.tsx` — lock the single-scroll-owner and sticky-header behavior.

---

## Task 1: Reproduce and fix the outgoing-snapshot timing regression

**Files:**

- Modify: `frontend/src/react/router/NavigationScrollRestoration.test.tsx`
- Modify: `frontend/src/react/router/NavigationScrollRestoration.tsx`

**Interfaces:**

- Consumes: `subscribeRoute(onChange: () => void)` as the synchronous pre-commit router seam.
- Produces: corrected history-entry snapshots; no new exported API.

- [ ] **Step 1: Add a destination-layout regression helper**

Change the React import in `NavigationScrollRestoration.test.tsx` to include `useLayoutEffect`:

```tsx
import { act, useLayoutEffect, useState } from "react";
```

Add this helper below `recordScroll`. It deliberately changes the shared main target during the destination commit, matching the real list-to-detail transition:

```tsx
function RouteScrollMetrics({
  scrollHeight,
  resetScrollTop = false,
}: {
  scrollHeight: number;
  resetScrollTop?: boolean;
}) {
  useLayoutEffect(() => {
    const main = document.querySelector<HTMLElement>(
      `[data-scroll-restoration-id='${MAIN_SCROLL_RESTORATION_ID}']`
    );
    if (!main) throw new Error("Missing main scroll target");
    Object.defineProperty(main, "scrollHeight", {
      configurable: true,
      value: scrollHeight,
    });
    if (resetScrollTop) main.scrollTop = 0;
  }, [resetScrollTop, scrollHeight]);

  return null;
}
```

Extend `renderRouter` so both route elements are injectable:

```tsx
async function renderRouter(
  firstElement = <div>First</div>,
  secondElement = <div>Second</div>
) {
  const router = createMemoryRouter(
    [
      {
        path: "/",
        element: <TestShell />,
        children: [
          { path: "first", element: firstElement },
          { path: "second", element: secondElement },
        ],
      },
    ],
    {
      initialEntries: [{ pathname: "/first", key: "first-entry" }],
    }
  );
```

- [ ] **Step 2: Write the failing history-snapshot test**

Add this test after `snapshots the outgoing entry without waiting for a scroll event`:

```tsx
test("does not overwrite the outgoing position after destination layout commits", async () => {
  const { main, router, unmount } = await renderRouter(
    <RouteScrollMetrics scrollHeight={1000} />,
    <RouteScrollMetrics scrollHeight={200} resetScrollTop />
  );

  main.scrollTop = 640;
  await act(async () => {
    await router.navigate("/second");
  });
  expect(main.scrollTop).toBe(0);

  await act(async () => {
    await router.navigate(-1);
  });

  expect(main.scrollTop).toBe(640);
  unmount();
});
```

Run:

```bash
pnpm --dir frontend exec vitest run src/react/router/NavigationScrollRestoration.test.tsx
```

Expected: the new test fails with `expected 0 to be 640`, proving that the destination commit overwrote the outgoing entry.

- [ ] **Step 3: Remove the post-commit snapshot**

In the navigation `useLayoutEffect` in `NavigationScrollRestoration.tsx`, replace:

```tsx
const previousLocationKey = currentLocationKeyRef.current;
if (previousLocationKey !== locationKey) {
  recordAllTargets(previousLocationKey);
}
cancelPendingRestores();
```

with:

```tsx
const previousLocationKey = currentLocationKeyRef.current;
cancelPendingRestores();
```

Remove `recordAllTargets` from that layout effect's dependency array. Do not remove this pre-commit subscription:

```tsx
useEffect(() => subscribeRoute(recordAllTargets), [recordAllTargets]);
```

That subscription is the authoritative outgoing snapshot; scroll events remain the incremental fast path, and `pagehide`/unmount remain persistence fallbacks.

- [ ] **Step 4: Run the focused restoration tests**

Run:

```bash
pnpm --dir frontend exec vitest run src/react/router/NavigationScrollRestoration.test.tsx
```

Expected: all tests pass, including the new route-content-change regression and the existing repeated Push/Back, `preventScrollReset`, cancellation, and slow-page tests.

- [ ] **Step 5: Inspect the timing-fix diff**

Run `git diff --check` and keep the timing fix uncommitted on `main`.

---

## Task 2: Add semantic anchor correction for variable-height issue rows

**Files:**

- Modify: `frontend/src/react/router/NavigationScrollRestoration.test.tsx`
- Modify: `frontend/src/react/router/NavigationScrollRestoration.tsx`

**Interfaces:**

- Consumes: registered targets using `data-scroll-restoration-id` and `MAIN_SCROLL_RESTORATION_ID`.
- Produces: optional `ScrollPosition.anchor` and the DOM contract `data-scroll-restoration-anchor="stable-resource-name"` used by Task 5.

- [ ] **Step 1: Add an anchor geometry test helper**

Add `useRef` to the React import in `NavigationScrollRestoration.test.tsx`. First make `TestShell` capable of rendering the route directly under `main` while preserving the nested `panel` used by existing tests:

```tsx
function TestShell({ routeInsidePanel = true }: { routeInsidePanel?: boolean }) {
  return (
    <NavigationScrollRestoration>
      <div data-scroll-restoration-id={MAIN_SCROLL_RESTORATION_ID}>
        <div data-scroll-restoration-id="panel">
          {routeInsidePanel ? <Outlet /> : null}
        </div>
        {routeInsidePanel ? null : <Outlet />}
      </div>
    </NavigationScrollRestoration>
  );
}
```

Extend the `renderRouter` signature introduced in Task 1 and pass the option into the shell. The empty `panel` remains mounted so its existing return type and all earlier tests stay unchanged:

```tsx
async function renderRouter(
  firstElement = <div>First</div>,
  secondElement = <div>Second</div>,
  { routeInsidePanel = true }: { routeInsidePanel?: boolean } = {}
) {
  const router = createMemoryRouter(
    [
      {
        path: "/",
        element: <TestShell routeInsidePanel={routeInsidePanel} />,
        children: [
          { path: "first", element: firstElement },
          { path: "second", element: secondElement },
        ],
      },
    ],
    {
      initialEntries: [{ pathname: "/first", key: "first-entry" }],
    }
  );
```

Then extend `makeScrollable` with a real viewport rectangle so visibility checks do not use jsdom's zero-sized default:

```tsx
function makeScrollable(element: HTMLElement): void {
  Object.defineProperties(element, {
    clientHeight: { configurable: true, value: 200 },
    clientWidth: { configurable: true, value: 200 },
    scrollHeight: { configurable: true, value: 1000 },
    scrollWidth: { configurable: true, value: 1000 },
  });
  element.getBoundingClientRect = () => ({
    x: 0,
    y: 0,
    top: 0,
    bottom: 200,
    left: 0,
    right: 200,
    width: 200,
    height: 200,
    toJSON: () => ({}),
  });
}
```

Then add:

```tsx
function AnchorPage({ anchorTop }: { anchorTop: () => number }) {
  const anchorRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const anchor = anchorRef.current;
    const main = anchor?.closest<HTMLElement>(
      `[data-scroll-restoration-id='${MAIN_SCROLL_RESTORATION_ID}']`
    );
    if (!anchor || !main) throw new Error("Missing anchor test elements");

    anchor.getBoundingClientRect = () => {
      const top = anchorTop() - main.scrollTop;
      return {
        x: 0,
        y: top,
        top,
        bottom: top + 40,
        left: 0,
        right: 200,
        width: 200,
        height: 40,
        toJSON: () => ({}),
      };
    };
  }, [anchorTop]);

  return (
    <div
      ref={anchorRef}
      data-scroll-restoration-anchor="issues/anchor"
    >
      Anchor
    </div>
  );
}
```

- [ ] **Step 2: Write the failing variable-height test**

Add:

```tsx
test("keeps the same semantic anchor offset when row geometry changes", async () => {
  let anchorTop = 400;
  const { main, router, unmount } = await renderRouter(
    <AnchorPage anchorTop={() => anchorTop} />,
    <div>Second</div>,
    { routeInsidePanel: false }
  );

  recordScroll(main, 380);
  await act(async () => {
    await router.navigate("/second");
  });

  // Simulate content above the saved row becoming 60px taller before Back.
  anchorTop = 460;
  await act(async () => {
    await router.navigate(-1);
  });

  // Coordinate-only restoration would leave scrollTop at 380 and move the
  // anchor from 20px to 80px below the scroll viewport's top edge.
  expect(main.scrollTop).toBe(440);
  unmount();
});
```

Run:

```bash
pnpm --dir frontend exec vitest run src/react/router/NavigationScrollRestoration.test.tsx
```

Expected: the new test fails with `expected 380 to be 440`.

- [ ] **Step 3: Extend saved positions with an optional anchor**

Near the restoration constants and types, add:

```tsx
const ANCHOR_ATTRIBUTE = "data-scroll-restoration-anchor";

type ScrollAnchor = {
  key: string;
  offset: number;
};

type ScrollPosition = {
  x: number;
  y: number;
  anchor?: ScrollAnchor;
};
```

Replace the existing `ScrollPosition` declaration rather than defining it twice.

- [ ] **Step 4: Capture only anchors owned by the current scroll target**

Add these helpers before `readPosition`:

```tsx
function anchorBelongsToTarget(
  anchor: HTMLElement,
  target: ScrollTarget
): boolean {
  const owner = anchor.closest<HTMLElement>(`[${TARGET_ATTRIBUTE}]`);
  return isWindowTarget(target) ? owner === null : owner === target;
}

function readAnchor(target: ScrollTarget): ScrollAnchor | undefined {
  const root: ParentNode = isWindowTarget(target) ? document : target;
  const viewportTop = isWindowTarget(target)
    ? 0
    : target.getBoundingClientRect().top;
  const viewportBottom = isWindowTarget(target)
    ? window.innerHeight
    : target.getBoundingClientRect().bottom;

  for (const anchor of root.querySelectorAll<HTMLElement>(
    `[${ANCHOR_ATTRIBUTE}]`
  )) {
    if (!anchorBelongsToTarget(anchor, target)) continue;
    const key = anchor.getAttribute(ANCHOR_ATTRIBUTE);
    const rect = anchor.getBoundingClientRect();
    if (key && rect.bottom > viewportTop && rect.top < viewportBottom) {
      return { key, offset: rect.top - viewportTop };
    }
  }
  return undefined;
}
```

Change `readPosition` to take an explicit anchor flag:

```tsx
function readPosition(
  target: ScrollTarget,
  includeAnchor = false
): ScrollPosition {
  const coordinate = isWindowTarget(target)
    ? { x: window.scrollX, y: window.scrollY }
    : { x: target.scrollLeft, y: target.scrollTop };
  const anchor = includeAnchor ? readAnchor(target) : undefined;
  return anchor ? { ...coordinate, anchor } : coordinate;
}
```

Keep high-frequency scroll-event recording coordinate-only. In `recordAllTargets`, change every full snapshot to:

```tsx
savePosition(savedPositions, key, id, readPosition(target, true));
```

This covers both the router-subscription snapshot and the `pagehide`/unmount flush, because both call `recordAllTargets`. Anchors remain serializable in session storage, while avoiding row scans on every scroll event.

- [ ] **Step 5: Apply the anchor after the coordinate becomes reachable**

Add:

```tsx
function applyAnchor(target: ScrollTarget, anchor: ScrollAnchor): boolean {
  const root: ParentNode = isWindowTarget(target) ? document : target;
  const element = Array.from(
    root.querySelectorAll<HTMLElement>(`[${ANCHOR_ATTRIBUTE}]`)
  ).find(
    (candidate) =>
      candidate.getAttribute(ANCHOR_ATTRIBUTE) === anchor.key &&
      anchorBelongsToTarget(candidate, target)
  );
  if (!element) return false;

  const viewportTop = isWindowTarget(target)
    ? 0
    : target.getBoundingClientRect().top;
  const delta = element.getBoundingClientRect().top - viewportTop - anchor.offset;
  if (Math.abs(delta) < 1) return true;

  if (isWindowTarget(target)) {
    window.scrollTo(window.scrollX, window.scrollY + delta);
  } else {
    target.scrollTop += delta;
  }
  return true;
}
```

In `restoreWhenReady`'s `attempt`, replace the `reached` branch with:

```tsx
const { range, reached } = applyPosition(observedTarget, position);
if (reached) {
  if (position.anchor) applyAnchor(observedTarget, position.anchor);
  stop();
  return;
}
```

The coordinate remains the required fallback. The anchor is best-effort: a deleted or filtered-out row must not keep restoration alive for 30 seconds. During cold restoration, the existing range-growth loop delays this branch until enough pages have rendered; during warm restoration, the cached row is already in the DOM.

- [ ] **Step 6: Run the restoration suite and inspect the diff**

```bash
pnpm --dir frontend exec vitest run src/react/router/NavigationScrollRestoration.test.tsx
git diff --check
```

Expected: all restoration tests pass; the anchor test ends at `scrollTop === 440`.

---

## Task 3: Add an absolutely-expiring tab-memory cache for paged view snapshots

**Files:**

- Create: `frontend/src/react/hooks/pagedDataCache.ts`
- Create: `frontend/src/react/hooks/pagedDataCache.test.ts`

**Interfaces:**

- Produces: `readPagedDataCache<T>(key)`, `writePagedDataCache<T>(key, snapshot)`, `deletePagedDataCache(key)`, `clearPagedDataCache()`, and `PagedDataCacheSnapshot<T>` for Task 4.
- Lifetime: absolute five minutes from write; reads refresh only LRU order.

- [ ] **Step 1: Write failing cache contract tests**

Create `pagedDataCache.test.ts`:

```ts
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
```

Run:

```bash
pnpm --dir frontend exec vitest run src/react/hooks/pagedDataCache.test.ts
```

Expected: the suite fails because `pagedDataCache.ts` does not exist.

- [ ] **Step 2: Implement the cache**

Create `pagedDataCache.ts`:

```ts
export type PagedDataCacheSnapshot<T> = {
  dataList: T[];
  hasMore: boolean;
  nextPageToken: string;
};

type StoredSnapshot<T> = PagedDataCacheSnapshot<T> & {
  cachedAt: number;
};

const MAX_ENTRIES = 20;
const MAX_AGE_MS = 5 * 60 * 1000;
const cache = new Map<string, StoredSnapshot<unknown>>();

// List arrays are copied so consumers cannot change membership/order in the
// stored view. Items remain shared under the app's immutable proto convention.
const cloneSnapshot = <T>(
  snapshot: PagedDataCacheSnapshot<T>
): PagedDataCacheSnapshot<T> => ({
  dataList: [...snapshot.dataList],
  hasMore: snapshot.hasMore,
  nextPageToken: snapshot.nextPageToken,
});

export function readPagedDataCache<T>(
  key: string | undefined
): PagedDataCacheSnapshot<T> | undefined {
  if (!key) return undefined;
  const stored = cache.get(key) as StoredSnapshot<T> | undefined;
  if (!stored) return undefined;
  if (Date.now() - stored.cachedAt > MAX_AGE_MS) {
    cache.delete(key);
    return undefined;
  }

  // Refresh Map insertion order for LRU eviction without extending expiry.
  cache.delete(key);
  cache.set(key, stored as StoredSnapshot<unknown>);
  return cloneSnapshot(stored);
}

export function writePagedDataCache<T>(
  key: string | undefined,
  snapshot: PagedDataCacheSnapshot<T>
): void {
  if (!key) return;
  cache.delete(key);
  cache.set(key, {
    ...cloneSnapshot(snapshot),
    cachedAt: Date.now(),
  } as StoredSnapshot<unknown>);

  while (cache.size > MAX_ENTRIES) {
    const oldestKey = cache.keys().next().value as string | undefined;
    if (!oldestKey) break;
    cache.delete(oldestKey);
  }
}

export function deletePagedDataCache(key: string | undefined): void {
  if (key) cache.delete(key);
}

export function clearPagedDataCache(): void {
  cache.clear();
}
```

- [ ] **Step 3: Run the cache tests**

```bash
pnpm --dir frontend exec vitest run src/react/hooks/pagedDataCache.test.ts
```

Expected: 3 tests pass.

---

## Task 4: Hydrate `usePagedData` only for browser-history POP

**Files:**

- Modify: `frontend/src/react/hooks/usePagedData.test.tsx`
- Modify: `frontend/src/react/hooks/usePagedData.tsx`

**Interfaces:**

- Consumes: the cache functions and snapshot type from Task 3.
- Produces: optional `usePagedData({ cacheKey?: string, cacheRestoreToken?: string, ... })`; a history-entry token authorizes one synchronous cache read for that POP activation without changing `PagedDataResult<T>`.

- [ ] **Step 1: Write failing POP-hydration and PUSH-bypass tests**

Import `clearPagedDataCache` into `usePagedData.test.tsx` and call it in the existing `afterEach` inside `describe("usePagedData")`:

```tsx
import { clearPagedDataCache } from "./pagedDataCache";
```

```tsx
clearPagedDataCache();
```

Add this module-level harness below the existing `Harness`:

```tsx
function CacheHarness({
  cacheKey,
  cacheRestoreToken,
  fetchList,
}: {
  cacheKey: string;
  cacheRestoreToken?: string;
  fetchList: FetchList;
}) {
  const paged = usePagedData<Item>({
    sessionKey: "test-cached-paged-data",
    cacheKey,
    cacheRestoreToken,
    fetchList,
  });

  return (
    <div>
      <div data-state={paged.isLoading ? "loading" : "ready"}>
        {paged.dataList.map((item) => item.name).join(",")}
      </div>
      <button type="button" onClick={paged.loadMore}>
        load more
      </button>
    </div>
  );
}
```

Place both new tests inside the first `describe("usePagedData")` block, where the existing `container` and `root` bindings are defined. The first test proves a POP-style remount restores every page and the continuation token:

```tsx
test("restores every loaded page and cached token for history POP", async () => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);

  const initialFetch = vi
    .fn<FetchList>()
    .mockResolvedValueOnce({
      list: [{ name: "items/1" }],
      nextPageToken: "page-2",
    })
    .mockResolvedValueOnce({
      list: [{ name: "items/2" }],
      nextPageToken: "page-3",
    });

  await act(async () => {
    root!.render(
      <CacheHarness cacheKey="issues" fetchList={initialFetch} />
    );
    await Promise.resolve();
  });
  await act(async () => {
    container!.querySelector("button")?.click();
    await Promise.resolve();
  });
  expect(container.querySelector("[data-state]")?.textContent).toBe(
    "items/1,items/2"
  );

  act(() => root?.unmount());
  root = createRoot(container);
  const resumedFetch = vi.fn<FetchList>(async () => ({
    list: [{ name: "items/3" }],
  }));

  act(() => {
    root!.render(
      <CacheHarness
        cacheKey="issues"
        cacheRestoreToken="entry-1"
        fetchList={resumedFetch}
      />
    );
  });
  expect(container.querySelector("[data-state]")?.getAttribute("data-state")).toBe(
    "ready"
  );
  expect(container.querySelector("[data-state]")?.textContent).toBe(
    "items/1,items/2"
  );
  expect(resumedFetch).not.toHaveBeenCalled();

  await act(async () => {
    container!.querySelector("button")?.click();
    await Promise.resolve();
  });
  expect(resumedFetch).toHaveBeenCalledWith({
    pageSize: 50,
    pageToken: "page-3",
  });
});
```

The second test proves a PUSH-style remount ignores an existing snapshot and fetches current data:

```tsx
test("fetches current data for a push-style remount", async () => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);

  const initialFetch = vi.fn<FetchList>(async () => ({
    list: [{ name: "items/cached" }],
  }));
  await act(async () => {
    root!.render(
      <CacheHarness cacheKey="issues" fetchList={initialFetch} />
    );
    await Promise.resolve();
  });

  act(() => root?.unmount());
  root = createRoot(container);
  const currentFetch = vi.fn<FetchList>(async () => ({
    list: [{ name: "items/current" }],
  }));
  await act(async () => {
    root!.render(
      <CacheHarness cacheKey="issues" fetchList={currentFetch} />
    );
    await Promise.resolve();
  });

  expect(currentFetch).toHaveBeenCalledOnce();
  expect(container.querySelector("[data-state]")?.textContent).toBe(
    "items/current"
  );
});
```

Add a fourth regression that keeps the hook mounted, visits another cache key
with no POP token, then returns to the original history token. The original
snapshot must hydrate again without a request; this proves token consumption is
scoped to one POP activation rather than permanently suppressing that history
entry.

Add another test to prove that merely hydrating the snapshot does not renew its absolute age:

```tsx
test("does not extend absolute expiry when a history view is hydrated", async () => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date("2026-07-20T00:00:00Z"));
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);

  const initialFetch = vi.fn<FetchList>(async () => ({
    list: [{ name: "items/cached" }],
  }));
  await act(async () => {
    root!.render(
      <CacheHarness cacheKey="issues" fetchList={initialFetch} />
    );
    await Promise.resolve();
  });
  act(() => root?.unmount());

  act(() => vi.advanceTimersByTime(4 * 60 * 1000));
  root = createRoot(container);
  const warmFetch = vi.fn<FetchList>(async () => ({ list: [] }));
  act(() => {
    root!.render(
      <CacheHarness
        cacheKey="issues"
        cacheRestoreToken="entry-1"
        fetchList={warmFetch}
      />
    );
  });
  expect(warmFetch).not.toHaveBeenCalled();
  expect(container.querySelector("[data-state]")?.textContent).toBe(
    "items/cached"
  );
  act(() => root?.unmount());

  act(() => vi.advanceTimersByTime(60 * 1000 + 1));
  root = createRoot(container);
  const expiredFetch = vi.fn<FetchList>(async () => ({
    list: [{ name: "items/current" }],
  }));
  await act(async () => {
    root!.render(
      <CacheHarness
        cacheKey="issues"
        cacheRestoreToken="entry-1"
        fetchList={expiredFetch}
      />
    );
    await Promise.resolve();
  });

  expect(expiredFetch).toHaveBeenCalledOnce();
  expect(container.querySelector("[data-state]")?.textContent).toBe(
    "items/current"
  );
});
```

Run:

```bash
pnpm --dir frontend exec vitest run src/react/hooks/usePagedData.test.tsx
```

Expected: TypeScript/Vitest fails because `usePagedData` does not accept `cacheKey` or `cacheRestoreToken`, does not persist/hydrate snapshots, and has no protection against hydration renewing cache age.

- [ ] **Step 2: Add cache ownership and dirty-state tracking to the reducer**

In `usePagedData.tsx`, import `useLayoutEffect` and `useState`, plus the cache helpers:

```tsx
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useReducer,
  useRef,
  useState,
} from "react";
import {
  deletePagedDataCache,
  type PagedDataCacheSnapshot,
  readPagedDataCache,
  writePagedDataCache,
} from "./pagedDataCache";
```

Extend the reducer state/actions. `cacheWriteState` is the critical distinction: hydrated snapshots are `clean`, while successful fetches and local list mutations are `dirty` and may reset absolute cache age.

```tsx
type PagedDataState<T> = {
  cacheKey?: string;
  cacheWriteState: "clean" | "dirty";
  dataList: T[];
  status: "idle" | "loading" | "ready" | "loadingMore";
  hasMore: boolean;
};

type PagedDataAction<T> =
  | {
      type: "restore-cache";
      cacheKey?: string;
      snapshot?: PagedDataCacheSnapshot<T>;
    }
  | { type: "fetch-start"; mode: FetchMode; cacheKey?: string }
  | {
      type: "fetch-success";
      mode: FetchMode;
      cacheKey?: string;
      list: T[];
      hasMore: boolean;
    }
  | { type: "fetch-error" }
  | { type: "update-cache"; items: T[] }
  | { type: "remove-cache"; item: T }
  | { type: "cache-persisted"; cacheKey?: string };

const stateFromSnapshot = <T extends { name: string }>(
  cacheKey: string | undefined,
  snapshot?: PagedDataCacheSnapshot<T>
): PagedDataState<T> => ({
  cacheKey,
  cacheWriteState: "clean",
  dataList: snapshot?.dataList ?? [],
  status: snapshot ? "ready" : "idle",
  hasMore: snapshot?.hasMore ?? false,
});
```

Replace the corresponding reducer cases with:

```tsx
case "restore-cache":
  return stateFromSnapshot(action.cacheKey, action.snapshot);
case "fetch-start":
  return {
    ...state,
    cacheKey: action.cacheKey,
    status: action.mode === "refresh" ? "loading" : "loadingMore",
  };
case "fetch-success":
  return {
    cacheKey: action.cacheKey,
    cacheWriteState: "dirty",
    dataList:
      action.mode === "refresh"
        ? action.list
        : [...state.dataList, ...action.list],
    status: "ready",
    hasMore: action.hasMore,
  };
case "fetch-error":
  return {
    ...state,
    cacheWriteState: "clean",
    status: "ready",
  };
case "update-cache": {
  const dataList = [...state.dataList];
  for (const item of action.items) {
    const index = dataList.findIndex((data) => data.name === item.name);
    if (index >= 0) dataList[index] = item;
    else dataList.push(item);
  }
  return { ...state, cacheWriteState: "dirty", dataList };
}
case "remove-cache":
  return {
    ...state,
    cacheWriteState: "dirty",
    dataList: state.dataList.filter(
      (data) => data.name !== action.item.name
    ),
  };
case "cache-persisted":
  return action.cacheKey === state.cacheKey
    ? { ...state, cacheWriteState: "clean" }
    : state;
```

- [ ] **Step 3: Seed cache only when the caller identifies history restoration**

Add both options to the hook; they remain optional so the other 14 call sites keep existing behavior:

```tsx
export function usePagedData<T extends { name: string }>({
  sessionKey,
  cacheKey,
  cacheRestoreToken,
  fetchList,
  enabled = true,
}: {
  sessionKey: string;
  cacheKey?: string;
  cacheRestoreToken?: string;
  fetchList: (params: {
    pageSize: number;
    pageToken: string;
  }) => Promise<{ list: T[]; nextPageToken?: string }>;
  enabled?: boolean;
}): PagedDataResult<T> {
```

After obtaining `pageSize`, initialize the reducer and refs from at most one POP-authorized cache read:

```tsx
const resolvedCacheKey = cacheKey
  ? `${cacheKey}:page-size=${pageSize}`
  : undefined;
const [initialCache] = useState(() => ({
  key: resolvedCacheKey,
  cacheRestoreToken,
  snapshot: cacheRestoreToken
    ? readPagedDataCache<T>(resolvedCacheKey)
    : undefined,
}));
const [state, dispatch] = useReducer(
  pagedDataReducer<T>,
  initialCache,
  ({ key, snapshot }) => stateFromSnapshot(key, snapshot)
);

const abortRef = useRef<AbortController | null>(null);
const fetchIdRef = useRef(0);
const nextPageTokenRef = useRef(initialCache.snapshot?.nextPageToken ?? "");
const activeCacheKeyRef = useRef(initialCache.key);
const activeCacheRestoreTokenRef = useRef(initialCache.cacheRestoreToken);
const consumedCacheRestoreTokenRef = useRef(initialCache.cacheRestoreToken);
const skipNextFetchRef = useRef(Boolean(initialCache.snapshot));
```

Delete the old reducer initialization and duplicate refs. Add this layout effect before the fetch-on-mount effect so filter/order history POP can switch views before paint:

```tsx
useLayoutEffect(() => {
  if (
    activeCacheKeyRef.current === resolvedCacheKey &&
    activeCacheRestoreTokenRef.current === cacheRestoreToken
  ) {
    return;
  }
  fetchIdRef.current++;
  abortRef.current?.abort();
  activeCacheKeyRef.current = resolvedCacheKey;
  activeCacheRestoreTokenRef.current = cacheRestoreToken;

  const shouldRestore =
    cacheRestoreToken !== undefined &&
    consumedCacheRestoreTokenRef.current !== cacheRestoreToken;
  consumedCacheRestoreTokenRef.current = cacheRestoreToken;
  const snapshot = shouldRestore
    ? readPagedDataCache<T>(resolvedCacheKey)
    : undefined;
  nextPageTokenRef.current = snapshot?.nextPageToken ?? "";
  skipNextFetchRef.current = Boolean(snapshot);
  dispatch({
    type: "restore-cache",
    cacheKey: resolvedCacheKey,
    snapshot,
  });
}, [cacheRestoreToken, resolvedCacheKey]);
```

- [ ] **Step 4: Persist only fetched or mutated state**

In `doFetch`, associate both dispatches with the active key:

```tsx
const mode: FetchMode = isRefresh ? "refresh" : "append";
const activeCacheKey = activeCacheKeyRef.current;
dispatch({ type: "fetch-start", mode, cacheKey: activeCacheKey });
```

```tsx
dispatch({
  type: "fetch-success",
  mode,
  cacheKey: activeCacheKey,
  list: result.list,
  hasMore: Boolean(result.nextPageToken),
});
```

After `removeCache`, add this effect. It writes once for dirty state and immediately marks that state clean, so a POP hydration never renews `cachedAt`:

```tsx
useEffect(() => {
  if (
    state.cacheWriteState !== "dirty" ||
    state.status !== "ready" ||
    state.cacheKey !== resolvedCacheKey
  ) {
    return;
  }
  writePagedDataCache(state.cacheKey, {
    dataList: state.dataList,
    hasMore: state.hasMore,
    nextPageToken: nextPageTokenRef.current,
  });
  dispatch({ type: "cache-persisted", cacheKey: state.cacheKey });
}, [resolvedCacheKey, state]);
```

Make explicit refresh remove the snapshot before fetching:

```tsx
const refresh = useCallback(() => {
  deletePagedDataCache(activeCacheKeyRef.current);
  doFetch(true);
}, [doFetch]);
```

An initial/refresh error stays `clean`, so it cannot write an empty or stale error state into a new five-minute snapshot.

- [ ] **Step 5: Skip the request only when a POP snapshot was actually found**

Update the existing fetch-on-mount effect:

```tsx
const isFirstLoad = useRef(true);
useEffect(() => {
  if (!enabled) return;
  if (skipNextFetchRef.current) {
    skipNextFetchRef.current = false;
    isFirstLoad.current = false;
    return;
  }
  if (isFirstLoad.current) {
    isFirstLoad.current = false;
    doFetch(true);
    return;
  }
  fetchIdRef.current++;
  abortRef.current?.abort();
  dispatch({
    type: "fetch-start",
    mode: "refresh",
    cacheKey: activeCacheKeyRef.current,
  });
  const timer = setTimeout(() => doFetch(true), 300);
  return () => clearTimeout(timer);
}, [cacheRestoreToken, doFetch, enabled, resolvedCacheKey]);
```

Do not background-revalidate page one after a successful POP hydration: replacing a multi-page view with page one collapses scroll range. PUSH/REPLACE already fetches current data; POP after the absolute five-minute age becomes a cold fetch and uses the existing page-growth protocol.

- [ ] **Step 6: Run both cache suites and inspect the diff**

```bash
pnpm --dir frontend exec vitest run src/react/hooks/pagedDataCache.test.ts src/react/hooks/usePagedData.test.tsx
git diff --check
```

Expected: cache tests, POP hydration, cached-token continuation, PUSH bypass, and the pre-existing debounced-refresh test all pass.

---

## Task 5: Opt issue dashboards into cached views and stable row anchors

**Files:**

- Modify: `frontend/src/react/pages/project/ProjectIssueDashboardPage.tsx`
- Modify: `frontend/src/react/pages/workspace/MyIssuesPage.tsx`
- Modify: `frontend/src/react/components/IssueTable.tsx`

**Interfaces:**

- Consumes: `cacheRestoreToken` from Task 4 and `data-scroll-restoration-anchor` from Task 2.
- Produces: independent filter/order/page-size view keys plus issue-row anchors keyed by `issue.name`.

- [ ] **Step 1: Add a POP-aware, filter-aware project issue view key**

In `ProjectIssueDashboardPage.tsx`, import `useLocation` and `useNavigationType`:

```tsx
import { useNavigationType } from "react-router";
```

Read the navigation action inside the page and add this memo after `orderBy`:

```tsx
const navigationType = useNavigationType();
const location = useLocation();
const viewCacheKey = useMemo(
  () =>
    JSON.stringify([
      "project-issues",
      projectName,
      serializeSearchParams(searchParams),
      orderBy,
    ]),
  [orderBy, projectName, searchParams]
);
```

Pass it to the hook:

```tsx
const paged = usePagedData<Issue>({
  sessionKey: "bb.issue-table.project-issues",
  cacheKey: viewCacheKey,
  cacheRestoreToken: navigationType === "POP" ? location.key : undefined,
  fetchList: fetchIssueList,
});
```

`POP` here authorizes only the synchronous view seed. It does not stand in for the active restoration lifecycle; `NavigationScrollRestoration` and `useScrollRestorationLoadMore` still own completion, cancellation, timeout, and page growth. Keep `useScrollRestorationLoadMore(paged)` for expired/missing snapshots and positions deeper than the warm snapshot.

- [ ] **Step 2: Add the same explicit POP policy to My Issues**

In `MyIssuesPage.tsx`, import `useLocation` and `useNavigationType`, then add:

```tsx
import { useLocation, useNavigationType } from "react-router";
```

```tsx
const navigationType = useNavigationType();
const location = useLocation();
const viewCacheKey = useMemo(
  () =>
    JSON.stringify([
      "my-issues",
      serializeSearchParams(searchParams),
      orderBy,
    ]),
  [orderBy, searchParams]
);
```

Pass it to `usePagedData`:

```tsx
const paged = usePagedData<Issue>({
  sessionKey: "bb.issue-table.my-issues",
  cacheKey: viewCacheKey,
  cacheRestoreToken: navigationType === "POP" ? location.key : undefined,
  fetchList: fetchIssueList,
});
```

Both pages already use URL param `q` through `useURLSearchParam({ param: "q", ... })`; the serialized search text therefore drives both the backend filter and `IssueListPanel` highlight state. No route/query-key change is needed for the E2E token.

- [ ] **Step 3: Mark each issue row as a semantic anchor**

In `IssueListItem`, change the outer row to:

```tsx
<div
  data-slot="issue-list-item"
  data-testid="issue-list-item"
  data-scroll-restoration-anchor={issue.name}
  className="flex items-start gap-x-2 px-4 py-3 cursor-pointer border-b border-block-border transition-colors last:border-b-0 hover:bg-control-bg/60"
  onClick={onRowClick}
>
```

The resource name is stable, unique within the list, and independent of translated title text or sort position.

- [ ] **Step 4: Run focused tests and type-check the integration**

```bash
pnpm --dir frontend exec vitest run src/react/router/NavigationScrollRestoration.test.tsx src/react/hooks/pagedDataCache.test.ts src/react/hooks/usePagedData.test.tsx
pnpm --dir frontend type-check
```

Expected: all focused tests pass and TypeScript accepts both opt-in call sites; the other `usePagedData` call sites remain unchanged because both `cacheKey` and `cacheRestoreToken` are optional. A normal nav-link PUSH to Issues fetches instead of painting a potentially stale snapshot.

- [ ] **Step 5: Inspect the issue-list integration**

Run `git diff --check` and keep the Issue integration uncommitted and unstaged.

---

## Task 6: Enforce one dashboard route-level vertical scroll owner

**Files:**

- Modify: `frontend/src/react/pages/project/plan-detail/ProjectPlanDetailPage.test.tsx`
- Modify: `frontend/src/react/pages/project/plan-detail/ProjectPlanDetailPage.tsx`

**Interfaces:**

- Consumes: the dashboard ancestor carrying `data-scroll-restoration-id={MAIN_SCROLL_RESTORATION_ID}`.
- Produces: no new exported API; plan detail delegates vertical scrolling and sticky state to that ancestor.

- [ ] **Step 1: Write the failing plan-detail ownership test**

Import the main restoration id in `ProjectPlanDetailPage.test.tsx`:

```tsx
import { MAIN_SCROLL_RESTORATION_ID } from "@/react/router/NavigationScrollRestoration";
```

Add this test near the start of the `ProjectPlanDetailPage` suite:

```tsx
it("uses the dashboard main pane as its vertical scroll owner", async () => {
  mocks.usePlanDetailPage.mockReturnValue(buildPage());

  await act(async () => {
    root.render(
      <div data-scroll-restoration-id={MAIN_SCROLL_RESTORATION_ID}>
        <ProjectPlanDetailPage
          planId="create"
          projectId="foo"
          specId="spec-1"
        />
      </div>
    );
    await Promise.resolve();
  });

  const scrollHost = container.querySelector<HTMLElement>(
    `[data-scroll-restoration-id='${MAIN_SCROLL_RESTORATION_ID}']`
  );
  const pageHost = container.querySelector<HTMLElement>(
    '[data-testid="plan-detail-page"]'
  );
  const header = pageHost?.querySelector("header");
  if (!scrollHost || !pageHost || !header) {
    throw new Error("Missing plan-detail scroll test elements");
  }

  const pageHostClasses = pageHost.className.split(/\s+/);
  expect(pageHostClasses).toContain("min-h-full");
  expect(pageHostClasses).toContain("overflow-x-clip");
  expect(pageHostClasses).not.toContain("h-full");
  expect(pageHostClasses).not.toContain("overflow-x-hidden");

  act(() => {
    scrollHost.scrollTop = 1;
    scrollHost.dispatchEvent(new Event("scroll"));
  });
  expect(header.className).toContain("border-b");
});
```

Run:

```bash
pnpm --dir frontend exec vitest run src/react/pages/project/plan-detail/ProjectPlanDetailPage.test.tsx
```

Expected: the test fails because the page host still has `h-full overflow-x-hidden`, lacks the test id, and listens to its own `scrollTop`.

- [ ] **Step 2: Listen to the registered dashboard main pane**

Import `MAIN_SCROLL_RESTORATION_ID` in `ProjectPlanDetailPage.tsx`:

```tsx
import { MAIN_SCROLL_RESTORATION_ID } from "@/react/router/NavigationScrollRestoration";
```

Replace the current `pageHost` scroll effect with:

```tsx
const [pageHost, setPageHost] = useState<HTMLDivElement | null>(null);
// The sticky title row is positioned inside the dashboard's registered main
// pane, so its divider follows that pane's scroll state.
const [headerStuck, setHeaderStuck] = useState(false);
useEffect(() => {
  if (!pageHost) return;
  const scrollHost = pageHost.closest<HTMLElement>(
    `[data-scroll-restoration-id="${MAIN_SCROLL_RESTORATION_ID}"]`
  );
  if (!scrollHost) return;

  const onScroll = () => setHeaderStuck(scrollHost.scrollTop > 0);
  onScroll();
  scrollHost.addEventListener("scroll", onScroll, { passive: true });
  return () => scrollHost.removeEventListener("scroll", onScroll);
}, [pageHost]);
```

- [ ] **Step 3: Stop the page host from creating an implicit y scroller**

Change the page host to:

```tsx
<div
  ref={setPageHost}
  data-testid="plan-detail-page"
  className="relative min-h-full overflow-x-clip bg-gray-50"
>
```

Update the sticky-header comment to describe `#bb-layout-main` as the scroll container. `overflow-x-clip` clips horizontal paint without causing `overflow-y: auto`; `overflow-x-hidden` does cause an implicit vertical scrolling box and must not remain. `min-h-full` gives the page a one-viewport minimum while allowing its box and `bg-gray-50` painting area to grow with long content; keeping `h-full` would leave below-fold content overflowing a one-viewport background box.

- [ ] **Step 4: Run the plan-detail and restoration tests**

```bash
pnpm --dir frontend exec vitest run src/react/pages/project/plan-detail/ProjectPlanDetailPage.test.tsx src/react/router/NavigationScrollRestoration.test.tsx
```

Expected: both suites pass. The sticky divider reacts to the outer main pane, and list-to-detail navigation no longer changes scroll ownership.

- [ ] **Step 5: Verify below-fold background and sticky behavior in a browser**

Run the local frontend/backend or use the Playwright environment, then open a plan detail containing more than one viewport of expanded content:

1. At the top, confirm `#bb-layout-main.scrollTop === 0` and `window.scrollY === 0`.
2. Scroll past the first viewport and confirm `#bb-layout-main.scrollTop > 0` while `window.scrollY` remains `0`.
3. Confirm the gray page background continues behind below-fold content instead of switching to the parent background.
4. Confirm the plan title header remains sticky at the main-pane top and gains its bottom border while scrolled.

Expected: one vertical scrollbar on `#bb-layout-main`, continuous gray background for the full plan, and no plan-detail inner scrollbar.

- [ ] **Step 6: Inspect the single-owner fix**

Run `git diff --check` and keep the layout changes uncommitted and unstaged.

---

## Task 7: Add dashboard browser regression coverage, including the banner

**Files:**

- Create: `frontend/tests/e2e/scroll-restoration/dashboard-scroll-restoration.spec.ts`

**Interfaces:**

- Consumes: `data-testid="issue-list-item"`, `#bb-layout-main`, real project issue routes, and the workspace external-URL banner.
- Produces: serial browser regression coverage for POP restoration with and without wrapped banner geometry.

- [ ] **Step 1: Create owned issue fixtures and reusable position helpers**

Create the E2E file with the following complete structure:

```ts
import {
  expect,
  test,
  type BrowserContext,
  type Locator,
  type Page,
} from "@playwright/test";
import { BytebaseApiClient } from "../framework/api-client";
import { loadTestEnv, type TestEnv } from "../framework/env";

test.setTimeout(240_000);

const ISSUE_COUNT = 18;
let env: TestEnv & { api: BytebaseApiClient };
let sharedContext: BrowserContext;
let page: Page;
let listUrl: string;
let originalExternalUrl = "";

type SavedViewport = {
  scrollTop: number;
  anchorOffset: number;
};

const readViewport = (row: Locator): Promise<SavedViewport> =>
  row.evaluate((element) => {
    const main = document.querySelector<HTMLElement>("#bb-layout-main");
    if (!main) throw new Error("Missing #bb-layout-main");
    return {
      scrollTop: main.scrollTop,
      anchorOffset:
        element.getBoundingClientRect().top -
        main.getBoundingClientRect().top,
    };
  });

async function expectBackRestored(): Promise<void> {
  await page.goto(listUrl);
  const rows = page.getByTestId("issue-list-item");
  await expect(rows).toHaveCount(ISSUE_COUNT, { timeout: 20_000 });

  const target = rows.last();
  await target.scrollIntoViewIfNeeded();
  const before = await readViewport(target);
  expect(before.scrollTop).toBeGreaterThan(100);

  await target.click();
  await expect(page).toHaveURL(/\/projects\/[^/]+\/plans\//, {
    timeout: 20_000,
  });
  await page.goBack();
  await expect(page).toHaveURL(listUrl);
  await expect(target).toBeVisible({ timeout: 20_000 });

  await expect
    .poll(async () => {
      const after = await readViewport(target);
      return Math.abs(after.scrollTop - before.scrollTop);
    })
    .toBeLessThanOrEqual(2);
  await expect
    .poll(async () => {
      const after = await readViewport(target);
      return Math.abs(after.anchorOffset - before.anchorOffset);
    })
    .toBeLessThanOrEqual(2);
  expect(await page.evaluate(() => window.scrollY)).toBe(0);
}

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const projectId = env.project.split("/").pop()!;
  const stamp = Date.now();
  const searchToken = `scroll-restoration-${stamp}`;
  listUrl = `${env.baseURL}/projects/${projectId}/issues?q=${encodeURIComponent(
    searchToken
  )}`;

  const setting = await env.api.getSetting("WORKSPACE_PROFILE");
  originalExternalUrl =
    (setting?.value as { workspaceProfile?: { externalUrl?: string } })
      ?.workspaceProfile?.externalUrl ?? "";

  const sheet = await env.api.createSheet(env.project, "SELECT 1;");
  for (let i = 0; i < ISSUE_COUNT; i++) {
    const title = `${searchToken} issue ${i}`;
    const plan = await env.api.createPlan(env.project, title, [
      {
        id: `scroll-restoration-${stamp}-${i}`,
        targets: [env.database],
        sheet,
      },
    ]);
    const repeatedDescription = Array.from(
      { length: (i % 4) + 1 },
      () => `${searchToken} variable-height content`
    ).join("\n");
    await env.api.createIssue(
      env.project,
      title,
      plan.name,
      repeatedDescription
    );
  }

  sharedContext = await browser.newContext({
    storageState: ".auth/state.json",
    // Below Tailwind's sm breakpoint, the banner CTA moves onto its own row.
    viewport: { width: 600, height: 720 },
  });
  page = await sharedContext.newPage();
});

test.afterAll(async () => {
  await env.api.setWorkspaceExternalUrl(originalExternalUrl).catch(() => {});
  await sharedContext?.close();
});

test.describe("dashboard scroll restoration", () => {
  test.describe.configure({ mode: "serial" });

  test("restores the nested main pane without a banner", async () => {
    await env.api.setWorkspaceExternalUrl(env.baseURL);
    await page.goto(listUrl);
    await expect(
      page.getByRole("link", { name: /Configure now/i })
    ).toHaveCount(0);
    await expectBackRestored();
  });

  test("restores the nested main pane below a wrapped banner", async () => {
    await env.api.setWorkspaceExternalUrl("");
    await page.goto(listUrl);
    await expect(
      page.getByRole("link", { name: /Configure now/i }).first()
    ).toBeVisible({ timeout: 10_000 });
    await expectBackRestored();
  });
});
```

The project page reads this exact `q` parameter through `useURLSearchParam`, so the query token both scopes the backend result to owned rows and activates description highlighting. Descriptions of different lengths exercise semantic-anchor correction. The test asserts both coordinate and row-relative position; coordinate alone would miss a variable-height drift. It also asserts `window.scrollY === 0`, locking the nested-scroller architecture.

`ISSUE_COUNT` intentionally remains below the default page size of 50. This E2E suite owns the cross-layer browser contract—Router POP, real list/detail DOM, nested main scroller, semantic row offset, and banner geometry—without creating more than 50 plans and issues. Task 4's focused hook test owns the multi-page snapshot and cached-token continuation contract; Task 1/2 own cold page-growth behavior. Do not inflate this browser suite solely to duplicate those deterministic unit tests.

- [ ] **Step 2: Run the E2E spec**

Follow `frontend/tests/e2e/README.md` prerequisites: build the embedded frontend/server, provide the enterprise test license, and let the setup project create `.auth/state.json` and `.e2e-env.json`. Then run:

```bash
cd frontend
pnpm exec playwright test scroll-restoration/dashboard-scroll-restoration.spec.ts
```

Expected: 3 tests pass in serial mode. There are no arbitrary sleeps; all waits are tied to URLs, rows, banner visibility, or measured position convergence.

- [ ] **Step 3: Inspect the browser coverage**

Run `git diff --check` and keep the E2E coverage uncommitted and unstaged.

---

## Task 8: Connect Project Plans to the generic paginated-view contract

**Files:**

- Create: `frontend/src/react/pages/project/ProjectPlanDashboardPage.test.tsx`
- Modify: `frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx`
- Modify: `frontend/tests/e2e/scroll-restoration/dashboard-scroll-restoration.spec.ts`

**Interfaces:**

- Consumes: resource-agnostic `usePagedData({ cacheKey, cacheRestoreToken, fetchList })`, `useScrollRestorationLoadMore(paged)`, `useURLSearchParam`, and `data-scroll-restoration-anchor`.
- Produces: a filter-stable Project Plans view whose loaded pages and semantic row position restore on POP exactly like Project Issues.

- [ ] **Step 1: Write failing Plans integration tests**

Render `ProjectPlanDashboardPage` inside a memory router with `usePagedData` mocked only at its public boundary. Assert the page passes a project/filter-aware key and the current POP history key:

```tsx
expect(mockUsePagedData).toHaveBeenCalledWith(
  expect.objectContaining({
    cacheKey: expect.stringContaining("project-plans"),
    cacheRestoreToken: "default",
  })
);
```

Render one Plan and assert its existing table row exposes the generic semantic contract:

```tsx
expect(
  container.querySelector(
    '[data-scroll-restoration-anchor="projects/foo/plans/1"]'
  )
).not.toBeNull();
```

Start the router at `?q=state:DELETED` and assert the view key contains the serialized URL-owned filter rather than the component default.

- [ ] **Step 2: Run the Plans test and verify RED**

```bash
pnpm --dir frontend exec vitest run \
  src/react/pages/project/ProjectPlanDashboardPage.test.tsx
```

Expected: the cache-token and row-anchor assertions fail because Plans has not opted into the generic contract.

- [ ] **Step 3: Make the Plan filter URL-owned and opt into generic restoration**

Use the shared Advanced Search parser/serializer and the same page-level wiring already used by Issues:

```tsx
const [searchParams, setSearchParams] = useURLSearchParam({
  param: "q",
  parse: parsePlanSearchParams,
  serialize: serializeAdvancedSearch,
  defaultValue: defaultSearchParams,
});
const viewCacheKey = JSON.stringify([
  "project-plans",
  projectName,
  serializeAdvancedSearch(searchParams),
]);
const paged = usePagedData<Plan>({
  sessionKey: `bb.${projectName}.plan-table`,
  cacheKey: viewCacheKey,
  cacheRestoreToken:
    navigationType === "POP" ? location.key : undefined,
  fetchList: fetchPlanList,
});
```

Do not put Plan-specific behavior inside `usePagedData`, `pagedDataCache`, or `NavigationScrollRestoration`.

- [ ] **Step 4: Mark every Plan row with its stable resource name**

```tsx
<TableRow
  data-testid="plan-list-item"
  data-scroll-restoration-anchor={plan.name}
  ...
>
```

- [ ] **Step 5: Run the Plans and generic restoration suites**

```bash
pnpm --dir frontend exec vitest run \
  src/react/pages/project/ProjectPlanDashboardPage.test.tsx \
  src/react/hooks/usePagedData.test.tsx \
  src/react/router/NavigationScrollRestoration.test.tsx
```

Expected: all pass; generic hook/router tests remain resource-neutral.

- [ ] **Step 6: Extend browser coverage to Project Plans**

Reuse the owned Plans already created by the scroll-restoration E2E fixture. Navigate to the Project Plans URL with its URL-backed `q`, load more than one page through the real footer, click the last `plan-list-item`, go Back, and assert both `#bb-layout-main.scrollTop` and the Plan row's viewport-relative offset converge while `window.scrollY` remains `0`.

---

## Task 9: Run the complete frontend verification gates

**Files:** validation only; review any formatter changes before keeping them.

**Interfaces:**

- Consumes: all production and test changes from Tasks 1–8.
- Produces: formatter, static-check, type-check, unit-test, E2E, and diff evidence required before completion.

- [ ] **Step 1: Run focused unit tests once more**

```bash
pnpm --dir frontend exec vitest run \
  src/react/router/NavigationScrollRestoration.test.tsx \
  src/react/hooks/pagedDataCache.test.ts \
  src/react/hooks/usePagedData.test.tsx \
  src/react/pages/project/ProjectPlanDashboardPage.test.tsx \
  src/react/pages/project/plan-detail/ProjectPlanDetailPage.test.tsx
```

Expected: all focused tests pass.

- [ ] **Step 2: Run the repository-required frontend fixer**

```bash
pnpm --dir frontend fix
```

Expected: exits successfully. Inspect `git diff` and keep only formatting/import-order changes in the files covered by this plan.

- [ ] **Step 3: Run the CI-equivalent frontend check**

```bash
pnpm --dir frontend check
```

Expected: Biome, UUID, i18n, layering, and locale-sort checks all pass.

- [ ] **Step 4: Run TypeScript and all Vitest suites**

```bash
pnpm --dir frontend type-check
pnpm --dir frontend test
```

Expected: TypeScript build passes and all frontend unit tests pass.

- [ ] **Step 5: Re-run the browser regression after formatting**

```bash
cd frontend
pnpm exec playwright test scroll-restoration/dashboard-scroll-restoration.spec.ts
```

Expected: both Issue banner variants and the paginated Plans case pass.

- [ ] **Step 6: Inspect the final diff and whitespace**

```bash
git status --short
git diff --check
git diff --stat
```

Expected:

- No backend, proto, SQL Editor, dependency, locale, or generated-file changes.
- `git diff --check` produces no output.
- The dashboard shells still expose the banner and header outside `#bb-layout-main`.
- Every added test fails for the intended reason before its implementation step and passes afterward.

---

## Acceptance checklist

- [ ] Browser Back from detail restores both Project Issues and Project Plans to their prior `#bb-layout-main.scrollTop`.
- [ ] The same Issue or Plan row returns to the same viewport-relative offset when row heights differ.
- [ ] Previously loaded pages paint on the first warm POP render; the UI does not replay Load More page-by-page on a cache hit.
- [ ] PUSH/REPLACE mounts never skip their list request merely because the same view key exists in cache.
- [ ] Cache reads and POP hydrations do not extend the five-minute absolute expiry.
- [ ] An expired/missing cache still uses `useScrollRestorationLoadMore` until the coordinate is reachable.
- [ ] Filter and order variants do not share cached rows.
- [ ] My Issues and project Issues have independent cache namespaces.
- [ ] Project Plans uses an independent cache namespace and retains its URL-backed filter across remounts.
- [ ] A workspace banner, including a wrapped/narrow variant, does not change the restored row offset.
- [ ] The dashboard header and banner do not scroll with route content.
- [ ] `window.scrollY` remains `0` throughout the dashboard flow.
- [ ] Plan detail does not create a second route-level vertical scroller.
- [ ] SQL Editor behavior is unchanged and untested by this delivery.
