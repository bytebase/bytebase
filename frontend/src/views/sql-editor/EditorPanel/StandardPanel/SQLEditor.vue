<template>
  <div
    class="w-full h-full flex-grow flex flex-col justify-start items-start overflow-auto"
  >
    <MonacoEditor
      class="w-full h-full"
      :key="filename"
      ref="monacoEditorRef"
      :enable-decorations="true"
      :filename="filename"
      :content="content"
      :language="language"
      :dialect="dialect"
      :readonly="readonly"
      :advices="advices"
      :auto-complete-context="{
        instance: instance.name,
        database: database.name,
        schema: currentTab?.connection.schema,
        scene: 'query',
      }"
      @update:content="handleUpdateStatement"
      @select-content="handleUpdateSelectedStatement"
      @update:selection="handleUpdateSelection"
      @ready="handleEditorReady"
    >
      <template #corner-prefix>
        <UploadFileButton @upload="handleUploadFile" />
      </template>
    </MonacoEditor>
  </div>
</template>

<script lang="ts" setup>
import { type IDisposable } from "monaco-editor";
import { storeToRefs } from "pinia";
import { v1 as uuidv1 } from "uuid";
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue";
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
  positionWithOffset,
} from "@/components/MonacoEditor/utils";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { useAIActions } from "@/plugins/ai";
import { useAIContext } from "@/plugins/ai/logic";
import * as promptUtils from "@/plugins/ai/logic/prompt";
import {
  useUIStateStore,
  useWorkSheetAndTabStore,
  useSQLEditorTabStore,
  useConnectionOfCurrentSQLEditorTab,
} from "@/store";
import type { SQLDialect, SQLEditorQueryParams, SQLEditorTab } from "@/types";
import { dialectOfEngineV1 } from "@/types";
import { Advice_Status, type Advice } from "@/types/proto-es/v1/sql_service_pb";
import {
  nextAnimationFrame,
  useInstanceV1EditorLanguage,
  instanceV1AllowsExplain,
} from "@/utils";
import { useSQLEditorContext } from "../../context";
import UploadFileButton from "./UploadFileButton.vue";
import { activeSQLEditorRef } from "./state";

const emit = defineEmits<{
  (e: "execute", params: SQLEditorQueryParams): void;
  (e: "execute-in-new-tab", params: SQLEditorQueryParams): void;
}>();

const tabStore = useSQLEditorTabStore();
const sheetAndTabStore = useWorkSheetAndTabStore();
const uiStateStore = useUIStateStore();
const {
  showAIPanel,
  pendingInsertAtCaret,
  events: editorEvents,
} = useSQLEditorContext();
const { currentTab, isSwitchingTab } = storeToRefs(tabStore);
const AIContext = useAIContext();
const pendingFormatContentCommand = ref(false);
const pendingSetSelectionCommand = ref<{
  start: { line: number; column: number };
  end?: { line: number; column: number };
}>();
const { events: executeSQLEvents } = useExecuteSQL();
const monacoEditorRef = ref<InstanceType<typeof MonacoEditor>>();

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
  if (!tab || value === tab.statement) {
    return;
  }
  // Directly update the reactive tab for immediate UI feedback
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

const getActiveStatement = () => {
  const tab = tabStore.currentTab;
  if (!tab) {
    return "";
  }
  return monacoEditorRef.value?.getActiveStatement() || tab.statement || "";
};

watch(
  () => tabStore.currentTab?.editorState.selection,
  (selection) => {
    if (!selection || !activeSQLEditorRef.value) {
      return;
    }
    activeSQLEditorRef.value.setSelection(selection);
  }
);

const runQueryAction = ({
  explain = false,
  newTab = false,
}: {
  explain: boolean;
  newTab: boolean;
}) => {
  const tab = tabStore.currentTab;
  if (!tab) {
    return;
  }
  const statement = getActiveStatement();
  const params: SQLEditorQueryParams = {
    connection: { ...tab.connection },
    statement,
    engine: instance.value.engine,
    explain,
    selection: null,
  };
  if (!newTab) {
    params.selection = tab.editorState.selection;
    emit("execute", params);
  } else {
    emit("execute-in-new-tab", params);
  }
  uiStateStore.saveIntroStateByKey({
    key: "data.query",
    newState: true,
  });
};

