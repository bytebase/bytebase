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
import { usePlanSQLCheckContext } from "@/components/Plan/components/SQLCheckSection/context";
import { databaseForTask } from "@/components/Rollout/RolloutDetail";
import { useCurrentProjectV1 } from "@/store";
import { TaskTypeListWithStatement } from "@/types";
import { isValidTaskName } from "@/utils";
import { useEventListener } from "@vueuse/core";
import { computed, nextTick, ref } from "vue";
import { useI18n } from "vue-i18n";
import { onBeforeRouteLeave, useRouter } from "vue-router";
import { useIssueContext } from "../../logic";
import EditorView from "./EditorView";

const { t } = useI18n();
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

const beforeUnloadHandler = (e: BeforeUnloadEvent) => {
  // For created issues and not editing, no need to prompt.
  if (!isCreating.value && !editorViewRef.value?.isEditing) {
    return;
  }

  // For creating issues or editing task statement, prompt to confirm leaving.
  e.preventDefault();
  // Included for legacy support, e.g. Chrome/Edge < 119
  e.returnValue = t("common.leave-without-saving");
  return e.returnValue;
};

useEventListener("beforeunload", beforeUnloadHandler);

onBeforeRouteLeave((to, from, next) => {
  if (isCreating.value || editorViewRef.value?.isEditing) {
    if (!window.confirm(t("common.leave-without-saving"))) {
      return;
    }
  }
  next();
});
</script>
