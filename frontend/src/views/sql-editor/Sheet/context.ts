import { useDebounceFn } from "@vueuse/core";
import Emittery from "emittery";
import { isEqual, orderBy } from "lodash-es";
import type { TreeOption } from "naive-ui";
import { storeToRefs } from "pinia";
import scrollIntoView from "scroll-into-view-if-needed";
import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, nextTick, provide, ref, watch } from "vue";
import { t } from "@/plugins/i18n";
import {
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useSQLEditorStore,
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

export interface WorksheetLikeItem {
  name: string;
  title: string;
  folders: string[];
  type: "worksheet" | "draft";
}

export interface WorksheetFolderNode extends TreeOption {
  key: string;
  label: string;
  editable: boolean;
  empty?: boolean;
  worksheet?: WorksheetLikeItem;
  children: WorksheetFolderNode[];
}

interface WorksheetFilter {
  keyword: string;
  onlyShowStarred: boolean;
  showMine: boolean;
  showShared: boolean;
  showDraft: boolean;
}

const convertToWorksheetLikeItem = (
  worksheet: Worksheet
): WorksheetLikeItem => {
  return {
    name: worksheet.name,
    title: worksheet.title,
    folders: worksheet.folders,
    type: "worksheet",
  };
};

const useSheetTreeByView = (
  viewMode: SheetViewMode,
  filter: ComputedRef<WorksheetFilter>,
  project: ComputedRef<string>
) => {
  const sheetStore = useWorkSheetStore();
  const folderContext = useFolderByView(viewMode, project);
  const me = useCurrentUserV1();
  const events: SheetTreeEvents = new Emittery();
  const tabStore = useSQLEditorTabStore();

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
    if (filter.value.onlyShowStarred) {
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
        return tabStore.tabList
          .filter((tab) => !tab.worksheet)
          .map((tab) => ({
            name: tab.id,
            title: tab.title,
            folders: [],
            type: "draft",
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
    return folderContext.ensureFolderPath(
      [folderContext.rootPath.value, ...worksheet.folders].join("/")
    );
  };

  const getKeyForWorksheet = (worksheet: WorksheetLikeItem): string => {
    return [
      getPwdForWorksheet(worksheet),
      `bytebase-${worksheet.type}-${worksheet.name.split("/").slice(-1)[0]}`,
    ].join("/");
  };

  const getFoldersForWorksheet = (path: string): string[] => {
    return path
      .replace(folderContext.rootPath.value, "")
      .split("/")
      .slice(0, -1) // the last element should be the getKeyForWorksheet(worksheet)
      .filter((p) => p);
  };

  const buildTree = (
    parent: WorksheetFolderNode,
    worksheetsByFolder: Map<string, WorksheetLikeItem[]>
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
    if (parent.key !== folderContext.rootPath.value) {
      parent.isLeaf = parent.children.length === 0;
    }

    return parent;
  };

  const folderTree = computed(() => buildTree(getRootTreeNode(), new Map()));

  const rebuildTree = useDebounceFn(() => {
    const folderPathes = new Set<string>([]);
    for (const worksheet of sheetLikeItemList.value) {
      for (const path of getPathesForWorksheet(worksheet)) {
        folderPathes.add(path);
      }
    }
    folderContext.mergeFolders(folderPathes);

    const worksheetsByFolder = new Map<string, WorksheetLikeItem[]>();
    for (const worksheet of sheetLikeItemList.value) {
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
  // Use deep comparison to avoid unnecessary triggers when array references change
  // but content is the same
  watch(
    [() => folderContext.folders.value, () => sheetLikeItemList.value],
    ([newFolders, newSheetList], [oldFolders, oldSheetList]) => {
      // Only rebuild if the actual content changed, not just the array reference
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
            [
              `project == "${project.value}"`,
              `creator == "users/${me.value.email}"`,
            ].join(" && ")
          );
          break;
        case "shared":
          await sheetStore.fetchWorksheetList(
            [
              `project == "${project.value}"`,
              `creator != "users/${me.value.email}"`,
              // TODO(ed): do we need the visibility filter?
              // If not provide it, the call will fetch all worksheet with read access.
              `visibility in ["${Worksheet_Visibility[Worksheet_Visibility.PROJECT_READ]}","${Worksheet_Visibility[Worksheet_Visibility.PROJECT_WRITE]}"]`,
            ].join(" && ")
          );
          break;
        default:
          return;
      }

      // const folderPathes = new Set<string>([]);
      // for (const worksheet of sheetLikeItemList.value) {
      //   for (const path of getPathesForWorksheet(worksheet)) {
      //     folderPathes.add(path);
      //   }
      // }
      // folderContext.mergeFolders(folderPathes);
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

export type SheetContext = {
  view: Ref<SheetViewMode>;
  viewContexts: Record<SheetViewMode, ReturnType<typeof useSheetTreeByView>>;
  filter: Ref<WorksheetFilter>;
  filterChanged: ComputedRef<boolean>;
  expandedKeys: Ref<Set<string>>;
  selectedKeys: Ref<string[]>;
  editingNode: Ref<{ node: WorksheetFolderNode; rawLabel: string } | undefined>;
  isWorksheetCreator: (worksheet: Worksheet) => boolean;
  batchUpdateWorksheetFolders: (
    worksheets: { name: string; folders: string[] }[]
  ) => Promise<void>;
};

export const KEY = Symbol("bb.sql-editor.sheet") as InjectionKey<SheetContext>;

export const useSheetContext = () => {
  return inject(KEY)!;
};

export const useSheetContextByView = (view: SheetViewMode) => {
  const context = useSheetContext();
  return context.viewContexts[view];
};

export const revealWorksheets = <T>(
  node: WorksheetFolderNode,
  callback: (node: WorksheetFolderNode) => T | undefined
): T[] => {
  if (node.worksheet) {
    const worksheet = callback(node);
    return worksheet ? [worksheet] : [];
  }

  const worksheets: T[] = [];
  for (const child of node.children) {
    worksheets.push(...revealWorksheets(child, callback));
  }
  return worksheets;
};

export const provideSheetContext = () => {
  const me = useCurrentUserV1();
  const tabStore = useSQLEditorTabStore();
  const worksheetV1Store = useWorkSheetStore();
  const { project } = storeToRefs(useSQLEditorStore());

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
    computed(
      () => `bb.sql-editor.${project.value}.worksheet-filter.${me.value.name}`
    ),
    {
      ...initialFilter.value,
    }
  );
  const filterChanged = computed(
    () => !isEqual(filter.value, initialFilter.value)
  );

  const viewContexts = {
    my: useSheetTreeByView(
      "my",
      computed(() => filter.value),
      computed(() => project.value)
    ),
    shared: useSheetTreeByView(
      "shared",
      computed(() => filter.value),
      computed(() => project.value)
    ),
    draft: useSheetTreeByView(
      "draft",
      computed(() => filter.value),
      computed(() => project.value)
    ),
  };

  const expandedKeys = useDynamicLocalStorage<Set<string>>(
    computed(
      () =>
        `bb.sql-editor.${project.value}.worksheet-tree-expand-keys.${me.value.name}`
    ),
    new Set([
      ...Object.values(viewContexts).map(
        (item) => item.folderContext.rootPath.value
      ),
    ])
  );
  const selectedKeys = ref<string[]>([]);

  const isWorksheetCreator = (worksheet: Worksheet) => {
    return worksheet.creator === `users/${me.value.email}`;
  };

  // Only watch the worksheet name, not the entire tab object
  // This prevents unnecessary re-runs when tab statement/status/etc changes
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
        worksheetLikeItem = convertToWorksheetLikeItem(worksheet);
        const isCreator = isWorksheetCreator(worksheet);
        viewMode = isCreator ? "my" : "shared";
      } else {
        worksheetLikeItem = {
          name: tabId,
          folders: [],
          title: "", // don't care about the name.
          type: "draft",
        };
      }

      const viewContext = viewContexts[viewMode];
      if (!viewContext) {
        return;
      }

      const key = viewContext.getKeyForWorksheet(worksheetLikeItem);
      selectedKeys.value = [key];

      // Expand all parent folders to make the selected node visible
      for (const path of viewContext.getPathesForWorksheet(worksheetLikeItem)) {
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
          starred: false, // don't care about the starred
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
    viewContexts,
    filter,
    filterChanged,
    isWorksheetCreator,
    editingNode: ref<
      { node: WorksheetFolderNode; rawLabel: string } | undefined
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
