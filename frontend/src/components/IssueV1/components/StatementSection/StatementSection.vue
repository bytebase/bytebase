<template>
  <div v-if="viewMode !== 'NONE'" class="px-4 py-2 flex flex-col gap-y-2">
    <EditorView
      v-if="viewMode === 'EDITOR'"
      ref="editorViewRef"
      :advices="advices"
    />
    <SDLView v-if="viewMode === 'SDL'" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ref } from "vue";
import { nextTick } from "vue";
import { useRouter } from "vue-router";
import { EMPTY_ID, TaskTypeListWithStatement } from "@/types";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import type { Advice } from "@/types/proto/v1/sql_service";
import { useIssueContext } from "../../logic";
import EditorView from "./EditorView";
import SDLView from "./SDLView";

defineProps<{
  advices?: Advice[];
}>();

const { isCreating, selectedTask, selectedSpec } = useIssueContext();

const editorViewRef = ref<InstanceType<typeof EditorView>>();
const router = useRouter();

type ViewMode = "NONE" | "EDITOR" | "SDL";

const viewMode = computed((): ViewMode => {
  if (selectedTask.value.uid !== String(EMPTY_ID)) {
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
  }
  if (selectedSpec.value.id !== String(EMPTY_ID)) {
    return "EDITOR";
  }

  return "NONE";
});

router.afterEach((to) => {
  if (to.hash) {
    scrollToLineByHash(to.hash);
  }
});

const scrollToLineByHash = (hash: string) => {
  const match = hash.match(/^#L(\d+)$/);
  if (!match) return;
  const lineNumber = parseInt(match[1], 10);
  nextTick(() => {
    editorViewRef.value?.editor?.monacoEditor?.editor?.codeEditor?.revealLineNearTop(
      lineNumber
    );
  });
};
</script>
