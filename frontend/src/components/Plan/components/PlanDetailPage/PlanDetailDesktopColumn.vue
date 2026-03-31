<template>
  <component
    :is="props.tag"
    class="min-w-0 border-l bg-white"
    :style="columnStyle"
  >
    <StickySidePanel>
      <template v-if="$slots.header" #header>
        <slot name="header" />
      </template>
      <slot />
    </StickySidePanel>
  </component>
</template>

<script setup lang="ts">
import type { CSSProperties } from "vue";
import { computed } from "vue";
import StickySidePanel from "./StickySidePanel.vue";
import { PLAN_DETAIL_CONTENT_MIN_HEIGHT_VAR } from "./usePlanDetailViewportVars";

const props = withDefaults(
  defineProps<{
    tag?: "aside" | "div";
  }>(),
  {
    tag: "div",
  }
);

const columnStyle = computed<CSSProperties>(() => ({
  minHeight: `var(${PLAN_DETAIL_CONTENT_MIN_HEIGHT_VAR})`,
}));
</script>
