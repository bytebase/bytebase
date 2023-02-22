<template>
  <NTooltip trigger="manual" :show="state.tooltipVisible">
    <template #trigger>
      <!--
        Allowing to overwrite the entire button
        with manual tooltip control functions
      -->
      <slot
        name="button"
        :show-tooltip="showTooltip"
        :hide-tooltip="hideTooltip"
      >
        <!--
          <button disabled> will swallow all mouse related events such as mouseover/mouseout...
          so we need to handle it manually with lower level DOM pointer events
        -->
        <button
          type="button"
          v-bind="$attrs"
          :disabled="disabled"
          :class="[`btn-${props.type}`, $attrs.class]"
          @click.prevent="$emit('click')"
          @pointerenter="showTooltip"
          @pointerleave="hideTooltip"
        >
          <slot name="default"></slot>
        </button>
      </slot>
    </template>
    <slot name="tooltip"></slot>
  </NTooltip>
</template>

<script lang="ts">
export default {
  name: "BBTooltipButton",
  inheritAttrs: false,
};
</script>

<script lang="ts" setup>
import { withDefaults, computed, reactive, watch, useSlots } from "vue";

export type ButtonType =
  | "normal"
  | "primary"
  | "secondary"
  | "cancel"
  | "danger"
  | "success";

export type TooltipMode = "ALWAYS" | "DISABLED-ONLY";

const props = withDefaults(
  defineProps<{
    type?: ButtonType;
    disabled: boolean;
    tooltipMode: TooltipMode;
  }>(),
  {
    type: "normal",
    disabled: false,
    tooltipMode: "ALWAYS",
  }
);

defineEmits<{
  (event: "click"): void;
}>();

const slots = useSlots();

const state = reactive({
  tooltipVisible: false,
});

const tooltipEnabled = computed(() => {
  if (props.tooltipMode === "ALWAYS") return true;
  if (!slots.tooltip) return false;
  return props.disabled;
});

const showTooltip = () => {
  if (!tooltipEnabled.value) return;
  state.tooltipVisible = true;
};

const hideTooltip = () => {
  state.tooltipVisible = false;
};

watch(tooltipEnabled, (enable) => {
  if (!enable) {
    state.tooltipVisible = false;
  }
});
</script>
