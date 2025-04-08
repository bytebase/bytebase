import { head } from "lodash-es";
import { storeToRefs } from "pinia";
import { computed, inject, provide, watch, type InjectionKey } from "vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
  useTabViewStateStore,
} from "@/store";
import {
  type EditorPanelView,
  type EditorPanelViewState as ViewState,
} from "@/types";
import {
  instanceV1SupportsExternalTable,
  instanceV1SupportsPackage,
  instanceV1SupportsSequence,
} from "@/utils";

const KEY = Symbol(
  "bb.sql-editor.editor-panel"
) as InjectionKey<EditorPanelContext>;

export const provideCurrentTabViewStateContext = () => {
  const tabStore = useSQLEditorTabStore();
  const tabViewStateStore = useTabViewStateStore();
  const { currentTab: tab } = storeToRefs(tabStore);

  const { instance } = useConnectionOfCurrentSQLEditorTab();

  const viewState = computed(() => {
    if (!tab.value) return undefined;
    return tabViewStateStore.getViewState(tab.value.id);
  });

  const availableViews = computed(() => {
    const views: EditorPanelView[] = [
      "CODE",
      "INFO",
      "TABLES",
      "VIEWS",
      "FUNCTIONS",
      "PROCEDURES",
    ];
    if (instanceV1SupportsSequence(instance.value)) {
      views.push("SEQUENCES");
    }
    if (instanceV1SupportsExternalTable(instance.value)) {
      views.push("EXTERNAL_TABLES");
    }
    if (instanceV1SupportsPackage(instance.value)) {
      views.push("PACKAGES");
    }
    views.push("DIAGRAM");
    return views;
  });

  const selectedSchemaName = computed({
    get() {
      return viewState.value?.schema;
    },
    set(schema) {
      if (schema === undefined) return;
      updateViewState({
        schema,
      });
    },
  });

  const updateViewState = (patch: Partial<ViewState>) => {
    if (!tab.value) return;
    tabViewStateStore.updateViewState(tab.value.id, patch);
  };

  watch(
    [() => viewState.value?.view, availableViews],
    ([view, availableViews]) => {
      if (view && !availableViews.includes(view)) {
        updateViewState({
          view: head(availableViews),
        });
      }
    },
    { immediate: true }
  );

  watch(
    () => tab.value?.connection.schema,
    (schema) => {
      if (schema === undefined) return;
      if (!viewState.value) return;
      if (viewState.value.schema === schema) return;
      viewState.value.schema = schema;
    },
    { immediate: true }
  );

  watch(
    () => viewState.value?.schema,
    (schema) => {
      if (!schema) return;
      if (!tab.value) return;
      if (tab.value.connection.schema === undefined) return; // if schema chooser is "ALL", don't sync
      if (tab.value.connection.schema === schema) return;
      tab.value.connection.schema = schema;
    },
    { immediate: true }
  );

  const context = {
    viewState,
    selectedSchemaName,
    availableViews,
    updateViewState,
  };

  provide(KEY, context);

  return context;
};

export const useCurrentTabViewStateContext = () => {
  return inject(KEY)!;
};

export type EditorPanelContext = ReturnType<
  typeof provideCurrentTabViewStateContext
>;
