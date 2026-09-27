import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { DatabaseFilter } from "@/lib/databaseFilter";
import { useSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import {
  buildTreeImpl,
  mapTreeNodeByType,
} from "@/modules/sql-editor/store/tree-utils";
import { useAppStore } from "@/stores/app";
import type {
  SQLEditorTreeNode,
  StatefulSQLEditorTreeFactor as StatefulFactor,
} from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  getDefaultPagination,
  isDatabaseV1Queryable,
  storageKeySqlEditorConnExpanded,
  workspaceCacheScope,
} from "@/utils";

const defaultEnvironmentFactor: StatefulFactor = {
  factor: "environment",
  disabled: false,
};

type FetchDataState = {
  readonly loading: boolean;
  readonly nextPageToken?: string;
};

type ExpandedState = {
  readonly initialized: boolean;
  readonly expandedKeys: string[];
};

const DEFAULT_EXPANDED: ExpandedState = {
  initialized: false,
  expandedKeys: [],
};

const readExpandedState = (storageKey: string): ExpandedState => {
  try {
    const raw = localStorage.getItem(storageKey);
    if (!raw) return DEFAULT_EXPANDED;
    const parsed = JSON.parse(raw) as Partial<ExpandedState>;
    return {
      initialized: !!parsed?.initialized,
      expandedKeys: Array.isArray(parsed?.expandedKeys)
        ? (parsed.expandedKeys as string[])
        : [],
    };
  } catch {
    return DEFAULT_EXPANDED;
  }
};

const writeExpandedState = (storageKey: string, state: ExpandedState) => {
  try {
    localStorage.setItem(storageKey, JSON.stringify(state));
  } catch {
    /* ignore quota + serialization errors */
  }
};

export type TreeByEnvironment = {
  readonly tree: SQLEditorTreeNode[];
  readonly expandedState: ExpandedState;
  readonly setExpandedKeys: (keys: string[]) => void;
  readonly buildTree: (showMissingQueryDatabases: boolean) => void;
  readonly prepareDatabases: (filter?: DatabaseFilter) => Promise<void>;
  readonly fetchDatabases: (filter?: DatabaseFilter) => Promise<void>;
  readonly fetchDataState: FetchDataState;
};

type Options = {
  /** Current user email — scoped into the localStorage key. */
  readonly email: string;
};

/**
 * Replaces frontend/src/views/sql-editor/ConnectionPanel/ConnectionPane/tree.ts.
 * React hook equivalent of `useSQLEditorTreeByEnvironment`. Owns the
 * environment-scoped database list, debounced paged fetch, computed tree
 * nodes, and localStorage-backed expand state.
 */
export function useSQLEditorTreeByEnvironment(
  environment: string,
  { email }: Options
): TreeByEnvironment {
  const fetchDatabaseList = useAppStore((s) => s.fetchDatabases);
  const project = useSQLEditorEditorState((s) => s.project);
  const getEnvironmentByName = useAppStore((s) => s.getEnvironmentByName);
  const isSaaS = useAppStore((s) => s.isSaaSMode());
  const workspace = useAppStore((s) => s.currentUser?.workspace ?? "");

  const storageKey = useMemo(
    () =>
      storageKeySqlEditorConnExpanded(
        workspaceCacheScope(isSaaS, workspace),
        environment,
        email
      ),
    [isSaaS, workspace, environment, email]
  );

  const [tree, setTree] = useState<SQLEditorTreeNode[]>([]);
  const databaseListRef = useRef<Database[]>([]);
  const [fetchDataState, setFetchDataState] = useState<FetchDataState>({
    loading: false,
  });
  const [expandedState, setExpandedState] = useState<ExpandedState>(() =>
    readExpandedState(storageKey)
  );

  // Re-read localStorage when the storage key changes (e.g. email rotation).
  useEffect(() => {
    setExpandedState(readExpandedState(storageKey));
  }, [storageKey]);

  const setExpandedKeys = useCallback(
    (keys: string[]) => {
      const next: ExpandedState = {
        initialized: true,
        expandedKeys: keys,
      };
      setExpandedState(next);
      writeExpandedState(storageKey, next);
    },
    [storageKey]
  );

  const fetchDatabasesImpl = useCallback(
    async (filter?: DatabaseFilter) => {
      setFetchDataState((prev) => ({ ...prev, loading: true }));
      const pageToken = fetchDataStateRef.current.nextPageToken;
      try {
        const { databases, nextPageToken } = await fetchDatabaseList({
          parent: project,
          pageToken,
          pageSize: getDefaultPagination(),
          filter: {
            ...filter,
            environment,
          },
        });
        databaseListRef.current = pageToken
          ? [...databaseListRef.current, ...databases]
          : [...databases];
        setFetchDataState({ loading: false, nextPageToken });
      } catch {
        databaseListRef.current = [];
        setFetchDataState({ loading: false });
      }
    },
    [fetchDatabaseList, project, environment]
  );

  // Keep a ref to the latest page token for refresh and load-more requests.
  const fetchDataStateRef = useRef(fetchDataState);
  useEffect(() => {
    fetchDataStateRef.current = fetchDataState;
  }, [fetchDataState]);

  const fetchDatabases = fetchDatabasesImpl;

  const prepareDatabases = useCallback(
    async (filter?: DatabaseFilter) => {
      setFetchDataState((prev) => ({ ...prev, nextPageToken: "" }));
      fetchDataStateRef.current = {
        ...fetchDataStateRef.current,
        nextPageToken: "",
      };
      await fetchDatabases(filter);
    },
    [fetchDatabases]
  );

  const buildTree = useCallback(
    (showMissingQueryDatabases: boolean) => {
      let list = [...databaseListRef.current];
      if (!showMissingQueryDatabases) {
        list = list.filter((db) => isDatabaseV1Queryable(db));
      }
      let next = buildTreeImpl(list, [defaultEnvironmentFactor.factor]);
      if (next.length === 0) {
        const env = getEnvironmentByName(environment);
        const rootNode = mapTreeNodeByType("environment", env, undefined);
        rootNode.children = [];
        next = [rootNode];
      }
      setTree(next);
    },
    [getEnvironmentByName, environment]
  );

  return {
    tree,
    expandedState,
    setExpandedKeys,
    buildTree,
    prepareDatabases,
    fetchDatabases,
    fetchDataState,
  };
}
