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
import { computed, nextTick, ref, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { TaskTypeListWithStatement } from "@/types";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import type { Advice } from "@/types/proto/v1/sql_service";
import { isValidTaskName } from "@/utils";
import { useIssueContext } from "../../logic";
import EditorView from "./EditorView";
import SDLView from "./SDLView";

defineProps<{
  advices?: Advice[];
}>();

const { isCreating, selectedTask } = useIssueContext();

const editorViewRef = ref<InstanceType<typeof EditorView>>();
const router = useRouter();

type ViewMode = "NONE" | "EDITOR" | "SDL";

const viewMode = computed((): ViewMode => {
  if (isValidTaskName(selectedTask.value.name)) {
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

const beforeUnloadHandler = (e: BeforeUnloadEvent) => {
  // For created issues and not editing, no need to prompt.
  if (!isCreating.value && !editorViewRef.value?.isEditing) {
    return;
  }

  // For creating issues or editing task statement, prompt to confirm leaving.
  e.preventDefault();
  // Included for legacy support, e.g. Chrome/Edge < 119
  e.returnValue = true;
};

onMounted(() => {
  window.addEventListener("beforeunload", beforeUnloadHandler);
});

onUnmounted(() => {
  window.removeEventListener("beforeunload", beforeUnloadHandler);
});
</script>
