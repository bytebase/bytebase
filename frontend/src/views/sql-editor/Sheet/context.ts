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
import type { Worksheet } from "@/types/proto/v1/worksheet_service";
import {
  emptySQLEditorConnection,
  getSheetStatement,
  isWorksheetReadableV1,
  suggestedTabTitleForSQLEditorConnection,
} from "@/utils";
import type { SQLEditorContext } from "../context";
import type { SheetViewMode } from "./types";

type SheetEvents = Emittery<{
  refresh: { views: SheetViewMode[] };
  "add-sheet": undefined;
}>;

const useSheetListByView = (viewMode: SheetViewMode) => {
  const sheetStore = useWorkSheetStore();

  const { currentProject } = storeToRefs(useSQLEditorStore());

  const isInitialized = ref(false);
  const isLoading = ref(false);
  const sheetList = computed(() => {
    let list = [];
    switch (viewMode) {
      case "my":
        list = sheetStore.mySheetList;
        break;
      case "shared":
        list = sheetStore.sharedSheetList;
        break;
      case "starred":
        list = sheetStore.starredSheetList;
        break;
    }
    return list.filter((worksheet) => {
      return (
        !currentProject.value || worksheet.project === currentProject.value.name
      );
    });
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

export const extractWorksheetConnection = (worksheet: Worksheet) => {
  const connection = emptySQLEditorConnection();
  if (worksheet.database) {
    try {
      const database = useDatabaseV1Store().getDatabaseByName(
        worksheet.database
      );
      connection.instance = database.instance;
      connection.database = database.name;
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

  const newTab: Partial<SQLEditorTab> = {
    connection: extractWorksheetConnection(sheet),
    sheet: sheet.name,
    title: sheet.title,
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
