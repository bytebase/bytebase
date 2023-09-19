<template>
  <div class="divide-y relative">
    <PipelineStageList />

    <div class="relative">
      <div
        ref="taskBar"
        class="task-list gap-2 p-2 grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 4xl:grid-cols-6 max-h-48 overflow-y-auto"
        :class="{
          'more-bottom': taskBarScrollState.bottom,
          'more-top': taskBarScrollState.top,
        }"
      >
        <template v-for="(task, i) in taskList" :key="i">
          <div
            class="task relative px-2 py-1 cursor-pointer border rounded lg:flex-1 flex justify-between items-center"
            :class="taskClass(task)"
            :data-task-id="create ? '' : (task as Task).id"
            @click="onClickTask(task, i)"
          >
            <div class="flex-1 overflow-x-hidden">
              <div class="flex items-center pb-1">
                <div class="flex flex-1 items-center gap-x-1 overflow-x-hidden">
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
                    <span>{{ databaseOfTask(task).databaseName }}</span>
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
                !create &&
                task.type === 'bb.task.database.schema.update.ghost.sync'
              "
              :task="(task as Task)"
              unit-key="row"
            />

            <div
              v-if="task.type === 'bb.task.database.schema.update.ghost.sync'"
              class="hidden md:flex items-center justify-center w-4 h-2 overflow-visible absolute -right-[13px]"
            >
              <!-- show an arrow indicator between tasks -->
              <heroicons-outline:chevron-right class="w-4 h-4" />
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useVerticalScrollState } from "@/composables/useScrollState";
import { useDatabaseV1Store } from "@/store";
import type { Pipeline, Stage, StageCreate, Task, TaskCreate } from "@/types";
import { activeTask, taskSlug } from "@/utils";
import PipelineStageList from "./PipelineStageList.vue";
import { TaskExtraActionsButton } from "./StatusTransitionButtonGroup";
import TaskProgressPie from "./TaskProgressPie.vue";
import TaskStatusIcon from "./TaskStatusIcon.vue";
import { useIssueLogic } from "./logic";

const { t } = useI18n();
const databaseStore = useDatabaseV1Store();

const { create, issue, selectedStage, selectedTask, selectStageOrTask } =
  useIssueLogic();
const taskBar = ref<HTMLDivElement>();
const taskBarScrollState = useVerticalScrollState(taskBar, 192);

const pipeline = computed(() => issue.value.pipeline!);

const taskList = computed(() => {
  return selectedStage.value.taskList;
});

const databaseOfTask = (task: Task | TaskCreate) => {
  const uid = create.value
    ? String((task as TaskCreate).databaseId!)
    : String((task as Task).database!.id);
  return databaseStore.getDatabaseByUID(uid);
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

.task-list::before {
  @apply absolute top-0 h-4 w-full -ml-2 z-10 pointer-events-none transition-shadow;
  content: "";
  box-shadow: none;
}
.task-list::after {
  @apply absolute bottom-0 h-4 w-full -ml-2 z-10 pointer-events-none transition-shadow;
  content: "";
  box-shadow: none;
}
.task-list.more-top::before {
  box-shadow: inset 0 0.5rem 0.25rem -0.25rem rgb(0 0 0 / 10%);
}
.task-list.more-bottom::after {
  box-shadow: inset 0 -0.5rem 0.25rem -0.25rem rgb(0 0 0 / 10%);
}
</style>
