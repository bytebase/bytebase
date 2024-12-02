<template>
  <div class="whitespace-pre-wrap w-full overflow-hidden bb-compact-sql-editor">
    <MonacoEditor
      class="w-full h-auto"
      :style="{
        'min-height': `${MIN_EDITOR_HEIGHT}px`,
        'max-height': `${MAX_EDITOR_HEIGHT}px`,
      }"
      :content="sql"
      :language="language"
      :dialect="dialect"
      :readonly="readonly"
      :options="EDITOR_OPTIONS"
      :auto-height="{
        min: MIN_EDITOR_HEIGHT,
        max: MAX_EDITOR_HEIGHT,
      }"
      :auto-complete-context="{
        instance: instance.name,
        database: database.name,
        schema: currentTab?.connection.schema,
        scene: 'query',
      }"
      @update:content="handleChange"
      @ready="handleEditorReady"
    />
  </div>
</template>

<script lang="ts" setup>
import type { editor as Editor } from "monaco-editor";
import { storeToRefs } from "pinia";
import { computed, nextTick, ref, toRef, watch } from "vue";
import type {
  IStandaloneCodeEditor,
  MonacoModule,
} from "@/components/MonacoEditor";
import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";
import {
  useEditorContextKey,
  formatEditorContent,
} from "@/components/MonacoEditor/utils";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  useSQLEditorTabStore,
  useConnectionOfCurrentSQLEditorTab,
} from "@/store";
import type { SQLDialect, SQLEditorQueryParams } from "@/types";
import { dialectOfEngineV1 } from "@/types";
import { useInstanceV1EditorLanguage } from "@/utils";
import { useSQLEditorContext } from "../../context";
import {
  checkCursorAtFirstLine,
  checkCursorAtLast,
  checkCursorAtLastLine,
  checkIsEnterEndsStatement,
} from "./utils";

