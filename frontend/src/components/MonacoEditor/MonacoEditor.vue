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
  (e: "run-query", content: string): void;
}>();

const editorRef = ref();
const sqlCode = toRef(props, "modelValue");
const language = toRef(props, "language");

// const vm = getCurrentInstance();
// console.log(vm)
// console.log(vm?.proxy)
// console.log(vm?.proxy?.$store)
// console.log(vm?.ctx)
// console.log(vm?.ctx.$store)

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
      emit("run-query", editorInstance.getValue());
    },
  });

  // for dark mode
  // watch(
  //   () => isDarkmode.value,
  //   (val) => {
  //     // TODO not support have multiple editors of different themes.
  //     monaco.editor.setTheme(val ? "vs-dark" : "vs-light");
  //   },
  //   { immediate: true }
  // );

  editorInstance.onDidChangeModelContent(() => {
    const value = editorInstance.getValue();
    emit("update:modelValue", value);
    emit("change", value);
  });
};

onMounted(init);
</script>
