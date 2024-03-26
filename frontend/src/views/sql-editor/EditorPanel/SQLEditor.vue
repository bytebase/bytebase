<template>
  <div
    class="w-full h-auto flex-grow flex flex-col justify-start items-start overflow-auto"
  >
    <MonacoEditor
      class="w-full h-full"
      :filename="filename"
      :content="content"
      :language="language"
      :dialect="dialect"
      :readonly="readonly"
      :advices="advices"
      :auto-complete-context="{
        instance: instance.name,
        database: database.name,
      }"
      @update:content="handleChange"
      @select-content="handleChangeSelection"
      @ready="handleEditorReady"
    />
  </div>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { v1 as uuidv1 } from "uuid";
import { computed, nextTick, ref, watch } from "vue";
import type {
  AdviceOption,
  IStandaloneCodeEditor,
  MonacoModule,
} from "@/components/MonacoEditor";
import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";
import {
  extensionNameOfLanguage,
  formatEditorContent,
} from "@/components/MonacoEditor/utils";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  useUIStateStore,
  useWorkSheetAndTabStore,
  useSQLEditorTabStore,
  useConnectionOfCurrentSQLEditorTab,
} from "@/store";
import type { SQLDialect, SQLEditorQueryParams } from "@/types";
import { dialectOfEngineV1 } from "@/types";
import { useInstanceV1EditorLanguage } from "@/utils";
import { useSQLEditorContext } from "../context";

const emit = defineEmits<{
  (e: "execute", params: SQLEditorQueryParams): void;
}>();

const tabStore = useSQLEditorTabStore();
const sheetAndTabStore = useWorkSheetAndTabStore();
const uiStateStore = useUIStateStore();
const { events: editorEvents } = useSQLEditorContext();
const { currentTab } = storeToRefs(tabStore);
const pendingFormatContentCommand = ref(false);

const content = computed(() => currentTab.value?.statement ?? "");
const advices = computed((): AdviceOption[] => {
  const tab = currentTab.value;
  if (!tab) {
    return [];
  }
  return (
    Array.from(tab.queryContext?.results.values() || [])
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
const { instance, database } = useConnectionOfCurrentSQLEditorTab();
const language = useInstanceV1EditorLanguage(instance);
const dialect = computed((): SQLDialect => {
  const engine = instance.value.engine;
  return dialectOfEngineV1(engine);
});
const readonly = computed(() => sheetAndTabStore.isReadOnly);
const currentTabId = computed(() => tabStore.currentTabId);
const isSwitchingTab = ref(false);

const filename = computed(() => {
  const name = currentTab.value?.id || uuidv1();
  const ext = extensionNameOfLanguage(language.value);
  return `${name}.${ext}`;
});

watch(currentTabId, () => {
  isSwitchingTab.value = true;
  nextTick(() => {
    isSwitchingTab.value = false;
  });
});

const handleChange = (value: string) => {
  // When we are switching between tabs, the MonacoEditor emits a 'change'
  // event, but we shouldn't update the current tab;
  if (isSwitchingTab.value) {
    return;
  }
  const tab = currentTab.value;
  if (!tab) {
    return;
  }
  if (value === tab.statement) {
    return;
  }
  // Clear old advices when the statement is changed.
  tab.queryContext?.results.forEach((result) => {
    result.advices = [];
  });
  tabStore.updateCurrentTab({
    statement: value,
    status: "DIRTY",
  });
};

const handleChangeSelection = (value: string) => {
  tabStore.updateCurrentTab({
    selectedStatement: value,
  });
};

const handleSaveSheet = () => {
  const tab = currentTab.value;
  if (!tab) {
    return;
  }
  editorEvents.emit("save-sheet", { tab });
};

const runQueryAction = (explain = false) => {
  const tab = tabStore.currentTab;
  if (!tab) {
    return;
  }
  const statement = tab.selectedStatement || tab.statement || "";
  emit("execute", {
    connection: { ...tab.connection },
    statement,
    engine: instance.value.engine,
    explain,
  });
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

  watch(
    pendingFormatContentCommand,
    (pending) => {
      if (pending) {
        formatEditorContent(editor, dialect.value);
        nextTick(() => {
          pendingFormatContentCommand.value = false;
        });
      }
    },
    { immediate: true }
  );
};
useEmitteryEventListener(editorEvents, "format-content", () => {
  pendingFormatContentCommand.value = true;
});
</script>
