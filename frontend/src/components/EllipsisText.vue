<template>
  <component
    :is="performant ? NPerformantEllipsis : NEllipsis"
    :line-clamp="lineClamp"
    :tooltip="{
      showArrow: false,
      raw: true,
      contentStyle: tooltipStyle,
    }"
  >
    <slot></slot>
  </component>
</template>

<script lang="ts" setup>
import { NEllipsis, NPerformantEllipsis } from "naive-ui";

defineProps({
  performant: {
    type: Boolean,
    default: false,
  },
  lineClamp: {
    type: Number,
    default: 1,
  },
});

const tooltipStyle = `
  border-radius: 0.25rem;
  background-color: rgb(0 0 0 / var(--tw-bg-opacity));
  --tw-bg-opacity: 0.75;
  padding-left: 0.5rem;
  padding-right: 0.5rem;
  padding-top: 0.25rem;
  padding-bottom: 0.25rem;
  --tw-text-opacity: 1;
  color: rgb(255 255 255 / var(--tw-text-opacity));
  max-width: 80vw;
`;
</script>
