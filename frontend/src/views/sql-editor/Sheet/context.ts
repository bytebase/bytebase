import Emittery from "emittery";
import { enableMapSet } from "immer";
import { debounce, isEqual, orderBy, sortBy } from "lodash-es";
import scrollIntoView from "scroll-into-view-if-needed";
import { create, type StoreApi, type UseBoundStore } from "zustand";
import { immer } from "zustand/middleware/immer";
import { useShallow } from "zustand/react/shallow";
import { t } from "@/plugins/i18n";
import { extractWorksheetConnection } from "@/react/lib/sqlEditorConnection";
import { useAppStore } from "@/react/stores/app";
import {
  getSQLEditorEditorState,
  subscribeSQLEditorEditorState,
  useSQLEditorEditorStore,
} from "@/react/stores/sqlEditor/editor";
import {
  getSQLEditorTabsState,
  subscribeSQLEditorTabsState,
  useSQLEditorTabsStore,
} from "@/react/stores/sqlEditor/tab";
import type { SQLEditorTab, SQLEditorTabMode } from "@/types";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import {
  type Worksheet,
  Worksheet_Visibility,
} from "@/types/proto-es/v1/worksheet_service_pb";
import {
  getSheetStatement,
  isWorksheetReadableV1,
  storageKeySqlEditorWorksheetFilter,
  storageKeySqlEditorWorksheetFolder,
  storageKeySqlEditorWorksheetTree,
  workspaceCacheScope,
} from "@/utils";
import { type SheetViewMode, SheetViewModeList } from "./types";

// Worksheet caches, folder sets, and the sheet-tree contain Map / Set
// values that immer needs to draft directly via mutation.
enableMapSet();

// ---- public types ----------------------------------------------------------

export interface WorksheetLikeItem {
  name: string;
  title: string;
  folders: string[];
  type: "worksheet" | "draft";
}

export interface WorksheetFolderNode {
  key: string;
  label: string;
  editable: boolean;
  isLeaf?: boolean;
  empty?: boolean;
  worksheet?: WorksheetLikeItem;
  children: WorksheetFolderNode[];
  [key: string]: unknown;
}

export interface WorksheetFilter {
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
  sheetTree: WorksheetFolderNode;
  folderTree: WorksheetFolderNode;
  folderContext: FolderContext;
  events: SheetTreeEvents;
  fetchSheetList: () => Promise<void>;
  rebuildTree: () => void;
  getKeyForWorksheet: (worksheet: WorksheetLikeItem) => string;
  getFoldersForWorksheet: (path: string) => string[];
  getPathesForWorksheet: (worksheet: { folders: string[] }) => string[];
  getPwdForWorksheet: (worksheet: { folders: string[] }) => string;
}

// ---- internal Zustand store ------------------------------------------------

const INITIAL_FILTER: WorksheetFilter = {
  keyword: "",
  showShared: true,
  showMine: true,
  showDraft: true,
  onlyShowStarred: false,
};

interface ViewState {
  isLoading: boolean;
  isInitialized: boolean;
  sheetTree: WorksheetFolderNode;
  folders: string[];
}

interface SheetContextState {
  filter: WorksheetFilter;
  expandedKeys: Set<string>;
  selectedKeys: string[];
  editingNode: { node: WorksheetFolderNode; rawLabel: string } | undefined;
  view: SheetViewMode;
  viewStates: Record<SheetViewMode, ViewState>;

  setFilter: (
    next: WorksheetFilter | ((prev: WorksheetFilter) => WorksheetFilter)
  ) => void;
  setView: (view: SheetViewMode) => void;
  setExpandedKeys: (
    next: Set<string> | ((prev: Set<string>) => Set<string>)
  ) => void;
  setSelectedKeys: (next: string[]) => void;
  setEditingNode: (
    next: { node: WorksheetFolderNode; rawLabel: string } | undefined
  ) => void;
  setViewIsLoading: (view: SheetViewMode, loading: boolean) => void;
  setViewIsInitialized: (view: SheetViewMode, initialized: boolean) => void;
  setViewSheetTree: (view: SheetViewMode, tree: WorksheetFolderNode) => void;
  setViewFolders: (view: SheetViewMode, folders: string[]) => void;
  /** Replace the entire state (used by project / user-scope reload). */
  hydrate: (next: Partial<SheetContextState>) => void;
}

const rootPathFor = (view: SheetViewMode) => `/${view}`;

