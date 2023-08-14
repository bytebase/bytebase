<template>
  <div v-if="viewMode !== 'NONE'" class="px-4 py-2 flex flex-col gap-y-2">
    <EditorView v-if="viewMode === 'EDITOR'" />
    <SDLView v-if="viewMode === 'SDL'" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { TaskTypeListWithStatement } from "@/types";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import { useIssueContext } from "../../logic";
import EditorView from "./EditorView";
import SDLView from "./SDLView";

const { isCreating, selectedTask } = useIssueContext();

type ViewMode = "NONE" | "EDITOR" | "SDL";

const viewMode = computed((): ViewMode => {
  const task = selectedTask.value;
  const { type } = task;
  if (type === Task_Type.DATABASE_SCHEMA_UPDATE_SDL) {
    return "SDL";
  }
  if (type === Task_Type.DATABASE_SCHEMA_BASELINE) {
    return isCreating.value ? "EDITOR" : "NONE";
  }
  if (TaskTypeListWithStatement.includes(type)) {
    return "EDITOR";
  }

  return "NONE";
});
</script>
