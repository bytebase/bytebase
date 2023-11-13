<template>
  <MonacoTextModelEditor :model="model" />
</template>

<script setup lang="ts">
import type monaco from "monaco-editor";
import { v4 as uuidv4 } from "uuid";
import { computed, toRef } from "vue";
import type { Language } from "@/types";
import MonacoTextModelEditor from "./MonacoTextModelEditor.vue";
import { useMonacoTextModel } from "./text-model";
import type { MonacoModule } from "./types";

const props = withDefaults(
  defineProps<{
    filename: string;
    content: string;
    language: Language;
  }>(),
  {
    filename: () => uuidv4(),
  }
);
const emit = defineEmits<{
  (event: "update:content", content: string): void;
  (e: "update:selected-content", content: string): void;
  (
    e: "ready",
    monaco: MonacoModule,
    editor: monaco.editor.IStandaloneCodeEditor
  ): void;
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
