<template>
  <router-link
    :to="task.name"
    exact-active-class=""
    class="font-medium text-main hover:border-b hover:border-b-main"
  >
    <span>{{ databaseForTask(project, task).databaseName }}</span>
    <span class="ml-1 text-control-placeholder">#{{ taskUID }}</span>
  </router-link>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  databaseForTask,
  extractProjectResourceName,
  extractTaskUID,
} from "@/utils";

const props = defineProps<{
  plan: Plan;
  task: Task;
}>();

const project = computed(() => {
  return useProjectV1Store().getProjectByName(
    `${projectNamePrefix}${extractProjectResourceName(props.plan.name)}`
  );
});

const taskUID = computed(() => {
  return extractTaskUID(props.task.name);
});
</script>
