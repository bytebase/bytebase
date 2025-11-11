import { useDebounceFn } from "@vueuse/core";
import Emittery from "emittery";
import { orderBy } from "lodash-es";
import type { TreeOption } from "naive-ui";
import type { InjectionKey, Ref, ComputedRef } from "vue";
import { inject, provide, ref, computed, watch } from "vue";
import { t } from "@/plugins/i18n";
import {
  pushNotification,
  useDatabaseV1Store,
  useSQLEditorTabStore,
  useWorkSheetStore,
  useCurrentUserV1,
} from "@/store";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import type { SQLEditorTab } from "@/types";
import {
  Worksheet_Visibility,
  type Worksheet,
} from "@/types/proto-es/v1/worksheet_service_pb";
import {
  emptySQLEditorConnection,
  getSheetStatement,
  isWorksheetReadableV1,
  extractWorksheetUID,
  suggestedTabTitleForSQLEditorConnection,
} from "@/utils";
import { useDynamicLocalStorage } from "@/utils";
import { setDefaultDataSourceForConn } from "../EditorCommon";
import type { SQLEditorContext } from "../context";
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

  const expandedKeys = useDynamicLocalStorage<Set<string>>(
    computed(
      () =>
        `bb.sql-editor.worksheet-tree-expand-keys.${viewMode}.${me.value.name}`
    ),
    new Set(["/"])
  );

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
    key: "/",
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

  const getKeyForWorksheet = (worksheet: Worksheet): string => {
    return folderContext.ensureFolderPath(
      [
        ...worksheet.folders,
        `worksheets-${extractWorksheetUID(worksheet.name)}`,
      ].join("/")
    );
  };

  const buildTree = (parent: WorsheetFolderNode, worksheets: Worksheet[]) => {
    parent.children = folderContext
      .listSubFolders(parent.key)
      .map((folder) => ({
        isLeaf: false,
        children: [],
        key: folder,
        label: folder.split("/").slice(-1)[0],
        editable: true,
      }));

    for (const childNode of parent.children) {
      buildTree(childNode, worksheets);
    }

    const sheets = orderBy(
      worksheets.filter(
        (worksheet) =>
          folderContext.ensureFolderPath(worksheet.folders.join("/")) ===
          parent.key
      ),
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

    if (sheets.length === 0) {
      parent.empty = parent.children.every((child) => child.empty);
    }
  };

  const rebuildTree = useDebounceFn(() => {
    sheetTree.value = getRootTreeNode();
    buildTree(sheetTree.value, sheetList.value);
    events.emit("on-built", { viewMode });
  }, DEBOUNCE_SEARCH_DELAY);

  watch(
    [() => folderContext.folders.value, () => sheetList.value],
    () => {
      rebuildTree();
    },
    { deep: true }
  );

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
        for (let i = 0; i < worksheet.folders.length; i++) {
          folderPathes.add(
            folderContext.ensureFolderPath(
              worksheet.folders.slice(0, i + 1).join("/")
            )
          );
        }
      }
      folderContext.mergeFolders([...folderPathes]);
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
    sheetList,
    expandedKeys,
    fetchSheetList,
    folderContext,
    getKeyForWorksheet,
  };
};

export type SheetContext = {
  isFetching: Ref<boolean>;
  view: Ref<SheetViewMode>;
  views: Record<SheetViewMode, ReturnType<typeof useSheetTreeByView>>;
  filter: Ref<WorksheetFilter>;
  initialFilter: ComputedRef<WorksheetFilter>;
};

export const KEY = Symbol("bb.sql-editor.sheet") as InjectionKey<SheetContext>;

export const useSheetContext = () => {
  return inject(KEY)!;
};

export const useSheetContextByView = (view: SheetViewMode) => {
  const context = useSheetContext();
  return context.views[view];
};

export const provideSheetContext = () => {
  const me = useCurrentUserV1();

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

  const context: SheetContext = {
    isFetching: ref(false),
    view: ref("my"),
    views: {
      my: useSheetTreeByView(
        "my",
        computed(() => filter.value)
      ),
      shared: useSheetTreeByView(
        "shared",
        computed(() => filter.value)
      ),
    },
    filter,
    initialFilter,
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
  worksheetContext: SheetContext,
  forceNewTab = false
) => {
  const cleanup = () => {
    worksheetContext.isFetching.value = false;
  };

  worksheetContext.isFetching.value = true;
  const worksheet = await useWorkSheetStore().getOrFetchWorksheetByName(name);
  if (!worksheet) {
    cleanup();
    return false;
  }

  if (!isWorksheetReadableV1(worksheet)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.access-denied"),
    });
    cleanup();
    return false;
  }

  cleanup();

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
