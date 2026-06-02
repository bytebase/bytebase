import { enableMapSet } from "immer";
import { cloneDeep, head, isUndefined, omitBy, pick } from "lodash-es";
import { create, type StoreApi, type UseBoundStore } from "zustand";
import { immer } from "zustand/middleware/immer";
import { useShallow } from "zustand/react/shallow";
import {
  extractWorksheetConnection,
  isConnectedSQLEditorTab,
} from "@/react/lib/sqlEditorConnection";
import { useAppStore } from "@/react/stores/app";
import {
  migrateDraftsFromCache,
  migrateTabViewState,
} from "@/store/modules/sqlEditor/legacy/migration";
import type {
  BatchQueryContext,
  SQLEditorDatabaseQueryContext,
  SQLEditorTab,
} from "@/types";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  defaultSQLEditorTab,
  getSheetStatement,
  storageKeySqlEditorCurrentTab,
  storageKeySqlEditorTabs,
  workspaceCacheScope,
} from "@/utils";
import {
  getSQLEditorEditorState,
  subscribeSQLEditorEditorState,
} from "./editor";

// SQL editor tabs are stored in a Map<id, tab> that this store mutates
// in place via immer drafts. Enabling the MapSet plugin teaches immer
// how to draft / produce Map and Set instances.
enableMapSet();

const PERSISTENT_TAB_FIELDS = [
  "id",
  "worksheet",
  "mode",
  "batchQueryContext",
  "treeState",
  "viewState",
] as const;
export type PersistentTab = Pick<
  SQLEditorTab,
  (typeof PERSISTENT_TAB_FIELDS)[number]
>;

export interface SQLEditorTabsState {
  /** Authoritative live tab objects keyed by id. */
  tabsById: Map<string, SQLEditorTab>;
  /** Persisted metadata that drives tab order and lightweight reload. */
  openTmpTabList: PersistentTab[];
  /** Currently active tab id; empty when no tabs are open. */
  currentTabId: string;

  setCurrentTabId: (id: string) => void;
  /** Rewrites the persisted tab order without touching individual tabs. */
  setOpenTabListOrder: (order: PersistentTab[]) => void;
  addTab: (payload?: Partial<SQLEditorTab>, beside?: boolean) => SQLEditorTab;
  cloneTab: (targetId: string, payload?: Partial<SQLEditorTab>) => SQLEditorTab;
  closeTab: (tabId: string) => void;
  updateTab: (
    id: string,
    payload: Partial<SQLEditorTab>
  ) => SQLEditorTab | undefined;
  updateCurrentTab: (
    payload: Partial<SQLEditorTab>
  ) => SQLEditorTab | undefined;
  updateBatchQueryContext: (
    payload: Partial<BatchQueryContext>
  ) => SQLEditorTab | undefined;
  removeDatabaseQueryContext: (params: {
    database: string;
    contextId: string;
  }) => SQLEditorDatabaseQueryContext | undefined;
  batchRemoveDatabaseQueryContext: (params: {
    database: string;
    contextIds: string[];
  }) => void;
  deleteDatabaseQueryContext: (database: string) => void;
  updateDatabaseQueryContext: (params: {
    database: string;
    contextId: string;
    context: Partial<SQLEditorDatabaseQueryContext>;
  }) => SQLEditorDatabaseQueryContext | undefined;
  initProject: (project: string) => Promise<void>;
  /** Reset all in-memory state. Test-only utility. */
  reset: () => void;
}

const safeRead = <T>(
  key: string,
  parse: (raw: unknown) => T | undefined,
  fallback: T
): T => {
  if (typeof window === "undefined") return fallback;
  try {
    const raw = window.localStorage.getItem(key);
    if (raw === null) return fallback;
    const parsed = JSON.parse(raw);
    const valid = parse(parsed);
    return valid === undefined ? fallback : valid;
  } catch {
    return fallback;
  }
};

const safeWrite = (key: string, value: unknown) => {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // Ignore quota / serialization errors.
  }
};

const isPersistentTabArray = (v: unknown): v is PersistentTab[] =>
  Array.isArray(v);

const currentScope = (): {
  wsScope: string;
  project: string;
  email: string;
} | null => {
  const project = getSQLEditorEditorState().project;
  if (!project) return null;
  // The current user is loaded into the app store during tab hydration
  // (see hydrateProjectTabs), so it's available by the time tabs are
  // persisted on user actions.
  const state = useAppStore.getState();
  const email = state.currentUser?.email ?? "";
  const wsScope = workspaceCacheScope(
    state.isSaaSMode(),
    state.currentUser?.workspace ?? ""
  );
  return { wsScope, project, email };
};

