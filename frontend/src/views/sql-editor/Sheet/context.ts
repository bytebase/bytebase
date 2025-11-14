import { useDebounceFn } from "@vueuse/core";
import Emittery from "emittery";
import { isEqual, orderBy } from "lodash-es";
import type { TreeOption } from "naive-ui";
import scrollIntoView from "scroll-into-view-if-needed";
import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, nextTick, provide, ref, watch } from "vue";
import { t } from "@/plugins/i18n";
import {
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useSQLEditorTabStore,
  useWorkSheetStore,
} from "@/store";
import type { SQLEditorTab } from "@/types";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import {
  type Worksheet,
  Worksheet_Visibility,
} from "@/types/proto-es/v1/worksheet_service_pb";
import {
  emptySQLEditorConnection,
  extractWorksheetUID,
  getSheetStatement,
  isWorksheetReadableV1,
  suggestedTabTitleForSQLEditorConnection,
  useDynamicLocalStorage,
} from "@/utils";
import type { SQLEditorContext } from "../context";
import { setDefaultDataSourceForConn } from "../EditorCommon";
import { useFolderByView } from "./folder";
import type { SheetViewMode } from "./types";

type SheetTreeEvents = Emittery<{
  "on-built": { viewMode: SheetViewMode };
}>;

export interface WorsheetFolderNode extends TreeOption {
  key: string;
  label: string;
  editable: boolean;
  empty?: boolean;
  worksheet?: Worksheet;
  children: WorsheetFolderNode[];
}

interface WorksheetFilter {
  keyword: string;
  onlyShowStarred: boolean;
  showMine: boolean;
  showShared: boolean;
  showDraft: boolean;
}

