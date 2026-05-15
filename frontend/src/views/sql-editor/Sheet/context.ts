import { useDebounceFn } from "@vueuse/core";
import Emittery from "emittery";
import { isEqual, orderBy } from "lodash-es";
import scrollIntoView from "scroll-into-view-if-needed";
import type { InjectionKey, Ref } from "vue";
import { computed, nextTick, ref, toRefs, watch } from "vue";
import { t } from "@/plugins/i18n";
import { useSQLEditorVueState } from "@/react/stores/sqlEditor/editor-vue-state";
import type { FolderContext } from "@/react/stores/sqlEditor/folder";
import { buildFolderContext } from "@/react/stores/sqlEditor/folder";
import { useSQLEditorTabStore } from "@/react/stores/sqlEditor/tab-vue-state";
import { pushNotification, useCurrentUserV1, useWorkSheetStore } from "@/store";
import type { SQLEditorTab, SQLEditorTabMode } from "@/types";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import {
  type Worksheet,
  Worksheet_Visibility,
} from "@/types/proto-es/v1/worksheet_service_pb";
import {
  extractWorksheetConnection,
  getSheetStatement,
  isWorksheetReadableV1,
  storageKeySqlEditorWorksheetFilter,
  storageKeySqlEditorWorksheetTree,
  useDynamicLocalStorage,
} from "@/utils";
import type { SheetViewMode } from "./types";

// ---- public types -----------------------------------------------------------

export type { FolderContext };

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

type SheetTreeEvents = Emittery<{
  "on-built": { viewMode: SheetViewMode };
}>;

// ---- per-view tree context --------------------------------------------------

const convertToWorksheetLikeItem = (
  worksheet: Worksheet
): WorksheetLikeItem => ({
  name: worksheet.name,
  title: worksheet.title,
  folders: worksheet.folders,
  type: "worksheet",
});

