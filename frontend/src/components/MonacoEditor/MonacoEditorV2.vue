<template>
  <MonacoTextModelEditor :model="model" />
</template>

<script setup lang="ts">
import { computed, toRef } from "vue";
import MonacoTextModelEditor from "./MonacoTextModelEditor.vue";
import { Language, useMonacoTextModel } from "./text-model";

const props = defineProps<{
  filename: string;
  content: string;
  language: Language;
}>();
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
const model = useMonacoTextModel(
  toRef(props, "filename"),
  content,
  toRef(props, "language")
);
</script>
