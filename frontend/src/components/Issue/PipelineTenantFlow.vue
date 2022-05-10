<template>
  <div class="divide-y">
    <PipelineStageList>
      <template #task-name-of-stage="{ stage }">
        {{ taskNameOfStage(stage) }}
      </template>
    </PipelineStageList>

    <div class="task-list gap-2 p-2 md:grid md:grid-cols-2 lg:grid-cols-4">
      <div
        v-for="(task, j) in taskList"
        :key="j"
        class="task px-2 py-1 cursor-pointer border rounded"
        :class="taskClass(task)"
        @click="onClickTask(task, j)"
      >
        <div class="flex items-center pb-1">
          <TaskStatusIcon
            :create="create"
            :active="isActiveTask(task)"
            :status="task.status"
            class="transform scale-75"
          />
          <heroicons-solid:arrow-narrow-right
            v-if="isActiveTask(task)"
            class="name w-5 h-5"
          />
          <div class="name">{{ j + 1 }} - {{ databaseForTask(task).name }}</div>
        </div>
        <div class="flex items-center px-1 py-1 whitespace-pre-wrap">
          <InstanceEngineIcon :instance="databaseForTask(task).instance" />
          <span
            class="flex-1 ml-2 overflow-x-hidden whitespace-nowrap overflow-ellipsis"
            >{{ instanceName(databaseForTask(task).instance) }}</span
          >
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import {
  Pipeline,
  Stage,
  StageCreate,
  Task,
  TaskCreate,
  Database,
} from "@/types";
import { activeTask, activeTaskInStage, taskSlug } from "@/utils";
import TaskStatusIcon from "./TaskStatusIcon.vue";
import { useDatabaseStore } from "@/store";
import { useIssueContext } from "./context";

const {
  create,
  issue,
  project,
  selectedStage,
  selectedTask,
  selectStageOrTask,
} = useIssueContext();

const pipeline = computed(() => issue.value.pipeline!);

const databaseStore = useDatabaseStore();

watchEffect(() => {
  if (create.value) {
    databaseStore.fetchDatabaseListByProjectId(project.value.id);
  }
});

const taskList = computed(() => selectedStage.value.taskList);

const isSelectedTask = (task: Task | TaskCreate): boolean => {
  return task === selectedTask.value;
};

const isActiveTask = (task: Task | TaskCreate): boolean => {
  if (create.value) return false;
  task = task as Task;
  return activeTask(pipeline.value as Pipeline).id === task.id;
};

const databaseForTask = (task: Task | TaskCreate): Database => {
  if (create.value) {
    return databaseStore.getDatabaseById((task as TaskCreate).databaseId!);
  } else {
    return (task as Task).database!;
  }
};

const selectedStageIdOrIndex = computed(() => {
  if (!create.value) {
    return (selectedStage.value as Stage).id;
  }
  return (pipeline.value.stageList as StageCreate[]).indexOf(
    selectedStage.value as StageCreate
  );
});

const taskNameOfStage = (stage: Stage | StageCreate) => {
  if (create.value) {
    return stage.taskList[0].status;
  }
  const activeTask = activeTaskInStage(stage as Stage);
  const { taskList } = stage as Stage;
  for (let i = 0; i < stage.taskList.length; i++) {
    if (taskList[i].id == activeTask.id) {
      return `${activeTask.name} (${i + 1}/${stage.taskList.length})`;
    }
  }
  return activeTask.name;
};

const taskClass = (task: Task | TaskCreate) => {
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

  selectStageOrTask(stageId, ts);
};
</script>

<style scoped lang="postcss">
.task.selected {
  @apply border-info;
}
.task .name {
  @apply ml-1 overflow-x-hidden whitespace-nowrap overflow-ellipsis;
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
