<template>
  <div ref="editorContainerRef" v-bind="$attrs"></div>
  <BBSpin
    v-if="!isEditorLoaded"
    class="h-full w-full flex items-center justify-center"
  />
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
import type { editor as Editor } from "monaco-editor";
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
  (e: "change", content: string): void;
  (e: "change-selection", content: string): void;
  (e: "save", content: string): void;
  (e: "ready"): void;
}>();

const sqlCode = toRef(props, "value");
const language = toRef(props, "language");
const readOnly = toRef(props, "readonly");
const monacoInstanceRef = ref();
const editorContainerRef = ref<HTMLDivElement>();
// use shallowRef to avoid deep conversion which will cause page crash.
const editorInstanceRef = shallowRef<Editor.IStandaloneCodeEditor>();

const isEditorLoaded = ref(false);

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

  // add `Format SQL` action into context menu
  editorInstance.addAction({
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
  const {
    monaco,
    dispose,
    formatContent,
    setContent,
    setAutoCompletionContext,
    setPositionAtEndOfLine,
  } = await useMonaco(language.value);

  monacoInstanceRef.value = {
    monaco,
    dispose,
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

  isEditorLoaded.value = true;

  nextTick(() => {
    emit("ready");
  });
});

onUnmounted(() => {
  editorInstanceRef.value?.dispose();
  monacoInstanceRef.value?.dispose();
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

const setEditorContentHeight = (height: number) => {
  editorContainerRef.value!.style.height = `${
    height ?? getEditorContentHeight()
  }px`;
};

const formatEditorContent = () => {
  if (readOnly.value) {
    return;
  }
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
  editorInstance: editorInstanceRef,
  formatEditorContent,
  getEditorContent,
  setEditorContent,
  getEditorContentHeight,
  setEditorContentHeight,
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
