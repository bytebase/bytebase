<template>
  <NTooltip trigger="manual" :show="state.tooltipVisible">
    <template #trigger>
      <!--
        <button disabled> will swallow all mouse related events like mouseover/mouseout...
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
    </template>
    <slot name="tooltip"></slot>
  </NTooltip>
</template>

<script lang="ts">
export default {
  name: "BBButton",
  inheritAttrs: false,
};
</script>

<script lang="ts" setup>
import {
  defineProps,
  withDefaults,
  defineEmits,
  computed,
  reactive,
  watch,
  useSlots,
} from "vue";

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
    type: ButtonType;
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
