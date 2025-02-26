<template>
  <MonacoTextModelEditor
    ref="textModelEditorRef"
    class="bb-monaco-editor"
    :model="model"
    @update:content="handleChange"
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
import { v4 as uuidv4 } from "uuid";
import { computed, toRef } from "vue";
import { ref } from "vue";
import type { Language } from "@/types";
import MonacoTextModelEditor from "./MonacoTextModelEditor.vue";
import { useSQLParser } from "./composables";
import { useMonacoTextModel, getUriByFilename } from "./text-model";
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

const filename = computed(() => {
  if (props.filename) return props.filename;

  return `${uuidv4()}.${extensionNameOfLanguage(props.language)}`;
});

const model = useMonacoTextModel(filename, content, toRef(props, "language"));

const handleChange = (value: string) => {
  emit("update:content", value);
};

defineExpose({
  get editor() {
    return textModelEditorRef.value;
  },
  getActiveStatementByCursor: () => {
    if (!textModelEditorRef.value || !textModelEditorRef.value.codeEditor) {
      return "";
    }
    const model = textModelEditorRef.value.codeEditor.getModel();
    if (!model) {
      return "";
    }

    const position = textModelEditorRef.value.codeEditor.getPosition();
    if (!position) {
      return "";
    }
    const range = getActiveStatementRange(
      getUriByFilename(filename.value).toString(),
      position.lineNumber
    );
    if (!range) {
      return "";
    }
    textModelEditorRef.value.codeEditor.setSelection(range);
    return model.getValueInRange(range);
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
