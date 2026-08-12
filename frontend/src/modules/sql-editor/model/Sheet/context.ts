import Emittery from "emittery";
import { enableMapSet } from "immer";
import { debounce, isEqual, sortBy } from "lodash-es";
import scrollIntoView from "scroll-into-view-if-needed";
import { create, type StoreApi, type UseBoundStore } from "zustand";
import { immer } from "zustand/middleware/immer";
import { useShallow } from "zustand/react/shallow";
import i18n from "@/lib/i18n";
import { extractSavedQueryConnection } from "@/lib/sqlEditorConnection";
import {
  getSQLEditorEditorState,
  subscribeSQLEditorEditorState,
  useSQLEditorEditorStore,
} from "@/modules/sql-editor/store/editor";
import {
  getSQLEditorTabsState,
  subscribeSQLEditorTabsState,
  useSQLEditorTabsStore,
} from "@/modules/sql-editor/store/tab";
import { useAppStore } from "@/stores/app";
import type { SQLEditorTab, SQLEditorTabMode } from "@/types";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import {
  type SavedQuery,
  SavedQuery_Visibility,
} from "@/types/proto-es/v1/saved_query_service_pb";
import {
  getDefaultPagination,
  getSheetStatement,
  isSavedQueryReadableV1,
  storageKeySqlEditorSavedQueryFolder,
  storageKeySqlEditorSavedQueryTree,
  workspaceCacheScope,
} from "@/utils";
import { escapeCELStringLiteral } from "@/utils/v1/cel";
import { isSubFolder } from "./folder";
import { type SheetViewMode, SheetViewModeList } from "./types";

// SavedQuery caches, folder sets, and the sheet-tree contain Map / Set
// values that immer needs to draft directly via mutation.
enableMapSet();

// ---- public types ----------------------------------------------------------

export interface SavedQueryLikeItem {
  name: string;
  title: string;
  folders: string[];
  type: "savedQuery" | "draft";
}

export interface SavedQueryFolderNode {
  key: string;
  label: string;
  editable: boolean;
  isLeaf?: boolean;
  empty?: boolean;
  loadMore?: boolean;
  loadMoreFolderKey?: string;
  savedQuery?: SavedQueryLikeItem;
  children: SavedQueryFolderNode[];
  [key: string]: unknown;
}

export interface SavedQueryFilter {
  keyword: string;
  onlyShowStarred: boolean;
  showMine: boolean;
  showShared: boolean;
  showDraft: boolean;
}

export interface FolderContext {
  rootPath: string;
  folders: string[];
  listSubFolders: (parent: string) => string[];
  ensureFolderPath: (path: string) => string;
  addFolder: (path: string) => string;
  removeFolder: (path: string) => void;
  moveFolder: (from: string, to: string) => void;
  mergeFolders: (paths: Set<string>) => void;
  replaceFolders: (paths: Set<string>) => void;
  isSubFolder: (args: {
    parent: string;
    path: string;
    dig: boolean;
  }) => boolean;
}

type SheetTreeEvents = Emittery<{
  "on-built": { viewMode: SheetViewMode };
}>;

export interface ViewContext {
  isLoading: boolean;
  isInitialized: boolean;
  sheetTree: SavedQueryFolderNode;
  folderTree: SavedQueryFolderNode;
  folderContext: FolderContext;
  events: SheetTreeEvents;
  fetchSheetList: () => Promise<void>;
  fetchNextPage: (folderKey?: string) => Promise<void>;
  fetchSavedQueriesByFolder: (folderKey: string) => Promise<void>;
  hasMore: boolean;
  hasMoreForFolder: (folderKey: string) => boolean;
  isFetchingNextPage: boolean;
  rebuildTree: () => void;
  getKeyForSavedQuery: (savedQuery: SavedQueryLikeItem) => string;
  getFoldersForSavedQuery: (path: string) => string[];
  getPathesForSavedQuery: (savedQuery: { folders: string[] }) => string[];
  getPwdForSavedQuery: (savedQuery: { folders: string[] }) => string;
}

export type SavedQueryFolderPathUpdate = {
  sourceFolder: string[];
  targetFolder: string[];
};

// ---- internal Zustand store ------------------------------------------------

const INITIAL_FILTER: SavedQueryFilter = {
  keyword: "",
  showShared: true,
  showMine: true,
  showDraft: true,
  onlyShowStarred: false,
};

interface ViewState {
  isLoading: boolean;
  isFetchingNextPage: boolean;
  isInitialized: boolean;
  sheetTree: SavedQueryFolderNode;
  folders: string[];
  savedQueryNames: string[];
  nextPageToken: string;
  folderNextPageTokens: Map<string, string>;
  fetchingFolderKeys: Set<string>;
  fetchedFolderKeys: Set<string>;
}

interface SheetContextState {
  filter: SavedQueryFilter;
  expandedKeys: Set<string>;
  selectedKeys: string[];
  editingNode: { node: SavedQueryFolderNode; rawLabel: string } | undefined;
  view: SheetViewMode;
  viewStates: Record<SheetViewMode, ViewState>;

  setFilter: (
    next: SavedQueryFilter | ((prev: SavedQueryFilter) => SavedQueryFilter)
  ) => void;
  setView: (view: SheetViewMode) => void;
  setExpandedKeys: (
    next: Set<string> | ((prev: Set<string>) => Set<string>)
  ) => void;
  setSelectedKeys: (next: string[]) => void;
  setEditingNode: (
    next: { node: SavedQueryFolderNode; rawLabel: string } | undefined
  ) => void;
  setViewIsLoading: (view: SheetViewMode, loading: boolean) => void;
  setViewIsFetchingNextPage: (view: SheetViewMode, loading: boolean) => void;
  setViewIsInitialized: (view: SheetViewMode, initialized: boolean) => void;
  setViewSheetTree: (view: SheetViewMode, tree: SavedQueryFolderNode) => void;
  setViewFolders: (view: SheetViewMode, folders: string[]) => void;
  setViewSavedQueryNames: (view: SheetViewMode, names: string[]) => void;
  setViewNextPageToken: (view: SheetViewMode, token: string) => void;
  setViewFolderNextPageToken: (
    view: SheetViewMode,
    folderKey: string,
    token: string
  ) => void;
  resetViewFolderPageState: (view: SheetViewMode) => void;
  setViewFolderFetching: (
    view: SheetViewMode,
    folderKey: string,
    fetching: boolean
  ) => void;
  setViewFolderFetched: (view: SheetViewMode, folderKey: string) => void;
  invalidateViewPageState: (
    view: SheetViewMode,
    folderKeys: Iterable<string>
  ) => void;
  moveViewFolderPageState: (
    view: SheetViewMode,
    fromPath: string,
    toPath: string,
    merge: boolean
  ) => void;
  /** Replace the entire state (used by project / user-scope reload). */
  hydrate: (next: Partial<SheetContextState>) => void;
}

const rootPathFor = (view: SheetViewMode) => `/${view}`;

