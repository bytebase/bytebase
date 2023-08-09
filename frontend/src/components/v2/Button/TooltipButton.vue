<template>
  <NTooltip v-bind="tooltipProps" :disabled="!tooltipDisabled">
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

<script lang="ts">
export default {
  name: "BBTooltipButton",
  inheritAttrs: false,
};
</script>

<script lang="ts" setup>
import { withDefaults, computed, useSlots } from "vue";
import { NTooltip, type TooltipProps } from "naive-ui";
import { TooltipMode } from "./types";

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

const tooltipDisabled = computed(() => {
  if (props.tooltipMode === "ALWAYS") return false;
  if (!slots.tooltip) return true;
  return props.disabled;
});
</script>