const buildViewContext = (
  viewMode: SheetViewMode,
  filterRef: Ref<WorksheetFilter>
) => {
  const sheetStore = useWorkSheetStore();
  const folderContext = buildFolderContext(viewMode);
  const me = useCurrentUserV1();
  const events: SheetTreeEvents = new Emittery();
  const tabStore = useSQLEditorTabStore();
  const { project } = toRefs(useSQLEditorVueState());

  const isInitialized = ref(false);
  const isLoading = ref(false);

  const rootNodeLabel = computed(() => {
    switch (viewMode) {
      case "my":
        return t("sheet.mine");
      case "shared":
        return t("sheet.shared");
      case "draft":
        return t("common.draft");
      default:
        return "";
    }
  });

  const getRootTreeNode = (): WorksheetFolderNode => ({
    isLeaf: false,
    children: [],
    key: folderContext.rootPath.value,
    label: rootNodeLabel.value,
    editable: false,
  });

  const sheetTree = ref<WorksheetFolderNode>(getRootTreeNode());

  const worksheetList = computed((): Worksheet[] => {
    let list: Worksheet[] = [];
    switch (viewMode) {
      case "my":
        list = sheetStore.myWorksheetList;
        break;
      case "shared":
        list = sheetStore.sharedWorksheetList;
        break;
      default:
        break;
    }
    list = list.filter((sheet) => sheet.project === project.value);
    if (filterRef.value.onlyShowStarred) {
      return list.filter((sheet) => sheet.starred);
    }
    return list;
  });

  const sheetLikeItemList = computed((): WorksheetLikeItem[] => {
    switch (viewMode) {
      case "my":
      case "shared":
        return worksheetList.value.map(convertToWorksheetLikeItem);
      case "draft":
        return tabStore.openTabList
          .filter((tab) => !tab.worksheet)
          .map((tab) => ({
            name: tab.id,
            title: tab.title,
            folders: [],
            type: "draft" as const,
          }));
      default:
        return [];
    }
  });

  const getPathesForWorksheet = (worksheet: {
    folders: string[];
  }): string[] => {
    const pathes = [folderContext.rootPath.value];
    let currentPath = folderContext.rootPath.value;
    for (const folder of worksheet.folders) {
      currentPath = folderContext.ensureFolderPath(`${currentPath}/${folder}`);
      pathes.push(currentPath);
    }
    return pathes;
  };

  const getPwdForWorksheet = (worksheet: { folders: string[] }): string => {
    return folderContext.ensureFolderPath(worksheet.folders.join("/"));
  };

  const getKeyForWorksheet = (worksheet: WorksheetLikeItem): string => {
    return [
      getPwdForWorksheet(worksheet),
      `bytebase-${worksheet.type}-${worksheet.name.split("/").slice(-1)[0]}.sql`,
    ].join("/");
  };

  const getFoldersForWorksheet = (path: string): string[] => {
    const pathes = path.replace(folderContext.rootPath.value, "").split("/");
    if (pathes.slice(-1)[0].endsWith(".sql")) {
      pathes.pop();
    }
    return pathes.map((p) => p.trim()).filter((p) => p);
  };

  const buildTree = (
    parent: WorksheetFolderNode,
    worksheetsByFolder: Map<string, WorksheetLikeItem[]>,
    hideEmpty: boolean
  ) => {
    const subfolders = folderContext
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
      const subtree = buildTree(childNode, worksheetsByFolder, hideEmpty);
      if (!subtree.empty || !hideEmpty) {
        parent.children.push(subtree);
      }
      if (!subtree.empty) {
        empty = false;
      }
    }

    const sheets = orderBy(
      worksheetsByFolder.get(parent.key) || [],
      (item) => item.title
    ).map((worksheet) => ({
      isLeaf: true,
      key: getKeyForWorksheet(worksheet),
      label: worksheet.title,
      worksheet,
      editable: true,
      children: [],
    }));

    parent.children.push(...sheets);
    parent.empty = sheets.length === 0 && empty;
    if (parent.key !== folderContext.rootPath.value) {
      parent.isLeaf = parent.children.length === 0;
    }

    return parent;
  };

  const folderTree = computed(() =>
    buildTree(getRootTreeNode(), new Map(), false)
  );

  const rebuildTree = useDebounceFn(() => {
    const folderPaths = new Set<string>([]);
    const worksheetsByFolder = new Map<string, WorksheetLikeItem[]>();

    for (const worksheet of sheetLikeItemList.value) {
      for (const path of getPathesForWorksheet(worksheet)) {
        folderPaths.add(path);
      }
      const pwd = getPwdForWorksheet(worksheet);
      if (!worksheetsByFolder.has(pwd)) {
        worksheetsByFolder.set(pwd, []);
      }
      worksheetsByFolder.get(pwd)!.push(worksheet);
    }

    folderContext.mergeFolders(folderPaths);
    sheetTree.value = buildTree(
      getRootTreeNode(),
      worksheetsByFolder,
      filterRef.value.onlyShowStarred
    );
    events.emit("on-built", { viewMode });
  }, DEBOUNCE_SEARCH_DELAY);

  watch(
    [() => folderContext.folders.value, () => sheetLikeItemList.value],
    ([newFolders, newSheetList], [oldFolders, oldSheetList]) => {
      if (
        isEqual(newFolders, oldFolders) &&
        isEqual(newSheetList, oldSheetList)
      ) {
        return;
      }
      rebuildTree();
    }
  );

  const fetchSheetList = async () => {
    isLoading.value = true;
    try {
      switch (viewMode) {
        case "my":
          await sheetStore.fetchWorksheetList(
            project.value,
            `creator == "users/${me.value.email}"`
          );
          break;
        case "shared":
          await sheetStore.fetchWorksheetList(
            project.value,
            [
              `creator != "users/${me.value.email}"`,
              `visibility in ["${Worksheet_Visibility[Worksheet_Visibility.PROJECT_READ]}","${Worksheet_Visibility[Worksheet_Visibility.PROJECT_WRITE]}"]`,
            ].join(" && ")
          );
          break;
        default:
          break;
      }
      rebuildTree();
      isInitialized.value = true;
    } finally {
      isLoading.value = false;
    }
  };

  return {
    events,
    isInitialized,
    isLoading,
    sheetTree,
    folderTree,
    fetchSheetList,
    folderContext,
    getKeyForWorksheet,
    getFoldersForWorksheet,
    getPathesForWorksheet,
    getPwdForWorksheet,
  };
};

