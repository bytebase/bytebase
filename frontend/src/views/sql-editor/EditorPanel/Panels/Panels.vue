<template>
  <div class="flex-1 flex items-stretch overflow-hidden">
    <GutterBar class="border-r border-control-border" :class="gutterBarClass" />

    <div class="flex-1" :class="contentClass">
      <slot name="code-panel" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { VueClass } from "@/utils";
import GutterBar from "../GutterBar";

defineProps<{
  gutterBarClass?: VueClass;
  contentClass?: VueClass;
}>();
</script>
