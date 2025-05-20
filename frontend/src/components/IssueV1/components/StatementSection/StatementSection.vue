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
import { useEventListener } from "@vueuse/core";
import { computed, nextTick, ref } from "vue";
import { useI18n } from "vue-i18n";
import { onBeforeRouteLeave } from "vue-router";
import { useRouter } from "vue-router";
import { databaseForTask } from "@/components/Rollout/RolloutDetail";
import { TaskTypeListWithStatement } from "@/types";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import { isValidTaskName } from "@/utils";
import { useIssueContext } from "../../logic";
import { useIssueSQLCheckContext } from "../SQLCheckSection/context";
import EditorView from "./EditorView";
import SDLView from "./SDLView";

const { t } = useI18n();
const router = useRouter();
const { issue, isCreating, selectedTask } = useIssueContext();
const { resultMap } = useIssueSQLCheckContext();

const editorViewRef = ref<InstanceType<typeof EditorView>>();

const advices = computed(() => {
  const database = databaseForTask(
    issue.value.projectEntity,
    selectedTask.value
  );
  return resultMap.value[database.name]?.advices || [];
});

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
