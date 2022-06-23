<template>
  <div
    class="w-full h-auto flex-grow flex flex-col justify-start items-start overflow-scroll"
  >
    <MonacoEditor
      ref="editorRef"
      v-model:value="sqlCode"
      :language="selectedLanguage"
      :readonly="readonly"
      @change="handleChange"
      @change-selection="handleChangeSelection"
      @run-query="handleRunQuery"
      @save="handleSaveSheet"
    />
  </div>
</template>

<script lang="ts" setup>
import { debounce } from "lodash-es";
import { computed, defineEmits, ref, watch } from "vue";
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
  () => selectedInstance.value,
  () => {
    if (selectedInstance.value) {
      const databaseList = databaseStore.getDatabaseListByInstanceId(
        selectedInstance.value.id
      );
      const tableList = databaseList
        .map((item) => tableStore.getTableListByDatabaseId(item.id))
        .flat();

      editorRef.value?.setEditorAutoCompletionContext(databaseList, tableList);
    }
  }
);

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

const handleRunQuery = async ({
  explain,
  query,
}: {
  explain: boolean;
  query: string;
}) => {
  await execute({ databaseType: selectedInstanceEngine.value }, { explain });
};

const handleSaveSheet = () => {
  emit("save-sheet");
};
</script>
