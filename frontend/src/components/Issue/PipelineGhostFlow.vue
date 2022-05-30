<template>
  <div class="divide-y">
    <PipelineStageList>
      <template #task-name-of-stage="{ stage }">
        {{ taskNameOfStage(stage) }}
      </template>
    </PipelineStageList>

    <div
      class="task-list p-2 lg:flex lg:items-center relative space-y-2 lg:space-y-0"
    >
      <template v-for="(task, i) in taskList" :key="i">
        <div
          class="task px-2 py-1 cursor-pointer border rounded lg:flex-1 flex justify-between items-center"
          :class="taskClass(task)"
          @click="onClickTask(task, i)"
        >
          <div class="flex-1">
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
              <div class="name">{{ databaseOfTask(task).name }}</div>
            </div>
            <div
              class="flex items-center px-1 py-1 whitespace-pre-wrap break-all"
            >
              {{ taskNameOfTask(task) }}
            </div>
          </div>
          <div v-if="getTaskProgress(task) > 0">
            <BBProgressPie
              class="w-9 h-9 text-info"
              :thickness="2"
              :percent="getTaskProgress(task)"
            >
              <template #default="{ percent }">
                <span class="text-xs scale-90">{{ percent }}%</span>
              </template>
            </BBProgressPie>
          </div>
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

    <div class="absolute right-0 top-0 bottom-0 w-10 hidden md:block"></div>
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import type {
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
import { BBProgressPie } from "@/bbkit";
import PipelineStageList from "./PipelineStageList.vue";
import { useIssueLogic } from "./logic";

const { t } = useI18n();
const databaseStore = useDatabaseStore();

const {
  create,
  issue,
  project,
  selectedStage,
  selectedTask,
  selectStageOrTask,
} = useIssueLogic();

const pipeline = computed(() => issue.value.pipeline!);

watchEffect(() => {
  if (create.value) {
    databaseStore.fetchDatabaseListByProjectId(project.value.id);
  }
});

const taskList = computed(() => {
  return selectedStage.value.taskList;
});

const databaseOfTask = (task: Task | TaskCreate): Database => {
  if (create.value) {
    return databaseStore.getDatabaseById((task as TaskCreate).databaseId!);
  }
  return (task as Task).database!;
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

const taskNameOfTask = (task: Task | TaskCreate) => {
  return t(`task.type.${task.type.replace(/\./g, "-")}`);
};

const selectedStageIdOrIndex = computed(() => {
  if (!create.value) {
    return (selectedStage.value as Stage).id;
  }
  return (pipeline.value.stageList as StageCreate[]).indexOf(
    selectedStage.value as StageCreate
  );
});

const getTaskProgress = (task: Task | TaskCreate) => {
  if (create.value) {
    return 0;
  }
  if (task.type !== "bb.task.database.schema.update.ghost.sync") return 0;
  const taskRun = (task as Task).taskRunList.find((run) => {
    // TODO(Jim): find the correct taskRun which indicates the sync progress.
    return false;
  });
  if (taskRun) {
    // TODO(Jim): get progress from taskRun.result.detail
  }

  // nothing found
  return 0;
};

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
