<template>
  <div v-if="viewMode !== 'NONE'" class="px-4 py-2 flex flex-col gap-y-2">
    <EditorView
      v-if="viewMode === 'EDITOR'"
      ref="editorViewRef"
      :advices="advices"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from "vue";
import { useRouter } from "vue-router";
import { usePlanSQLCheckContext } from "@/components/Plan/components/SQLCheckSection/context";
import { useRouteChangeGuard } from "@/composables/useRouteChangeGuard";
import { useCurrentProjectV1 } from "@/store";
import { TaskTypeListWithStatement } from "@/types";
import { databaseForTask, isValidTaskName } from "@/utils";
import { useIssueContext } from "../../logic";
import EditorView from "./EditorView";

const router = useRouter();
const { isCreating, selectedTask } = useIssueContext();
const { project } = useCurrentProjectV1();
const { resultMap } = usePlanSQLCheckContext();

const editorViewRef = ref<InstanceType<typeof EditorView>>();

const advices = computed(() => {
  const database = databaseForTask(project.value, selectedTask.value);
  return resultMap.value[database.name]?.advices || [];
});

type ViewMode = "NONE" | "EDITOR";

const viewMode = computed((): ViewMode => {
  if (isValidTaskName(selectedTask.value.name)) {
    const task = selectedTask.value;
    const { type } = task;
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

useRouteChangeGuard(
  computed(() => isCreating.value || (editorViewRef.value?.isEditing ?? false))
);
</script>