const emptyRootNode = (view: SheetViewMode): SavedQueryFolderNode => ({
  key: rootPathFor(view),
  label: "",
  editable: false,
  isLeaf: false,
  children: [],
});

const emptyViewState = (view: SheetViewMode): ViewState => ({
  isLoading: false,
  isFetchingNextPage: false,
  isInitialized: false,
  sheetTree: emptyRootNode(view),
  folders: [rootPathFor(view)],
  savedQueryNames: [],
  nextPageToken: "",
  folderNextPageTokens: new Map<string, string>(),
  fetchingFolderKeys: new Set<string>(),
  fetchedFolderKeys: new Set<string>(),
});

const initialViewStates: Record<SheetViewMode, ViewState> = {
  my: emptyViewState("my"),
  shared: emptyViewState("shared"),
  draft: emptyViewState("draft"),
};

const useSheetContextStore: UseBoundStore<StoreApi<SheetContextState>> =
  create<SheetContextState>()(
    immer((set) => ({
      filter: { ...INITIAL_FILTER },
      expandedKeys: new Set<string>([
        rootPathFor("my"),
        rootPathFor("shared"),
        rootPathFor("draft"),
      ]),
      selectedKeys: [],
      editingNode: undefined,
      view: "my",
      viewStates: initialViewStates,

      setFilter(next) {
        set((s) => {
          s.filter = typeof next === "function" ? next(s.filter) : next;
        });
      },
      setView(view) {
        set((s) => {
          s.view = view;
        });
      },
      setExpandedKeys(next) {
        set((s) => {
          s.expandedKeys =
            typeof next === "function" ? next(s.expandedKeys) : next;
        });
      },
      setSelectedKeys(next) {
        set((s) => {
          s.selectedKeys = next;
        });
      },
      setEditingNode(next) {
        set((s) => {
          s.editingNode = next;
        });
      },
      setViewIsLoading(view, loading) {
        set((s) => {
          s.viewStates[view].isLoading = loading;
        });
      },
      setViewIsFetchingNextPage(view, loading) {
        set((s) => {
          s.viewStates[view].isFetchingNextPage = loading;
        });
      },
      setViewIsInitialized(view, initialized) {
        set((s) => {
          s.viewStates[view].isInitialized = initialized;
        });
      },
      setViewSheetTree(view, tree) {
        set((s) => {
          s.viewStates[view].sheetTree = tree;
        });
      },
      setViewFolders(view, folders) {
        set((s) => {
          s.viewStates[view].folders = folders;
        });
      },
      setViewSavedQueryNames(view, names) {
        set((s) => {
          s.viewStates[view].savedQueryNames = names;
        });
      },
      setViewNextPageToken(view, token) {
        set((s) => {
          s.viewStates[view].nextPageToken = token;
        });
      },
      setViewFolderNextPageToken(view, folderKey, token) {
        set((s) => {
          if (token) {
            s.viewStates[view].folderNextPageTokens.set(folderKey, token);
          } else {
            s.viewStates[view].folderNextPageTokens.delete(folderKey);
          }
        });
      },
      resetViewFolderPageState(view) {
        set((s) => {
          s.viewStates[view].folderNextPageTokens.clear();
          s.viewStates[view].fetchingFolderKeys.clear();
          s.viewStates[view].fetchedFolderKeys.clear();
        });
      },
      setViewFolderFetching(view, folderKey, fetching) {
        set((s) => {
          if (fetching) {
            s.viewStates[view].fetchingFolderKeys.add(folderKey);
          } else {
            s.viewStates[view].fetchingFolderKeys.delete(folderKey);
          }
        });
      },
      setViewFolderFetched(view, folderKey) {
        set((s) => {
          s.viewStates[view].fetchedFolderKeys.add(folderKey);
        });
      },
      invalidateViewPageState(view, folderKeys) {
        set((s) => {
          const viewState = s.viewStates[view];
          const rootPath = rootPathFor(view);
          const folderContext = getFolderContext(view);
          for (const folderKey of folderKeys) {
            const key = folderContext.ensureFolderPath(folderKey);
            if (key === rootPath) {
              viewState.nextPageToken = "";
              continue;
            }
            viewState.folderNextPageTokens.delete(key);
            viewState.fetchingFolderKeys.delete(key);
            viewState.fetchedFolderKeys.delete(key);
          }
        });
      },
      moveViewFolderPageState(view, fromPath, toPath, merge) {
        const moveKey = (key: string) => {
          if (key === fromPath) return toPath;
          if (isSubFolder({ parent: fromPath, path: key, dig: true })) {
            return key.replace(fromPath, toPath);
          }
          return key;
        };
        set((s) => {
          const viewState = s.viewStates[view];
          if (merge) {
            const shouldInvalidate = (key: string) =>
              key === fromPath ||
              key === toPath ||
              isSubFolder({ parent: fromPath, path: key, dig: true }) ||
              isSubFolder({ parent: toPath, path: key, dig: true });
            viewState.folderNextPageTokens = new Map(
              [...viewState.folderNextPageTokens.entries()].filter(
                ([key]) => !shouldInvalidate(key)
              )
            );
            viewState.fetchingFolderKeys = new Set(
              [...viewState.fetchingFolderKeys].filter(
                (key) => !shouldInvalidate(key)
              )
            );
            viewState.fetchedFolderKeys = new Set(
              [...viewState.fetchedFolderKeys].filter(
                (key) => !shouldInvalidate(key)
              )
            );
            return;
          }
          viewState.folderNextPageTokens = new Map(
            [...viewState.folderNextPageTokens.entries()].map(
              ([key, token]) => [moveKey(key), token]
            )
          );
          viewState.fetchingFolderKeys = new Set(
            [...viewState.fetchingFolderKeys].map(moveKey)
          );
          viewState.fetchedFolderKeys = new Set(
            [...viewState.fetchedFolderKeys].map(moveKey)
          );
        });
      },
      hydrate(next) {
        set((s) => {
          Object.assign(s, next);
        });
      },
    }))
  );

// ---- localStorage persistence ----------------------------------------------

const safeReadJSON = <T>(
  key: string,
  parse: (raw: unknown) => T | undefined
): T | undefined => {
  if (typeof window === "undefined") return undefined;
  try {
    const raw = window.localStorage.getItem(key);
    if (raw === null) return undefined;
    return parse(JSON.parse(raw));
  } catch {
    return undefined;
  }
};

const safeWriteJSON = (key: string, value: unknown) => {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // Ignore quota / serialization errors.
  }
};

const currentScope = (): {
  wsScope: string;
  project: string;
  email: string;
} | null => {
  const project = getSQLEditorEditorState().project;
  if (!project) return null;
  try {
    const state = useAppStore.getState();
    const email = state.currentUser?.email ?? "";
    const wsScope = workspaceCacheScope(
      state.isSaaSMode(),
      state.currentUser?.workspace ?? ""
    );
    return { wsScope, project, email };
  } catch {
    return null;
  }
};

