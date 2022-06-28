<template>
  <div
    class="w-full h-auto flex-grow flex flex-col justify-start items-start overflow-scroll"
  >
    <MonacoEditor
      ref="editorRef"
      v-model:value="sqlCode"
      class="w-full h-full"
      :language="selectedLanguage"
      :readonly="readonly"
      @change="handleChange"
      @change-selection="handleChangeSelection"
      @save="handleSaveSheet"
      @ready="handleEditorReady"
    />
  </div>
</template>

<script lang="ts" setup>
import { debounce } from "lodash-es";
import { computed, defineEmits, ref, watch, watchEffect } from "vue";

import {
  useInstanceStore,
  useTabStore,
  useSQLEditorStore,
  useDatabaseStore,
  useTableStore,
  useSheetStore,
} from "@/store";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";

const emit = defineEmits<{
  (e: "save-sheet", content?: string): void;
}>();

const instanceStore = useInstanceStore();
const tabStore = useTabStore();
const databaseStore = useDatabaseStore();
const tableStore = useTableStore();
const sqlEditorStore = useSQLEditorStore();
const sheetStore = useSheetStore();

const editorRef = ref<InstanceType<typeof MonacoEditor>>();

const { execute } = useExecuteSQL();

const sqlCode = computed(() => tabStore.currentTab.statement);
const selectedInstance = computed(() => {
  const ctx = sqlEditorStore.connectionContext;
  return instanceStore.getInstanceById(ctx.instanceId);
});
const selectedInstanceEngine = computed(() => {
  return instanceStore.formatEngine(selectedInstance.value);
});
const selectedLanguage = computed(() => {
  const engine = selectedInstanceEngine.value;
  if (engine === "MySQL") {
    return "mysql";
  }
  if (engine === "PostgreSQL") {
    return "pgsql";
  }
  return "sql";
});
const readonly = computed(() => sheetStore.isReadOnly);

watch(
  () => sqlEditorStore.shouldSetContent,
  () => {
    if (sqlEditorStore.shouldSetContent) {
      editorRef.value?.setEditorContent(tabStore.currentTab.statement);
      sqlEditorStore.setShouldSetContent(false);
    }
  }
);

watch(
  () => sqlEditorStore.shouldFormatContent,
  () => {
    if (sqlEditorStore.shouldFormatContent) {
      editorRef.value?.formatEditorContent();
      sqlEditorStore.setShouldFormatContent(false);
    }
  }
);

const handleChange = debounce((value: string) => {
  tabStore.updateCurrentTab({
    statement: value,
    isSaved: false,
  });
}, 300);

const handleChangeSelection = debounce((value: string) => {
  tabStore.updateCurrentTab({
    selectedStatement: value,
  });
}, 300);

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
      await execute(query, { databaseType: selectedInstanceEngine.value });
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
      await execute(
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
