<template>
  <NTooltip v-if="enabled" :disabled="!tooltip" placement="top">
    <template #trigger>
      <heroicons-solid:shield-exclamation
        class="text-control inline-block"
        :class="tooltip ? 'pointer-events-auto' : 'pointer-events-none'"
        v-bind="$attrs"
      />
    </template>

    <span>{{ $t("environment.production-environment") }}</span>
  </NTooltip>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { featureToRef } from "@/store";
import type { Environment } from "@/types/v1/environment";
import { PlanFeature } from "@/types/proto/v1/subscription_service";

const props = withDefaults(
  defineProps<{
    environment: Environment;
    tooltip?: boolean;
  }>(),
  {
    tooltip: false,
  }
);

const hasEnvironmentTierPolicyFeature = featureToRef(
  PlanFeature.FEATURE_ENVIRONMENT_TIERS
);

const enabled = computed((): boolean => {
  if (!hasEnvironmentTierPolicyFeature.value) {
    return false;
  }
  return props.environment?.tags?.protected === "protected"
});
</script>
