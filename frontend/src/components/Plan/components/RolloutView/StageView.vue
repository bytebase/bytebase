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
        :rollout="rollout || emptyRollout"
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
import { create } from "@bufbuild/protobuf";
import { NDivider, NTag } from "naive-ui";
import { computed, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { useCurrentProjectV1, useEnvironmentV1Store } from "@/store";
import type { Stage, Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import {
  GetRolloutRequestSchema,
  RolloutSchema,
  Task_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import TaskFilter from "./TaskFilter.vue";
import TaskTableView from "./TaskTableView.vue";

const props = defineProps<{
  rolloutId: string;
  stageId: string;
}>();

const { t: _t } = useI18n();
const route = useRoute();
const { project } = useCurrentProjectV1();
const environmentStore = useEnvironmentV1Store();

const rolloutRef = ref<Rollout>();
const routeStageRef = ref<Stage>();
const taskStatusFilter = ref<Task_Status[]>([]);

// Create an empty rollout for TaskFilter when rollout is not available
const emptyRollout = create(RolloutSchema, {});

// Get the stage - either from props or from fetched rollout
const stage = computed(() => {
  if (routeStageRef.value) return routeStageRef.value;
  return undefined;
});

// Get rollout
const rollout = computed(() => rolloutRef.value);

// Fetch rollout and stage when in route mode
watchEffect(async () => {
  const rolloutId = props.rolloutId || (route.params.rolloutId as string);
  const stageId = props.stageId || (route.params.stageId as string);

  if (!rolloutId || !stageId) return;

  try {
    const rolloutName = `projects/${project.value.name.split("/")[1]}/rollouts/${rolloutId}`;
    const request = create(GetRolloutRequestSchema, { name: rolloutName });
    const rollout = await rolloutServiceClientConnect.getRollout(request);
    rolloutRef.value = rollout;

    // Find the specific stage
    for (const rolloutStage of rollout.stages) {
      if (rolloutStage.name.endsWith(`/${stageId}`)) {
        routeStageRef.value = rolloutStage;
        return;
      }
    }
  } catch (error) {
    console.error("Failed to fetch rollout:", error);
  }
});

// Stage environment info
const environmentTitle = computed(() => {
  if (!stage.value) return "";
  return environmentStore.getEnvironmentByName(stage.value.environment).title;
});
</script>
