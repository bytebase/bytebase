<template>
  <div ref="editorRef" style="height: 100%; width: 100%"></div>
</template>

<script lang="ts" setup>
import { onMounted, ref, toRef, toRaw } from "vue";
import setupMonaco from "./setupMonaco";

const props = defineProps({
  modelValue: { type: String, default: "" },
  language: { type: String, default: "mysql" },
});

const emit = defineEmits<{
  (e: "update:modelValue", content: string): void;
  (e: "change", content: string): void;
  (e: "change-selection", content: string): void;
  (e: "run-query", content: string): void;
}>();

const editorRef = ref();
const sqlCode = toRef(props, "modelValue");
const language = toRef(props, "language");

// let editorInstance: Editor.IStandaloneCodeEditor

// const setContent = (content: string) => {
//   if (editorInstance) editorInstance.setValue(content)
// }

// const formatContent = () => {
//   if (editorInstance) editorInstance.getAction('editor.action.formatDocument').run()
// }

const init = async () => {
  const { monaco } = await setupMonaco(language.value);

  const model = monaco.editor.createModel(sqlCode.value, toRaw(language.value));

  const editorInstance = monaco.editor.create(editorRef.value, {
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
    id: "Bytebase",
    label: "Run Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
    contextMenuGroupId: "operation",
    contextMenuOrder: 0,
    run: async () => {
      console.log("run query");

      const typedValue = editorInstance.getValue();
      const selectedValue = editorInstance
        .getModel()
        // @ts-expect-error
        ?.getValueInRange(editorInstance.getSelection()) as string;

      const queryStatement = selectedValue || typedValue;

      emit("run-query", queryStatement);
    },
  });

  // typed something, change the text
  editorInstance.onDidChangeModelContent(() => {
    const value = editorInstance.getValue();
    emit("update:modelValue", value);
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
</script>
