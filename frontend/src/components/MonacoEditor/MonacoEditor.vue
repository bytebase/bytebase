<template>
  <MonacoTextModelEditor
    class="bb-monaco-editor"
    :model="model"
    @update:content="(...args) => $emit('update:content', ...args)"
  />
</template>

<script setup lang="ts">
import { v4 as uuidv4 } from "uuid";
import { computed, toRef } from "vue";
import type { Language } from "@/types";

const [
  { default: MonacoTextModelEditor },
  { useMonacoTextModel },
  { extensionNameOfLanguage },
] = await Promise.all([
  import("./MonacoTextModelEditor.vue"),
  import("./text-model"),
  import("./utils"),
]);

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

const filename = computed(() => {
  if (props.filename) return props.filename;

  return `${uuidv4()}.${extensionNameOfLanguage(props.language)}`;
});
const model = useMonacoTextModel(filename, content, toRef(props, "language"));
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
