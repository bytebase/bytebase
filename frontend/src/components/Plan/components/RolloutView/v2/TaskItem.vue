<template>
  <div
    class="collapsible-task-item relative bg-white border border-gray-300 rounded-lg transition-all group"
    :class="[
      isExpanded ? 'shadow-sm' : 'hover:shadow-sm',
    ]"
  >
    <!-- Absolute checkbox - center-aligned on left border -->
    <div
      v-if="isSelectable"
      class="absolute left-0 -translate-y-1/2 -translate-x-1/2 z-1 transition-opacity"
      :class="[inSelectMode || isSelected ? 'opacity-100' : 'opacity-0 group-hover:opacity-100', isExpanded ? 'top-8': 'top-6']"
      @click.stop
    >
      <NCheckbox
        :checked="isSelected"
        @update:checked="emit('toggle-select')"
      />
    </div>

    <!-- Task content -->
    <div class="w-full" :class="isExpanded ? 'p-4 space-y-3' : 'py-3 px-4'">
      <!-- Header section -->
      <div class="flex items-center justify-between gap-x-3 mb-2">
        <div class="flex items-center gap-x-2 flex-1 min-w-0">
          <div
            :class="readonly ? '' : 'cursor-pointer hover:opacity-80'"
            class="transition-opacity"
          >
            <TaskStatus :status="task.status" :size="isExpanded ? 'large' : 'small'" />
          </div>
          <RouterLink
            v-if="!readonly"
            :to="`/${task.name}`"
            class="transition-opacity shrink-0 hover:opacity-80"
          >
            <DatabaseDisplay
              :database="task.target"
              :size="isExpanded ? 'large' : 'medium'"
              :link="false"
            />
          </RouterLink>
          <div v-else class="shrink-0">
            <DatabaseDisplay
              :database="task.target"
              :size="isExpanded ? 'large' : 'medium'"
              :link="false"
            />
          </div>
          <div v-if="!isExpanded" class="flex items-center gap-x-2 ml-auto shrink-0">
            <NTag size="tiny" round class="opacity-80">
              {{ taskTypeDisplay }}
            </NTag>
            <!-- Scheduled time indicator -->
            <ScheduledTimeIndicator
              v-if="timingType === 'scheduled'"
              :time="scheduledTime"
              :title="t('task.scheduled-time')"
            />
            <!-- Running indicator -->
            <span
              v-else-if="timingType === 'running'"
              class="flex items-center gap-x-1 text-xs text-yellow-600"
            >
              <LoaderCircleIcon class="w-3 h-3 animate-spin" />
              {{ timingDisplay }}
            </span>
          </div>
        </div>

        <!-- Expand/Collapse button -->
        <button
          v-if="!readonly"
          class="shrink-0 p-1 hover:bg-gray-100 rounded transition-colors"
          :class="{ 'self-start': isExpanded }"
          @click.stop="emit('toggle-expand')"
        >
          <ChevronRightIcon v-if="!isExpanded" class="w-4 h-4 text-gray-600" />
          <ChevronDownIcon v-else class="w-4 h-4 text-gray-600" />
        </button>
      </div>

      <!-- SQL preview - collapsed only -->
      <div v-if="!isExpanded" class="space-y-1">
        <div class="flex flex-row justify-start items-center gap-2">
          <NTag size="tiny" round class="opacity-80">
            {{ t("common.statement") }}
          </NTag>
          <div
            class="text-xs text-gray-600 font-mono truncate min-w-0"
          >
            {{ statementPreview }}
          </div>
        </div>
        <!-- Error preview for failed tasks -->
        <div v-if="errorPreview" class="text-xs text-red-600 truncate pl-1">
          {{ t("common.error") }}: {{ errorPreview }}
        </div>
        <!-- Skipped reason -->
        <div v-if="task.status === Task_Status.SKIPPED && task.skippedReason" class="text-xs text-gray-500 italic truncate pl-1">
          {{ task.skippedReason }}
        </div>
      </div>

      <!-- Task metadata - expanded only -->
      <div v-else class="space-y-3">
        <!-- Task information line with quick actions -->
        <div class="flex items-center justify-between gap-x-2">
          <div class="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-gray-600">
            <span>{{ t("common.type") }}: {{ taskTypeDisplay }}</span>
            <span v-if="executorEmail" class="text-gray-400">·</span>
            <span v-if="executorEmail">{{ t("task.executed-by") }}: {{ executorEmail }}</span>
            <template v-if="timingType === 'scheduled'">
              <span class="text-gray-400">·</span>
              <ScheduledTimeIndicator
                :time="scheduledTime"
                :label="t('task.scheduled-time')"
              />
            </template>
            <template v-else-if="timingDisplay">
              <span class="text-gray-400">·</span>
              <span>{{ t("common.duration") }}: {{ timingDisplay }}</span>
            </template>
            <span v-if="affectedRowsDisplay" class="text-gray-400">·</span>
            <span v-if="affectedRowsDisplay">{{ t("task.affected-rows") }}: {{ affectedRowsDisplay }}</span>
          </div>
          <!-- Quick action buttons -->
          <div v-if="hasActions" class="flex items-center gap-x-2 shrink-0">
            <NButton
              v-if="canRun"
              size="tiny"
              type="primary"
              @click.stop="runTask"
            >
              <template #icon>
                <PlayIcon class="w-3 h-3" />
              </template>
              {{ task.status === Task_Status.FAILED ? t("common.retry") : t("common.run") }}
            </NButton>
            <NButton
              v-if="canSkip"
              size="tiny"
              @click.stop="skipTask"
            >
              <template #icon>
                <SkipForwardIcon class="w-3 h-3" />
              </template>
              {{ t("common.skip") }}
            </NButton>
            <NButton
              v-if="canCancel"
              size="tiny"
              @click.stop="cancelTask"
            >
              <template #icon>
                <XIcon class="w-3 h-3" />
              </template>
              {{ t("common.cancel") }}
            </NButton>
          </div>
        </div>

        <!-- SQL Statement section -->
        <div>
          <div class="flex items-center justify-between mb-1">
            <span class="text-sm font-medium text-gray-700">{{ t("common.statement") }}</span>
            <RouterLink
              v-if="!readonly"
              :to="`/${task.name}`"
            >
            <NButton text icon-placement="right" size="tiny" type="info">
              <template #icon>
                <ArrowUpRightIcon />
              </template>
              {{ t("rollout.task.view-full-details") }}
            </NButton>
            </RouterLink>
          </div>

          <BBSpin v-if="loading" />
          <template v-else>
            <HighlightCodeBlock
              :code="displayedStatement"
              language="sql"
              :lazy="true"
              class="text-sm whitespace-pre-wrap wrap-break-word max-h-64 overflow-auto rounded-t p-2 bg-white border border-gray-200"
              :class="isStatementTruncated ? 'rounded-b-none border-b-0' : 'rounded-b'"
            />
            <div
              v-if="isStatementTruncated"
              class="px-3 py-1.5 text-xs text-gray-500 bg-gray-50 border border-gray-200 rounded-b"
            >
              {{ t("rollout.task.statement-truncated-hint") }}
            </div>
          </template>
        </div>

        <!-- Latest Task Run Info -->
        <div v-if="latestTaskRun">
          <div class="flex items-center gap-x-2 mb-2">
            <span class="text-sm font-medium text-gray-700">{{ t("task-run.latest") }}</span>
            <span class="text-sm text-gray-500">
              <Timestamp :timestamp="latestTaskRun.updateTime" />
            </span>
          </div>

          <!-- Error message for failed tasks -->
          <div v-if="task.status === Task_Status.FAILED && latestTaskRun.detail" class="mb-2 p-2 bg-red-50 border border-red-200 rounded">
            <div class="text-sm text-red-600 whitespace-pre-wrap">
              {{ latestTaskRun.detail }}
            </div>
          </div>

          <!-- Waiting message for pending tasks -->
          <div v-else-if="waitingMessage" class="mb-2 p-2 bg-blue-50 border border-blue-200 rounded">
            <div class="text-sm text-blue-700 flex items-center gap-x-1">
              <span>⏳</span>
              <span>{{ waitingMessage }}</span>
            </div>
          </div>

          <!-- Success message with result summary -->
          <div v-else-if="task.status === Task_Status.DONE">
            <div class="flex items-center gap-x-2 text-sm text-green-700 mb-1">
              <span>✓</span>
              <span>{{ t("common.success") }}</span>
              <span v-if="resultSummary" class="text-gray-600">· {{ resultSummary }}</span>
            </div>
          </div>

          <!-- Default detail display for other statuses -->
          <div v-else-if="latestTaskRun.detail" class="text-sm text-gray-700 whitespace-pre-wrap">
            {{ latestTaskRun.detail }}
          </div>
          <div v-else class="text-xs text-gray-500 italic">
            {{ t("task-run.no-detail") }}
          </div>
        </div>
      </div>
    </div>

    <!-- Task Rollout Action Panel -->
    <TaskRolloutActionPanel
      v-if="currentAction && actionTarget"
      :show="showActionPanel"
      :action="currentAction"
      :target="actionTarget"
      @close="closeActionPanel"
      @confirm="handleActionConfirm"
    />
  </div>
