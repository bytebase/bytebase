<template>
  <div class="whitespace-pre-wrap w-full overflow-hidden">
    <MonacoEditor
      ref="editorRef"
      class="w-full h-auto max-h-[360px]"
      :value="sql"
      :language="selectedLanguage"
      :dialect="selectedDialect"
      :readonly="readonly"
      :options="EDITOR_OPTIONS"
      @change="handleChange"
      @change-selection="handleChangeSelection"
      @save="handleSaveSheet"
      @ready="handleEditorReady"
    />
  </div>
</template>

<script lang="ts" setup>
import { editor as Editor } from "monaco-editor";
import { computed, nextTick, ref, watch, watchEffect } from "vue";
import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";
import {
  useTabStore,
  useSQLEditorStore,
  useDBSchemaV1Store,
  useDatabaseV1Store,
  useInstanceV1ByUID,
} from "@/store";
import {
  ComposedDatabase,
  dialectOfEngineV1,
  ExecuteConfig,
  ExecuteOption,
  SQLDialect,
  UNKNOWN_ID,
} from "@/types";
import { TableMetadata } from "@/types/proto/store/database";
import { formatEngineV1, useInstanceV1EditorLanguage } from "@/utils";
import {
  checkCursorAtFirstLine,
  checkCursorAtLast,
  checkCursorAtLastLine,
  checkEndsWithSemicolon,
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
  (e: "save-sheet", content?: string): void;
  (e: "update:sql", sql: string): void;
  (
    e: "execute",
    sql: string,
    config: ExecuteConfig,
    option?: ExecuteOption
  ): void;
  (e: "history", direction: "up" | "down"): void;
  (e: "clear-screen"): void;
}>();

const MIN_EDITOR_HEIGHT = 40; // ~= 1 line

const tabStore = useTabStore();
const databaseStore = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const sqlEditorStore = useSQLEditorStore();

const editorRef = ref<InstanceType<typeof MonacoEditor>>();

const { instance: selectedInstance } = useInstanceV1ByUID(
  computed(() => tabStore.currentTab.connection.instanceId)
);
const selectedDatabase = computed(() => {
  const id = tabStore.currentTab.connection.databaseId;
  if (id === String(UNKNOWN_ID)) return undefined;
  return databaseStore.getDatabaseByUID(id);
});
const selectedInstanceEngine = computed(() => {
  return formatEngineV1(selectedInstance.value);
});
const selectedLanguage = useInstanceV1EditorLanguage(selectedInstance);
const selectedDialect = computed((): SQLDialect => {
  const engine = selectedInstance.value.engine;
  return dialectOfEngineV1(engine);
});
const currentTabId = computed(() => tabStore.currentTabId);
const isSwitchingTab = ref(false);

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

