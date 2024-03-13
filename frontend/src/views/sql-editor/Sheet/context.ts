import Emittery from "emittery";
import { InjectionKey, Ref, inject, provide, ref, computed } from "vue";
import { t } from "@/plugins/i18n";
import {
  pushNotification,
  useSQLEditorTabStore,
  useWorkSheetStore,
} from "@/store";
import { SQLEditorTab } from "@/types";
import {
  emptySQLEditorConnection,
  getSheetStatement,
  isWorksheetReadableV1,
  suggestedTabTitleForSQLEditorConnection,
} from "@/utils";
import { SQLEditorContext } from "../context";
import { SheetViewMode } from "./types";

type SheetEvents = Emittery<{
  refresh: { views: SheetViewMode[] };
  "add-sheet": undefined;
}>;

const useSheetListByView = (viewMode: SheetViewMode) => {
  const sheetStore = useWorkSheetStore();

  const isInitialized = ref(false);
  const isLoading = ref(false);
  const sheetList = computed(() => {
    switch (viewMode) {
      case "my":
        return sheetStore.mySheetList;
      case "shared":
        return sheetStore.sharedSheetList;
      case "starred":
        return sheetStore.starredSheetList;
    }
    // Only to make TypeScript happy
    throw "Should never reach this line";
  });

  const fetchSheetList = async () => {
    isLoading.value = true;
    try {
      switch (viewMode) {
        case "my":
          await sheetStore.fetchMySheetList();
          break;
        case "shared":
          await sheetStore.fetchSharedSheetList();
          break;
        case "starred":
          await sheetStore.fetchStarredSheetList();
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

  context.events.on("refresh", ({ views }) => {
    // Nothing todo
  });

  provide(KEY, context);

  return context;
};

export const openSheet = async (
  name: string,
  editorContext: SQLEditorContext,
  worksheetContext: SheetContext,
  forceNewTab = false
) => {
  const cleanup = () => {
    worksheetContext.isFetching.value = false;
  };

  worksheetContext.isFetching.value = true;
  const sheet = await useWorkSheetStore().getOrFetchSheetByName(name);
  if (!sheet) {
    cleanup();
    return false;
  }

  if (!isWorksheetReadableV1(sheet)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.access-denied"),
    });
    cleanup();
    return false;
  }

  cleanup();

  await editorContext.maybeSwitchProject(sheet.project);
  const tabStore = useSQLEditorTabStore();
  const openingSheetTab = tabStore.tabList.find(
    (tab) => tab.sheet === sheet.name
  );

  const statement = getSheetStatement(sheet);
  // Won't set connection to the worksheet's database
  // since we are considering to unbind worksheets and databases
  // const connection = emptySQLEditorConnection();
  // if (sheet.database) {
  //   try {
  //     const database = await useDatabaseV1Store().getOrFetchDatabaseByName(
  //       sheet.database,
  //       true /* silent */
  //     );
  //     connection.instance = database.instance;
  //     connection.database = database.name;
  //   } catch {
  //     // Skip.
  //   }
  // }

  const newTab: Partial<SQLEditorTab> = {
    sheet: sheet.name,
    title: sheet.title,
    statement,
    status: "CLEAN",
    // connection,
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
