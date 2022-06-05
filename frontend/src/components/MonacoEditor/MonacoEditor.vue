<template>
  <div ref="editorRef" style="height: 100%; width: 100%"></div>
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
  defineExpose,
} from "vue";
import { editor as Editor } from "monaco-editor";
import { SQLDialect } from "@/types";
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
}>();

const editorRef = ref();
const sqlCode = toRef(props, "value");
const language = toRef(props, "language");
const readonly = toRef(props, "readonly");

let editorInstance: Editor.IStandaloneCodeEditor;

const {
  monaco,
  formatContent,
  setContent,
  setAutoCompletionContext,
  setPositionAtEndOfLine,
} = await useMonaco(language.value);

const init = async () => {
  const model = monaco.editor.createModel(sqlCode.value, toRaw(language.value));

  editorInstance = monaco.editor.create(editorRef.value, {
    model,
    tabSize: 2,
    insertSpaces: true,
    autoClosingQuotes: "always",
    detectIndentation: false,
    folding: false,
    automaticLayout: true,
    theme: "vs-light",
    minimap: {
      enabled: false,
    },
    wordWrap: "on",
    fixedOverflowWidgets: true,
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
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-expect-error
        ?.getValueInRange(editorInstance.getSelection()) as string;

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
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-expect-error
        ?.getValueInRange(editorInstance.getSelection()) as string;

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
      if (readonly.value) {
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
  editorInstance.onDidChangeCursorSelection((e) => {
    const selectedText = editorInstance
      .getModel()
      ?.getValueInRange(e.selection) as string;
    emit("change-selection", selectedText);
  });

  editorInstance.onDidChangeCursorPosition((e) => {
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
};

onMounted(() => {
  init();

  // set the editor focus when the tab is selected
  if (!readonly.value) {
    editorInstance.focus();
    nextTick(() => setPositionAtEndOfLine(editorInstance));
  }
});

onUnmounted(() => {
  editorInstance.dispose();
});

watch(
  () => readonly.value,
  (readOnly) => {
    if (editorInstance) {
      editorInstance.updateOptions({ readOnly });
    }
  },
  {
    deep: true,
    immediate: true,
  }
);

const setEditorContent = (content: string) => {
  setContent(editorInstance, content);
};

const formatEditorContent = () => {
  formatContent(editorInstance, language.value as SQLDialect);
  nextTick(() => {
    setPositionAtEndOfLine(editorInstance);
    editorInstance.focus();
  });
};

defineExpose({
  formatEditorContent,
  setEditorContent,
  setAutoCompletionContext,
});
</script>

<style>
.monaco-editor .cldr.sql-fragment {
  @apply bg-indigo-600;
  width: 3px !important;
  margin-left: 2px;
}
</style>