const firstLinePrompt = computed(() => {
  const lang = selectedLanguage.value;
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
  tabStore.updateCurrentTab({
    statement: value,
    isSaved: false,
  });

  emit("update:sql", value);
  updateEditorHeight();
};

const handleChangeSelection = (value: string) => {
  tabStore.updateCurrentTab({
    selectedStatement: value,
  });
};

const handleSaveSheet = () => {
  emit("save-sheet");
};

const handleEditorReady = async () => {
  const monaco = await import("monaco-editor");
  const editor = editorRef.value?.editorInstance;
  const readonly = editor?.createContextKey<boolean>(
    "readonly",
    props.readonly
  );
  watch(
    () => props.readonly,
    () => readonly?.set(props.readonly)
  );

  editor?.addAction({
    id: "RunQuery",
    label: "Run Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
    contextMenuGroupId: "operation",
    contextMenuOrder: 0,
    precondition: "!readonly",
    run: async () => {
      emit("execute", props.sql, {
        databaseType: selectedInstanceEngine.value,
      });
    },
  });

  editor?.addAction({
    id: "ExplainQuery",
    label: "Explain Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyE],
    contextMenuGroupId: "operation",
    contextMenuOrder: 1,
    precondition: "!readonly",
    run: async () => {
      emit(
        "execute",
        props.sql,
        {
          databaseType: selectedInstanceEngine.value,
        },
        { explain: true }
      );
    },
  });

  editor?.addAction({
    id: "ClearScreen",
    label: "Clear Screen",
    keybindings: [
      monaco.KeyMod.Alt | monaco.KeyMod.Shift | monaco.KeyCode.KeyC,
    ],
    contextMenuGroupId: "operation",
    contextMenuOrder: 3,
    precondition: "!readonly",
    run: () => {
      emit("clear-screen");
    },
  });

  // Create an editor context value to check if the SQL ends with semicolon ";"
  const endsWithSemicolon = editor?.createContextKey<boolean>(
    "endsWithSemicolon",
    checkEndsWithSemicolon(editor)
  );
  editor?.onDidChangeModelContent(() => {
    endsWithSemicolon?.set(checkEndsWithSemicolon(editor));
  });
  // Another editor context value to check if the cursor is at the end of the
  // editor.
  const cursorAtLast = editor?.createContextKey<boolean>(
    "cursorAtLast",
    checkCursorAtLast(editor)
  );
  editor?.onDidChangeCursorPosition(() => {
    cursorAtLast?.set(checkCursorAtLast(editor));
  });
  editor?.addCommand(
    monaco.KeyCode.Enter,
    () => {
      // When
      // - the SQL ends with ";"
      // - and the cursor is at the end of the editor
      // - then press "Enter"
      // We trigger the "execute" event
      emit("execute", props.sql, {
        databaseType: selectedInstanceEngine.value,
      });
    },
    // Tell the editor this should be only
    // triggered when both of the two conditions are satisfied.
    "!readonly && endsWithSemicolon && cursorAtLast && editorTextFocus && !suggestWidgetVisible && !renameInputVisible && !inSnippetMode && !quickFixWidgetVisible"
  );

  const cursorAtFirstLine = editor?.createContextKey<boolean>(
    "cursorAtFirstLine",
    checkCursorAtFirstLine(editor)
  );
  const cursorAtLastLine = editor?.createContextKey<boolean>(
    "cursorAtLastLine",
    checkCursorAtLastLine(editor)
  );
  editor?.onDidChangeCursorPosition(() => {
    cursorAtFirstLine?.set(checkCursorAtFirstLine(editor));
  });
  editor?.onDidChangeCursorPosition(() => {
    cursorAtLastLine?.set(checkCursorAtLastLine(editor));
  });
  editor?.addCommand(
    monaco.KeyCode.UpArrow,
    () => {
      // When
      // - the cursor is at the first line
      // - then press "CtrlCmd + Up"
      // We trigger the "history" event
      emit("history", "up");
    },
    // Tell the editor this should be only
    // triggered when both of the two conditions are satisfied.
    "!readonly && cursorAtFirstLine && editorTextFocus && !suggestWidgetVisible && !renameInputVisible && !inSnippetMode && !quickFixWidgetVisible"
  );
  editor?.addCommand(
    monaco.KeyCode.DownArrow,
    () => {
      // When
      // - the cursor is at the last line
      // - then press "CtrlCmd + Down"
      // We trigger the "history" event
      emit("history", "down");
    },
    // Tell the editor this should be only
    // triggered when both of the two conditions are satisfied.
    "!readonly && cursorAtLastLine && editorTextFocus && !suggestWidgetVisible && !renameInputVisible && !inSnippetMode && !quickFixWidgetVisible"
  );

  watchEffect(async () => {
    if (selectedInstance.value) {
      const databaseMap: Map<ComposedDatabase, TableMetadata[]> = new Map();
      const databaseList = selectedDatabase.value
        ? [selectedDatabase.value]
        : databaseStore.databaseListByInstance(selectedInstance.value.name);
      // Only provide auto-complete context for those opened database.
      for (const database of databaseList) {
        const tableList = dbSchemaStore.getTableList(database.name);
        if (tableList.length > 0) {
          databaseMap.set(database, tableList);
        }
      }
      const connectionScope = selectedDatabase.value ? "database" : "instance";
      editorRef.value?.setEditorAutoCompletionContextV1(
        databaseMap,
        connectionScope
      );
    }
  });

  updateEditorHeight();
};

const updateEditorHeight = () => {
  const contentHeight =
    editorRef.value?.editorInstance?.getContentHeight() as number;
  let actualHeight = contentHeight;
  if (actualHeight < MIN_EDITOR_HEIGHT) {
    actualHeight = MIN_EDITOR_HEIGHT;
  }
  editorRef.value?.setEditorContentHeight(actualHeight);
};

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
watch(
  firstLinePrompt,
  (prompt) => (EDITOR_OPTIONS.value.lineNumbersMinChars = prompt.length + 1)
);
</script>
