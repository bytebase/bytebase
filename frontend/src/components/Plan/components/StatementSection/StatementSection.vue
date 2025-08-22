<template>
  <div v-show="viewMode !== 'NONE'" class="flex flex-col gap-y-2">
    <EditorView v-if="viewMode === 'EDITOR'" :key="selectedSpec.id" />
    <ReleaseView v-else-if="viewMode === 'RELEASE'" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
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
</script>
