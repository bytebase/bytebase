<template>
  <div v-if="selectedStage" class="w-full">
    <!-- Stage header -->
    <div class="px-4">
      <div class="w-full flex flex-row items-center justify-between py-4 gap-2">
        <!-- Left: Stage info and filters -->
        <div class="flex flex-row items-center gap-3">
          <div class="flex flex-row items-center gap-2">
            <NTag round>{{ $t("common.stage") }}</NTag>
            <span class="font-medium text-lg">
              <EnvironmentV1Name
                :environment="
                  environmentStore.getEnvironmentByName(selectedStage.environment)
                "
                :null-environment-placeholder="'Null'"
                :link="false"
              />
            </span>
          </div>
          <TaskFilter
            v-if="isStageCreated(selectedStage)"
            :stage="selectedStage"
            :selected-statuses="filterStatuses"
            @update:selected-statuses="handleFilterStatusesChange"
          />
        </div>

        <!-- Right: Stage actions -->
        <div class="flex items-center gap-x-2 shrink-0">
          <NButton
            v-if="isStageCreated(selectedStage)"
            type="primary"
            :disabled="!canRunStage"
            :size="'small'"
            @click="$emit('run-stage', selectedStage)"
          >
            <template #icon>
              <PlayIcon :size="16" />
            </template>
            {{ $t("rollout.stage.run-stage") }}
          </NButton>
          <NButton
            v-else
            type="primary"
            :size="'small'"
            @click="$emit('create-stage', selectedStage)"
          >
            <template #icon>
              <PlusIcon :size="16" />
            </template>
            {{ $t("rollout.stage.create-stage") }}
          </NButton>
        </div>
      </div>
    </div>

    <!-- Main content area: responsive layout -->
    <div class="flex" :class="isWideScreen ? 'flex-row' : 'flex-col'">
      <!-- Timeline: inline above on narrow screen -->
      <div v-if="!isWideScreen && isStageCreated(selectedStage)" class="mb-4">
        <StageTimeline
          :stage="selectedStage"
          :task-runs="taskRuns"
          :rollout="rollout"
          :rollbackable-task-runs="rollbackableTaskRuns"
          :is-inline="true"
          class="border-y"
        />
      </div>

      <!-- Task list content -->
      <div class="flex-1 min-w-0">
        <TaskList
          :stage="selectedStage"
          :rollout="rollout"
          :filter-statuses="filterStatuses"
          :readonly="!isStageCreated(selectedStage)"
        />
      </div>

      <!-- Timeline: sidebar on wide screen (sticky) -->
      <div
        v-if="isWideScreen && isStageCreated(selectedStage)"
        class="shrink-0 pr-4 self-start sticky top-0"
      >
        <div class="w-64 border rounded-lg">
          <StageTimeline
            :stage="selectedStage"
            :task-runs="taskRuns"
            :rollout="rollout"
            :rollbackable-task-runs="rollbackableTaskRuns"
            :is-inline="false"
          />
        </div>
      </div>
    </div>
  </div>

  <div v-else class="flex items-center justify-center py-12">
    <p class="text-gray-500">
      {{ $t("rollout.no-stages") }}
    </p>
  </div>
</template>

<script lang="ts" setup>
import { useMediaQuery } from "@vueuse/core";
import { PlayIcon, PlusIcon } from "lucide-vue-next";
import { NButton, NTag } from "naive-ui";
import { computed, ref, toRef } from "vue";
import { EnvironmentV1Name } from "@/components/v2";
import { useEnvironmentV1Store } from "@/store";
import type { Rollout, Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanContextWithRollout } from "../../../logic";
import { useRollbackableTasks } from "./composables/useRollbackableTasks";
import StageTimeline from "./StageTimeline.vue";
import TaskFilter from "./TaskFilter.vue";
import TaskList from "./TaskList.vue";

const props = defineProps<{
  selectedStage: Stage | null | undefined;
  rollout: Rollout;
  isStageCreated: (stage: Stage) => boolean;
}>();

defineEmits<{
  (event: "run-stage", stage: Stage): void;
  (event: "create-stage", stage: Stage): void;
}>();

const environmentStore = useEnvironmentV1Store();
const filterStatuses = ref<Task_Status[]>([]);

// Responsive layout: sidebar on wide screen, inline on narrow
const isWideScreen = useMediaQuery("(min-width: 1024px)");

// Rollback functionality
const { taskRuns } = usePlanContextWithRollout();
const selectedStageRef = toRef(props, "selectedStage");
const { rollbackableTaskRuns } = useRollbackableTasks(
  selectedStageRef,
  taskRuns
);

const canRunStage = computed(() => {
  if (!props.selectedStage || !props.isStageCreated(props.selectedStage)) {
    return false;
  }
  // Can run if there are NOT_STARTED, FAILED, or CANCELED tasks
  // PENDING tasks cannot be run (only canceled)
  return props.selectedStage.tasks.some(
    (task) =>
      task.status === Task_Status.NOT_STARTED ||
      task.status === Task_Status.FAILED ||
      task.status === Task_Status.CANCELED
  );
});

const handleFilterStatusesChange = (statuses: Task_Status[]) => {
  filterStatuses.value = statuses;
};
</script>