const reloadFromStorage = () => {
  const scope = currentScope();
  if (!scope) return;

  const expandedArray = safeReadJSON<string[]>(
    storageKeySqlEditorSavedQueryTree(
      scope.wsScope,
      scope.project,
      scope.email
    ),
    (v) =>
      Array.isArray(v) && v.every((entry) => typeof entry === "string")
        ? (v as string[])
        : undefined
  );
  const expandedKeys = expandedArray
    ? new Set(expandedArray)
    : new Set<string>([
        rootPathFor("my"),
        rootPathFor("shared"),
        rootPathFor("draft"),
      ]);
  expandedKeys.add(rootPathFor("my"));
  expandedKeys.add(rootPathFor("shared"));
  expandedKeys.add(rootPathFor("draft"));

  const viewStates: Record<SheetViewMode, ViewState> = {
    my: emptyViewState("my"),
    shared: emptyViewState("shared"),
    draft: emptyViewState("draft"),
  };
  for (const view of SheetViewModeList) {
    viewStates[view].folders = sortBy([
      rootPathFor(view),
      ...readPersistedFolders(view),
    ]);
  }

  useSheetContextStore.getState().hydrate({
    filter: { ...INITIAL_FILTER },
    expandedKeys,
    selectedKeys: [],
    editingNode: undefined,
    viewStates,
  });
};

const persistExpandedKeys = (keys: Set<string>) => {
  const scope = currentScope();
  if (!scope) return;
  safeWriteJSON(
    storageKeySqlEditorSavedQueryTree(
      scope.wsScope,
      scope.project,
      scope.email
    ),
    [...keys]
  );
};

const persistedFolderStorageKey = (view: SheetViewMode): string | undefined => {
  const scope = currentScope();
  if (!scope) return undefined;
  return storageKeySqlEditorSavedQueryFolder(
    scope.wsScope,
    scope.project,
    view,
    scope.email
  );
};

const readPersistedFolders = (view: SheetViewMode): Set<string> => {
  const key = persistedFolderStorageKey(view);
  if (!key) return new Set();
  const folders = safeReadJSON<string[]>(key, (v) =>
    Array.isArray(v) && v.every((entry) => typeof entry === "string")
      ? (v as string[])
      : undefined
  );
  return new Set(
    (folders ?? []).map((folder) => ensureFolderPath(view, folder))
  );
};

const writePersistedFolders = (view: SheetViewMode, folders: Set<string>) => {
  const key = persistedFolderStorageKey(view);
  if (!key) return;
  const root = rootPathFor(view);
  safeWriteJSON(key, sortBy([...folders].filter((folder) => folder !== root)));
};

const addPersistedFolder = (view: SheetViewMode, path: string) => {
  const folders = readPersistedFolders(view);
  folders.add(ensureFolderPath(view, path));
  writePersistedFolders(view, folders);
};

const removePersistedFolder = (view: SheetViewMode, path: string) => {
  const folderContext = getFolderContext(view);
  const target = folderContext.ensureFolderPath(path);
  const folders = readPersistedFolders(view);
  const next = new Set(
    [...folders].filter(
      (folder) =>
        folder !== target &&
        !folderContext.isSubFolder({ parent: target, path: folder, dig: true })
    )
  );
  writePersistedFolders(view, next);
};

const movePersistedFolder = (view: SheetViewMode, from: string, to: string) => {
  const folderContext = getFolderContext(view);
  const fromPath = folderContext.ensureFolderPath(from);
  const toPath = folderContext.ensureFolderPath(to);
  const folders = readPersistedFolders(view);
  const next = new Set(
    [...folders].map((folder) => {
      if (folder === fromPath) return toPath;
      if (
        folderContext.isSubFolder({ parent: fromPath, path: folder, dig: true })
      ) {
        return folder.replace(fromPath, toPath);
      }
      return folder;
    })
  );
  writePersistedFolders(view, next);
};

// ---- per-view helpers + folder context -------------------------------------

const convertToSavedQueryLikeItem = (
  savedQuery: SavedQuery
): SavedQueryLikeItem => ({
  name: savedQuery.name,
  title: savedQuery.title,
  folders: savedQuery.folders,
  type: "savedQuery",
});

const rootLabelFor = (view: SheetViewMode): string => {
  switch (view) {
    case "my":
      return i18n.t("sheet.mine");
    case "shared":
      return i18n.t("sheet.shared");
    case "draft":
      return i18n.t("common.draft");
    default:
      return "";
  }
};

const rootTreeNodeFor = (view: SheetViewMode): SavedQueryFolderNode => ({
  isLeaf: false,
  children: [],
  key: rootPathFor(view),
  label: rootLabelFor(view),
  editable: false,
});

const getLoadMoreNodeKey = (folderKey: string) =>
  `__savedQuery_load_more__:${folderKey}`;

const ensureFolderPath = (view: SheetViewMode, path: string): string => {
  const root = rootPathFor(view);
  let p = path
    .split("/")
    .map((seg) => seg.trim())
    .filter((seg) => seg)
    .join("/");
  if (!p) return root;
  if (!p.startsWith("/")) p = `/${p}`;
  if (!p.startsWith(root)) p = `${root}${p}`;
  return p;
};

const buildFolderContext = (view: SheetViewMode): FolderContext => {
  const rootPath = rootPathFor(view);

  return {
    get rootPath() {
      return rootPath;
    },
    get folders() {
      return useSheetContextStore.getState().viewStates[view].folders;
    },
    listSubFolders(parent) {
      return useSheetContextStore
        .getState()
        .viewStates[view].folders.filter((path) =>
          isSubFolder({ parent, path, dig: false })
        );
    },
    ensureFolderPath(path) {
      return ensureFolderPath(view, path);
    },
    addFolder(path) {
      const newPath = ensureFolderPath(view, path);
      const current = useSheetContextStore.getState().viewStates[view].folders;
      const set = new Set(current);
      set.add(newPath);
      const next = sortBy([...set]);
      useSheetContextStore.getState().setViewFolders(view, next);
      addPersistedFolder(view, newPath);
      return newPath;
    },
    removeFolder(path) {
      const current = useSheetContextStore.getState().viewStates[view].folders;
      const next = current.filter(
        (value) =>
          !(
            value === path ||
            isSubFolder({ parent: path, path: value, dig: true })
          )
      );
      useSheetContextStore.getState().setViewFolders(view, next);
      removePersistedFolder(view, path);
    },
    moveFolder(from, to) {
      const fromPath = ensureFolderPath(view, from);
      const toPath = ensureFolderPath(view, to);
      const current = useSheetContextStore.getState().viewStates[view].folders;
      const merge = current.includes(toPath);
      const next = current.map((path) => {
        if (path === fromPath) return toPath;
        if (isSubFolder({ parent: fromPath, path, dig: true })) {
          return path.replace(fromPath, toPath);
        }
        return path;
      });
      const deduped = sortBy([...new Set(next)]);
      useSheetContextStore.getState().setViewFolders(view, deduped);
      useSheetContextStore
        .getState()
        .moveViewFolderPageState(view, fromPath, toPath, merge);
      movePersistedFolder(view, fromPath, toPath);
    },
    mergeFolders(paths) {
      const current = useSheetContextStore.getState().viewStates[view].folders;
      const set = new Set(current);
      for (const folder of paths.values()) {
        set.add(ensureFolderPath(view, folder));
      }
      const next = sortBy([...set]);
      if (
        next.length !== current.length ||
        !next.every((p, i) => current[i] === p)
      ) {
        useSheetContextStore.getState().setViewFolders(view, next);
      }
    },
    replaceFolders(paths) {
      const current = useSheetContextStore.getState().viewStates[view].folders;
      const set = new Set<string>([rootPath]);
      for (const folder of paths.values()) {
        set.add(ensureFolderPath(view, folder));
      }
      const next = sortBy([...set]);
      if (
        next.length !== current.length ||
        !next.every((p, i) => current[i] === p)
      ) {
        useSheetContextStore.getState().setViewFolders(view, next);
      }
    },
    isSubFolder,
  };
};

