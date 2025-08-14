<template>
  <span class="font-medium text-main">
    <EnvironmentV1Name
      :link="false"
      :environment="environmentStore.getEnvironmentByName(stage.environment)"
    />
  </span>
</template>

<script lang="ts" setup>
import { EnvironmentV1Name } from "@/components/v2";
import { useEnvironmentV1Store } from "@/store";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";

defineProps<{
  stage: Stage;
}>();

const environmentStore = useEnvironmentV1Store();
</script>
