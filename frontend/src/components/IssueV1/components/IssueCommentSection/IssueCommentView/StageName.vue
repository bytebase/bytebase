<template>
  <span class="font-medium text-main">
    {{ environment.title }}
  </span>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useEnvironmentV1Store } from "@/store";
import type { ComposedIssue } from "@/types";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";

const props = defineProps<{
  issue: ComposedIssue;
  stage: Stage;
}>();

const environmentStore = useEnvironmentV1Store();

const environment = computed(() =>
  environmentStore.getEnvironmentByName(props.stage.environment)
);
</script>
