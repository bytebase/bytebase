<template>
  <div class="stage" :class="stageClass">
    <TaskStatusIcon
      :create="isCreating"
      :active="isActiveStage"
      :status="activeTaskInStage.status"
      :ignore-task-check-status="true"
    />

    <div class="text" @click="handleClickStage">
      <div class="text-sm min-w-32 lg:min-w-fit with-underline space-x-1">
        <heroicons:arrow-small-right
          v-if="isActiveStage"
          class="w-5 h-5 inline-block mb-0.5"
        />
        <span>{{ $t("common.stage") }}</span>
        <span>-</span>
        <span>{{ stageTitle }}</span>
      </div>
      <div class="text-xs flex gap-1 flex-row items-center">
        <div class="whitespace-pre-wrap break-all with-underline">
          {{ taskTitle }}
        </div>
        <StageSummary :stage="(stage as Stage)" />
      </div>
    </div>

    <NTooltip v-if="isCreating && !isValid" trigger="hover" placement="top">
      <template #trigger>
        <heroicons:exclamation-circle-solid
          class="w-6 h-6 ml-2 text-error hover:text-error-hover"
        />
      </template>
      <span>Missing SQL statement</span>
    </NTooltip>
  </div>
</template>

<script lang="ts" setup>
import { first } from "lodash-es";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  isTaskFinished,
  isValidStage,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { EMPTY_TASK_NAME, emptyTask } from "@/types";
import { Stage, task_StatusToJSON } from "@/types/proto/v1/rollout_service";
import { activeTaskInStageV1 } from "@/utils";
import TaskStatusIcon from "../TaskStatusIcon.vue";
import StageSummary from "./StageSummary.vue";

const props = defineProps<{
  stage: Stage;
}>();

const { t } = useI18n();
const { isCreating, activeTask, selectedStage, events } = useIssueContext();

const activeTaskInStage = computed(() => {
  return activeTaskInStageV1(props.stage);
});

const isValid = computed(() => {
  return isValidStage(props.stage);
});

const isSelectedStage = computed(() => {
  return props.stage === selectedStage.value;
});

const isActiveStage = computed(() => {
  if (isCreating.value) {
    // In create mode we don't have an ActiveStage
    return false;
  }

  const taskFound = props.stage.tasks.find(
    (t) => t.uid === activeTask.value.uid
  );
  if (taskFound && !isTaskFinished(taskFound)) {
    // A stage is "Active" if the ActiveTaskOfPipeline is inside this stage
    return true;
  }

  return false;
});

const stageClass = computed(() => {
  const classList: string[] = [];
  if (isCreating.value) {
    classList.push("create");
    if (!isValid.value) {
      classList.push("invalid");
    }
  }
  if (isSelectedStage.value) classList.push("selected");
  if (isActiveStage.value) classList.push("active");
  const task = activeTaskInStage.value;
  classList.push(`status_${task_StatusToJSON(task.status).toLowerCase()}`);

  return classList;
});

const stageTitle = computed(() => {
  const { stage } = props;
  return !isCreating.value && isActiveStage.value
    ? t("issue.stage-select.active", { name: stage.title })
    : stage.title;
});

const taskTitle = computed(() => {
  const task = isCreating ? first(props.stage.tasks) : activeTaskInStage.value;
  return (task ?? emptyTask()).title;
});

const activeOrFirstTaskInStage = (stage: Stage) => {
  if (isCreating.value) {
    return first(stage.tasks);
  }
  const activeTask = activeTaskInStageV1(stage);
  if (activeTask.name === EMPTY_TASK_NAME) {
    return first(stage.tasks);
  }
  return activeTask;
};

const handleClickStage = () => {
  const { stage } = props;
  if (stage === selectedStage.value) return;

  if (stage) {
    const task = activeOrFirstTaskInStage(stage);
    if (task) {
      events.emit("select-task", { task });
    }
  }
};
</script>

<style scoped lang="postcss">
.stage {
  @apply cursor-default flex items-center justify-start w-full text-sm font-medium relative;
  @apply lg:flex-1;
}
.stage.invalid {
  @apply pr-10;
}
.stage.selected .text .with-underline {
  @apply underline;
}

.stage .text {
  @apply cursor-pointer flex-1 ml-4 flex flex-col gap-y-0.5;
}
.stage.active .text {
  @apply font-bold;
}
.stage.status_done .text {
  @apply text-control;
}
.stage.status_pending .text,
.stage.status_pending_approval .text {
  @apply text-control;
}
.stage.active.status_pending .text,
.stage.active.status_pending_approval .text {
  @apply text-info;
}
.stage.status_running .text {
  @apply text-info;
}
.stage.status_failed .text {
  @apply text-red-500;
}
</style>
