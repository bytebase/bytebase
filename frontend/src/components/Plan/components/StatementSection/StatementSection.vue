<template>
  <div v-if="viewMode !== 'NONE'" class="px-4 py-2 flex flex-col gap-y-2">
    <EditorView v-if="viewMode === 'EDITOR'" />
    <ReleaseView v-else-if="viewMode === 'RELEASE'" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { usePlanContext } from "../..";
import EditorView from "./EditorView";
import ReleaseView from "./ReleaseView";

const { selectedSpec } = usePlanContext();

const viewMode = computed((): "NONE" | "EDITOR" | "RELEASE" => {
  if (selectedSpec.value) {
    const spec = selectedSpec.value;
    // Check if this is a release-based spec (has release but no sheet)
    if (
      spec.changeDatabaseConfig?.release &&
      !spec.changeDatabaseConfig?.sheet
    ) {
      return "RELEASE";
    }
    // Otherwise, it's a sheet-based spec
    return "EDITOR";
  }
  return "NONE";
});
</script>
