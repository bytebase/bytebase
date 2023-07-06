<template>
  <div class="issue-debug">
    <div>activeStage: {{ activeStage.name }} '{{ activeStage.title }}'</div>
    <div>
      selectedStage: {{ selectedStage.name }} '{{ selectedStage.title }}'
    </div>
  </div>

  <div v-if="show" class="max-w-full lg:flex divide-y lg:divide-y-0">
    <div class="stage-item" :class="stageClass(stage)">
      <TaskStatusIcon
        :create="isCreating"
        :active="isActiveStage(stage)"
        :status="activeTaskInStageV1(stage).status"
        :ignore-task-check-status="true"
      />

      <div class="text">
        <span class="text-sm min-w-32 lg:min-w-fit with-underline">
          {{ $t("common.stage") }} - {{ stage.title }}
        </span>
        <span class="text-xs flex flex-col gap-1 md:flex-row md:items-center">
          <slot name="task-name-of-stage" :stage="stage" :index="0">
            <div class="whitespace-pre-wrap break-all with-underline">
              {{ taskTitleOfStage(stage) }}
            </div>
            <StageSummary :stage="(stage as Stage)" />
          </slot>
        </span>
      </div>

      <NPopover v-if="!isValidStage(stage)" trigger="hover" placement="top">
        <template #trigger>
          <span
            class="ml-2 w-5 h-5 flex justify-center rounded-full select-none bg-error text-white hover:bg-error-hover font-normal absolute right-3"
          >
            !
          </span>
        </template>
        <span>Missing SQL statement</span>
      </NPopover>
    </div>
  </div>

  <div v-if="show" class="flex items-center justify-between">
    <div class="flex items-center justify-start">
      <StageSelect />
    </div>
    <div class="flex items-center justify-end">right</div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NPopover } from "naive-ui";
import { first } from "lodash-es";

import { EMPTY_STAGE_NAME, emptyTask } from "@/types";
import TaskStatusIcon from "./TaskStatusIcon.vue";
import StageSummary from "./StageSummary.vue";
import StageSelect from "./StageSelect.vue";
import { activeTaskInStageV1, activeTaskInRollout } from "@/utils";
import { useIssueContext } from "../logic";
import { Stage, task_StatusToJSON } from "@/types/proto/v1/rollout_service";

const { isCreating, issue, activeStage, selectedStage } = useIssueContext();

const stage = selectedStage;

const show = computed(() => {
  return stage.value.name !== EMPTY_STAGE_NAME;
});

const isValidStage = (stage: Stage): boolean => {
  return true; // todo
};

const isActiveStage = (stage: Stage): boolean => {
  if (isCreating.value) {
    // In create mode we don't have an ActiveStage
    return false;
  }

  const activeTask = activeTaskInRollout(issue.value.rolloutEntity);
  const taskFound = stage.tasks.find((t) => t.uid === activeTask.uid);
  if (taskFound) {
    // A stage is "Active" if the ActiveTaskOfPipeline is inside this stage
    return true;
  }

  return false;
};

const stageClass = (stage: Stage): string[] => {
  const classList: string[] = [];

  if (!isValidStage(stage)) classList.push("invalid");
  if (isCreating.value) classList.push("create");
  if (isActiveStage(stage)) classList.push("active");
  const task = activeTaskInStageV1(stage);
  classList.push(`status_${task_StatusToJSON(task.status).toLowerCase()}`);

  return classList;
};

const taskTitleOfStage = (stage: Stage) => {
  const task = isCreating ? first(stage.tasks) : activeTaskInStageV1(stage);
  return (task ?? emptyTask()).title;
};
</script>

<style scoped lang="postcss">
.stage-item {
  @apply cursor-default flex items-center justify-start w-full px-4 py-2 text-sm font-medium relative;
  @apply lg:w-auto lg:flex-1;
}
.stage-item.invalid {
  @apply pr-10;
}

.stage-item .text {
  @apply cursor-pointer ml-4 flex-col space-y-1;
}
.stage-item.active .text {
  @apply font-bold;
}
.stage-item.status_done .text {
  @apply text-control;
}
.stage-item.status_pending .text,
.stage-item.status_pending_approval .text {
  @apply text-control;
}
.stage-item.active.status_pending .text,
.stage-item.active.status_pending_approval .text {
  @apply text-info;
}
.stage-item.status_running .text {
  @apply text-info;
}
.stage-item.status_failed .text {
  @apply text-red-500;
}
</style>
