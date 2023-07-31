import { InjectionKey, Ref, inject, provide, ref } from "vue";
import { t } from "@/plugins/i18n";
import Emittery from "emittery";

import {
  pushNotification,
  useInstanceV1Store,
  useSheetV1Store,
  useTabStore,
} from "@/store";
import { Sheet } from "@/types/proto/v1/sheet_service";
import {
  emptyConnection,
  getSheetIssueBacktracePayloadV1,
  isSheetReadableV1,
} from "@/utils";
import { UNKNOWN_ID } from "@/types";
import { getInstanceAndDatabaseId } from "@/store/modules/v1/common";

export const SheetViewModeList = ["my", "shared", "starred"] as const;
export type SheetViewMode = typeof SheetViewModeList[number];

type SheetPanelEvents = Emittery<{
  refresh: { views: SheetViewMode[] };
}>;

const useFetchSheetList = (viewMode: SheetViewMode) => {
  const sheetStore = useSheetV1Store();

  const isInitialized = ref(false);
  const isLoading = ref(false);
  const sheetList = ref<Sheet[]>([]);

  const fetchSheetList = async () => {
    isLoading.value = true;
    try {
      let list: Sheet[] = [];
      switch (viewMode) {
        case "my":
          list = await sheetStore.fetchMySheetList();
          break;
        case "shared":
          list = await sheetStore.fetchSharedSheetList();
          break;
        case "starred":
          list = await sheetStore.fetchStarredSheetList();
          break;
      }

      // Hide those sheets from issue.
      sheetList.value = list.filter((sheet) => {
        return !getSheetIssueBacktracePayloadV1(sheet);
      });

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

export type SheetPanelContext = {
  showPanel: Ref<boolean>;
  view: Ref<SheetViewMode>;
  views: Record<SheetViewMode, ReturnType<typeof useFetchSheetList>>;
  events: SheetPanelEvents;
};

export const KEY = Symbol(
  "bb.sql-editor.sheet-panel"
) as InjectionKey<SheetPanelContext>;

export const useSheetPanelContext = () => {
  return inject(KEY)!;
};

export const useSheetPanelContextByView = (view: SheetViewMode) => {
  const context = useSheetPanelContext();
  return context.views[view];
};

export const provideSheetPanelContext = () => {
  const context: SheetPanelContext = {
    showPanel: ref(false),
    view: ref("my"),
    views: {
      my: useFetchSheetList("my"),
      shared: useFetchSheetList("shared"),
      starred: useFetchSheetList("starred"),
    },
    events: new Emittery(),
  };

  context.events.on("refresh", ({ views }) => {
    views.forEach((view) => {
      // If the panel is shown and the view is current view, re-fetch immediately.
      if (context.showPanel.value && view === context.view.value) {
        context.views[view].fetchSheetList();
      } else {
        // Otherwise reset the view status to isInitialized=false
        context.views[view].isInitialized.value = false;
        context.views[view].sheetList.value = [];
      }
    });
  });

  provide(KEY, context);
};

export const openSheet = async (sheet: Sheet) => {
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
  if (openingSheetTab) {
    // Switch to a sheet tab if it's open already.
    tabStore.setCurrentTabId(openingSheetTab.id);
  } else {
    // Open the sheet in a "temp" tab otherwise.
    tabStore.selectOrAddTempTab();
  }

  let insId = String(UNKNOWN_ID);
  let dbId = String(UNKNOWN_ID);
  if (sheet.database) {
    const [instanceName, databaseId] = getInstanceAndDatabaseId(sheet.database);
    const ins = await useInstanceV1Store().getOrFetchInstanceByName(
      `instances/${instanceName}`
    );
    insId = ins.uid;
    dbId = databaseId;
  }

  tabStore.updateCurrentTab({
    sheetName: sheet.name,
    name: sheet.title,
    statement: new TextDecoder().decode(sheet.content),
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
