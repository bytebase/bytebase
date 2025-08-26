import Emittery from "emittery";
import { storeToRefs } from "pinia";
import type { InjectionKey, Ref } from "vue";
import { inject, provide, ref, computed } from "vue";
import { t } from "@/plugins/i18n";
import {
  pushNotification,
  useDatabaseV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useWorkSheetStore,
} from "@/store";
import type { SQLEditorTab } from "@/types";
import type { Worksheet } from "@/types/proto-es/v1/worksheet_service_pb";
import {
  emptySQLEditorConnection,
  getSheetStatement,
  isWorksheetReadableV1,
  suggestedTabTitleForSQLEditorConnection,
} from "@/utils";
import { setDefaultDataSourceForConn } from "../EditorCommon";
import type { SQLEditorContext } from "../context";
import type { SheetViewMode } from "./types";

type SheetEvents = Emittery<{
  refresh: { views: SheetViewMode[] };
  "add-sheet": undefined;
}>;

const useSheetListByView = (viewMode: SheetViewMode) => {
  const sheetStore = useWorkSheetStore();

  const { project } = storeToRefs(useSQLEditorStore());

  const isInitialized = ref(false);
  const isLoading = ref(false);
  const sheetList = computed(() => {
    let list = [];
    switch (viewMode) {
      case "my":
        list = sheetStore.myWorksheetList;
        break;
      case "shared":
        list = sheetStore.sharedWorksheetList;
        break;
      case "starred":
        list = sheetStore.starredWorksheetList;
        break;
    }
    return list.filter((worksheet) => {
      return !project.value || worksheet.project === project.value;
    });
  });

  const fetchSheetList = async () => {
    isLoading.value = true;
    try {
      switch (viewMode) {
        case "my":
          await sheetStore.fetchMyWorksheetList();
          break;
        case "shared":
          await sheetStore.fetchSharedWorksheetList();
          break;
        case "starred":
          await sheetStore.fetchStarredWorksheetList();
          break;
      }

      isInitialized.value = true;
    } finally {
      isLoading.value = false;
    }
  };

  return {
    isInitialized,
    isLoading,
    sheetList,
    fetchSheetList,
  };
};

export type SheetContext = {
  showPanel: Ref<boolean>;
  isFetching: Ref<boolean>;
  view: Ref<SheetViewMode>;
  views: Record<SheetViewMode, ReturnType<typeof useSheetListByView>>;
  events: SheetEvents;
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
  const context: SheetContext = {
    showPanel: ref(false),
    isFetching: ref(false),
    view: ref("my"),
    views: {
      my: useSheetListByView("my"),
      shared: useSheetListByView("shared"),
      starred: useSheetListByView("starred"),
    },
    events: new Emittery(),
  };

  context.events.on("refresh", () => {
    // Nothing todo
  });

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