const folderContextCache: Partial<Record<SheetViewMode, FolderContext>> = {};
const getFolderContext = (view: SheetViewMode): FolderContext => {
  const existed = folderContextCache[view];
  if (existed) return existed;
  const ctx = buildFolderContext(view);
  folderContextCache[view] = ctx;
  return ctx;
};

const viewEvents: Partial<Record<SheetViewMode, SheetTreeEvents>> = {};
const getEvents = (view: SheetViewMode): SheetTreeEvents => {
  const existed = viewEvents[view];
  if (existed) return existed;
  const events: SheetTreeEvents = new Emittery();
  viewEvents[view] = events;
  return events;
};

const getPathesForSavedQuery = (
  view: SheetViewMode,
  savedQuery: { folders: string[] }
): string[] => {
  const folderContext = getFolderContext(view);
  const pathes = [folderContext.rootPath];
  let currentPath = folderContext.rootPath;
  for (const folder of savedQuery.folders) {
    currentPath = folderContext.ensureFolderPath(`${currentPath}/${folder}`);
    pathes.push(currentPath);
  }
  return pathes;
};

const getPwdForSavedQuery = (
  view: SheetViewMode,
  savedQuery: { folders: string[] }
): string =>
  getFolderContext(view).ensureFolderPath(savedQuery.folders.join("/"));

const getKeyForSavedQuery = (
  view: SheetViewMode,
  savedQuery: SavedQueryLikeItem
): string =>
  [
    getPwdForSavedQuery(view, savedQuery),
    `bytebase-${savedQuery.type}-${savedQuery.name.split("/").at(-1)}.sql`,
  ].join("/");

const getFoldersForSavedQuery = (
  view: SheetViewMode,
  path: string
): string[] => {
  const root = getFolderContext(view).rootPath;
  const segs = path.replace(root, "").split("/");
  if (segs.slice(-1)[0]?.endsWith(".sql")) segs.pop();
  return segs.map((p) => p.trim()).filter((p) => p);
};

const buildTree = (
  view: SheetViewMode,
  parent: SavedQueryFolderNode,
  savedQueriesByFolder: Map<string, SavedQueryLikeItem[]>,
  hideEmpty: boolean,
  includeLoadMore = true
): SavedQueryFolderNode => {
  const folderContext = getFolderContext(view);
  const subfolders: SavedQueryFolderNode[] = folderContext
    .listSubFolders(parent.key)
    .map((folder) => ({
      isLeaf: false,
      children: [],
      key: folder,
      label: folder.split("/").slice(-1)[0],
      editable: true,
    }));

  let empty = true;
  for (const childNode of subfolders) {
    const subtree = buildTree(
      view,
      childNode,
      savedQueriesByFolder,
      hideEmpty,
      includeLoadMore
    );
    if (!subtree.empty || !hideEmpty) {
      parent.children.push(subtree);
    }
    if (!subtree.empty) {
      empty = false;
    }
  }

  const sheets = (
    savedQueriesByFolder.get(parent.key) ?? []
  ).map<SavedQueryFolderNode>((savedQuery) => ({
    isLeaf: true,
    key: getKeyForSavedQuery(view, savedQuery),
    label: savedQuery.title,
    savedQuery,
    editable: true,
    children: [],
  }));

  parent.children.push(...sheets);
  const viewState = useSheetContextStore.getState().viewStates[view];
  const isRoot = parent.key === folderContext.rootPath;
  const hasMore = isRoot
    ? !!viewState.nextPageToken
    : viewState.folderNextPageTokens.has(parent.key);
  if (includeLoadMore && hasMore) {
    const loadMoreNode: SavedQueryFolderNode = {
      key: getLoadMoreNodeKey(parent.key),
      label: i18n.t("common.load-more"),
      editable: false,
      isLeaf: true,
      loadMore: true,
      loadMoreFolderKey: parent.key,
      children: [],
    };
    parent.children.push(loadMoreNode);
  }
  parent.empty = sheets.length === 0 && empty;
  if (parent.key !== folderContext.rootPath) {
    parent.isLeaf = parent.children.length === 0;
  }
  return parent;
};

const savedQueriesForView = (view: SheetViewMode): SavedQuery[] => {
  if (view !== "my" && view !== "shared") return [];
  const filter = useSheetContextStore.getState().filter;
  const project = getSQLEditorEditorState().project;
  // SQLEditorLayout awaits `loadCurrentUser()` in its bootstrap, so by the
  // time saved queries land here the app-store `currentUser` is populated.
  // Empty `email` falls through to creator `"users/"`, which matches no
  // saved queries — they'd render as Shared rather than Mine, same fallback
  // the previous Pinia path had.
  const email = useAppStore.getState().currentUser?.email ?? "";
  const creator = `users/${email}`;
  const appState = useAppStore.getState();
  let list = useSheetContextStore
    .getState()
    .viewStates[view].savedQueryNames.map((name) =>
      appState.getSavedQueryByName(name)
    )
    .filter((sheet): sheet is SavedQuery => {
      if (!sheet) return false;
      if (sheet.project !== project) return false;
      const mine = sheet.creator === creator;
      return view === "my" ? mine : !mine;
    });
  if (filter.onlyShowStarred) {
    list = list.filter((sheet) => sheet.starred);
  }
  return list;
};

const sheetLikeItemsForView = (view: SheetViewMode): SavedQueryLikeItem[] => {
  if (view === "draft") {
    const tabsState = getSQLEditorTabsState();
    return tabsState.openTmpTabList
      .map((p) => tabsState.tabsById.get(p.id))
      .filter((tab): tab is SQLEditorTab => !!tab && !tab.savedQuery)
      .map((tab) => ({
        name: tab.id,
        title: tab.title,
        folders: [],
        type: "draft" as const,
      }));
  }
  return savedQueriesForView(view).map(convertToSavedQueryLikeItem);
};

