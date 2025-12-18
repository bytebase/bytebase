<template>
  <div v-if="viewMode === 'EDITOR'" class="flex-1">
    <EditorView :key="editorViewKey" />
  </div>
  <ReleaseView v-else-if="viewMode === 'RELEASE'" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import { sheetNameForSpec } from "../../logic";
import { useSelectedSpec } from "../SpecDetailView/context";
import EditorView from "./EditorView";
import ReleaseView from "./ReleaseView";

const { selectedSpec } = useSelectedSpec();

const viewMode = computed((): "NONE" | "EDITOR" | "RELEASE" => {
  if (selectedSpec.value) {
    const spec = selectedSpec.value;
    // Check if this is a release-based spec first.
    if (
      spec.config?.case === "changeDatabaseConfig" &&
      spec.config.value.release
    ) {
      return "RELEASE";
    }
    // Otherwise, it's a sheet-based spec.
    return "EDITOR";
  }
  return "NONE";
});

const editorViewKey = computed(() => {
  return `${selectedSpec.value.id}-${sheetNameForSpec(selectedSpec.value)}`;
});
</script>
