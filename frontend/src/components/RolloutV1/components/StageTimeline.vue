<template>
  <div>
    <!-- Header -->
    <div
      class="flex items-center justify-between"
      :class="isSidebarMode ? 'px-3 py-2' : 'pb-2'"
    >
      <div class="flex items-center gap-2">
        <span class="text-sm font-medium text-gray-700">
          {{ $t("task.run-history") }}
        </span>
      </div>
    </div>

    <!-- Timeline entries -->
    <div
      class="overflow-y-auto pt-1 pb-3"
      :class="isSidebarMode ? 'px-3 max-h-[calc(100vh-300px)]' : 'px-1'"
    >
      <div v-if="isLoading" class="py-4 flex justify-center">
        <BBSpin />
      </div>
      <div
        v-else-if="stageTaskRuns.length === 0"
        class="py-4 text-sm text-gray-500 flex flex-col justify-center items-center gap-1"
      >
        <SquareChartGanttIcon :stroke-width="1" :size="36" />
        <span>{{ $t("common.no-data") }}</span>
      </div>
      <NTimeline v-else size="medium">
        <NTimelineItem
          v-for="taskRun in displayedTaskRuns"
          :key="taskRun.name"
          :type="getTimelineType(taskRun.status)"
        >
          <template #icon>
            <TaskStatus
              :status="mapTaskRunStatusToTaskStatus(taskRun.status)"
              size="small"
              :disabled="true"
            />
          </template>
          <div
            class="-ml-1 px-1 py-0.5 rounded -mt-0.5"
          >
            <NTooltip :delay="500">
              <template #trigger>
                <div
                  class="text-sm leading-4 truncate cursor-pointer hover:underline"
                  @click="handleClickTarget(taskRun)"
                >
                  <span
                    v-if="getTaskTargetDisplay(taskRun).instance"
                    class="text-gray-500"
                  >
                    {{ getTaskTargetDisplay(taskRun).instance }}
                  </span>
                  <span
                    v-if="getTaskTargetDisplay(taskRun).instance"
                    class="text-gray-400 mx-0.5"
                    >/</span
                  >
                  <span>{{
                    getTaskTargetDisplay(taskRun).database
                  }}</span>
                </div>
              </template>
              <span class="max-w-xs">{{
                getTaskTargetDisplay(taskRun).fullPath
              }}</span>
            </NTooltip>
            <!-- Error preview for failed -->
            <div
              v-if="taskRun.status === TaskRun_Status.FAILED && taskRun.detail"
              class="text-xs text-red-600 line-clamp-3 mt-0.5 cursor-pointer hover:underline"
              @click="showDetail(taskRun)"
            >
              {{ taskRun.detail }}
            </div>
            <!-- Scheduled run time for pending -->
            <ScheduledTimeIndicator
              v-if="taskRun.status === TaskRun_Status.PENDING && taskRun.runTime"
              :time="taskRun.runTime"
              :label="$t('task.scheduled-at')"
              format="datetime"
              class="mt-0.5"
            />
            <!-- Duration and Timestamp -->
            <div class="text-xs text-gray-400 mt-0.5 flex items-center gap-1">
              <NTooltip v-if="getTaskRunDuration(taskRun)">
                <template #trigger>
                  <span>{{ getTaskRunDuration(taskRun) }}</span>
                </template>
                {{ $t("common.duration") }}
              </NTooltip>
              <span v-if="getTaskRunDuration(taskRun)">·</span>
              <TimestampDisplay
                :timestamp="taskRun.updateTime"
                custom-class="!text-xs !text-gray-400"
              />
            </div>
          </div>
        </NTimelineItem>
      </NTimeline>
      <!-- Show message if there are more items -->
      <div
        v-if="sortedTaskRuns.length > MAX_DISPLAY_ITEMS"
        class="text-xs text-gray-400 text-left"
      >
        {{ $t("task.only-showing-latest-n-runs", { n: MAX_DISPLAY_ITEMS }) }}
      </div>
    </div>

  </div>

  <Drawer v-model:show="taskRunDetailContext.show">
    <DrawerContent
      :title="$t('common.detail')"
      style="width: calc(100vw - 14rem)"
    >
      <TaskRunDetail
        v-if="taskRunDetailContext.taskRun"
        :key="taskRunDetailContext.taskRun.name"
        :task-run="taskRunDetailContext.taskRun"
        :database="getDatabaseForTaskRun(taskRunDetailContext.taskRun)"
      />
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { SquareChartGanttIcon } from "lucide-vue-next";
import { NTimeline, NTimelineItem, NTooltip } from "naive-ui";
import { computed, onUnmounted, ref, watch } from "vue";
import BBSpin from "@/bbkit/BBSpin.vue";
import TaskRunDetail from "@/components/IssueV1/components/TaskRunSection/TaskRunDetail.vue";
import TimestampDisplay from "@/components/misc/Timestamp.vue";
import { useResourcePoller } from "@/components/Plan/logic/poller";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useCurrentProjectV1 } from "@/store";
import type { Stage, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  databaseForTask,
  extractDatabaseResourceName,
  extractInstanceResourceName,
} from "@/utils";
import { useTaskNavigation } from "./composables/useTaskNavigation";
import {
  getTaskNameFromTaskRun,
  getTaskRunDuration,
  getTimelineType,
  mapTaskRunStatusToTaskStatus,
} from "./composables/useTaskRunUtils";
import ScheduledTimeIndicator from "./ScheduledTimeIndicator.vue";

