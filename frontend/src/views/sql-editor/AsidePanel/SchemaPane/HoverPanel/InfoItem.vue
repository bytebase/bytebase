<template>
  <div class="w-full grid grid-cols-2 gap-x-2">
    <NEllipsis class="max-w-1/3">
      <span class="text-gray-500 font-medium whitespace-nowrap">
        <slot name="title">{{ title }}</slot>
      </span>
    </NEllipsis>
    <NEllipsis class="max-w-2/3 text-right">
      <span class="whitespace-nowrap">
        <slot name="default"></slot>
      </span>
    </NEllipsis>
  </div>
</template>

<script setup lang="ts">
import { NEllipsis } from "naive-ui";

defineProps<{
  title?: string;
}>();
</script>