const rebuildTreeImpl = (view: SheetViewMode) => {
  const folderContext = getFolderContext(view);

  const folderPaths = new Set<string>();
  const savedQueriesByFolder = new Map<string, SavedQueryLikeItem[]>();

  for (const savedQuery of sheetLikeItemsForView(view)) {
    for (const path of getPathesForSavedQuery(view, savedQuery)) {
      folderPaths.add(path);
    }
    const pwd = getPwdForSavedQuery(view, savedQuery);
    if (!savedQueriesByFolder.has(pwd)) savedQueriesByFolder.set(pwd, []);
    savedQueriesByFolder.get(pwd)!.push(savedQuery);
  }

  folderContext.mergeFolders(folderPaths);

  const root: SavedQueryFolderNode = {
    ...rootTreeNodeFor(view),
    label: rootLabelFor(view),
    key: folderContext.rootPath,
  };
  const tree = buildTree(view, root, savedQueriesByFolder, false);
  useSheetContextStore.getState().setViewSheetTree(view, tree);
  getEvents(view).emit("on-built", { viewMode: view });
};

const rebuildTreeDebounced: Partial<Record<SheetViewMode, () => void>> = {};
const getRebuildTreeFn = (view: SheetViewMode): (() => void) => {
  const existed = rebuildTreeDebounced[view];
  if (existed) return existed;
  const debounced = debounce(
    () => rebuildTreeImpl(view),
    DEBOUNCE_SEARCH_DELAY
  );
  rebuildTreeDebounced[view] = debounced;
  return debounced;
};

const fetchSheetListFor = async (view: SheetViewMode) => {
  const state = useSheetContextStore.getState();
  state.setViewIsLoading(view, true);
  try {
    state.resetViewFolderPageState(view);
    await fetchSavedQueryFoldersForView(view);
    const { savedQueries, nextPageToken } = await fetchSavedQueriesPage(
      view,
      "",
      [`folder == ""`]
    );
    state.setViewSavedQueryNames(
      view,
      savedQueries.map((savedQuery) => savedQuery.name)
    );
    state.setViewNextPageToken(view, nextPageToken);
    rebuildTreeImpl(view);
    state.setViewIsInitialized(view, true);
  } finally {
    state.setViewIsLoading(view, false);
  }
};

const fetchNextSheetPageFor = async (
  view: SheetViewMode,
  folderKey?: string
) => {
  const state = useSheetContextStore.getState();
  const viewState = state.viewStates[view];
  const folderContext = getFolderContext(view);
  const key = folderKey
    ? folderContext.ensureFolderPath(folderKey)
    : rootPathFor(view);
  const pageToken =
    key === folderContext.rootPath
      ? viewState.nextPageToken
      : viewState.folderNextPageTokens.get(key) || "";
  if (
    (view !== "my" && view !== "shared") ||
    !pageToken ||
    viewState.isFetchingNextPage ||
    viewState.isLoading
  ) {
    return;
  }

  state.setViewIsFetchingNextPage(view, true);
  try {
    const { savedQueries, nextPageToken } = await fetchSavedQueriesPage(
      view,
      pageToken,
      [folderFilterForKey(view, key)]
    );
    const names = new Set(
      useSheetContextStore.getState().viewStates[view].savedQueryNames
    );
    for (const savedQuery of savedQueries) {
      names.add(savedQuery.name);
    }
    state.setViewSavedQueryNames(view, [...names]);
    if (key === folderContext.rootPath) {
      state.setViewNextPageToken(view, nextPageToken);
    } else {
      state.setViewFolderNextPageToken(view, key, nextPageToken);
    }
    rebuildTreeImpl(view);
  } finally {
    state.setViewIsFetchingNextPage(view, false);
  }
};

const folderFilterForKey = (view: SheetViewMode, folderKey: string): string => {
  const folderContext = getFolderContext(view);
  const key = folderContext.ensureFolderPath(folderKey);
  if (key === folderContext.rootPath) {
    return `folder == ""`;
  }
  const folder = getFoldersForSavedQuery(view, key).join("/");
  return `folder == "${escapeCELStringLiteral(folder)}"`;
};

const savedQuerySearchFilters = (): string[] => {
  const filter = useSheetContextStore.getState().filter;
  const filters: string[] = [];
  const keyword = filter.keyword.trim().toLowerCase();
  if (keyword) {
    filters.push(`title.contains("${escapeCELStringLiteral(keyword)}")`);
  }
  if (filter.onlyShowStarred) {
    filters.push("starred == true");
  }
  return filters;
};

const sheetFilterForView = (
  view: SheetViewMode,
  extraFilters: string[] = [],
  includeDisplayFilters = true
): string => {
  const email = useAppStore.getState().currentUser?.email ?? "";
  const baseFilters =
    view === "my"
      ? [`creator == "users/${escapeCELStringLiteral(email)}"`]
      : [
          `creator != "users/${escapeCELStringLiteral(email)}"`,
          `visibility in ["${SavedQuery_Visibility[SavedQuery_Visibility.PROJECT_READ]}","${SavedQuery_Visibility[SavedQuery_Visibility.PROJECT_WRITE]}"]`,
        ];
  return [
    ...baseFilters,
    ...extraFilters,
    ...(includeDisplayFilters ? savedQuerySearchFilters() : []),
  ].join(" && ");
};

const fetchSavedQueriesPage = async (
  view: SheetViewMode,
  pageToken: string,
  extraFilters: string[] = []
): Promise<{ savedQueries: SavedQuery[]; nextPageToken: string }> => {
  if (view !== "my" && view !== "shared") {
    return { savedQueries: [], nextPageToken: "" };
  }
  const sheetStore = useAppStore.getState();
  const project = getSQLEditorEditorState().project;
  const filter = sheetFilterForView(view, extraFilters);
  return sheetStore.fetchSavedQueryList(project, filter, {
    pageSize: getDefaultPagination(),
    pageToken,
  });
};

const fetchSavedQueryFoldersForView = async (view: SheetViewMode) => {
  if (view !== "my" && view !== "shared") {
    return;
  }
  const project = getSQLEditorEditorState().project;
  const paths = await useAppStore.getState().listSavedQueryFolders(project);
  const folderPaths = new Set<string>();
  for (const { folders, category } of paths) {
    if (category !== view) {
      continue;
    }
    for (const path of getPathesForSavedQuery(view, { folders })) {
      folderPaths.add(path);
    }
  }
  for (const path of readPersistedFolders(view)) {
    folderPaths.add(path);
  }
  getFolderContext(view).replaceFolders(folderPaths);
};