const props = defineProps({
  sql: {
    type: String,
    default: "",
  },
  readonly: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits<{
  (e: "update:sql", sql: string): void;
  (e: "execute", params: SQLEditorQueryParams): void;
  (e: "history", direction: "up" | "down", editor: IStandaloneCodeEditor): void;
  (e: "clear-screen"): void;
}>();

const MIN_EDITOR_HEIGHT = 40; // ~= 1 line
const MAX_EDITOR_HEIGHT = 360; // ~= 2 lines

const tabStore = useSQLEditorTabStore();

const { events: editorEvents } = useSQLEditorContext();
const { connection, instance, database } = useConnectionOfCurrentSQLEditorTab();
const language = useInstanceV1EditorLanguage(instance);
const { currentTab, isSwitchingTab } = storeToRefs(tabStore);
const pendingFormatContentCommand = ref(false);
const dialect = computed((): SQLDialect => {
  const engine = instance.value.engine;
  return dialectOfEngineV1(engine);
});

const firstLinePrompt = computed(() => {
  const lang = language.value;
  if (lang === "javascript") return "MONGO>";
  if (lang === "redis") return "REDIS>";
  return "SQL>";
});

const getLineNumber = (lineNumber: number) => {
  /*
    Show a SQL CLI like command prompt.
    SQL> first_line
      -> second_line
      -> more_lines
  */
  if (lineNumber === 1) {
    return firstLinePrompt.value;
  }
  return "->";
};

const handleChange = (value: string) => {
  // When we are switching between tabs, the MonacoEditor emits a 'change'
  // event, but we shouldn't update the current tab;
  if (isSwitchingTab.value) {
    return;
  }

  emit("update:sql", value);
};

const execute = (explain = false) => {
  emit("execute", {
    connection: { ...connection.value },
    statement: props.sql,
    engine: instance.value.engine,
    explain,
    selection: null,
  });
};

const handleEditorReady = (
  monaco: MonacoModule,
  editor: IStandaloneCodeEditor
) => {
  useEditorContextKey(editor, "readonly", toRef(props, "readonly"));

  editor.addAction({
    id: "RunQuery",
    label: "Run Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
    contextMenuGroupId: "operation",
    contextMenuOrder: 0,
    precondition: "!readonly",
    run: () => execute(false),
  });

  editor.addAction({
    id: "ExplainQuery",
    label: "Explain Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyE],
    contextMenuGroupId: "operation",
    contextMenuOrder: 1,
    precondition: "!readonly",
    run: () => execute(true),
  });

  editor.addAction({
    id: "ClearScreen",
    label: "Clear Screen",
    keybindings: [
      monaco.KeyMod.Alt | monaco.KeyMod.Shift | monaco.KeyCode.KeyC,
    ],
    contextMenuGroupId: "operation",
    contextMenuOrder: 3,
    precondition: "!readonly",
    run: () => emit("clear-screen"),
  });

  // Create an editor context value to check if the SQL ends with semicolon ";"
  const isEnterEndsStatement = useEditorContextKey(
    editor,
    "isEnterEndsStatement",
    checkIsEnterEndsStatement(editor, language.value)
  );
  editor.onDidChangeModelContent(() => {
    isEnterEndsStatement.set(checkIsEnterEndsStatement(editor, language.value));
  });
  // Another editor context value to check if the cursor is at the end of the
  // editor.
  const cursorAtLast = useEditorContextKey(
    editor,
    "cursorAtLast",
    checkCursorAtLast(editor)
  );
  editor.onDidChangeCursorPosition(() => {
    cursorAtLast.set(checkCursorAtLast(editor));
  });
  editor.addCommand(
    monaco.KeyCode.Enter,
    () => {
      // When
      // - the SQL ends with ";"
      // - and the cursor is at the end of the editor
      // - then press "Enter"
      // We trigger the "execute" event
      execute(false);
    },
    // Tell the editor this should be only
    // triggered when both of the two conditions are satisfied.
    "!readonly && isEnterEndsStatement && cursorAtLast && editorTextFocus && !suggestWidgetVisible && !renameInputVisible && !inSnippetMode && !quickFixWidgetVisible"
  );

  const cursorAtFirstLine = useEditorContextKey(
    editor,
    "cursorAtFirstLine",
    checkCursorAtFirstLine(editor)
  );
  const cursorAtLastLine = useEditorContextKey(
    editor,
    "cursorAtLastLine",
    checkCursorAtLastLine(editor)
  );
  editor.onDidChangeCursorPosition(() => {
    cursorAtFirstLine?.set(checkCursorAtFirstLine(editor));
  });
  editor.onDidChangeCursorPosition(() => {
    cursorAtLastLine?.set(checkCursorAtLastLine(editor));
  });
  editor.addCommand(
    monaco.KeyCode.UpArrow,
    () => {
      // When
      // - the cursor is at the first line
      // - then press "CtrlCmd + Up"
      // We trigger the "history" event
      emit("history", "up", editor);
    },
    // Tell the editor this should be only
    // triggered when both of the two conditions are satisfied.
    "!readonly && cursorAtFirstLine && editorTextFocus && !suggestWidgetVisible && !renameInputVisible && !inSnippetMode && !quickFixWidgetVisible"
  );
  editor.addCommand(
    monaco.KeyCode.DownArrow,
    () => {
      // When
      // - the cursor is at the last line
      // - then press "CtrlCmd + Down"
      // We trigger the "history" event
      emit("history", "down", editor);
    },
    // Tell the editor this should be only
    // triggered when both of the two conditions are satisfied.
    "!readonly && cursorAtLastLine && editorTextFocus && !suggestWidgetVisible && !renameInputVisible && !inSnippetMode && !quickFixWidgetVisible"
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
};

useEmitteryEventListener(editorEvents, "format-content", () => {
  pendingFormatContentCommand.value = true;
});

const EDITOR_OPTIONS = computed<Editor.IStandaloneEditorConstructionOptions>(
  () => ({
    theme: "bb-dark",
    minimap: {
      enabled: false,
    },
    scrollbar: {
      vertical: "hidden",
      horizontal: "hidden",
      alwaysConsumeMouseWheel: false,
    },
    overviewRulerLanes: 0,
    lineNumbers: getLineNumber,
    lineNumbersMinChars: firstLinePrompt.value.length + 1,
    glyphMargin: false,
    cursorStyle: "block",
  })
);
</script>

<style lang="postcss" scoped>
.bb-compact-sql-editor :deep(.monaco-editor .line-numbers) {
  @apply !pr-0;
}
</style>
