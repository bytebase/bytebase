<template>
  <div
    class="w-full grid grid-cols-2 gap-x-4 items-center"
    style="grid-template-columns: auto 1fr"
  >
    <div class="text-gray-500 font-medium whitespace-nowrap">
      <slot name="title">{{ title }}</slot>
    </div>
    <NPerformantEllipsis class="text-right flex! items-center justify-end">
      <span class="whitespace-nowrap flex items-center leading-[20px] h-[20px]">
        <slot name="default"></slot>
      </span>
    </NPerformantEllipsis>
  </div>
</template>

<script setup lang="ts">
import { NPerformantEllipsis } from "naive-ui";

defineProps<{
  title?: string;
}>();
</script>
