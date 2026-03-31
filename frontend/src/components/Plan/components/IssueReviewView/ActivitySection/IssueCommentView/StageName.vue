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
import { buildStageRoute } from "@/router/dashboard/projectV1RouteHelpers";
import { useEnvironmentV1Store } from "@/store";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";

const props = defineProps<{
  stage: Stage;
}>();

const environmentStore = useEnvironmentV1Store();

const link = computed(() => {
  return buildStageRoute(props.stage.name);
});
</script>
