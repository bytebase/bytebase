<template>
  <MonacoTextModelEditor
    ref="textModelEditorRef"
    class="bb-monaco-editor"
    :model="model"
    @update:content="handleContentChange"
    @update:selection="emit('update:selection', $event)"
    @update:active-content="(c) => (activeContent = c)"
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
import { computed, toRef, ref } from "vue";
import type { Language } from "@/types";
import MonacoTextModelEditor from "./MonacoTextModelEditor.vue";
import { useMonacoTextModel } from "./text-model";
import type { Selection } from "./types";
import { extensionNameOfLanguage } from "./utils";

const textModelEditorRef = ref<InstanceType<typeof MonacoTextModelEditor>>();

const props = withDefaults(
  defineProps<{
    content: string;
    filename?: string;
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

const activeContent = ref<string>("");

const handleContentChange = (value: string) => {
  content.value = value;
};

const filename = computed(() => {
  if (props.filename) return props.filename;

  return `${uuidv4()}.${extensionNameOfLanguage(props.language)}`;
});

const model = useMonacoTextModel(filename, content, toRef(props, "language"));

defineExpose({
  get editor() {
    return textModelEditorRef.value;
  },
  getActiveStatement: () => activeContent.value,
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
