<template>
  <div
    class="w-full h-auto flex-grow flex flex-col justify-start items-start overflow-auto"
  >
    <MonacoEditor
      ref="editorRef"
      v-model:value="sqlCode"
      class="w-full h-full"
      :language="selectedLanguage"
      :dialect="selectedDialect"
      :readonly="readonly"
      :advices="advices"
      @change="handleChange"
      @change-selection="handleChangeSelection"
      @save="handleSaveSheet"
      @ready="handleEditorReady"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, defineEmits, nextTick, ref, watch, watchEffect } from "vue";
import { AdviceOption } from "@/components/MonacoEditor";
import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";
import {
  useTabStore,
  useSQLEditorStore,
  useDBSchemaV1Store,
  useUIStateStore,
  useDatabaseV1Store,
  useInstanceV1ByUID,
  useSheetAndTabStore,
} from "@/store";
import {
  ComposedDatabase,
  dialectOfEngineV1,
  ExecuteConfig,
  ExecuteOption,
  SQLDialect,
  UNKNOWN_ID,
} from "@/types";
import { TableMetadata } from "@/types/proto/v1/database_service";
import { formatEngineV1, useInstanceV1EditorLanguage } from "@/utils";
import { useSQLEditorContext } from "../context";

const emit = defineEmits<{
  (
    e: "execute",
    sql: string,
    config: ExecuteConfig,
    option?: ExecuteOption
  ): void;
}>();

const tabStore = useTabStore();
const databaseStore = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const sqlEditorStore = useSQLEditorStore();
const sheetAndTabStore = useSheetAndTabStore();
const uiStateStore = useUIStateStore();
const { events: editorEvents } = useSQLEditorContext();

const editorRef = ref<InstanceType<typeof MonacoEditor>>();

const sqlCode = computed(() => tabStore.currentTab.statement);
const advices = computed((): AdviceOption[] => {
  return (
    Array.from(tabStore.currentTab?.databaseQueryResultMap?.values() || [])
      .map((result) => result?.advices || [])
      .flat() ?? []
  ).map((advice) => ({
    severity: "ERROR",
    message: advice.content,
    startLineNumber: advice.line,
    endLineNumber: advice.line,
    startColumn: advice.column,
    endColumn: advice.column,
    source: advice.detail,
  }));
});
const { instance: selectedInstance } = useInstanceV1ByUID(
  computed(() => tabStore.currentTab.connection.instanceId)
);
const selectedDatabase = computed(() => {
  const uid = tabStore.currentTab.connection.databaseId;
  if (uid === String(UNKNOWN_ID)) return undefined;
  return databaseStore.getDatabaseByUID(uid);
});
const selectedInstanceEngine = computed(() => {
  return formatEngineV1(selectedInstance.value);
});
const selectedLanguage = useInstanceV1EditorLanguage(selectedInstance);
const selectedDialect = computed((): SQLDialect => {
  const engine = selectedInstance.value.engine;
  return dialectOfEngineV1(engine);
});
const readonly = computed(() => sheetAndTabStore.isReadOnly);
const currentTabId = computed(() => tabStore.currentTabId);
const isSwitchingTab = ref(false);

watch(currentTabId, () => {
  isSwitchingTab.value = true;
  nextTick(() => {
    isSwitchingTab.value = false;
  });
});

watch(
  () => sqlEditorStore.shouldFormatContent,
  async () => {
    if (sqlEditorStore.shouldFormatContent) {
      await editorRef.value?.formatEditorContent();
      sqlEditorStore.setShouldFormatContent(false);
    }
  }
);

const handleChange = (value: string) => {
  // When we are switching between tabs, the MonacoEditor emits a 'change'
  // event, but we shouldn't update the current tab;
  if (isSwitchingTab.value) {
    return;
  }
  if (value === tabStore.currentTab.statement) {
    return;
  }
  // Clear old advices when the statement is changed.
  tabStore.currentTab.databaseQueryResultMap?.forEach((result) => {
    result.advices = [];
  });
  tabStore.updateCurrentTab({
    statement: value,
    isSaved: false,
  });
};

const handleChangeSelection = (value: string) => {
  tabStore.updateCurrentTab({
    selectedStatement: value,
  });
};

const handleSaveSheet = () => {
  editorEvents.emit("save-sheet", { title: tabStore.currentTab.name });
};

const handleEditorReady = async () => {
  const monaco = await import("monaco-editor");
  editorRef.value?.editorInstance?.addAction({
    id: "RunQuery",
    label: "Run Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
    contextMenuGroupId: "operation",
    contextMenuOrder: 0,
    run: async () => {
      const typedValue = editorRef.value?.editorInstance?.getValue();
      const selectedValue = editorRef.value?.editorInstance
        ?.getModel()
        ?.getValueInRange(
          editorRef.value?.editorInstance?.getSelection() as any
        ) as string;
      const query = selectedValue || typedValue || "";
      await emit("execute", query, {
        databaseType: selectedInstanceEngine.value,
      });
      uiStateStore.saveIntroStateByKey({
        key: "data.query",
        newState: true,
      });
    },
  });

  editorRef.value?.editorInstance?.addAction({
    id: "ExplainQuery",
    label: "Explain Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyE],
    contextMenuGroupId: "operation",
    contextMenuOrder: 0,
    run: async () => {
      const typedValue = editorRef.value?.editorInstance?.getValue();
      const selectedValue = editorRef.value?.editorInstance
        ?.getModel()
        ?.getValueInRange(
          editorRef.value?.editorInstance?.getSelection() as any
        ) as string;
      const query = selectedValue || typedValue || "";
      await emit(
        "execute",
        query,
        { databaseType: selectedInstanceEngine.value },
        { explain: true }
      );
    },
  });

  watchEffect(() => {
    if (selectedInstance.value) {
      const databaseMap: Map<ComposedDatabase, TableMetadata[]> = new Map();

      const databaseList = selectedDatabase.value
        ? [selectedDatabase.value]
        : databaseStore.databaseListByInstance(selectedInstance.value.name);
      // Only provide auto-complete context for those opened database.
      for (const database of databaseList) {
        const tableList = dbSchemaStore.getTableList(database.name);
        if (tableList.length > 0) {
          databaseMap.set(database, tableList);
        }
      }
      const connectionScope = selectedDatabase.value ? "database" : "instance";
      editorRef.value?.setEditorAutoCompletionContextV1(
        databaseMap,
        connectionScope
      );
    }
  });
};
</script>
