<template>
  <NTooltip v-bind="tooltipProps" :disabled="!tooltipEnabled">
    <template #trigger>
      <NButton
        v-bind="$attrs"
        tag="div"
        :disabled="disabled"
        @click="$emit('click')"
      >
        <template #icon>
          <slot name="icon" />
        </template>
        <template #default>
          <slot />
        </template>
      </NButton>
    </template>
    <template #default>
      <slot name="tooltip" />
    </template>
  </NTooltip>
</template>

<script lang="ts" setup>
import { NButton, NTooltip, type TooltipProps } from "naive-ui";
import { computed, useSlots } from "vue";
import type { TooltipMode } from "./types";

const props = withDefaults(
  defineProps<{
    disabled: boolean;
    tooltipMode: TooltipMode;
    tooltipProps?: TooltipProps;
  }>(),
  {
    disabled: false,
    tooltipMode: "ALWAYS",
    tooltipProps: undefined,
  }
);

defineEmits<{
  (event: "click"): void;
}>();

const slots = useSlots();

const tooltipEnabled = computed(() => {
  // Disable tooltip if no tooltip contents
  if (!slots.tooltip) return false;
  // Enable tooltip when tooltipMode is "ALWAYS"
  if (props.tooltipMode === "ALWAYS") return true;
  // Enable tooltip if button is disabled
  // Disable tooltip otherwise
  return props.disabled;
});
</script>