const useSheetTreeByView = (
  viewMode: SheetViewMode,
  filter: ComputedRef<WorksheetFilter>
) => {
  const sheetStore = useWorkSheetStore();
  const folderContext = useFolderByView(viewMode);
  const me = useCurrentUserV1();
  const events: SheetTreeEvents = new Emittery();

  const isInitialized = ref(false);
  const isLoading = ref(false);

  const rootNodeLabel = computed(() => {
    switch (viewMode) {
      case "my":
        return t("sheet.mine");
      case "shared":
        return t("sheet.shared");
      default:
        return "";
    }
  });

  const getRootTreeNode = (): WorsheetFolderNode => ({
    isLeaf: false,
    children: [],
    key: folderContext.rootPath.value,
    label: rootNodeLabel.value,
    editable: false,
  });
  const sheetTree = ref<WorsheetFolderNode>(getRootTreeNode());

  const sheetList = computed(() => {
    let list: Worksheet[] = [];
    switch (viewMode) {
      case "my":
        list = sheetStore.myWorksheetList;
        break;
      case "shared":
        list = sheetStore.sharedWorksheetList;
        break;
      default:
        return [];
    }
    if (filter.value.onlyShowStarred) {
      return list.filter((sheet) => sheet.starred);
    }
    return list;
  });

  const getPathesForWorksheet = (worksheet: Worksheet): string[] => {
    const pathes = [folderContext.rootPath.value];
    let currentPath = folderContext.rootPath.value;

    for (const folder of worksheet.folders) {
      currentPath = folderContext.ensureFolderPath(`${currentPath}/${folder}`);
      pathes.push(currentPath);
    }

    return pathes;
  };

  const getPwdForWorksheet = (worksheet: Worksheet): string => {
    return folderContext.ensureFolderPath(
      [folderContext.rootPath.value, ...worksheet.folders].join("/")
    );
  };

  const getKeyForWorksheet = (worksheet: Worksheet): string => {
    return [
      getPwdForWorksheet(worksheet),
      `bytebase-worksheets-${extractWorksheetUID(worksheet.name)}`,
    ].join("/");
  };

  const getFoldersForWorksheet = (path: string): string[] => {
    return path
      .replace(folderContext.rootPath.value, "")
      .split("/")
      .slice(0, -1) // the last element should be the getKeyForWorksheet(worksheet)
      .filter((p) => p);
  };

  // Extract only tree-relevant properties to avoid rebuilding on unrelated changes
  // (e.g., starred, content, visibility changes should NOT trigger rebuild)
  const treeRelevantSheetData = computed(() => {
    return sheetList.value.map((worksheet) => ({
      name: worksheet.name,
      title: worksheet.title,
      folders: worksheet.folders,
    }));
  });

  const buildTree = (
    parent: WorsheetFolderNode,
    worksheetsByFolder: Map<string, Worksheet[]>
  ) => {
    parent.children = folderContext
      .listSubFolders(parent.key)
      .map((folder) => ({
        isLeaf: false,
        children: [],
        key: folder,
        label: folder.split("/").slice(-1)[0],
        editable: true,
      }));

    let empty = true;
    for (const childNode of parent.children) {
      const subtree = buildTree(childNode, worksheetsByFolder);
      if (!subtree.empty) {
        empty = false;
      }
    }

    const sheets = orderBy(
      worksheetsByFolder.get(parent.key) || [],
      (item) => item.title
    ).map((worksheet) => {
      return {
        isLeaf: true,
        key: getKeyForWorksheet(worksheet),
        label: worksheet.title,
        worksheet,
        editable: true,
        children: [],
      };
    });

    parent.children.push(...sheets);
    parent.empty = sheets.length === 0 && empty;

    return parent;
  };

  const folderTree = computed(() => buildTree(getRootTreeNode(), new Map()));

  const rebuildTree = useDebounceFn(() => {
    const worksheetsByFolder = new Map<string, Worksheet[]>();
    for (const worksheet of sheetList.value) {
      const pwd = getPwdForWorksheet(worksheet);
      if (!worksheetsByFolder.has(pwd)) {
        worksheetsByFolder.set(pwd, []);
      }
      worksheetsByFolder.get(pwd)!.push(worksheet);
    }

    sheetTree.value = buildTree(getRootTreeNode(), worksheetsByFolder);
    events.emit("on-built", { viewMode });
  }, DEBOUNCE_SEARCH_DELAY);

  // Watch only relevant properties: folder structure and worksheet name/folders
  // No deep watch needed - computed will track changes to name, folders automatically
  watch([() => folderContext.folders.value, treeRelevantSheetData], () => {
    rebuildTree();
  });

  const fetchSheetList = async (project: string) => {
    isLoading.value = true;
    const filter = [];
    if (project) {
      filter.push(`project == "${project}"`);
    }
    try {
      switch (viewMode) {
        case "my":
          filter.push(`creator == "users/${me.value.email}"`);
          break;
        case "shared":
          filter.push(
            `creator != "users/${me.value.email}"`,
            `visibility in ["${Worksheet_Visibility[Worksheet_Visibility.PROJECT_READ]}","${Worksheet_Visibility[Worksheet_Visibility.PROJECT_WRITE]}"]`
          );
          break;
      }
      await sheetStore.fetchWorksheetList(filter.join(" && "));

      const folderPathes = new Set<string>([]);
      for (const worksheet of sheetList.value) {
        for (const path of getPathesForWorksheet(worksheet)) {
          folderPathes.add(path);
        }
      }
      folderContext.mergeFolders(folderPathes);
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
    sheetList,
    fetchSheetList,
    folderContext,
    getKeyForWorksheet,
    getFoldersForWorksheet,
    getPathesForWorksheet,
    getPwdForWorksheet,
  };
};

export type SheetContext = {
  view: Ref<SheetViewMode>;
  views: Record<SheetViewMode, ReturnType<typeof useSheetTreeByView>>;
  filter: Ref<WorksheetFilter>;
  filterChanged: ComputedRef<boolean>;
  expandedKeys: Ref<Set<string>>;
  selectedKeys: Ref<string[]>;
  editingNode: Ref<{ node: WorsheetFolderNode; rawLabel: string } | undefined>;
  isWorksheetCreator: (worksheet: Worksheet) => boolean;
  batchUpdateWorksheetFolders: (worksheets: Worksheet[]) => Promise<void>;
};

export const KEY = Symbol("bb.sql-editor.sheet") as InjectionKey<SheetContext>;

export const useSheetContext = () => {
  return inject(KEY)!;
};

export const useSheetContextByView = (view: SheetViewMode) => {
  const context = useSheetContext();
  return context.views[view];
};

export const revealWorksheets = (
  node: WorsheetFolderNode,
  callback: (node: WorsheetFolderNode) => Worksheet | undefined
): Worksheet[] => {
  if (node.worksheet) {
    const worksheet = callback(node);
    return worksheet ? [worksheet] : [];
  }

  const worksheets: Worksheet[] = [];
  for (const child of node.children) {
    worksheets.push(...revealWorksheets(child, callback));
  }
  return worksheets;
};

export const provideSheetContext = () => {
  const me = useCurrentUserV1();
  const tabStore = useSQLEditorTabStore();
  const worksheetV1Store = useWorkSheetStore();

  const initialFilter = computed(
    (): WorksheetFilter => ({
      keyword: "",
      showShared: true,
      showMine: true,
      showDraft: true,
      onlyShowStarred: false,
    })
  );
  const filter = useDynamicLocalStorage<WorksheetFilter>(
    computed(() => `bb.sql-editor.worksheet-filter.${me.value.name}`),
    {
      ...initialFilter.value,
    }
  );
  const filterChanged = computed(
    () => !isEqual(filter.value, initialFilter.value)
  );

  const views = {
    my: useSheetTreeByView(
      "my",
      computed(() => filter.value)
    ),
    shared: useSheetTreeByView(
      "shared",
      computed(() => filter.value)
    ),
  };

  const expandedKeys = useDynamicLocalStorage<Set<string>>(
    computed(() => `bb.sql-editor.worksheet-tree-expand-keys.${me.value.name}`),
    new Set([
      ...Object.values(views).map((item) => item.folderContext.rootPath.value),
    ])
  );
  const selectedKeys = ref<string[]>([]);

  const isWorksheetCreator = (worksheet: Worksheet) => {
    return worksheet.creator === `users/${me.value.email}`;
  };

  // Only watch the worksheet name, not the entire tab object
  // This prevents unnecessary re-runs when tab statement/status/etc changes
  watch(
    () => tabStore.currentTab?.worksheet,
    async (worksheetName) => {
      selectedKeys.value = [];

      if (!worksheetName) {
        return;
      }

      const worksheet = worksheetV1Store.getWorksheetByName(worksheetName);
      if (!worksheet) {
        return;
      }

      const isCreator = isWorksheetCreator(worksheet);
      const view: SheetViewMode = isCreator ? "my" : "shared";
      const viewContext = views[view];

      if (!viewContext) {
        return;
      }

      const key = viewContext.getKeyForWorksheet(worksheet);
      selectedKeys.value = [key];

      // Expand all parent folders to make the selected node visible
      for (const path of viewContext.getPathesForWorksheet(worksheet)) {
        expandedKeys.value.add(path);
      }

      // Scroll the selected node into view
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

  const batchUpdateWorksheetFolders = async (worksheets: Worksheet[]) => {
    await worksheetV1Store.batchUpsertWorksheetOrganizers(
      worksheets.map((worksheet) => ({
        organizer: {
          worksheet: worksheet.name,
          starred: worksheet.starred,
          folders: worksheet.folders,
        },
        updateMask: ["folders"],
      }))
    );
  };

  const context: SheetContext = {
    expandedKeys,
    selectedKeys,
    view: ref("my"),
    views,
    filter,
    filterChanged,
    isWorksheetCreator,
    editingNode: ref<
      { node: WorsheetFolderNode; rawLabel: string } | undefined
    >(),
    batchUpdateWorksheetFolders,
  };

  provide(KEY, context);

  return context;
};

export const extractWorksheetConnection = async (worksheet: Worksheet) => {
  const connection = emptySQLEditorConnection();
  if (worksheet.database) {
    try {
      const database = await useDatabaseV1Store().getOrFetchDatabaseByName(
        worksheet.database
      );
      connection.instance = database.instance;
      connection.database = database.name;
      setDefaultDataSourceForConn(connection, database);
    } catch {
      // Skip.
    }
  }
  return connection;
};

export const openWorksheetByName = async (
  name: string,
  editorContext: SQLEditorContext,
  forceNewTab = false
) => {
  const worksheet = await useWorkSheetStore().getOrFetchWorksheetByName(name);
  if (!worksheet) {
    return false;
  }

  if (!isWorksheetReadableV1(worksheet)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.access-denied"),
    });
    return false;
  }

  await editorContext.maybeSwitchProject(worksheet.project);
  const tabStore = useSQLEditorTabStore();
  const openingSheetTab = tabStore.tabList.find(
    (tab) => tab.worksheet === worksheet.name
  );

  const statement = getSheetStatement(worksheet);
  const connection = await extractWorksheetConnection(worksheet);

  const newTab: Partial<SQLEditorTab> = {
    connection,
    worksheet: worksheet.name,
    title: worksheet.title,
    statement,
    status: "CLEAN",
  };

  if (openingSheetTab) {
    // Switch to a sheet tab if it's open already.
    tabStore.setCurrentTabId(openingSheetTab.id);
    return true;
  } else if (forceNewTab) {
    tabStore.addTab(newTab, true /* beside */);
  } else {
    // Open the sheet in a "temp" tab otherwise.
    tabStore.addTab(newTab);
  }

  return true;
};

export const addNewSheet = () => {
  const tabStore = useSQLEditorTabStore();
  const curr = tabStore.currentTab;
  const connection = curr ? { ...curr.connection } : emptySQLEditorConnection();
  const title = suggestedTabTitleForSQLEditorConnection(connection);
  tabStore.addTab({
    title,
    connection,
    status: "NEW",
  });
};