const fetchSavedQueriesByFolder = async (
  view: SheetViewMode,
  folderKey: string
) => {
  if (view !== "my" && view !== "shared") {
    return;
  }
  const folderContext = getFolderContext(view);
  const key = folderContext.ensureFolderPath(folderKey);
  if (key === folderContext.rootPath) {
    return;
  }

  const state = useSheetContextStore.getState();
  const viewState = state.viewStates[view];
  if (
    viewState.fetchingFolderKeys.has(key) ||
    viewState.fetchedFolderKeys.has(key)
  ) {
    return;
  }

  const folder = getFoldersForSavedQuery(view, key).join("/");
  if (!folder) {
    return;
  }

  state.setViewFolderFetching(view, key, true);
  try {
    const { savedQueries, nextPageToken } = await fetchSavedQueriesPage(
      view,
      "",
      [`folder == "${escapeCELStringLiteral(folder)}"`]
    );
    const names = new Set(
      useSheetContextStore.getState().viewStates[view].savedQueryNames
    );
    for (const savedQuery of savedQueries) {
      names.add(savedQuery.name);
    }
    useSheetContextStore.getState().setViewSavedQueryNames(view, [...names]);
    useSheetContextStore
      .getState()
      .setViewFolderNextPageToken(view, key, nextPageToken);
    useSheetContextStore.getState().setViewFolderFetched(view, key);
    rebuildTreeImpl(view);
  } finally {
    useSheetContextStore.getState().setViewFolderFetching(view, key, false);
  }
};

const buildViewContext = (view: SheetViewMode): ViewContext => {
  const folderContext = getFolderContext(view);
  return {
    get isLoading() {
      return useSheetContextStore.getState().viewStates[view].isLoading;
    },
    get isFetchingNextPage() {
      return useSheetContextStore.getState().viewStates[view]
        .isFetchingNextPage;
    },
    get hasMore() {
      return !!useSheetContextStore.getState().viewStates[view].nextPageToken;
    },
    hasMoreForFolder(folderKey) {
      const key = folderContext.ensureFolderPath(folderKey);
      return !!useSheetContextStore
        .getState()
        .viewStates[view].folderNextPageTokens.get(key);
    },
    get isInitialized() {
      return useSheetContextStore.getState().viewStates[view].isInitialized;
    },
    get sheetTree() {
      return useSheetContextStore.getState().viewStates[view].sheetTree;
    },
    get folderTree() {
      const root: SavedQueryFolderNode = {
        ...rootTreeNodeFor(view),
        key: folderContext.rootPath,
      };
      return buildTree(view, root, new Map(), false, false);
    },
    folderContext,
    events: getEvents(view),
    fetchSheetList: () => fetchSheetListFor(view),
    fetchNextPage: (folderKey) => fetchNextSheetPageFor(view, folderKey),
    fetchSavedQueriesByFolder: (folderKey) =>
      fetchSavedQueriesByFolder(view, folderKey),
    rebuildTree: () => getRebuildTreeFn(view)(),
    getKeyForSavedQuery: (ws) => getKeyForSavedQuery(view, ws),
    getFoldersForSavedQuery: (path) => getFoldersForSavedQuery(view, path),
    getPathesForSavedQuery: (ws) => getPathesForSavedQuery(view, ws),
    getPwdForSavedQuery: (ws) => getPwdForSavedQuery(view, ws),
  };
};

const viewContextCache: Partial<Record<SheetViewMode, ViewContext>> = {};
const getViewContext = (view: SheetViewMode): ViewContext => {
  const existed = viewContextCache[view];
  if (existed) return existed;
  const ctx = buildViewContext(view);
  viewContextCache[view] = ctx;
  return ctx;
};

// ---- side effects (initialized lazily on first use) ------------------------

const isSavedQueryCreator = (savedQuery: { creator: string }) => {
  const email = useAppStore.getState().currentUser?.email;
  if (!email) return false;
  return savedQuery.creator === `users/${email}`;
};

const viewForSavedQuery = (
  savedQuery: SavedQuery
): SheetViewMode | undefined => {
  const project = getSQLEditorEditorState().project;
  if (savedQuery.project !== project) return undefined;
  if (isSavedQueryCreator(savedQuery)) return "my";
  if (
    savedQuery.visibility === SavedQuery_Visibility.PROJECT_READ ||
    savedQuery.visibility === SavedQuery_Visibility.PROJECT_WRITE
  ) {
    return "shared";
  }
  return undefined;
};

const addNewSavedQueriesToViewMembership = (
  savedQueriesByKey: Record<string, SavedQuery>,
  prevSavedQueriesByKey: Record<string, SavedQuery>
) => {
  const prevNames = new Set(
    Object.values(prevSavedQueriesByKey).map((savedQuery) => savedQuery.name)
  );
  for (const savedQuery of Object.values(savedQueriesByKey)) {
    if (prevNames.has(savedQuery.name)) continue;
    const view = viewForSavedQuery(savedQuery);
    if (!view) continue;
    const state = useSheetContextStore.getState();
    const viewState = state.viewStates[view];
    if (
      !viewState.isInitialized ||
      viewState.savedQueryNames.includes(savedQuery.name)
    ) {
      continue;
    }
    state.setViewSavedQueryNames(view, [
      ...viewState.savedQueryNames,
      savedQuery.name,
    ]);
  }
};

const batchUpdateSavedQueryFolders = async (
  savedQueries: { name: string; folders: string[] }[]
): Promise<void> => {
  const affectedFolderKeysByView = new Map<SheetViewMode, Set<string>>();
  const addAffectedFolderKey = (view: SheetViewMode, folderKey: string) => {
    const keys = affectedFolderKeysByView.get(view) ?? new Set<string>();
    keys.add(folderKey);
    affectedFolderKeysByView.set(view, keys);
  };

  const requestByKey = new Map<
    string,
    { parent: string; folders: string[]; names: string[] }
  >();
  for (const savedQuery of savedQueries) {
    const current = useAppStore.getState().getSavedQueryByName(savedQuery.name);
    const view = current ? viewForSavedQuery(current) : undefined;
    if (current && (view === "my" || view === "shared")) {
      addAffectedFolderKey(view, getPwdForSavedQuery(view, current));
      addAffectedFolderKey(view, getPwdForSavedQuery(view, savedQuery));
    }

    const index = savedQuery.name.lastIndexOf("/savedQueries/");
    if (index < 0) {
      continue;
    }
    const parent = savedQuery.name.slice(0, index);
    const key = JSON.stringify([parent, savedQuery.folders]);
    const request = requestByKey.get(key);
    if (request) {
      request.names.push(savedQuery.name);
    } else {
      requestByKey.set(key, {
        parent,
        folders: savedQuery.folders,
        names: [savedQuery.name],
      });
    }
  }
  const requests = [...requestByKey.values()];
  if (requests.length === 0) return;
  await useAppStore.getState().batchUpdateSavedQueryOrganizers(
    requests.map((request) => ({
      parent: request.parent,
      filter: `name in [${request.names
        .map((name) => '"' + escapeCELStringLiteral(name) + '"')
        .join(",")}]`,
      organizer: {
        folders: request.folders,
      },
      updateMask: ["folders"],
    }))
  );
  for (const [view, folderKeys] of affectedFolderKeysByView) {
    useSheetContextStore.getState().invalidateViewPageState(view, folderKeys);
    rebuildTreeImpl(view);
  }
};

