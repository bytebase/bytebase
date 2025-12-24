<template>
  <div class="space-y-2">
    <!-- Line 1: Title + Status + Timestamp + Metadata -->
    <div class="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
      <!-- Title -->
      <span class="font-medium text-gray-700 shrink-0">{{
        t("task-run.latest")
      }}</span>
      <Separator />
      <!-- Status badge -->
      <span
        class="flex items-center gap-x-1 shrink-0"
        :class="statusConfig.class"
      >
        <component
          :is="statusConfig.icon"
          class="w-4 h-4"
          :class="{ 'animate-spin': status === Task_Status.RUNNING }"
        />
        <span>{{ t(statusConfig.textKey) }}</span>
      </span>
      <Separator />
      <!-- Timestamp -->
      <span class="text-gray-500 shrink-0">
        <Timestamp :timestamp="updateTime" />
      </span>
      <!-- Metadata items -->
      <template v-if="executorEmail">
        <Separator />
        <MetadataItem :tooltip="t('task.executed-by')">
          <UserIcon class="w-3.5 h-3.5 shrink-0" />
          <span class="truncate">{{ executorEmail }}</span>
        </MetadataItem>
      </template>
      <template v-if="duration">
        <Separator />
        <MetadataItem :tooltip="t('common.duration')">
          <ClockIcon class="w-3.5 h-3.5" />
          {{ duration }}
        </MetadataItem>
      </template>
      <template v-if="affectedRowsDisplay">
        <Separator />
        <MetadataItem :tooltip="t('task.affected-rows')">
          <RowsIcon class="w-3.5 h-3.5" />
          {{ affectedRowsDisplay }}
        </MetadataItem>
      </template>
    </div>

    <!-- Line 2: Task run logs -->
    <TaskRunLogViewer
      v-if="summary.entries.length > 0"
      :entries="summary.entries"
      :sheet="sheet"
    />
  </div>
</template>

<script lang="ts" setup>
import type { Timestamp as TimestampType } from "@bufbuild/protobuf/wkt";
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
import type { Component, FunctionalComponent } from "vue";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import Timestamp from "@/components/misc/Timestamp.vue";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import type { TaskRunLogSummary } from "./composables/useTaskRunLogSummary";
import { TaskRunLogViewer } from "./TaskRunLogViewer";

// Types
interface StatusConfig {
  icon: Component;
  class: string;
  textKey: string;
}

// Functional components for repeated patterns
const Separator: FunctionalComponent = () =>
  h("span", { class: "text-gray-300" }, "·");

const MetadataItem: FunctionalComponent<{ tooltip: string }> = (
  props,
  { slots }
) =>
  h(NTooltip, null, {
    trigger: () =>
      h(
        "span",
        {
          class:
            "flex items-center gap-x-1 text-gray-500 shrink-0 cursor-default",
        },
        slots.default?.()
      ),
    default: () => props.tooltip,
  });

// Status configuration - defined outside setup for better performance
const STATUS_CONFIG: Record<Task_Status, StatusConfig> = {
  [Task_Status.FAILED]: {
    icon: XCircleIcon,
    class: "text-red-600",
    textKey: "task.status.failed",
  },
  [Task_Status.RUNNING]: {
    icon: LoaderCircleIcon,
    class: "text-blue-600",
    textKey: "task.status.running",
  },
  [Task_Status.DONE]: {
    icon: CheckCircle2Icon,
    class: "text-green-600",
    textKey: "task.status.done",
  },
  [Task_Status.PENDING]: {
    icon: CircleDotIcon,
    class: "text-gray-500",
    textKey: "task.status.pending",
  },
  [Task_Status.SKIPPED]: {
    icon: CircleDotIcon,
    class: "text-gray-500",
    textKey: "task.status.skipped",
  },
  [Task_Status.CANCELED]: {
    icon: CircleDotIcon,
    class: "text-gray-500",
    textKey: "task.status.canceled",
  },
  [Task_Status.NOT_STARTED]: {
    icon: CircleDotIcon,
    class: "text-gray-500",
    textKey: "task.status.not-started",
  },
  [Task_Status.STATUS_UNSPECIFIED]: {
    icon: CircleDotIcon,
    class: "text-gray-500",
    textKey: "common.unknown",
  },
};

// Props
const props = defineProps<{
  status: Task_Status;
  updateTime?: TimestampType;
  sheet?: Sheet;
  executorEmail?: string;
  duration?: string;
  affectedRowsDisplay?: string;
  summary: TaskRunLogSummary;
  taskName?: string;
}>();

const { t } = useI18n();

// Single computed for status - eliminates computed chain
const statusConfig = computed(() => STATUS_CONFIG[props.status]);
</script>
