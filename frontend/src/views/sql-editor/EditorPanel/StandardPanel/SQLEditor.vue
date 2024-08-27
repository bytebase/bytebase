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
        scene: 'query',
      }"
      @update:content="handleUpdateStatement"
      @select-content="handleUpdateSelectedStatement"
      @update:selection="handleUpdateSelection"
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
import type { Selection as MonacoSelection } from "@/components/MonacoEditor";
import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";
import {
  extensionNameOfLanguage,
  formatEditorContent,
} from "@/components/MonacoEditor/utils";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import {
  useUIStateStore,
  useWorkSheetAndTabStore,
  useSQLEditorTabStore,
  useConnectionOfCurrentSQLEditorTab,
} from "@/store";
import type { SQLDialect, SQLEditorQueryParams, SQLEditorTab } from "@/types";
import { dialectOfEngineV1 } from "@/types";
import { Advice_Status, type Advice } from "@/types/proto/v1/sql_service";
import { useInstanceV1EditorLanguage } from "@/utils";
import { useSQLEditorContext } from "../../context";

const emit = defineEmits<{
  (e: "execute", params: SQLEditorQueryParams): void;
}>();

const tabStore = useSQLEditorTabStore();
const sheetAndTabStore = useWorkSheetAndTabStore();
const uiStateStore = useUIStateStore();
const { events: editorEvents } = useSQLEditorContext();
const { currentTab, isSwitchingTab } = storeToRefs(tabStore);
const pendingFormatContentCommand = ref(false);
const { events: executeSQLEvents } = useExecuteSQL();

const content = computed(() => currentTab.value?.statement ?? "");
const advices = computed((): AdviceOption[] => {
  const tab = currentTab.value;
  if (!tab) {
    return [];
  }
  return tab.editorState.advices;
});
const { instance, database } = useConnectionOfCurrentSQLEditorTab();
const language = useInstanceV1EditorLanguage(instance);
const dialect = computed((): SQLDialect => {
  const engine = instance.value.engine;
  return dialectOfEngineV1(engine);
});
const readonly = computed(() => sheetAndTabStore.isReadOnly);

const filename = computed(() => {
  const name = currentTab.value?.id || uuidv1();
  const ext = extensionNameOfLanguage(language.value);
  return `${name}.${ext}`;
});

const handleUpdateStatement = (value: string) => {
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

const handleUpdateSelectedStatement = (value: string) => {
  tabStore.updateCurrentTab({
    selectedStatement: value,
  });
};

const handleUpdateSelection = (selection: MonacoSelection | null) => {
  const tab = currentTab.value;
  if (!tab) return;
  tabStore.updateCurrentTab({
    editorState: {
      ...tab.editorState,
      selection,
    },
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
    selection: tab.editorState.selection,
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
const updateAdvices = (
  tab: SQLEditorTab,
  params: SQLEditorQueryParams,
  advices: Advice[]
) => {
  const withOffset = (advice: Advice) => {
    let line = advice.line;
    let column = advice.column + 1;
    const { selection } = params;
    if (!selection) {
      return { line, column };
    }
    if (
      selection.endLineNumber > selection.startLineNumber ||
      selection.endColumn > selection.startColumn
    ) {
      if (line === 1) {
        column += selection.startColumn - 1;
      }
      line += selection.startLineNumber - 1;
    }
    return { line, column };
  };
  tab.editorState.advices = advices.map<AdviceOption>((advice) => {
    const { line, column } = withOffset(advice);
    const code = advice.code;
    const source = [`L${line}:C${column - 1}`];
    if (code > 0) {
      source.unshift(`(${code})`);
    }
    if (advice.title) {
      source.unshift(advice.title);
    }
    return {
      severity: advice.status === Advice_Status.ERROR ? "ERROR" : "WARNING",
      message: advice.content,
      source: source.join(" "),
      startLineNumber: line,
      endLineNumber: line,
      startColumn: column,
      endColumn: column,
    };
  });
};

useEmitteryEventListener(editorEvents, "format-content", () => {
  pendingFormatContentCommand.value = true;
});

useEmitteryEventListener(
  executeSQLEvents,
  "update:advices",
  ({ tab, params, advices }) => {
    if (tab.id !== currentTab.value?.id) return;
    updateAdvices(tab, params, advices);
  }
);
</script>
