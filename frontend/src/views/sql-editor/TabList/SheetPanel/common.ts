import { InjectionKey, Ref, inject, provide, ref } from "vue";
import { useSheetV1Store } from "@/store";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { getSheetIssueBacktracePayloadV1 } from "@/utils";

export const SheetViewModeList = ["my", "shared", "starred"] as const;
export type SheetViewMode = typeof SheetViewModeList[number];

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
  view: Ref<SheetViewMode>;
  views: Record<SheetViewMode, ReturnType<typeof useFetchSheetList>>;
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
    view: ref("my"),
    views: {
      my: useFetchSheetList("my"),
      shared: useFetchSheetList("shared"),
      starred: useFetchSheetList("starred"),
    },
  };
  provide(KEY, context);
};