const emptyRootNode = (view: SheetViewMode): WorksheetFolderNode => ({
  key: rootPathFor(view),
  label: "",
  editable: false,
  isLeaf: false,
  children: [],
});

const emptyViewState = (view: SheetViewMode): ViewState => ({
  isLoading: false,
  isInitialized: false,
  sheetTree: emptyRootNode(view),
  folders: [rootPathFor(view)],
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

const isWorksheetFilter = (v: unknown): v is WorksheetFilter =>
  typeof v === "object" &&
  v !== null &&
  "keyword" in v &&
  "onlyShowStarred" in v &&
  "showMine" in v &&
  "showShared" in v &&
  "showDraft" in v;

const reloadFromStorage = () => {
  const scope = currentScope();
  if (!scope) return;

  const filter = safeReadJSON(
    storageKeySqlEditorWorksheetFilter(
      scope.wsScope,
      scope.project,
      scope.email
    ),
    (v) => (isWorksheetFilter(v) ? v : undefined)
  ) ?? { ...INITIAL_FILTER };

  const expandedArray = safeReadJSON<string[]>(
    storageKeySqlEditorWorksheetTree(scope.wsScope, scope.project, scope.email),
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

  for (const view of ["my", "shared", "draft"] as const) {
    const folders = safeReadJSON<string[]>(
      storageKeySqlEditorWorksheetFolder(
        scope.wsScope,
        scope.project,
        view,
        scope.email
      ),
      (v) =>
        Array.isArray(v) && v.every((entry) => typeof entry === "string")
          ? (v as string[])
          : undefined
    );
    const set = new Set(folders ?? []);
    set.add(rootPathFor(view));
    viewStates[view].folders = sortBy([...set]);
  }

  useSheetContextStore.getState().hydrate({
    filter,
    expandedKeys,
    selectedKeys: [],
    editingNode: undefined,
    viewStates,
  });
};

const persistFilter = (filter: WorksheetFilter) => {
  const scope = currentScope();
  if (!scope) return;
  safeWriteJSON(
    storageKeySqlEditorWorksheetFilter(
      scope.wsScope,
      scope.project,
      scope.email
    ),
    filter
  );
};

const persistExpandedKeys = (keys: Set<string>) => {
  const scope = currentScope();
  if (!scope) return;
  safeWriteJSON(
    storageKeySqlEditorWorksheetTree(scope.wsScope, scope.project, scope.email),
    [...keys]
  );
};

const persistViewFolders = (view: SheetViewMode, folders: string[]) => {
  const scope = currentScope();
  if (!scope) return;
  safeWriteJSON(
    storageKeySqlEditorWorksheetFolder(
      scope.wsScope,
      scope.project,
      view,
      scope.email
    ),
    folders
  );
};

// ---- per-view helpers + folder context -------------------------------------

const convertToWorksheetLikeItem = (
  worksheet: Worksheet
): WorksheetLikeItem => ({
  name: worksheet.name,
  title: worksheet.title,
  folders: worksheet.folders,
  type: "worksheet",
});

const rootLabelFor = (view: SheetViewMode): string => {
  switch (view) {
    case "my":
      return t("sheet.mine");
    case "shared":
      return t("sheet.shared");
    case "draft":
      return t("common.draft");
    default:
      return "";
  }
};

const rootTreeNodeFor = (view: SheetViewMode): WorksheetFolderNode => ({
  isLeaf: false,
  children: [],
  key: rootPathFor(view),
  label: rootLabelFor(view),
  editable: false,
});

const isSubFolder = ({
  parent,
  path,
  dig,
}: {
  parent: string;
  path: string;
  dig: boolean;
}) => {
  const parentPrefix = `${parent}/`;
  return path !== parentPrefix && path.startsWith(parentPrefix) && dig
    ? true
    : !path.replace(parentPrefix, "").includes("/");
};

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
      persistViewFolders(view, next);
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
      persistViewFolders(view, next);
    },
    moveFolder(from, to) {
      const fromPath = ensureFolderPath(view, from);
      const toPath = ensureFolderPath(view, to);
      const current = useSheetContextStore.getState().viewStates[view].folders;
      const next = current.map((path) => {
        if (path === fromPath) return toPath;
        if (isSubFolder({ parent: fromPath, path, dig: true })) {
          return path.replace(fromPath, toPath);
        }
        return path;
      });
      const deduped = sortBy([...new Set(next)]);
      useSheetContextStore.getState().setViewFolders(view, deduped);
      persistViewFolders(view, deduped);
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
        persistViewFolders(view, next);
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

const getPathesForWorksheet = (
  view: SheetViewMode,
  worksheet: { folders: string[] }
): string[] => {
  const folderContext = getFolderContext(view);
  const pathes = [folderContext.rootPath];
  let currentPath = folderContext.rootPath;
  for (const folder of worksheet.folders) {
    currentPath = folderContext.ensureFolderPath(`${currentPath}/${folder}`);
    pathes.push(currentPath);
  }
  return pathes;
};

const getPwdForWorksheet = (
  view: SheetViewMode,
  worksheet: { folders: string[] }
): string =>
  getFolderContext(view).ensureFolderPath(worksheet.folders.join("/"));

const getKeyForWorksheet = (
  view: SheetViewMode,
  worksheet: WorksheetLikeItem
): string =>
  [
    getPwdForWorksheet(view, worksheet),
    `bytebase-${worksheet.type}-${worksheet.name.split("/").slice(-1)[0]}.sql`,
  ].join("/");

const getFoldersForWorksheet = (
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
  parent: WorksheetFolderNode,
  worksheetsByFolder: Map<string, WorksheetLikeItem[]>,
  hideEmpty: boolean
): WorksheetFolderNode => {
  const folderContext = getFolderContext(view);
  const subfolders: WorksheetFolderNode[] = folderContext
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
    const subtree = buildTree(view, childNode, worksheetsByFolder, hideEmpty);
    if (!subtree.empty || !hideEmpty) {
      parent.children.push(subtree);
    }
    if (!subtree.empty) {
      empty = false;
    }
  }

  const sheets = orderBy(
    worksheetsByFolder.get(parent.key) ?? [],
    (item) => item.title
  ).map<WorksheetFolderNode>((worksheet) => ({
    isLeaf: true,
    key: getKeyForWorksheet(view, worksheet),
    label: worksheet.title,
    worksheet,
    editable: true,
    children: [],
  }));

  parent.children.push(...sheets);
  parent.empty = sheets.length === 0 && empty;
  if (parent.key !== folderContext.rootPath) {
    parent.isLeaf = parent.children.length === 0;
  }
  return parent;
};

const worksheetsForView = (view: SheetViewMode): Worksheet[] => {
  if (view !== "my" && view !== "shared") return [];
  const filter = useSheetContextStore.getState().filter;
  const project = getSQLEditorEditorState().project;
  // SQLEditorLayout awaits `loadCurrentUser()` in its bootstrap, so by the
  // time worksheets land here the app-store `currentUser` is populated.
  // Empty `email` falls through to creator `"users/"`, which matches no
  // worksheets — they'd render as Shared rather than Mine, same fallback
  // the previous Pinia path had.
  const email = useAppStore.getState().currentUser?.email ?? "";
  const creator = `users/${email}`;
  let list = useAppStore
    .getState()
    .worksheetList()
    .filter((sheet) => {
      if (sheet.project !== project) return false;
      const mine = sheet.creator === creator;
      return view === "my" ? mine : !mine;
    });
  if (filter.onlyShowStarred) {
    list = list.filter((sheet) => sheet.starred);
  }
  return list;
};

const sheetLikeItemsForView = (view: SheetViewMode): WorksheetLikeItem[] => {
  if (view === "draft") {
    const tabsState = getSQLEditorTabsState();
    return tabsState.openTmpTabList
      .map((p) => tabsState.tabsById.get(p.id))
      .filter((tab): tab is SQLEditorTab => !!tab && !tab.worksheet)
      .map((tab) => ({
        name: tab.id,
        title: tab.title,
        folders: [],
        type: "draft" as const,
      }));
  }
  return worksheetsForView(view).map(convertToWorksheetLikeItem);
};

const rebuildTreeImpl = (view: SheetViewMode) => {
  const filter = useSheetContextStore.getState().filter;
  const folderContext = getFolderContext(view);

  const folderPaths = new Set<string>();
  const worksheetsByFolder = new Map<string, WorksheetLikeItem[]>();

  for (const worksheet of sheetLikeItemsForView(view)) {
    for (const path of getPathesForWorksheet(view, worksheet)) {
      folderPaths.add(path);
    }
    const pwd = getPwdForWorksheet(view, worksheet);
    if (!worksheetsByFolder.has(pwd)) worksheetsByFolder.set(pwd, []);
    worksheetsByFolder.get(pwd)!.push(worksheet);
  }

  folderContext.mergeFolders(folderPaths);

  const root: WorksheetFolderNode = {
    ...rootTreeNodeFor(view),
    label: rootLabelFor(view),
    key: folderContext.rootPath,
  };
  const tree = buildTree(
    view,
    root,
    worksheetsByFolder,
    filter.onlyShowStarred
  );
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
    const sheetStore = useAppStore.getState();
    const email = sheetStore.currentUser?.email ?? "";
    const project = getSQLEditorEditorState().project;
    switch (view) {
      case "my":
        await sheetStore.fetchWorksheetList(
          project,
          `creator == "users/${email}"`
        );
        break;
      case "shared":
        await sheetStore.fetchWorksheetList(
          project,
          [
            `creator != "users/${email}"`,
            `visibility in ["${Worksheet_Visibility[Worksheet_Visibility.PROJECT_READ]}","${Worksheet_Visibility[Worksheet_Visibility.PROJECT_WRITE]}"]`,
          ].join(" && ")
        );
        break;
      default:
        break;
    }
    rebuildTreeImpl(view);
    state.setViewIsInitialized(view, true);
  } finally {
    state.setViewIsLoading(view, false);
  }
};

const buildViewContext = (view: SheetViewMode): ViewContext => {
  const folderContext = getFolderContext(view);
  return {
    get isLoading() {
      return useSheetContextStore.getState().viewStates[view].isLoading;
    },
    get isInitialized() {
      return useSheetContextStore.getState().viewStates[view].isInitialized;
    },
    get sheetTree() {
      return useSheetContextStore.getState().viewStates[view].sheetTree;
    },
    get folderTree() {
      const root: WorksheetFolderNode = {
        ...rootTreeNodeFor(view),
        key: folderContext.rootPath,
      };
      return buildTree(view, root, new Map(), false);
    },
    folderContext,
    events: getEvents(view),
    fetchSheetList: () => fetchSheetListFor(view),
    rebuildTree: () => getRebuildTreeFn(view)(),
    getKeyForWorksheet: (ws) => getKeyForWorksheet(view, ws),
    getFoldersForWorksheet: (path) => getFoldersForWorksheet(view, path),
    getPathesForWorksheet: (ws) => getPathesForWorksheet(view, ws),
    getPwdForWorksheet: (ws) => getPwdForWorksheet(view, ws),
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

const isWorksheetCreator = (worksheet: { creator: string }) => {
  const email = useAppStore.getState().currentUser?.email;
  if (!email) return false;
  return worksheet.creator === `users/${email}`;
};

const batchUpdateWorksheetFolders = async (
  worksheets: { name: string; folders: string[] }[]
): Promise<void> => {
  if (worksheets.length === 0) return;
  await useAppStore.getState().batchUpsertWorksheetOrganizers(
    worksheets.map((worksheet) => ({
      organizer: {
        worksheet: worksheet.name,
        folders: worksheet.folders,
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
// no worksheet, keyed by id + title. The "draft" tree derives from
// these (see `sheetLikeItemsForView`), so the tree must rebuild when a
// draft tab is opened, closed, or renamed.
const computeDraftSignature = (
  tabsState: ReturnType<typeof getSQLEditorTabsState>
): string =>
  tabsState.openTmpTabList
    .map((persisted) => {
      const tab = tabsState.tabsById.get(persisted.id);
      if (!tab || tab.worksheet) return "";
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
    if (state.filter !== prev.filter) persistFilter(state.filter);
    if (state.expandedKeys !== prev.expandedKeys)
      persistExpandedKeys(state.expandedKeys);

    // Rebuild the worksheet-backed trees when `onlyShowStarred` toggles
    // — it changes both the worksheet list (`worksheetsForView` filters
    // to starred) and the `hideEmpty` pass inside `buildTree`. Mirrors
    // the Vue `watch(filter, rebuildTree)`. Keyword filtering is applied
    // component-side (SheetTree's `nodeMatches`), so it doesn't need a
    // rebuild here.
    if (state.filter.onlyShowStarred !== prev.filter.onlyShowStarred) {
      getRebuildTreeFn("my")();
      getRebuildTreeFn("shared")();
    }

    // Rebuild a view's tree when its folder set changes (add / remove /
    // move folder). Mirrors the Vue `watch(folderContext.folders,
    // rebuildTree)`. Debounced + idempotent `mergeFolders` inside the
    // rebuild keeps this from looping: the rebuild only writes folders
    // back when a worksheet introduces a brand-new path, which settles
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
    const tabKey = `${tab?.id ?? ""}|${tab?.worksheet ?? ""}`;
    if (tabKey !== _lastTabKey) {
      _lastTabKey = tabKey;
      onCurrentTabChanged(tab);
    }

    // Draft-tab set / title change → rebuild the draft tree. The draft
    // view's `sheetLikeItemList` is derived from open tabs without a
    // worksheet, so adding / closing / renaming a draft must refresh it.
    // Mirrors the Vue `watch(sheetLikeItemList, rebuildTree)` for the
    // draft view context.
    const draftSig = computeDraftSignature(tabsState);
    if (draftSig !== _lastDraftSig) {
      _lastDraftSig = draftSig;
      getRebuildTreeFn("draft")();
    }
  });

  // Rebuild the worksheet-backed trees whenever the worksheet cache
  // mutates — a fetch, a title patch (e.g. renamed from the editor tab),
  // a star toggle, or a delete. Mirrors the Vue
  // `watch(sheetLikeItemList, rebuildTree)` that drove the tree off the
  // worksheet list. The "draft" view is tab-derived, not worksheet-
  // derived, so it doesn't need this. Debounced rebuilds coalesce the
  // burst of cache writes a single fetch produces.
  useAppStore.subscribe((state, prev) => {
    if (state.worksheetsByKey === prev.worksheetsByKey) return;
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
  let worksheetLikeItem: WorksheetLikeItem | undefined;
  const tabId = tab.id;
  const worksheetName = tab.worksheet;

  if (worksheetName) {
    const worksheet = useAppStore.getState().getWorksheetByName(worksheetName);
    if (!worksheet) return;
    worksheetLikeItem = {
      name: worksheet.name,
      title: worksheet.title,
      folders: worksheet.folders,
      type: "worksheet",
    };
    view = isWorksheetCreator(worksheet) ? "my" : "shared";
  } else {
    worksheetLikeItem = {
      name: tabId,
      folders: [],
      title: "",
      type: "draft",
    };
  }

  const viewCtx = getViewContext(view);
  const key = viewCtx.getKeyForWorksheet(worksheetLikeItem);
  useSheetContextStore.getState().setSelectedKeys([key]);

  const expanded = new Set(useSheetContextStore.getState().expandedKeys);
  for (const path of viewCtx.getPathesForWorksheet(worksheetLikeItem)) {
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
  filter: WorksheetFilter;
  filterChanged: boolean;
  expandedKeys: Set<string>;
  selectedKeys: string[];
  editingNode: { node: WorksheetFolderNode; rawLabel: string } | undefined;
  view: SheetViewMode;
  viewContexts: Record<SheetViewMode, ViewContext>;
  isWorksheetCreator: (worksheet: { creator: string }) => boolean;
  batchUpdateWorksheetFolders: (
    worksheets: { name: string; folders: string[] }[]
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
    isWorksheetCreator,
    batchUpdateWorksheetFolders,
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
  node: WorksheetFolderNode,
  callback: (node: WorksheetFolderNode) => T | undefined
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

export const revealWorksheets = <T>(
  node: WorksheetFolderNode,
  callback: (node: WorksheetFolderNode) => T | undefined
): T[] => {
  return revealNodes(node, (n) => {
    if (!n.worksheet) return undefined;
    return callback(n);
  });
};

// ---- openWorksheetByName (unchanged behavior) ------------------------------

export const openWorksheetByName = async ({
  worksheet,
  forceNewTab,
  mode,
}: {
  worksheet: string;
  forceNewTab: boolean;
  mode?: SQLEditorTabMode;
}) => {
  const sheet = await useAppStore
    .getState()
    .getOrFetchWorksheetByName(worksheet);
  if (!sheet) return undefined;

  if (!isWorksheetReadableV1(sheet)) {
    useAppStore.getState().notify({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.access-denied"),
    });
    return undefined;
  }

  const tabsState = getSQLEditorTabsState();
  const openingSheetTab = (() => {
    for (const persisted of tabsState.openTmpTabList) {
      const tab = tabsState.tabsById.get(persisted.id);
      if (tab?.worksheet === sheet.name) return tab;
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
  const connection = await extractWorksheetConnection(sheet);
  const newTab: Partial<SQLEditorTab> = {
    connection,
    worksheet: sheet.name,
    title: sheet.title,
    statement,
    status: "CLEAN",
    mode: mode ?? "WORKSHEET",
  };

  return tabsState.addTab(newTab, forceNewTab /* beside */);
};

// Touch the editor store import so it's eagerly evaluated alongside the
// tab store (matches the historical Pinia-era load order).
void useSQLEditorEditorStore;
void useSQLEditorTabsStore;
