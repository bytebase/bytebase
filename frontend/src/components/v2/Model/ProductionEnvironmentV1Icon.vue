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

<script lang="ts">
import { computed, defineComponent, PropType } from "vue";
import { featureToRef } from "@/store";
import {
  Environment,
  EnvironmentTier,
} from "@/types/proto/v1/environment_service";

export default defineComponent({
  name: "ProductionEnvironmentV1Icon",
  inheritAttrs: false,
  props: {
    environment: {
      type: Object as PropType<Environment>,
      default: undefined,
    },
    tier: {
      type: Number as PropType<EnvironmentTier>,
      default: EnvironmentTier.UNPROTECTED,
    },
    tooltip: {
      type: Boolean,
      default: false,
    },
  },
  setup(props) {
    const hasEnvironmentTierPolicyFeature = featureToRef(
      "bb.feature.environment-tier-policy"
    );

    const enabled = computed((): boolean => {
      if (!hasEnvironmentTierPolicyFeature.value) {
        return false;
      }
      return (
        (props.environment?.tier ?? props.tier) === EnvironmentTier.PROTECTED
      );
    });

    return { enabled };
  },
});
</script>
