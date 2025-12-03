<template>
  <div class="space-y-2">
    <!-- Line 1: Title + Status + Timestamp + Metadata -->
    <div class="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
      <!-- Title -->
      <span class="font-medium text-gray-700 shrink-0">{{
        t("task-run.latest")
      }}</span>
      <span class="text-gray-300">·</span>
      <!-- Status badge -->
      <span class="flex items-center gap-x-1 shrink-0" :class="statusClass">
        <component
          :is="statusIcon"
          class="w-4 h-4"
          :class="{ 'animate-spin': taskStatus === Task_Status.RUNNING }"
        />
        <span>{{ statusText }}</span>
      </span>
      <span class="text-gray-300">·</span>
      <!-- Timestamp -->
      <span class="text-gray-500 shrink-0">
        <Timestamp :timestamp="taskRun.updateTime" />
      </span>
      <!-- Metadata items -->
      <template v-if="executorEmail">
        <span class="text-gray-300">·</span>
        <NTooltip>
          <template #trigger>
            <span class="flex items-center gap-x-1 text-gray-500 min-w-0 cursor-default">
              <UserIcon class="w-3.5 h-3.5 shrink-0" />
              <span class="truncate">{{ executorEmail }}</span>
            </span>
          </template>
          {{ t("task.executed-by") }}
        </NTooltip>
      </template>
      <template v-if="duration">
        <span class="text-gray-300">·</span>
        <NTooltip>
          <template #trigger>
            <span class="flex items-center gap-x-1 text-gray-500 shrink-0 cursor-default">
              <ClockIcon class="w-3.5 h-3.5" />
              {{ duration }}
            </span>
          </template>
          {{ t("common.duration") }}
        </NTooltip>
      </template>
      <template v-if="affectedRowsDisplay">
        <span class="text-gray-300">·</span>
        <NTooltip>
          <template #trigger>
            <span class="flex items-center gap-x-1 text-gray-500 shrink-0 cursor-default">
              <RowsIcon class="w-3.5 h-3.5" />
              {{ affectedRowsDisplay }}
            </span>
          </template>
          {{ t("task.affected-rows") }}
        </NTooltip>
      </template>
    </div>

    <!-- Line 2: Task run logs -->
    <TaskRunLogEntries
      v-if="summary.latestEntries.length > 0"
      :entries="summary.latestEntries"
      :sheet="sheet"
      class="pt-1"
    />
  </div>
</template>

<script lang="ts" setup>
import {
  CheckCircle2Icon,
  CircleDotIcon,
  ClockIcon,
  LoaderCircleIcon,
  Rows3Icon as RowsIcon,
  UserIcon,
  XCircleIcon,
} from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import type { Component } from "vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import Timestamp from "@/components/misc/Timestamp.vue";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { useTaskRunLogSummary } from "./composables/useTaskRunLogSummary";
import TaskRunLogEntries from "./TaskRunLogEntries.vue";

// Types
interface StatusConfig {
  icon: Component;
  class: string;
  textKey: string;
}

// Default status config
const DEFAULT_STATUS_CONFIG: StatusConfig = {
  icon: CircleDotIcon,
  class: "text-gray-500",
  textKey: "common.unknown",
};

// Constants
const STATUS_CONFIG: Partial<Record<Task_Status, StatusConfig>> = {
  [Task_Status.FAILED]: {
    icon: XCircleIcon,
    class: "text-red-600",
    textKey: "common.failed",
  },
  [Task_Status.RUNNING]: {
    icon: LoaderCircleIcon,
    class: "text-blue-600",
    textKey: "common.running",
  },
  [Task_Status.DONE]: {
    icon: CheckCircle2Icon,
    class: "text-green-600",
    textKey: "common.success",
  },
  [Task_Status.PENDING]: {
    icon: CircleDotIcon,
    class: "text-gray-500",
    textKey: "common.pending",
  },
  [Task_Status.SKIPPED]: {
    icon: CircleDotIcon,
    class: "text-gray-500",
    textKey: "common.skipped",
  },
  [Task_Status.CANCELED]: {
    icon: CircleDotIcon,
    class: "text-gray-500",
    textKey: "common.canceled",
  },
  [Task_Status.NOT_STARTED]: {
    icon: CircleDotIcon,
    class: "text-gray-500",
    textKey: "common.pending",
  },
};

// Props
const props = defineProps<{
  task: Task;
  taskRun: TaskRun;
  sheet?: Sheet;
  executorEmail?: string;
  duration?: string;
}>();

const { t } = useI18n();

// Status display
const taskStatus = computed(() => props.task.status);

const statusConfig = computed(() => {
  return STATUS_CONFIG[taskStatus.value] ?? DEFAULT_STATUS_CONFIG;
});
const statusIcon = computed(() => statusConfig.value.icon);
const statusClass = computed(() => statusConfig.value.class);
const statusText = computed(() => t(statusConfig.value.textKey));

// Fetch and summarize task run log
const { summary } = useTaskRunLogSummary(
  () => props.taskRun,
  () => true
);

const affectedRowsDisplay = computed(() => {
  if (!summary.value.hasAffectedRows) return "";
  const rows = summary.value.totalAffectedRows;
  if (rows <= BigInt(0)) return "";
  return `${rows.toLocaleString()} row${rows === BigInt(1) ? "" : "s"}`;
});
</script>