const persistOpenTabs = (openTabs: PersistentTab[]) => {
  const scope = currentScope();
  if (!scope) return;
  safeWrite(
    storageKeySqlEditorTabs(scope.wsScope, scope.project, scope.email),
    openTabs
  );
};

const persistCurrentTabId = (id: string) => {
  const scope = currentScope();
  if (!scope) return;
  safeWrite(
    storageKeySqlEditorCurrentTab(scope.wsScope, scope.project, scope.email),
    id
  );
};

const readOpenTabs = (
  wsScope: string,
  project: string,
  email: string
): PersistentTab[] =>
  safeRead<PersistentTab[]>(
    storageKeySqlEditorTabs(wsScope, project, email),
    (v) => (isPersistentTabArray(v) ? v : undefined),
    []
  );

export const useSQLEditorTabsStore: UseBoundStore<
  StoreApi<SQLEditorTabsState>
> = create<SQLEditorTabsState>()(
  immer((set, get) => ({
    tabsById: new Map(),
    openTmpTabList: [],
    currentTabId: "",

    setCurrentTabId(id) {
      set((s) => {
        s.currentTabId = id;
      });
      persistCurrentTabId(id);
    },

    setOpenTabListOrder(order) {
      set((s) => {
        s.openTmpTabList = order;
      });
      persistOpenTabs(order);
    },

    addTab(payload, beside = false) {
      const defaultTab: SQLEditorTab = {
        ...defaultSQLEditorTab(),
        ...omitBy(payload, isUndefined),
      };
      const { id } = defaultTab;

      set((s) => {
        let tab = s.tabsById.get(id);
        if (tab) {
          Object.assign(tab, omitBy(payload, isUndefined));
        } else {
          tab = defaultTab;
          s.tabsById.set(id, tab);
        }

        upsertOpenTabDraft(s, tab, beside);
        s.currentTabId = id;
      });

      persistOpenTabs(get().openTmpTabList);
      persistCurrentTabId(id);

      // Return the live (post-immer) tab reference so callers reading
      // fields immediately see the seeded values.
      return get().tabsById.get(id) ?? defaultTab;
    },

    cloneTab(targetId, payload) {
      const targetTab = get().tabsById.get(targetId);
      const cloned: Partial<SQLEditorTab> = {
        statement: targetTab?.statement,
        connection: cloneDeep(targetTab?.connection),
        treeState: cloneDeep(targetTab?.treeState),
        editorState: cloneDeep(targetTab?.editorState),
        batchQueryContext: cloneDeep(targetTab?.batchQueryContext),
        title: "",
        ...payload,
      };
      return get().addTab(cloned, true);
    },

    closeTab(tabId) {
      const position = get().openTmpTabList.findIndex(
        (item) => item.id === tabId
      );
      if (position < 0) return;

      const wasCurrent = tabId === get().currentTabId;

      set((s) => {
        s.openTmpTabList.splice(position, 1);
        s.tabsById.delete(tabId);
        if (wasCurrent) {
          const nextIndex = Math.min(position, s.openTmpTabList.length - 1);
          s.currentTabId = s.openTmpTabList[nextIndex]?.id ?? "";
        }
      });

      persistOpenTabs(get().openTmpTabList);
      if (wasCurrent) {
        persistCurrentTabId(get().currentTabId);
      }

      // Dynamic import avoids a static cycle with the web terminal
      // service module (which transitively re-imports this module).
      void import("@/react/stores/sqlEditor/webTerminal-service").then(
        ({ disposeWebTerminalQuerySession }) => {
          disposeWebTerminalQuerySession(tabId);
        }
      );
    },

    updateTab(id, payload) {
      if (!get().tabsById.has(id)) return undefined;
      set((s) => {
        const tab = s.tabsById.get(id);
        if (!tab) return;
        Object.assign(tab, payload);
        upsertOpenTabDraft(s, tab, false);
      });
      persistOpenTabs(get().openTmpTabList);
      return get().tabsById.get(id);
    },

    updateCurrentTab(payload) {
      const id = get().currentTabId;
      if (!id) return undefined;
      return get().updateTab(id, payload);
    },

    updateBatchQueryContext(payload) {
      const id = get().currentTabId;
      if (!id) return undefined;
      const existing = get().tabsById.get(id);
      if (!existing) return undefined;
      const previousCtx = existing.batchQueryContext;
      return get().updateTab(id, {
        batchQueryContext: {
          dataSourceType:
            previousCtx?.dataSourceType ?? DataSourceType.READ_ONLY,
          ...previousCtx,
          ...payload,
        },
      });
    },

    removeDatabaseQueryContext({ database, contextId }) {
      const id = get().currentTabId;
      const tab = get().tabsById.get(id);
      if (!tab?.databaseQueryContexts?.has(database)) return undefined;
      const contexts = tab.databaseQueryContexts.get(database)!;
      const index = contexts.findIndex((c) => c.id === contextId);
      if (index < 0) return undefined;
      let next: SQLEditorDatabaseQueryContext | undefined;
      set((s) => {
        const draftTab = s.tabsById.get(id);
        if (!draftTab?.databaseQueryContexts?.has(database)) return;
        const arr = draftTab.databaseQueryContexts.get(database)!;
        arr.splice(index, 1);
        next = arr[index] ?? arr[index - 1];
      });
      return next;
    },

    batchRemoveDatabaseQueryContext({ database, contextIds }) {
      if (contextIds.length === 0) return;
      const id = get().currentTabId;
      const tab = get().tabsById.get(id);
      if (!tab?.databaseQueryContexts?.has(database)) return;
      const target = new Set(contextIds);
      set((s) => {
        const draftTab = s.tabsById.get(id);
        if (!draftTab?.databaseQueryContexts?.has(database)) return;
        const existing = draftTab.databaseQueryContexts.get(database)!;
        const filtered = existing.filter((c) => !target.has(c.id));
        if (filtered.length !== existing.length) {
          draftTab.databaseQueryContexts.set(database, filtered);
        }
      });
    },

    deleteDatabaseQueryContext(database) {
      const id = get().currentTabId;
      const tab = get().tabsById.get(id);
      if (!tab?.databaseQueryContexts?.has(database)) return;
      set((s) => {
        const draftTab = s.tabsById.get(id);
        draftTab?.databaseQueryContexts?.delete(database);
      });
    },

    updateDatabaseQueryContext({ database, contextId, context }) {
      // Resolve the tab that OWNS this context by its globally-unique id
      // rather than assuming `currentTabId`. A query that completes after
      // the user switches tabs must still update its own tab's context —
      // the Vue original mutated the reactive context object directly,
      // which had the same cross-tab effect; immer freezes store objects
      // so React must route through this action, hence the lookup.
      const owner = locateDatabaseQueryContext(
        get().tabsById,
        database,
        contextId
      );
      if (!owner) return undefined;
      set((s) => {
        const target = s.tabsById
          .get(owner.tabId)
          ?.databaseQueryContexts?.get(database)?.[owner.index];
        if (!target) return;
        Object.assign(target, context);
      });
      return get()
        .tabsById.get(owner.tabId)
        ?.databaseQueryContexts?.get(database)?.[owner.index];
    },

    async initProject(project) {
      // Dedupe concurrent inits for the same project. During bootstrap
      // the editor-state subscriber fires a fire-and-forget initProject
      // while SQLEditorRouteShell separately awaits initProject and then
      // opens the URL route tab. Sharing one in-flight hydration prevents
      // the unawaited copy from resolving afterwards and replacing
      // `tabsById` — which would drop the just-opened route tab.
      if (_initProjectInFlight?.project === project) {
        return _initProjectInFlight.promise;
      }
      const promise = hydrateProjectTabs(project);
      _initProjectInFlight = { project, promise };
      try {
        await promise;
      } finally {
        if (_initProjectInFlight?.project === project) {
          _initProjectInFlight = undefined;
        }
      }
    },

    reset() {
      set((s) => {
        s.tabsById = new Map();
        s.openTmpTabList = [];
        s.currentTabId = "";
      });
    },
  }))
);

