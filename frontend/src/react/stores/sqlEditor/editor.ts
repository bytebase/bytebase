import { useSyncExternalStore } from "react";
import { create, type StoreApi, type UseBoundStore } from "zustand";
import { immer } from "zustand/middleware/immer";
import { useAppStore } from "@/react/stores/app";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { QueryOption_RedisRunCommandsOn } from "@/types/proto-es/v1/sql_service_pb";
import {
  STORAGE_KEY_SQL_EDITOR_REDIS_NODE,
  STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT,
  storageKeySqlEditorLastProject,
  workspaceCacheScope,
} from "@/utils/storage-keys";

export interface SQLEditorEditorState {
  project: string;
  projectContextReady: boolean;
  resultRowsLimit: number;
  redisCommandOption: QueryOption_RedisRunCommandsOn;
  isShowExecutingHint: boolean;
  executingHintDatabase: Database | undefined;

  setProject: (project: string) => void;
  setProjectContextReady: (ready: boolean) => void;
  setResultRowsLimit: (limit: number) => void;
  setRedisCommandOption: (option: QueryOption_RedisRunCommandsOn) => void;
  setShowExecutingHint: (show: boolean) => void;
  setExecutingHintDatabase: (db: Database | undefined) => void;
}

const DEFAULT_RESULT_ROWS_LIMIT = 1000;

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

// Workspace-scoped in SaaS so the last project of workspace A is never read
// after switching to workspace B (where it doesn't exist). Reads the app store
// at call time; if it isn't hydrated yet the scope falls back to "" and the
// project is re-resolved from the route anyway.
const lastProjectKey = () => {
  const state = useAppStore.getState();
  // Defensive: this runs at module-init (store may be pre-hydration) and the
  // app store is mocked in some tests, so isSaaSMode may be absent. Default to
  // the unscoped (self-host) key in that case.
  const isSaaS =
    typeof state?.isSaaSMode === "function" ? state.isSaaSMode() : false;
  return storageKeySqlEditorLastProject(
    workspaceCacheScope(isSaaS, state?.currentUser?.workspace ?? "")
  );
};

const readProject = () =>
  safeRead<string>(
    lastProjectKey(),
    (v) => (typeof v === "string" ? v : undefined),
    ""
  );

const readResultRowsLimit = () =>
  safeRead<number>(
    STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT,
    (v) =>
      typeof v === "number" && Number.isFinite(v) && v > 0 ? v : undefined,
    DEFAULT_RESULT_ROWS_LIMIT
  );

const readRedisOption = () =>
  safeRead<QueryOption_RedisRunCommandsOn>(
    STORAGE_KEY_SQL_EDITOR_REDIS_NODE,
    (v) =>
      v === QueryOption_RedisRunCommandsOn.SINGLE_NODE ||
      v === QueryOption_RedisRunCommandsOn.ALL_NODES
        ? v
        : undefined,
    QueryOption_RedisRunCommandsOn.SINGLE_NODE
  );

export const useSQLEditorEditorStore: UseBoundStore<
  StoreApi<SQLEditorEditorState>
> = create<SQLEditorEditorState>()(
  immer((set) => ({
    project: readProject(),
    projectContextReady: false,
    resultRowsLimit: readResultRowsLimit(),
    redisCommandOption: readRedisOption(),
    isShowExecutingHint: false,
    executingHintDatabase: undefined,

    setProject(project) {
      set((s) => {
        s.project = project;
        s.projectContextReady = true;
      });
      safeWrite(lastProjectKey(), project);
    },
    setProjectContextReady(ready) {
      set((s) => {
        s.projectContextReady = ready;
      });
    },
    setResultRowsLimit(limit) {
      set((s) => {
        s.resultRowsLimit = limit;
      });
      safeWrite(STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT, limit);
    },
    setRedisCommandOption(option) {
      set((s) => {
        s.redisCommandOption = option;
      });
      safeWrite(STORAGE_KEY_SQL_EDITOR_REDIS_NODE, option);
    },
    setShowExecutingHint(show) {
      set((s) => {
        s.isShowExecutingHint = show;
      });
    },
    setExecutingHintDatabase(db) {
      set((s) => {
        s.executingHintDatabase = db;
      });
    },
  }))
);

/**
 * Subscribe a React component to a slice of SQL editor editor state.
 *
 * @example
 *   const project = useSQLEditorEditorState((s) => s.project);
 *   const setProject = useSQLEditorEditorState((s) => s.setProject);
 */
export function useSQLEditorEditorState<T>(
  selector: (state: SQLEditorEditorState) => T
): T {
  return useSQLEditorEditorStore(selector);
}

/**
 * Read SQL editor editor state outside React (in actions, effects, or
 * non-React modules). Mirrors Zustand's `getState()`.
 */
export const getSQLEditorEditorState = (): SQLEditorEditorState =>
  useSQLEditorEditorStore.getState();

/**
 * Low-level subscription primitive for non-React consumers (Vue compat
 * shim, side-effect modules). Returns an unsubscribe function.
 */
export const subscribeSQLEditorEditorState = (
  listener: (state: SQLEditorEditorState) => void
): (() => void) => useSQLEditorEditorStore.subscribe(listener);

/**
 * useSyncExternalStore-friendly helpers for callers that want to read
 * the full state object without a selector (e.g. tests).
 */
export const useSQLEditorEditorSnapshot = (): SQLEditorEditorState =>
  useSyncExternalStore(
    useSQLEditorEditorStore.subscribe,
    useSQLEditorEditorStore.getState,
    useSQLEditorEditorStore.getState
  );
