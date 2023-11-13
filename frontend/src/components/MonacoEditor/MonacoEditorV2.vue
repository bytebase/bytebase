<template>
  <MonacoTextModelEditor
    :model="model"
    @update:content="(...args) => $emit('update:content', ...args)"
  />
</template>

<script setup lang="ts">
import { v4 as uuidv4 } from "uuid";
import { computed, toRef } from "vue";
import type { Language } from "@/types";
import MonacoTextModelEditor from "./MonacoTextModelEditor.vue";
import { useMonacoTextModel } from "./text-model";
import { extensionNameOfLanguage } from "./utils";

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
