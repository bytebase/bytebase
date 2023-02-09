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

    <span>{{ $t("environment.protected") }}</span>
  </NTooltip>
</template>

<script lang="ts">
import { computed, defineComponent, PropType } from "vue";
import { Environment } from "@/types";
import { featureToRef } from "@/store";

export default defineComponent({
  name: "ProductionEnvironmentIcon",
  inheritAttrs: false,
  props: {
    environment: {
      type: Object as PropType<Environment>,
      required: true,
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
      return props.environment.tier === "PROTECTED";
    });

    return { enabled };
  },
});
</script>
