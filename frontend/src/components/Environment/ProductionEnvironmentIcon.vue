<template>
  <NTooltip
    v-if="enabled"
    :disabled="!tooltip"
    :delay="0"
    :show-arrow="false"
    :animated="false"
    placement="top-start"
  >
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
import { Environment } from "@/types";

export default defineComponent({
  name: "ProductionEnvironmentIcon",
  inheritAttrs: false,
  props: {
    environment: {
      type: Object as PropType<Environment>,
      default: undefined,
    },
    tier: {
      type: String,
      default: "",
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
      return (props.environment?.tier || props.tier) === "PROTECTED";
    });

    return { enabled };
  },
});
</script>
