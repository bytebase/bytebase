<template>
  <div class="whitespace-pre-wrap w-full overflow-hidden">
    <MonacoEditor
      ref="editorRef"
      class="w-full h-auto max-h-[360px]"
      :value="sql"
      :dialect="selectedDialect"
      :readonly="readonly"
      :options="{
        theme: 'bb-dark',
        minimap: {
          enabled: false,
        },
        scrollbar: {
          vertical: 'hidden',
          horizontal: 'hidden',
          alwaysConsumeMouseWheel: false,
        },
        overviewRulerLanes: 0,
        lineNumbers: getLineNumber,
        lineNumbersMinChars: 5,
        glyphMargin: false,
        cursorStyle: 'block',
      }"
      @change="handleChange"
      @change-selection="handleChangeSelection"
      @save="handleSaveSheet"
      @ready="handleEditorReady"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, nextTick, ref, watch, watchEffect } from "vue";

import {
  useInstanceStore,
  useTabStore,
  useSQLEditorStore,
  useDatabaseStore,
  useDBSchemaStore,
  useInstanceById,
} from "@/store";
import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";
import { Database, ExecuteConfig, ExecuteOption, SQLDialect } from "@/types";
import { TableMetadata } from "@/types/proto/database";

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
}>();

const MIN_EDITOR_HEIGHT = 40; // ~= 1 line

const instanceStore = useInstanceStore();
const tabStore = useTabStore();
const databaseStore = useDatabaseStore();
const dbSchemaStore = useDBSchemaStore();
const sqlEditorStore = useSQLEditorStore();

const editorRef = ref<InstanceType<typeof MonacoEditor>>();

const selectedInstance = useInstanceById(
  computed(() => tabStore.currentTab.connection.instanceId)
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

const getLineNumber = (lineNumber: number) => {
  /*
    Show a SQL CLI like command prompt.
    SQL> first_line
      -> second_line
      -> more_lines
  */
  if (lineNumber === 1) return "SQL>";
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
  editor?.addAction({
    id: "RunQuery",
    label: "Run Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
    contextMenuGroupId: "operation",
    contextMenuOrder: 0,
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
    contextMenuOrder: 0,
    run: async () => {
      emit(
        "execute",
        props.sql,
        { databaseType: selectedInstanceEngine.value },
        { explain: true }
      );
    },
  });

  // Create an editor context value to check if the SQL ends with semicolon ";"
  const endsWithSemicolon = editor?.createContextKey<boolean>(
    "endsWithSemicolon",
    false
  );
  editor?.onDidChangeModelContent(() => {
    const value = editor.getValue();
    if (value.endsWith(";")) {
      endsWithSemicolon?.set(true);
    } else {
      endsWithSemicolon?.set(false);
    }
  });
  // Another editor context value to check if the cursor is at the end of the
  // editor.
  const cursorAtLast = editor?.createContextKey<boolean>("cursorAtLast", false);
  editor?.onDidChangeCursorPosition(() => {
    const model = editor.getModel();
    if (model) {
      const maxLine = model.getLineCount();
      const maxColumn = model.getLineMaxColumn(maxLine);
      const cursor = editor.getPosition();
      const isCursorAtLast = !!cursor?.equals({
        lineNumber: maxLine,
        column: maxColumn,
      });
      if (isCursorAtLast) {
        cursorAtLast?.set(true);
        return;
      }
    }
    cursorAtLast?.set(false);
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
    "endsWithSemicolon && cursorAtLast"
  );

  watchEffect(async () => {
    if (selectedInstance.value) {
      const databaseMap: Map<Database, TableMetadata[]> = new Map();
      const databaseList = databaseStore.getDatabaseListByInstanceId(
        selectedInstance.value.id
      );
      for (const database of databaseList) {
        const tableList = await dbSchemaStore.getOrFetchTableListByDatabaseId(
          database.id
        );
        databaseMap.set(database, tableList);
      }
      editorRef.value?.setEditorAutoCompletionContext(databaseMap);
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
</script>
