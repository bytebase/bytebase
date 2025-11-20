<template>
  <div class="flex items-center gap-2">
    <TaskStatus :status="stageStatus" size="small" />
    <span class="whitespace-nowrap">
      <EnvironmentV1Name :environment="environment" :link="false" />
    </span>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import EnvironmentV1Name from "@/components/v2/Model/EnvironmentV1Name.vue";
import { useEnvironmentV1Store } from "@/store";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { useStageStatus } from "./composables/useStageStatus";

const props = defineProps<{
  stage: Stage;
  isCreated: boolean;
}>();

const environmentStore = useEnvironmentV1Store();

const environment = computed(() => {
  return environmentStore.getEnvironmentByName(props.stage.environment);
});

const stageComputed = computed(() => props.stage);
const { stageStatus } = useStageStatus(stageComputed, props.isCreated);
</script>
