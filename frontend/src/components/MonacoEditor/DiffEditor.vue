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
import { editor as Editor } from "monaco-editor";
import {
  onMounted,
  ref,
  toRef,
  nextTick,
  watch,
  shallowRef,
  PropType,
  onBeforeUnmount,
} from "vue";
import { Language } from "@/types";

const props = defineProps({
  original: {
    type: String,
    default: "",
  },
  value: {
    type: String,
    default: "",
  },
  language: {
    type: String as PropType<Language>,
    default: "sql",
  },
  readonly: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits<{
  (e: "change", content: string): void;
  (e: "ready"): void;
}>();

const sqlCode = toRef(props, "value");
const readOnly = toRef(props, "readonly");
const editorContainerRef = ref<HTMLDivElement>();
// use shallowRef to avoid deep conversion which will cause page crash.
const editorInstanceRef = shallowRef<Editor.IStandaloneDiffEditor>();

const isEditorLoaded = ref(false);

const initEditorInstance = () => {
  const originalModel = Editor.createModel(props.original, props.language);
  const modifiedEditor = Editor.createModel(sqlCode.value, props.language);
  const editorInstance = Editor.createDiffEditor(editorContainerRef.value!, {
    readOnly: readOnly.value,
    // Learn more: https://github.com/microsoft/monaco-editor/issues/311
    enableSplitViewResizing: false,
    renderValidationDecorations: "on",
    theme: "bb",
    autoClosingQuotes: "always",
    folding: false,
    automaticLayout: true,
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
  });

  editorInstance.setModel({
    original: originalModel,
    modified: modifiedEditor,
  });

  // When typed something, change the text
  editorInstance.onDidUpdateDiff(() => {
    const value = editorInstance.getModifiedEditor().getValue();
    emit("change", value);
  });

  return editorInstance;
};

onMounted(async () => {
  if (!editorContainerRef.value) {
    // Give up creating monaco editor if the component has been unmounted
    // very quickly.
    console.debug(
      "<MonacoEditor> has been unmounted before useMonaco is ready"
    );
    return;
  }

  const editorInstance = initEditorInstance();
  editorInstanceRef.value = editorInstance;
  isEditorLoaded.value = true;

  nextTick(() => {
    emit("ready");
  });
});

onBeforeUnmount(() => {
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
</script>

<style>
.monaco-editor .monaco-mouse-cursor-text {
  box-shadow: none !important;
}
.monaco-editor .scroll-decoration {
  display: none !important;
}
</style>