// Tracks an in-flight `initProject` so concurrent callers share one
// hydration (see the dedupe note in the action above).
let _initProjectInFlight:
  | { project: string; promise: Promise<void> }
  | undefined;

// Hydrates the persisted tabs for a project and commits them to the
// store. Extracted from the `initProject` action so the action can wrap
// it with in-flight dedup; uses the store's external setState/getState
// (immer-enabled) rather than the action closure's set/get.
const hydrateProjectTabs = async (project: string): Promise<void> => {
  await migrateDraftsFromCache(project);
  migrateTabViewState(project);

  // Ensure the current user is loaded so tab storage is scoped to the
  // right email — the SQL editor route doesn't otherwise populate it.
  await useAppStore.getState().loadCurrentUser();
  const state = useAppStore.getState();
  const email = state.currentUser?.email ?? "";
  const wsScope = workspaceCacheScope(
    state.isSaaSMode(),
    state.currentUser?.workspace ?? ""
  );

  const storedTabs = readOpenTabs(wsScope, project, email);

  const hydratedTabs: SQLEditorTab[] = [];
  const validPersistent: PersistentTab[] = [];
  const seen = new Set<string>();

  for (const persisted of storedTabs) {
    if (seen.has(persisted.id)) continue;
    if (!persisted.worksheet) continue;

    const worksheet = await useAppStore
      .getState()
      .getOrFetchWorksheetByName(persisted.worksheet, true);
    if (!worksheet) continue;

    const statement = getSheetStatement(worksheet);
    const connection = await extractWorksheetConnection(worksheet);

    const fullTab: SQLEditorTab = {
      ...defaultSQLEditorTab(),
      ...omitBy(persisted, isUndefined),
      connection,
      worksheet: worksheet.name,
      title: worksheet.title,
      statement,
      status: "CLEAN",
      databaseQueryContexts: undefined,
    };

    seen.add(persisted.id);
    validPersistent.push(persisted);
    hydratedTabs.push(fullTab);
  }

  useSQLEditorTabsStore.setState({
    tabsById: new Map(hydratedTabs.map((t) => [t.id, t])),
    openTmpTabList: validPersistent,
    currentTabId: head(validPersistent)?.id ?? "",
  });

  persistOpenTabs(validPersistent);
  persistCurrentTabId(useSQLEditorTabsStore.getState().currentTabId);
};

