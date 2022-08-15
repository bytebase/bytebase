<template>
  <span class="tooltip-wrapper w-full">
    <span v-if="hasTextOverflow" class="tooltip whitespace-nowrap">
      <slot></slot>
    </span>
    <div ref="textRef" class="truncate w-full"><slot></slot></div>
  </span>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";

const textRef = ref<HTMLElement | null>(null);
const hasTextOverflow = computed(() => {
  return textRef.value && textRef.value.offsetWidth < textRef.value.scrollWidth;
});
</script>