const batchUpdateSavedQueryFolderPaths = async (
  view: SheetViewMode,
  updates: SavedQueryFolderPathUpdate[]
): Promise<void> => {
  if (view !== "my" && view !== "shared") return;
  if (updates.length === 0) return;
  const project = getSQLEditorEditorState().project;
  const sortedUpdates = [...updates].sort(
    (a, b) => b.sourceFolder.length - a.sourceFolder.length
  );
  await useAppStore.getState().batchUpdateSavedQueryOrganizers(
    sortedUpdates.map((update) => ({
      parent: project,
      filter: sheetFilterForView(
        view,
        [
          `folder == "${escapeCELStringLiteral(update.sourceFolder.join("/"))}"`,
        ],
        false
      ),
      organizer: {
        folders: update.targetFolder,
      },
      updateMask: ["folders"],
    }))
  );
};

let _watchersBound = false;
let _lastProject = "";
let _lastTabKey = "";
let _lastDraftSig = "";

// Signature of the draft view's source data — the open tabs that have
// no saved query, keyed by id + title. The "draft" tree derives from
// these (see `sheetLikeItemsForView`), so the tree must rebuild when a
// draft tab is opened, closed, or renamed.
const computeDraftSignature = (
  tabsState: ReturnType<typeof getSQLEditorTabsState>
): string =>
  tabsState.openTmpTabList
    .map((persisted) => {
      const tab = tabsState.tabsById.get(persisted.id);
      if (!tab || tab.savedQuery) return "";
      return `${tab.id}:${tab.title}`;
    })
    .filter((s) => s)
    .join("|");

const bindWatchers = () => {
  if (_watchersBound) return;
  _watchersBound = true;

  // Subscribe handlers immediately — these only register listeners and
  // don't mutate any store, so they're safe to invoke during a React
  // render. The initial state hydration is deferred to a microtask so
  // the calling component finishes its current render before any
  // setState lands.
  subscribeSQLEditorEditorState((state) => {
    if (state.project === _lastProject) return;
    _lastProject = state.project;
    reloadFromStorage();
    rebuildTreeImpl("my");
    rebuildTreeImpl("shared");
    rebuildTreeImpl("draft");
  });

  useSheetContextStore.subscribe((state, prev) => {
    if (state.expandedKeys !== prev.expandedKeys)
      persistExpandedKeys(state.expandedKeys);

    if (
      state.filter.keyword !== prev.filter.keyword ||
      state.filter.onlyShowStarred !== prev.filter.onlyShowStarred
    ) {
      for (const view of ["my", "shared"] as const) {
        if (state.viewStates[view].isInitialized) {
          void fetchSheetListFor(view);
        }
      }
    }

    // Rebuild a view's tree when its folder set changes (add / remove /
    // move folder). Mirrors the Vue `watch(folderContext.folders,
    // rebuildTree)`. Debounced + idempotent `mergeFolders` inside the
    // rebuild keeps this from looping: the rebuild only writes folders
    // back when a saved query introduces a brand-new path, which settles
    // after one extra pass.
    for (const view of SheetViewModeList) {
      if (state.viewStates[view].folders !== prev.viewStates[view].folders) {
        getRebuildTreeFn(view)();
      }
    }
  });

  subscribeSQLEditorTabsState((tabsState) => {
    // Current-tab change → update tree selection + scroll-into-view.
    const tab = tabsState.tabsById.get(tabsState.currentTabId);
    const tabKey = `${tab?.id ?? ""}|${tab?.savedQuery ?? ""}`;
    if (tabKey !== _lastTabKey) {
      _lastTabKey = tabKey;
      onCurrentTabChanged(tab);
    }

    // Draft-tab set / title change → rebuild the draft tree. The draft
    // view's `sheetLikeItemList` is derived from open tabs without a
    // saved query, so adding / closing / renaming a draft must refresh it.
    // Mirrors the Vue `watch(sheetLikeItemList, rebuildTree)` for the
    // draft view context.
    const draftSig = computeDraftSignature(tabsState);
    if (draftSig !== _lastDraftSig) {
      _lastDraftSig = draftSig;
      getRebuildTreeFn("draft")();
    }
  });

  // Rebuild the saved query-backed trees whenever the saved query cache
  // mutates — a fetch, a title patch (e.g. renamed from the editor tab),
  // a star toggle, or a delete. Mirrors the Vue
  // `watch(sheetLikeItemList, rebuildTree)` that drove the tree off the
  // saved query list. The "draft" view is tab-derived, not saved query-
  // derived, so it doesn't need this. Debounced rebuilds coalesce the
  // burst of cache writes a single fetch produces.
  useAppStore.subscribe((state, prev) => {
    if (state.savedQueriesByKey === prev.savedQueriesByKey) return;
    addNewSavedQueriesToViewMembership(
      state.savedQueriesByKey,
      prev.savedQueriesByKey
    );
    getRebuildTreeFn("my")();
    getRebuildTreeFn("shared")();
  });

  // Synchronous initial hydration. Runs once during the SQLEditorLayout
  // render via `provideSheetContext()`; descendants render AFTER this
  // and so their `useSyncExternalStore` snapshots match the already-
  // hydrated state. Deferring this to `queueMicrotask` caused the
  // snapshot to flip between render and effect-mount, which made React
  // force a re-render in a loop with strict-mode double-mounts.
  //
  // Seed `_lastProject` from the current editor state so the
  // `subscribeSQLEditorEditorState` listener above doesn't fire a
  // redundant reload on the next unrelated editor mutation.
  _lastProject = getSQLEditorEditorState().project;
  reloadFromStorage();
};

const onCurrentTabChanged = (tab: SQLEditorTab | undefined) => {
  useSheetContextStore.getState().setSelectedKeys([]);
  if (!tab) return;

  let view: SheetViewMode = "draft";
  let savedQueryLikeItem: SavedQueryLikeItem | undefined;
  const tabId = tab.id;
  const savedQueryName = tab.savedQuery;

  if (savedQueryName) {
    const savedQuery = useAppStore
      .getState()
      .getSavedQueryByName(savedQueryName);
    if (!savedQuery) return;
    savedQueryLikeItem = {
      name: savedQuery.name,
      title: savedQuery.title,
      folders: savedQuery.folders,
      type: "savedQuery",
    };
    view = isSavedQueryCreator(savedQuery) ? "my" : "shared";
  } else {
    savedQueryLikeItem = {
      name: tabId,
      folders: [],
      title: "",
      type: "draft",
    };
  }

  const viewCtx = getViewContext(view);
  const key = viewCtx.getKeyForSavedQuery(savedQueryLikeItem);
  useSheetContextStore.getState().setSelectedKeys([key]);

  const expanded = new Set(useSheetContextStore.getState().expandedKeys);
  for (const path of viewCtx.getPathesForSavedQuery(savedQueryLikeItem)) {
    expanded.add(path);
  }
  useSheetContextStore.getState().setExpandedKeys(expanded);

  // Defer DOM scroll until after React paints.
  queueMicrotask(() => {
    const dom = document.querySelector(`[data-item-key="${key}"]`);
    if (dom) {
      scrollIntoView(dom, {
        scrollMode: "if-needed",
        block: "nearest",
      });
    }
  });
};

// ---- public hook API -------------------------------------------------------

