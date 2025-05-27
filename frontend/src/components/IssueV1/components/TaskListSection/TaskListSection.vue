<template>
  <div class="relative">
    <div
      v-if="shouldShowTaskFilter"
      class="w-full sticky top-0 z-10 bg-white px-4 pt-2 pb-1"
    >
      <TaskFilter
        :disabled="stageState.isRequesting || !stageState.initialized"
        :task-status-list="stageState.taskStatusFilters"
        :advice-status-list="stageState.adviceStatusFilters"
        @update:advice-status-list="
          (adviceStatusFilters) => updateStageState({ adviceStatusFilters })
        "
        @update:task-status-list="
          (taskStatusFilters) => updateStageState({ taskStatusFilters })
        "
      />
    </div>
    <div class="relative w-full">
      <div
        v-if="!stageState.initialized && stageState.isRequesting"
        class="w-full flex items-center justify-center py-8"
      >
        <BBSpin />
      </div>
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
          v-for="(task, i) in filteredTaskList.slice(0, stageState.index)"
          :key="i"
          :task="task"
        />
        <div
          v-if="filteredTaskList.length > stageState.index"
          class="col-span-full flex flex-row items-center justify-end"
        >
          <NButton
            size="small"
            quaternary
            :loading="stageState.isRequesting"
            @click="loadNextPage"
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
import { BBSpin } from "@/bbkit";
import { usePlanSQLCheckContext } from "@/components/Plan/components/SQLCheckSection/context";
import { databaseForTask } from "@/components/Rollout/RolloutDetail";
import { useVerticalScrollState } from "@/composables/useScrollState";
import { batchGetOrFetchDatabases } from "@/store";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import type { Task_Status } from "@/types/proto/v1/rollout_service";
import type { Advice_Status } from "@/types/proto/v1/sql_service";
import { isDev } from "@/utils";
import { useIssueContext, projectOfIssue } from "../../logic";
import CurrentTaskSection from "./CurrentTaskSection.vue";
import TaskCard from "./TaskCard.vue";
import TaskFilter from "./TaskFilter.vue";
import { filterTask } from "./filter";

interface StageState {
  // Index is the current number of tasks to show.
  index: number;
  initialized: boolean;
  taskStatusFilters: Task_Status[];
  adviceStatusFilters: Advice_Status[];
  isRequesting: boolean;
}

interface LocalState {
  pageStatePerStage: Map<string, StageState>;
}

const MAX_LIST_HEIGHT = 256;

// The default number of tasks to show per page.
// This is set to 4 in development mode for easier testing.
const TASK_PER_PAGE = isDev() ? 4 : 20;

const state = reactive<LocalState>({
  pageStatePerStage: new Map<string, StageState>(),
});

const issueContext = useIssueContext();
const { selectedStage, issue, selectedTask } = issueContext;
const { resultMap } = usePlanSQLCheckContext();
const taskBar = ref<HTMLDivElement>();
const taskBarScrollState = useVerticalScrollState(taskBar, MAX_LIST_HEIGHT);

const taskList = computed(() => {
  return selectedStage.value.tasks;
});

const stageState = computed(
  () =>
    state.pageStatePerStage.get(selectedStage.value.name) ?? {
      index: 0,
      initialized: false,
      taskStatusFilters: [],
      adviceStatusFilters: [],
      isRequesting: false,
    }
);

const updateStageState = (patch: Partial<StageState>) => {
  state.pageStatePerStage.set(selectedStage.value.name, {
    ...stageState.value,
    ...patch,
  });
};

const filteredTaskList = computed(() => {
  return taskList.value.filter((task) => {
    if (stageState.value.taskStatusFilters.length > 0) {
      if (!stageState.value.taskStatusFilters.includes(task.status)) {
        return false;
      }
    }
    if (stageState.value.adviceStatusFilters.length > 0) {
      if (
        !stageState.value.adviceStatusFilters.some((status) =>
          filterTask(issueContext, resultMap.value, task, {
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
  const fromIndex = stageState.value.index;
  const toIndex = fromIndex + TASK_PER_PAGE;

  const databaseNames = filteredTaskList.value
    .slice(fromIndex, toIndex)
    .map((task) => databaseForTask(projectOfIssue(issue.value), task).name);

  try {
    await batchGetOrFetchDatabases(databaseNames);
  } catch {
    // Ignore errors
    // If the issue type is create database,
    // the API will throw error cause it cannot found the pending created database.
  } finally {
    updateStageState({
      index: toIndex,
      initialized: true,
    });
  }
}, DEBOUNCE_SEARCH_DELAY);

const loadNextPage = async () => {
  if (stageState.value.isRequesting) {
    return;
  }
  updateStageState({
    isRequesting: true,
  });
  try {
    await loadMore();
  } catch {
    // Ignore errors
  } finally {
    updateStageState({
      isRequesting: false,
    });
  }
};

watch(
  [
    () => stageState.value.taskStatusFilters,
    () => stageState.value.adviceStatusFilters,
  ],
  async () => {
    // Reset the index when the filters change.
    if (!stageState.value.isRequesting && stageState.value.initialized) {
      updateStageState({
        index: 0,
      });
      await loadNextPage();
    }
  }
);

watch(
  () => selectedStage.value.name,
  async () => {
    if (!stageState.value.initialized) {
      await loadNextPage();
    }
  },
  { immediate: true }
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
