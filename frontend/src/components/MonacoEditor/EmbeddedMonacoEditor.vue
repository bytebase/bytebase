<template>
  <div ref="editorRef" class="w-full overflow-hidden"></div>
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
  PropType,
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
  databaseList: {
    type: Array as PropType<Database[]>,
    default: () => [],
  },
  tableList: {
    type: Array as PropType<Table[]>,
    default: () => [],
  },
  minHeight: {
    type: Number,
    default: 0,
  },
  maxHeight: {
    type: Number,
    default: 0,
  },
});

const emit = defineEmits<{
  (e: "change", content: string): void;
}>();

const editorRef = ref();
const statement = toRef(props, "value");
const language = toRef(props, "language");
const readonly = toRef(props, "readonly");

let editorInstance: Editor.IStandaloneCodeEditor;

const {
  monaco,
  formatContent,
  setAutoCompletionContext,
  setPositionAtEndOfLine,
} = await useMonaco(language.value);

const init = async () => {
  const model = monaco.editor.createModel(
    statement.value,
    toRaw(language.value)
  );

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
    scrollBeyondLastLine: false,
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

  // set the editor's autoComplete context
  watch(
    [() => props.databaseList, () => props.tableList],
    ([databaseList, tableList]) => {
      setAutoCompletionContext(databaseList, tableList);
    },
    {
      immediate: true,
    }
  );

  // set the editor focus when the tab is selected
  if (!readonly.value) {
    editorInstance.focus();

    nextTick(() => setPositionAtEndOfLine(editorInstance));
  }

  watch(
    () => readonly.value,
    (readOnly) => {
      if (editorInstance) {
        editorInstance.updateOptions({
          readOnly,
          renderLineHighlight: readOnly ? "none" : "line",
        });
        if (!readOnly) {
          editorInstance.focus();
          nextTick(() => setPositionAtEndOfLine(editorInstance));
        }
      }
    },
    {
      immediate: true,
    }
  );

  const fitSize = () => {
    const container = editorRef.value;
    if (!container) return;

    const contentHeight = editorInstance.getContentHeight();
    let actualHeight = contentHeight;
    if (props.minHeight > 0 && actualHeight < props.minHeight) {
      actualHeight = props.minHeight;
    }
    if (props.maxHeight > 0 && actualHeight > props.maxHeight) {
      actualHeight = props.maxHeight;
    }
    container.style.height = `${actualHeight}px`;
  };

  watch(statement, (statement) => {
    if (statement !== model.getValue()) {
      model.setValue(statement);
    }
    requestAnimationFrame(fitSize);
  });

  fitSize();
};

onMounted(() => {
  init();
});

onUnmounted(() => {
  editorInstance.dispose();
});
</script>

<style>
.monaco-editor .cldr.sql-fragment {
  @apply bg-indigo-600;
  width: 3px !important;
  margin-left: 2px;
}
</style>
