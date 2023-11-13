<template>
  <div
    class="w-full h-auto flex-grow flex flex-col justify-start items-start overflow-auto"
  >
    <MonacoEditorV2
      ref="editorRef"
      class="w-full h-full"
      :filename="filename"
      :content="content"
      :language="language"
      :dialect="dialect"
      :readonly="readonly"
      :advices="advices"
      @update:content="handleChange"
      @select-content="handleChangeSelection"
      @ready="handleEditorReady"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, defineEmits, nextTick, ref, watch } from "vue";
import {
  AdviceOption,
  IStandaloneCodeEditor,
  MonacoEditorV2,
  MonacoModule,
} from "@/components/MonacoEditor";
import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";
import { extensionNameOfLanguage } from "@/components/MonacoEditor/utils";
import {
  useTabStore,
  useSQLEditorStore,
  useUIStateStore,
  useInstanceV1ByUID,
  useSheetAndTabStore,
} from "@/store";
import {
  dialectOfEngineV1,
  ExecuteConfig,
  ExecuteOption,
  SQLDialect,
} from "@/types";
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
const sqlEditorStore = useSQLEditorStore();
const sheetAndTabStore = useSheetAndTabStore();
const uiStateStore = useUIStateStore();
const { events: editorEvents } = useSQLEditorContext();

const editorRef = ref<InstanceType<typeof MonacoEditor>>();

const content = computed(() => tabStore.currentTab.statement);
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
const instanceEngine = computed(() => {
  return formatEngineV1(selectedInstance.value);
});
const language = useInstanceV1EditorLanguage(selectedInstance);
const dialect = computed((): SQLDialect => {
  const engine = selectedInstance.value.engine;
  return dialectOfEngineV1(engine);
});
const readonly = computed(() => sheetAndTabStore.isReadOnly);
const currentTabId = computed(() => tabStore.currentTabId);
const isSwitchingTab = ref(false);

const filename = computed(
  () => `${tabStore.currentTab.id}.${extensionNameOfLanguage(language.value)}`
);

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

const runQueryAction = (explain = false) => {
  const tab = tabStore.currentTab;
  const query = tab.selectedStatement || tab.statement || "";
  emit("execute", query, { databaseType: instanceEngine.value }, { explain });
  uiStateStore.saveIntroStateByKey({
    key: "data.query",
    newState: true,
  });
};

const handleEditorReady = (
  monaco: MonacoModule,
  editor: IStandaloneCodeEditor
) => {
  editor.addAction({
    id: "RunQuery",
    label: "Run Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
    contextMenuGroupId: "operation",
    contextMenuOrder: 0,
    run: () => runQueryAction(false),
  });
  editor.addAction({
    id: "ExplainQuery",
    label: "Explain Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyE],
    contextMenuGroupId: "operation",
    contextMenuOrder: 0,
    run: () => runQueryAction(true),
  });

  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
    handleSaveSheet();
  });
};
</script>
