<template>
  <NTooltip
    v-if="environment.tier === 'PROTECTED'"
    :disabled="!tooltip"
    :delay="0"
    :show-arrow="false"
    :animated="false"
    placement="top-start"
  >
    <template #trigger>
      <heroicons-solid:shield-exclamation
        class="text-control inline-block"
        v-bind="$attrs"
      />
    </template>

    <span>{{ $t("environment.protected") }}</span>
  </NTooltip>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { Environment } from "@/types";

export default defineComponent({
  name: "ProtectedEnvironmentIcon",
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
});
</script>
