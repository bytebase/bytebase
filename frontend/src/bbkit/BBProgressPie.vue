<template>
  <div class="relative flex justify-center items-center">
    <svg :viewBox="viewBox" class="absolute inset-0 z-0">
      <path
        class="stroke-control-bg"
        fill="none"
        :stroke-width="thickness + 1"
        d="M18 2.0854 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
      />
      <path
        fill="none"
        stroke="currentColor"
        stroke-linecap="round"
        animation="progress 1s ease-out forwards"
        :stroke-width="thickness"
        :stroke-dasharray="strokeDasharray"
        d="M18 2.0854 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
      />
    </svg>
    <slot name="default" :percent="displayPercent">
      {{ displayPercent }}%
    </slot>
  </div>
</template>

<script lang="ts" setup>
import { computed, withDefaults } from "vue";

/**
 * [Learn more](https://medium.com/@pppped/how-to-code-a-responsive-circular-percentage-chart-with-svg-and-css-3632f8cd7705)
 */

const props = withDefaults(
  defineProps<{
    percent?: number; // range: [0, 100]
    thickness?: number; // min value: 1
  }>(),
  {
    percent: 0,
    thickness: 3,
  }
);

const viewBox = computed(() => {
  const offset = Math.max(props.thickness - 3, 0);
  const vb = {
    minX: 0 - offset,
    minY: 0 - offset,
    width: 36 + offset * 2,
    height: 36 + offset * 2,
  };
  return [vb.minX, vb.minY, vb.width, vb.height].join(" ");
});

const displayPercent = computed(() => {
  const { percent } = props;
  if (percent >= 100) return 100;
  if (percent <= 0) return 0;
  if (percent >= 99 && percent < 100) return 99; // do not round to 100 if percent >= 99.5
  return Math.round(percent);
});

const strokeDasharray = computed(() => {
  let percent = displayPercent.value;
  const offset = props.thickness - 3;
  const max = 96 - offset;
  if (percent >= max && percent < 100) {
    // keep a tiny gap when it's very closed to 100%
    percent = max;
  }

  return `${percent}, 100`;
});
</script>
