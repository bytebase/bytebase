import { InfoIcon } from "lucide-vue-next";
import { storeToRefs } from "pinia";
import type { VNodeChild } from "vue";
import { computed, type InjectionKey, inject, provide, watch } from "vue";
import {
  ExternalTableIcon,
  FunctionIcon,
  PackageIcon,
  ProcedureIcon,
  SequenceIcon,
  TableIcon,
  ViewIcon,
} from "@/components/Icon";
import { SchemaDiagramIcon } from "@/components/SchemaDiagram";
import { t } from "@/plugins/i18n";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
} from "@/store";
import {
  defaultViewState,
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
  const { currentTab: tab } = storeToRefs(tabStore);

  const { instance } = useConnectionOfCurrentSQLEditorTab();

  const viewState = computed(() => {
    return tab.value?.viewState;
  });

  const availableActions = computed(() => {
    const actions: {
      view: EditorPanelView;
      title: string;
      icon: () => VNodeChild;
    }[] = [
      {
        view: "INFO",
        title: t("common.info"),
        icon: () => <InfoIcon class="w-4 h-4" />,
      },
      {
        view: "TABLES",
        title: t("db.tables"),
        icon: () => <TableIcon class="w-4 h-4" />,
      },
      {
        view: "VIEWS",
        title: t("db.views"),
        icon: () => <ViewIcon class="w-4 h-4" />,
      },
      {
        view: "FUNCTIONS",
        title: t("db.functions"),
        icon: () => <FunctionIcon class="w-4 h-4" />,
      },
      {
        view: "PROCEDURES",
        title: t("db.procedures"),
        icon: () => <ProcedureIcon class="w-4 h-4" />,
      },
    ];
    if (instanceV1SupportsSequence(instance.value)) {
      actions.push({
        view: "SEQUENCES",
        title: t("db.sequences"),
        icon: () => <SequenceIcon class="w-4 h-4" />,
      });
    }
    if (instanceV1SupportsExternalTable(instance.value)) {
      actions.push({
        view: "EXTERNAL_TABLES",
        title: t("db.external-tables"),
        icon: () => <ExternalTableIcon class="w-4 h-4" />,
      });
    }
    if (instanceV1SupportsPackage(instance.value)) {
      actions.push({
        view: "PACKAGES",
        title: t("db.packages"),
        icon: () => <PackageIcon class="w-4 h-4" />,
      });
    }
    actions.push({
      view: "DIAGRAM",
      title: t("schema-diagram.self"),
      icon: () => <SchemaDiagramIcon class="w-4 h-4" />,
    });
    return actions;
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
    tabStore.updateTab(tab.value.id, {
      viewState: {
        ...defaultViewState(),
        ...tab.value.viewState,
        ...patch,
      },
    });
  };

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
    availableActions,
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
