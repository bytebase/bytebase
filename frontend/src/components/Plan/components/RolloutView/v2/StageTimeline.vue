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
        <span v-if="stageTaskRuns.length > 0" class="text-gray-500">
          ({{ stageTaskRuns.length }})
        </span>
      </div>
    </div>

    <!-- Rollback entry -->
    <div
      v-if="rollbackableTaskRuns.length > 0"
      class="px-3 pb-3"
    >
      <NButton
        size="small"
        text
        @click="showRollbackDrawer = true"
      >
        <template #icon>
          <DatabaseBackupIcon :size="16" />
        </template>
        {{ $t("task-run.rollback.available", rollbackableTaskRuns.length) }}
      </NButton>
    </div>

    <!-- Timeline entries: always show in sidebar mode, toggle in inline mode -->
    <div
      v-if="isExpanded"
      class="overflow-y-auto px-3 pt-1 pb-3"
      :class="isInline ? 'max-h-48' : 'max-h-[calc(100vh-300px)]'"
    >
      <div
        v-if="isLoading"
        class="py-4 text-sm text-gray-400 text-center"
      >
        {{ $t("common.loading") }}
      </div>
      <div
        v-else-if="stageTaskRuns.length === 0"
        class="py-4 text-sm text-gray-500 text-center"
      >
        {{ $t("common.no-data") }}
      </div>
      <NTimeline v-else size="medium">
        <NTimelineItem
          v-for="taskRun in sortedTaskRuns"
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
            @click="handleClickTaskRun(taskRun)"
          >
            <div
              class="text-sm leading-4 truncate cursor-pointer hover:underline"
              @click.stop="handleClickTarget(taskRun)"
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
              class="text-xs text-red-600 truncate mt-0.5"
            >
              {{ getErrorPreview(taskRun.detail) }}
            </div>
            <!-- Scheduled run time for pending -->
            <div
              v-if="
                taskRun.status === TaskRun_Status.PENDING && taskRun.runTime
              "
              class="text-xs text-gray-500 mt-0.5 flex items-center gap-1"
            >
{{ $t("task.scheduled-at") }}
              <TimestampDisplay
                :timestamp="taskRun.runTime"
                custom-class="!text-xs !text-gray-500"
              />
            </div>
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
    </div>

    <!-- Rollback drawer -->
    <TaskRunRollbackDrawer
      v-model:show="showRollbackDrawer"
      :rollout="rollout"
      :rollbackable-task-runs="rollbackableTaskRuns"
      @close="showRollbackDrawer = false"
    />
  </div>
</template>

<script lang="ts" setup>
import {
  ChevronDownIcon,
  ChevronRightIcon,
  DatabaseBackupIcon,
} from "lucide-vue-next";
import { NButton, NTimeline, NTimelineItem, NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import TimestampDisplay from "@/components/misc/Timestamp.vue";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import type {
  Rollout,
  Stage,
  TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
} from "@/utils";
import { useResourcePoller } from "../../../logic/poller";
import TaskRunRollbackDrawer from "../TaskRunRollbackDrawer.vue";
import type { RollbackableTaskRun } from "./composables/useRollbackableTasks";
import { useTaskNavigation } from "./composables/useTaskNavigation";
import {
  getErrorPreview,
  getTaskNameFromTaskRun,
  getTaskRunDuration,
  getTimelineType,
  mapTaskRunStatusToTaskStatus,
} from "./composables/useTaskRunUtils";

const props = withDefaults(
  defineProps<{
    stage: Stage | null | undefined;
    taskRuns: TaskRun[];
    rollout: Rollout;
    rollbackableTaskRuns: RollbackableTaskRun[];
    isInline?: boolean;
  }>(),
  {
    isInline: false,
  }
);

const emit = defineEmits<{
  (event: "click-task-run", taskRun: TaskRun): void;
}>();

const isCollapsed = ref(false);
const showRollbackDrawer = ref(false);
const { navigateToTaskDetail } = useTaskNavigation();
const { lastRefreshTime } = useResourcePoller();

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

// Sort by updateTime descending (most recent first)
const sortedTaskRuns = computed(() => {
  return [...stageTaskRuns.value].sort((a, b) => {
    const timeA = a.updateTime?.seconds ?? BigInt(0);
    const timeB = b.updateTime?.seconds ?? BigInt(0);
    return Number(timeB - timeA);
  });
});

const getTaskFromTaskRun = (taskRun: TaskRun) => {
  const taskName = getTaskNameFromTaskRun(taskRun.name);
  return props.stage?.tasks.find((t) => t.name === taskName);
};

const getTaskTargetDisplay = (
  taskRun: TaskRun
): { instance: string; database: string } => {
  const task = getTaskFromTaskRun(taskRun);
  const target = task?.target;

  if (!target) {
    return {
      instance: "",
      database: taskRun.name.split("/").pop() || "unknown",
    };
  }

  return {
    instance: extractInstanceResourceName(target) || "",
    database: extractDatabaseResourceName(target).databaseName || "unknown",
  };
};

const handleClickTarget = (taskRun: TaskRun) => {
  const task = getTaskFromTaskRun(taskRun);
  if (task) {
    navigateToTaskDetail(task);
  }
};

const handleClickTaskRun = (taskRun: TaskRun) => {
  emit("click-task-run", taskRun);
};
</script>
