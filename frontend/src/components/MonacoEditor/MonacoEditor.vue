<template>
  <div ref="editorRef" style="height: 100%; width: 100%"></div>
</template>

<script lang="ts" setup>
import {
  onMounted,
  ref,
  toRef,
  toRaw,
  PropType,
  nextTick,
  defineProps,
  defineEmits,
  onUnmounted,
} from "vue";
import type { editor as Editor } from "monaco-editor";

import { useMonaco } from "./useMonaco";
import { SqlDialect } from "../../types";

const props = defineProps({
  value: {
    type: String,
    required: true,
  },
  language: {
    type: String as PropType<SqlDialect>,
    default: "mysql",
  },
});

const emit = defineEmits<{
  (e: "update:value", content: string): void;
  (e: "change", content: string): void;
  (e: "change-selection", content: string): void;
  (e: "run-query", content: string): void;
}>();

const editorRef = ref();
const sqlCode = toRef(props, "value");
const language = toRef(props, "language");

let editorInstance: Editor.IStandaloneCodeEditor;

const {
  monaco,
  setPositionAtEndOfLine,
  formatContent,
  completionItemProvider,
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

      const queryStatement = selectedValue || typedValue;

      emit("run-query", queryStatement);
    },
  });

  // add format sql action in context menu
  editorInstance.addAction({
    id: "FormatSQL",
    label: "Format SQL",
    keybindings: [
      monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.KeyF,
    ],
    contextMenuGroupId: "operation",
    contextMenuOrder: 1,
    run: () => {
      formatContent(editorInstance, language.value);
      nextTick(() => setPositionAtEndOfLine(editorInstance));
    },
  });

  // typed something, change the text
  editorInstance.onDidChangeModelContent(() => {
    const value = editorInstance.getValue();
    // emit("update:value", value);
    emit("change", value);
  });

  // when editor change selection, emit change-selection event with selected text
  editorInstance.onDidChangeCursorSelection((e) => {
    const selectedText = editorInstance.getModel()?.getValueInRange({
      startLineNumber: e.selection.startLineNumber,
      startColumn: e.selection.startColumn,
      endLineNumber: e.selection.endLineNumber,
      endColumn: e.selection.endColumn,
    }) as string;
    emit("change-selection", selectedText);
  });
};

onMounted(init);

onUnmounted(() => {
  completionItemProvider.dispose();
  editorInstance.dispose();
});
</script>
