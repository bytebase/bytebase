<template>
  <div class="w-full">
    <div class="lg:flex lg:items-center divide-y lg:divide-y-0 border-b">
      <div
        v-for="(stage, i) in pipeline.stageList"
        :key="i"
        class="stage-wrapper flex-1 md:flex md:flex-col items-center justify-start"
      >
        <div class="stage-header" :class="stageClass(stage)">
          <span class="pl-4 py-2 flex items-center text-sm font-medium">
            <TaskStatusIcon
              :create="create"
              :active="isActiveStage(stage)"
              :status="taskStatusOfStage(stage)"
            />
            <div
              class="text cursor-pointer hover:underline hidden lg:ml-4 lg:flex lg:flex-col"
              @click.prevent="onClickStage(stage, i)"
            >
              <span class="text-xs">{{ stage.name }}</span>
              <span class="text-sm">{{ taskNameOfStage(stage) }}</span>
            </div>
            <div
              class="text ml-4 cursor-pointer flex items-center space-x-2 lg:hidden"
              @click.prevent="onClickStage(stage, i)"
            >
              <span class="text-sm min-w-32">{{ stage.name }}</span>
              <span class="text-sm flex-1">{{ taskNameOfStage(stage) }}</span>
            </div>
            <div
              class="tooltip-wrapper"
              @click.prevent="onClickStage(stage, i)"
            >
              <span class="tooltip whitespace-nowrap">
                Missing SQL statement
              </span>
              <span
                v-if="!isValidStage(stage)"
                class="ml-2 w-5 h-5 flex justify-center rounded-full select-none bg-error text-white hover:bg-error-hover"
              >
                <span class="text-center font-normal" aria-hidden="true">
                  !
                </span>
              </span>
            </div>
          </span>

          <!-- Arrow separator -->
          <div
            v-if="i < pipeline.stageList.length - 1"
            class="hidden lg:block absolute top-0 right-0 h-full w-5"
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
    </div>

    <!--
      We don't parse the tasks' dependency relationships here, since we have exactly 1
      [sync->cutover->drop-original-table] thread in each stage.

      If we support multi-tenant-gh-ost mode in the future, we may have more than
      one series of [sync->cutover->drop-original-table] threads.
      Then we may repeat the horizon scroller. Each row is a thread of gh-ost migration.
    -->
    <NScrollbar x-scrollable>
      <div class="task-list p-2 md:flex md:items-center relative">
        <template v-for="(task, j) in taskList" :key="j">
          <div
            class="task px-2 py-1 cursor-pointer border rounded md:w-64 flex justify-between items-center"
            :class="taskClass(task)"
            @click="
              onClickTask(selectedStageId, task.name, create ? j + 1 : task.id)
            "
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
            v-if="j < taskList.length - 1"
            class="hidden md:flex items-center justify-center w-4 h-2 overflow-visible relative"
          >
            <!-- show an arrow indicator between tasks -->
            <heroicons-outline:chevron-right class="w-4 h-4" />
          </div>
        </template>
      </div>

      <div class="absolute right-0 top-0 bottom-0 w-10 hidden md:block"></div>
    </NScrollbar>
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, watchEffect } from "vue";
import {
  Pipeline,
  Stage,
  StageCreate,
  PipelineCreate,
  Task,
  TaskCreate,
  Project,
  Database,
} from "@/types";
import { activeTask, activeTaskInStage, taskSlug } from "@/utils";
import { isEmpty } from "lodash-es";
import TaskStatusIcon from "./TaskStatusIcon.vue";
import { useDatabaseStore } from "@/store";
import { NScrollbar } from "naive-ui";
import { BBProgressPie } from "@/bbkit";
import { useI18n } from "vue-i18n";

const props = defineProps({
  create: {
    required: true,
    type: Boolean,
  },
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
  pipeline: {
    required: true,
    type: Object as PropType<Pipeline | PipelineCreate>,
  },
  selectedStage: {
    required: true,
    type: Object as PropType<Stage | StageCreate>,
  },
  selectedTask: {
    type: Object as PropType<Task | TaskCreate | undefined>,
    default: undefined,
  },
});

const emit = defineEmits<{
  (e: "select-task", stageIdOrIndex: number, taskSlug: string): void;
  (e: "select-stage", stageIdOrIndex: number): void;
}>();

