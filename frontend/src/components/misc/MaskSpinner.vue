<template>
  <div
    v-zindexable="{ enabled: zindexable }"
    class="absolute inset-0 bg-white/50 flex flex-col items-center justify-center gap-y-2"
  >
    <NSpin v-bind="spinProps" />
    <slot name="default"></slot>
  </div>
</template>

<script lang="ts" setup>
import { NSpin, SpinProps } from "naive-ui";
import { zindexable as vZindexable } from "vdirs";

withDefaults(
  defineProps<{
    spinProps?: SpinProps;
    zindexable?: boolean;
  }>(),
  { spinProps: undefined, zindexable: true }
);
</script>