export type ViewContext = ReturnType<typeof buildViewContext>;

// ---- top-level singleton ----------------------------------------------------

const INITIAL_FILTER: WorksheetFilter = {
  keyword: "",
  showShared: true,
  showMine: true,
  showDraft: true,
  onlyShowStarred: false,
};

/**
 * The shared sheet-context state. Pulled out of Pinia's `defineStore`
 * into a module-level lazy singleton because none of this state is
 * actually shared with Vue components (all live consumers are React)
 * — dropping Pinia removes the cross-framework store layer without
 * needing to rewrite the Vue-reactive pipeline. The lazy singleton
 * preserves "set up once, watchers stay alive" semantics that Pinia
 * gave us for free.
 */
const buildSheetState = () => {
  const tabStore = useSQLEditorTabStore();
  const me = useCurrentUserV1();
  const worksheetV1Store = useWorkSheetStore();
  const { project } = toRefs(useSQLEditorVueState());

  // ---- shared filter ------------------------------------------------------

  const filter = useDynamicLocalStorage<WorksheetFilter>(
    computed(() =>
      storageKeySqlEditorWorksheetFilter(project.value, me.value.email)
    ),
    { ...INITIAL_FILTER }
  );
  const filterChanged = computed(() => !isEqual(filter.value, INITIAL_FILTER));

  // Safe computed ref — useDynamicLocalStorage can yield null during SSR
  // or on first boot; fall back to INITIAL_FILTER so consumers always get
  // a non-null WorksheetFilter.
  const safeFilter = computed<WorksheetFilter>(
    () => filter.value ?? { ...INITIAL_FILTER }
  );

  // ---- per-view contexts (lazily initialised) -----------------------------
  // Use a plain Map (not reactive) — the ViewContext objects have their own
  // internal reactive state. Wrapping in reactive() would auto-unwrap refs.

  const contexts = new Map<SheetViewMode, ViewContext>();

  const ensureContext = (view: SheetViewMode): ViewContext => {
    if (!contexts.has(view)) {
      contexts.set(view, buildViewContext(view, safeFilter));
    }
    return contexts.get(view)!;
  };

  const getContextByView = (view: SheetViewMode) => ensureContext(view);

  // Eagerly build all three so watchers/selection logic works on startup.
  const viewContexts = {
    get my() {
      return ensureContext("my");
    },
    get shared() {
      return ensureContext("shared");
    },
    get draft() {
      return ensureContext("draft");
    },
  };

  // ---- shared tree-selection state ----------------------------------------

  const expandedKeys = useDynamicLocalStorage<Set<string>>(
    computed(() =>
      storageKeySqlEditorWorksheetTree(project.value, me.value.email)
    ),
    new Set([
      ensureContext("my").folderContext.rootPath.value,
      ensureContext("shared").folderContext.rootPath.value,
      ensureContext("draft").folderContext.rootPath.value,
    ])
  );
  const selectedKeys = ref<string[]>([]);

  const isWorksheetCreator = (worksheet: { creator: string }) =>
    worksheet.creator === `users/${me.value.email}`;

  watch(
    () => ({
      tabId: tabStore.currentTab?.id,
      worksheetName: tabStore.currentTab?.worksheet,
    }),
    async ({ tabId, worksheetName }) => {
      selectedKeys.value = [];
      if (!tabId) {
        return;
      }

      let viewMode: SheetViewMode = "draft";
      let worksheetLikeItem: WorksheetLikeItem | undefined;

      if (worksheetName) {
        const worksheet = worksheetV1Store.getWorksheetByName(worksheetName);
        if (!worksheet) {
          return;
        }
        worksheetLikeItem = {
          name: worksheet.name,
          title: worksheet.title,
          folders: worksheet.folders,
          type: "worksheet",
        };
        viewMode = isWorksheetCreator(worksheet) ? "my" : "shared";
      } else {
        worksheetLikeItem = {
          name: tabId,
          folders: [],
          title: "",
          type: "draft",
        };
      }

      const viewContext = ensureContext(viewMode);
      const key = viewContext.getKeyForWorksheet(worksheetLikeItem);
      selectedKeys.value = [key];

      for (const path of viewContext.getPathesForWorksheet(worksheetLikeItem)) {
        expandedKeys.value.add(path);
      }

      await nextTick();
      const dom = document.querySelector(`[data-item-key="${key}"]`);
      if (dom) {
        scrollIntoView(dom, {
          scrollMode: "if-needed",
          block: "nearest",
        });
      }
    },
    { immediate: true }
  );

  // ---- editing node -------------------------------------------------------

  const editingNode = ref<
    { node: WorksheetFolderNode; rawLabel: string } | undefined
  >();

  // ---- batch operations ---------------------------------------------------

  const batchUpdateWorksheetFolders = async (
    worksheets: { name: string; folders: string[] }[]
  ) => {
    if (worksheets.length === 0) {
      return;
    }
    await worksheetV1Store.batchUpsertWorksheetOrganizers(
      worksheets.map((worksheet) => ({
        organizer: {
          worksheet: worksheet.name,
          folders: worksheet.folders,
        },
        updateMask: ["folders"],
      }))
    );
  };

  // ---- view ref -----------------------------------------------------------

  const view = ref<SheetViewMode>("my");

  return {
    view,
    viewContexts,
    filter,
    filterChanged,
    expandedKeys,
    selectedKeys,
    editingNode,
    isWorksheetCreator,
    batchUpdateWorksheetFolders,
    getContextByView,
  };
};

