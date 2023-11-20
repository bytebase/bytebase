<template>
  <div
    v-zindexable="{ enabled: true }"
    class="absolute inset-0 bg-white/50 flex flex-col items-center justify-center"
  >
    <NSpin v-bind="spinProps" />
  </div>
</template>

<script lang="ts" setup>
import { NSpin, SpinProps } from "naive-ui";
import { zindexable as vZindexable } from "vdirs";

defineProps<{
  spinProps?: SpinProps;
}>();
</script>
