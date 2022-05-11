<template>
  <div class="lg:flex divide-y lg:divide-y-0">
    <div
      v-for="(stage, i) in issue.pipeline?.stageList"
      :key="i"
      class="stage-item"
      :class="stageClass(stage, i)"
    >
      <TaskStatusIcon
        :create="create"
        :active="isActiveStage(stage)"
        :status="taskStatusOfStage(stage, i)"
      />

      <div class="text" @click.prevent="onClickStage(stage, i)">
        <span class="text-sm min-w-32 lg:text-xs lg:min-w-fit">{{
          stage.name
        }}</span>
        <span class="text-sm flex-1 ml-4 lg:ml-0">
          <slot name="task-name-of-stage" :stage="stage" :index="i" />
        </span>
      </div>

      <NPopover v-if="!isValidStage(stage, i)" trigger="hover" placement="top">
        <template #trigger>
          <span
            class="ml-2 w-5 h-5 flex justify-center rounded-full select-none bg-error text-white hover:bg-error-hover font-normal absolute right-3"
            @click="onClickStage(stage, i)"
          >
            !
          </span>
        </template>
        <span>Missing SQL statement</span>
      </NPopover>

      <!-- Arrow separator -->
      <div
        v-if="i < issue.pipeline?.stageList.length - 1"
        class="hidden lg:block absolute top-0 bottom-0 right-0 w-5 pointer-events-none"
        aria-hidden="true"
      >
        <svg
          class="h-full w-full text-gray-300"
          viewBox="0 0 22 80"
          fill="none"
          preserveAspectRatio="none"
        >
          <path
            d="M0 -2L20 40L0 82"
            vector-effect="non-scaling-stroke"
            stroke="currentcolor"
            stroke-linejoin="round"
          />
        </svg>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NPopover } from "naive-ui";
import { Issue, Stage, StageCreate } from "@/types";
import TaskStatusIcon from "./TaskStatusIcon.vue";
import { useIssueContext } from "./context";

const {
  create,
  issue,
  selectedStage,
  isValidStage,
  activeTaskOfPipeline,
  activeTaskOfStage,
  taskStatusOfStage,
  selectStageOrTask,
} = useIssueContext();

const isSelectedStage = (stage: Stage | StageCreate): boolean => {
  return stage === selectedStage.value;
};

const isActiveStage = (stage: Stage | StageCreate): boolean => {
  if (create.value) {
    // In create mode we don't have an ActiveStage
    return false;
  }

  const activeTask = activeTaskOfPipeline((issue.value as Issue).pipeline);
  const taskFound = (stage as Stage).taskList.find(
    (t) => t.id === activeTask.id
  );
  if (taskFound) {
    // A stage is "Active" if the ActiveTaskOfPipeline is inside this stage
    return true;
  }

  return false;
};

const stageClass = (stage: Stage | StageCreate, index: number): string[] => {
  const classList: string[] = [];

  if (!isValidStage(stage, index)) classList.push("invalid");
  if (create.value) classList.push("create");
  if (isSelectedStage(stage)) classList.push("selected");
  if (isActiveStage(stage)) classList.push("active");
  const task = activeTaskOfStage(stage as Stage);
  classList.push(`status_${task.status.toLowerCase()}`);

  return classList;
};

const onClickStage = (stage: Stage | StageCreate, index: number) => {
  if (create.value) {
    selectStageOrTask(index);
    return;
  }
  selectStageOrTask((stage as Stage).id);
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
  @apply cursor-pointer ml-4 flex items-center flex-1;
  @apply lg:flex-col lg:items-start;
}
.stage-item.selected .text {
  @apply underline;
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