const props = withDefaults(
  defineProps<{
    stage: Stage | null | undefined;
    taskRuns: TaskRun[];
    // "sidebar" - sticky sidebar on desktop with padding and max-height
    // "drawer" - inside a drawer with minimal padding for drawer's own spacing
    mode?: "sidebar" | "drawer";
  }>(),
  {
    mode: "sidebar",
  }
);

// Mode-based styling
const isSidebarMode = computed(() => props.mode === "sidebar");

const { project } = useCurrentProjectV1();
const { navigateToTaskDetail } = useTaskNavigation();
const { lastRefreshTime } = useResourcePoller();

// Track stage transitions to prevent flicker
const isStageTransitioning = ref(false);
const lastStageName = ref<string | undefined>(props.stage?.name);

watch(
  () => props.stage?.name,
  (newName) => {
    if (newName !== lastStageName.value) {
      isStageTransitioning.value = true;
      lastStageName.value = newName;
    }
  }
);

// Drawer state for task run detail
const taskRunDetailContext = ref<{
  show: boolean;
  taskRun?: TaskRun;
}>({
  show: false,
});

// Filter task runs for this stage
const stageTaskRuns = computed(() => {
  if (!props.stage) return [];

  const stageTaskNames = new Set(props.stage.tasks.map((t) => t.name));
  return props.taskRuns.filter((run) =>
    stageTaskNames.has(getTaskNameFromTaskRun(run.name))
  );
});

// Check if we're loading (initial load or stage transition)
const isLoading = computed(() => {
  const hasNoRuns = stageTaskRuns.value.length === 0;
  const isInitialLoad = lastRefreshTime.value === 0;
  // Show loading during initial load or stage transition when no runs available
  return hasNoRuns && (isInitialLoad || isStageTransitioning.value);
});

// Clear stage transition flag when we have data for the current stage
watch(
  stageTaskRuns,
  (runs) => {
    if (runs.length > 0) {
      isStageTransitioning.value = false;
    }
  },
  { immediate: true }
);

// Also clear transition flag after a short delay to handle stages with no runs
let transitionTimeoutId: ReturnType<typeof setTimeout> | undefined;

watch(
  () => props.stage?.name,
  () => {
    if (transitionTimeoutId !== undefined) {
      clearTimeout(transitionTimeoutId);
    }
    transitionTimeoutId = setTimeout(() => {
      isStageTransitioning.value = false;
      transitionTimeoutId = undefined;
    }, 300);
  }
);

onUnmounted(() => {
  if (transitionTimeoutId !== undefined) {
    clearTimeout(transitionTimeoutId);
  }
});

// Maximum number of task runs to display in the timeline
const MAX_DISPLAY_ITEMS = 50;

// Sort by updateTime descending (most recent first)
const sortedTaskRuns = computed(() => {
  return [...stageTaskRuns.value].sort((a, b) => {
    const timeA = a.updateTime?.seconds ?? BigInt(0);
    const timeB = b.updateTime?.seconds ?? BigInt(0);
    return Number(timeB - timeA);
  });
});

// Limit displayed items for performance
const displayedTaskRuns = computed(() =>
  sortedTaskRuns.value.slice(0, MAX_DISPLAY_ITEMS)
);

const getTaskFromTaskRun = (taskRun: TaskRun) => {
  const taskName = getTaskNameFromTaskRun(taskRun.name);
  return props.stage?.tasks.find((t) => t.name === taskName);
};

interface TaskTargetDisplay {
  instance: string;
  database: string;
  fullPath: string;
}

// Pre-compute display data for displayed task runs to avoid repeated calls in template
const taskRunDisplayMap = computed(() => {
  const map = new Map<string, TaskTargetDisplay>();
  for (const taskRun of displayedTaskRuns.value) {
    const task = getTaskFromTaskRun(taskRun);
    const target = task?.target;
    const instance = target ? extractInstanceResourceName(target) || "" : "";
    const database = target
      ? extractDatabaseResourceName(target).databaseName || "unknown"
      : taskRun.name.split("/").pop() || "unknown";
    const fullPath = instance ? `${instance} / ${database}` : database;
    map.set(taskRun.name, { instance, database, fullPath });
  }
  return map;
});

const getTaskTargetDisplay = (taskRun: TaskRun): TaskTargetDisplay => {
  return (
    taskRunDisplayMap.value.get(taskRun.name) ?? {
      instance: "",
      database: "unknown",
      fullPath: "unknown",
    }
  );
};

const handleClickTarget = (taskRun: TaskRun) => {
  const task = getTaskFromTaskRun(taskRun);
  if (task) {
    navigateToTaskDetail(task);
  }
};

// Get database for a task run (used by TaskRunDetail in drawer)
const getDatabaseForTaskRun = (taskRun: TaskRun) => {
  const task = getTaskFromTaskRun(taskRun);
  return task ? databaseForTask(project.value, task) : undefined;
};

const showDetail = (taskRun: TaskRun) => {
  taskRunDetailContext.value = {
    show: true,
    taskRun,
  };
};
</script>
