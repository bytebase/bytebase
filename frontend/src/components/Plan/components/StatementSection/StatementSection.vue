<template>
  <div v-if="viewMode !== 'NONE'" class="px-4 py-2 flex flex-col gap-y-2">
    <EditorView ref="editorViewRef" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ref } from "vue";
import { usePlanContext } from "../..";
import EditorView from "./EditorView";

const { selectedSpec } = usePlanContext();
const editorViewRef = ref<InstanceType<typeof EditorView>>();

const viewMode = computed((): "NONE" | "EDITOR" => {
  if (selectedSpec.value) {
    return "EDITOR";
  }
  return "NONE";
});
</script>
