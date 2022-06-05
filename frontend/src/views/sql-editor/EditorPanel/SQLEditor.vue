<template>
  <MonacoEditor
    ref="editorRef"
    v-model:value="sqlCode"
    :language="selectedLanguage"
    @change="handleChange"
    @change-selection="handleChangeSelection"
    @run-query="handleRunQuery"
    @save="handleSaveSheet"
  />
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

watch(selectedInstance, () => {
  if (selectedInstance.value) {
    const databaseList = databaseStore.getDatabaseListByInstanceId(
      selectedInstance.value.id
    );
    const tableList = databaseList
      .map((item) => tableStore.getTableListByDatabaseId(item.id))
      .flat();

    editorRef.value?.setAutoCompletionContext(databaseList, tableList);
  }
});

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

const handleRunQuery = ({
  explain,
  query,
}: {
  explain: boolean;
  query: string;
}) => {
  execute({ databaseType: selectedInstanceEngine.value }, { explain });
};

const handleSaveSheet = () => {
  emit("save-sheet");
};
</script>
