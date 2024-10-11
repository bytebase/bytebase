<template>
  <div class="inline-block relative">
    <AlignLeftIcon
      :style="{
        width: `${size}px`,
        height: `${size}px`,
      }"
    />
    <ArrowLeftIcon
      class="text-accent absolute"
      :stroke-width="4"
      :style="{
        width: `${size * 0.7}px`,
        height: `${size * 0.7}px`,
        right: `${size * -0.2}px`,
        bottom: `${size * 0.035}px`,
      }"
    />
  </div>
</template>

<script setup lang="ts">
import { AlignLeftIcon, ArrowLeftIcon } from "lucide-vue-next";

defineProps<{
  size: number;
}>();
</script>
