<template>
  <RouterLink :to="planRoute">
    <NButton icon-placement="right" secondary strong>
      <span class="mr-1">#{{ planID }}</span>
      <span>{{ $t("common.plan") }}</span>
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
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/router/dashboard/projectV1";
import { buildPlanDeployRouteFromPlanName } from "@/router/dashboard/projectV1RouteHelpers";
import { extractPlanUID, extractProjectResourceName } from "@/utils";

const { plan } = usePlanContext();

const planID = computed(() => extractPlanUID(plan.value.name));

const planRoute = computed(() => {
  if (plan.value.hasRollout) {
    return buildPlanDeployRouteFromPlanName(plan.value.name);
  }

  return {
    name: PROJECT_V1_ROUTE_PLAN_DETAIL,
    params: {
      projectId: extractProjectResourceName(plan.value.name),
      planId: planID.value,
    },
  };
});
</script>
