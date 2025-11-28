<template>
  <RouterLink v-if="rollout" :to="rolloutRoute">
    <NButton icon-placement="right" size="small" text type="info">
      {{ $t("issue.approval.ready-for-rollout") }}
      <template #icon>
        <ArrowRightIcon class="w-4 h-4" />
      </template>
    </NButton>
  </RouterLink>
</template>

<script lang="ts" setup>
import { ArrowRightIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { usePlanContext } from "@/components/Plan/logic";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL } from "@/router/dashboard/projectV1";
import { extractProjectResourceName, extractRolloutUID } from "@/utils";

const { plan, rollout } = usePlanContext();

const rolloutRoute = computed(() => ({
  name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
  params: {
    projectId: extractProjectResourceName(plan.value.name),
    rolloutId: extractRolloutUID(rollout.value?.name ?? ""),
  },
}));
</script>