export interface SheetContext {
  filter: SavedQueryFilter;
  filterChanged: boolean;
  expandedKeys: Set<string>;
  selectedKeys: string[];
  editingNode: { node: SavedQueryFolderNode; rawLabel: string } | undefined;
  view: SheetViewMode;
  viewContexts: Record<SheetViewMode, ViewContext>;
  isSavedQueryCreator: (savedQuery: { creator: string }) => boolean;
  batchUpdateSavedQueryFolders: (
    savedQueries: { name: string; folders: string[] }[]
  ) => Promise<void>;
  batchUpdateSavedQueryFolderPaths: (
    view: SheetViewMode,
    updates: SavedQueryFolderPathUpdate[]
  ) => Promise<void>;
  getContextByView: (view: SheetViewMode) => ViewContext;
  setFilter: SheetContextState["setFilter"];
  setView: SheetContextState["setView"];
  setExpandedKeys: SheetContextState["setExpandedKeys"];
  setSelectedKeys: SheetContextState["setSelectedKeys"];
  setEditingNode: SheetContextState["setEditingNode"];
}

const VIEW_CONTEXTS_LAZY: Record<SheetViewMode, ViewContext> = {
  get my() {
    return getViewContext("my");
  },
  get shared() {
    return getViewContext("shared");
  },
  get draft() {
    return getViewContext("draft");
  },
} as Record<SheetViewMode, ViewContext>;

// Module-level action helpers — always-stable references, never go
// through the React subscription path.
const setFilterAction: SheetContextState["setFilter"] = (next) =>
  useSheetContextStore.getState().setFilter(next);
const setViewAction: SheetContextState["setView"] = (view) =>
  useSheetContextStore.getState().setView(view);
const setExpandedKeysAction: SheetContextState["setExpandedKeys"] = (next) =>
  useSheetContextStore.getState().setExpandedKeys(next);
const setSelectedKeysAction: SheetContextState["setSelectedKeys"] = (next) =>
  useSheetContextStore.getState().setSelectedKeys(next);
const setEditingNodeAction: SheetContextState["setEditingNode"] = (next) =>
  useSheetContextStore.getState().setEditingNode(next);

/**
 * Top-level sheet context. Returns plain values that re-render on the
 * Zustand store changes the hook subscribes to.
 *
 * The hook also lazy-initializes the cross-store watchers
 * (project change, current-tab change, persistence) the first time
 * it's called from anywhere in the app.
 *
 * Only the *reactive* subset goes through `useShallow` / the Zustand
 * subscription. Stable helpers (action setters, view contexts, pure
 * functions) are merged in afterwards as module-level references so
 * they never destabilize the shallow comparison.
 */
export function useSheetContext(): SheetContext {
  const reactive = useSheetContextStore(
    useShallow((state) => ({
      filter: state.filter,
      filterChanged: !isEqual(state.filter, INITIAL_FILTER),
      expandedKeys: state.expandedKeys,
      selectedKeys: state.selectedKeys,
      editingNode: state.editingNode,
      view: state.view,
    }))
  );
  return {
    ...reactive,
    viewContexts: VIEW_CONTEXTS_LAZY,
    isSavedQueryCreator,
    batchUpdateSavedQueryFolders,
    batchUpdateSavedQueryFolderPaths,
    getContextByView: getViewContext,
    setFilter: setFilterAction,
    setView: setViewAction,
    setExpandedKeys: setExpandedKeysAction,
    setSelectedKeys: setSelectedKeysAction,
    setEditingNode: setEditingNodeAction,
  };
}

export type { SheetContext as SheetContextType };

/**
 * Returns the per-view context. Components subscribed via this hook
 * re-render when ANY of that view's state changes
 * (isLoading / isInitialized / sheetTree / folders).
 */
export function useSheetContextByView(view: SheetViewMode): ViewContext {
  // Subscribe to the view's state so React re-renders on changes.
  // The returned ViewContext is the same stable reference; its
  // property getters read the latest Zustand state.
  useSheetContextStore((s) => s.viewStates[view]);
  return getViewContext(view);
}

/**
 * Eagerly initialize the sheet-state watchers + storage hydration so
 * the rest of the app sees a populated context on mount. Kept as a
 * no-op-shaped function for source compatibility with the Pinia-era
 * `provideSheetContext()` boot call.
 */
export function provideSheetContext(): void {
  bindWatchers();
}

// `KEY` was a Vue `InjectionKey`; kept exported as a Symbol for source
// compatibility (no current consumers).
export const KEY = Symbol("bb.sql-editor.sheet");

// ---- tree helpers (unchanged) ----------------------------------------------

export const revealNodes = <T>(
  node: SavedQueryFolderNode,
  callback: (node: SavedQueryFolderNode) => T | undefined
): T[] => {
  const results: T[] = [];
  const item = callback(node);
  if (item) {
    results.push(item);
  }
  for (const child of node.children) {
    results.push(...revealNodes(child, callback));
  }
  return results;
};

export const revealSavedQueries = <T>(
  node: SavedQueryFolderNode,
  callback: (node: SavedQueryFolderNode) => T | undefined
): T[] => {
  return revealNodes(node, (n) => {
    if (!n.savedQuery) return undefined;
    return callback(n);
  });
};

// ---- openSavedQueryByName (unchanged behavior) ------------------------------

export const openSavedQueryByName = async ({
  savedQuery,
  forceNewTab,
  mode,
}: {
  savedQuery: string;
  forceNewTab: boolean;
  mode?: SQLEditorTabMode;
}) => {
  const sheet = await useAppStore
    .getState()
    .getOrFetchSavedQueryByName(savedQuery);
  if (!sheet) return undefined;

  if (!isSavedQueryReadableV1(sheet)) {
    useAppStore.getState().notify({
      module: "bytebase",
      style: "CRITICAL",
      title: i18n.t("common.access-denied"),
    });
    return undefined;
  }

  const tabsState = getSQLEditorTabsState();
  const openingSheetTab = (() => {
    for (const persisted of tabsState.openTmpTabList) {
      const tab = tabsState.tabsById.get(persisted.id);
      if (tab?.savedQuery === sheet.name) return tab;
    }
    return undefined;
  })();

  if (openingSheetTab && !forceNewTab) {
    tabsState.setCurrentTabId(openingSheetTab.id);
    if (mode && mode !== openingSheetTab.mode) {
      tabsState.updateTab(openingSheetTab.id, { mode });
    }
    return openingSheetTab;
  }

  const statement = getSheetStatement(sheet);
  const connection = await extractSavedQueryConnection(sheet);
  const newTab: Partial<SQLEditorTab> = {
    connection,
    savedQuery: sheet.name,
    title: sheet.title,
    statement,
    status: "CLEAN",
    mode: mode ?? "SAVED_QUERY",
  };

  return tabsState.addTab(newTab, forceNewTab /* beside */);
};

// Touch the editor store import so it's eagerly evaluated alongside the
// tab store (matches the historical Pinia-era load order).
void useSQLEditorEditorStore;
void useSQLEditorTabsStore;
