<template>
  <div v-if="selectedStage" class="w-full">
    <!-- Stage header -->
    <div class="px-4">
      <div class="w-full flex flex-row items-center justify-between py-4 gap-2">
        <!-- Left: Stage info and filters -->
        <div class="flex flex-row items-center gap-3">
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
          <NPopconfirm
            v-else
            @positive-click="$emit('create-stage', selectedStage)"
          >
            <template #trigger>
              <NButton type="primary" :size="'small'">
                <template #icon>
                  <PlusIcon :size="16" />
                </template>
                {{ $t("common.create") }}
              </NButton>
            </template>
            {{ $t("rollout.stage.confirm-create") }}
          </NPopconfirm>
          <!-- Mobile: Timeline drawer trigger (rightmost) -->
          <StageContentSidebar
            v-if="!isWideScreen && isStageCreated(selectedStage)"
            :stage="selectedStage"
            :task-runs="taskRuns"
          />
        </div>
      </div>
    </div>

    <!-- Main content area: responsive layout -->
    <div class="flex flex-row">
      <!-- Task list content -->
      <div class="flex-1 min-w-0">
        <TaskList
          :stage="selectedStage"
          :rollout="rollout"
          :filter-statuses="filterStatuses"
          :readonly="!isStageCreated(selectedStage)"
        />
      </div>

      <!-- Desktop: Timeline sidebar -->
      <StageContentSidebar
        v-if="isWideScreen && isStageCreated(selectedStage)"
        :stage="selectedStage"
        :task-runs="taskRuns"
      />
    </div>
  </div>

  <div v-else class="flex items-center justify-center py-12">
    <p class="text-gray-500">
      {{ $t("rollout.no-stages") }}
    </p>
  </div>
</template>

<script lang="ts" setup>
import { useWindowSize } from "@vueuse/core";
import { PlayIcon, PlusIcon } from "lucide-vue-next";
import { NButton, NPopconfirm } from "naive-ui";
import { computed, ref } from "vue";
import { usePlanContextWithRollout } from "@/components/Plan/logic";
import type { Rollout, Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import StageContentSidebar from "./StageContentSidebar.vue";
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

const filterStatuses = ref<Task_Status[]>([]);

// Responsive layout: sidebar on wide screen (>= 768px), drawer on narrow
const { width: windowWidth } = useWindowSize();
const isWideScreen = computed(() => windowWidth.value >= 768);

const { taskRuns } = usePlanContextWithRollout();

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
