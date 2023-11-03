<template>
  <div class="contents">
    <div class="text-gray-500 font-medium">
      <slot name="title">{{ title }}</slot>
    </div>
    <div class="text-main text-right">
      <slot name="default"></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title?: string;
}>();
</script>
