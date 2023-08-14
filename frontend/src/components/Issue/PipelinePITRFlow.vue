<template>
  <div class="divide-y">
    <PipelineStageList />

    <div
      v-if="taskList.length > 1"
      class="task-list p-2 lg:flex lg:items-center relative space-y-2 lg:space-y-0"
    >
      <template v-for="(task, i) in taskList" :key="i">
        <div
          class="task px-2 py-1 cursor-pointer border rounded lg:flex-1 flex justify-between items-center overflow-x-hidden"
          :class="taskClass(task)"
          :data-task-id="create ? '' : (task as Task).id"
          @click="onClickTask(task, i)"
        >
          <div class="flex-1">
            <div class="flex items-center pb-1">
              <div class="flex flex-1 items-center gap-x-1">
                <TaskStatusIcon
                  :create="create"
                  :active="isActiveTask(task)"
                  :status="task.status"
                  :task="task"
                  class="transform scale-75"
                />
                <div class="name flex-1 space-x-1 overflow-x-hidden">
                  <heroicons-solid:arrow-narrow-right
                    v-if="isActiveTask(task)"
                    class="w-5 h-5 inline-block"
                  />
                  <span>{{ databaseNameOfTask(task) }}</span>
                </div>
              </div>
              <TaskExtraActionsButton :task="(task as Task)" />
            </div>

            <div class="flex items-center justify-between px-1 py-1">
              <div
                class="flex flex-1 items-center whitespace-pre-wrap break-all"
              >
                {{ taskNameOfTask(task) }}
              </div>
            </div>
          </div>

          <TaskProgressPie
            v-if="
              !create && task.type === 'bb.task.database.restore.pitr.restore'
            "
            :task="(task as Task)"
          >
            <template #unit="{ unit }">{{ bytesToString(unit) }}</template>
          </TaskProgressPie>
        </div>

        <div
          v-if="i < taskList.length - 1"
          class="hidden lg:flex items-center justify-center w-4 h-2 overflow-visible relative"
        >
          <!-- show an arrow indicator between tasks -->
          <heroicons-outline:chevron-right class="w-4 h-4" />
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { Pipeline, Stage, StageCreate, Task, TaskCreate } from "@/types";
import {
  activeTask,
  extractDatabaseNameFromTask,
  taskSlug,
  bytesToString,
} from "@/utils";
import PipelineStageList from "./PipelineStageList.vue";
import { TaskExtraActionsButton } from "./StatusTransitionButtonGroup";
import TaskStatusIcon from "./TaskStatusIcon.vue";
import { useIssueLogic } from "./logic";

const { create, issue, selectedStage, selectedTask, selectStageOrTask } =
  useIssueLogic();

const pipeline = computed(() => issue.value.pipeline!);

const taskList = computed(() => {
  return selectedStage.value.taskList;
});

const databaseNameOfTask = (task: Task | TaskCreate): string => {
  return extractDatabaseNameFromTask(task);
};

const isSelectedTask = (task: Task | TaskCreate): boolean => {
  return task === selectedTask.value;
};

const isActiveTask = (task: Task | TaskCreate): boolean => {
  if (create.value) {
    return false;
  }
  task = task as Task;
  return activeTask(pipeline.value as Pipeline).id === task.id;
};

const taskNameOfTask = (task: Task | TaskCreate) => {
  // return t(`task.type.${task.type.replace(/\./g, "-")}`);
  return task.name;
};

const selectedStageIdOrIndex = computed(() => {
  if (!create.value) {
    return (selectedStage.value as Stage).id;
  }
  return (pipeline.value.stageList as StageCreate[]).indexOf(
    selectedStage.value as StageCreate
  );
});

const taskClass = (task: Task | TaskCreate): string[] => {
  const classes: string[] = [];
  if (isSelectedTask(task)) classes.push("selected");
  if (isActiveTask(task)) classes.push("active");
  if (create.value) classes.push("create");
  classes.push(`status_${task.status.toLowerCase()}`);
  return classes;
};

const onClickTask = (task: Task | TaskCreate, index: number) => {
  const stageId = selectedStageIdOrIndex.value;
  const taskName = task.name;
  const taskId = create.value ? index + 1 : (task as Task).id;
  const ts = taskSlug(taskName, taskId);

  selectStageOrTask(Number(stageId), ts);
};
</script>

<style scoped lang="postcss">
.task.selected {
  @apply border-info;
}
.task .name {
  @apply whitespace-pre-wrap break-all;
}
.task.active .name {
  @apply font-bold;
}
.task.status_done .name {
  @apply text-control;
}
.task.status_pending .name,
.task.status_pending_approval .name {
  @apply text-control;
}
.task.active.status_pending .name,
.task.active.status_pending_approval .name {
  @apply text-info;
}
.task.status_running .name {
  @apply text-info;
}
.task.status_failed .name {
  @apply text-red-500;
}
</style>
