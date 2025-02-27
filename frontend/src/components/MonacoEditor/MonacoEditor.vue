<template>
  <MonacoTextModelEditor
    ref="textModelEditorRef"
    class="bb-monaco-editor"
    :model="model"
    :line-highlights="lineHighlights"
    @update:content="handleContentChange"
    @update:selection="handleSelectionChange"
  >
    <template v-if="$slots['corner-prefix']" #corner-prefix>
      <slot name="corner-prefix" />
    </template>
    <template v-if="$slots['corner-suffix']" #corner-suffix>
      <slot name="corner-suffix" />
    </template>
  </MonacoTextModelEditor>
</template>

<script setup lang="ts">
import { editor, type IRange } from "monaco-editor";
import { v4 as uuidv4 } from "uuid";
import { computed, toRef, watchEffect } from "vue";
import { ref } from "vue";
import type { Language } from "@/types";
import { isDev } from "@/utils";
import MonacoTextModelEditor from "./MonacoTextModelEditor.vue";
import { useSQLParser } from "./composables";
import { useMonacoTextModel, getUriByFilename } from "./text-model";
import type { LineHighlightOption, Selection } from "./types";
import { extensionNameOfLanguage } from "./utils";

const textModelEditorRef = ref<InstanceType<typeof MonacoTextModelEditor>>();

const props = withDefaults(
  defineProps<{
    filename?: string;
    content: string;
    language?: Language;
  }>(),
  {
    filename: undefined,
    language: "sql",
  }
);
const emit = defineEmits<{
  (event: "update:content", content: string): void;
  (event: "update:selection", selection: Selection | null): void;
}>();

const content = computed({
  get() {
    return props.content;
  },
  set(content) {
    emit("update:content", content);
  },
});

const { getActiveStatementRange } = useSQLParser();
const activeLineNumber = ref<number | undefined>();
const activeSelection = ref<Selection | null>();

const handleSelectionChange = (selection: Selection | null) => {
  activeSelection.value = selection;
  emit("update:selection", selection);
};

const handleContentChange = (value: string) => {
  content.value = value;
};

watchEffect(() => {
  textModelEditorRef.value?.codeEditor?.onDidChangeCursorPosition(
    (e: editor.ICursorPositionChangedEvent) => {
      activeLineNumber.value = e.position.lineNumber;
    }
  );
});

const activeRangeByCursor = computed((): IRange | undefined => {
  if (activeLineNumber.value === undefined) {
    return;
  }
  if (!textModelEditorRef.value?.codeEditor) {
    return;
  }
  const model = textModelEditorRef.value.codeEditor.getModel();
  if (!model) {
    return;
  }
  const range = getActiveStatementRange(
    getUriByFilename(filename.value).toString(),
    activeLineNumber.value
  );
  if (!range) {
    return;
  }

  // Check if the last line is empty
  const lastLineStatement = model.getValueInRange({
    startLineNumber: range.endLineNumber,
    startColumn: 1,
    endLineNumber: range.endLineNumber,
    endColumn: range.endColumn,
  });
  if (!lastLineStatement && range.endLineNumber > range.startLineNumber) {
    const newRange = {
      startLineNumber: range.startLineNumber,
      startColumn: range.startColumn,
      endLineNumber: range.endLineNumber - 1,
      endColumn: Infinity,
    };
    if (activeLineNumber.value > newRange.endLineNumber) {
      return;
    }
    return newRange;
  }

  return range;
});

const activeRange = computed((): IRange | undefined => {
  const rangeByCursor = activeRangeByCursor.value;
  const rangeBySelection = activeSelection.value;

  if (!rangeBySelection) {
    return rangeByCursor;
  }
  if (
    rangeBySelection.startLineNumber === rangeBySelection.endLineNumber &&
    rangeBySelection.startColumn === rangeBySelection.endColumn
  ) {
    // Means no selection, just active cursor.
    return rangeByCursor;
  }

  if (!rangeByCursor) {
    return rangeBySelection;
  }

  // Calculate the maximum range
  return {
    startLineNumber: Math.min(
      rangeByCursor.startLineNumber,
      rangeBySelection.startLineNumber
    ),
    startColumn: Math.min(
      rangeByCursor.startColumn,
      rangeBySelection.startColumn
    ),
    endLineNumber: Math.max(
      rangeByCursor.endLineNumber,
      rangeBySelection.endLineNumber
    ),
    endColumn: Math.max(rangeByCursor.endColumn, rangeBySelection.endColumn),
  };
});

const lineHighlights = computed((): LineHighlightOption[] => {
  const range = activeRange.value;
  if (!range || !isDev()) {
    return [];
  }
  return [
    {
      startLineNumber: range.startLineNumber,
      endLineNumber: range.endLineNumber,
      options: {
        blockClassName: "border border-accent",
      },
    },
  ];
});

const filename = computed(() => {
  if (props.filename) return props.filename;

  return `${uuidv4()}.${extensionNameOfLanguage(props.language)}`;
});

const model = useMonacoTextModel(filename, content, toRef(props, "language"));

defineExpose({
  get editor() {
    return textModelEditorRef.value;
  },
  getActiveStatement: () => {
    if (!activeRange.value) {
      return "";
    }
    const model = textModelEditorRef.value?.codeEditor?.getModel();
    textModelEditorRef.value?.codeEditor?.setSelection(activeRange.value);
    return model?.getValueInRange(activeRange.value);
  },
});
</script>

<style lang="postcss" scoped>
.bb-monaco-editor :deep(.monaco-editor .monaco-mouse-cursor-text) {
  box-shadow: none !important;
}
.bb-monaco-editor :deep(.monaco-editor .scroll-decoration) {
  display: none !important;
}
.bb-monaco-editor :deep(.monaco-editor .line-numbers) {
  @apply pr-2;
}
</style>
