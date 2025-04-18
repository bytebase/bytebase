<template>
  <div class="relative">
    <div
      v-if="shouldShowTaskFilter"
      class="w-full sticky top-0 z-10 bg-white px-4 pt-2 pb-1"
    >
      <TaskFilter
        v-model:task-status-list="state.taskStatusFilters"
        v-model:advice-status-list="state.adviceStatusFilters"
      />
    </div>
    <div class="relative w-full">
      <div
        ref="taskBar"
        class="task-list gap-2 px-4 py-2 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 3xl:grid-cols-5 4xl:grid-cols-6 overflow-y-auto"
        :class="{
          'more-bottom': taskBarScrollState.bottom,
          'more-top': taskBarScrollState.top,
        }"
        :style="{
          'max-height': `${MAX_LIST_HEIGHT}px`,
        }"
      >
        <TaskCard
          v-for="(task, i) in filteredTaskList.slice(0, state.index)"
          :key="i"
          :task="task"
        />
        <div
          v-if="filteredTaskList.length > state.index"
          class="col-span-full flex flex-row items-center justify-end"
        >
          <NButton
            size="small"
            quaternary
            :loading="isRequesting"
            @click="state.index += TASK_PER_PAGE"
          >
            {{ $t("common.load-more") }}
          </NButton>
        </div>
      </div>
    </div>
  </div>

  <CurrentTaskSection v-if="shouldShowCurrentTaskView" />
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { NButton } from "naive-ui";
import { computed, ref, reactive, watch } from "vue";
import { useVerticalScrollState } from "@/composables/useScrollState";
import { batchGetOrFetchDatabases } from "@/store";
import type { Task_Status } from "@/types/proto/v1/rollout_service";
import type { Advice_Status } from "@/types/proto/v1/sql_service";
import { isDev } from "@/utils";
import { useIssueContext, databaseForTask } from "../../logic";
import { useIssueSQLCheckContext } from "../SQLCheckSection/context";
import CurrentTaskSection from "./CurrentTaskSection.vue";
import TaskCard from "./TaskCard.vue";
import TaskFilter from "./TaskFilter.vue";
import { filterTask } from "./filter";

interface LocalState {
  // index is the number of tasks to show.
  // It is initialized to TASK_PER_PAGE.
  index: number;
  taskStatusFilters: Task_Status[];
  adviceStatusFilters: Advice_Status[];
}

const MAX_LIST_HEIGHT = 256;

// The default number of tasks to show per page.
// This is set to 4 in development mode for easier testing.
const TASK_PER_PAGE = isDev() ? 4 : 20;

const state = reactive<LocalState>({
  index: TASK_PER_PAGE,
  taskStatusFilters: [],
  adviceStatusFilters: [],
});
const isRequesting = ref(false);

const issueContext = useIssueContext();
const { selectedStage, issue, selectedTask } = issueContext;
const sqlCheckContext = useIssueSQLCheckContext();
const taskBar = ref<HTMLDivElement>();
const taskBarScrollState = useVerticalScrollState(taskBar, MAX_LIST_HEIGHT);

const taskList = computed(() => {
  return selectedStage.value.tasks;
});

const filteredTaskList = computed(() => {
  return taskList.value.filter((task) => {
    if (state.taskStatusFilters.length > 0) {
      if (!state.taskStatusFilters.includes(task.status)) {
        return false;
      }
    }
    if (state.adviceStatusFilters.length > 0) {
      if (
        !state.adviceStatusFilters.some((status) =>
          filterTask(issueContext, sqlCheckContext, task, {
            adviceStatus: status,
          })
        )
      ) {
        return false;
      }
    }
    return true;
  });
});

const shouldShowTaskFilter = computed(() => {
  return taskList.value.length > 10;
});

const shouldShowCurrentTaskView = computed(() => {
  // Only show the current task view when the selected task is not in the filtered task list.
  return !filteredTaskList.value.some(
    (task) => task.name === selectedTask.value.name
  );
});

const loadMore = useDebounceFn(async () => {
  if (state.index >= filteredTaskList.value.length) {
    return;
  }
  const databaseNames = filteredTaskList.value
    .slice(0, state.index)
    .map((task) => databaseForTask(issue.value, task).name);
  await batchGetOrFetchDatabases(databaseNames);
}, 500);

watch(
  () => filteredTaskList.value,
  () => (state.index = TASK_PER_PAGE)
);

watch(
  () => state.index,
  async () => {
    isRequesting.value = true;
    try {
      await loadMore();
    } catch {
      // Ignore errors
    }
    isRequesting.value = false;
  },
  { immediate: true }
);

watch(
  () => selectedStage.value.name,
  () => {
    // Clear the index when the stage changes.
    state.index = TASK_PER_PAGE;
    state.taskStatusFilters = [];
    state.adviceStatusFilters = [];
  }
);
</script>

<style scoped lang="postcss">
.task-list::before {
  @apply absolute top-0 h-4 w-full -ml-4 z-10 pointer-events-none transition-shadow;
  content: "";
  box-shadow: none;
}
.task-list::after {
  @apply absolute bottom-0 h-4 w-full -ml-4 z-10 pointer-events-none transition-shadow;
  content: "";
  box-shadow: none;
}
.task-list.more-top::before {
  box-shadow: inset 0 0.3rem 0.25rem -0.25rem rgb(0 0 0 / 10%);
}
.task-list.more-bottom::after {
  box-shadow: inset 0 -0.3rem 0.25rem -0.25rem rgb(0 0 0 / 10%);
}
</style>
