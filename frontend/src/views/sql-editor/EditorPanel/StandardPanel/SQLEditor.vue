<template>
  <div
    class="w-full h-full grow flex flex-col justify-start items-start overflow-auto"
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
import { type IDisposable, type IRange, Selection } from "monaco-editor";
import { storeToRefs } from "pinia";
import { v1 as uuidv1 } from "uuid";
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue";
import type {
  IStandaloneCodeEditor,
  MonacoModule,
  Selection as MonacoSelection,
} from "@/components/MonacoEditor";
import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";
import {
  extensionNameOfLanguage,
  formatEditorContent,
} from "@/components/MonacoEditor/utils";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useAIActions } from "@/plugins/ai";
import { useAIContext } from "@/plugins/ai/logic";
import * as promptUtils from "@/plugins/ai/logic/prompt";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
  useUIStateStore,
  useWorkSheetAndTabStore,
} from "@/store";
import type { SQLDialect, SQLEditorQueryParams } from "@/types";
import { dialectOfEngineV1 } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  instanceV1AllowsExplain,
  nextAnimationFrame,
  useInstanceV1EditorLanguage,
} from "@/utils";
import { useSQLEditorContext } from "../../context";
import { activeSQLEditorRef } from "./state";
import UploadFileButton from "./UploadFileButton.vue";

const emit = defineEmits<{
  (
    e: "execute",
    params: { params: SQLEditorQueryParams; newTab: boolean }
  ): void;
}>();

const tabStore = useSQLEditorTabStore();
const sheetAndTabStore = useWorkSheetAndTabStore();
const uiStateStore = useUIStateStore();
const {
  showAIPanel,
  pendingInsertAtCaret,
  events: editorEvents,
} = useSQLEditorContext();
const { currentTab } = storeToRefs(tabStore);
const AIContext = useAIContext();

const monacoEditorRef = ref<InstanceType<typeof MonacoEditor>>();

const content = computed(() => currentTab.value?.statement ?? "");
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
  () => currentTab.value?.editorState.selection?.toString(),
  () => {
    if (!tabStore.currentTab?.editorState.selection) {
      return;
    }
    activeSQLEditorRef.value?.setSelection(
      tabStore.currentTab.editorState.selection
    );
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
  }
  emit("execute", { params, newTab });
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
    contextMenuOrder: 1,
    run: () => runQueryAction({ explain: false, newTab: false }),
  });
  editor.addAction({
    id: "RunQueryInNewTab",
    label: "Run Query in New Tab",
    keybindings: [
      monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.Enter,
    ],
    contextMenuGroupId: "operation",
    contextMenuOrder: 1,
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
      const shouldShowAction =
        instanceV1AllowsExplain(instance.value) ||
        instance.value.engine === Engine.BIGQUERY;

      if (shouldShowAction) {
        const isBigQuery = instance.value.engine === Engine.BIGQUERY;
        const label = isBigQuery ? "Dry Run Query" : "Explain Query";

        // Remove existing action if label changed
        if (explainQueryAction) {
          explainQueryAction.dispose();
          explainQueryAction = undefined;
        }

        if (!editor.getAction("ExplainQuery")) {
          explainQueryAction = editor.addAction({
            id: "ExplainQuery",
            label: label,
            keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyE],
            contextMenuGroupId: "operation",
            contextMenuOrder: 0,
            run: () => runQueryAction({ explain: true, newTab: false }),
          });
        }
      } else {
        explainQueryAction?.dispose();
        explainQueryAction = undefined;
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
  if (activeSQLEditorRef.value) {
    formatEditorContent(activeSQLEditorRef.value, dialect.value);
  }
});

useEmitteryEventListener(
  editorEvents,
  "set-editor-selection",
  (selection: IRange) => {
    activeSQLEditorRef.value?.setSelection(selection);
    activeSQLEditorRef.value?.revealLineNearTop(selection.startLineNumber);
    activeSQLEditorRef.value?.focus();
  }
);

useEmitteryEventListener(
  editorEvents,
  "append-editor-content",
  ({ content, select }: { content: string; select: boolean }) => {
    const oldStatement = currentTab.value?.statement ?? "";
    const newStatement = [oldStatement, content].filter((s) => s).join("\n\n");
    activeSQLEditorRef.value?.setValue(newStatement);

    if (select) {
      const selection = new Selection(
        oldStatement.split("\n").length + 1,
        0,
        newStatement.split("\n").length + 1,
        0
      );
      nextTick(() => {
        editorEvents.emit("set-editor-selection", selection);
      });
    }
  }
);

onBeforeUnmount(() => {
  activeSQLEditorRef.value = undefined;
});

defineExpose({
  getActiveStatement,
});
</script>
