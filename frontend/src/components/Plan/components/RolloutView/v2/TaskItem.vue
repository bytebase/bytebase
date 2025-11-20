<template>
  <div
    class="collapsible-task-item relative bg-white border rounded-lg transition-all group"
    :class="[
      borderClass,
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
        <div class="flex items-center gap-x-2">
          <div
            :class="readonly ? '' : 'cursor-pointer hover:opacity-80'"
            class="transition-opacity"
            @click="handleNavigateToDetail"
          >
            <TaskStatus :status="task.status" :size="isExpanded ? 'large' : 'small'" />
          </div>
          <div
            :class="readonly ? '' : 'cursor-pointer hover:opacity-80'"
            class="transition-opacity shrink-0"
            @click="handleNavigateToDetail"
          >
            <DatabaseDisplay
              :database="task.target"
              :size="isExpanded ? 'large' : 'medium'"
              :link="false"
            />
          </div>
        </div>

        <!-- Expand/Collapse button -->
        <button
          class="shrink-0 p-1 hover:bg-gray-100 rounded transition-colors"
          :class="{ 'self-start': isExpanded }"
          @click.stop="emit('toggle-expand')"
        >
          <ChevronRightIcon v-if="!isExpanded" class="w-4 h-4 text-gray-600" />
          <ChevronDownIcon v-else class="w-4 h-4 text-gray-600" />
        </button>
      </div>

      <!-- SQL preview - collapsed only -->
      <div
        v-if="!isExpanded"
        class="flex flex-row justify-start items-start gap-2"
      >
      <NTag size="tiny" round class="opacity-80">
        {{ $t("common.statement") }}
      </NTag>
      <div
        :class="readonly ? '' : 'cursor-pointer'"
        class="text-xs text-gray-600 font-mono truncate"
        @click="handleNavigateToDetail">

        {{ statementPreview }}
      </div>
      </div>

      <!-- SQL Statement section - expanded only -->
      <div v-else>
        <div class="flex items-center justify-between mb-2">
          <span class="text-sm font-medium text-gray-700">{{ t("common.statement") }}</span>
        </div>

        <BBSpin v-if="loading" />
        <HighlightCodeBlock
          v-else
          :code="displayedStatement"
          language="sql"
          class="text-sm whitespace-pre-wrap wrap-break-word max-h-96 overflow-y-auto rounded p-2 bg-white border border-gray-200"
        />
      </div>

      <!-- Latest Task Run Info - expanded only -->
      <div v-if="isExpanded && latestTaskRun">
        <div class="flex items-center justify-between mb-2">
          <div class="flex items-center gap-x-2">
            <span class="text-sm font-medium text-gray-700">{{ t("task-run.latest") }}</span>
            <span class="text-sm text-gray-500">
              <Timestamp :timestamp="latestTaskRun.updateTime" />
            </span>
          </div>
          <a
          v-if="!readonly"
          href="javascript:void(0)"
          class="text-xs text-blue-600 hover:text-blue-700"
          @click="handleNavigateToDetail"
        >
          {{ t("rollout.task.view-full-details") }} →
        </a>
        </div>
        <div v-if="latestTaskRun.detail" class="text-sm text-gray-700 whitespace-pre-wrap">
          {{ latestTaskRun.detail }}
        </div>
        <div v-else class="text-xs text-gray-500 italic">
          {{ t("task-run.no-detail") }}
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { last } from "lodash-es";
import { ChevronDownIcon, ChevronRightIcon } from "lucide-vue-next";
import { NCheckbox, NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import BBSpin from "@/bbkit/BBSpin.vue";
import HighlightCodeBlock from "@/components/HighlightCodeBlock";
import Timestamp from "@/components/misc/Timestamp.vue";
import { usePlanContextWithRollout } from "@/components/Plan";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { taskRunNamePrefix } from "@/store";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { useTaskNavigation } from "./composables/useTaskNavigation";
import { useTaskStatement } from "./composables/useTaskStatement";
import { getTaskBorderClass } from "./utils/taskStatus";

const props = withDefaults(
  defineProps<{
    task: Task;
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
const { taskRuns: allTaskRuns } = usePlanContextWithRollout();
const { navigateToTaskDetail } = useTaskNavigation();

const { loading, statementPreview, displayedStatement } = useTaskStatement(
  () => props.task,
  () => props.isExpanded
);

const borderClass = computed(() => {
  return getTaskBorderClass(props.task.status);
});

const latestTaskRun = computed(() => {
  const taskRunsForTask = allTaskRuns.value.filter((run) =>
    run.name.startsWith(`${props.task.name}/${taskRunNamePrefix}`)
  );
  return last(taskRunsForTask);
});

const handleNavigateToDetail = () => {
  if (props.readonly) {
    return;
  }
  navigateToTaskDetail(props.task);
};
</script>
