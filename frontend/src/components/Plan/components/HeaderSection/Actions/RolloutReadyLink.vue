<template>
  <RouterLink v-if="rollout" :to="rolloutRoute">
    <NButton icon-placement="right" quaternary>
      <TaskStatus :status="rolloutStatus" size="tiny" />
      <span class="mx-1">{{ $t("common.rollout") }}</span>
      <span>#{{ planID }}</span>
      <template #icon>
        <ArrowUpRightIcon class="opacity-60" />
      </template>
    </NButton>
  </RouterLink>
</template>

<script lang="ts" setup>
import { ArrowUpRightIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { usePlanContext } from "@/components/Plan/logic";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { PROJECT_V1_ROUTE_PLAN_ROLLOUT } from "@/router/dashboard/projectV1";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
  getRolloutStatus,
} from "@/utils";

const { plan, rollout } = usePlanContext();

const rolloutStatus = computed(() => {
  if (!rollout.value) return Task_Status.NOT_STARTED;
  return getRolloutStatus(rollout.value);
});

const planID = computed(() => {
  if (!rollout.value?.name) return "";
  return extractPlanUIDFromRolloutName(rollout.value.name);
});

const rolloutRoute = computed(() => ({
  name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  params: {
    projectId: extractProjectResourceName(plan.value.name),
    planId: extractPlanUIDFromRolloutName(rollout.value?.name ?? ""),
  },
}));
</script>
