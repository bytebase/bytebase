<template>
  <div v-if="show" class="w-full px-4 py-2 flex flex-col gap-y-2">
    <div class="stage" :class="stageClass">
      <TaskStatusIcon
        :create="isCreating"
        :active="isActiveStage"
        :status="activeTaskInStage.status"
        :ignore-task-check-status="true"
      />

      <div class="text">
        <div class="text-sm min-w-32 lg:min-w-fit with-underline space-x-1">
          <heroicons:arrow-small-right
            v-if="isActiveStage"
            class="w-5 h-5 inline-block mb-0.5"
          />
          <span>{{ $t("common.stage") }} - {{ stage.title }}</span>
        </div>
        <div class="text-xs flex flex-col gap-1 md:flex-row md:items-center">
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

    <div class="lg:flex items-start justify-between">
      <StageInfo />

      <Actions />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NTooltip } from "naive-ui";
import { first } from "lodash-es";

import { Stage, task_StatusToJSON } from "@/types/proto/v1/rollout_service";
import { EMPTY_STAGE_NAME, emptyTask } from "@/types";
import { activeTaskInStageV1 } from "@/utils";
import {
  isTaskFinished,
  isValidStage,
  useIssueContext,
} from "@/components/IssueV1/logic";
import TaskStatusIcon from "../TaskStatusIcon.vue";
import StageSummary from "./StageSummary.vue";
import StageInfo from "./StageInfo";
import Actions from "./Actions";

const { isCreating, activeTask, selectedStage: stage } = useIssueContext();

const show = computed(() => {
  return stage.value.name !== EMPTY_STAGE_NAME;
});

const activeTaskInStage = computed(() => {
  return activeTaskInStageV1(stage.value);
});

const isValid = computed(() => {
  return isValidStage(stage.value);
});

const isActiveStage = computed(() => {
  if (isCreating.value) {
    // In create mode we don't have an ActiveStage
    return false;
  }

  const taskFound = stage.value.tasks.find(
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
  if (isActiveStage.value) classList.push("active");
  const task = activeTaskInStage.value;
  classList.push(`status_${task_StatusToJSON(task.status).toLowerCase()}`);

  return classList;
});

const taskTitle = computed(() => {
  const task = isCreating ? first(stage.value.tasks) : activeTaskInStage.value;
  return (task ?? emptyTask()).title;
});
</script>

<style scoped lang="postcss">
.stage {
  @apply cursor-default flex items-center justify-start w-full text-sm font-medium relative;
  @apply lg:w-auto lg:flex-1;
}
.stage.invalid {
  @apply pr-10;
}

.stage .text {
  @apply cursor-pointer ml-4 flex flex-col gap-y-0.5;
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
