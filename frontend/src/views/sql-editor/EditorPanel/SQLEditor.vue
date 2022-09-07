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
import { debounce } from "lodash-es";
import { computed, defineEmits, nextTick, ref, watch } from "vue";

import {
  useInstanceStore,
  useTabStore,
  useSQLEditorStore,
  useDatabaseStore,
  useTableStore,
  useSheetStore,
  useInstanceById,
} from "@/store";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";
import { Database, SQLDialect, Table, UNKNOWN_ID } from "@/types";

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

const sqlCode = computed(() => tabStore.currentTab.statement || "");
const selectedInstance = useInstanceById(
  computed(() => sqlEditorStore.connectionContext.instanceId)
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
watch(
  currentTabId,
  () => {
    isSwitchingTab.value = true;
    nextTick(() => {
      isSwitchingTab.value = false;
    });
  },
  { immediate: true }
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

const handleChange = (value: string) => {
  // When we are switching between tabs, the MonacoEditor emits a 'change'
  // event, but we shouldn't update the current tab;
  if (isSwitchingTab.value) {
    return;
  }

  tabStore.updateCurrentTab({
    statement: value,
    isModified: true,
  });
};

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

  // Prepare auto-completion context when selected instance changed.
  watch(
    [
      () => sqlEditorStore.connectionContext.instanceId,
      () => sqlEditorStore.connectionContext.databaseId,
      selectedDialect,
    ],
    async ([instanceId, databaseId, dialect]) => {
      let databaseList: Database[] = [];
      let tableList: Table[] = [];

      const finish = () => {
        editorRef.value?.setEditorAutoCompletionContext(
          databaseList,
          tableList
        );
      };

      if (instanceId === UNKNOWN_ID) {
        // Don't go further if we are not connected to an instance.
        return finish();
      }

      if (dialect === "mysql") {
        // Prepare all databases and all tables for mysql
        databaseList = databaseStore.getDatabaseListByInstanceId(instanceId);
        const requests = Promise.all(
          databaseList.map((db) =>
            tableStore.getOrFetchTableListByDatabaseId(db.id)
          )
        );
        tableList = (await requests).flat();
      } else {
        // A PostgreSQL connection context must be database-scoped.
        if (databaseId === UNKNOWN_ID) {
          return finish();
        }
        // Prepare the selected database and tables for PostgreSQL
        databaseList = [databaseStore.getDatabaseById(databaseId)];
        tableList = await tableStore.getOrFetchTableListByDatabaseId(
          databaseId
        );
      }

      finish();
    },
    { immediate: true }
  );
};
</script>