</template>

<script lang="ts" setup>
import { last } from "lodash-es";
import {
  ArrowUpRightIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  LoaderCircleIcon,
  PlayIcon,
  SkipForwardIcon,
  XIcon,
} from "lucide-vue-next";
import { NButton, NCheckbox, NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink } from "vue-router";
import BBSpin from "@/bbkit/BBSpin.vue";
import HighlightCodeBlock from "@/components/HighlightCodeBlock.vue";
import Timestamp from "@/components/misc/Timestamp.vue";
import { usePlanContextWithRollout } from "@/components/Plan";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import TaskRolloutActionPanel from "@/components/Plan/components/RolloutView/TaskRolloutActionPanel.vue";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { taskRunNamePrefix } from "@/store";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status, Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import { useTaskActions } from "./composables/useTaskActions";
import { useTaskRunSummary } from "./composables/useTaskRunSummary";
import { useTaskStatement } from "./composables/useTaskStatement";
import { useTaskTiming } from "./composables/useTaskTiming";
import ScheduledTimeIndicator from "./ScheduledTimeIndicator.vue";

const props = withDefaults(
  defineProps<{
    task: Task;
    stage: Stage;
    isExpanded: boolean;
    isSelected: boolean;
    isSelectable: boolean;
    inSelectMode: boolean;
    readonly?: boolean;
  }>(),
  {
    readonly: false,
  }
);

