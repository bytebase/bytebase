<template>
  <div
    v-show="viewMode !== 'NONE'"
    class="flex-1 max-h-[50vh] flex flex-col gap-y-2"
  >
    <EditorView v-if="viewMode === 'EDITOR'" :key="editorViewKey" />
    <ReleaseView v-else-if="viewMode === 'RELEASE'" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { sheetNameForSpec } from "../../logic";
import { useSelectedSpec } from "../SpecDetailView/context";
import EditorView from "./EditorView";
import ReleaseView from "./ReleaseView";

const selectedSpec = useSelectedSpec();

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
