<template>
  <div class="sticky top-0 flex flex-col bg-white" :style="panelStyle">
    <slot name="header" />
    <div class="min-h-0 flex-1 overflow-y-auto overscroll-y-contain">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { PLAN_DETAIL_VIEWPORT_HEIGHT_VAR } from "./usePlanDetailViewportVars";

const panelStyle = computed(() => ({
  maxHeight: `var(${PLAN_DETAIL_VIEWPORT_HEIGHT_VAR})`,
}));
</script>
