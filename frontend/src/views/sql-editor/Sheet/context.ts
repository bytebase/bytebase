import Emittery from "emittery";
import { InjectionKey, Ref, inject, provide, ref, computed } from "vue";
import { t } from "@/plugins/i18n";
import {
  pushNotification,
  useInstanceV1Store,
  useSheetV1Store,
  useTabStore,
} from "@/store";
import { getInstanceAndDatabaseId } from "@/store/modules/v1/common";
import { AnyTabInfo, UNKNOWN_ID } from "@/types";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { emptyConnection, getSheetStatement, isSheetReadableV1 } from "@/utils";
import { SheetViewMode } from "./types";

type SheetEvents = Emittery<{
  refresh: { views: SheetViewMode[] };
}>;

const useSheetListByView = (viewMode: SheetViewMode) => {
  const sheetStore = useSheetV1Store();

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

export const openSheet = async (sheet: Sheet, forceNewTab = false) => {
  const tabStore = useTabStore();
  const openingSheetTab = tabStore.tabList.find(
    (tab) => tab.sheetName == sheet.name
  );

  if (!isSheetReadableV1(sheet)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.access-denied"),
    });
    return false;
  }
  const statement = getSheetStatement(sheet);
  const newTab: AnyTabInfo = {
    sheetName: sheet.name,
    name: sheet.title,
    statement,
  };
  if (openingSheetTab) {
    // Switch to a sheet tab if it's open already.
    tabStore.setCurrentTabId(openingSheetTab.id);
  } else if (forceNewTab) {
    tabStore.addTab(newTab, true /* beside */);
  } else {
    // Open the sheet in a "temp" tab otherwise.
    tabStore.selectOrAddTempTab(newTab);
  }

  let insId = String(UNKNOWN_ID);
  let dbId = String(UNKNOWN_ID);
  if (sheet.database) {
    const [instanceName, databaseId] = getInstanceAndDatabaseId(sheet.database);
    const ins = useInstanceV1Store().getInstanceByName(
      `instances/${instanceName}`
    );
    insId = ins.uid;
    dbId = databaseId;
  }

  tabStore.updateCurrentTab({
    sheetName: sheet.name,
    name: sheet.title,
    statement,
    isSaved: true,
    connection: {
      ...emptyConnection(),
      // TODO: legacy instance id.
      instanceId: insId,
      databaseId: dbId,
    },
  });

  return true;
};
