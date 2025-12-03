<template>
  <div class="h-full flex flex-col">
    <div class="flex-1 mb-6">
      <slot name="title">
        <div v-if="title" class="text-lg leading-6 font-medium text-main">
          {{ title }}
        </div>
        <NDivider v-if="title" />
      </slot>
      <slot name="body" />
    </div>
    <div class="w-full sticky bottom-0 z-10">
      <div
        class="w-full py-4 border-t border-block-border bg-white"
      >
        <slot name="footer" />
      </div>
    </div>
  </div>
</template>

<script setup lang="tsx">
import { NDivider } from "naive-ui";

defineProps<{
  title?: string;
}>();
</script>
