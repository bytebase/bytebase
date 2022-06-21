<template>
  <div ref="editorContainerRef" style="width: 100%; height: 100%"></div>
</template>

<script lang="ts" setup>
import {
  onMounted,
  ref,
  toRef,
  toRaw,
  nextTick,
  onUnmounted,
  watch,
  shallowRef,
} from "vue";
import { editor as Editor } from "monaco-editor";
import { Database, SQLDialect, Table } from "@/types";
import { useMonaco } from "./useMonaco";
import { useLineDecorations } from "./lineDecorations";

const props = defineProps({
  value: {
    type: String,
    required: true,
  },
  language: {
    type: String,
    default: "mysql",
  },
  readonly: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits<{
  (e: "update:value", content: string): void;
  (e: "change", content: string): void;
  (e: "change-selection", content: string): void;
  (
    e: "run-query",
    content: {
      explain: boolean;
      query: string;
    }
  ): void;
  (e: "save", content: string): void;
  (e: "ready"): void;
}>();

const sqlCode = toRef(props, "value");
const language = toRef(props, "language");
const readOnly = toRef(props, "readonly");
const monacoInstanceRef = ref();
const editorContainerRef = ref();
// use shallowRef to avoid deep conversion which will cause page crash.
const editorInstanceRef = shallowRef<Editor.IStandaloneCodeEditor>();

const getEditorInstance = () => {
  const { monaco, formatContent, setPositionAtEndOfLine } =
    monacoInstanceRef.value;

  const model = monaco.editor.createModel(sqlCode.value, toRaw(language.value));
  const editorInstance = monaco.editor.create(editorContainerRef.value, {
    model,
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
  });

  // add the run query action in context menu
  editorInstance.addAction({
    id: "RunQuery",
    label: "Run Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
    contextMenuGroupId: "operation",
    contextMenuOrder: 0,
    run: async () => {
      const typedValue = editorInstance.getValue();
      const selectedValue = editorInstance
        .getModel()
        ?.getValueInRange(editorInstance.getSelection()!) as string;
      const query = selectedValue || typedValue;
      emit("run-query", { explain: false, query });
    },
  });

  // add the run query action in context menu
  editorInstance.addAction({
    id: "ExplainQuery",
    label: "Explain Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyE],
    contextMenuGroupId: "operation",
    contextMenuOrder: 0,
    run: async () => {
      const typedValue = editorInstance.getValue();
      const selectedValue = editorInstance
        .getModel()
        ?.getValueInRange(editorInstance.getSelection()!) as string;

      const query = selectedValue || typedValue;
      emit("run-query", { explain: true, query });
    },
  });

  // add format sql action in context menu
  editorInstance.addAction({
    id: "FormatSQL",
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
      formatContent(editorInstance, language.value as SQLDialect);
      nextTick(() => setPositionAtEndOfLine(editorInstance));
    },
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

  editorInstance.onDidChangeCursorPosition((e: any) => {
    const { defineLineDecorations, disposeLineDecorations } =
      useLineDecorations(editorInstance, e.position);
    // clear the old decorations
    disposeLineDecorations();

    // define the new decorations
    nextTick(() => {
      defineLineDecorations();
    });
  });

  editorInstance.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
    const value = editorInstance.getValue();
    emit("save", value);
  });

  return editorInstance;
};

onMounted(async () => {
  const {
    monaco,
    formatContent,
    setContent,
    setAutoCompletionContext,
    setPositionAtEndOfLine,
  } = await useMonaco(language.value);

  monacoInstanceRef.value = {
    monaco,
    formatContent,
    setContent,
    setAutoCompletionContext,
    setPositionAtEndOfLine,
  };

  const editorInstance = getEditorInstance();
  editorInstanceRef.value = editorInstance;

  // set the editor focus when the tab is selected
  if (!readOnly.value) {
    editorInstance.focus();
    nextTick(() => setPositionAtEndOfLine(editorInstance));
  }

  nextTick(() => {
    emit("ready");
  });
});

onUnmounted(() => {
  editorInstanceRef.value?.dispose();
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
  monacoInstanceRef.value?.setContent(editorInstanceRef.value!, content);
};

const getEditorContentHeight = () => {
  return editorInstanceRef.value?.getContentHeight();
};

const formatEditorContent = () => {
  monacoInstanceRef.value?.formatContent(
    editorInstanceRef.value!,
    language.value as SQLDialect
  );
  nextTick(() => {
    monacoInstanceRef.value?.setPositionAtEndOfLine(editorInstanceRef.value!);
    editorInstanceRef.value?.focus();
  });
};

const setEditorAutoCompletionContext = (
  databases: Database[],
  tables: Table[]
) => {
  monacoInstanceRef.value?.setAutoCompletionContext(databases, tables);
};

defineExpose({
  formatEditorContent,
  getEditorContent,
  setEditorContent,
  getEditorContentHeight,
  setEditorAutoCompletionContext,
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
