<template>
  <div v-if="stage" class="w-full h-full flex flex-col">
    <div class="px-4">
      <!-- Stage Header -->
      <div class="w-full flex flex-row pt-2 gap-2">
        <NTag round>{{ $t("common.stage") }}</NTag>
        <span class="font-medium text-lg">{{ environmentTitle }}</span>
      </div>
    </div>

    <NDivider class="!my-4" />

    <div class="px-4">
      <!-- Task Filter -->
      <TaskFilter
        :rollout="rollout"
        :task-status-list="taskStatusFilter"
        :stage="stage"
        @update:task-status-list="taskStatusFilter = $event"
      />
    </div>

    <!-- Tasks Table View -->
    <div class="flex-1 min-h-0">
      <TaskTableView :task-status-filter="taskStatusFilter" :stage="stage" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { NDivider, NTag } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { useEnvironmentV1Store } from "@/store";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanContextWithRollout } from "../../logic";
import TaskFilter from "./TaskFilter.vue";
import TaskTableView from "./TaskTableView.vue";

const props = defineProps<{
  rolloutId: string;
  stageId: string;
}>();

const route = useRoute();
const environmentStore = useEnvironmentV1Store();
const { rollout } = usePlanContextWithRollout();

const stage = computed(() => {
  return rollout.value.stages.find((s) => s.id === props.stageId) as Stage;
});

const taskStatusFilter = ref<Task_Status[]>([]);

// Watch for query parameter changes
watch(
  () => route.query.taskStatus,
  (taskStatus) => {
    if (taskStatus && typeof taskStatus === "string") {
      // Find the Task_Status enum value from the string
      const statusValue = Object.entries(Task_Status).find(
        ([key]) => key === taskStatus
      )?.[1];

      if (statusValue !== undefined && typeof statusValue === "number") {
        taskStatusFilter.value = [statusValue as Task_Status];
      } else {
        taskStatusFilter.value = [];
      }
    } else {
      taskStatusFilter.value = [];
    }
  },
  { immediate: true }
);

// Stage environment info
const environmentTitle = computed(() => {
  if (!stage.value) return "";
  return environmentStore.getEnvironmentByName(stage.value.environment).title;
});
</script>
