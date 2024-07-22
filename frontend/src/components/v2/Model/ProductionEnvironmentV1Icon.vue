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
import type { Environment } from "@/types/proto/v1/environment_service";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";

const props = withDefaults(
  defineProps<{
    environment: Environment;
    tier?: EnvironmentTier;
    tooltip?: boolean;
  }>(),
  {
    tier: EnvironmentTier.UNPROTECTED,
    tooltip: false,
  }
);

const hasEnvironmentTierPolicyFeature = featureToRef(
  "bb.feature.environment-tier-policy"
);

const enabled = computed((): boolean => {
  if (!hasEnvironmentTierPolicyFeature.value) {
    return false;
  }
  return (props.environment?.tier ?? props.tier) === EnvironmentTier.PROTECTED;
});
</script>
