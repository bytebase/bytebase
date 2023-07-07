<template>
  <div ref="editorContainerRef" v-bind="$attrs" class="relative">
    <div
      v-if="!isEditorLoaded"
      class="absolute inset-0 flex flex-col items-center justify-center"
    >
      <BBSpin />
    </div>
  </div>
</template>

<script lang="ts" setup>
import {
  onMounted,
  ref,
  toRef,
  nextTick,
  watch,
  shallowRef,
  PropType,
  onBeforeUnmount,
  watchEffect,
} from "vue";
import type { editor as Editor } from "monaco-editor";
import { ComposedDatabase, Database, Language, SQLDialect } from "@/types";
import { TableMetadata } from "@/types/proto/store/database";
import { MonacoHelper, useMonaco } from "./useMonaco";
import { useLineDecorations } from "./lineDecorations";
import type { useLanguageClient } from "@sql-lsp/client";
import type { AdviceOption } from "./types";
import { useAdvices } from "./plugins/useAdvices";

const props = defineProps({
  value: {
    type: String,
    default: "",
  },
  language: {
    type: String as PropType<Language>,
    default: "sql",
  },
  dialect: {
    type: String as PropType<SQLDialect>,
    default: "MYSQL",
  },
  readonly: {
    type: Boolean,
    default: false,
  },
  autoFocus: {
    type: Boolean,
    default: true,
  },
  advices: {
    type: Array as PropType<AdviceOption[]>,
    default: () => [],
  },
  options: {
    type: Object as PropType<Editor.IStandaloneEditorConstructionOptions>,
    default: undefined,
  },
});

const emit = defineEmits<{
  (e: "change", content: string): void;
  (e: "change-selection", content: string): void;
  (e: "save", content: string): void;
  (e: "ready"): void;
}>();

const sqlCode = toRef(props, "value");
const dialect = toRef(props, "dialect");
const readOnly = toRef(props, "readonly");
const monacoInstanceRef = ref<MonacoHelper>();
const editorContainerRef = ref<HTMLDivElement>();
// use shallowRef to avoid deep conversion which will cause page crash.
const editorInstanceRef = shallowRef<Editor.IStandaloneCodeEditor>();
const languageClientRef = ref<ReturnType<typeof useLanguageClient>>();

const isEditorLoaded = ref(false);

const initEditorInstance = () => {
  const { monaco, formatContent, setPositionAtEndOfLine } =
    monacoInstanceRef.value!;

  const model = monaco.editor.createModel(sqlCode.value);
  const editorInstance = monaco.editor.create(editorContainerRef.value!, {
    model,
    // Learn more: https://github.com/microsoft/monaco-editor/issues/311
    renderValidationDecorations: "on",
    theme: "bb",
    tabSize: 2,
    insertSpaces: true,
    autoClosingQuotes: "always",
    detectIndentation: false,
    folding: false,
    automaticLayout: true,
    readOnly: readOnly.value,
    minimap: {
      enabled: false,
    },
    wordWrap: "on",
    fixedOverflowWidgets: true,
    fontSize: 15,
    lineHeight: 24,
    scrollBeyondLastLine: false,
    padding: {
      top: 8,
      bottom: 8,
    },
    renderLineHighlight: "none",
    codeLens: false,
    scrollbar: {
      alwaysConsumeMouseWheel: false,
    },
    ...props.options,
  });

  const defaultSuggestOption = {
    ...editorInstance.getOption(monaco.editor.EditorOption.suggest),
  };

  watchEffect((onCleanup) => {
    const { language } = props;

    monaco.editor.setModelLanguage(model, language);

    if (props.language === "sql") {
      editorInstance.updateOptions({
        suggest: defaultSuggestOption,
      });

      // add `Format SQL` action into context menu
      const action = editorInstance.addAction({
        id: "format-sql",
        label: "Format SQL",
        keybindings: [
          monaco.KeyMod.Alt | monaco.KeyMod.Shift | monaco.KeyCode.KeyF,
        ],
        contextMenuGroupId: "operation",
        contextMenuOrder: 1,
        run: () => {
          if (readOnly.value) {
            return;
          }
          formatContent(editorInstance, dialect.value);
          nextTick(() => setPositionAtEndOfLine(editorInstance));
        },
      });
      onCleanup(() => {
        action.dispose();
      });
    } else {
      // Disable auto-complete suggestions for javascript language (MongoDB)
      editorInstance.updateOptions({
        suggest: {
          showConstants: false,
          showFunctions: false,
          showInterfaces: false,
          showClasses: false,
          showConstructors: false,
          showColors: false,
          showDeprecated: false,
          showEnumMembers: false,
          showEnums: false,
          showEvents: false,
          showFields: false,
          showFiles: false,
          showFolders: false,
          showIcons: false,
          showInlineDetails: false,
          showIssues: false,
          showKeywords: false,
          showMethods: false,
          showModules: false,
          showOperators: false,
          showProperties: false,
          showReferences: false,
          showSnippets: false,
          showStatusBar: false,
          showStructs: false,
          showTypeParameters: false,
          showUnits: false,
          showUsers: false,
          showValues: false,
          showVariables: false,
          showWords: false,
        },
      });

      // When the language is "javascript", we can still use Alt+Shift+F to
      // format the document (the native feature of monaco-editor).
    }
  });

  // typed something, change the text
  editorInstance.onDidChangeModelContent(() => {
    const value = editorInstance.getValue();
    emit("change", value);
  });

  // when editor change selection, emit change-selection event with selected text
  editorInstance.onDidChangeCursorSelection((e: any) => {
    const selectedText = editorInstance
      .getModel()
      ?.getValueInRange(e.selection) as string;
    emit("change-selection", selectedText);
  });

  editorInstance.onDidChangeCursorPosition(async (e: any) => {
    const { defineLineDecorations, disposeLineDecorations } =
      await useLineDecorations(editorInstance, e.position);
    // clear the old decorations
    disposeLineDecorations();

    // define the new decorations
    nextTick(async () => {
      await defineLineDecorations();
    });
  });

  editorInstance.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
    const value = editorInstance.getValue();
    emit("save", value);
  });

  return editorInstance;
};

