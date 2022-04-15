<template>
  <div class="w-full">
    <div class="lg:flex divide-y lg:divide-y-0 border-b">
      <div
        v-for="(item, i) in itemList"
        :key="i"
        class="stage-wrapper flex-1 md:flex md:flex-col items-center justify-start"
      >
        <div class="stage-header" :class="stageClass(item.stage)">
          <span class="pl-4 py-2 flex items-center text-sm font-medium">
            <TaskStatusIcon
              :create="create"
              :active="isActiveStage(item.stage)"
              :status="item.taskStatus"
            />
            <div
              class="text cursor-pointer hover:underline hidden lg:ml-4 lg:flex lg:flex-col"
              @click.prevent="clickItem(item)"
            >
              <span class="text-xs">{{ item.stageName }}</span>
              <span class="text-sm">{{ item.taskName }}</span>
            </div>
            <div
              class="text ml-4 cursor-pointer flex items-center space-x-2 lg:hidden"
              @click.prevent="clickItem(item)"
            >
              <span class="text-sm min-w-32">{{ item.stageName }}</span>
              <span class="text-sm flex-1">{{ item.taskName }}</span>
            </div>
            <div class="tooltip-wrapper" @click.prevent="clickItem(item)">
              <span class="tooltip whitespace-nowrap"
                >Missing SQL statement</span
              >
              <span
                v-if="!item.valid"
                class="ml-2 w-5 h-5 flex justify-center rounded-full select-none bg-error text-white hover:bg-error-hover"
              >
                <span class="text-center font-normal" aria-hidden="true"
                  >!</span
                >
              </span>
            </div>
          </span>

          <!-- Arrow separator -->
          <div
            v-if="i < itemList.length - 1"
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

    <div class="task-list gap-2 p-2 md:grid md:grid-cols-2 lg:grid-cols-4">
      <div
        v-for="(task, j) in taskList"
        :key="j"
        class="task px-2 py-1 cursor-pointer border rounded"
        :class="taskClass(task)"
        @click="clickTask(selectedStageId, task.name, create ? j + 1 : task.id)"
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

<script lang="ts">
import { computed, defineComponent, PropType, watchEffect } from "vue";
import {
  Pipeline,
  TaskStatus,
  TaskId,
  Stage,
  StageId,
  StageCreate,
  PipelineCreate,
  Task,
  TaskCreate,
  Project,
  Database,
} from "../../types";
import { activeTask, activeTaskInStage, taskSlug } from "../../utils";
import { isEmpty } from "lodash-es";
import TaskStatusIcon from "./TaskStatusIcon.vue";
import { useDatabaseStore } from "@/store";

interface FlowItem {
  stageId: StageId;
  stageName: string;
  taskId: TaskId;
  taskName: string;
  taskStatus: TaskStatus;
  valid: boolean;
  stage: Stage | StageCreate;
  taskList: (Task | TaskCreate)[];
}

export default defineComponent({
  name: "PipelineTenantFlow",
  components: {
    TaskStatusIcon,
  },
  props: {
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
  },
  emits: ["select-stage-id", "select-task"],
  setup(props, { emit }) {
    const databaseStore = useDatabaseStore();

    watchEffect(function prepare() {
      if (props.create) {
        databaseStore.fetchDatabaseListByProjectId(props.project.id);
      }
    });

    const databaseForTask = (task: Task | TaskCreate): Database => {
      if (props.create) {
        return databaseStore.getDatabaseById((task as TaskCreate).databaseId!);
      } else {
        return (task as Task).database!;
      }
    };

    const isSelectedStage = (stage: Stage | StageCreate): boolean => {
      return stage === props.selectedStage;
    };

    const isSelectedTask = (task: Task | TaskCreate): boolean => {
      return task === props.selectedTask;
    };

    const isActiveStage = (stage: Stage | StageCreate): boolean => {
      if (props.create) return false;

      const task = activeTaskInStage(stage as Stage);
      if (activeTask(props.pipeline as Pipeline).id === task.id) {
        return true;
      }
      return false;
    };

    const isActiveTask = (task: Task | TaskCreate): boolean => {
      if (props.create) return false;
      task = task as Task;
      return activeTask(props.pipeline as Pipeline).id === task.id;
    };

    const itemList = computed<FlowItem[]>(() => {
      return props.pipeline.stageList.map((stage, index) => {
        let activeTask = stage.taskList[0];
        let taskName = activeTask.name;
        let valid = true;
        if (props.create) {
          for (const task of stage.taskList) {
            if (
              task.type == "bb.task.database.create" ||
              task.type == "bb.task.database.schema.update" ||
              task.type == "bb.task.database.data.update"
            ) {
              if (isEmpty((task as TaskCreate).statement)) {
                valid = false;
                break;
              }
            }
          }
        } else {
          activeTask = activeTaskInStage(stage as Stage);
          if (stage.taskList.length > 1) {
            for (let i = 0; i < stage.taskList.length; i++) {
              if ((stage.taskList[i] as Task).id == (activeTask as Task).id) {
                taskName = `${activeTask.name} (${i + 1}/${
                  stage.taskList.length
                })`;
                break;
              }
            }
          }
        }

        return {
          stageId: props.create ? index : (stage as Stage).id,
          stageName: stage.name,
          taskId: props.create ? index : (activeTask as Task).id,
          taskName: taskName,
          taskStatus: activeTask.status,
          stage,
          taskList: stage.taskList,
          valid,
        };
      });
    });

    const taskList = computed(() => {
      return props.selectedStage.taskList;
    });

    const selectedStageId = computed(() => {
      if (!props.create) {
        return (props.selectedStage as Stage).id;
      } else {
        return props.pipeline.stageList.indexOf(props.selectedStage as any);
      }
    });

    const stageClass = (stage: Stage | StageCreate): string[] => {
      const classes: string[] = [];
      if (props.create) classes.push("create");
      if (isSelectedStage(stage)) classes.push("selected");
      if (isActiveStage(stage)) classes.push("active");
      const task = activeTaskInStage(stage as Stage);
      classes.push(`status_${task.status.toLowerCase()}`);

      return classes;
    };

    const taskClass = (task: Task | TaskCreate) => {
      const classes: string[] = [];
      if (isSelectedTask(task)) classes.push("selected");
      if (isActiveTask(task)) classes.push("active");
      if (props.create) classes.push("create");
      classes.push(`status_${task.status.toLowerCase()}`);
      return classes;
    };

    const clickItem = (item: FlowItem) => {
      emit("select-stage-id", item.stageId);
    };

    const clickTask = (stageId: number, taskName: string, taskId: number) => {
      const ts = taskSlug(taskName, taskId);
      emit("select-task", stageId, ts);
    };

    return {
      isSelectedStage,
      isSelectedTask,
      isActiveStage,
      isActiveTask,
      itemList,
      taskList,
      selectedStageId,
      activeTask,
      stageClass,
      taskClass,
      clickItem,
      clickTask,
      databaseForTask,
    };
  },
});
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