const { t } = useI18n();
const databaseStore = useDatabaseStore();

watchEffect(function prepare() {
  if (props.create) {
    databaseStore.fetchDatabaseListByProjectId(props.project.id);
  }
});

const databaseOfTask = (task: Task | TaskCreate): Database => {
  if (props.create) {
    return databaseStore.getDatabaseById((task as TaskCreate).databaseId!);
  }
  return (task as Task).database!;
};

const isSelectedStage = (stage: Stage | StageCreate): boolean => {
  return stage === props.selectedStage;
};

const isSelectedTask = (task: Task | TaskCreate): boolean => {
  return task === props.selectedTask;
};

const isActiveStage = (stage: Stage | StageCreate): boolean => {
  if (props.create) {
    return false;
  }

  const task = activeTaskInStage(stage as Stage);
  if (activeTask(props.pipeline as Pipeline).id === task.id) {
    return true;
  }
  return false;
};

const isActiveTask = (task: Task | TaskCreate): boolean => {
  if (props.create) {
    return false;
  }
  task = task as Task;
  return activeTask(props.pipeline as Pipeline).id === task.id;
};

const taskStatusOfStage = (stage: Stage | StageCreate) => {
  if (props.create) {
    return stage.taskList[0].status;
  }
  const activeTask = activeTaskInStage(stage as Stage);
  return activeTask.status;
};

const taskNameOfStage = (stage: Stage | StageCreate) => {
  if (props.create) {
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

const isValidStage = (stage: Stage | StageCreate) => {
  if (!props.create) {
    return true;
  }
  for (const task of (stage as StageCreate).taskList) {
    if (task.type === "bb.task.database.schema.update.ghost.sync") {
      if (isEmpty(task.statement)) {
        return false;
      }
    }
  }
  return true;
};

const taskList = computed(() => {
  return props.selectedStage.taskList;
});

const selectedStageId = computed(() => {
  if (!props.create) {
    return (props.selectedStage as Stage).id;
  }
  return (props.pipeline.stageList as StageCreate[]).indexOf(
    props.selectedStage as StageCreate
  );
});

const getTaskProgress = (task: Task | TaskCreate) => {
  if (props.create) {
    return 0;
  }
  if (task.type !== "bb.task.database.schema.update.ghost.sync") return 0;
  const taskRun = (task as Task).taskRunList.find((run) => {
    // TODO(Jim): find the correct taskRun which indicates the sync progress.
  });
  const progress = 66.6; // TODO(jim): not implemented yet
  return progress;
};

const stageClass = (stage: Stage | StageCreate): string[] => {
  const classes: string[] = [];
  if (props.create) classes.push("create");
  if (isSelectedStage(stage)) classes.push("selected");
  if (isActiveStage(stage)) classes.push("active");
  const task = activeTaskInStage(stage as Stage);
  classes.push(`status_${task.status.toLowerCase()}`);

  return classes;
};

const taskClass = (task: Task | TaskCreate): string[] => {
  const classes: string[] = [];
  if (isSelectedTask(task)) classes.push("selected");
  if (isActiveTask(task)) classes.push("active");
  if (props.create) classes.push("create");
  classes.push(`status_${task.status.toLowerCase()}`);
  return classes;
};

const onClickStage = (stage: Stage | StageCreate, index: number) => {
  if (props.create) {
    emit("select-stage", index);
    return;
  }
  const { id } = stage as Stage;
  emit("select-stage", id);
};

const onClickTask = (stageId: number, taskName: string, taskId: number) => {
  const ts = taskSlug(taskName, taskId);
  emit("select-task", stageId, ts);
};
</script>

<style scoped lang="postcss">
.stage-header {
  @apply cursor-default flex items-center justify-start w-full relative;
}

.stage-header.selected .text {
  @apply underline;
}
.stage-header.active .text {
  @apply font-bold;
}
.stage-header.status_done .text {
  @apply text-control;
}
.stage-header.status_pending .text,
.stage-header.status_pending_approval .text {
  @apply text-control;
}
.stage-header.active.status_pending .text,
.stage-header.active.status_pending_approval .text {
  @apply text-info;
}
.stage-header.status_running .text {
  @apply text-info;
}
.stage-header.status_failed .text {
  @apply text-red-500;
}

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