const emit = defineEmits<{
  (event: "toggle-expand"): void;
  (event: "toggle-select"): void;
}>();

const { t } = useI18n();
const {
  taskRuns: allTaskRuns,
  readonly: contextReadonly,
  events,
} = usePlanContextWithRollout();

// Task actions (Run/Skip/Cancel)
const {
  canRun,
  canSkip,
  canCancel,
  hasActions,
  showActionPanel,
  currentAction,
  actionTarget,
  runTask,
  skipTask,
  cancelTask,
  closeActionPanel,
} = useTaskActions(
  () => props.task,
  () => props.stage,
  { readonly: () => props.readonly || contextReadonly.value }
);

// Handle action confirmed - trigger data refresh
const handleActionConfirm = () => {
  events.emit("status-changed", { eager: true });
};

const { loading, statementPreview, displayedStatement, isStatementTruncated } =
  useTaskStatement(
    () => props.task,
    () => props.isExpanded
  );

const latestTaskRun = computed(() => {
  const taskRunsForTask = allTaskRuns.value.filter((run) =>
    run.name.startsWith(`${props.task.name}/${taskRunNamePrefix}`)
  );
  return last(taskRunsForTask);
});

const { timingDisplay, timingType, scheduledTime } = useTaskTiming(
  () => props.task,
  () => latestTaskRun.value
);

const { totalAffectedRows } = useTaskRunSummary(
  () => latestTaskRun.value,
  () => props.isExpanded
);

const affectedRowsDisplay = computed(() => {
  const rows = totalAffectedRows.value;
  if (rows === undefined) return "";
  return `${rows.toLocaleString()} row${rows === BigInt(1) ? "" : "s"}`;
});

const taskTypeDisplay = computed(() => {
  switch (props.task.type) {
    case Task_Type.DATABASE_CREATE:
      return t("task.type.database-create");
    case Task_Type.DATABASE_MIGRATE:
      return t("task.type.migrate");
    case Task_Type.DATABASE_SDL:
      return t("task.type.database-sdl");
    case Task_Type.DATABASE_EXPORT:
      return t("task.type.database-export");
    case Task_Type.GENERAL:
      return t("task.type.general");
    default:
      return "";
  }
});

const errorPreview = computed(() => {
  if (props.task.status !== Task_Status.FAILED) return "";
  const detail = latestTaskRun.value?.detail || "";
  const firstLine = detail.split("\n")[0];
  const maxLength = 80;
  return firstLine.length > maxLength
    ? firstLine.substring(0, maxLength) + "..."
    : firstLine;
});

const executorEmail = computed(() => {
  const creator = latestTaskRun.value?.creator || "";
  // Extract email from format: users/email@example.com
  const match = creator.match(/users\/([^/]+)/);
  return match?.[1] || "";
});

const resultSummary = computed(() => {
  const taskRun = latestTaskRun.value;
  if (!taskRun) return "";

  // For migrations and SDL, show schema version
  if (taskRun.schemaVersion) {
    return t("task.result.schema-version", { version: taskRun.schemaVersion });
  }

  // For exports, show archive status
  if (taskRun.exportArchiveStatus) {
    return t("task.result.export-archive-ready");
  }

  return "";
});

const waitingMessage = computed(() => {
  const taskRun = latestTaskRun.value;
  if (!taskRun || props.task.status !== Task_Status.PENDING) return "";

  const schedulerInfo = taskRun.schedulerInfo;
  if (!schedulerInfo?.waitingCause) return "";

  const cause = schedulerInfo.waitingCause.cause;
  if (cause.case === "connectionLimit") {
    return t("task.waiting.connection-limit");
  }
  if (cause.case === "parallelTasksLimit") {
    return t("task.waiting.parallel-tasks-limit");
  }
  if (cause.case === "task") {
    return t("task.waiting.blocking-task");
  }

  return "";
});
</script>