onMounted(async () => {
  // Load monaco-editor and sql-lsp/client asynchronously.
  const [monacoHelper, { useLanguageClient }] = await Promise.all([
    useMonaco(),
    import("@sql-lsp/client"),
  ]);

  if (!editorContainerRef.value) {
    // Give up creating monaco editor if the component has been unmounted
    // very quickly.
    console.debug(
      "<MonacoEditor> has been unmounted before useMonaco is ready"
    );
    return;
  }

  const { setPositionAtEndOfLine } = monacoHelper;
  monacoInstanceRef.value = monacoHelper;

  const editorInstance = initEditorInstance();
  editorInstanceRef.value = editorInstance;

  const languageClient = useLanguageClient();
  languageClientRef.value = languageClient;
  languageClient.start();

  // set the editor focus when the tab is selected
  if (!readOnly.value && props.autoFocus) {
    editorInstance.focus();
    nextTick(() => setPositionAtEndOfLine(editorInstance));
  }

  useAdvices(editorInstance, toRef(props, "advices"));

  isEditorLoaded.value = true;

  nextTick(() => {
    emit("ready");

    watch(dialect, () => languageClient.changeDialect(dialect.value), {
      immediate: true,
      // Delay the flush timing to ensure it performs after the language client started.
      flush: "post",
    });

    watch(
      () => props.options,
      (opts) => {
        if (opts) {
          editorInstance.updateOptions(opts);
        }
      },
      { deep: true }
    );
  });
});

onBeforeUnmount(() => {
  editorInstanceRef.value?.dispose();
  monacoInstanceRef.value?.dispose();
  languageClientRef.value?.stop();
});

watch(
  () => readOnly.value,
  (readOnly) => {
    editorInstanceRef.value?.updateOptions({
      readOnly: readOnly,
    });
  },
  {
    deep: true,
    immediate: true,
  }
);

const getEditorContent = () => {
  return editorInstanceRef.value?.getValue();
};

const setEditorContent = (content: string) => {
  if (readOnly.value) {
    // workaround: setContent doesn't work in readonly mode
    // we temporarily set it to false
    editorInstanceRef.value?.updateOptions({
      readOnly: false,
    });
  }

  monacoInstanceRef.value?.setContent(editorInstanceRef.value!, content);

  if (readOnly.value) {
    // then set it back
    editorInstanceRef.value?.updateOptions({
      readOnly: true,
    });
  }
};

watch(
  () => props.value,
  (value) => {
    if (value !== getEditorContent()) {
      setEditorContent(value);
    }
  }
);

const getEditorContentHeight = () => {
  return editorInstanceRef.value?.getContentHeight();
};

const setEditorContentHeight = (height: number) => {
  editorContainerRef.value!.style.height = `${
    height ?? getEditorContentHeight()
  }px`;
};

const formatEditorContent = () => {
  if (readOnly.value) {
    return;
  }
  if (props.language !== "sql") {
    return;
  }
  monacoInstanceRef.value?.formatContent(
    editorInstanceRef.value!,
    dialect.value
  );
  nextTick(() => {
    monacoInstanceRef.value?.setPositionAtEndOfLine(editorInstanceRef.value!);
    editorInstanceRef.value?.focus();
  });
};

const setEditorAutoCompletionContext = (
  databaseMap: Map<Database, TableMetadata[]>,
  connectionScope: "instance" | "database" = "database"
) => {
  const databases = [];
  for (const [database, tableList] of databaseMap) {
    databases.push({
      name: database.name,
      tables: tableList.map((table) => ({
        database: database.name,
        name: table.name,
        columns: table.columns.map((column) => ({
          name: column.name,
        })),
      })),
    });
  }
  languageClientRef.value?.changeSchema({
    databases: databases,
  });
  languageClientRef.value?.changeConnectionScope(connectionScope);
};

const setEditorAutoCompletionContextV1 = (
  databaseMap: Map<ComposedDatabase, TableMetadata[]>,
  connectionScope: "instance" | "database" = "database"
) => {
  const databases = [];
  for (const [database, tableList] of databaseMap) {
    databases.push({
      name: database.databaseName,
      tables: tableList.map((table) => ({
        database: database.databaseName,
        name: table.name,
        columns: table.columns.map((column) => ({
          name: column.name,
        })),
      })),
    });
  }
  languageClientRef.value?.changeSchema({
    databases: databases,
  });
  languageClientRef.value?.changeConnectionScope(connectionScope);
};

defineExpose({
  editorInstance: editorInstanceRef,
  formatEditorContent,
  getEditorContent,
  setEditorContent,
  getEditorContentHeight,
  setEditorContentHeight,
  setEditorAutoCompletionContext,
  setEditorAutoCompletionContextV1,
});
</script>

<style>
.monaco-editor .monaco-mouse-cursor-text {
  box-shadow: none !important;
}
.monaco-editor .scroll-decoration {
  display: none !important;
}
</style>
