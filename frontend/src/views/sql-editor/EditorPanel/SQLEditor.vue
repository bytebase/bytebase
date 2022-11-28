<template>
  <div
    class="w-full h-auto flex-grow flex flex-col justify-start items-start overflow-scroll"
  >
    <MonacoEditor
      ref="editorRef"
      v-model:value="sqlCode"
      class="w-full h-full"
      :dialect="selectedDialect"
      :readonly="readonly"
      @change="handleChange"
      @change-selection="handleChangeSelection"
      @save="handleSaveSheet"
      @ready="handleEditorReady"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, defineEmits, nextTick, ref, watch, watchEffect } from "vue";

import {
  useInstanceStore,
  useTabStore,
  useSQLEditorStore,
  useDatabaseStore,
  useTableStore,
  useSheetStore,
  useInstanceById,
} from "@/store";
import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";
import type { ExecuteConfig, ExecuteOption, SQLDialect } from "@/types";

const emit = defineEmits<{
  (e: "save-sheet", content?: string): void;
  (
    e: "execute",
    sql: string,
    config: ExecuteConfig,
    option?: ExecuteOption
  ): void;
}>();

const instanceStore = useInstanceStore();
const tabStore = useTabStore();
const databaseStore = useDatabaseStore();
const tableStore = useTableStore();
const sqlEditorStore = useSQLEditorStore();
const sheetStore = useSheetStore();

const editorRef = ref<InstanceType<typeof MonacoEditor>>();

const sqlCode = computed(() => tabStore.currentTab.statement);
const selectedInstance = useInstanceById(
  computed(() => tabStore.currentTab.connection.instanceId)
);
const selectedInstanceEngine = computed(() => {
  return instanceStore.formatEngine(selectedInstance.value);
});
const selectedDialect = computed((): SQLDialect => {
  const engine = selectedInstanceEngine.value;
  if (engine === "PostgreSQL") {
    return "postgresql";
  }
  return "mysql";
});
const readonly = computed(() => sheetStore.isReadOnly);
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
  () => {
    if (sqlEditorStore.shouldFormatContent) {
      editorRef.value?.formatEditorContent();
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
  emit("save-sheet");
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
      const databaseList = databaseStore.getDatabaseListByInstanceId(
        selectedInstance.value.id
      );
      const tableList = databaseList
        .map((item) => tableStore.getTableListByDatabaseId(item.id))
        .flat();

      editorRef.value?.setEditorAutoCompletionContext(databaseList, tableList);
    }
  });
};
</script>