let _sheetState: ReturnType<typeof buildSheetState> | undefined;
const getSheetState = () => {
  if (!_sheetState) _sheetState = buildSheetState();
  return _sheetState;
};

// ---- public API ------------------------------------------------------------

/**
 * Returns the full sheet context (filter, expandedKeys, selectedKeys, etc.).
 * Previously delegated to a Pinia store; now reads directly from a
 * module-level Vue-reactive singleton built by `buildSheetState()`.
 */
export const useSheetContext = () => getSheetState();

export type SheetContext = ReturnType<typeof useSheetContext>;

// ---- InjectionKey kept for source-compat (no longer used) ------------------

export const KEY = Symbol("bb.sql-editor.sheet") as InjectionKey<SheetContext>;

/**
 * Returns the per-view context for `view`.
 */
export const useSheetContextByView = (view: SheetViewMode) =>
  getSheetState().getContextByView(view);

// ---- tree helpers ----------------------------------------------------------

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
    if (!n.worksheet) {
      return undefined;
    }
    return callback(n);
  });
};

/**
 * Eagerly boot the sheet-state singleton so its watchers start firing on
 * app mount. Kept as a no-op-shaped function so call sites match the
 * Pinia-era `useSQLEditorWorksheetStore()` boot call.
 */
export const provideSheetContext = () => getSheetState();

// ---- openWorksheetByName (unchanged) ----------------------------------------

export const openWorksheetByName = async ({
  worksheet,
  forceNewTab,
  mode,
}: {
  worksheet: string;
  forceNewTab: boolean;
  mode?: SQLEditorTabMode;
}) => {
  const sheet = await useWorkSheetStore().getOrFetchWorksheetByName(worksheet);
  if (!sheet) {
    return undefined;
  }

  if (!isWorksheetReadableV1(sheet)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.access-denied"),
    });
    return undefined;
  }

  const tabStore = useSQLEditorTabStore();
  const openingSheetTab = tabStore.getTabByWorksheet(sheet.name);

  if (openingSheetTab && !forceNewTab) {
    tabStore.setCurrentTabId(openingSheetTab.id);
    if (mode && mode !== openingSheetTab.mode) {
      tabStore.updateTab(openingSheetTab.id, { mode });
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

  return tabStore.addTab(newTab, forceNewTab /* beside */);
};