const handleEditorReady = (
  monaco: MonacoModule,
  editor: IStandaloneCodeEditor
) => {
  activeSQLEditorRef.value = editor;

  editor.addAction({
    id: "RunQuery",
    label: "Run Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
    contextMenuGroupId: "operation",
    contextMenuOrder: 0,
    run: () => runQueryAction({ explain: false, newTab: false }),
  });
  editor.addAction({
    id: "RunQueryInNewTab",
    label: "Run Query in New Tab",
    keybindings: [
      monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.Enter,
    ],
    contextMenuGroupId: "operation",
    contextMenuOrder: 0,
    run: () => runQueryAction({ explain: false, newTab: true }),
  });
  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
    handleSaveSheet();
  });
  useAIActions(monaco, editor, AIContext, {
    actions: ["explain-code", "find-problems", "new-chat"],
    callback: async (action) => {
      // start new chat if AI panel is not open
      // continue current chat otherwise
      const newChat = !showAIPanel.value;

      showAIPanel.value = true;
      const tab = tabStore.currentTab;
      const statement = getActiveStatement();
      if (!statement) return;

      await nextAnimationFrame();
      if (action === "explain-code") {
        AIContext.events.emit("send-chat", {
          content: promptUtils.explainCode(statement, instance.value.engine),
          newChat,
        });
      }
      if (action === "find-problems") {
        AIContext.events.emit("send-chat", {
          content: promptUtils.findProblems(statement, instance.value.engine),
          newChat,
        });
      }
      if (action === "new-chat") {
        const statement = tab?.selectedStatement ?? "";
        if (!statement) return;
        const inputs = [
          "", // just an empty line
          promptUtils.wrapStatementMarkdown(statement),
        ];
        AIContext.events.emit("new-conversation", {
          input: inputs.join("\n"),
        });
      }
    },
  });

  let explainQueryAction: IDisposable | undefined;
  watch(
    () => instance.value.engine,
    () => {
      if (instanceV1AllowsExplain(instance.value)) {
        if (!editor.getAction("ExplainQuery")) {
          explainQueryAction = editor.addAction({
            id: "ExplainQuery",
            label: "Explain Query",
            keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyE],
            contextMenuGroupId: "operation",
            contextMenuOrder: 0,
            run: () => runQueryAction({ explain: true, newTab: false }),
          });
        }
      } else {
        explainQueryAction?.dispose();
      }
    },
    { immediate: true }
  );

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

  watch(
    pendingSetSelectionCommand,
    (range) => {
      if (range) {
        const start = range.start;
        const end = range.end ?? start;
        editor.setSelection({
          startLineNumber: start.line,
          startColumn: start.column,
          endLineNumber: end.line,
          endColumn: end.column,
        });
        editor.revealLineNearTop(start.line);
        editor.focus();
        nextTick(() => {
          pendingSetSelectionCommand.value = undefined;
        });
      }
    },
    { immediate: true }
  );

  watch(
    pendingInsertAtCaret,
    () => {
      const editor = activeSQLEditorRef.value;
      if (!editor) return;
      const text = pendingInsertAtCaret.value;
      if (!text) return;
      pendingInsertAtCaret.value = undefined;

      requestAnimationFrame(() => {
        const selection = editor.getSelection();
        const maxLineNumber = editor.getModel()?.getLineCount() ?? 0;
        const range =
          selection ??
          new monaco.Range(maxLineNumber + 1, 1, maxLineNumber + 1, 1);
        editor.executeEdits("bb.event.insert-at-caret", [
          {
            forceMoveMarkers: true,
            text,
            range,
          },
        ]);
        editor.focus();
        editor.revealLine(range.startLineNumber);
      });
    },
    { immediate: true }
  );
};
const updateAdvices = (
  tab: SQLEditorTab,
  params: SQLEditorQueryParams,
  advices: Advice[]
) => {
  tab.editorState.advices = advices.map<AdviceOption>((advice) => {
    const [startLine, startColumn] = positionWithOffset(
      advice.startPosition?.line ?? 1,
      advice.startPosition?.column ?? Number.MAX_SAFE_INTEGER,
      params.selection
    );
    const [endLine, endColumn] = positionWithOffset(
      advice.endPosition?.line ?? advice.startPosition?.line ?? 1,
      advice.endPosition?.column ??
        advice.startPosition?.column ??
        Number.MAX_SAFE_INTEGER,
      params.selection
    );
    const code = advice.code;
    const source = [`L${startLine}:C${startColumn}`];
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
      startLineNumber: startLine,
      endLineNumber: endLine,
      startColumn: startColumn,
      endColumn: endColumn,
    };
  });
};
const handleUploadFile = (content: string) => {
  const editor = activeSQLEditorRef.value;
  if (!editor) return;
  const tab = currentTab.value;
  if (!tab) return;
  if (tab.statement.trim() !== "") {
    content = "\n" + content;
  }
  const maxLineNumber = editor.getModel()?.getLineCount() ?? 0;
  editor.executeEdits("bb.event.upload-file", [
    {
      forceMoveMarkers: true,
      text: content,
      range: {
        startLineNumber: maxLineNumber + 1,
        startColumn: 1,
        endLineNumber: maxLineNumber + 1,
        endColumn: 1,
      },
    },
  ]);
  const newMaxLineNumber = editor.getModel()?.getLineCount() ?? 0;
  editor.revealLine(newMaxLineNumber);
};

useEmitteryEventListener(editorEvents, "format-content", () => {
  pendingFormatContentCommand.value = true;
});
useEmitteryEventListener(editorEvents, "set-editor-selection", (range) => {
  pendingSetSelectionCommand.value = range;
});

useEmitteryEventListener(
  executeSQLEvents,
  "update:advices",
  ({ tab, params, advices }) => {
    if (tab.id !== currentTab.value?.id) return;
    updateAdvices(tab, params, advices);
  }
);

onBeforeUnmount(() => {
  activeSQLEditorRef.value = undefined;
});

defineExpose({
  getActiveStatement,
});
</script>
