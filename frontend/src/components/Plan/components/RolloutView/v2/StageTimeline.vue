<template>
  <div>
    <!-- Header -->
    <div
      class="flex items-center justify-between px-3 py-2"
      @click="isInline && (isCollapsed = !isCollapsed)"
    >
      <div class="flex items-center gap-2">
        <!-- Only show toggle icon in inline mode -->
        <component
          :is="isCollapsed ? ChevronRightIcon : ChevronDownIcon"
          v-if="isInline"
          class="w-4 h-4 text-gray-500"
        />
        <span class="text-sm font-medium text-gray-700">
          {{ $t("task.run-history") }}
        </span>
      </div>
    </div>

    <!-- Rollback entry -->
    <StageTaskRunsRollback :stage="stage" />

    <!-- Timeline entries: always show in sidebar mode, toggle in inline mode -->
    <div
      v-if="isExpanded"
      class="overflow-y-auto px-3 pt-1 pb-3"
      :class="isInline ? 'max-h-48' : 'max-h-[calc(100vh-300px)]'"
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
import {
  ChevronDownIcon,
  ChevronRightIcon,
  SquareChartGanttIcon,
} from "lucide-vue-next";
import { NTimeline, NTimelineItem, NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import BBSpin from "@/bbkit/BBSpin.vue";
import TaskRunDetail from "@/components/IssueV1/components/TaskRunSection/TaskRunDetail.vue";
import TimestampDisplay from "@/components/misc/Timestamp.vue";
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
import { useResourcePoller } from "../../../logic/poller";
import { useTaskNavigation } from "./composables/useTaskNavigation";
import {
  getTaskNameFromTaskRun,
  getTaskRunDuration,
  getTimelineType,
  mapTaskRunStatusToTaskStatus,
} from "./composables/useTaskRunUtils";
import ScheduledTimeIndicator from "./ScheduledTimeIndicator.vue";
import StageTaskRunsRollback from "./StageTaskRunsRollback.vue";

const props = withDefaults(
  defineProps<{
    stage: Stage | null | undefined;
    taskRuns: TaskRun[];
    isInline?: boolean;
  }>(),
  {
    isInline: false,
  }
);

const { project } = useCurrentProjectV1();
const isCollapsed = ref(false);
const { navigateToTaskDetail } = useTaskNavigation();
const { lastRefreshTime } = useResourcePoller();

// Drawer state for task run detail
const taskRunDetailContext = ref<{
  show: boolean;
  taskRun?: TaskRun;
}>({
  show: false,
});

// In sidebar mode (not inline), always show expanded
const isExpanded = computed(() => !props.isInline || !isCollapsed.value);

// Filter task runs for this stage
const stageTaskRuns = computed(() => {
  if (!props.stage) return [];

  const stageTaskNames = new Set(props.stage.tasks.map((t) => t.name));
  return props.taskRuns.filter((run) =>
    stageTaskNames.has(getTaskNameFromTaskRun(run.name))
  );
});

// Check if we're still in initial loading state (never refreshed yet)
const isLoading = computed(
  () => lastRefreshTime.value === 0 && stageTaskRuns.value.length === 0
);

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

// Pre-compute display data for displayed task runs to avoid repeated calls in template
const taskRunDisplayMap = computed(() => {
  const map = new Map<string, { instance: string; database: string }>();
  for (const taskRun of displayedTaskRuns.value) {
    const task = getTaskFromTaskRun(taskRun);
    const target = task?.target;
    if (!target) {
      map.set(taskRun.name, {
        instance: "",
        database: taskRun.name.split("/").pop() || "unknown",
      });
    } else {
      map.set(taskRun.name, {
        instance: extractInstanceResourceName(target) || "",
        database: extractDatabaseResourceName(target).databaseName || "unknown",
      });
    }
  }
  return map;
});

const getTaskTargetDisplay = (taskRun: TaskRun) => {
  return (
    taskRunDisplayMap.value.get(taskRun.name) ?? {
      instance: "",
      database: "unknown",
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
