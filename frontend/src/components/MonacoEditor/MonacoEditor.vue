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
  watch,
} from "vue";
import { useStore } from "vuex";
import type { editor as Editor } from "monaco-editor";

import { useMonaco } from "./useMonaco";
import {
  TabGetters,
  SqlDialect,
  SqlEditorActions,
  SqlEditorState,
  SheetGetters,
} from "../../types";
import {
  useNamespacedActions,
  useNamespacedGetters,
  useNamespacedState,
} from "vuex-composition-helpers";

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
  (e: "save", content: string): void;
}>();

const editorRef = ref();
const sqlCode = toRef(props, "value");
const language = toRef(props, "language");

const store = useStore();
const { shouldSetContent } = useNamespacedState<SqlEditorState>("sqlEditor", [
  "shouldSetContent",
]);
const { currentTab } = useNamespacedGetters<TabGetters>("tab", ["currentTab"]);
const { isReadOnly } = useNamespacedGetters<SheetGetters>("sheet", [
  "isReadOnly",
]);
const { setShouldSetContent } = useNamespacedActions<SqlEditorActions>(
  "sqlEditor",
  ["setShouldSetContent"]
);

let editorInstance: Editor.IStandaloneCodeEditor;

const {
  monaco,
  setPositionAtEndOfLine,
  formatContent,
  setContent,
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

      const query = selectedValue || typedValue;
      emit("run-query", query);
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
      if (isReadOnly.value) {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "INFO",
          title: "The shared sheet is read-only.",
        });
        return;
      }
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

  editorInstance.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
    const value = editorInstance.getValue();
    emit("save", value);
  });

  watch(
    () => isReadOnly.value,
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
};

onMounted(init);

onUnmounted(() => {
  completionItemProvider.dispose();
  editorInstance.dispose();
});

watch(
  () => shouldSetContent.value,
  () => {
    if (shouldSetContent.value) {
      setShouldSetContent(false);
      setContent(editorInstance, currentTab.value.statement);
    }
  }
);
</script>
