<template>
  <router-link
    :to="link"
    exact-active-class=""
    class="font-medium text-main hover:border-b hover:border-b-main"
  >
    <EnvironmentV1Name
      :link="false"
      :environment="environmentStore.getEnvironmentByName(stage.environment)"
    />
  </router-link>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { EnvironmentV1Name } from "@/components/v2";
import { PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE } from "@/router/dashboard/projectV1";
import { useEnvironmentV1Store } from "@/store";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
  extractStageUID,
} from "@/utils";

const props = defineProps<{
  stage: Stage;
}>();

const environmentStore = useEnvironmentV1Store();

const link = computed(() => {
  const { stage } = props;

  return {
    name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
    params: {
      projectId: extractProjectResourceName(stage.name),
      planId: extractPlanUIDFromRolloutName(stage.name),
      stageId: extractStageUID(stage.name),
    },
  };
});
</script>
