<template>
  <div
    class="collapsible-task-item relative bg-white border border-gray-300 rounded-lg transition-all group"
    :class="[
      isExpanded ? 'shadow-sm' : 'hover:shadow-sm',
    ]"
  >
    <!-- Task content -->
    <div class="w-full" :class="isExpanded ? 'py-4 pl-3 pr-4 space-y-3' : 'py-2.5 pl-3 pr-4'">
      <!-- Header section -->
      <div
        class="flex items-center justify-between gap-x-3"
        :class="{ 'mb-2': isExpanded }"
      >
        <div class="flex items-center gap-x-2 flex-1 min-w-0">
          <!-- Inline checkbox - always visible like Gmail/GitHub -->
          <NCheckbox
            v-if="isSelectable"
            :checked="isSelected"
            class="shrink-0"
            @click.stop
            @update:checked="emit('toggle-select')"
          />
          <TaskStatus :status="task.status" :size="isExpanded ? 'large' : 'small'" />
          <RouterLink
            v-if="!readonly"
            :to="taskDetailRoute"
            class="shrink-0 hover:opacity-80 transition-opacity"
          >
            <DatabaseDisplay
              :database="task.target"
              :size="isExpanded ? 'large' : 'medium'"
              :link="false"
            />
          </RouterLink>
          <DatabaseDisplay
            v-else
            :database="task.target"
            :size="isExpanded ? 'large' : 'medium'"
            :link="false"
            class="shrink-0"
          />
          <!-- Collapsed view: contextual info + type -->
          <div v-if="!isExpanded" class="flex items-center gap-x-1.5 ml-auto shrink-0 text-xs text-gray-500">
            <!-- Status-contextual info first -->
            <template v-if="timingType === 'scheduled'">
              <ScheduledTimeIndicator
                :time="scheduledTime"
                :title="t('task.scheduled-time')"
              />
              <span class="text-gray-300">·</span>
            </template>
            <template v-else-if="timingType === 'running'">
              <span class="flex items-center gap-x-1 text-blue-600">
                <LoaderCircleIcon class="w-3 h-3 animate-spin" />
                {{ timingDisplay }}
              </span>
              <span class="text-gray-300">·</span>
            </template>
            <template v-else-if="collapsedContextInfo">
              <span>{{ collapsedContextInfo }}</span>
              <span class="text-gray-300">·</span>
            </template>
            <!-- Task type last -->
            <NTag round size="tiny" class="opacity-80">
              {{ taskTypeDisplay }}
            </NTag>
          </div>
        </div>

        <!-- Expand/Collapse button -->
        <button
          v-if="!readonly"
          class="shrink-0 p-1 hover:bg-gray-100 rounded transition-colors"
          :class="{ 'self-start': isExpanded }"
          @click.stop="emit('toggle-expand')"
        >
          <ChevronRightIcon v-if="!isExpanded" class="w-4 h-4 text-gray-500" />
          <ChevronDownIcon v-else class="w-4 h-4 text-gray-500" />
        </button>
      </div>

      <!-- Collapsed: contextual status line (time + error/skip) -->
      <div
        v-if="!isExpanded && collapsedStatusText"
        class="flex items-center gap-x-2 text-xs mt-1"
      >
        <NTag v-if="latestTaskRun?.createTime" size="tiny" round>
          <Timestamp :timestamp="latestTaskRun.createTime" />
        </NTag>
        <span
          class="truncate cursor-pointer"
          :class="task.status === Task_Status.FAILED ? 'text-red-600' : 'text-gray-500 italic'"
          @click.stop="emit('toggle-expand')"
        >
          {{ collapsedStatusText }}
        </span>
      </div>

      <!-- Task metadata - expanded only -->
      <div v-if="isExpanded" class="space-y-3">
        <!-- Task information line with quick actions -->
        <div class="flex items-center justify-between gap-x-2">
          <div class="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-gray-600">
            <span>{{ t("common.type") }}: {{ taskTypeDisplay }}</span>
            <template v-if="timingType === 'scheduled'">
              <span class="text-gray-400">·</span>
              <ScheduledTimeIndicator
                :time="scheduledTime"
                :label="t('task.scheduled-time')"
              />
            </template>
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
        <div v-if="!isReleaseTask">
          <div class="flex items-center justify-between mb-1">
            <span class="text-sm font-medium text-gray-700">{{ t("common.statement") }}</span>
            <RouterLink
              v-if="!readonly"
              :to="taskDetailRoute"
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

        <!-- Release Info section for release-based tasks -->
        <ReleaseInfoCard
          v-else
          :release-name="releaseName"
          :compact="true"
        />

        <!-- Latest Task Run Info -->
        <LatestTaskRunInfo
          v-if="latestTaskRun"
          :status="task.status"
          :update-time="latestTaskRun.updateTime"
          :sheet="taskSheet"
          :executor-email="executorEmail"
          :duration="timingType !== 'scheduled' ? timingDisplay : undefined"
          :affected-rows-display="affectedRowsDisplay"
          :summary="taskRunLogSummary"
          :task-name="task.name"
        />
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
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK } from "@/router/dashboard/projectV1";
import { taskRunNamePrefix, useSheetV1Store } from "@/store";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
  extractStageNameFromTaskName,
  extractStageUID,
  extractTaskUID,
  isReleaseBasedTask,
  releaseNameOfTaskV1,
  sheetNameOfTaskV1,
} from "@/utils";
import { useTaskActions } from "./composables/useTaskActions";
import { useTaskDisplay } from "./composables/useTaskDisplay";
import { useTaskRunLogSummary } from "./composables/useTaskRunLogSummary";
import { useTaskStatement } from "./composables/useTaskStatement";
import { useTaskTiming } from "./composables/useTaskTiming";
import LatestTaskRunInfo from "./LatestTaskRunInfo.vue";
import ReleaseInfoCard from "./ReleaseInfoCard.vue";
import ScheduledTimeIndicator from "./ScheduledTimeIndicator.vue";
import TaskRolloutActionPanel from "./TaskRolloutActionPanel.vue";

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
const { taskRuns: allTaskRuns, events } = usePlanContextWithRollout();

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
  () => props.stage
);