// Locates the tab + index owning a query context. Context ids are
// globally unique (uuid), so a context is found regardless of which tab
// is currently active — this is what lets an async query that completes
// after the user switched tabs still resolve to its own tab.
const locateDatabaseQueryContext = (
  tabsById: Map<string, SQLEditorTab>,
  database: string,
  contextId: string
): { tabId: string; index: number } | undefined => {
  for (const [tabId, tab] of tabsById) {
    const arr = tab.databaseQueryContexts?.get(database);
    if (!arr) continue;
    const index = arr.findIndex((c) => c.id === contextId);
    if (index >= 0) return { tabId, index };
  }
  return undefined;
};

const upsertOpenTabDraft = (
  state: { openTmpTabList: PersistentTab[]; currentTabId: string },
  tab: SQLEditorTab,
  beside: boolean
) => {
  const persistent = pick(tab, ...PERSISTENT_TAB_FIELDS) as PersistentTab;
  const position = state.openTmpTabList.findIndex((item) => item.id === tab.id);
  if (position >= 0) {
    Object.assign(state.openTmpTabList[position], persistent);
    return;
  }
  const currentPosition = state.openTmpTabList.findIndex(
    (item) => item.id === state.currentTabId
  );
  if (beside && currentPosition >= 0) {
    state.openTmpTabList.splice(currentPosition + 1, 0, persistent);
  } else {
    state.openTmpTabList.push(persistent);
  }
};

/**
 * Direct (non-React) accessor. Mirrors Zustand's `getState()`.
 */
export const getSQLEditorTabsState = (): SQLEditorTabsState =>
  useSQLEditorTabsStore.getState();

/**
 * Imperative read of the current tab. Use inside event handlers /
 * callbacks that need the tab at fire time rather than at render time.
 */
export const getCurrentSQLEditorTab = (): SQLEditorTab | undefined => {
  const s = getSQLEditorTabsState();
  return s.tabsById.get(s.currentTabId);
};

/**
 * Live read of a query context by (database, contextId) across all tabs.
 * The execute flow uses this to re-read a context after an async step
 * without assuming it still lives in the active tab.
 */
export const getDatabaseQueryContext = (
  database: string,
  contextId: string
): SQLEditorDatabaseQueryContext | undefined => {
  const owner = locateDatabaseQueryContext(
    getSQLEditorTabsState().tabsById,
    database,
    contextId
  );
  if (!owner) return undefined;
  return getSQLEditorTabsState()
    .tabsById.get(owner.tabId)
    ?.databaseQueryContexts?.get(database)?.[owner.index];
};

export const subscribeSQLEditorTabsState = (
  listener: (state: SQLEditorTabsState) => void
): (() => void) => useSQLEditorTabsStore.subscribe(listener);

/**
 * React selector hook over the tabs store.
 *
 * @example
 *   const currentTabId = useSQLEditorTabState((s) => s.currentTabId);
 */
export function useSQLEditorTabState<T>(
  selector: (state: SQLEditorTabsState) => T
): T {
  return useSQLEditorTabsStore(selector);
}

// Re-hydrate tabs whenever the active project changes. Mirrors the
// historical `watch(() => project.value, initProject)` side effect of
// the legacy Vue SQL editor tab store. Errors are intentionally swallowed —
// explicit callers (e.g. SQLEditorRouteShell) own user-facing failure
// reporting and may invoke `initProject` directly with full error
// handling.
let _lastInitializedProject: string | undefined;
subscribeSQLEditorEditorState((state) => {
  if (state.project === _lastInitializedProject) return;
  _lastInitializedProject = state.project;
  if (!state.project) return;
  getSQLEditorTabsState()
    .initProject(state.project)
    .catch((err) => {
      console.debug("[sql-editor] tab auto-init failed", err);
    });
});

// Test-only helper — clears the per-module init cursor so a fresh
// `setProject(...)` re-triggers `initProject` in tests.
export const __resetTabStoreProjectCursor = () => {
  _lastInitializedProject = undefined;
  _initProjectInFlight = undefined;
};

// ---------- Derived hooks ----------

/**
 * The currently selected tab. Returns `undefined` when no tabs are
 * open. Subscribes to BOTH `currentTabId` and `tabsById` so in-place
 * mutations of the active tab propagate through React.
 */
export const useCurrentSQLEditorTab = (): SQLEditorTab | undefined =>
  useSQLEditorTabsStore((s) => s.tabsById.get(s.currentTabId));

/**
 * Live ordered list of open tabs.
 *
 * `useShallow` is required: the `.map()` produces a fresh array every
 * call, which would fail `useSyncExternalStore`'s `Object.is` snapshot
 * check and trigger an infinite re-render loop. Shallow-equal arrays
 * (same tab references in the same order) get treated as unchanged.
 *
 * Persisted entries whose hydrated tab is missing from `tabsById` are
 * dropped rather than backfilled with a fresh `defaultSQLEditorTab()`
 * — that helper synthesises a new UUID on every call, which would
 * defeat shallow-equality and re-introduce the snapshot-cache warning
 * + infinite loop whenever a tab is briefly missing during hydration.
 */
export const useOpenTabList = (): SQLEditorTab[] =>
  useSQLEditorTabsStore(
    useShallow((s) =>
      s.openTmpTabList
        .map((persisted) => s.tabsById.get(persisted.id))
        .filter((tab): tab is SQLEditorTab => tab !== undefined)
    )
  );

export const useTabById = (tabId: string): SQLEditorTab | undefined =>
  useSQLEditorTabsStore((s) => s.tabsById.get(tabId));

export const isSQLEditorTabClosable = (tab: SQLEditorTab): boolean => {
  const open = getSQLEditorTabsState().openTmpTabList;
  if (open.length > 1) return true;
  if (open.length === 1) return !!tab.worksheet;
  return false;
};

/**
 * `true` when the current tab has no valid database connection.
 */
export const useIsDisconnected = (): boolean =>
  useSQLEditorTabsStore((s) => {
    const tab = s.tabsById.get(s.currentTabId);
    if (!tab) return true;
    return !isConnectedSQLEditorTab(tab);
  });

/**
 * Batch query mode flags. Derived from current tab + feature gates.
 */
export const useSupportBatchMode = (): boolean =>
  useSQLEditorTabsStore((s) => {
    const tab = s.tabsById.get(s.currentTabId);
    return tab?.mode !== "ADMIN";
  });

export const useIsInBatchMode = (): boolean =>
  useSQLEditorTabsStore((s) => {
    const tab = s.tabsById.get(s.currentTabId);
    if (!tab) return false;
    if (tab.mode === "ADMIN") return false;
    const appStore = useAppStore.getState();
    if (!appStore.hasFeature(PlanFeature.FEATURE_BATCH_QUERY)) return false;
    const ctx = tab.batchQueryContext;
    if (!ctx) return false;
    const { databaseGroups = [], databases = [] } = ctx;
    if (!appStore.hasFeature(PlanFeature.FEATURE_DATABASE_GROUPS)) {
      return databases.length > 1;
    }
    return databaseGroups.length > 0 || databases.length > 1;
  });