// Handle action confirmed - trigger data refresh
const handleActionConfirm = () => {
  events.emit("status-changed", { eager: true });
};

const taskDetailRoute = computed(() => {
  const projectName = extractProjectResourceName(props.task.name);
  const planId = extractPlanUIDFromRolloutName(props.task.name);
  const stageName = extractStageNameFromTaskName(props.task.name);
  const stageId = extractStageUID(stageName);
  const taskId = extractTaskUID(props.task.name);

  return {
    name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
    params: {
      projectId: projectName,
      planId: planId || "-",
      stageId: stageId || "-",
      taskId: taskId || "-",
    },
  };
});

const isReleaseTask = computed(() => isReleaseBasedTask(props.task));

// Release name for release-based tasks
const releaseName = computed(() => releaseNameOfTaskV1(props.task));

const { loading, displayedStatement, isStatementTruncated } = useTaskStatement(
  () => props.task,
  () => props.isExpanded
);

// Get sheet for extracting individual commands in log display
const sheetStore = useSheetV1Store();
const taskSheet = computed(() => {
  const sheetName = sheetNameOfTaskV1(props.task);
  if (!sheetName) return undefined;
  return sheetStore.getSheetByName(sheetName);
});

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

const { summary: taskRunLogSummary } = useTaskRunLogSummary(
  () => latestTaskRun.value,
  () => props.isExpanded
);

const affectedRowsDisplay = computed(() => {
  if (!taskRunLogSummary.value.hasAffectedRows) return "";
  const rows = taskRunLogSummary.value.totalAffectedRows;
  if (rows <= BigInt(0)) return "";
  return `${rows.toLocaleString()} row${rows === BigInt(1) ? "" : "s"}`;
});

// Task display formatting (type, executor, messages, collapsed view info)
const {
  taskTypeDisplay,
  executorEmail,
  collapsedContextInfo,
  collapsedStatusText,
} = useTaskDisplay(
  () => props.task,
  () => latestTaskRun.value,
  () => timingDisplay.value,
  () => affectedRowsDisplay.value
);
</script>
